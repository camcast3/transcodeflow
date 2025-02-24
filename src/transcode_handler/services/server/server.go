package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"
	"transcode_handler/client/redis"
	"transcode_handler/telemetry"

	"go.uber.org/zap"
)

type Job struct {
	InputFilePath  string `json:"input_file_path"`
	OutputFilePath string `json:"output_file_path"`
	ContainerType  string `json:"container_type"`
	Flags          string `json:"flags"`
}

func submitJobHandler(w http.ResponseWriter, r *http.Request, metrics *telemetry.Metrics, redis *redis.DefaultRedisClient) {
	var job Job

	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		telemetry.Logger.Error("User error: Failed to decode job from request", zap.Any("request_body", r.Body), zap.Error(err))
		metrics.ServerRequestCounter.WithLabelValues("failed").Inc()
		http.Error(w, "Invalid job format", http.StatusBadRequest)
		return
	}

	// Convert job to JSON string
	jobBytes, err := json.Marshal(job)
	if err != nil {
		telemetry.Logger.Error("System error: Failed to marshal job into JSON string", zap.Error(err))
		metrics.ServerRequestCounter.WithLabelValues("failed").Inc()
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	jobStr := string(jobBytes)

	// Create a context for Redis operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Enqueue the job into Redis
	if err := redis.EnqueueJob(ctx, jobStr); err != nil {
		telemetry.Logger.Error("System error: Failed to enqueue job", zap.Error(err))
		metrics.ServerRequestCounter.WithLabelValues("failed").Inc()
		http.Error(w, "Failed to enqueue job", http.StatusInternalServerError)
		return
	}

	telemetry.Logger.Info("Job submitted successfully",
		zap.String("input_file_path", job.InputFilePath),
		zap.String("output_file_path", job.OutputFilePath),
		zap.String("container_type", job.ContainerType),
		zap.String("flags", job.Flags),
	)

	metrics.QueuePushCounter.WithLabelValues("job").Inc()
	metrics.ServerRequestCounter.WithLabelValues("success").Inc()
	w.WriteHeader(http.StatusAccepted)
}

func Run(metrics *telemetry.Metrics, redis *redis.DefaultRedisClient) {
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		submitJobHandler(w, r, metrics, redis)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	telemetry.Logger.Info("Starting server", zap.String("port", port))
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		telemetry.Logger.Error("System Error: Failed to start server", zap.Error(err))
	}
}
