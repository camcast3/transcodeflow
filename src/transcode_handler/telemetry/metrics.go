package telemetry

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Metrics holds all the Prometheus metrics for the application
type Metrics struct {
	JobCounter           *prometheus.CounterVec
	ServerRequestSucceed *prometheus.CounterVec
	ServerRequestFailed  *prometheus.CounterVec
	ServerTotalRequests  *prometheus.CounterVec
}

// NewMetrics initializes and registers Prometheus metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		JobCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "transcoding_jobs_total",
				Help: "Total number of transcoding jobs processed",
			},
			[]string{"status"},
		),
		ServerRequestSucceed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "server_request_succeed_total",
				Help: "Total number of successful server requests",
			},
			[]string{"endpoint"},
		),
		ServerRequestFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "server_request_failed_total",
				Help: "Total number of failed server requests",
			},
			[]string{"endpoint"},
		),
		ServerTotalRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "server_request_failed_total",
				Help: "Total number of server requests",
			},
			[]string{"endpoint"},
		),
	}

	// Register metrics
	prometheus.MustRegister(m.JobCounter)
	prometheus.MustRegister(m.ServerRequestSucceed)
	prometheus.MustRegister(m.ServerRequestFailed)

	// Register metrics
	prometheus.MustRegister(m.JobCounter)

	return m
}

// StartMetricsServer starts an HTTP server for exposing Prometheus metrics
func StartMetricsServer(port string, logger *zap.Logger) {
	http.Handle("/metrics", promhttp.Handler())
	logger.Info("Starting metrics server on port " + port + "...")

	server := &http.Server{Addr: ":" + port}

	// Create a channel to listen for interrupt or terminate signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start metrics server", zap.Error(err))
		}
	}()

	// Block until a signal is received
	<-stop

	// Create a context with a timeout for the shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to gracefully shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Failed to gracefully shutdown metrics server", zap.Error(err))
	}

	logger.Info("Metrics server gracefully stopped")
}
