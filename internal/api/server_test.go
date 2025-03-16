package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"transcodeflow/internal/model"
	"transcodeflow/internal/service"
	"transcodeflow/test/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestStart tests the server startup and graceful shutdown
func TestStart(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Create services container
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	// Find available port for test
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Set port using environment variable
	os.Setenv("PORT", fmt.Sprintf("%d", port))
	defer os.Unsetenv("PORT")

	// Create server
	server := NewServer(svc)

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Send a test request
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/", port))
	if err == nil {
		resp.Body.Close()
	}

	// Trigger graceful shutdown
	cancel()

	// Wait for server to shut down
	err = <-errCh
	assert.NoError(t, err, "Server should shutdown gracefully")
}

// TestStartServerError tests handling of server startup errors
func TestStartServerError(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Create services container
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	// Create server
	server := NewServer(svc)

	// Use context that's already cancelled to simulate immediate error
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Start server - should return right away due to cancelled context
	err := server.Start(ctx)
	assert.NoError(t, err, "Server should handle cancelled context gracefully")
}

// TestHandleSubmitJob tests the handleSubmitJob method - success case
func TestHandleSubmitJob(t *testing.T) {
	// Create mocks using mockery-generated constructors
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Set expected behavior on the metrics mock
	metricsMock.On("IncrementQueuePushCounter", "job_pushed").Return()
	metricsMock.On("IncrementServerRequestCounter", "success").Return()

	// Set expected behavior on the redis mock
	redisMock.On("EnqueueJob", mock.Anything, mock.AnythingOfType("string")).Return(nil)

	// Create services container with mocks
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	// Create the server instance with mock services
	server := NewServer(svc)

	// Create a sample job payload with CORRECT field names
	job := model.Job{
		InputFilePath:       "input.mp4",
		OutputFilePath:      "output.mp4",
		InputContainerType:  "mp4",
		OutputContainerType: "mp4",
		DryRun:              "false",
		GlobalArguments:     "--dry-run",
	}

	jobJSON, err := json.Marshal(job)
	require.NoError(t, err, "failed to marshal job payload")

	// Create an HTTP request with the job JSON payload
	req, err := http.NewRequest("POST", "/submit", bytes.NewBuffer(jobJSON))
	require.NoError(t, err, "failed to create HTTP request")

	// Create a ResponseRecorder to capture the response
	rr := httptest.NewRecorder()

	// Call the handler method directly on the server instance
	server.handleSubmitJob(rr, req)

	// Validate the HTTP response code
	assert.Equal(t, http.StatusAccepted, rr.Code, "expected HTTP 202 Accepted status")

	// Assert that the expected calls on the mocks were made
	metricsMock.AssertExpectations(t)
	redisMock.AssertExpectations(t)
}

// Test for invalid JSON request
func TestHandleSubmitJobInvalidJSON(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Set up expectations
	metricsMock.On("IncrementServerRequestCounter", "failed").Return()

	// Create services container
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	server := NewServer(svc)

	// Create invalid JSON request
	req, err := http.NewRequest("POST", "/submit", bytes.NewBufferString("{invalid json}"))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.handleSubmitJob(rr, req)

	// Should be bad request
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	metricsMock.AssertExpectations(t)
}

// Test for method not allowed
func TestHandleSubmitJobMethodNotAllowed(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Create services container
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	server := NewServer(svc)

	// Test several non-POST methods
	for _, method := range []string{"GET", "PUT", "DELETE", "PATCH"} {
		req, err := http.NewRequest(method, "/submit", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		server.handleSubmitJob(rr, req)

		// Should be method not allowed
		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code, "Method %s should not be allowed", method)
	}
}

// Test for Redis failure
func TestHandleSubmitJobRedisFailure(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Configure the Redis mock to return an error
	metricsMock.On("IncrementServerRequestCounter", "failed").Return()
	redisMock.On("EnqueueJob", mock.Anything, mock.AnythingOfType("string")).Return(
		errors.New("redis connection error"),
	)

	// Create services with mocks
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	server := NewServer(svc)

	// Create a valid job
	job := model.Job{
		InputFilePath:      "input.mp4",
		OutputFilePath:     "output.mp4",
		InputContainerType: "mp4",
	}

	jobJSON, _ := json.Marshal(job)
	req, _ := http.NewRequest("POST", "/submit", bytes.NewBuffer(jobJSON))
	rr := httptest.NewRecorder()

	// Call the handler
	server.handleSubmitJob(rr, req)

	// Should return internal server error
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	// Verify mock expectations
	metricsMock.AssertExpectations(t)
	redisMock.AssertExpectations(t)
}

// Test for empty input job fields (required fields validation)
func TestHandleSubmitJobEmptyRequiredFields(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Set expectations
	metricsMock.On("IncrementServerRequestCounter", "failed").Return()

	// Create services container
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	server := NewServer(svc)

	// Create job with empty required fields
	job := model.Job{
		// Missing InputFilePath
		OutputFilePath: "output.mp4",
	}

	jobJSON, _ := json.Marshal(job)
	req, _ := http.NewRequest("POST", "/submit", bytes.NewBuffer(jobJSON))
	rr := httptest.NewRecorder()

	// Call the handler
	server.handleSubmitJob(rr, req)

	// Should return bad request
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	metricsMock.AssertExpectations(t)
}
