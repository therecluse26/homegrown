package shared

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── Pagination Params ────────────────────────────────────────────────────────

// PaginationParams holds cursor-based pagination parameters for list endpoints. [ARCH §1.4]
type PaginationParams struct {
	// Opaque cursor from previous response's NextCursor. Omit for the first page.
	Cursor *string `query:"cursor"`

	// Items per page. Default: 20. Max: 100.
	Limit *int `query:"limit"`
}

// EffectiveLimit returns the clamped limit value (1–100, default 20).
func (p PaginationParams) EffectiveLimit() int {
	if p.Limit == nil {
		return 20
	}
	limit := *p.Limit
	if limit > 100 {
		return 100
	}
	if limit < 1 {
		return 20
	}
	return limit
}

// ─── Paginated Response ───────────────────────────────────────────────────────

// PaginatedResponse wraps a page of results with cursor metadata.
type PaginatedResponse[T any] struct {
	Data       []T     `json:"data"`
	NextCursor *string `json:"next_cursor"` // nil if no more results
	HasMore    bool    `json:"has_more"`
}

// ─── Cursor Encoding ──────────────────────────────────────────────────────────

// EncodeCursor encodes an ID and timestamp into an opaque base64url cursor string.
// Format: base64url(id:timestamp_ms)
func EncodeCursor(id uuid.UUID, createdAt time.Time) string {
	raw := fmt.Sprintf("%s:%d", id.String(), createdAt.UnixMilli())
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes an opaque cursor string into an ID and timestamp.
// Returns ErrBadRequest on malformed input — clients must treat cursors as opaque.
func DecodeCursor(cursor string) (uuid.UUID, time.Time, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return uuid.Nil, time.Time{}, ErrBadRequest("invalid cursor")
	}

	raw := string(decoded)
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return uuid.Nil, time.Time{}, ErrBadRequest("invalid cursor")
	}

	id, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, time.Time{}, ErrBadRequest("invalid cursor")
	}

	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return uuid.Nil, time.Time{}, ErrBadRequest("invalid cursor")
	}

	createdAt := time.UnixMilli(ts)
	return id, createdAt, nil
}
