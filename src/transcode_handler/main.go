package main

import (
	"fmt"
	"os"
	"transcode_handler/client/redis"
	"transcode_handler/services/server"
	"transcode_handler/telemetry"
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

	metrics := initializeTelemarty()
	redisClient := initializeClients()

	switch mode {
	case ServerMode:
		// Start the metrics server
		server.Run(metrics, redisClient)
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

func initializeTelemarty() (metrics *telemetry.Metrics) {
	metrics, err := telemetry.NewMetrics()
	if err != nil {
		panic("Failed to initialize metrics: " + err.Error())
	}
	telemetry.StartMetricsServer("9090", telemetry.Logger)
}

func initializeClients() (redis *redis.RedisClient) {
	redisClient := redis.NewRedisClient()
	if redisClient == nil {
		panic("Failed to initialize Redis client")
	}
}
