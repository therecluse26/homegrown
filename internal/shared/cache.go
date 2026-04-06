package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Cache is the generic caching port. No vendor types are referenced here — implementations
// (e.g. redisCache) live in the same package and are unexported. [ARCH §4.1]
type Cache interface {
	// Get retrieves a string value by key.
	// Returns ("", nil) on a cache miss — callers check for empty string to detect a miss.
	Get(ctx context.Context, key string) (string, error)

	// Set stores a string value with a TTL. Overwrites any existing value.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Delete removes a key. No error if the key does not exist.
	Delete(ctx context.Context, key string) error

	// IncrementWithExpiry atomically increments a counter and sets its expiry on the
	// first increment. Returns the new counter value. Used by rate limiting. [§13.2]
	IncrementWithExpiry(ctx context.Context, key string, window time.Duration) (int64, error)

	// Ping checks connectivity to the cache backend. Used by health checks. [P2-2]
	Ping(ctx context.Context) error

	// Close releases the underlying connection pool.
	Close() error
}

// ─── Generic Helpers ──────────────────────────────────────────────────────────
// Package-level generic functions are required because Go interfaces cannot have
// generic methods (type parameters are not allowed on interface method signatures).

// CacheGet retrieves a JSON-serialized value from the cache.
// Returns (nil, nil) on a cache miss.
func CacheGet[T any](ctx context.Context, c Cache, key string) (*T, error) {
	val, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if val == "" {
		// Cache miss — key not found.
		return nil, nil
	}

	var result T
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, ErrInternal(fmt.Errorf("cache unmarshal: %w", err))
	}
	return &result, nil
}

// CacheSet stores a JSON-serialized value in the cache with a TTL.
func CacheSet[T any](ctx context.Context, c Cache, key string, value T, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return ErrInternal(fmt.Errorf("cache marshal: %w", err))
	}
	return c.Set(ctx, key, string(data), ttl)
}

// CacheDelete removes a key from the cache.
func CacheDelete(ctx context.Context, c Cache, key string) error {
	return c.Delete(ctx, key)
}

// CacheIncrementWithExpiry atomically increments a rate-limit counter.
// Delegates directly to the Cache interface method.
func CacheIncrementWithExpiry(ctx context.Context, c Cache, key string, window time.Duration) (int64, error) {
	return c.IncrementWithExpiry(ctx, key, window)
}
