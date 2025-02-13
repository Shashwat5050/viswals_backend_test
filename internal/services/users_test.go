package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/viswals_backend_test/core/models"
	"github.com/viswals_backend_test/core/encryptions"
	"github.com/viswals_backend_test/core/postgres/mockpostgres"
	"github.com/viswals_backend_test/core/redis/mockredis"
	"go.uber.org/zap"
	"testing"
)

type GetUserTestCase struct {
	name       string
	service    *UserService
	input      string
	output     *models.UserDetails
	throwError bool
}

func TestGetUser(t *testing.T) {
	// mock variable declaration
	mockMemStore := new(mockredis.MockRedis)
	mockMemStoreNoData := new(mockredis.MockRedis)
	mockUserStore := new(mockpostgres.MockDatabase)
	mockUserStoreError := new(mockpostgres.MockDatabase)
	encryp,err := encryptions.New([]byte("testtesttesttest"))
	assert.NoError(t,err)

	actualEmail := "test@test.com"
	encryptedEmail, err := encryp.Encrypt(actualEmail)
	assert.NoError(t, err)

	decrypted, err := encryp.Decrypt(encryptedEmail)
	assert.NoError(t, err)
	assert.Equal(t, decrypted, actualEmail)

	outputData := &models.UserDetails{
		ID:           1,
		FirstName:    "test",
		LastName:     "test",
		EmailAddress: encryptedEmail,
		CreatedAt:    sql.NullTime{},
		DeletedAt:    sql.NullTime{},
		MergedAt:     sql.NullTime{},
		ParentUserId: 0,
	}

	log, err := zap.NewDevelopment()
	assert.NoError(t, err)

	var nilOutput *models.UserDetails = nil

	mockMemStore.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(outputData, nil)
	mockUserStore.On("GetUserByID", mock.Anything, mock.AnythingOfType("string")).Return(outputData, nil)
	mockMemStoreNoData.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nilOutput, errors.New("test error"))
	mockMemStoreNoData.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*models.UserDetails")).Return(nil)
	mockUserStoreError.On("GetUserByID", mock.Anything, mock.AnythingOfType("string")).Return(nilOutput, errors.New("test error"))

	testCases := []GetUserTestCase{
		{
			name: "Success: get data from cache",
			service: &UserService{
				dataStore: mockUserStore,
				memStore:  mockMemStore,
				encryp: encryp,
				logger:    log,
			},
			input: "1",
			output: &models.UserDetails{
				ID:           1,
				FirstName:    "test",
				LastName:     "test",
				EmailAddress: actualEmail,
				CreatedAt:    sql.NullTime{},
				DeletedAt:    sql.NullTime{},
				MergedAt:     sql.NullTime{},
				ParentUserId: 0,
			},
			throwError: false,
		}, {
			name: "Success: get data from database",
			service: &UserService{
				dataStore: mockUserStore,
				memStore:  mockMemStoreNoData,
				encryp: encryp,
				logger:    log,
			},
			input: "1",
			output: &models.UserDetails{
				ID:           1,
				FirstName:    "test",
				LastName:     "test",
				EmailAddress: actualEmail,
				CreatedAt:    sql.NullTime{},
				DeletedAt:    sql.NullTime{},
				MergedAt:     sql.NullTime{},
				ParentUserId: 0,
			},
			throwError: false,
		}, {
			name: "Fail: get data from database",
			service: &UserService{
				dataStore: mockUserStoreError,
				memStore:  mockMemStoreNoData,
				encryp: encryp,
				logger:    log,
			},
			input:      "1",
			output:     nil,
			throwError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			outputData.EmailAddress = encryptedEmail
			d, err := testCase.service.GetUser(context.Background(), testCase.input)
			if testCase.throwError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.output, d)
		})
	}

	mockMemStore.AssertExpectations(t)
	mockUserStore.AssertExpectations(t)
	mockMemStoreNoData.AssertExpectations(t)
	mockUserStoreError.AssertExpectations(t)
}

type GetAllUserTestCase struct {
	name       string
	service    *UserService
	output     []*models.UserDetails
	throwError bool
}

