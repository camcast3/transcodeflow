package server

import (
	"net/http"
	"os"
	"transcode_handler/telemetry"

	"go.uber.org/zap"
)

type Job struct {
	InputFilePath  string `json:"input_file_path"`
	OutputFilePath string `json:"output_file_path"`
	ContainerType  string `json:"container_type"`
	Flags          string `json:"flags"`
}

func submitJobHandler(w http.ResponseWriter, r *http.Request) {
	var job Job

	// Enqueue the job into Redis
	/*if err := redis.EnqueueJob(job); err != nil {
		telemarty.Logger.WithError(err).Error("Failed to enqueue job")
		telemarty.Metrics.JobCounter.WithLabelValues("failed").Inc()
		http.Error(w, "Failed to enqueue job", http.StatusInternalServerError)
		return
	}*/

	/*telemarty.Logger.WithFields(.Fields{
		"input_file_path":  job.InputFilePath,
		"output_file_path": job.OutputFilePath,
		"container_type":   job.ContainerType,
		"flags":            job.Flags,
	}).Info("Job submitted successfully")
	telemarty.Metrics.JobCounter.WithLabelValues("success").Inc()*/

	w.WriteHeader(http.StatusAccepted)
}

func Run() {
	http.HandleFunc("/submit", submitJobHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	telemetry.Logger.Info("Starting server", zap.String("port", port))
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		telemetry.Logger.Error("Failed to start server", zap.Error(err))
	}
}
