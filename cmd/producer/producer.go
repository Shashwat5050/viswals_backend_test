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
	defaultBatchSize = 5
	DevEnvironment   = "dev"
)

func main() {
	csvFilePath := flag.String("csv", "", "Path to CSV file")
	flag.Parse()

	if csvFilePath == nil || *csvFilePath == "" {
		fmt.Println("CSV file path is not found in -csv flag looking for env var")
		path, ok := os.LookupEnv("CSV_FILE_PATH")
		if !ok {
			fmt.Println("csv file path is not found in flag and env var,")
			fmt.Println("you can specify csv file with either -csv flag or providing CSV_FILE_PATH environment variable")
			return
		}
		csvFilePath = &path
	}

	// initialize the logger
	log, err := logger.Init(os.Stdout, strings.ToLower(os.Getenv("ENVIRONMENT")) == DevEnvironment)
	if err != nil {
		fmt.Println("Error initializing logger:", err)
		return
	}

	// open csv file as csv reader.
	csvReader, err := csvutils.OpenFile(*csvFilePath)
	if err != nil {
		log.Error("failed to open csv file ", zap.Error(err), zap.String("csvFilePath", *csvFilePath))
		return
	}

	// create connection with queue provider.
	queueConnection, ok := os.LookupEnv("RABBITMQ_CONNECTION_STRING")
	if !ok {
		log.Error("can't find rabbitmq connection string please provide environment variable RABBITMQ_CONNECTION_STRING")
		return
	}

	queueName, ok := os.LookupEnv("RABBITMQ_QUEUE_NAME")
	if !ok {
		log.Error("can't find queue name please provide environment variable RABBITMQ_QUEUE_NAME")
		return
	}

	queueService, err := rabbitmq.New(queueConnection, queueName)
	if err != nil {
		log.Error("failed to create rabbitmq queue", zap.Error(err), zap.String("queueName", queueName))
		return
	}

	// initializing producer service.
	producer := services.NewProducer(csvReader, queueService, log)
	defer func() {
		err := producer.Close()
		if err != nil {
			log.Error("failed to close rabbitmq producer", zap.Error(err), zap.String("queueName", queueName))
		}
	}()

	// start the producer service
	batchSize := defaultBatchSize
	batchSizeStr, ok := os.LookupEnv("BATCH_SIZE_PRODUCER")
	if !ok {
		log.Error(fmt.Sprintf("can't find batch size using default batch size of %v, you can provide environment variable BATCH_SIZE for custome batch size", defaultBatchSize))
	} else {
		size, err := strconv.Atoi(batchSizeStr)
		if err != nil {
			log.Error("batch size is not a number", zap.Error(err), zap.String("batchSizeStr", batchSizeStr))
			return
		}
		batchSize = size
	}

	log.Info("starting producer", zap.Int("batchSize", batchSize))

	err = producer.Start(batchSize)
	if err != nil {
		log.Error("failed to start producer", zap.Error(err), zap.String("queueName", queueName))
		return
	}

	log.Info("Producer has completed its work")
}
