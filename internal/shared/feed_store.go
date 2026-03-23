package shared

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"github.com/redis/go-redis/v9"
)

// FeedStore provides a sorted-set-backed timeline feed for each family.
// Keys follow the pattern "feed:{family_id}". Scores are Unix milliseconds
// for chronological ordering. Values are post ID strings. [05-social §11]
type FeedStore interface {
	// AddToFeed inserts a post ID into a family's feed with the given score (Unix ms).
	AddToFeed(ctx context.Context, familyID string, postID string, scoreMs float64) error

	// GetFeed returns post IDs from a family's feed in reverse-chronological order.
	// offset and limit implement cursor-based pagination.
	GetFeed(ctx context.Context, familyID string, offset, limit int64) ([]string, error)

	// RemoveFromFeed removes a specific post ID from a family's feed.
	RemoveFromFeed(ctx context.Context, familyID string, postID string) error

	// RemoveFromFeedByFamily removes all post IDs from a family's feed that belong to a specific author.
	// authorPostIDs is the list of post IDs to remove.
	RemoveFromFeedByFamily(ctx context.Context, feedOwnerID string, authorPostIDs []string) error

	// TrimFeed trims a family's feed to keep only the most recent maxSize entries.
	TrimFeed(ctx context.Context, familyID string, maxSize int64) error

	// FeedSize returns the number of entries in a family's feed.
	FeedSize(ctx context.Context, familyID string) (int64, error)

	// Close releases the underlying connection.
	Close() error
}

// NoopFeedStore satisfies FeedStore for tests and environments without Redis.
type NoopFeedStore struct{}

func (NoopFeedStore) AddToFeed(_ context.Context, _, _ string, _ float64) error       { return nil }
func (NoopFeedStore) GetFeed(_ context.Context, _ string, _, _ int64) ([]string, error) {
	return nil, nil
}
func (NoopFeedStore) RemoveFromFeed(_ context.Context, _, _ string) error               { return nil }
func (NoopFeedStore) RemoveFromFeedByFamily(_ context.Context, _ string, _ []string) error {
	return nil
}
func (NoopFeedStore) TrimFeed(_ context.Context, _ string, _ int64) error { return nil }
func (NoopFeedStore) FeedSize(_ context.Context, _ string) (int64, error) { return 0, nil }
func (NoopFeedStore) Close() error                                        { return nil }

// CreateFeedStore creates a FeedStore backed by Redis sorted sets.
func CreateFeedStore(ctx context.Context, cfg *config.AppConfig) (FeedStore, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL for feed store: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed (feed store): %w", err)
	}
	return &redisFeedStore{client: client}, nil
}

// ─── redisFeedStore ──────────────────────────────────────────────────────────

type redisFeedStore struct {
	client *redis.Client
}

func feedKey(familyID string) string {
	return "feed:" + familyID
}

func (r *redisFeedStore) AddToFeed(ctx context.Context, familyID string, postID string, scoreMs float64) error {
	if err := r.client.ZAdd(ctx, feedKey(familyID), redis.Z{
		Score:  scoreMs,
		Member: postID,
	}).Err(); err != nil {
		return ErrInternal(fmt.Errorf("feed zadd: %w", err))
	}
	return nil
}

func (r *redisFeedStore) GetFeed(ctx context.Context, familyID string, offset, limit int64) ([]string, error) {
	ids, err := r.client.ZRevRange(ctx, feedKey(familyID), offset, offset+limit-1).Result()
	if err != nil {
		return nil, ErrInternal(fmt.Errorf("feed zrevrange: %w", err))
	}
	return ids, nil
}

func (r *redisFeedStore) RemoveFromFeed(ctx context.Context, familyID string, postID string) error {
	if err := r.client.ZRem(ctx, feedKey(familyID), postID).Err(); err != nil {
		return ErrInternal(fmt.Errorf("feed zrem: %w", err))
	}
	return nil
}

func (r *redisFeedStore) RemoveFromFeedByFamily(ctx context.Context, feedOwnerID string, authorPostIDs []string) error {
	if len(authorPostIDs) == 0 {
		return nil
	}
	members := make([]any, len(authorPostIDs))
	for i, id := range authorPostIDs {
		members[i] = id
	}
	if err := r.client.ZRem(ctx, feedKey(feedOwnerID), members...).Err(); err != nil {
		return ErrInternal(fmt.Errorf("feed zrem batch: %w", err))
	}
	return nil
}

func (r *redisFeedStore) TrimFeed(ctx context.Context, familyID string, maxSize int64) error {
	// Keep only the top maxSize entries by removing everything ranked below.
	if err := r.client.ZRemRangeByRank(ctx, feedKey(familyID), 0, -maxSize-1).Err(); err != nil {
		return ErrInternal(fmt.Errorf("feed trim: %w", err))
	}
	return nil
}

func (r *redisFeedStore) FeedSize(ctx context.Context, familyID string) (int64, error) {
	size, err := r.client.ZCard(ctx, feedKey(familyID)).Result()
	if err != nil {
		return 0, ErrInternal(fmt.Errorf("feed zcard: %w", err))
	}
	return size, nil
}

func (r *redisFeedStore) Close() error {
	return r.client.Close()
}
