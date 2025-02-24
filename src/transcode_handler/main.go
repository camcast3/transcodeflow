package main

import (
	"fmt"
	"os"
	"transcode_handler/clients/redis"
	"transcode_handler/services/server"
	"transcode_handler/telemetry"

	"go.uber.org/zap"
)

type Mode string

const (
	ServerMode      Mode = "server"
	TranscoderMode  Mode = "transcoder"
	HealthCheckMode Mode = "healthcheck"
	FileReplaceMode Mode = "filereplace"
)

func main() {
	mode := Mode(os.Getenv("MODE"))
	if mode == "" {
		panic("MODE environment variable must be set")
	}

	fmt.Println("Detected mode:", mode)

	// Initialize the Redis client
	redis := initializeClients()

	// Initialize the metrics server
	metrics := initializeTelemarty()

	switch mode {
	case ServerMode:
		// Start the metrics server
		server.Run(metrics, redis)
	case TranscoderMode:
		panic("MODE:transcoder Not Implemented Yet")
		// transcoder.Run()  // Implementation pending
	case HealthCheckMode:
		panic("MODE:healthcheck Not Implemented Yet")
		// healthcheck.Run() // Implementation pending
	case FileReplaceMode:
		panic("MODE:filereplace Not Implemented Yet")
		// filereplace.Run() // Implementation pending
	default:
		panic(fmt.Sprintf("Unknown mode: %s. Please use one of: server, transcoder, healthcheck, filereplace", mode))
	}
}

func initializeTelemarty() (metrics *telemetry.DefaultMetricsCleint) {
	metrics, err := telemetry.NewDefaultMetricsClient()
	if err != nil {
		telemetry.Logger.Error("System Error: Failed to initialize metrics", zap.Error(err))
		panic("Failed to initialize metrics: " + err.Error())
	}

	return
}

func initializeClients() (redisClient *redis.DefaultRedisClient) {
	redisClient, err := redis.NewDefaultRedisClient()
	if err != nil {
		telemetry.Logger.Error("System Error: Failed to initialize Redis client", zap.Error(err))
		panic("Failed to initialize Redis client" + err.Error())
	}

	return
}
