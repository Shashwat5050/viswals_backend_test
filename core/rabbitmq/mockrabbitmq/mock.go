package mockrabbitmq

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/mock"
)

type MockRabbitMQ struct {
	mock.Mock
}

func (m *MockRabbitMQ) Publish(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockRabbitMQ) Subscribe() (<-chan amqp.Delivery, error) {
	args := m.Called()
	return args.Get(0).(<-chan amqp.Delivery), args.Error(1)
}

func (m *MockRabbitMQ) Close() error {
	args := m.Called()
	return args.Error(0)
}
