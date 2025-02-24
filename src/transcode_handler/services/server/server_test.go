package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"transcode_handler/mocks"

	"github.com/stretchr/testify/mock"
)

// TestSubmitJobHandler tests the submitJobHandler function (or SubmitJobHandler if exported)
// using the mockery-generated mocks for MetricsClient and RedisClient.
func TestSubmitJobHandler(t *testing.T) {
	// Create mocks using mockery-generated constructors.
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Set expected behavior on the metrics mock.
	// We expect that when a valid job is submitted, the following calls are made.
	metricsMock.On("IncrementQueuePushCounter", "job_pushed").Return()
	metricsMock.On("IncrementServerRequestCounter", "success").Return()

	// Set expected behavior on the redis mock.
	// We expect that EnqueueJob is called with any context and a string payload,
	// and it should return nil error.
	redisMock.On("EnqueueJob", mock.Anything, mock.AnythingOfType("string")).Return(nil)

	// Create a sample job payload.
	job := Job{
		InputFilePath:  "input.mp4",
		OutputFilePath: "output.mp4",
		ContainerType:  "mp4",
		Flags:          "-vf scale=1280:720",
	}
	jobJSON, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("failed to marshal job: %v", err)
	}

	// Create an HTTP request with the job JSON payload.
	req, err := http.NewRequest("POST", "/submit", bytes.NewBuffer(jobJSON))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Create a ResponseRecorder to capture the response.
	rr := httptest.NewRecorder()

	// Call the handler.
	// Note: if submitJobHandler is unexported (lowercase), this test file must be in the same package.
	submitJobHandler(rr, req, metricsMock, redisMock)

	// Validate the HTTP response.
	if rr.Code != http.StatusAccepted {
		t.Errorf("expected HTTP status %d but got %d", http.StatusAccepted, rr.Code)
	}

	// Assert that the mocks' expectations were met.
	metricsMock.AssertExpectations(t)
	redisMock.AssertExpectations(t)
}
