package media

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Service Interface ────────────────────────────────────────────────────────

// MediaService is the authoritative media service interface consumed by all domains.
// Supersedes the MediaAdapter sketches in learn:: and mkt:: domain specs.
// All methods that access user data require family_id for family-scoping. [CODING §2.4, 09-media §5]
type MediaService interface {
	// ─── Commands ───────────────────────────────────────────────────────

	// RequestUpload generates a presigned upload URL for direct client → S3 upload.
	RequestUpload(ctx context.Context, input *RequestUploadInput) (*UploadResponse, error)

	// ConfirmUpload confirms that the client has completed a direct upload.
	ConfirmUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) (*UploadInfo, error)

	// DeleteUpload soft-deletes an upload. (Phase 2)
	DeleteUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) error

	// ─── Queries ────────────────────────────────────────────────────────

	// GetUpload returns upload status and URLs (original + variants).
	GetUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) (*UploadInfo, error)

	// PresignedGet generates a presigned GET URL for secure file download.
	PresignedGet(ctx context.Context, storageKey string, expiresSeconds uint32) (string, error)

	// ValidateAttachment validates attachment metadata against context-based rules.
	ValidateAttachment(ctx context.Context, uploadCtx UploadContext, contentType string, sizeBytes uint64) error
}

// ─── Repository Interfaces ────────────────────────────────────────────────────

// UploadRepository defines persistence operations for media_uploads.
// All user-data queries are family-scoped via FamilyScope parameter. [CODING §2.4, 09-media §6.1]
type UploadRepository interface {
	// Create creates a new upload record in pending status.
	Create(ctx context.Context, scope shared.FamilyScope, input *CreateUploadRow) (*Upload, error)

	// FindByID finds an upload by ID, scoped to family.
	FindByID(ctx context.Context, scope shared.FamilyScope, uploadID uuid.UUID) (*Upload, error)

	// FindByIDUnscoped finds an upload by ID without family scope (for background jobs).
	FindByIDUnscoped(ctx context.Context, uploadID uuid.UUID) (*Upload, error)

	// UpdateStatus updates upload status and optional fields.
	UpdateStatus(ctx context.Context, uploadID uuid.UUID, status UploadStatus, updates *UploadStatusUpdate) (*Upload, error)

	// UpdateProbeMetadata updates probe metadata and compression info after processing.
	UpdateProbeMetadata(ctx context.Context, uploadID uuid.UUID, probe json.RawMessage, wasCompressed bool, originalSizeBytes *int64) error

	// SetVariantFlags sets variant flags after image processing.
	SetVariantFlags(ctx context.Context, uploadID uuid.UUID, hasThumb bool, hasMedium bool) error

	// SetModerationLabels sets moderation labels (from Rekognition results).
	SetModerationLabels(ctx context.Context, uploadID uuid.UUID, labels json.RawMessage) error

	// FindExpiredPending finds expired pending uploads for orphan cleanup.
	FindExpiredPending(ctx context.Context, before time.Time, limit uint32) ([]Upload, error)
}

// ProcessingJobRepository defines persistence operations for media_processing_jobs.
// System-internal — no family-scoping needed. [09-media §6.2]
type ProcessingJobRepository interface {
	// Create creates a new processing job record.
	Create(ctx context.Context, uploadID uuid.UUID, jobType string) (*ProcessingJob, error)

	// MarkRunning marks a job as running.
	MarkRunning(ctx context.Context, jobID uuid.UUID) error

	// MarkCompleted marks a job as completed.
	MarkCompleted(ctx context.Context, jobID uuid.UUID) error

	// MarkFailed marks a job as failed with an error message.
	MarkFailed(ctx context.Context, jobID uuid.UUID, errorMessage string) error

	// FindRetryable finds jobs eligible for retry (failed, attempts < max_attempts).
	FindRetryable(ctx context.Context, limit uint32) ([]ProcessingJob, error)
}

// ─── Adapter Interfaces ───────────────────────────────────────────────────────

// ObjectStorageAdapter defines the S3-compatible object storage interface.
// Provider-agnostic — works with R2, S3, MinIO, etc. [ARCH §2.10, 09-media §7.1]
type ObjectStorageAdapter interface {
	// PresignedPut generates a presigned PUT URL for direct client upload.
	PresignedPut(ctx context.Context, key string, maxSizeBytes uint64, contentType string, expiresSeconds uint32) (string, error)

	// PresignedGet generates a presigned GET URL for secure file download.
	PresignedGet(ctx context.Context, key string, expiresSeconds uint32) (string, error)

	// PutObject uploads data from the server (for generated variants).
	PutObject(ctx context.Context, key string, data []byte, contentType string) error

	// GetObjectHead returns object metadata without downloading.
	GetObjectHead(ctx context.Context, key string) (*ObjectMetadata, error)

	// GetObjectBytes reads a byte range from an object.
	GetObjectBytes(ctx context.Context, key string, start uint64, end uint64) ([]byte, error)

	// DeleteObject deletes an object from storage.
	DeleteObject(ctx context.Context, key string) error
}

// SafetyScanAdapter defines the safety scanning interface — delegates to safety:: domain. [09-media §7.2]
type SafetyScanAdapter interface {
	// ScanCSAM scans for CSAM using Thorn Safer (PhotoDNA hash matching).
	ScanCSAM(ctx context.Context, storageKey string) (*CSAMScanResult, error)

	// ScanModeration scans for content moderation violations using Rekognition.
	ScanModeration(ctx context.Context, storageKey string) (*ModerationResult, error)

	// ReportCSAM reports confirmed/suspected CSAM to NCMEC.
	ReportCSAM(ctx context.Context, uploadID uuid.UUID, scanResult *CSAMScanResult) error
}
