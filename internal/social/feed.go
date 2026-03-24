package social

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// FanOutPostPayload is the asynq job payload for fan-out-on-write feed distribution.
// When a post is created, this job is enqueued to add the post to each friend's
// Redis sorted set feed. [05-social §11]
type FanOutPostPayload struct {
	PostID   uuid.UUID `json:"post_id"`
	FamilyID uuid.UUID `json:"family_id"`
	ScoreMs  float64   `json:"score_ms"` // Unix milliseconds for sorted set score
}

func (FanOutPostPayload) TaskType() string { return "social:fan_out_post" }

// MaxFeedSize is the maximum number of entries kept per family's Redis feed.
// Older posts fall back to PostgreSQL query. [05-social §11]
const MaxFeedSize int64 = 500

// FeedRebuildPayload is the asynq job payload for backfilling a family's feed
// when a new friendship is accepted. [05-social §10]
type FeedRebuildPayload struct {
	FamilyID       uuid.UUID `json:"family_id"`
	FriendFamilyID uuid.UUID `json:"friend_family_id"`
}

func (FeedRebuildPayload) TaskType() string { return "social:feed_rebuild" }

// RegisterFeedWorkers registers asynq handlers for feed-related background jobs.
// Called from main.go during worker setup. [05-social §11]
func RegisterFeedWorkers(
	worker shared.JobWorker,
	feedStore shared.FeedStore,
	friendshipRepo FriendshipRepository,
	postRepo PostRepository,
) {
	// C1: Fan-out handler — distributes a new post to all friends' Redis feeds.
	worker.Handle("social:fan_out_post", func(ctx context.Context, payload []byte) error {
		var p FanOutPostPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			slog.Error("fan_out_post: unmarshal failed", "error", err)
			return err
		}

		friendIDs, err := friendshipRepo.ListFriendFamilyIDs(ctx, p.FamilyID)
		if err != nil {
			slog.Error("fan_out_post: list friends failed", "family_id", p.FamilyID, "error", err)
			return err
		}

		postIDStr := p.PostID.String()
		for _, friendID := range friendIDs {
			feedOwner := friendID.String()
			if addErr := feedStore.AddToFeed(ctx, feedOwner, postIDStr, p.ScoreMs); addErr != nil {
				slog.Debug("fan_out_post: add failed", "feed_owner", feedOwner, "error", addErr)
				continue
			}
			// Trim to keep feed bounded.
			if trimErr := feedStore.TrimFeed(ctx, feedOwner, MaxFeedSize); trimErr != nil {
				slog.Debug("fan_out_post: trim failed", "feed_owner", feedOwner, "error", trimErr)
			}
		}

		// Also add to author's own feed.
		authorFeed := p.FamilyID.String()
		if addErr := feedStore.AddToFeed(ctx, authorFeed, postIDStr, p.ScoreMs); addErr != nil {
			slog.Debug("fan_out_post: add to own feed failed", "error", addErr)
		}
		_ = feedStore.TrimFeed(ctx, authorFeed, MaxFeedSize)

		return nil
	})

	// C4: Feed rebuild handler — backfills recent posts between new friends.
	worker.Handle("social:feed_rebuild", func(ctx context.Context, payload []byte) error {
		var p FeedRebuildPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			slog.Error("feed_rebuild: unmarshal failed", "error", err)
			return err
		}

		// Get recent posts from the friend and add to family's feed.
		friendPosts, err := postRepo.ListByFamilyIDs(ctx, []uuid.UUID{p.FriendFamilyID}, 0, int(MaxFeedSize))
		if err != nil {
			slog.Error("feed_rebuild: list friend posts failed", "error", err)
			return err
		}
		feedOwner := p.FamilyID.String()
		for _, post := range friendPosts {
			if post.Visibility != "friends" {
				continue
			}
			_ = feedStore.AddToFeed(ctx, feedOwner, post.ID.String(), float64(post.CreatedAt.UnixMilli()))
		}
		_ = feedStore.TrimFeed(ctx, feedOwner, MaxFeedSize)

		return nil
	})
}
