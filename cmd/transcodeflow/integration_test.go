package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
	"transcodeflow/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationServerMode_Submit_GetRequest(t *testing.T) {
	// Set the environment variables required by main().
	// MODE must be set to "server" and we set PORT to avoid conflicts.
	os.Setenv("MODE", "server")
	os.Setenv("PORT", "8082")
	defer os.Unsetenv("MODE")
	defer os.Unsetenv("PORT")

	// Start the application in a separate goroutine.
	// main() will block because server.Run listens indefinitely,
	// so running it in a goroutine allows the test to continue.
	go main()

	// Give the server some time to start up.
	time.Sleep(5 * time.Second)

	// In our server, GET requests to /submit are now handled.
	resp, err := http.Get("http://localhost:8082/submit")
	if err != nil {
		t.Fatalf("HTTP GET request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	// Since GET is now allowed on /submit, we expect a successful response status.
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 200, got %d. Response body: %s", resp.StatusCode, string(body))
	}
}

func TestIntegrationServerMode_Submit_PostRequest(t *testing.T) {
	// Set the environment variables required by main().
	os.Setenv("MODE", "server")
	os.Setenv("PORT", "8082")
	defer os.Unsetenv("MODE")
	defer os.Unsetenv("PORT")

	// Start the application in a separate goroutine.
	go main()

	// Allow time for the server to start.
	time.Sleep(5 * time.Second)

	// Create a sample job payload using the public model.
	job := model.Job{
		InputFilePath:  "input.mp4",
		OutputFilePath: "output.mp4",
		ContainerType:  "mp4",
		Flags:          "--dry-run",
	}

	payload, err := json.Marshal(job)
	require.NoError(t, err, "failed to marshal job payload")

	// Create an HTTP request with the JSON payload.
	req, err := http.NewRequest("POST", "http://localhost:8082/submit", bytes.NewBuffer(payload))
	require.NoError(t, err, "failed to create HTTP request")
	req.Header.Set("Content-Type", "application/json")

	// Use an HTTP client to send the request.
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "failed to perform POST request")
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")

	// Validate the HTTP response code.
	expectedStatus := http.StatusAccepted
	assert.Equal(t, expectedStatus, resp.StatusCode,
		"unexpected HTTP status code; expected %d, got %d. Response body: %s",
		expectedStatus, resp.StatusCode, string(respBody))
}
