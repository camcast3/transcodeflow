package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"transcode_handler/mocks"
	"transcode_handler/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestSubmitJobHandler tests the submitJobHandler function (or SubmitJobHandler if exported)
// using the mockery-generated mocks for MetricsClient and RedisClient.
func TestSubmitJobHandler(t *testing.T) {
	// Create mocks using mockery-generated constructors.
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Set expected behavior on the metrics mock.
	metricsMock.On("IncrementQueuePushCounter", "job_pushed").Return()
	metricsMock.On("IncrementServerRequestCounter", "success").Return()

	// Set expected behavior on the redis mock.
	redisMock.On("EnqueueJob", mock.Anything, mock.AnythingOfType("string")).Return(nil)

	// Create a sample job payload.
	job := model.Job{
		InputFilePath:  "input.mp4",
		OutputFilePath: "output.mp4",
		ContainerType:  "mp4",
		Flags:          "--dry-run",
	}

	jobJSON, err := json.Marshal(job)
	require.NoError(t, err, "failed to marshal job payload")

	// Create an HTTP request with the job JSON payload.
	req, err := http.NewRequest("POST", "/submit", bytes.NewBuffer(jobJSON))
	require.NoError(t, err, "failed to create HTTP request")

	// Create a ResponseRecorder to capture the response.
	rr := httptest.NewRecorder()

	// Call the handler.
	submitJobHandler(rr, req, metricsMock, redisMock)

	// Validate the HTTP response code.
	assert.Equal(t, http.StatusAccepted, rr.Code, "unexpected HTTP status code; response body: %s", rr.Body.String())

	// Assert that the expected calls on the mocks were made.
	metricsMock.AssertExpectations(t)
	redisMock.AssertExpectations(t)
}
