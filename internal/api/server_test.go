package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"transcodeflow/internal/model"
	"transcodeflow/internal/service"
	"transcodeflow/test/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestHandleSubmitJob tests the handleSubmitJob method using mockery-generated mocks
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
		InputContainerType:  "mp4", // Corrected from ContainerType
		OutputContainerType: "mp4",
		DryRun:              "false",
		GlobalArguments:     "--dry-run", // Corrected from Flags
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

	// Create GET request (not POST)
	req, err := http.NewRequest("GET", "/submit", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.handleSubmitJob(rr, req)

	// Should be method not allowed
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
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
