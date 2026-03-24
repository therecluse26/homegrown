package media

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── PgUploadRepository ───────────────────────────────────────────────────────

// PgUploadRepository implements UploadRepository using PostgreSQL/GORM. [09-media §6.1]
type PgUploadRepository struct {
	db *gorm.DB
}

// NewPgUploadRepository constructs an UploadRepository.
func NewPgUploadRepository(db *gorm.DB) UploadRepository {
	return &PgUploadRepository{db: db}
}

func (r *PgUploadRepository) Create(_ context.Context, scope shared.FamilyScope, input *CreateUploadRow) (*Upload, error) {
	upload := &Upload{
		ID:               input.ID,
		FamilyID:         scope.FamilyID(),
		UploadedBy:       input.UploadedBy,
		Context:          input.Context,
		ContentType:      input.ContentType,
		OriginalFilename: input.OriginalFilename,
		StorageKey:       input.StorageKey,
		Status:           UploadStatusPending,
		ExpiresAt:        &input.ExpiresAt,
	}

	if err := r.db.Create(upload).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return upload, nil
}

func (r *PgUploadRepository) FindByID(_ context.Context, scope shared.FamilyScope, uploadID uuid.UUID) (*Upload, error) {
	var upload Upload
	if err := r.db.Where("id = ? AND family_id = ?", uploadID, scope.FamilyID()).First(&upload).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &MediaError{Err: ErrUploadNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &upload, nil
}

func (r *PgUploadRepository) FindByIDUnscoped(_ context.Context, uploadID uuid.UUID) (*Upload, error) {
	var upload Upload
	if err := r.db.Where("id = ?", uploadID).First(&upload).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &MediaError{Err: ErrUploadNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &upload, nil
}

func (r *PgUploadRepository) UpdateStatus(_ context.Context, uploadID uuid.UUID, status UploadStatus, updates *UploadStatusUpdate) (*Upload, error) {
	updateMap := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	if updates != nil {
		if updates.SizeBytes != nil {
			updateMap["size_bytes"] = *updates.SizeBytes
		}
		if updates.PublishedAt != nil {
			updateMap["published_at"] = *updates.PublishedAt
		}
	}

	var upload Upload
	result := r.db.Model(&upload).Where("id = ?", uploadID).Updates(updateMap)
	if result.Error != nil {
		return nil, shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, &MediaError{Err: ErrUploadNotFound}
	}

	// Reload
	if err := r.db.Where("id = ?", uploadID).First(&upload).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &upload, nil
}

func (r *PgUploadRepository) UpdateProbeMetadata(_ context.Context, uploadID uuid.UUID, probe json.RawMessage, wasCompressed bool, originalSizeBytes *int64) error {
	updateMap := map[string]any{
		"probe_metadata": probe,
		"was_compressed": wasCompressed,
		"updated_at":     time.Now(),
	}
	if originalSizeBytes != nil {
		updateMap["original_size_bytes"] = *originalSizeBytes
	}

	result := r.db.Model(&Upload{}).Where("id = ?", uploadID).Updates(updateMap)
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	return nil
}

func (r *PgUploadRepository) SetVariantFlags(_ context.Context, uploadID uuid.UUID, hasThumb bool, hasMedium bool) error {
	result := r.db.Model(&Upload{}).Where("id = ?", uploadID).Updates(map[string]any{
		"has_thumb":  hasThumb,
		"has_medium": hasMedium,
		"updated_at": time.Now(),
	})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	return nil
}

func (r *PgUploadRepository) SetModerationLabels(_ context.Context, uploadID uuid.UUID, labels json.RawMessage) error {
	result := r.db.Model(&Upload{}).Where("id = ?", uploadID).Updates(map[string]any{
		"moderation_labels": labels,
		"updated_at":        time.Now(),
	})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	return nil
}

func (r *PgUploadRepository) SetCSAMScannedAt(_ context.Context, uploadID uuid.UUID) error {
	result := r.db.Model(&Upload{}).Where("id = ?", uploadID).Updates(map[string]any{
		"last_csam_scanned_at": time.Now(),
		"updated_at":           time.Now(),
	})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	return nil
}

func (r *PgUploadRepository) FindExpiredPending(_ context.Context, before time.Time, limit uint32) ([]Upload, error) {
	var uploads []Upload
	if err := r.db.Where("status = ? AND expires_at < ?", UploadStatusPending, before).
		Limit(int(limit)).
		Find(&uploads).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return uploads, nil
}

// ─── PgProcessingJobRepository ────────────────────────────────────────────────

// PgProcessingJobRepository implements ProcessingJobRepository using PostgreSQL/GORM. [09-media §6.2]
type PgProcessingJobRepository struct {
	db *gorm.DB
}

// NewPgProcessingJobRepository constructs a ProcessingJobRepository.
func NewPgProcessingJobRepository(db *gorm.DB) ProcessingJobRepository {
	return &PgProcessingJobRepository{db: db}
}

func (r *PgProcessingJobRepository) Create(_ context.Context, uploadID uuid.UUID, jobType string) (*ProcessingJob, error) {
	job := &ProcessingJob{
		UploadID: uploadID,
		JobType:  jobType,
		Status:   "queued",
	}
	if err := r.db.Create(job).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return job, nil
}

func (r *PgProcessingJobRepository) MarkRunning(_ context.Context, jobID uuid.UUID) error {
	now := time.Now()
	result := r.db.Model(&ProcessingJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":     "running",
		"started_at": now,
		"attempts":   gorm.Expr("attempts + 1"),
	})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	return nil
}

func (r *PgProcessingJobRepository) MarkCompleted(_ context.Context, jobID uuid.UUID) error {
	now := time.Now()
	result := r.db.Model(&ProcessingJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":       "completed",
		"completed_at": now,
	})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	return nil
}

