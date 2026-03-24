package media

import "github.com/google/uuid"

// Domain events published by the media domain. [CODING §8.4, 09-media §14]
// All events implement shared.DomainEvent.
// Subscribers are registered in cmd/server/main.go via eventBus.Subscribe().

// UploadPublished is published after an upload passes all processing stages
// and transitions to published status.
// Subscribers:
//   - learn:: updates attachment references
//   - social:: updates post attachment status
//   - mkt:: updates listing file status
type UploadPublished struct {
	UploadID    uuid.UUID     `json:"upload_id"`
	FamilyID    uuid.UUID     `json:"family_id"`
	Context     UploadContext `json:"context"`
	StorageKey  string        `json:"storage_key"`
	ContentType string        `json:"content_type"`
	SizeBytes   int64         `json:"size_bytes"`
	HasThumb    bool          `json:"has_thumb"`
	HasMedium   bool          `json:"has_medium"`
}

func (UploadPublished) EventName() string { return "media.UploadPublished" }

// UploadQuarantined is published when CSAM is detected in an upload.
// Subscribers:
//   - safety:: initiates NCMEC report and account review
//   - notify:: (admin notification)
type UploadQuarantined struct {
	UploadID uuid.UUID     `json:"upload_id"`
	FamilyID uuid.UUID     `json:"family_id"`
	Context  UploadContext `json:"context"`
}

func (UploadQuarantined) EventName() string { return "media.UploadQuarantined" }

// UploadRejected is published when content moderation auto-rejects an upload.
// Subscribers:
//   - notify:: notifies family of rejection
//   - safety:: logs moderation action
type UploadRejected struct {
	UploadID uuid.UUID        `json:"upload_id"`
	FamilyID uuid.UUID        `json:"family_id"`
	Context  UploadContext     `json:"context"`
	Labels   []ModerationLabel `json:"labels"`
}

func (UploadRejected) EventName() string { return "media.UploadRejected" }

// UploadFlagged is published when content moderation flags an upload for admin review.
// Subscribers:
//   - safety:: creates moderation queue entry
//   - notify:: (admin notification)
type UploadFlagged struct {
	UploadID uuid.UUID        `json:"upload_id"`
	FamilyID uuid.UUID        `json:"family_id"`
	Context  UploadContext     `json:"context"`
	Labels   []ModerationLabel `json:"labels"`
	Priority *string          `json:"priority,omitempty"`
}

func (UploadFlagged) EventName() string { return "media.UploadFlagged" }
