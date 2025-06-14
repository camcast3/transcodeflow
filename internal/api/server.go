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

	// Decode job from request body
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		// Check if the error is due to a type mismatch (e.g., complex128)
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			telemetry.Logger.Error("System error: Failed to decode job from request due to type mismatch",
				zap.Any("request_body", r.Body), zap.Error(err))
			s.services.Metrics.IncrementServerRequestCounter("failed")
			http.Error(w, "Server error", http.StatusInternalServerError)
		} else if _, ok := err.(*json.UnmarshalTypeError); ok {
			telemetry.Logger.Error("System error: Failed to decode job from request due to type mismatch",
				zap.Any("request_body", r.Body), zap.Error(err))
			s.services.Metrics.IncrementServerRequestCounter("failed")
			http.Error(w, "Server error", http.StatusInternalServerError)
		} else {
			telemetry.Logger.Error("User error: Failed to decode job from request",
				zap.Any("request_body", r.Body), zap.Error(err))
			s.services.Metrics.IncrementServerRequestCounter("failed")
			http.Error(w, "Invalid job format", http.StatusBadRequest)
		}
		return
	}

	// Validate required fields
	if job.InputFilePath == "" || job.OutputFilePath == "" {
		telemetry.Logger.Error("User error: Missing required job fields",
			zap.String("input_file_path", job.InputFilePath),
			zap.String("output_file_path", job.OutputFilePath))
		s.services.Metrics.IncrementServerRequestCounter("failed")
		http.Error(w, "Missing required job fields", http.StatusBadRequest)
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

	// Log job submission
	logJob(job)

	s.services.Metrics.IncrementQueuePushCounter("job_pushed")
	s.services.Metrics.IncrementServerRequestCounter("success")
	w.WriteHeader(http.StatusAccepted)
}

func logJob(job model.Job) {
	// Prepare safe values for potentially nil fields
	inputContainerType := job.InputContainerType
	outputContainerType := job.OutputContainerType
	hardwareDevice := job.HardwareDevice

	// Log job submission with appropriate fields based on job type
	if job.IsAdvancedMode() {
		// Advanced mode logging
		telemetry.Logger.Info("Advanced job submitted successfully",
			zap.String("input_file_path", job.InputFilePath),
			zap.String("output_file_path", job.OutputFilePath),
			zap.String("input_container_type", inputContainerType),
			zap.String("output_container_type", outputContainerType),
			zap.Bool("dry_run", job.IsDryRun()),
			zap.Bool("has_global_args", job.GlobalArguments != ""),
			zap.Bool("has_input_args", job.InputArguments != ""),
			zap.Bool("has_output_args", job.OutputArguments != ""),
			zap.String("hardware_device", hardwareDevice),
		)
	} else {
		// Simple options mode logging
		var preset string
		var hwAccel bool
		var audioQuality string
		var resolution string
		var keepOriginalResolution bool

		if job.SimpleOptions != nil {
			preset = string(job.SimpleOptions.QualityPreset)
			hwAccel = job.SimpleOptions.UseHardwareAcceleration
			audioQuality = job.SimpleOptions.AudioQuality
			resolution = job.SimpleOptions.Resolution
			keepOriginalResolution = job.SimpleOptions.KeepOriginalResolution
		} else {
			preset = string(model.DefaultQualityPreset)
			hwAccel = false
			audioQuality = "medium"
			resolution = ""
			keepOriginalResolution = false
		}

		telemetry.Logger.Info("Simple job submitted successfully",
			zap.String("input_file_path", job.InputFilePath),
			zap.String("output_file_path", job.OutputFilePath),
			zap.String("input_container_type", inputContainerType),
			zap.String("output_container_type", outputContainerType),
			zap.Bool("dry_run", job.IsDryRun()),
			zap.String("quality_preset", preset),
			zap.Bool("hardware_acceleration", hwAccel),
			zap.String("audio_quality", audioQuality),
			zap.String("resolution", resolution),
			zap.Bool("keep_original_resolution", keepOriginalResolution),
			zap.String("hardware_device", hardwareDevice),
		)
	}
}
