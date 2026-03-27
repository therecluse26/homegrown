package media

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// eventPublisher is a narrow interface for publishing domain events.
// Satisfied by *shared.EventBus. Allows mocking in unit tests.
type eventPublisher interface {
	Publish(ctx context.Context, event shared.DomainEvent) error
}

// mediaServiceImpl is the concrete implementation of MediaService. [09-media §5.1]
type mediaServiceImpl struct {
	uploads  UploadRepository
	procJobs ProcessingJobRepository
	storage  ObjectStorageAdapter
	safety   SafetyScanAdapter
	events   eventPublisher
	jobs     shared.JobEnqueuer
	config   *MediaConfig
}

// NewMediaService constructs a MediaService with all dependencies. [09-media §5.1]
func NewMediaService(
	uploads UploadRepository,
	procJobs ProcessingJobRepository,
	storage ObjectStorageAdapter,
	safety SafetyScanAdapter,
	events eventPublisher,
	jobs shared.JobEnqueuer,
	config *MediaConfig,
) MediaService {
	return &mediaServiceImpl{
		uploads:  uploads,
		procJobs: procJobs,
		storage:  storage,
		safety:   safety,
		events:   events,
		jobs:     jobs,
		config:   config,
	}
}

// ─── RequestUpload ────────────────────────────────────────────────────────────

func (s *mediaServiceImpl) RequestUpload(ctx context.Context, input *RequestUploadInput) (*UploadResponse, error) {
	// 1. Validate context rules
	if err := validateAttachmentRules(input.Context, input.ContentType, input.SizeBytes); err != nil {
		return nil, err
	}

	// 2. Generate storage key
	uploadID := uuid.Must(uuid.NewV7())
	sanitized := sanitizeFilename(input.Filename)
	storageKey := fmt.Sprintf("uploads/%s/%s/%s", input.FamilyID, uploadID, sanitized)

	// 3. Generate presigned PUT URL
	rules := getContextRules(input.Context)
	presignedURL, err := s.storage.PresignedPut(
		ctx,
		storageKey,
		rules.MaxSizeBytes,
		input.ContentType,
		s.config.PresignedUploadExpiry,
	)
	if err != nil {
		return nil, &MediaError{Err: ErrObjectStorageError}
	}

	// 4. Create upload record
	expiresAt := time.Now().Add(time.Duration(s.config.PresignedUploadExpiry) * time.Second)
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: input.FamilyID})
	upload, err := s.uploads.Create(ctx, scope, &CreateUploadRow{
		ID:               uploadID,
		FamilyID:         input.FamilyID,
		UploadedBy:       input.UploadedBy,
		Context:          input.Context,
		ContentType:      input.ContentType,
		OriginalFilename: sanitized,
		StorageKey:       storageKey,
		ExpiresAt:        expiresAt,
	})
	if err != nil {
		return nil, err
	}

	return &UploadResponse{
		UploadID:         upload.ID,
		PresignedURL:     presignedURL,
		StorageKey:       storageKey,
		ExpiresInSeconds: s.config.PresignedUploadExpiry,
	}, nil
}

// ─── ConfirmUpload ────────────────────────────────────────────────────────────

func (s *mediaServiceImpl) ConfirmUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) (*UploadInfo, error) {
	// 1. Find upload scoped to family
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
	upload, err := s.uploads.FindByID(ctx, scope, uploadID)
	if err != nil {
		return nil, err
	}

	// 2. Check status
	if upload.Status != UploadStatusPending {
		return nil, &MediaError{Err: ErrUploadNotConfirmed}
	}

	// 3. Check expiry
	if upload.ExpiresAt != nil && upload.ExpiresAt.Before(time.Now()) {
		return nil, &MediaError{Err: ErrUploadExpired}
	}

	// 4. HEAD check on S3 to verify object exists and get actual size
	meta, err := s.storage.GetObjectHead(ctx, upload.StorageKey)
	if err != nil {
		return nil, &MediaError{Err: ErrObjectStorageError}
	}

	// 5. Update size from actual object
	actualSize := int64(meta.ContentLength)

	// 6. Transition: pending → uploaded (with actual size)
	_, err = s.uploads.UpdateStatus(ctx, uploadID, UploadStatusUploaded, &UploadStatusUpdate{
		SizeBytes: &actualSize,
	})
	if err != nil {
		return nil, err
	}

	// 7. Transition: uploaded → processing
	upload, err = s.uploads.UpdateStatus(ctx, uploadID, UploadStatusProcessing, nil)
	if err != nil {
		return nil, err
	}

	// 8. Create processing job record
	_, err = s.procJobs.Create(ctx, uploadID, "process_upload")
	if err != nil {
		return nil, err
	}

	// 9. Enqueue ProcessUploadJob
	if err := s.jobs.Enqueue(ctx, &ProcessUploadPayload{UploadID: uploadID}); err != nil {
		return nil, err
	}

	return uploadToInfo(upload, s.config), nil
}

// ─── GetUpload ────────────────────────────────────────────────────────────────

