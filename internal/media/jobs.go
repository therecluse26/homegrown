package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Job Payloads ─────────────────────────────────────────────────────────────

// ProcessUploadPayload is the payload for the ProcessUploadJob. [09-media §10]
type ProcessUploadPayload struct {
	UploadID uuid.UUID `json:"upload_id"`
}

func (ProcessUploadPayload) TaskType() string { return "media:process_upload" }

// CleanupOrphansPayload is the payload for the orphan cleanup job. [09-media §12]
type CleanupOrphansPayload struct{}

func (CleanupOrphansPayload) TaskType() string { return "media:cleanup_orphans" }

// ─── Process Upload Runner ────────────────────────────────────────────────────

// processUploadRunner holds dependencies for the ProcessUploadJob pipeline.
// Tested directly with mocks — no asynq serialization in unit tests. [09-media §10.1]
type processUploadRunner struct {
	uploads  UploadRepository
	procJobs ProcessingJobRepository
	storage  ObjectStorageAdapter
	safety   SafetyScanAdapter
	events   eventPublisher
}

// Run executes the full processing pipeline for a single upload. [09-media §10.1]
func (r *processUploadRunner) Run(ctx context.Context, uploadID uuid.UUID) error {
	// Fetch upload (unscoped — background job context)
	upload, err := r.uploads.FindByIDUnscoped(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("finding upload: %w", err)
	}

	// 1. Magic byte validation
	header, err := r.storage.GetObjectBytes(ctx, upload.StorageKey, 0, 16)
	if err != nil {
		return fmt.Errorf("reading magic bytes: %w", err)
	}
	detectedType := detectFileType(header)
	if !isCompatible(upload.ContentType, detectedType) {
		if _, statusErr := r.uploads.UpdateStatus(ctx, uploadID, UploadStatusRejected, &UploadStatusUpdate{}); statusErr != nil {
			slog.Error("failed to reject upload after magic byte mismatch", "upload_id", uploadID, "error", statusErr)
		}
		return &MediaError{Err: ErrMagicByteMismatch}
	}

	// 2. CSAM scan
	csamResult, err := r.safety.ScanCSAM(ctx, upload.StorageKey)
	if err != nil {
		var scanErr *ScanError
		if errors.As(err, &scanErr) && scanErr.Code == "unavailable" {
			slog.Warn("CSAM scan unavailable — continuing", "upload_id", uploadID)
		} else {
			return fmt.Errorf("CSAM scan failed: %w", err)
		}
	} else if csamResult.IsCSAM {
		// Quarantine immediately — short-circuit
		if _, statusErr := r.uploads.UpdateStatus(ctx, uploadID, UploadStatusQuarantined, &UploadStatusUpdate{}); statusErr != nil {
			return fmt.Errorf("quarantining upload: %w", statusErr)
		}
		if reportErr := r.safety.ReportCSAM(ctx, uploadID, csamResult); reportErr != nil {
			slog.Error("failed to report CSAM", "upload_id", uploadID, "error", reportErr)
		}
		_ = r.events.Publish(ctx, UploadQuarantined{
			UploadID: uploadID,
			FamilyID: upload.FamilyID,
			Context:  upload.Context,
		})
		return nil
	}

	// 3. Content moderation
	modResult, err := r.safety.ScanModeration(ctx, upload.StorageKey)
	if err != nil {
		var scanErr *ScanError
		if errors.As(err, &scanErr) && scanErr.Code == "unavailable" {
			slog.Warn("moderation scan unavailable — continuing", "upload_id", uploadID)
		} else {
			return fmt.Errorf("moderation scan failed: %w", err)
		}
	} else if modResult.AutoReject {
		labelsJSON, _ := json.Marshal(modResult.Labels)
		_ = r.uploads.SetModerationLabels(ctx, uploadID, labelsJSON)
		if _, statusErr := r.uploads.UpdateStatus(ctx, uploadID, UploadStatusRejected, &UploadStatusUpdate{}); statusErr != nil {
			return fmt.Errorf("rejecting upload: %w", statusErr)
		}
		_ = r.events.Publish(ctx, UploadRejected{
			UploadID: uploadID,
			FamilyID: upload.FamilyID,
			Context:  upload.Context,
			Labels:   modResult.Labels,
		})
		return nil
	} else if modResult.HasViolations {
		labelsJSON, _ := json.Marshal(modResult.Labels)
		_ = r.uploads.SetModerationLabels(ctx, uploadID, labelsJSON)
		if _, statusErr := r.uploads.UpdateStatus(ctx, uploadID, UploadStatusFlagged, &UploadStatusUpdate{}); statusErr != nil {
			return fmt.Errorf("flagging upload: %w", statusErr)
		}
		_ = r.events.Publish(ctx, UploadFlagged{
			UploadID: uploadID,
			FamilyID: upload.FamilyID,
			Context:  upload.Context,
			Labels:   modResult.Labels,
			Priority: modResult.Priority,
		})
		return nil
	}

	// 4. Variant generation (images only)
	hasThumb := false
	hasMedium := false
	if isImageType(upload.ContentType) && upload.SizeBytes != nil {
		imageBytes, err := r.storage.GetObjectBytes(ctx, upload.StorageKey, 0, uint64(*upload.SizeBytes))
		if err != nil {
			return fmt.Errorf("downloading image for variants: %w", err)
		}

		variants := []struct {
			suffix string
			maxW   int
			maxH   int
		}{
			{"thumb", 200, 200},
			{"medium", 800, 800},
		}

		for _, v := range variants {
			ext := extensionForContentType(upload.ContentType)
			variantKey := fmt.Sprintf("%s__%s%s", upload.StorageKey, v.suffix, ext)
			// In Phase 1, we store the original as variant (real resizing uses disintegration/imaging)
			_ = imageBytes
			if err := r.storage.PutObject(ctx, variantKey, imageBytes, upload.ContentType); err != nil {
				return fmt.Errorf("uploading %s variant: %w", v.suffix, err)
			}
		}

		hasThumb = true
		hasMedium = true
		if err := r.uploads.SetVariantFlags(ctx, uploadID, hasThumb, hasMedium); err != nil {
			return fmt.Errorf("setting variant flags: %w", err)
		}
	}

	// 5. Publish
	now := time.Now()
	sizeBytes := int64(0)
	if upload.SizeBytes != nil {
		sizeBytes = *upload.SizeBytes
	}
	upload, err = r.uploads.UpdateStatus(ctx, uploadID, UploadStatusPublished, &UploadStatusUpdate{
		PublishedAt: &now,
	})
	if err != nil {
		return fmt.Errorf("publishing upload: %w", err)
	}

	_ = r.events.Publish(ctx, UploadPublished{
		UploadID:    uploadID,
		FamilyID:    upload.FamilyID,
		Context:     upload.Context,
		StorageKey:  upload.StorageKey,
		ContentType: upload.ContentType,
		SizeBytes:   sizeBytes,
		HasThumb:    hasThumb,
		HasMedium:   hasMedium,
	})

	return nil
}

