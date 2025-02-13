package controller

import (
	"context"

	"github.com/viswals_backend_test/core/models"
)

type UserService interface {
	GetAllUsers(context.Context) ([]*models.UserDetails, error)
	GetUser(context.Context, string) (*models.UserDetails, error)
	CreateUser(context.Context, *models.UserDetails) error
	DeleteUser(context.Context, string) error
	GetAllUsersSSE(ctx context.Context, limit, lastKey int64) ([]byte, error)
}
