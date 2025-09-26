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