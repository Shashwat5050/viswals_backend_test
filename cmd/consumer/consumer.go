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

	encryptor, err := encryptions.New([]byte(os.Getenv("ENCRYPTION_KEY")))
	if err != nil {
		log.Error("encryption key not provided or invalid ", zap.Error(err))
		return
	}

	// initialize environment variables.
	var DbUrl string
	var cacheURL string
	var QueueUrl string

	if url, ok := os.LookupEnv("POSTGRES_CONNECTION_STRING"); ok {
		DbUrl = url
	} else {
		log.Error("postgres connection string is not set using default url")
		return
	}

	if url, ok := os.LookupEnv("REDIS_CONNECTION_STRING"); ok {
		cacheURL = url
	} else {
		log.Error("redis connection string is not set using default url")
		return
	}

	if url, ok := os.LookupEnv("RABBITMQ_CONNECTION_STRING"); ok {
		QueueUrl = url
	} else {
		log.Error("rabbitmq connection string is not set using default url")
		return
	}

	// initializing database service.
	dataStore, err := database.New(DbUrl)
	if err != nil {
		log.Error("can't initialise database throws error", zap.Error(err))
		return
	}

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

	dataconsumer, err := services.NewConsumer(queueService, dataStore, memStore, encryptor, log)
	if err != nil {
		log.Error("can't initialise database throws error", zap.Error(err))
		return
	}

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

	// Initialize Gin router
	// router := gin.Default()
	// registerRoutes(router, ginController)

	// Start the HTTP server
	// httpPort := os.Getenv("HTTP_PORT")
	// if httpPort == "" {
	// 	httpPort = "8080"
	// 	log.Warn("HTTP_PORT is not set, using default port 8080")
	// }

	log.Info("starting HTTP server", zap.String("port", ginController.HttpPort))
	// err = router.Run(":" + httpPort)
	// if err != nil {
	// 	log.Error("failed to start HTTP server", zap.Error(err))
	// }
	go func() {
		defer wg.Done()
		log.Info("starting http server", zap.String("port", ginController.HttpPort))
		if err := ginController.Start(); err != nil {
			log.Error("cannot run http server", zap.Error(err), zap.String("port", ginController.HttpPort))
			panic(err)
		}
	}()

	log.Info("stopping consumer")
	wg.Wait()
}

// func registerRoutes(router *gin.Engine, ctl *controller.Controller) {
// 	router.GET("/users", ctl.GetAllUsers)
// 	router.GET("/users/:id", ctl.GetUser)
// 	router.POST("/users", ctl.CreateUser)
// 	router.DELETE("/users/:id", ctl.DeleteUser)
// 	router.GET("/users/sse", ctl.GetAllUsersSSE)
// 	router.Static("/static", "./client")
// }
