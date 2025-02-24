package telemetry

import (
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type MetricsClient interface {
	IncrementQueuePushCounter(submitted string)
	IncrementServerRequestCounter(status string)
}

// Metrics holds all the Prometheus metrics for the application
type DefaultMetricsCleint struct {
	QueuePushCounter     *prometheus.CounterVec
	ServerRequestCounter *prometheus.CounterVec
}

// NewMetrics initializes and registers Prometheus metrics
func NewDefaultMetricsClient() (*DefaultMetricsCleint, error) {
	metrics := &DefaultMetricsCleint{
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
		Logger.Error("System error: Failed to register QueuePushCounter", zap.Error(err))
		return nil, err
	}
	if err := prometheus.Register(metrics.ServerRequestCounter); err != nil {
		Logger.Error("System error: Failed to register ServerRequestCounter", zap.Error(err))
		return nil, err
	}

	Logger.Info("Expected Metrics registered successfully")

	startMetricsServer(os.Getenv("9090"))

	return metrics, nil
}

// StartMetricsServer starts an HTTP server for exposing Prometheus metrics
func startMetricsServer(port string) {
	http.Handle("/metrics", promhttp.Handler())
	Logger.Info("Starting metrics server on port " + port + "...")

	server := &http.Server{Addr: ":" + port}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			Logger.Fatal("Failed to start metrics server", zap.Error(err))
		}
	}()
}

func (metricsClient *DefaultMetricsCleint) IncrementQueuePushCounter(submitted string) {
	metricsClient.QueuePushCounter.WithLabelValues(submitted).Inc()
}

func (metricsClient *DefaultMetricsCleint) IncrementServerRequestCounter(status string) {
	metricsClient.ServerRequestCounter.WithLabelValues(status).Inc()
}
