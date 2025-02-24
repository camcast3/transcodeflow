# Transcoding Pipeline Design Overview

# Transcoding Pipeline Design Overview

This document describes a scalable and maintainable transcoding pipeline that combines several components:
- A Redis queue for job management.
- A Go-based worker service for transcoding.
- Grafana Loki for centralized logging.

---

## 1. Job Submission

**API/CLI Interface:**  
Users or systems can submit transcoding jobs through a REST API or a CLI client.

**Job Parameters:**  
Each job submission includes:
- **Input File Path:** Full path to the file to be transcoded.
- **Output File Path:** Target location for the transcoded file.
- **Container Type:** File container type (e.g., `.mp4`, `.mkv`).
- **Flags:** Options such as a `dry-run` flag to test without actual transcoding.

**Enqueueing:**  
Submitted jobs are serialized (typically as JSON) and pushed to a Redis list or stream under a designated key (e.g., `transcode:jobs`).

---

## 2. Go Worker Service

**Redis Integration:**  
Multiple worker goroutines use blocking Redis commands (e.g., `BLPOP`) or process Redis streams to dequeue jobs atomically.

**Processing Steps:**
- **Transcoding:**  
  The worker invokes an external command (commonly `ffmpeg`) based on job parameters.
- **Follow-Up Tasks:**  
  After transcoding, a new job is pushed into a secondary Redis queue (e.g., `transcode:healthcheck`) for further processing like file health checking.

**Logging:**  
Each major step (job reception, processing start, errors, successes) is logged to standard output, with logs collected by container logging drivers.

---

## 3. Health Check and File Replacement Workers

**Health Check Service:**  
Dedicated workers listen to the `transcode:healthcheck` queue to:
- Validate the new file (integrity, file size, file usage).
- Publish a task to the `transcode:replace` queue upon successful validation.

**File Replacement Service:**  
This service listens on the `transcode:replace` queue to:
- Replace the original file with the validated file.
- Log the operation and send notifications when necessary.

---

## 4. Centralized Logging with Grafana Loki

**Log Aggregation:**  
All containerized components output logs to standard output.

**Promtail:**  
Running as a sidecar or daemon, Promtail tails container log files and forwards them to Grafana Loki.

**Grafana Visualization:**  
Grafana queries and visualizes logs stored in Loki, facilitating troubleshooting, performance monitoring, and real-time alerting.

**Benefits:**
- Advanced search and filtering.
- Centralized log storage and analysis.
- Immediate monitoring and alert configuration.

---

## 5. Example Docker Compose Extension for Grafana Loki

Use the following Docker Compose configuration snippet to set up your development environment:

```yaml
version: '3'
services:
  app:
    build:
      context: .
      dockerfile: .devcontainer/Dockerfile
    volumes:
      - .:/workspace:cached
    command: sleep infinity
    depends_on:
      - redis
      - loki

  redis:
    image: redis:6-alpine
    ports:
      - "6379:6379"

  loki:
    image: grafana/loki:2.8.2
    ports:
      - "3100:3100"
    command: -config.file=/etc/loki/local-config.yaml
    volumes:
      - ./loki-config.yaml:/etc/loki/local-config.yaml

  promtail:
    image: grafana/promtail:2.8.2
    volumes:
      - /var/log:/var/log
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - ./promtail-config.yaml:/etc/promtail/config.yaml
    command: -config.file=/etc/promtail/config.yaml
```

---

## 6. Repository Structure

A suggested repository layout for this transcoding pipeline:

```
/workspace
├── cmd
│   └── transcoding
│       └── main.go         // Single entry point that handles subcommands for worker, health check, and file replacement.
├── internal
│   ├── config
│   │   └── config.go       // Configuration loader for services.
│   ├── redis
│   │   └── client.go       // Redis operations (enqueueing/dequeuing).
│   ├── transcoder
│   │   └── transcoder.go   // Logic for invoking ffmpeg.
│   ├── logging
│   │   └── logger.go       // Centralized logging setup (Grafana Loki).
│   └── server
│       └── server.go       // REST API/CLI interface for job submissions.
├── pkg
│   └── cli
│       └── client.go       // Optional CLI client for job submissions.
├── docker
│   ├── redis.conf          // Redis configuration (including ACLs).
│   ├── loki-config.yaml    // Grafana Loki configuration.
│   └── promtail-config.yaml// Promtail configuration.
├── .devcontainer
│   ├── devcontainer.json   // VS Code dev container configuration.
│   └── Dockerfile          // Dockerfile for Go service container.
├── docker-compose.yaml     // Docker Compose for app, Redis, Loki, and Promtail.
├── go.mod                  // Go module definition.
└── README.md               // Project overview and documentation.
```