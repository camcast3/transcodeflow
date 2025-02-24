package telemetry

import (
	"go.uber.org/zap"
)

// Logger is the global logger instance
var Logger *zap.Logger

func init() {
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
	defer Logger.Sync() // flushes buffer, if any
}
