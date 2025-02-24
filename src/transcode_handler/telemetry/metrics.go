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

// Counter is an interface representing a counter.
type Counter interface {
	Inc()
	Add(value float64)
}

// Metrics holds all the Prometheus metrics for the application
type Metrics struct {
	QueuePushCounter     *prometheus.CounterVec
	ServerRequestCounter *prometheus.CounterVec
}

// NewMetrics initializes and registers Prometheus metrics
func NewMetrics() (*Metrics, error) {
	metrics := &Metrics{
		QueuePushCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "transcoding_jobs_total",
				Help: "Total number of queue elements submitted",
			},
			[]string{"submitted"},
		),
		ServerRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "server_request_total",
				Help: "Total number of server requests",
			},
			[]string{"status"},
		),
	}

	// Register metrics
	if err := prometheus.Register(metrics.QueuePushCounter); err != nil {
		Logger.Error("Failed to register QueuePushCounter", zap.Error(err))
		return nil, err
	}
	if err := prometheus.Register(metrics.ServerRequestCounter); err != nil {
		Logger.Error("Failed to register ServerRequestCounter", zap.Error(err))
		return nil, err
	}

	return metrics, nil
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
