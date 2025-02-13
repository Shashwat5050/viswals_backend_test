package mockpostgres

import (
	"context"
	"github.com/stretchr/testify/mock"
	"github.com/viswals_backend_test/core/models"
)

type MockDatabase struct {
	mock.Mock
}

func (db *MockDatabase) GetUserByID(ctx context.Context, id string) (*models.UserDetails, error) {
	args := db.Called(ctx, id)
	return args.Get(0).(*models.UserDetails), args.Error(1)
}

func (db *MockDatabase) CreateUser(ctx context.Context, user *models.UserDetails) error {
	args := db.Called(ctx, user)
	return args.Error(0)
}

func (db *MockDatabase) CreateBulkUsers(ctx context.Context, user []*models.UserDetails) error {
	args := db.Called(ctx, user)
	return args.Error(0)
}

func (db *MockDatabase) GetAllUsers(ctx context.Context) ([]*models.UserDetails, error) {
	args := db.Called(ctx)
	return args.Get(0).([]*models.UserDetails), args.Error(1)
}

func (db *MockDatabase) DeleteUser(ctx context.Context, id string) error {
	args := db.Called(ctx, id)
	return args.Error(0)
}

func (db *MockDatabase) ListUsers(ctx context.Context, limit, offset int64) ([]*models.UserDetails, error) {
	args := db.Called(ctx, limit, offset)
	return args.Get(0).([]*models.UserDetails), args.Error(1)
}
