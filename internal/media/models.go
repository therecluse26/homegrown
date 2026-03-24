package media

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── Upload Context ───────────────────────────────────────────────────────────
// UploadContext determines validation rules (max size, allowed types).
// Maps to the CHECK constraint on media_uploads.context. [09-media §8.3]

type UploadContext string

const (
	UploadContextProfilePhoto       UploadContext = "profile_photo"
	UploadContextPostAttachment     UploadContext = "post_attachment"
	UploadContextMessageAttachment  UploadContext = "message_attachment"
	UploadContextActivityAttachment UploadContext = "activity_attachment"
	UploadContextJournalImage       UploadContext = "journal_image"
	UploadContextNatureJournalImage UploadContext = "nature_journal_image"
	UploadContextProjectAttachment  UploadContext = "project_attachment"
	UploadContextReadingCover       UploadContext = "reading_cover"
	UploadContextMarketplaceFile    UploadContext = "marketplace_file"
	UploadContextListingPreview     UploadContext = "listing_preview"
	UploadContextListingThumbnail   UploadContext = "listing_thumbnail"
	UploadContextCreatorLogo        UploadContext = "creator_logo"
	UploadContextDataExport         UploadContext = "data_export"
	UploadContextAudioAttachment    UploadContext = "audio_attachment"
	UploadContextVideoLesson        UploadContext = "video_lesson"
)

// ─── Upload Status ────────────────────────────────────────────────────────────
// UploadStatus maps to CHECK constraint on media_uploads.status. [09-media §3.1]

type UploadStatus string

const (
	UploadStatusPending     UploadStatus = "pending"
	UploadStatusUploaded    UploadStatus = "uploaded"
	UploadStatusProcessing  UploadStatus = "processing"
	UploadStatusPublished   UploadStatus = "published"
	UploadStatusQuarantined UploadStatus = "quarantined"
	UploadStatusRejected    UploadStatus = "rejected"
	UploadStatusFlagged     UploadStatus = "flagged"
	UploadStatusExpired     UploadStatus = "expired"
	UploadStatusDeleted     UploadStatus = "deleted"
)

// ─── Image Variant ────────────────────────────────────────────────────────────
// ImageVariant identifies image variant types. [09-media §10.7]

type ImageVariant int

const (
	// ImageVariantThumb is 200x200 max, fit-within.
	ImageVariantThumb ImageVariant = iota
	// ImageVariantMedium is 800x800 max, fit-within.
	ImageVariantMedium
)

// ─── GORM Models ──────────────────────────────────────────────────────────────

// Upload maps to the media_uploads table. [09-media §3.2]
type Upload struct {
	ID               uuid.UUID        `gorm:"type:uuid;primaryKey"`
	FamilyID         uuid.UUID        `gorm:"type:uuid;not null"`
	UploadedBy       uuid.UUID        `gorm:"type:uuid;not null"`
	Context          UploadContext     `gorm:"type:text;not null"`
	Status           UploadStatus     `gorm:"type:text;not null;default:pending"`
	OriginalFilename string           `gorm:"type:text;not null"`
	ContentType      string           `gorm:"type:text;not null"`
	SizeBytes        *int64           `gorm:"type:bigint"`
	StorageKey       string           `gorm:"type:text;not null;uniqueIndex"`
	HasThumb         bool             `gorm:"not null;default:false"`
	HasMedium        bool             `gorm:"not null;default:false"`
	ProbeMetadata    *json.RawMessage `gorm:"type:jsonb"`
	OriginalSizeBytes *int64          `gorm:"type:bigint"`
	WasCompressed    bool             `gorm:"not null;default:false"`
	ModerationLabels *json.RawMessage `gorm:"type:jsonb"`
	LastCSAMScannedAt *time.Time      `gorm:"type:timestamptz"`
	ExpiresAt        *time.Time       `gorm:"type:timestamptz"`
	PublishedAt      *time.Time       `gorm:"type:timestamptz"`
	CreatedAt        time.Time        `gorm:"type:timestamptz;not null"`
	UpdatedAt        time.Time        `gorm:"type:timestamptz;not null"`
}

func (Upload) TableName() string { return "media_uploads" }

func (u *Upload) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

// ProcessingJob maps to the media_processing_jobs table. [09-media §3.2]
type ProcessingJob struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UploadID     uuid.UUID  `gorm:"type:uuid;not null"`
	JobType      string     `gorm:"type:text;not null"`
	Status       string     `gorm:"type:text;not null;default:queued"`
	ErrorMessage *string    `gorm:"type:text"`
	Attempts     int        `gorm:"not null;default:0"`
	MaxAttempts  int        `gorm:"not null;default:3"`
	StartedAt    *time.Time `gorm:"type:timestamptz"`
	CompletedAt  *time.Time `gorm:"type:timestamptz"`
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null"`
}

func (ProcessingJob) TableName() string { return "media_processing_jobs" }

func (j *ProcessingJob) BeforeCreate(_ *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

// TranscodeJob maps to the media_transcode_jobs table. [09-media §3.2]
type TranscodeJob struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey"`
	UploadID        uuid.UUID        `gorm:"type:uuid;not null"`
	Status          string           `gorm:"type:text;not null;default:pending"`
	InputKey        string           `gorm:"type:text;not null"`
	OutputKeys      *json.RawMessage `gorm:"type:jsonb"`
	Resolutions     json.RawMessage  `gorm:"type:jsonb;not null"`
	DurationSeconds *int             `gorm:"type:integer"`
	ErrorMessage    *string          `gorm:"type:text"`
	StartedAt       *time.Time       `gorm:"type:timestamptz"`
	CompletedAt     *time.Time       `gorm:"type:timestamptz"`
	CreatedAt       time.Time        `gorm:"type:timestamptz;not null"`
	UpdatedAt       time.Time        `gorm:"type:timestamptz;not null"`
}

