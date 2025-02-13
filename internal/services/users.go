package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viswals_backend_test/core/models"
	"github.com/viswals_backend_test/core/encryptions"
	database "github.com/viswals_backend_test/core/postgres"
	"go.uber.org/zap"
)

type UserService struct {
	dataStore userStoreProvider
	memStore  CacheManager
	encryp *encryptions.Encryption
	logger    *zap.Logger
}

func NewUserService(dataStore userStoreProvider, memStore CacheManager, encryption *encryptions.Encryption,logger *zap.Logger) *UserService {
	return &UserService{
		dataStore: dataStore,
		memStore:  memStore,
		encryp: encryption,
		logger:    logger,
	}
}

func (us *UserService) GetUser(ctx context.Context, userID string) (*models.UserDetails, error) {
	// first try to fetch data from cache.
	var user *models.UserDetails
	var err error
	
	user, err = us.memStore.Get(ctx, userID)
	if err != nil {
		us.logger.Warn("UserService: error getting user from cache", zap.String("user_id", userID), zap.Error(err))
		// if error fetch data from database
		user, err = us.dataStore.GetUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		us.logger.Debug("UserService: user fetched from database", zap.String("user_id", userID))
		// if data is successfully fetched, update the cache.
		err = us.memStore.Set(ctx, userID, user)
		if err != nil {
			// log the error and we can safely ignore this error.
			us.logger.Warn("UserService: error setting user in cache", zap.String("user_id", userID), zap.Error(err))
		}
	}
	decryptedEmail, err := us.encryp.Decrypt(user.EmailAddress)
	if err != nil {
		return nil, err
	}

	user.EmailAddress = decryptedEmail

	return user, nil
}

func (us *UserService) GetAllUsers(ctx context.Context) ([]*models.UserDetails, error) {
	// fetch and return data from db for now.
	users, err := us.dataStore.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		decryptedEmail, err := us.encryp.Decrypt(user.EmailAddress)
		if err != nil {
			us.logger.Error("error decrypting email", zap.String("email", user.EmailAddress), zap.Error(err))
			return nil, err
		}

		user.EmailAddress = decryptedEmail
	}
	return users, nil
}

func (us *UserService) DeleteUser(ctx context.Context, userID string) error {
	// delete user from db first
	err := us.dataStore.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}

	// delete user from memory.
	err = us.memStore.Delete(ctx, userID)
	if err != nil {
		us.logger.Warn("UserService: error deleting user from cache", zap.String("user_id", userID))
		// the data will be automatically expired with TTL.
	}

	return nil
}

func (us *UserService) CreateUser(ctx context.Context, user *models.UserDetails) error {
	// encrypt users email id
	newEmail, err := us.encryp.Encrypt(user.EmailAddress)
	if err != nil {
		return err
	}

	user.EmailAddress = newEmail

	// first, insert the data in a database.
	err = us.dataStore.CreateUser(ctx, user)
	if err != nil {
		return err
	}

	// upon successful insertion update the cache
	err = us.memStore.Set(ctx, fmt.Sprint(user.ID), user)
	if err != nil {
		us.logger.Warn("UserService: error setting user in cache", zap.Error(err), zap.Any("user", user))
	}

	return nil
}

func (us *UserService) GetAllUsersSSE(ctx context.Context, limit, offset int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var isLastData bool

	users, err := us.dataStore.ListUsers(ctx, limit, offset)
	if err != nil {
		if errors.Is(err, database.ErrNoData) {
			isLastData = true
		}
		return nil, err
	}

	for _, user := range users {
		decryptedEmail, err := us.encryp.Decrypt(user.EmailAddress)
		if err != nil {
			us.logger.Error("error decrypting email", zap.String("email", user.EmailAddress), zap.Error(err))
			return nil, err
		}

		user.EmailAddress = decryptedEmail
	}

	data, err := json.Marshal(users)
	if err != nil {
		return nil, err
	}

	if isLastData {
		return data, database.ErrNoData
	}

	return data, nil
}
