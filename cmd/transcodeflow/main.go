package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"transcodeflow/internal/api"
	"transcodeflow/internal/repository/redis"
	"transcodeflow/internal/service"
	"transcodeflow/internal/telemetry"
	"transcodeflow/internal/worker"

	"go.uber.org/zap"
)

func main() {
	// Initialize metrics
	metrics, err := telemetry.NewDefaultMetricsClient()
	if err != nil {
		telemetry.Logger.Fatal("Failed to initialize metrics", zap.Error(err))
	}

	// Initialize Redis client
	redisClient, err := redis.NewDefaultRedisClient()
	if err != nil {
		telemetry.Logger.Fatal("Failed to initialize Redis client", zap.Error(err))
	}
	defer redisClient.Close()

	// Create services container
	svc := service.NewServices(metrics, redisClient)

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Determine mode from environment variable
	mode := strings.ToLower(os.Getenv("APP_MODE"))
	if mode == "" {
		// Default to server mode if not specified
		mode = "server"
	}

	telemetry.Logger.Info("Starting application", zap.String("mode", mode))

	// Run in the appropriate mode
	switch mode {
	case "server", "api":
		// Create and start server
		server := api.NewServer(svc)
		if err := server.Start(ctx); err != nil {
			telemetry.Logger.Fatal("Server error", zap.Error(err))
		}
	case "worker":
		workerSvc := worker.NewWorkerService(svc)
		if err := workerSvc.Start(ctx); err != nil {
			telemetry.Logger.Fatal("Worker error", zap.Error(err))
		}
	default:
		telemetry.Logger.Fatal("Unknown application mode", zap.String("mode", mode))
	}
}
