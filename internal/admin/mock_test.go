package admin

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Repositories [16-admin plan mock pattern]
// ═══════════════════════════════════════════════════════════════════════════════

// ─── stubFlagRepo ───────────────────────────────────────────────────────────

type stubFlagRepo struct {
	listAllFn  func(ctx context.Context) ([]FeatureFlag, error)
	findByKeyFn func(ctx context.Context, key string) (*FeatureFlag, error)
	createFn   func(ctx context.Context, input *CreateFlagInput, adminID uuid.UUID) (*FeatureFlag, error)
	updateFn   func(ctx context.Context, key string, input *UpdateFlagInput, adminID uuid.UUID) (*FeatureFlag, error)
	deleteFn   func(ctx context.Context, key string) error
}

func (s *stubFlagRepo) ListAll(ctx context.Context) ([]FeatureFlag, error) {
	if s.listAllFn != nil {
		return s.listAllFn(ctx)
	}
	panic("stubFlagRepo.ListAll not stubbed")
}

func (s *stubFlagRepo) FindByKey(ctx context.Context, key string) (*FeatureFlag, error) {
	if s.findByKeyFn != nil {
		return s.findByKeyFn(ctx, key)
	}
	panic("stubFlagRepo.FindByKey not stubbed")
}

func (s *stubFlagRepo) Create(ctx context.Context, input *CreateFlagInput, adminID uuid.UUID) (*FeatureFlag, error) {
	if s.createFn != nil {
		return s.createFn(ctx, input, adminID)
	}
	panic("stubFlagRepo.Create not stubbed")
}

func (s *stubFlagRepo) Update(ctx context.Context, key string, input *UpdateFlagInput, adminID uuid.UUID) (*FeatureFlag, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, key, input, adminID)
	}
	panic("stubFlagRepo.Update not stubbed")
}

func (s *stubFlagRepo) Delete(ctx context.Context, key string) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, key)
	}
	panic("stubFlagRepo.Delete not stubbed")
}

// ─── stubAuditRepo ──────────────────────────────────────────────────────────

type stubAuditRepo struct {
	createFn       func(ctx context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error)
	searchFn       func(ctx context.Context, query *AuditLogQuery, pagination *shared.PaginationParams) ([]AuditLogEntry, error)
	findByTargetFn func(ctx context.Context, targetType string, targetID uuid.UUID, pagination *shared.PaginationParams) ([]AuditLogEntry, error)
}

func (s *stubAuditRepo) Create(ctx context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
	if s.createFn != nil {
		return s.createFn(ctx, entry)
	}
	return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil // safe default for audit logging
}

func (s *stubAuditRepo) Search(ctx context.Context, query *AuditLogQuery, pagination *shared.PaginationParams) ([]AuditLogEntry, error) {
	if s.searchFn != nil {
		return s.searchFn(ctx, query, pagination)
	}
	panic("stubAuditRepo.Search not stubbed")
}

func (s *stubAuditRepo) FindByTarget(ctx context.Context, targetType string, targetID uuid.UUID, pagination *shared.PaginationParams) ([]AuditLogEntry, error) {
	if s.findByTargetFn != nil {
		return s.findByTargetFn(ctx, targetType, targetID, pagination)
	}
	panic("stubAuditRepo.FindByTarget not stubbed")
}

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Cross-Domain Services
// ═══════════════════════════════════════════════════════════════════════════════

// ─── stubIamService ─────────────────────────────────────────────────────────

type stubIamService struct {
	searchUsersFn    func(ctx context.Context, query *UserSearchQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AdminUserSummary], error)
	getFamilyDetailFn func(ctx context.Context, familyID uuid.UUID) (*AdminFamilyInfo, error)
	getParentsFn     func(ctx context.Context, familyID uuid.UUID) ([]AdminParentInfo, error)
	getStudentsFn    func(ctx context.Context, familyID uuid.UUID) ([]AdminStudentInfo, error)
}

func (s *stubIamService) SearchUsers(ctx context.Context, query *UserSearchQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AdminUserSummary], error) {
	if s.searchUsersFn != nil {
		return s.searchUsersFn(ctx, query, pagination)
	}
	panic("stubIamService.SearchUsers not stubbed")
}

func (s *stubIamService) GetFamilyDetail(ctx context.Context, familyID uuid.UUID) (*AdminFamilyInfo, error) {
	if s.getFamilyDetailFn != nil {
		return s.getFamilyDetailFn(ctx, familyID)
	}
	panic("stubIamService.GetFamilyDetail not stubbed")
}

func (s *stubIamService) GetParents(ctx context.Context, familyID uuid.UUID) ([]AdminParentInfo, error) {
	if s.getParentsFn != nil {
		return s.getParentsFn(ctx, familyID)
	}
	panic("stubIamService.GetParents not stubbed")
}

func (s *stubIamService) GetStudents(ctx context.Context, familyID uuid.UUID) ([]AdminStudentInfo, error) {
	if s.getStudentsFn != nil {
		return s.getStudentsFn(ctx, familyID)
	}
	panic("stubIamService.GetStudents not stubbed")
}

