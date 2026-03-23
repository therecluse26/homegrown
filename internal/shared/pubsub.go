package shared

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"github.com/redis/go-redis/v9"
)

// PubSub provides real-time message delivery via Redis pub/sub.
// Used by social:: WebSocket handler to push events to connected clients. [05-social §12]
// Channel pattern: "ws:parent:{parent_id}"
type PubSub interface {
	// Publish sends a message to the given channel.
	Publish(ctx context.Context, channel string, message []byte) error

	// Subscribe creates a subscription to the given channel.
	Subscribe(ctx context.Context, channel string) (Subscription, error)

	// Close releases the underlying connection.
	Close() error
}

// Subscription represents an active pub/sub subscription.
type Subscription interface {
	// Channel returns a receive-only channel of messages.
	Channel() <-chan []byte

	// Close unsubscribes and releases resources.
	Close() error
}

// NoopPubSub satisfies PubSub for tests and environments without Redis.
type NoopPubSub struct{}

func (NoopPubSub) Publish(_ context.Context, _ string, _ []byte) error { return nil }
func (NoopPubSub) Subscribe(_ context.Context, _ string) (Subscription, error) {
	return &noopSubscription{ch: make(chan []byte)}, nil
}
func (NoopPubSub) Close() error { return nil }

type noopSubscription struct {
	ch chan []byte
}

func (s *noopSubscription) Channel() <-chan []byte { return s.ch }
func (s *noopSubscription) Close() error           { close(s.ch); return nil }

// CreatePubSub creates a PubSub backed by Redis pub/sub.
func CreatePubSub(ctx context.Context, cfg *config.AppConfig) (PubSub, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL for pubsub: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed (pubsub): %w", err)
	}
	return &redisPubSub{client: client}, nil
}

// ─── redisPubSub ─────────────────────────────────────────────────────────────

type redisPubSub struct {
	client *redis.Client
}

func (r *redisPubSub) Publish(ctx context.Context, channel string, message []byte) error {
	if err := r.client.Publish(ctx, channel, message).Err(); err != nil {
		return ErrInternal(fmt.Errorf("pubsub publish: %w", err))
	}
	return nil
}

func (r *redisPubSub) Subscribe(ctx context.Context, channel string) (Subscription, error) {
	sub := r.client.Subscribe(ctx, channel)
	// Verify subscription is active.
	if _, err := sub.Receive(ctx); err != nil {
		return nil, ErrInternal(fmt.Errorf("pubsub subscribe: %w", err))
	}
	return &redisSubscription{sub: sub}, nil
}

func (r *redisPubSub) Close() error {
	return r.client.Close()
}

// ─── redisSubscription ───────────────────────────────────────────────────────

type redisSubscription struct {
	sub *redis.PubSub
}

func (s *redisSubscription) Channel() <-chan []byte {
	redisCh := s.sub.Channel()
	out := make(chan []byte, 64)
	go func() {
		defer close(out)
		for msg := range redisCh {
			out <- []byte(msg.Payload)
		}
	}()
	return out
}

func (s *redisSubscription) Close() error {
	return s.sub.Close()
}
