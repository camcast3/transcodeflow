package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"transcode_handler/client/redis" // Import the Redis package (which now contains our mock)
	"transcode_handler/telemetry"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

func TestIntegerStuff(t *testing.T) {
	Convey("Given some integer with a starting value", t, func() {
		x := 1

		Convey("When the integer is incremented", func() {
			x++

			Convey("The value should be greater by one", func() {
				So(x, ShouldEqual, 2)
			})
		})
	})
}

func TestSubmitJobHandler(t *testing.T) {
	// Build metrics using our mock counter vector from mocks package.
	metrics := &telemetry.Metrics{
		ServerRequestCounter: mocks.NewMockCounterVec("server_requests_total", "Total number of server requests", []string{"status"}),
		QueuePushCounter:     mocks.NewMockCounterVec("queue_push_total", "Total number of jobs pushed to the queue", []string{"type"}),
	}
	// Use the mock Redis client from our redis package.
	redisClient := redis.NewMockRedisClient()

	tests := []struct {
		name           string
		job            Job
		expectedStatus int
	}{
		{
			name: "Valid job",
			job: Job{
				InputFilePath:  "input.mp4",
				OutputFilePath: "output.mp4",
				ContainerType:  "mp4",
				Flags:          "-vf scale=1280:720",
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "Invalid job format",
			job:            Job{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.job)
			req, err := http.NewRequest("POST", "/submit", bytes.NewBuffer(body))
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				submitJobHandler(w, r, metrics, redisClient)
			})

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestRun(t *testing.T) {
	metrics := &telemetry.Metrics{
		ServerRequestCounter: mocks.NewMockCounterVec("mock_server_requests_total", "Mock total number of server requests", []string{"status"}),
		QueuePushCounter:     mocks.NewMockCounterVec("mock_queue_push_total", "Mock total number of jobs pushed to the queue", []string{"type"}),
	}
	redisClient := redis.NewMockRedisClient()

	os.Setenv("PORT", "8081")
	defer os.Unsetenv("PORT")

	// Run the server in the background.
	go Run(metrics, redisClient)

	// Tiny sleep to let the server start.
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:8081/submit")
	assert.NoError(t, err)
	// Expecting NotFound because there's no route for GET /submit.
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
