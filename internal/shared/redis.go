package shared

import (
	"context"
	"fmt"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"github.com/redis/go-redis/v9"
)

// CreateCache creates a Cache backed by Redis and validates connectivity via PING. [§10.1]
// The returned Cache interface hides all Redis types — callers import only shared.Cache.
func CreateCache(ctx context.Context, cfg *config.AppConfig) (Cache, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Validate connectivity with PING
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &redisCache{client: client}, nil
}

// ─── redisCache ───────────────────────────────────────────────────────────────
// redisCache is the unexported Redis implementation of Cache.
// The redis package is isolated here — it MUST NOT appear in any other file
// except cmd/server/main.go (which imports it only to call CreateCache). [ARCH §4.1]

type redisCache struct {
	client *redis.Client
}

func (r *redisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Cache miss — return empty string sentinel as documented by Cache.Get.
		return "", nil
	}
	if err != nil {
		return "", ErrInternal(fmt.Errorf("cache get: %w", err))
	}
	return val, nil
}

func (r *redisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return ErrInternal(fmt.Errorf("cache set: %w", err))
	}
	return nil
}

func (r *redisCache) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return ErrInternal(fmt.Errorf("cache del: %w", err))
	}
	return nil
}

func (r *redisCache) IncrementWithExpiry(ctx context.Context, key string, window time.Duration) (int64, error) {
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, ErrInternal(fmt.Errorf("cache incr: %w", err))
	}

	if count == 1 {
		// First request in window — set expiry so the key self-cleans.
		if err := r.client.Expire(ctx, key, window).Err(); err != nil {
			return 0, ErrInternal(fmt.Errorf("cache expire: %w", err))
		}
	}

	return count, nil
}

func (r *redisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *redisCache) Close() error {
	return r.client.Close()
}