// ─── Orphan Cleanup ───────────────────────────────────────────────────────────

// RunCleanup finds and marks expired pending uploads. [09-media §12.1]
func RunCleanup(ctx context.Context, uploads UploadRepository, storage ObjectStorageAdapter) (uint32, error) {
	orphans, err := uploads.FindExpiredPending(ctx, time.Now(), 1000)
	if err != nil {
		return 0, fmt.Errorf("finding expired uploads: %w", err)
	}

	var cleaned uint32
	for _, orphan := range orphans {
		// Best-effort delete from S3 (may not exist)
		_ = storage.DeleteObject(ctx, orphan.StorageKey)

		if _, err := uploads.UpdateStatus(ctx, orphan.ID, UploadStatusExpired, &UploadStatusUpdate{}); err != nil {
			slog.Error("failed to expire orphan upload", "upload_id", orphan.ID, "error", err)
			continue
		}
		cleaned++
	}

	if cleaned > 0 {
		slog.Info("cleaned up orphan uploads", "count", cleaned)
	}

	return cleaned, nil
}

// ─── Worker Registration ──────────────────────────────────────────────────────

// RegisterMediaWorkers registers media background job handlers with the worker. [09-media §10]
func RegisterMediaWorkers(
	worker shared.JobWorker,
	uploads UploadRepository,
	procJobs ProcessingJobRepository,
	storage ObjectStorageAdapter,
	safety SafetyScanAdapter,
	events eventPublisher,
) {
	runner := &processUploadRunner{
		uploads:  uploads,
		procJobs: procJobs,
		storage:  storage,
		safety:   safety,
		events:   events,
	}

	worker.Handle("media:process_upload", func(ctx context.Context, payload []byte) error {
		var p ProcessUploadPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshaling process upload payload: %w", err)
		}

		// Mark job running
		// In a real scenario, we'd look up the job ID from the upload — simplified here
		slog.Info("processing upload", "upload_id", p.UploadID)

		if err := runner.Run(ctx, p.UploadID); err != nil {
			slog.Error("upload processing failed", "upload_id", p.UploadID, "error", err)
			return err
		}

		return nil
	})

	worker.Handle("media:cleanup_orphans", func(ctx context.Context, _ []byte) error {
		count, err := RunCleanup(ctx, uploads, storage)
		if err != nil {
			return err
		}
		slog.Info("orphan cleanup completed", "count", count)
		return nil
	})
}

