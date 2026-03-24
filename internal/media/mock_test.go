package media

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Mock MediaService ────────────────────────────────────────────────────────

type mockMediaService struct {
	requestUploadFn      func(ctx context.Context, input *RequestUploadInput) (*UploadResponse, error)
	confirmUploadFn      func(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) (*UploadInfo, error)
	deleteUploadFn       func(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) error
	getUploadFn          func(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) (*UploadInfo, error)
	presignedGetFn       func(ctx context.Context, storageKey string, expiresSeconds uint32) (string, error)
	validateAttachmentFn func(ctx context.Context, uploadCtx UploadContext, contentType string, sizeBytes uint64) error
}

func newMockMediaService() *mockMediaService {
	return &mockMediaService{}
}

func (m *mockMediaService) RequestUpload(ctx context.Context, input *RequestUploadInput) (*UploadResponse, error) {
	if m.requestUploadFn != nil {
		return m.requestUploadFn(ctx, input)
	}
	panic("RequestUpload not mocked")
}

func (m *mockMediaService) ConfirmUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) (*UploadInfo, error) {
	if m.confirmUploadFn != nil {
		return m.confirmUploadFn(ctx, uploadID, familyID)
	}
	panic("ConfirmUpload not mocked")
}

func (m *mockMediaService) DeleteUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) error {
	if m.deleteUploadFn != nil {
		return m.deleteUploadFn(ctx, uploadID, familyID)
	}
	panic("DeleteUpload not mocked")
}

func (m *mockMediaService) GetUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) (*UploadInfo, error) {
	if m.getUploadFn != nil {
		return m.getUploadFn(ctx, uploadID, familyID)
	}
	panic("GetUpload not mocked")
}

func (m *mockMediaService) PresignedGet(ctx context.Context, storageKey string, expiresSeconds uint32) (string, error) {
	if m.presignedGetFn != nil {
		return m.presignedGetFn(ctx, storageKey, expiresSeconds)
	}
	panic("PresignedGet not mocked")
}

func (m *mockMediaService) ValidateAttachment(ctx context.Context, uploadCtx UploadContext, contentType string, sizeBytes uint64) error {
	if m.validateAttachmentFn != nil {
		return m.validateAttachmentFn(ctx, uploadCtx, contentType, sizeBytes)
	}
	panic("ValidateAttachment not mocked")
}

// ─── Mock UploadRepository ────────────────────────────────────────────────────

type mockUploadRepository struct {
	createFn             func(ctx context.Context, scope shared.FamilyScope, input *CreateUploadRow) (*Upload, error)
	findByIDFn           func(ctx context.Context, scope shared.FamilyScope, uploadID uuid.UUID) (*Upload, error)
	findByIDUnscopedFn   func(ctx context.Context, uploadID uuid.UUID) (*Upload, error)
	updateStatusFn       func(ctx context.Context, uploadID uuid.UUID, status UploadStatus, updates *UploadStatusUpdate) (*Upload, error)
	updateProbeMetadataFn func(ctx context.Context, uploadID uuid.UUID, probe json.RawMessage, wasCompressed bool, originalSizeBytes *int64) error
	setVariantFlagsFn    func(ctx context.Context, uploadID uuid.UUID, hasThumb bool, hasMedium bool) error
	setModerationLabelsFn func(ctx context.Context, uploadID uuid.UUID, labels json.RawMessage) error
	findExpiredPendingFn func(ctx context.Context, before time.Time, limit uint32) ([]Upload, error)
}

func newMockUploadRepository() *mockUploadRepository {
	return &mockUploadRepository{}
}

func (m *mockUploadRepository) Create(ctx context.Context, scope shared.FamilyScope, input *CreateUploadRow) (*Upload, error) {
	if m.createFn != nil {
		return m.createFn(ctx, scope, input)
	}
	panic("Create not mocked")
}

func (m *mockUploadRepository) FindByID(ctx context.Context, scope shared.FamilyScope, uploadID uuid.UUID) (*Upload, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, scope, uploadID)
	}
	panic("FindByID not mocked")
}

