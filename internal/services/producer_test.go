package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/viswals_backend_test/core/csvutils"
	"github.com/viswals_backend_test/core/models"
	"github.com/viswals_backend_test/core/rabbitmq/mockrabbitmq"
	"go.uber.org/zap"
)

func TestStart(t *testing.T) {
	batchSize := 1
	mockQueue := new(mockrabbitmq.MockRabbitMQ)

	mockQueue.On("Publish", mock.Anything, mock.AnythingOfType("[]uint8")).Return(nil)

	csvReader, err := csvutils.OpenFile("../../csvfiles/test.csv")
	assert.NoError(t, err)

	log, err := zap.NewDevelopment()
	assert.NoError(t, err)

	producer := &Producer{
		logger:    log,
		broker:     mockQueue,
		csvReader: csvReader,
	}
	err = producer.Start(batchSize)
	assert.NoError(t, err)

	mockQueue.AssertExpectations(t)
}

func TestPublish(t *testing.T) {
	// Mock publishers
	publisher := new(mockrabbitmq.MockRabbitMQ)
	publisherWithError := new(mockrabbitmq.MockRabbitMQ)

	publisher.On("Publish", mock.Anything, mock.AnythingOfType("[]uint8")).Return(nil)
	publisherWithError.On("Publish", mock.Anything, mock.AnythingOfType("[]uint8")).Return(errors.New("mock error"))

	log, err := zap.NewDevelopment()
	assert.NoError(t, err)

	// Test cases
	testCases := []struct {
		name       string
		producer   *Producer
		input      []*models.UserDetails
		throwError bool
	}{
		{
			name: "Valid Input",
			producer: &Producer{
				broker:  publisher,
				logger: log,
			},
			input: []*models.UserDetails{
				{
					ID:           0,
					FirstName:    "test",
					LastName:     "test",
					EmailAddress: "test",
				},
			},
			throwError: false,
		},
		{
			name: "Publisher Error",
			producer: &Producer{
				broker:  publisherWithError,
				logger: log,
			},
			input: []*models.UserDetails{
				{
					ID:           0,
					FirstName:    "test",
					LastName:     "test",
					EmailAddress: "test",
				},
			},
			throwError: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.producer.PublishMessages(context.Background(), tt.input)
			if tt.throwError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	publisher.AssertExpectations(t)
	publisherWithError.AssertExpectations(t)
}

func TestCsvParser(t *testing.T) {
	// Test cases for CSV parsing
	testCases := []struct {
		name   string
		input  [][]string
		output []*models.UserDetails
	}{
		{
			name: "Valid Input with Timestamps",
			input: [][]string{
				{"1", "test", "test", "test@test.com", "1737481973", "1737481973", "1737481973", "-1"},
			},
			output: []*models.UserDetails{
				{
					ID:           1,
					FirstName:    "test",
					LastName:     "test",
					EmailAddress: "test@test.com",
					CreatedAt: sql.NullTime{Time: time.UnixMilli(1737481973), Valid: true},
					DeletedAt: sql.NullTime{Time: time.UnixMilli(1737481973), Valid: true},
					MergedAt:  sql.NullTime{Time: time.UnixMilli(1737481973), Valid: true},
					ParentUserId: -1,
				},
			},
		},
		{
			name: "Invalid Input Types",
			input: [][]string{
				{"not a number", "test", "test", "test@test.com", "-1", "-1", "-1", "-1"},
			},
			output: nil,
		},
	}

	log, err := zap.NewDevelopment()
	assert.NoError(t, err)

	producer := &Producer{
		logger: log,
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			output := producer.ConvertCsvToStruct(tt.input)
			assert.Equal(t, tt.output, output)
		})
	}
}

func TestClose(t *testing.T) {
	mockQueue := new(mockrabbitmq.MockRabbitMQ)
	mockQueue.On("Close").Return(nil)

	producer := &Producer{broker: mockQueue}

	err := producer.Close()
	assert.NoError(t, err)

	mockQueue.AssertExpectations(t)
}
