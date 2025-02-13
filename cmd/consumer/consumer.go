package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/viswals_backend_test/controller"
	"github.com/viswals_backend_test/core/encryptions"
	"github.com/viswals_backend_test/core/logger"
	database "github.com/viswals_backend_test/core/postgres"
	"github.com/viswals_backend_test/core/rabbitmq"
	"github.com/viswals_backend_test/core/redis"
	"github.com/viswals_backend_test/internal/services"
	"go.uber.org/zap"
)

var (
	DevEnvironment     = "dev"
	defaultRedisTTLStr = "60s"
	defaultBufferSize  = "50"
)

func main() {
	// initializing logger
	log, err := logger.Init(os.Stdout, strings.ToLower(os.Getenv("ENVIRONMENT")) == DevEnvironment)
	if err != nil {
		fmt.Printf("can't initialise logger throws error : %v", err)
		return
	}

	// Initialize encryption service
	encryptor, err := encryptions.New([]byte(os.Getenv("ENCRYPTION_KEY")))
	if err != nil {
		log.Error("encryption key not provided or invalid ", zap.Error(err))
		return
	}

	// Fetch and validate environment variables for services
	DbUrl := os.Getenv("POSTGRES_CONNECTION_STRING")
	if DbUrl == "" {
		log.Error("postgres connection string is not set")
		return
	}
		

	cacheURL:= os.Getenv("REDIS_CONNECTION_STRING")
	if cacheURL == "" {	
		log.Error("redis connection string is not se")
		return
	}

	QueueUrl := os.Getenv("RABBITMQ_CONNECTION_STRING")
	if QueueUrl == "" {
		log.Error("rabbitmq connection string is not set")		
		return
	}


	// initializing database service.
	dataStore, err := database.New(DbUrl)
	if err != nil {
		log.Error("can't initialise database throws error", zap.Error(err))
		return
	}

	// Perform database migration if enabled
	doMigration := strings.ToLower(os.Getenv("MIGRATION")) == "true"

	if doMigration {
		dbName, ok := os.LookupEnv("DATABASE_NAME")
		if !ok {
			log.Error("database name is not set can't process migration")
			return
		}
		err := dataStore.Migrate(dbName)
		if err != nil {
			log.Error("can't migrate database throws error", zap.Error(err))
			return
		}
	}

	// Set up Redis TTL (time-to-live)
	ttlstr, ok := os.LookupEnv("REDIS_TTL")
	if !ok {
		ttlstr = defaultRedisTTLStr
		log.Warn("redis TTL is not set using default values, you can specify ttl with REDIS_TTL flag", zap.String("ttl", ttlstr))
	}

	ttl, err := time.ParseDuration(ttlstr)
	if err != nil {
		log.Error("error fetching redis TTL throws error", zap.Error(err), zap.String("ttl", ttlstr))
		return
	}

	// initializing caching service.
	memStore, err := redis.New(cacheURL, ttl)
	if err != nil {
		log.Error("can't initialise redis throws error", zap.Error(err))
		return
	}

	queueName, ok := os.LookupEnv("RABBITMQ_QUEUE_NAME")
	if !ok {
		log.Error("queue name is not set using environment variable 'QUEUE_NAME'")
		return
	}

	// initializing queuing service.
	queueService, err := rabbitmq.New(QueueUrl, queueName)
	if err != nil {
		log.Error("can't initialise rabbitmq throws error", zap.Error(err))
		return
	}

	// Initialize data consumer
	dataconsumer, err := services.NewConsumer(queueService, dataStore, memStore, encryptor, log)
	if err != nil {
		log.Error("can't initialise database throws error", zap.Error(err))
		return
	}

	// Fetch and validate buffer size for channel
	channelCapacity := os.Getenv("CHANNEL_SIZE")
	if channelCapacity == "" {
		log.Warn("Buffer size is not set using environment variable 'CHANNEL_SIZE', using default buffer size", zap.Any("buffer_size", defaultBufferSize))

		channelCapacity = defaultBufferSize
	}

	bufferSize, err := strconv.Atoi(channelCapacity)
	if err != nil {
		log.Error("error parsing buffer size throws error", zap.Error(err), zap.Int("buffer_size", bufferSize))
		return
	}

	// create a separate go routine to handle upcoming data.
	wg := &sync.WaitGroup{}
	log.Info("starting consumer")

	wg.Add(2)
	go dataconsumer.Consume(wg, bufferSize)

	// initialize user service.
	userService := services.NewUserService(dataStore, memStore, encryptor, log)

	httpMux := http.NewServeMux()
	// Initialize Gin controller
	ginController := controller.New(userService, log, controller.WithHttpMux(httpMux), controller.WithHttpPort(os.Getenv("HTTP_PORT")))

	
	log.Info("starting HTTP server", zap.String("port", ginController.HttpPort))
	
	go func() {
		defer wg.Done()
		log.Info("starting http server", zap.String("port", ginController.HttpPort))
		if err := ginController.Start(); err != nil {
			log.Error("cannot run http server", zap.Error(err), zap.String("port", ginController.HttpPort))
			panic(err)
		}
	}()

	// Wait for goroutines to finish
	log.Info("stopping consumer")
	wg.Wait()
}

