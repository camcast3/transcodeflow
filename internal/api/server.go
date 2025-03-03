package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"
	"transcodeflow/internal/model"
	"transcodeflow/internal/service"
	"transcodeflow/internal/telemetry"

	"go.uber.org/zap"
)

// Server encapsulates the HTTP server functionality
type Server struct {
	services *service.Services
	port     string
	server   *http.Server
}

// NewServer creates a new API server with the provided services
func NewServer(svc *service.Services) *Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Server{
		services: svc,
		port:     port,
	}
}

// Start initializes routes and starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// Create router and register routes
	mux := http.NewServeMux()
	mux.HandleFunc("/submit", s.handleSubmitJob)

	// Create server with context support
	s.server = &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to capture server errors
	errCh := make(chan error, 1)

	// Start server in a separate goroutine
	go func() {
		telemetry.Logger.Info("Starting server", zap.String("port", s.port))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		// Perform graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		telemetry.Logger.Info("Shutting down server gracefully")
		return s.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// handleSubmitJob processes job submission requests
func (s *Server) handleSubmitJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var job model.Job

	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		telemetry.Logger.Error("User error: Failed to decode job from request",
			zap.Any("request_body", r.Body), zap.Error(err))
		s.services.Metrics.IncrementServerRequestCounter("failed")
		http.Error(w, "Invalid job format", http.StatusBadRequest)
		return
	}

	// Convert job to JSON string
	jobBytes, err := json.Marshal(job)
	if err != nil {
		telemetry.Logger.Error("System error: Failed to marshal job into JSON string", zap.Error(err))
		s.services.Metrics.IncrementServerRequestCounter("failed")
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	jobStr := string(jobBytes)

	// Create a context for Redis operations
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Enqueue the job into Redis
	if err := s.services.Redis.EnqueueJob(ctx, jobStr); err != nil {
		telemetry.Logger.Error("System error: Failed to enqueue job", zap.Error(err))
		s.services.Metrics.IncrementServerRequestCounter("failed")
		http.Error(w, "Failed to enqueue job", http.StatusInternalServerError)
		return
	}

	telemetry.Logger.Info("Job submitted successfully",
		zap.String("input_file_path", job.InputFilePath),
		zap.String("output_file_path", job.OutputFilePath),
		zap.String("container_type", job.ContainerType),
		zap.String("flags", job.Flags),
	)

	s.services.Metrics.IncrementQueuePushCounter("job_pushed")
	s.services.Metrics.IncrementServerRequestCounter("success")
	w.WriteHeader(http.StatusAccepted)
}