func (r *PgProcessingJobRepository) MarkFailed(_ context.Context, jobID uuid.UUID, errorMessage string) error {
	result := r.db.Model(&ProcessingJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":        "failed",
		"error_message": errorMessage,
	})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	return nil
}

func (r *PgProcessingJobRepository) FindRetryable(_ context.Context, limit uint32) ([]ProcessingJob, error) {
	var jobs []ProcessingJob
	if err := r.db.Where("status = ? AND attempts < max_attempts", "failed").
		Limit(int(limit)).
		Find(&jobs).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return jobs, nil
}

// ─── PgTranscodeJobRepository ─────────────────────────────────────────────────

// PgTranscodeJobRepository implements TranscodeJobRepository using PostgreSQL/GORM. [09-media §6.3]
type PgTranscodeJobRepository struct {
	db *gorm.DB
}

// NewPgTranscodeJobRepository constructs a TranscodeJobRepository.
func NewPgTranscodeJobRepository(db *gorm.DB) TranscodeJobRepository {
	return &PgTranscodeJobRepository{db: db}
}

func (r *PgTranscodeJobRepository) Create(_ context.Context, uploadID uuid.UUID, inputKey string) (*TranscodeJob, error) {
	job := &TranscodeJob{
		UploadID: uploadID,
		InputKey: inputKey,
		Status:   "pending",
	}
	if err := r.db.Create(job).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return job, nil
}

func (r *PgTranscodeJobRepository) MarkRunning(_ context.Context, jobID uuid.UUID) error {
	now := time.Now()
	result := r.db.Model(&TranscodeJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":     "processing",
		"started_at": now,
		"updated_at": now,
	})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	return nil
}

func (r *PgTranscodeJobRepository) MarkCompleted(_ context.Context, jobID uuid.UUID, outputKeys json.RawMessage, durationSeconds int) error {
	now := time.Now()
	result := r.db.Model(&TranscodeJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":           "completed",
		"output_keys":      outputKeys,
		"duration_seconds": durationSeconds,
		"completed_at":     now,
		"updated_at":       now,
	})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	return nil
}

func (r *PgTranscodeJobRepository) MarkFailed(_ context.Context, jobID uuid.UUID, errorMessage string) error {
	now := time.Now()
	result := r.db.Model(&TranscodeJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":        "failed",
		"error_message": errorMessage,
		"updated_at":    now,
	})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	return nil
}
