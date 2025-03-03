package telemetry

import (
	"log"
	"os"

	"go.uber.org/zap"
)

// Logger is the global logger instance
var Logger *zap.Logger

func init() {
	// Ensure the logs directory exists
	if err := os.MkdirAll("./logs", 0755); err != nil {
		log.Fatalf("failed to create logs directory: %v", err)
	}

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		"stdout",
		"./logs/server.log",
	}
	config.ErrorOutputPaths = []string{
		"stderr",
		"./logs/server_error.log",
	}

	var err error
	Logger, err = config.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
}
