package redis

import (
	"context"
	"time"

	"transcodeflow/internal/telemetry"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type RedisClient interface {
	EnqueueJob(ctx context.Context, job string) error
	DequeueJob(ctx context.Context) (string, error)
	EnqueueJobResult(ctx context.Context, jobResult string) error
	Close() error
}

type DefaultRedisClient struct {
	client      *redis.Client
	jobQueue    string
	resultQueue string
}

func NewDefaultRedisClient() (*DefaultRedisClient, error) {
	options := &redis.Options{
		Addr: "redis:6379",
	}

	client := redis.NewClient(options)
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		telemetry.Logger.Error("System Error: Failed to connect to Redis", zap.Error(err))
		return nil, err
	}
	telemetry.Logger.Info("Connected to Redis")

	return &DefaultRedisClient{client: client, jobQueue: "jobs", resultQueue: "results"}, nil
}

// EnqueueJob pushes a job onto the Redis jobQueue, using LPUSH.
func (r *DefaultRedisClient) EnqueueJob(ctx context.Context, job string) error {
	return r.enqueue(ctx, r.jobQueue, job)
}

// EnqueueJobResult pushes a jobresult into the result queue
func (r *DefaultRedisClient) EnqueueJobResult(ctx context.Context, jobResult string) error {
	return r.enqueue(ctx, r.resultQueue, jobResult)
}

// enqueue pushes some generic thing onto a given queue using LPUSH
func (r *DefaultRedisClient) enqueue(ctx context.Context, queue string, obj string) error {
	err := r.client.LPush(ctx, queue, obj).Err()
	if err != nil {
		telemetry.Logger.Error("System Error: Failed to enqueue item in Redis", zap.String("queue", queue), zap.Error(err))
		return err
	}
	telemetry.Logger.Info("Item enqueued in Redis", zap.String("queue", queue))
	return nil
}

// DequeueJob pops a job from the Redis jobQueue, using BRPOP.
func (r *DefaultRedisClient) DequeueJob(ctx context.Context) (string, error) {
	//TODO config timeout
	res, err := r.client.BRPop(ctx, time.Second*30, r.jobQueue).Result()
	if err != nil {
		if err == redis.Nil {
			telemetry.Logger.Info("No job available in Redis queue", zap.String("queue", r.jobQueue))
			return "", nil
		}
		telemetry.Logger.Error("System Error: Failed to dequeue job from Redis", zap.String("queue", r.jobQueue), zap.Error(err))
		return "", err
	}
	telemetry.Logger.Info("Job dequeued from Redis", zap.String("queue", r.jobQueue))
	//TODO check on this usage of BRPOP result
	return res[1], nil
}

// Close closes the Redis client connection
func (r *DefaultRedisClient) Close() error {
	err := r.client.Close()
	if err != nil {
		telemetry.Logger.Error("System Error: Failed to close Redis client", zap.Error(err))
		return err
	}
	telemetry.Logger.Info("Redis client closed")
	return nil
}
