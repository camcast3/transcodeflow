FROM golang:1.20-alpine AS builder

WORKDIR /app

COPY . .
RUN go mod download
RUN go build -o /transcodeflow ./cmd/transcodeflow

FROM alpine:3.17

WORKDIR /

COPY --from=builder /transcodeflow /transcodeflow

# Default to server mode
ENV APP_MODE=server

ENTRYPOINT ["/transcodeflow"]