func (m *mockUploadRepository) FindByIDUnscoped(ctx context.Context, uploadID uuid.UUID) (*Upload, error) {
	if m.findByIDUnscopedFn != nil {
		return m.findByIDUnscopedFn(ctx, uploadID)
	}
	panic("FindByIDUnscoped not mocked")
}

func (m *mockUploadRepository) UpdateStatus(ctx context.Context, uploadID uuid.UUID, status UploadStatus, updates *UploadStatusUpdate) (*Upload, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, uploadID, status, updates)
	}
	panic("UpdateStatus not mocked")
}

func (m *mockUploadRepository) UpdateProbeMetadata(ctx context.Context, uploadID uuid.UUID, probe json.RawMessage, wasCompressed bool, originalSizeBytes *int64) error {
	if m.updateProbeMetadataFn != nil {
		return m.updateProbeMetadataFn(ctx, uploadID, probe, wasCompressed, originalSizeBytes)
	}
	panic("UpdateProbeMetadata not mocked")
}

func (m *mockUploadRepository) SetVariantFlags(ctx context.Context, uploadID uuid.UUID, hasThumb bool, hasMedium bool) error {
	if m.setVariantFlagsFn != nil {
		return m.setVariantFlagsFn(ctx, uploadID, hasThumb, hasMedium)
	}
	panic("SetVariantFlags not mocked")
}

func (m *mockUploadRepository) SetModerationLabels(ctx context.Context, uploadID uuid.UUID, labels json.RawMessage) error {
	if m.setModerationLabelsFn != nil {
		return m.setModerationLabelsFn(ctx, uploadID, labels)
	}
	panic("SetModerationLabels not mocked")
}

func (m *mockUploadRepository) FindExpiredPending(ctx context.Context, before time.Time, limit uint32) ([]Upload, error) {
	if m.findExpiredPendingFn != nil {
		return m.findExpiredPendingFn(ctx, before, limit)
	}
	panic("FindExpiredPending not mocked")
}

// ─── Mock ProcessingJobRepository ─────────────────────────────────────────────

type mockProcessingJobRepository struct {
	createFn        func(ctx context.Context, uploadID uuid.UUID, jobType string) (*ProcessingJob, error)
	markRunningFn   func(ctx context.Context, jobID uuid.UUID) error
	markCompletedFn func(ctx context.Context, jobID uuid.UUID) error
	markFailedFn    func(ctx context.Context, jobID uuid.UUID, errorMessage string) error
	findRetryableFn func(ctx context.Context, limit uint32) ([]ProcessingJob, error)
}

func newMockProcessingJobRepository() *mockProcessingJobRepository {
	return &mockProcessingJobRepository{}
}

func (m *mockProcessingJobRepository) Create(ctx context.Context, uploadID uuid.UUID, jobType string) (*ProcessingJob, error) {
	if m.createFn != nil {
		return m.createFn(ctx, uploadID, jobType)
	}
	panic("ProcessingJob.Create not mocked")
}

func (m *mockProcessingJobRepository) MarkRunning(ctx context.Context, jobID uuid.UUID) error {
	if m.markRunningFn != nil {
		return m.markRunningFn(ctx, jobID)
	}
	panic("MarkRunning not mocked")
}

func (m *mockProcessingJobRepository) MarkCompleted(ctx context.Context, jobID uuid.UUID) error {
	if m.markCompletedFn != nil {
		return m.markCompletedFn(ctx, jobID)
	}
	panic("MarkCompleted not mocked")
}

func (m *mockProcessingJobRepository) MarkFailed(ctx context.Context, jobID uuid.UUID, errorMessage string) error {
	if m.markFailedFn != nil {
		return m.markFailedFn(ctx, jobID, errorMessage)
	}
	panic("MarkFailed not mocked")
}

func (m *mockProcessingJobRepository) FindRetryable(ctx context.Context, limit uint32) ([]ProcessingJob, error) {
	if m.findRetryableFn != nil {
		return m.findRetryableFn(ctx, limit)
	}
	panic("FindRetryable not mocked")
}

// ─── Mock ObjectStorageAdapter ────────────────────────────────────────────────

