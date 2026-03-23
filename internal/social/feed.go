package social

import "github.com/google/uuid"

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
// Used by the fan-out worker (social:fan_out_post asynq task handler).
const MaxFeedSize int64 = 500