func (s *mediaServiceImpl) GetUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) (*UploadInfo, error) {
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
	upload, err := s.uploads.FindByID(ctx, scope, uploadID)
	if err != nil {
		return nil, err
	}

	info := uploadToInfo(upload, s.config)

	// Generate URLs based on status and context
	if upload.Status == UploadStatusPublished {
		if upload.Context == UploadContextMarketplaceFile {
			// Marketplace files: presigned GET URL (never public)
			url, err := s.storage.PresignedGet(ctx, upload.StorageKey, s.config.PresignedDownloadExpiry)
			if err != nil {
				return nil, &MediaError{Err: ErrObjectStorageError}
			}
			info.URLs = &UploadURLs{Original: url}
		} else {
			// Published media: public CDN URL
			originalURL := fmt.Sprintf("%s/%s", strings.TrimRight(s.config.PublicURLBase, "/"), upload.StorageKey)
			urls := &UploadURLs{Original: originalURL}

			if upload.HasThumb {
				thumbURL := variantURL(s.config.PublicURLBase, upload.StorageKey, "thumb", upload.ContentType)
				urls.Thumb = &thumbURL
			}
			if upload.HasMedium {
				mediumURL := variantURL(s.config.PublicURLBase, upload.StorageKey, "medium", upload.ContentType)
				urls.Medium = &mediumURL
			}

			info.URLs = urls
		}
	}

	return info, nil
}

// ─── DeleteUpload ─────────────────────────────────────────────────────────────

func (s *mediaServiceImpl) DeleteUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) error {
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
	upload, err := s.uploads.FindByID(ctx, scope, uploadID)
	if err != nil {
		return err
	}

	// Delete from object storage (best-effort: if it fails the DB record is still marked deleted).
	// The orphan-cleanup job can retry storage deletion on next run. [09-media §6.1]
	_ = s.storage.DeleteObject(ctx, upload.StorageKey)

	_, err = s.uploads.UpdateStatus(ctx, uploadID, UploadStatusDeleted, nil)
	return err
}

// ─── ListUploads ──────────────────────────────────────────────────────────────

func (s *mediaServiceImpl) ListUploads(ctx context.Context, familyID uuid.UUID, limit uint32, afterID *uuid.UUID) (*UploadListResponse, error) {
	if limit == 0 || limit > 100 {
		limit = 20
	}
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
	uploads, err := s.uploads.ListByFamily(ctx, scope, limit, afterID)
	if err != nil {
		return nil, err
	}

	var nextCursor *string
	if uint32(len(uploads)) > limit {
		uploads = uploads[:limit]
		last := uploads[len(uploads)-1].ID.String()
		nextCursor = &last
	}

	items := make([]UploadInfo, len(uploads))
	for i := range uploads {
		items[i] = *uploadToInfo(&uploads[i], s.config)
	}
	return &UploadListResponse{Items: items, NextCursor: nextCursor}, nil
}

// ─── PresignedGet ─────────────────────────────────────────────────────────────

func (s *mediaServiceImpl) PresignedGet(ctx context.Context, storageKey string, expiresSeconds uint32) (string, error) {
	url, err := s.storage.PresignedGet(ctx, storageKey, expiresSeconds)
	if err != nil {
		return "", &MediaError{Err: ErrObjectStorageError}
	}
	return url, nil
}

// ─── ValidateAttachment ───────────────────────────────────────────────────────

func (s *mediaServiceImpl) ValidateAttachment(_ context.Context, uploadCtx UploadContext, contentType string, sizeBytes uint64) error {
	return validateAttachmentRules(uploadCtx, contentType, sizeBytes)
}

// ─── ReprocessUpload ──────────────────────────────────────────────────────────

func (s *mediaServiceImpl) ReprocessUpload(ctx context.Context, scope shared.FamilyScope, id uuid.UUID) error {
	// 1. Find upload scoped to family (returns ErrNotFound if wrong family or missing)
	if _, err := s.uploads.FindByID(ctx, scope, id); err != nil {
		return err
	}

	// 2. Re-enqueue the processing job
	if err := s.jobs.Enqueue(ctx, &ProcessUploadPayload{UploadID: id}); err != nil {
		return err
	}

	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// uploadToInfo converts an Upload model to an UploadInfo DTO.
func uploadToInfo(u *Upload, _ *MediaConfig) *UploadInfo {
	return &UploadInfo{
		UploadID:         u.ID,
		Status:           string(u.Status),
		Context:          string(u.Context),
		ContentType:      u.ContentType,
		OriginalFilename: u.OriginalFilename,
		SizeBytes:        u.SizeBytes,
		HasThumb:         u.HasThumb,
		HasMedium:        u.HasMedium,
		CreatedAt:        u.CreatedAt,
		PublishedAt:      u.PublishedAt,
	}
}

// variantURL generates the public CDN URL for an image variant.
func variantURL(publicBase, storageKey, suffix, contentType string) string {
	ext := extensionForContentType(contentType)
	base := strings.TrimRight(publicBase, "/")
	// Strip extension from storageKey to match upload-side variant naming
	baseKey := strings.TrimSuffix(storageKey, filepath.Ext(storageKey))
	return fmt.Sprintf("%s/%s__%s%s", base, baseKey, suffix, ext)
}

