package services

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/viswals_backend_test/core/models"
)

type MessageBroker interface {
	Publish(context.Context, []byte) error
	Subscribe() (<-chan amqp.Delivery, error)
	Close() error
}

type userStoreProvider interface {
	GetUserByID(context.Context, string) (*models.UserDetails, error)
	CreateUser(context.Context, *models.UserDetails) error
	CreateBulkUsers(context.Context, []*models.UserDetails) error
	GetAllUsers(context.Context) ([]*models.UserDetails, error)
	DeleteUser(context.Context, string) error
	ListUsers(context.Context, int64, int64) ([]*models.UserDetails, error)
}

type CacheManager interface {
	Get(context.Context, string) (*models.UserDetails, error)
	Set(context.Context, string, *models.UserDetails) error
	SetBulk(context.Context, []*models.UserDetails) error
	Delete(context.Context, string) error
}
