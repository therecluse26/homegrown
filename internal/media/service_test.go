package media

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

func defaultMediaConfig() *MediaConfig {
	return &MediaConfig{
		PublicURLBase:           "https://media.example.com",
		PresignedUploadExpiry:   3600,
		PresignedDownloadExpiry: 3600,
	}
}

// ─── RequestUpload Tests ──────────────────────────────────────────────────────

func TestRequestUpload_success(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	storageAdapter := newMockObjectStorageAdapter()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), storageAdapter, newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	storageAdapter.presignedPutFn = func(_ context.Context, key string, _ uint64, _ string, _ uint32) (string, error) {
		return "https://s3.example.com/presigned/" + key, nil
	}
	uploadRepo.createFn = func(_ context.Context, _ shared.FamilyScope, input *CreateUploadRow) (*Upload, error) {
		return &Upload{
			ID:               input.ID,
			FamilyID:         input.FamilyID,
			Context:          input.Context,
			ContentType:      input.ContentType,
			OriginalFilename: input.OriginalFilename,
			StorageKey:       input.StorageKey,
			Status:           UploadStatusPending,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}, nil
	}

	resp, err := svc.RequestUpload(context.Background(), &RequestUploadInput{
		FamilyID:    uuid.Must(uuid.NewV7()),
		UploadedBy:  uuid.Must(uuid.NewV7()),
		Context:     UploadContextJournalImage,
		ContentType: "image/jpeg",
		Filename:    "nature-walk.jpg",
		SizeBytes:   2 * mb,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UploadID == uuid.Nil {
		t.Error("expected non-nil upload ID")
	}
	if resp.PresignedURL == "" {
		t.Error("expected non-empty presigned URL")
	}
	if resp.StorageKey == "" {
		t.Error("expected non-empty storage key")
	}
	if resp.ExpiresInSeconds != 3600 {
		t.Errorf("ExpiresInSeconds = %d, want 3600", resp.ExpiresInSeconds)
	}
}

func TestRequestUpload_invalid_content_type(t *testing.T) {
	svc := NewMediaService(newMockUploadRepository(), newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	_, err := svc.RequestUpload(context.Background(), &RequestUploadInput{
		FamilyID:    uuid.Must(uuid.NewV7()),
		UploadedBy:  uuid.Must(uuid.NewV7()),
		Context:     UploadContextProfilePhoto,
		ContentType: "video/mp4",
		Filename:    "video.mp4",
		SizeBytes:   1 * mb,
	})

	if !errors.Is(err, ErrInvalidFileType) {
		t.Errorf("expected ErrInvalidFileType, got %v", err)
	}
}

func TestRequestUpload_file_too_large(t *testing.T) {
	svc := NewMediaService(newMockUploadRepository(), newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	_, err := svc.RequestUpload(context.Background(), &RequestUploadInput{
		FamilyID:    uuid.Must(uuid.NewV7()),
		UploadedBy:  uuid.Must(uuid.NewV7()),
		Context:     UploadContextProfilePhoto,
		ContentType: "image/jpeg",
		Filename:    "huge.jpg",
		SizeBytes:   100 * mb,
	})

	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestRequestUpload_storage_error(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	storageAdapter := newMockObjectStorageAdapter()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), storageAdapter, newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	storageAdapter.presignedPutFn = func(_ context.Context, _ string, _ uint64, _ string, _ uint32) (string, error) {
		return "", errors.New("S3 error")
	}

	_, err := svc.RequestUpload(context.Background(), &RequestUploadInput{
		FamilyID:    uuid.Must(uuid.NewV7()),
		UploadedBy:  uuid.Must(uuid.NewV7()),
		Context:     UploadContextJournalImage,
		ContentType: "image/jpeg",
		Filename:    "photo.jpg",
		SizeBytes:   1 * mb,
	})

	if !errors.Is(err, ErrObjectStorageError) {
		t.Errorf("expected ErrObjectStorageError, got %v", err)
	}
}

func TestRequestUpload_sanitizes_filename(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	storageAdapter := newMockObjectStorageAdapter()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), storageAdapter, newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	var capturedInput *CreateUploadRow
	storageAdapter.presignedPutFn = func(_ context.Context, _ string, _ uint64, _ string, _ uint32) (string, error) {
		return "https://s3.example.com/presigned", nil
	}
	uploadRepo.createFn = func(_ context.Context, _ shared.FamilyScope, input *CreateUploadRow) (*Upload, error) {
		capturedInput = input
		return &Upload{
			ID:               input.ID,
			FamilyID:         input.FamilyID,
			OriginalFilename: input.OriginalFilename,
			StorageKey:       input.StorageKey,
			Status:           UploadStatusPending,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}, nil
	}

	_, err := svc.RequestUpload(context.Background(), &RequestUploadInput{
		FamilyID:    uuid.Must(uuid.NewV7()),
		UploadedBy:  uuid.Must(uuid.NewV7()),
		Context:     UploadContextJournalImage,
		ContentType: "image/jpeg",
		Filename:    "/home/user/my photo.jpg",
		SizeBytes:   1 * mb,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedInput.OriginalFilename != "my-photo.jpg" {
		t.Errorf("OriginalFilename = %q, want %q", capturedInput.OriginalFilename, "my-photo.jpg")
	}
}

// ─── ConfirmUpload Tests ──────────────────────────────────────────────────────

func TestConfirmUpload_success(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	procJobRepo := newMockProcessingJobRepository()
	storageAdapter := newMockObjectStorageAdapter()
	svc := NewMediaService(uploadRepo, procJobRepo, storageAdapter, newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	familyID := uuid.Must(uuid.NewV7())
	uploadID := uuid.Must(uuid.NewV7())
	expiresAt := time.Now().Add(1 * time.Hour)

	uploadRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, id uuid.UUID) (*Upload, error) {
		return &Upload{
			ID:        id,
			FamilyID:  familyID,
			Status:    UploadStatusPending,
			StorageKey: "uploads/test/key",
			ExpiresAt: &expiresAt,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}
	storageAdapter.getObjectHeadFn = func(_ context.Context, _ string) (*ObjectMetadata, error) {
		return &ObjectMetadata{ContentLength: 2048576}, nil
	}
	uploadRepo.updateStatusFn = func(_ context.Context, id uuid.UUID, status UploadStatus, updates *UploadStatusUpdate) (*Upload, error) {
		var sizeBytes *int64
		if updates != nil {
			sizeBytes = updates.SizeBytes
		}
		return &Upload{
			ID:          id,
			FamilyID:    familyID,
			Status:      status,
			StorageKey:  "uploads/test/key",
			SizeBytes:   sizeBytes,
			ContentType: "image/jpeg",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}
	procJobRepo.createFn = func(_ context.Context, _ uuid.UUID, _ string) (*ProcessingJob, error) {
		return &ProcessingJob{ID: uuid.Must(uuid.NewV7())}, nil
	}

	info, err := svc.ConfirmUpload(context.Background(), uploadID, familyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Status != "processing" {
		t.Errorf("Status = %q, want %q", info.Status, "processing")
	}
}

func TestConfirmUpload_not_found(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	uploadRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, _ uuid.UUID) (*Upload, error) {
		return nil, &MediaError{Err: ErrUploadNotFound}
	}

	_, err := svc.ConfirmUpload(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrUploadNotFound) {
		t.Errorf("expected ErrUploadNotFound, got %v", err)
	}
}

func TestConfirmUpload_not_pending(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	uploadRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, id uuid.UUID) (*Upload, error) {
		return &Upload{ID: id, Status: UploadStatusPublished, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
	}

	_, err := svc.ConfirmUpload(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrUploadNotConfirmed) {
		t.Errorf("expected ErrUploadNotConfirmed, got %v", err)
	}
}

func TestConfirmUpload_expired(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	expired := time.Now().Add(-1 * time.Hour)
	uploadRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, id uuid.UUID) (*Upload, error) {
		return &Upload{ID: id, Status: UploadStatusPending, ExpiresAt: &expired, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
	}

	_, err := svc.ConfirmUpload(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrUploadExpired) {
		t.Errorf("expected ErrUploadExpired, got %v", err)
	}
}

func TestConfirmUpload_object_not_in_storage(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	storageAdapter := newMockObjectStorageAdapter()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), storageAdapter, newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	expiresAt := time.Now().Add(1 * time.Hour)
	uploadRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, id uuid.UUID) (*Upload, error) {
		return &Upload{ID: id, Status: UploadStatusPending, StorageKey: "key", ExpiresAt: &expiresAt, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
	}
	storageAdapter.getObjectHeadFn = func(_ context.Context, _ string) (*ObjectMetadata, error) {
		return nil, ErrObjectNotFound
	}

	_, err := svc.ConfirmUpload(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrObjectStorageError) {
		t.Errorf("expected ErrObjectStorageError, got %v", err)
	}
}

// ─── GetUpload Tests ──────────────────────────────────────────────────────────

func TestGetUpload_published_with_cdn_urls(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	familyID := uuid.Must(uuid.NewV7())
	publishedAt := time.Now()
	uploadRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, id uuid.UUID) (*Upload, error) {
		return &Upload{
			ID:          id,
			FamilyID:    familyID,
			Status:      UploadStatusPublished,
			Context:     UploadContextJournalImage,
			ContentType: "image/jpeg",
			StorageKey:  "uploads/fam/upload/photo.jpg",
			HasThumb:    true,
			HasMedium:   true,
			PublishedAt: &publishedAt,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	info, err := svc.GetUpload(context.Background(), uuid.Must(uuid.NewV7()), familyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.URLs == nil {
		t.Fatal("expected URLs to be set for published upload")
	}
	if info.URLs.Original != "https://media.example.com/uploads/fam/upload/photo.jpg" {
		t.Errorf("Original URL = %q, unexpected", info.URLs.Original)
	}
	if info.URLs.Thumb == nil {
		t.Fatal("expected thumb URL")
	}
	if *info.URLs.Thumb != "https://media.example.com/uploads/fam/upload/photo__thumb.jpg" {
		t.Errorf("Thumb URL = %q, want photo__thumb.jpg pattern", *info.URLs.Thumb)
	}
	if info.URLs.Medium == nil {
		t.Fatal("expected medium URL")
	}
	if *info.URLs.Medium != "https://media.example.com/uploads/fam/upload/photo__medium.jpg" {
		t.Errorf("Medium URL = %q, want photo__medium.jpg pattern", *info.URLs.Medium)
	}
}

func TestGetUpload_processing_has_no_urls(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	uploadRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, id uuid.UUID) (*Upload, error) {
		return &Upload{
			ID:        id,
			Status:    UploadStatusProcessing,
			Context:   UploadContextJournalImage,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}

	info, err := svc.GetUpload(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.URLs != nil {
		t.Error("expected nil URLs for processing upload")
	}
}

func TestGetUpload_marketplace_uses_presigned(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	storageAdapter := newMockObjectStorageAdapter()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), storageAdapter, newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	uploadRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, id uuid.UUID) (*Upload, error) {
		return &Upload{
			ID:         id,
			Status:     UploadStatusPublished,
			Context:    UploadContextMarketplaceFile,
			StorageKey: "uploads/fam/upload/resource.pdf",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}, nil
	}
	storageAdapter.presignedGetFn = func(_ context.Context, key string, _ uint32) (string, error) {
		return "https://s3.example.com/signed/" + key, nil
	}

	info, err := svc.GetUpload(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.URLs == nil || info.URLs.Original == "" {
		t.Fatal("expected presigned URL for marketplace file")
	}
	if info.URLs.Thumb != nil || info.URLs.Medium != nil {
		t.Error("marketplace files should not have variant URLs")
	}
}

func TestGetUpload_published_no_variants(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	uploadRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, id uuid.UUID) (*Upload, error) {
		return &Upload{
			ID:          id,
			Status:      UploadStatusPublished,
			Context:     UploadContextJournalImage,
			StorageKey:  "uploads/fam/upload/photo.jpg",
			ContentType: "image/jpeg",
			HasThumb:    false,
			HasMedium:   false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	info, err := svc.GetUpload(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.URLs == nil {
		t.Fatal("expected URLs for published upload")
	}
	if info.URLs.Thumb != nil {
		t.Error("expected nil thumb URL when has_thumb is false")
	}
	if info.URLs.Medium != nil {
		t.Error("expected nil medium URL when has_medium is false")
	}
}

func TestGetUpload_not_found(t *testing.T) {
	uploadRepo := newMockUploadRepository()
	svc := NewMediaService(uploadRepo, newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	uploadRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, _ uuid.UUID) (*Upload, error) {
		return nil, &MediaError{Err: ErrUploadNotFound}
	}

	_, err := svc.GetUpload(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrUploadNotFound) {
		t.Errorf("expected ErrUploadNotFound, got %v", err)
	}
}

// ─── ValidateAttachment Tests ─────────────────────────────────────────────────

func TestValidateAttachment_valid(t *testing.T) {
	svc := NewMediaService(newMockUploadRepository(), newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())
	err := svc.ValidateAttachment(context.Background(), UploadContextJournalImage, "image/jpeg", 5*mb)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAttachment_invalid(t *testing.T) {
	svc := NewMediaService(newMockUploadRepository(), newMockProcessingJobRepository(), newMockObjectStorageAdapter(), newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())
	err := svc.ValidateAttachment(context.Background(), UploadContextProfilePhoto, "video/mp4", 1*mb)
	if !errors.Is(err, ErrInvalidFileType) {
		t.Errorf("expected ErrInvalidFileType, got %v", err)
	}
}

// ─── PresignedGet Tests ───────────────────────────────────────────────────────

func TestPresignedGet_success(t *testing.T) {
	storageAdapter := newMockObjectStorageAdapter()
	svc := NewMediaService(newMockUploadRepository(), newMockProcessingJobRepository(), storageAdapter, newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	storageAdapter.presignedGetFn = func(_ context.Context, key string, _ uint32) (string, error) {
		return "https://s3.example.com/signed/" + key, nil
	}

	url, err := svc.PresignedGet(context.Background(), "uploads/test/key", 3600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}
}

func TestPresignedGet_storage_error(t *testing.T) {
	storageAdapter := newMockObjectStorageAdapter()
	svc := NewMediaService(newMockUploadRepository(), newMockProcessingJobRepository(), storageAdapter, newMockSafetyScanAdapter(), newMockEventBus(), &shared.NoopJobEnqueuer{}, defaultMediaConfig())

	storageAdapter.presignedGetFn = func(_ context.Context, _ string, _ uint32) (string, error) {
		return "", errors.New("S3 error")
	}

	_, err := svc.PresignedGet(context.Background(), "uploads/test/key", 3600)
	if !errors.Is(err, ErrObjectStorageError) {
		t.Errorf("expected ErrObjectStorageError, got %v", err)
	}
}