func TestGetAllUser(t *testing.T) {
	mockUserStore := new(mockpostgres.MockDatabase)
	mockUserStoreError := new(mockpostgres.MockDatabase)
	encryp,err := encryptions.New([]byte("testtesttesttest"))
	assert.NoError(t,err)
	log, err := zap.NewDevelopment()
	assert.NoError(t, err)
	actualEmail := "test@test.com"
	encryptedEmail, err := encryp.Encrypt(actualEmail)
	assert.NoError(t, err)

	outputData := []*models.UserDetails{
		{
			ID:           1,
			FirstName:    "test",
			LastName:     "test",
			EmailAddress: encryptedEmail,
			CreatedAt:    sql.NullTime{},
			DeletedAt:    sql.NullTime{},
			MergedAt:     sql.NullTime{},
			ParentUserId: 0,
		},
	}

	var nilOutput []*models.UserDetails = nil

	mockUserStore.On("GetAllUsers", mock.Anything).Return(outputData, nil)
	mockUserStoreError.On("GetAllUsers", mock.Anything).Return(nilOutput, errors.New("test error"))

	testcase := []GetAllUserTestCase{
		{
			name: "Success: get data from db",
			service: &UserService{
				dataStore: mockUserStore,
				memStore:  nil,
				encryp: encryp,
				logger:    log,
			},
			output: []*models.UserDetails{
				{
					ID:           1,
					FirstName:    "test",
					LastName:     "test",
					EmailAddress: actualEmail,
					CreatedAt:    sql.NullTime{},
					DeletedAt:    sql.NullTime{},
					MergedAt:     sql.NullTime{},
					ParentUserId: 0,
				},
			},
			throwError: false,
		}, {
			name: "Fail: get data from database",
			service: &UserService{
				dataStore: mockUserStoreError,
				memStore:  nil,
				encryp: encryp,
				logger:    log,
			},
			output:     nil,
			throwError: true,
		},
	}

	for _, testCase := range testcase {
		t.Run(testCase.name, func(t *testing.T) {
			output, err := testCase.service.GetAllUsers(context.Background())
			if testCase.throwError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.output, output)
		})
	}

	mockUserStoreError.AssertExpectations(t)
	mockUserStore.AssertExpectations(t)

}

type DeleteUserTestCase struct {
	name       string
	service    *UserService
	input      string
	throwError bool
}

