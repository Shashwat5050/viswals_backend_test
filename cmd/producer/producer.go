package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/viswals_backend_test/core/csvutils"
	"github.com/viswals_backend_test/core/logger"
	"github.com/viswals_backend_test/core/rabbitmq"
	"github.com/viswals_backend_test/internal/services"
	"go.uber.org/zap"
)

var (
	defaultChunkSize = 5
	DevelopmentMode  = "dev"
)

func main() {
	csvPath := flag.String("csv", "", "Path to the CSV file")
	flag.Parse()

	if csvPath == nil || *csvPath == "" {
		fmt.Println("CSV file path not provided in -csv flag. Checking environment variable...")
		filePath, exists := os.LookupEnv("CSV_FILE_PATH")
		if !exists {
			fmt.Println("CSV file path not found in flag or environment variable.")
			fmt.Println("Provide the CSV file path using the -csv flag or the CSV_FILE_PATH environment variable.")
			return
		}
		csvPath = &filePath
	}

	// Initialize the logger
	log, err := logger.Init(os.Stdout, strings.ToLower(os.Getenv("ENVIRONMENT")) == DevelopmentMode)
	if err != nil {
		fmt.Println("Logger initialization failed:", err)
		return
	}

	// Open the CSV file as a reader
	csvReader, err := csvutils.OpenFile(*csvPath)
	if err != nil {
		log.Error("Unable to open the CSV file", zap.Error(err), zap.String("csvPath", *csvPath))
		return
	}

	// Establish a connection with the message broker
	rabbitConnString := os.Getenv("RABBITMQ_CONNECTION_STRING")
	if rabbitConnString == "" {
		log.Error("Missing RabbitMQ connection string. Please set the RABBITMQ_CONNECTION_STRING environment variable.")
		return
	}

	rabbitQueueName := os.Getenv("RABBITMQ_QUEUE_NAME")
	if rabbitQueueName == "" {
		log.Error("Missing RabbitMQ queue name. Please set the RABBITMQ_QUEUE_NAME environment variable.")
		return
	}

	messageQueue, err := rabbitmq.New(rabbitConnString, rabbitQueueName)
	if err != nil {
		log.Error("Failed to initialize RabbitMQ queue", zap.Error(err), zap.String("queueName", rabbitQueueName))
		return
	}

	// Initialize the producer service
	dataProducer := services.NewProducer(csvReader, messageQueue, log)
	defer func() {
		err := dataProducer.Close()
		if err != nil {
			log.Error("Error closing RabbitMQ producer", zap.Error(err), zap.String("queueName", rabbitQueueName))
		}
	}()

	// Determine chunk size for processing
	chunkSize := defaultChunkSize
	chunkSizeEnv, exists := os.LookupEnv("BATCH_SIZE_PRODUCER")
	if !exists {
		log.Warn(fmt.Sprintf("Batch size not set. Using default chunk size of %d. Set the BATCH_SIZE_PRODUCER environment variable for customization.", defaultChunkSize))
	} else {
		size, err := strconv.Atoi(chunkSizeEnv)
		if err != nil {
			log.Error("Invalid batch size value. Ensure it is a number.", zap.Error(err), zap.String("batchSizeEnv", chunkSizeEnv))
			return
		}
		chunkSize = size
	}

	log.Info("Initializing the producer service", zap.Int("chunkSize", chunkSize))

	// Start the producer
	err = dataProducer.Start(chunkSize)
	if err != nil {
		log.Error("Producer service failed to start", zap.Error(err), zap.String("queueName", rabbitQueueName))
		return
	}

	log.Info("Producer operation completed successfully.")
}
