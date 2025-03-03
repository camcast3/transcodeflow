cat > cmd/transcodeflow/main.go << 'EOL'
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
    redisClient, err := redis.NewClient()
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
        // Create and start worker
        w := worker.NewWorker(svc)
        w.Start(ctx)
        
    default:
        telemetry.Logger.Fatal("Unknown application mode", zap.String("mode", mode))
    }
}
EOL

cat > internal/service/services.go << 'EOL'
package service

import (
    "transcodeflow/internal/repository/redis"
    "transcodeflow/internal/telemetry"
)

// Services holds all application dependencies
type Services struct {
    Metrics telemetry.MetricsClient
    Redis   redis.RedisClient
}

// NewServices creates a new Services instance
func NewServices(metrics telemetry.MetricsClient, redisClient redis.RedisClient) *Services {
    return &Services{
        Metrics: metrics,
        Redis:   redisClient,
    }
}
EOL

cat > Makefile << 'EOL'
.PHONY: build test clean run-server run-worker all

all: build

build:
    go build -o bin/transcodeflow ./cmd/transcodeflow

test:
    go test ./...

run-server: build
    APP_MODE=server ./bin/transcodeflow

run-worker: build
    APP_MODE=worker ./bin/transcodeflow

clean:
    rm -rf bin/
EOL

cat > scripts/run-server.sh << 'EOL'
#!/bin/bash
export APP_MODE=server
go run ./cmd/transcodeflow/main.go
EOL

cat > scripts/run-worker.sh << 'EOL'
#!/bin/bash
export APP_MODE=worker
go run ./cmd/transcodeflow/main.go
EOL

# Make scripts executable
chmod +x scripts/run-server.sh scripts/run-worker.sh