func TestDeleteUser(t *testing.T) {
	mockMemStore := new(mockredis.MockRedis)
	mockUserStore := new(mockpostgres.MockDatabase)
	mockUserStoreError := new(mockpostgres.MockDatabase)
	mockMemStoreError := new(mockredis.MockRedis)
	encryp,err := encryptions.New([]byte("testtesttesttest"))
	assert.NoError(t,err)

	log, err := zap.NewDevelopment()
	assert.NoError(t, err)

	mockUserStore.On("DeleteUser", mock.Anything, mock.AnythingOfType("string")).Return(nil)
	mockMemStore.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil)
	mockUserStoreError.On("DeleteUser", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("test error"))
	mockMemStoreError.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil)

	testCases := []DeleteUserTestCase{
		{
			name: "Success: delete data from both cache and db",
			service: &UserService{
				dataStore: mockUserStore,
				memStore:  mockMemStore,
				encryp: encryp,
				logger:    log,
			},
			input:      "1",
			throwError: false,
		}, {
			name: "Success: delete data from db Only",
			service: &UserService{
				dataStore: mockUserStore,
				memStore:  mockMemStoreError,
				encryp: encryp,
				logger:    log,
			},
			input:      "1",
			throwError: false,
		}, {
			name: "Fail: delete data from both cache and db",
			service: &UserService{
				dataStore: mockUserStoreError,
				memStore:  mockMemStoreError,
				encryp: encryp,
				logger:    log,
			},
			input:      "1",
			throwError: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.service.DeleteUser(context.Background(), testCase.input)
			if testCase.throwError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	mockMemStore.AssertExpectations(t)
	mockUserStore.AssertExpectations(t)
	mockMemStoreError.AssertExpectations(t)
	mockUserStoreError.AssertExpectations(t)
}

type CreateUserTestCase struct {
	name       string
	service    *UserService
	input      *models.UserDetails
	throwError bool
}

func TestCreateUser(t *testing.T) {
	mockMemStore := new(mockredis.MockRedis)
	mockUserStore := new(mockpostgres.MockDatabase)
	mockUserStoreError := new(mockpostgres.MockDatabase)
	mockMemStoreError := new(mockredis.MockRedis)
	encryp,err := encryptions.New([]byte("testtesttesttest"))
	assert.NoError(t,err)
	log, err := zap.NewDevelopment()
	assert.NoError(t, err)

	input := &models.UserDetails{
		ID:           0,
		FirstName:    "test",
		LastName:     "test",
		EmailAddress: "test@test.com",
		CreatedAt:    sql.NullTime{},
		DeletedAt:    sql.NullTime{},
		MergedAt:     sql.NullTime{},
		ParentUserId: 0,
	}

	mockUserStore.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.UserDetails")).Return(nil)
	mockUserStoreError.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.UserDetails")).Return(errors.New("test error"))
	mockMemStore.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*models.UserDetails")).Return(nil)
	mockMemStoreError.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*models.UserDetails")).Return(errors.New("test error"))

	testCases := []CreateUserTestCase{
		{
			name: "Success: create data from both cache and db",
			service: &UserService{
				dataStore: mockUserStore,
				memStore:  mockMemStore,
				encryp: encryp,
				logger:    log,
			},
			input:      input,
			throwError: false,
		}, {
			name: "Success: create data on DB Only",
			service: &UserService{
				dataStore: mockUserStore,
				memStore:  mockMemStoreError,
				encryp: encryp,
				logger:    log,
			},
			input:      input,
			throwError: false,
		}, {
			name: "Fail: create data from both cache and db",
			service: &UserService{
				dataStore: mockUserStoreError,
				memStore:  mockMemStoreError,
				encryp: encryp,
				logger:    log,
			},
			input:      input,
			throwError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.service.CreateUser(context.Background(), testCase.input)
			if testCase.throwError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	mockMemStore.AssertExpectations(t)
	mockUserStore.AssertExpectations(t)
	mockMemStoreError.AssertExpectations(t)
	mockUserStoreError.AssertExpectations(t)

}

type GetAllUserSSETestCase struct {
	name       string
	service    *UserService
	limit      int64
	offset     int64
	output     []byte
	throwError bool
}

func TestGetAllUserSSE(t *testing.T) {
	mockUserStore := new(mockpostgres.MockDatabase)
	mockUserStoreError := new(mockpostgres.MockDatabase)
	log, err := zap.NewDevelopment()
	assert.NoError(t, err)
	encryp,err := encryptions.New([]byte("testtesttesttest"))
	assert.NoError(t,err)
	actualEmail := "test@test.com"
	encryptedEmail, err := encryp.Encrypt(actualEmail)
	assert.NoError(t, err)

	limit := int64(1)
	offset := int64(1)

	outputData := []*models.UserDetails{
		{
			ID:           1,
			FirstName:    "test",
			LastName:     "test",
			EmailAddress: encryptedEmail,
			CreatedAt:    sql.NullTime{},
			DeletedAt:    sql.NullTime{},
			MergedAt:     sql.NullTime{},
			ParentUserId: 0,
		}, {
			ID:           2,
			FirstName:    "test",
			LastName:     "test",
			EmailAddress: encryptedEmail,
			CreatedAt:    sql.NullTime{},
			DeletedAt:    sql.NullTime{},
			MergedAt:     sql.NullTime{},
			ParentUserId: 0,
		}, {
			ID:           3,
			FirstName:    "test",
			LastName:     "test",
			EmailAddress: encryptedEmail,
			CreatedAt:    sql.NullTime{},
			DeletedAt:    sql.NullTime{},
			MergedAt:     sql.NullTime{},
			ParentUserId: 0,
		},
	}

	var nilOutput []*models.UserDetails = nil

	mockUserStore.On("ListUsers", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).Return(outputData[offset:offset+limit], nil)
	mockUserStoreError.On("ListUsers", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).Return(nilOutput, errors.New("test error"))

	expectedOutput := []*models.UserDetails{
		{
			ID:           2,
			FirstName:    "test",
			LastName:     "test",
			EmailAddress: actualEmail,
			CreatedAt:    sql.NullTime{},
			DeletedAt:    sql.NullTime{},
			MergedAt:     sql.NullTime{},
			ParentUserId: 0,
		},
	}
	expectedData, err := json.Marshal(expectedOutput)
	assert.NoError(t, err)

	testcase := []GetAllUserSSETestCase{
		{
			name: "Success: get data from db",
			service: &UserService{
				dataStore: mockUserStore,
				memStore:  nil,
				encryp: encryp,
				logger:    log,
			},
			limit:      limit,
			offset:     offset,
			output:     expectedData,
			throwError: false,
		}, {
			name: "Fail: get data from database",
			service: &UserService{
				dataStore: mockUserStoreError,
				memStore:  nil,
				encryp: encryp,
				logger:    log,
			},
			limit:      limit,
			offset:     offset,
			output:     nil,
			throwError: true,
		},
	}

	for _, testCase := range testcase {
		t.Run(testCase.name, func(t *testing.T) {
			output, err := testCase.service.GetAllUsersSSE(context.Background(), testCase.limit, testCase.offset)
			if testCase.throwError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.output, output)
		})
	}

	mockUserStoreError.AssertExpectations(t)
	mockUserStore.AssertExpectations(t)

}