// ─── stubSafetyService ──────────────────────────────────────────────────────

type stubSafetyService struct {
	getModerationHistoryFn func(ctx context.Context, familyID uuid.UUID) ([]ModerationActionSummary, error)
}

func (s *stubSafetyService) GetModerationHistory(ctx context.Context, familyID uuid.UUID) ([]ModerationActionSummary, error) {
	if s.getModerationHistoryFn != nil {
		return s.getModerationHistoryFn(ctx, familyID)
	}
	return nil, nil // safe default
}

// ─── stubBillingService ─────────────────────────────────────────────────────

type stubBillingService struct {
	getSubscriptionInfoFn func(ctx context.Context, familyID uuid.UUID) (*AdminSubscriptionInfo, error)
}

func (s *stubBillingService) GetSubscriptionInfo(ctx context.Context, familyID uuid.UUID) (*AdminSubscriptionInfo, error) {
	if s.getSubscriptionInfoFn != nil {
		return s.getSubscriptionInfoFn(ctx, familyID)
	}
	return nil, nil // safe default
}

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Infrastructure
// ═══════════════════════════════════════════════════════════════════════════════

// ─── stubCache ──────────────────────────────────────────────────────────────

type stubCache struct {
	getFn                  func(ctx context.Context, key string) (string, error)
	setFn                  func(ctx context.Context, key string, value string, ttl time.Duration) error
	deleteFn               func(ctx context.Context, key string) error
	incrementWithExpiryFn  func(ctx context.Context, key string, window time.Duration) (int64, error)
}

func (s *stubCache) Get(ctx context.Context, key string) (string, error) {
	if s.getFn != nil {
		return s.getFn(ctx, key)
	}
	return "", nil // cache miss
}

func (s *stubCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if s.setFn != nil {
		return s.setFn(ctx, key, value, ttl)
	}
	return nil
}

func (s *stubCache) Delete(ctx context.Context, key string) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, key)
	}
	return nil
}

func (s *stubCache) IncrementWithExpiry(ctx context.Context, key string, window time.Duration) (int64, error) {
	if s.incrementWithExpiryFn != nil {
		return s.incrementWithExpiryFn(ctx, key, window)
	}
	return 0, nil
}

func (s *stubCache) Close() error { return nil }

// ─── stubHealthChecker ──────────────────────────────────────────────────────

type stubHealthChecker struct {
	checkAllFn func(ctx context.Context) []ComponentHealth
}

func (s *stubHealthChecker) CheckAll(ctx context.Context) []ComponentHealth {
	if s.checkAllFn != nil {
		return s.checkAllFn(ctx)
	}
	return nil
}

// ─── stubJobInspector ───────────────────────────────────────────────────────

type stubJobInspector struct {
	getQueueStatusFn    func(ctx context.Context) (*JobStatusResponse, error)
	getDeadLetterJobsFn func(ctx context.Context, pagination *shared.PaginationParams) ([]DeadLetterJob, error)
	retryDeadLetterJobFn func(ctx context.Context, jobID string) error
}

func (s *stubJobInspector) GetQueueStatus(ctx context.Context) (*JobStatusResponse, error) {
	if s.getQueueStatusFn != nil {
		return s.getQueueStatusFn(ctx)
	}
	panic("stubJobInspector.GetQueueStatus not stubbed")
}

func (s *stubJobInspector) GetDeadLetterJobs(ctx context.Context, pagination *shared.PaginationParams) ([]DeadLetterJob, error) {
	if s.getDeadLetterJobsFn != nil {
		return s.getDeadLetterJobsFn(ctx, pagination)
	}
	panic("stubJobInspector.GetDeadLetterJobs not stubbed")
}

func (s *stubJobInspector) RetryDeadLetterJob(ctx context.Context, jobID string) error {
	if s.retryDeadLetterJobFn != nil {
		return s.retryDeadLetterJobFn(ctx, jobID)
	}
	panic("stubJobInspector.RetryDeadLetterJob not stubbed")
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func testAuth() *shared.AuthContext {
	return &shared.AuthContext{
		ParentID:        uuid.Must(uuid.NewV7()),
		FamilyID:        uuid.Must(uuid.NewV7()),
		IdentityID:      uuid.Must(uuid.NewV7()),
		IsPrimaryParent: true,
		IsPlatformAdmin: true,
	}
}

func newTestService(
	flagRepo FeatureFlagRepository,
	auditRepo AuditLogRepository,
	cache shared.Cache,
	iamSvc IamServiceForAdmin,
	safetySvc SafetyServiceForAdmin,
	billingSvc BillingServiceForAdmin,
	healthChecker HealthChecker,
	jobInspector JobInspector,
) AdminService {
	return NewAdminService(
		flagRepo, auditRepo, cache,
		iamSvc, safetySvc, billingSvc,
		healthChecker, jobInspector,
	)
}

func defaultPagination() *shared.PaginationParams {
	return &shared.PaginationParams{}
}
