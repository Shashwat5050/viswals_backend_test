package services

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/viswals_backend_test/core/encryptions"
	"github.com/viswals_backend_test/core/models"
	"github.com/viswals_backend_test/core/postgres/mockpostgres"
	"github.com/viswals_backend_test/core/rabbitmq/mockrabbitmq"
	"github.com/viswals_backend_test/core/redis/mockredis"
	"go.uber.org/zap"
)

// TestCase represents a test scenario for the Consumer workflow.
type TestCase struct {
	testName     string
	testConsumer *Consumer
	testChannel  chan amqp.Delivery
	testBody     []byte
}

// TestConsumer validates the consume workflow with different scenarios.
func TestConsumer(t *testing.T) {
	// Initialize mock dependencies
	mockUserStore := new(mockpostgres.MockDatabase)
	mockMemStore := new(mockredis.MockRedis)
	mockQueueStore := new(mockrabbitmq.MockRabbitMQ)

	// Setup logger
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)

	// Initialize encryption
	encryption, err := encryptions.New([]byte("rootrootrootroot"))
	assert.NoError(t, err)

	// Mock database and cache behavior
	mockUserStore.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.UserDetails")).Return(nil)
	mockMemStore.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*models.UserDetails")).Return(nil)

	// Create a delivery channel for messages
	var deliveryChannel = make(chan amqp.Delivery, 10)

	// Define test cases
	testCases := []TestCase{
		{
			testName: "Valid Input",
			testConsumer: &Consumer{
				broker:     mockQueueStore,
				channel:   deliveryChannel,
				encryp:    encryption,
				logger:    logger,
				userRepo: mockUserStore,
				memStore:  mockMemStore,
			},
			testChannel: deliveryChannel,
			testBody: []byte(`
			[
				{
					"id":1,
					"first_name":"John",
					"last_name":"Doe",
					"email_address":"john@doe.com",
					"created_at":null,
					"deleted_at":null,
					"merged_at":null,
					"parent_user_id":1
				}
			]
		`),
		},
	}

	// Execute each test case
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			var wg = new(sync.WaitGroup)
			wg.Add(1)

			// Start the consumer in a goroutine
			go tc.testConsumer.Consume(wg, 1)

			// Send a delivery message
			tc.testChannel <- amqp.Delivery{
				Body: tc.testBody,
			}

			time.Sleep(2 * time.Second) // Allow time for processing
			close(tc.testChannel)       // Close the channel

			wg.Wait() // Wait for processing to complete
		})
	}

	// Verify that all mocks were called as expected
	mockUserStore.AssertExpectations(t)
	mockMemStore.AssertExpectations(t)
	mockQueueStore.AssertExpectations(t)
}

// UserDetailsTest represents a test scenario for parsing user details.
type UserDetailsTest struct {
	testName   string
	testInput  []byte
	testOutput []*models.UserDetails
	hasError   bool
}

// TestToUserDetails validates the ToUserDetails function with different inputs.
func TestConvertToUserDetails(t *testing.T) {
	// Setup logger and encryption
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)

	encryption, err := encryptions.New([]byte("testtesttesttest"))
	assert.NoError(t, err)

	// Define test cases for user details parsing
	var userDetailsTestCases = []UserDetailsTest{
		{
			testName: "valid input",
			testInput: []byte(`
			[
				{
					"id":1,
					"first_name":"John",
					"last_name":"Doe",
					"email_address":"john@doe.com",
					"created_at":null,
					"deleted_at":null,
					"merged_at":null,
					"parent_user_id":1
				}
			]
		`),
			testOutput: []*models.UserDetails{{
				ID:           1,
				FirstName:    "John",
				LastName:     "Doe",
				EmailAddress: "john@doe.com",
				CreatedAt:    sql.NullTime{},
				DeletedAt:    sql.NullTime{},
				MergedAt:     sql.NullTime{},
				ParentUserId: 1,
			}},
			hasError: false,
		},
		{
			testName:   "nil input",
			testInput:  nil,
			testOutput: nil,
			hasError:   true,
		},
		{
			testName: "wrong type input",
			testInput: []byte(`
			[
				{
					"id":1,
					"first_name": 546,
					"last_name":"Doe",
					"email_address":"john@doe.com"
					"created_at":null
					"deleted_at":null
					"merged_at":null
					"parent_user_id":1
				}
			]
		`),
			testOutput: nil,
			hasError:   true,
		},
	}

	consumer := Consumer{
		encryp: encryption,
		logger: logger,
	}

	// Execute each test case
	for _, tt := range userDetailsTestCases {
		t.Run(tt.testName, func(t *testing.T) {
			var inputChan = make(chan []byte, 10)
			var outputChan = make(chan []*models.UserDetails, 10)
			var errorChan = make(chan error)
			wg := new(sync.WaitGroup)
			wg.Add(1)

			// Start the ToUserDetails function in a goroutine
			go consumer.convertToUserDetails(wg, inputChan, outputChan, errorChan)
			inputChan <- tt.testInput
			close(inputChan)

			// Verify outputs or errors
			if tt.hasError {
				e, ok := <-errorChan
				if !ok {
					e = nil
				}
				assert.Error(t, e)
			}
			o, ok := <-outputChan
			if !ok {
				o = nil
			}
			assert.Equal(t, tt.testOutput, o)
			wg.Wait()
		})
	}
}

// Other functions in the code follow a similar pattern of concise commenting.