type mockObjectStorageAdapter struct {
	presignedPutFn   func(ctx context.Context, key string, maxSizeBytes uint64, contentType string, expiresSeconds uint32) (string, error)
	presignedGetFn   func(ctx context.Context, key string, expiresSeconds uint32) (string, error)
	putObjectFn      func(ctx context.Context, key string, data []byte, contentType string) error
	getObjectHeadFn  func(ctx context.Context, key string) (*ObjectMetadata, error)
	getObjectBytesFn func(ctx context.Context, key string, start uint64, end uint64) ([]byte, error)
	deleteObjectFn   func(ctx context.Context, key string) error
}

func newMockObjectStorageAdapter() *mockObjectStorageAdapter {
	return &mockObjectStorageAdapter{}
}

func (m *mockObjectStorageAdapter) PresignedPut(ctx context.Context, key string, maxSizeBytes uint64, contentType string, expiresSeconds uint32) (string, error) {
	if m.presignedPutFn != nil {
		return m.presignedPutFn(ctx, key, maxSizeBytes, contentType, expiresSeconds)
	}
	panic("PresignedPut not mocked")
}

func (m *mockObjectStorageAdapter) PresignedGet(ctx context.Context, key string, expiresSeconds uint32) (string, error) {
	if m.presignedGetFn != nil {
		return m.presignedGetFn(ctx, key, expiresSeconds)
	}
	panic("PresignedGet not mocked")
}

func (m *mockObjectStorageAdapter) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	if m.putObjectFn != nil {
		return m.putObjectFn(ctx, key, data, contentType)
	}
	panic("PutObject not mocked")
}

func (m *mockObjectStorageAdapter) GetObjectHead(ctx context.Context, key string) (*ObjectMetadata, error) {
	if m.getObjectHeadFn != nil {
		return m.getObjectHeadFn(ctx, key)
	}
	panic("GetObjectHead not mocked")
}

func (m *mockObjectStorageAdapter) GetObjectBytes(ctx context.Context, key string, start uint64, end uint64) ([]byte, error) {
	if m.getObjectBytesFn != nil {
		return m.getObjectBytesFn(ctx, key, start, end)
	}
	panic("GetObjectBytes not mocked")
}

func (m *mockObjectStorageAdapter) DeleteObject(ctx context.Context, key string) error {
	if m.deleteObjectFn != nil {
		return m.deleteObjectFn(ctx, key)
	}
	panic("DeleteObject not mocked")
}

// ─── Mock SafetyScanAdapter ───────────────────────────────────────────────────

type mockSafetyScanAdapter struct {
	scanCSAMFn       func(ctx context.Context, storageKey string) (*CSAMScanResult, error)
	scanModerationFn func(ctx context.Context, storageKey string) (*ModerationResult, error)
	reportCSAMFn     func(ctx context.Context, uploadID uuid.UUID, scanResult *CSAMScanResult) error
}

func newMockSafetyScanAdapter() *mockSafetyScanAdapter {
	return &mockSafetyScanAdapter{}
}

func (m *mockSafetyScanAdapter) ScanCSAM(ctx context.Context, storageKey string) (*CSAMScanResult, error) {
	if m.scanCSAMFn != nil {
		return m.scanCSAMFn(ctx, storageKey)
	}
	panic("ScanCSAM not mocked")
}

func (m *mockSafetyScanAdapter) ScanModeration(ctx context.Context, storageKey string) (*ModerationResult, error) {
	if m.scanModerationFn != nil {
		return m.scanModerationFn(ctx, storageKey)
	}
	panic("ScanModeration not mocked")
}

func (m *mockSafetyScanAdapter) ReportCSAM(ctx context.Context, uploadID uuid.UUID, scanResult *CSAMScanResult) error {
	if m.reportCSAMFn != nil {
		return m.reportCSAMFn(ctx, uploadID, scanResult)
	}
	panic("ReportCSAM not mocked")
}

// ─── Mock EventBus ────────────────────────────────────────────────────────────

type mockEventBus struct {
	published []shared.DomainEvent
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{}
}

func (m *mockEventBus) Publish(_ context.Context, event shared.DomainEvent) error {
	m.published = append(m.published, event)
	return nil
}