func (TranscodeJob) TableName() string { return "media_transcode_jobs" }

func (j *TranscodeJob) BeforeCreate(_ *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

// ─── Request Types ────────────────────────────────────────────────────────────

// RequestUploadCommand is the HTTP request body for POST /v1/media/uploads. [09-media §8.1]
type RequestUploadCommand struct {
	Context     UploadContext `json:"context" validate:"required"`
	ContentType string        `json:"content_type" validate:"required,min=1,max=255"`
	Filename    string        `json:"filename" validate:"required,min=1,max=255"`
	SizeBytes   uint64        `json:"size_bytes" validate:"required"`
}

// RequestUploadInput is the internal input struct passed to MediaService.RequestUpload().
// Includes auth context fields not present in the HTTP body. [09-media §8.1]
type RequestUploadInput struct {
	FamilyID    uuid.UUID
	UploadedBy  uuid.UUID
	Context     UploadContext
	ContentType string
	Filename    string
	SizeBytes   uint64
}

// ─── Response Types ───────────────────────────────────────────────────────────

// UploadResponse is the response from RequestUpload. [09-media §8.2]
type UploadResponse struct {
	UploadID         uuid.UUID `json:"upload_id"`
	PresignedURL     string    `json:"presigned_url"`
	StorageKey       string    `json:"storage_key"`
	ExpiresInSeconds uint32    `json:"expires_in_seconds"`
}

// UploadInfo contains upload information — returned by confirm and get endpoints. [09-media §8.2]
type UploadInfo struct {
	UploadID         uuid.UUID   `json:"upload_id"`
	Status           string      `json:"status"`
	Context          string      `json:"context"`
	ContentType      string      `json:"content_type"`
	OriginalFilename string      `json:"original_filename"`
	SizeBytes        *int64      `json:"size_bytes"`
	URLs             *UploadURLs `json:"urls"`
	HasThumb         bool        `json:"has_thumb"`
	HasMedium        bool        `json:"has_medium"`
	CreatedAt        time.Time   `json:"created_at"`
	PublishedAt      *time.Time  `json:"published_at,omitempty"`
}

// UploadURLs contains URLs for accessing the upload and its variants. [09-media §8.2]
type UploadURLs struct {
	Original string  `json:"original"`
	Thumb    *string `json:"thumb,omitempty"`
	Medium   *string `json:"medium,omitempty"`
}

// ─── Repository Input Types ───────────────────────────────────────────────────

// CreateUploadRow is the input for creating a new upload record.
type CreateUploadRow struct {
	ID               uuid.UUID
	FamilyID         uuid.UUID
	UploadedBy       uuid.UUID
	Context          UploadContext
	ContentType      string
	OriginalFilename string
	StorageKey       string
	ExpiresAt        time.Time
}

// UploadStatusUpdate holds optional fields to set when updating upload status.
type UploadStatusUpdate struct {
	SizeBytes   *int64
	PublishedAt *time.Time
}

// ─── Adapter Types ────────────────────────────────────────────────────────────

// CSAMScanResult is the CSAM scan result from Thorn Safer. [09-media §8.3]
type CSAMScanResult struct {
	IsCSAM          bool
	Hash            *string
	Confidence      *float64
	MatchedDatabase *string
}

// ModerationResult is the content moderation result from Rekognition. [09-media §8.3]
type ModerationResult struct {
	HasViolations bool
	AutoReject    bool
	Labels        []ModerationLabel
	Priority      *string
}

// ModerationLabel represents a single moderation label from Rekognition. [09-media §8.3]
type ModerationLabel struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	ParentName *string `json:"parent_name,omitempty"`
}

// ObjectMetadata is returned by HEAD request. [09-media §8.3]
type ObjectMetadata struct {
	ContentLength uint64
	ContentType   *string
}

// StorageError represents adapter error types. [09-media §8.3]
type StorageError struct {
	Code    string
	Message string
}

func (e *StorageError) Error() string { return e.Message }

var (
	ErrObjectNotFoundInS3 = &StorageError{Code: "not_found", Message: "object not found"}
	ErrS3OperationFailed  = &StorageError{Code: "operation_failed", Message: "S3 operation failed"}
	ErrPresignFailed      = &StorageError{Code: "presign_failed", Message: "presigned URL generation failed"}
)

// ScanError represents safety scan error types. [09-media §8.3]
type ScanError struct {
	Code    string
	Message string
}

func (e *ScanError) Error() string { return e.Message }

var (
	ErrScanUnavailable = &ScanError{Code: "unavailable", Message: "scan service unavailable"}
	ErrScanFailed      = &ScanError{Code: "failed", Message: "scan failed"}
)

// ─── Config ───────────────────────────────────────────────────────────────────

// MediaConfig holds runtime configuration for the media domain. [09-media §5.1]
type MediaConfig struct {
	PublicURLBase           string
	PresignedUploadExpiry   uint32 // seconds, default 3600
	PresignedDownloadExpiry uint32 // seconds, default 3600
}
