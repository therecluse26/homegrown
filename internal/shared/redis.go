package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"github.com/redis/go-redis/v9"
)

// CreateRedisClient creates a Redis client and validates connectivity via PING. [§10.1]
func CreateRedisClient(ctx context.Context, cfg *config.AppConfig) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Validate connectivity with PING
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return client, nil
}

// ─── Generic Helpers ─────────────────────────────────────────────────────────

// RedisGet retrieves a JSON-serialized value from Redis.
// Returns nil, nil if the key does not exist (cache miss).
func RedisGet[T any](ctx context.Context, client *redis.Client, key string) (*T, error) {
	val, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, ErrInternal(fmt.Errorf("redis get: %w", err))
	}

	var result T
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, ErrInternal(fmt.Errorf("redis unmarshal: %w", err))
	}
	return &result, nil
}

// RedisSet stores a JSON-serialized value in Redis with a TTL.
func RedisSet[T any](ctx context.Context, client *redis.Client, key string, value T, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return ErrInternal(fmt.Errorf("redis marshal: %w", err))
	}
	if err := client.Set(ctx, key, string(data), ttl).Err(); err != nil {
		return ErrInternal(fmt.Errorf("redis set: %w", err))
	}
	return nil
}

// RedisDelete removes a key from Redis.
func RedisDelete(ctx context.Context, client *redis.Client, key string) error {
	if err := client.Del(ctx, key).Err(); err != nil {
		return ErrInternal(fmt.Errorf("redis del: %w", err))
	}
	return nil
}

// ─── Rate Limiting ────────────────────────────────────────────────────────────

// RedisIncrementWithExpiry increments a counter and sets expiry on first increment.
// Returns the new counter value. Used by rate limiting middleware. [§10.3]
func RedisIncrementWithExpiry(ctx context.Context, client *redis.Client, key string, window time.Duration) (int64, error) {
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return 0, ErrInternal(fmt.Errorf("redis incr: %w", err))
	}

	if count == 1 {
		// First request in window — set expiry so the key self-cleans.
		if err := client.Expire(ctx, key, window).Err(); err != nil {
			return 0, ErrInternal(fmt.Errorf("redis expire: %w", err))
		}
	}

	return count, nil
}
