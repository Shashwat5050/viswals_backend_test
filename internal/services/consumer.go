package services

import (
	"context"
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/viswals_backend_test/core/encryptions"
	"github.com/viswals_backend_test/core/models"
	"go.uber.org/zap"
	"sync"
	"time"
)

var (
	defaultTimeout = time.Second * 15
)

type Consumer struct {
	broker  MessageBroker
	channel <-chan amqp.Delivery
	logger  *zap.Logger
	userRepo UserRepository
	memStore CacheManager
	encryp   *encryptions.Encryption
}

func NewConsumer(broker MessageBroker, userRepo UserRepository, memStore CacheManager, encryp *encryptions.Encryption, logger *zap.Logger) (*Consumer, error) {
	// connect with the initialized queue.
	in, err := broker.Subscribe()
	if err != nil {
		return nil, err
	}

	return &Consumer{
		broker:   broker,
		channel:  in,
		logger:   logger,
		userRepo: userRepo,
		encryp:   encryp,
		memStore: memStore,
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
	go c.convertToUserDetails(internalWg, userDetailsInput, userDetailsOutput, errorChan)

	internalWg.Add(1)
	go c.encryptUserData(internalWg, userDetailsOutput, encryptionOutput, errorChan)

	internalWg.Add(1)
	go c.saveUserData(internalWg, encryptionOutput, errorChan)

	internalWg.Add(1)
	go c.logErrors(internalWg, errorChan)

	for data := range c.channel {
		body := data.Body

		if body == nil {
			c.logger.Error("maybe data is complete")
			break
		}

		userDetailsInput <- body

	}

	c.logger.Info(fmt.Sprintf("Consumer stopped"))
	close(userDetailsInput)
	internalWg.Wait()
}

func (c *Consumer) logErrors(wg *sync.WaitGroup, errorChan chan error) {
	defer wg.Done()
	for e := range errorChan {
		c.logger.Error("error in consumer as", zap.Error(e))
	}
}

func (c *Consumer)convertToUserDetails(wg *sync.WaitGroup, inputChan chan []byte, outputChan chan []*models.UserDetails, errorChan chan error) {
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

func (c *Consumer) encryptUserData(wg *sync.WaitGroup, inputChan chan []*models.UserDetails, outputChan chan []*models.UserDetails, errorChan chan error) {
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

func (c *Consumer) saveUserData(wg *sync.WaitGroup, inputChan chan []*models.UserDetails, errorChan chan error) {
	defer wg.Done()
	defer close(errorChan)
	for data := range inputChan {

		go func() {

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(data))*defaultTimeout)
			err := c.userRepo.CreateBulkUsers(ctx, data)
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
			}
		}()
		c.logger.Debug("Consumed data", zap.Int("size", len(data)))
	}
	c.logger.Info(fmt.Sprintf("User Data Saver stopped"))
}

func (c *Consumer) Close() error {
	return c.broker.Close()
}
