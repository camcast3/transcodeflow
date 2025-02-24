package main

import (
	"fmt"
	"os"
	"transcode_handler/services/server"
	"transcode_handler/telemetry"
)

type Mode string

const (
	ServerMode      Mode = "server"
	TranscoderMode  Mode = "transcoder"
	HealthCheckMode Mode = "healthcheck"
	FileReplaceMode Mode = "filereplace"
)

func main() {
	mode := Mode(os.Getenv("MODE"))
	if mode == "" {
		panic("MODE environment variable must be set")
	}

	fmt.Println("Detected mode:", mode)

	switch mode {
	case ServerMode:
		// Start the metrics server
		metrics := telemetry.NewMetrics()
		telemetry.StartMetricsServer("9090", telemetry.Logger)
		server.Run(metrics)
	case TranscoderMode:
		panic("MODE:transcoder Not Implemented Yet")
		// transcoder.Run()  // Implementation pending
	case HealthCheckMode:
		panic("MODE:healthcheck Not Implemented Yet")
		// healthcheck.Run() // Implementation pending
	case FileReplaceMode:
		panic("MODE:filereplace Not Implemented Yet")
		// filereplace.Run() // Implementation pending
	default:
		panic(fmt.Sprintf("Unknown mode: %s. Please use one of: server, transcoder, healthcheck, filereplace", mode))
	}
}

func initializeTelemarty() {
	// Initialize metrics

}
