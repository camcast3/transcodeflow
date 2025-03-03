package service

import (
    "transcodeflow/internal/repository/redis"
    "transcodeflow/internal/telemetry"
)

// Services holds all application dependencies
type Services struct {
    Metrics telemetry.MetricsClient
    Redis   redis.RedisClient
}

// NewServices creates a new Services instance
func NewServices(metrics telemetry.MetricsClient, redisClient redis.RedisClient) *Services {
    return &Services{
        Metrics: metrics,
        Redis:   redisClient,
    }
}
