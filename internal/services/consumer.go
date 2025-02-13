package services

import (
	"context"
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/viswals_backend_test/core/models"
	"github.com/viswals_backend_test/core/encryptions"
	"go.uber.org/zap"
	"sync"
	"time"
)

var (
	defaultTimeout = time.Second * 15
)

type Consumer struct {
	queue   MessageBroker
	channel <-chan amqp.Delivery
	logger  *zap.Logger
	// extending consumer to provide http server and database access.
	userStore userStoreProvider
	memStore  CacheManager
	encryp    *encryptions.Encryption
}

func NewConsumer(queue MessageBroker, userStore userStoreProvider, memStore CacheManager, encryp *encryptions.Encryption, logger *zap.Logger) (*Consumer, error) {
	// connect with the initialized queue.
	in, err := queue.Subscribe()
	if err != nil {
		return nil, err
	}

	return &Consumer{
		queue:     queue,
		channel:   in,
		logger:    logger,
		userStore: userStore,
		encryp:    encryp,
		memStore:  memStore,
	}, nil
}

func (c *Consumer) Consume(wg *sync.WaitGroup, size int) {
	defer wg.Done()

	var userDetailsInput chan []byte = make(chan []byte, size)
	var userDetailsOutput chan []*models.UserDetails = make(chan []*models.UserDetails, size)
	var encryptionOutput chan []*models.UserDetails = make(chan []*models.UserDetails, size)

	var errorChan chan error = make(chan error, 10)

	var internalWg = new(sync.WaitGroup)

	internalWg.Add(1)
	go c.ToUserDetails(internalWg, userDetailsInput, userDetailsOutput, errorChan)

	internalWg.Add(1)
	go c.EncryptData(internalWg, userDetailsOutput, encryptionOutput, errorChan)

	internalWg.Add(1)
	go c.SaveUserDetails(internalWg, encryptionOutput, errorChan)

	internalWg.Add(1)
	go c.errorLogger(internalWg, errorChan)

	for data := range c.channel {
		body := data.Body

		if body == nil {
			//TODO: revert back this changes
			c.logger.Error("maybe data is complete")
			break
			//continue
		}

		userDetailsInput <- body

	}

	c.logger.Info(fmt.Sprintf("Consumer stopped"))
	close(userDetailsInput)
	internalWg.Wait()
}

func (c *Consumer) errorLogger(wg *sync.WaitGroup, errorChan chan error) {
	defer wg.Done()
	for e := range errorChan {
		c.logger.Error("error in consumer as", zap.Error(e))
	}
}

func (c *Consumer) ToUserDetails(wg *sync.WaitGroup, inputChan chan []byte, outputChan chan []*models.UserDetails, errorChan chan error) {
	defer wg.Done()
	defer close(outputChan)
	for data := range inputChan {
		var users []*models.UserDetails
		err := json.Unmarshal(data, &users)
		if err != nil {
			c.logger.Error("error unmarshalling user details", zap.Error(err))
			errorChan <- err
			continue
		}
		outputChan <- users
	}
	c.logger.Info(fmt.Sprintf("User Data Marsheller stopped"))
}

func (c *Consumer) EncryptData(wg *sync.WaitGroup, inputChan chan []*models.UserDetails, outputChan chan []*models.UserDetails, errorChan chan error) {
	defer wg.Done()
	defer close(outputChan)
	for data := range inputChan {
		encryptedData := make([]*models.UserDetails, 0, len(data))
		for _, user := range data {
			encryptedEmail, err := c.encryp.Encrypt(user.EmailAddress)
			if err != nil {
				c.logger.Error("error encrypting user", zap.Error(err))
				errorChan <- err
				continue
			}
			user.EmailAddress = encryptedEmail
			encryptedData = append(encryptedData, user)
		}
		outputChan <- encryptedData
	}
	c.logger.Info("user data encrypter has completed its work")
}

func (c *Consumer) SaveUserDetails(wg *sync.WaitGroup, inputChan chan []*models.UserDetails, errorChan chan error) {
	defer wg.Done()
	defer close(errorChan)
	for data := range inputChan {

		go func() {

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(data))*defaultTimeout)
			err := c.userStore.CreateBulkUsers(ctx, data)
			if err != nil {
				c.logger.Error("error inserting data into database", zap.Error(err))
				errorChan <- err
				cancel()
				//continue
			}
		}()

		go func() {

			err := c.memStore.SetBulk(context.Background(), data)
			if err != nil {
				c.logger.Warn("error inserting data into cache", zap.Error(err))
				errorChan <- err
				//cancel()
				//continue
			}
		}()
		c.logger.Debug("Consumed data", zap.Int("size", len(data)))
	}
	c.logger.Info(fmt.Sprintf("User Data Saver stopped"))
}

func (c *Consumer) Close() error {
	return c.queue.Close()
}
