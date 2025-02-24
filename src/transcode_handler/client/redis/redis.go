package redis

import (
	"context"

	"transcode_handler/telemetry"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type RedisClientFactory interface {
	Create() RedisClient
}

type DefaultRedisClientFactory struct {
	client   *redis.Client
	jobQueue string
}

func NewDefaultRedisClientFactory() (*DefaultRedisClientFactory, error) {
	options := &redis.Options{
		Addr: "redis:6379",
	}

	client := redis.NewClient(options)
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		telemetry.Logger.Error("System Error: Failed to connect to Redis", zap.Error(err))
		return nil, err
	}

	jobQueue := "transcode:jobs"

	return &DefaultRedisClientFactory{client: client, jobQueue: jobQueue}, nil
}

func (f *DefaultRedisClientFactory) Create() *DefaultRedisClient {
	return &DefaultRedisClient{client: f.client, jobQueue: f.jobQueue}
}

type RedisClient interface {
	EnqueueJob(ctx context.Context, job string) error
	DequeueJob(ctx context.Context) (string, error)
	Close() error
}

type DefaultRedisClient struct {
	client   *redis.Client
	jobQueue string
}

// EnqueueJob pushes a job onto the Redis jobQueue, using LPUSH.
func (r *DefaultRedisClient) EnqueueJob(ctx context.Context, job string) error {
	err := r.client.LPush(ctx, r.jobQueue, job).Err()
	if err != nil {
		telemetry.Logger.Error("System Error: Failed to enqueue job in Redis", zap.String("queue", r.jobQueue), zap.Error(err))
		return err
	}
	telemetry.Logger.Info("Job enqueued in Redis", zap.String("queue", r.jobQueue))
	return nil
}

// DequeueJob pops a job from the Redis jobQueue, using RPOP.
func (r *DefaultRedisClient) DequeueJob(ctx context.Context) (string, error) {
	job, err := r.client.RPop(ctx, r.jobQueue).Result()
	if err != nil {
		if err == redis.Nil {
			telemetry.Logger.Info("No job available in Redis queue", zap.String("queue", r.jobQueue))
			return "", nil
		}
		telemetry.Logger.Error("System Error: Failed to dequeue job from Redis", zap.String("queue", r.jobQueue), zap.Error(err))
		return "", err
	}
	telemetry.Logger.Info("Job dequeued from Redis", zap.String("queue", r.jobQueue))
	return job, nil
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
