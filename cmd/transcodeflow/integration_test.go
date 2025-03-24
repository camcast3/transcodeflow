package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
	"transcodeflow/internal/api"
	"transcodeflow/internal/model"
	"transcodeflow/internal/repository/redis"
	"transcodeflow/internal/service"
	"transcodeflow/internal/telemetry"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestServerIntegration(t *testing.T) {
	// Skip in CI environments
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Set environment variables for test
	os.Setenv("APP_MODE", "server")
	os.Setenv("PORT", "8082") // Use a different port for testing
	defer os.Unsetenv("APP_MODE")
	defer os.Unsetenv("PORT")

	// Initialize test dependencies
	metrics, err := telemetry.NewDefaultMetricsClient()
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	// Initialize Redis client
	redisClient, err := redis.NewDefaultRedisClient()
	if err != nil {
		t.Fatalf("Failed to initialize Redis client: %v", err)
	}
	defer redisClient.Close()

	// Create services container
	svc := service.NewServices(metrics, redisClient)

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in a goroutine
	go func() {
		if err := runServer(ctx, svc); err != nil {
			telemetry.Logger.Error("Server error", zap.Error(err))
		}
	}()

	// Wait for server to start
	time.Sleep(1 * time.Second)

	// Test server endpoints
	t.Run("SubmitJob", testSubmitJob)
	t.Run("MethodNotAllowed", testMethodNotAllowed)
	t.Run("InvalidJobFormat", testInvalidJobFormat)
	t.Run("MissingRequiredFields", testMissingRequiredFields)
}

func runServer(ctx context.Context, svc *service.Services) error {
	server := api.NewServer(svc)
	return server.Start(ctx)
}

func testSubmitJob(t *testing.T) {
	job := model.Job{
		InputFilePath:  "/path/to/input.mp4",
		OutputFilePath: "/path/to/output.mkv",
		SimpleOptions: &model.SimpleOptions{
			QualityPreset:           model.PresetBalanced,
			UseHardwareAcceleration: true,
			AudioQuality:            "medium",
		},
	}

	jobBytes, _ := json.Marshal(job)
	resp, err := http.Post("http://localhost:8082/submit", "application/json", bytes.NewBuffer(jobBytes))
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode, "Expected status accepted")
}

func testMethodNotAllowed(t *testing.T) {
	resp, err := http.Get("http://localhost:8082/submit")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "Expected method not allowed")
}

func testInvalidJobFormat(t *testing.T) {
	invalidJSON := []byte(`{"invalidField": true, "brokenJson":}`)
	resp, err := http.Post("http://localhost:8082/submit", "application/json", bytes.NewBuffer(invalidJSON))
	if err != nil {
		t.Fatalf("Failed to submit invalid JSON: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected bad request for invalid JSON")
}

func testMissingRequiredFields(t *testing.T) {
	job := model.Job{
		// Missing required fields
	}

	jobBytes, _ := json.Marshal(job)
	resp, err := http.Post("http://localhost:8082/submit", "application/json", bytes.NewBuffer(jobBytes))
	if err != nil {
		t.Fatalf("Failed to submit job with missing fields: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected bad request for missing fields")
}

func TestWorkerDequeue(t *testing.T) {
	os.Setenv("APP_MODE", "worker")
	defer os.Unsetenv("APP_MODE")

	go main()
	time.Sleep(time.Second * 15)
}
