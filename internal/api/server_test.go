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

	// Create a sample job payload
	job := model.Job{
		InputFilePath:  "input.mp4",
		OutputFilePath: "output.mp4",
		ContainerType:  "mp4",
		Flags:          "--dry-run",
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
	assert.Equal(t, http.StatusAccepted, rr.Code, "unexpected HTTP status code; response body: %s", rr.Body.String())

	// Assert that the expected calls on the mocks were made
	metricsMock.AssertExpectations(t)
	redisMock.AssertExpectations(t)
}

// Test for method not allowed
func TestHandleSubmitJobMethodNotAllowed(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Create services container with mocks
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	// Create the server instance with mock services
	server := NewServer(svc)

	// Create a GET request instead of POST
	req, err := http.NewRequest("GET", "/submit", nil)
	require.NoError(t, err, "failed to create HTTP request")

	rr := httptest.NewRecorder()

	// Call the handler
	server.handleSubmitJob(rr, req)

	// Should return method not allowed
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

// Test for invalid JSON
func TestHandleSubmitJobInvalidJSON(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Set expected behavior for failure case
	metricsMock.On("IncrementServerRequestCounter", "failed").Return()

	// Create services container with mocks
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	// Create the server instance with mock services
	server := NewServer(svc)

	// Create invalid JSON
	invalidJSON := []byte(`{"inputFilePath": "input.mp4", invalid json}`)

	req, err := http.NewRequest("POST", "/submit", bytes.NewBuffer(invalidJSON))
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	// Call the handler
	server.handleSubmitJob(rr, req)

	// Should return bad request
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	metricsMock.AssertExpectations(t)
}

// Test for Redis failure
func TestHandleSubmitJobRedisFailure(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Configure mocks for the Redis failure scenario
	metricsMock.On("IncrementServerRequestCounter", "failed").Return()
	redisMock.On("EnqueueJob", mock.Anything, mock.AnythingOfType("string")).Return(
		errors.New("redis connection error"),
	)

	// Create services container with mocks
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	// Create the server instance with mock services
	server := NewServer(svc)

	// Create a valid job
	job := model.Job{
		InputFilePath:  "input.mp4",
		OutputFilePath: "output.mp4",
	}
	jobJSON, _ := json.Marshal(job)

	req, _ := http.NewRequest("POST", "/submit", bytes.NewBuffer(jobJSON))
	rr := httptest.NewRecorder()

	// Call the handler
	server.handleSubmitJob(rr, req)

	// Should return internal server error
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	metricsMock.AssertExpectations(t)
	redisMock.AssertExpectations(t)
}
