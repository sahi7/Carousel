package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis manages caching
type Redis struct {
	client *redis.Client
}

// NewRedis initializes a Redis client
func NewRedis(addr string) *Redis {
	return &Redis{
		client: redis.NewClient(&redis.Options{Addr: addr}),
	}
}

// Set stores a value in Redis
func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value from Redis
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// STREAMS
// XAdd adds a message to a Redis Stream
func (r *Redis) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
    return r.client.XAdd(ctx, &redis.XAddArgs{
        Stream: stream,
        Values: values,
    }).Result()
}

// XReadGroup reads messages from a Redis Stream using a consumer group
func (r *Redis) XReadGroup(ctx context.Context, group, consumer, stream string) ([]redis.XStream, error) {
    return r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
        Group:    group,
        Consumer: consumer,
        Streams:  []string{stream, ">"},
        Block:    0,
    }).Result()
}

// XAck acknowledges a message in a Redis Stream
func (r *Redis) XAck(ctx context.Context, stream, group, id string) error {
    return r.client.XAck(ctx, stream, group, id).Err()
}

// XTrimMaxLen trims a Redis Stream to a maximum length
func (r *Redis) XTrimMaxLen(ctx context.Context, stream string, maxLen int64) (int64, error) {
    return r.client.XTrimMaxLen(ctx, stream, maxLen).Result()
}

// CreateConsumerGroup creates a consumer group for a stream
func (r *Redis) CreateConsumerGroup(ctx context.Context, stream, group string) error {
    return r.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
}