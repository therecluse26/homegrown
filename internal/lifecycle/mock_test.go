package lifecycle

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Repositories [15-data-lifecycle plan mock pattern]
// ═══════════════════════════════════════════════════════════════════════════════

// ─── stubExportRepo ─────────────────────────────────────────────────────────

type stubExportRepo struct {
	createFn       func(ctx context.Context, scope *shared.FamilyScope, input *CreateExportRequest) (*ExportRequest, error)
	findByIDFn     func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ExportRequest, error)
	listByFamilyFn func(ctx context.Context, scope *shared.FamilyScope, pagination *PaginationParams) ([]ExportRequest, error)
	updateStatusFn func(ctx context.Context, id uuid.UUID, status ExportStatus, archiveKey *string, sizeBytes *int64) error
}

func (s *stubExportRepo) Create(ctx context.Context, scope *shared.FamilyScope, input *CreateExportRequest) (*ExportRequest, error) {
	if s.createFn != nil {
		return s.createFn(ctx, scope, input)
	}
	panic("stubExportRepo.Create not stubbed")
}

func (s *stubExportRepo) FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ExportRequest, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, scope, id)
	}
	panic("stubExportRepo.FindByID not stubbed")
}

func (s *stubExportRepo) ListByFamily(ctx context.Context, scope *shared.FamilyScope, pagination *PaginationParams) ([]ExportRequest, error) {
	if s.listByFamilyFn != nil {
		return s.listByFamilyFn(ctx, scope, pagination)
	}
	panic("stubExportRepo.ListByFamily not stubbed")
}

func (s *stubExportRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status ExportStatus, archiveKey *string, sizeBytes *int64) error {
	if s.updateStatusFn != nil {
		return s.updateStatusFn(ctx, id, status, archiveKey, sizeBytes)
	}
	panic("stubExportRepo.UpdateStatus not stubbed")
}

// ─── stubDeletionRepo ────────────────────────────────────────────────────────

type stubDeletionRepo struct {
	createFn               func(ctx context.Context, scope *shared.FamilyScope, input *CreateDeletionRequest) (*DeletionRequest, error)
	findActiveByFamilyFn   func(ctx context.Context, scope *shared.FamilyScope) (*DeletionRequest, error)
	updateStatusFn         func(ctx context.Context, id uuid.UUID, status DeletionStatus) error
	updateDomainStatusFn   func(ctx context.Context, id uuid.UUID, domain string, completed bool) error
	cancelFn               func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error
	findReadyForDeletionFn func(ctx context.Context) ([]DeletionRequest, error)
}

func (s *stubDeletionRepo) Create(ctx context.Context, scope *shared.FamilyScope, input *CreateDeletionRequest) (*DeletionRequest, error) {
	if s.createFn != nil {
		return s.createFn(ctx, scope, input)
	}
	panic("stubDeletionRepo.Create not stubbed")
}

func (s *stubDeletionRepo) FindActiveByFamily(ctx context.Context, scope *shared.FamilyScope) (*DeletionRequest, error) {
	if s.findActiveByFamilyFn != nil {
		return s.findActiveByFamilyFn(ctx, scope)
	}
	panic("stubDeletionRepo.FindActiveByFamily not stubbed")
}

func (s *stubDeletionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status DeletionStatus) error {
	if s.updateStatusFn != nil {
		return s.updateStatusFn(ctx, id, status)
	}
	panic("stubDeletionRepo.UpdateStatus not stubbed")
}

func (s *stubDeletionRepo) UpdateDomainStatus(ctx context.Context, id uuid.UUID, domain string, completed bool) error {
	if s.updateDomainStatusFn != nil {
		return s.updateDomainStatusFn(ctx, id, domain, completed)
	}
	panic("stubDeletionRepo.UpdateDomainStatus not stubbed")
}

func (s *stubDeletionRepo) Cancel(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error {
	if s.cancelFn != nil {
		return s.cancelFn(ctx, scope, id)
	}
	panic("stubDeletionRepo.Cancel not stubbed")
}

func (s *stubDeletionRepo) FindReadyForDeletion(ctx context.Context) ([]DeletionRequest, error) {
	if s.findReadyForDeletionFn != nil {
		return s.findReadyForDeletionFn(ctx)
	}
	panic("stubDeletionRepo.FindReadyForDeletion not stubbed")
}

// ─── stubRecoveryRepo ────────────────────────────────────────────────────────

type stubRecoveryRepo struct {
	createFn   func(ctx context.Context, input *CreateRecoveryRequest) (*RecoveryRequest, error)
	findByIDFn func(ctx context.Context, id uuid.UUID) (*RecoveryRequest, error)
}

func (s *stubRecoveryRepo) Create(ctx context.Context, input *CreateRecoveryRequest) (*RecoveryRequest, error) {
	if s.createFn != nil {
		return s.createFn(ctx, input)
	}
	panic("stubRecoveryRepo.Create not stubbed")
}

func (s *stubRecoveryRepo) FindByID(ctx context.Context, id uuid.UUID) (*RecoveryRequest, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, id)
	}
	panic("stubRecoveryRepo.FindByID not stubbed")
}

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Cross-Domain Services
// ═══════════════════════════════════════════════════════════════════════════════

// ─── stubIamService ──────────────────────────────────────────────────────────

type stubIamService struct {
	initiateRecoveryFlowFn func(ctx context.Context, email string) error
	listSessionsFn         func(ctx context.Context, parentID uuid.UUID) ([]SessionInfo, error)
	revokeSessionFn        func(ctx context.Context, sessionID string) error
	revokeAllSessionsFn    func(ctx context.Context, parentID uuid.UUID, currentSessionID string) (uint32, error)
	revokeFamilySessionsFn func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubIamService) InitiateRecoveryFlow(ctx context.Context, email string) error {
	if s.initiateRecoveryFlowFn != nil {
		return s.initiateRecoveryFlowFn(ctx, email)
	}
	return nil // safe default
}

func (s *stubIamService) ListSessions(ctx context.Context, parentID uuid.UUID) ([]SessionInfo, error) {
	if s.listSessionsFn != nil {
		return s.listSessionsFn(ctx, parentID)
	}
	panic("stubIamService.ListSessions not stubbed")
}

func (s *stubIamService) RevokeSession(ctx context.Context, sessionID string) error {
	if s.revokeSessionFn != nil {
		return s.revokeSessionFn(ctx, sessionID)
	}
	panic("stubIamService.RevokeSession not stubbed")
}

func (s *stubIamService) RevokeAllSessions(ctx context.Context, parentID uuid.UUID, currentSessionID string) (uint32, error) {
	if s.revokeAllSessionsFn != nil {
		return s.revokeAllSessionsFn(ctx, parentID, currentSessionID)
	}
	panic("stubIamService.RevokeAllSessions not stubbed")
}

func (s *stubIamService) RevokeFamilySessions(ctx context.Context, familyID uuid.UUID) error {
	if s.revokeFamilySessionsFn != nil {
		return s.revokeFamilySessionsFn(ctx, familyID)
	}
	return nil // safe default
}

// ─── stubBillingService ──────────────────────────────────────────────────────

type stubBillingService struct {
	cancelFamilySubscriptionsFn func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubBillingService) CancelFamilySubscriptions(ctx context.Context, familyID uuid.UUID) error {
	if s.cancelFamilySubscriptionsFn != nil {
		return s.cancelFamilySubscriptionsFn(ctx, familyID)
	}
	return nil // safe default
}

// ─── stubExportHandler ───────────────────────────────────────────────────────

type stubExportHandler struct {
	domainName       string
	exportFamilyData func(ctx context.Context, familyID uuid.UUID, format ExportFormat) ([]ExportFile, error)
}

func (s *stubExportHandler) DomainName() string { return s.domainName }

func (s *stubExportHandler) ExportFamilyData(ctx context.Context, familyID uuid.UUID, format ExportFormat) ([]ExportFile, error) {
	if s.exportFamilyData != nil {
		return s.exportFamilyData(ctx, familyID, format)
	}
	return nil, nil
}

// ─── stubDeletionHandler ─────────────────────────────────────────────────────

type stubDeletionHandler struct {
	domainName        string
	deleteFamilyData  func(ctx context.Context, familyID uuid.UUID) error
	deleteStudentData func(ctx context.Context, familyID uuid.UUID, studentID uuid.UUID) error
}

func (s *stubDeletionHandler) DomainName() string { return s.domainName }

func (s *stubDeletionHandler) DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error {
	if s.deleteFamilyData != nil {
		return s.deleteFamilyData(ctx, familyID)
	}
	return nil
}

func (s *stubDeletionHandler) DeleteStudentData(ctx context.Context, familyID uuid.UUID, studentID uuid.UUID) error {
	if s.deleteStudentData != nil {
		return s.deleteStudentData(ctx, familyID, studentID)
	}
	return nil
}

// ─── stubJobEnqueuer ─────────────────────────────────────────────────────────

type stubJobEnqueuer struct {
	enqueueFn func(ctx context.Context, payload shared.JobPayload) error
}

func (s *stubJobEnqueuer) Enqueue(ctx context.Context, payload shared.JobPayload) error {
	if s.enqueueFn != nil {
		return s.enqueueFn(ctx, payload)
	}
	return nil
}

func (s *stubJobEnqueuer) EnqueueIn(_ context.Context, _ shared.JobPayload, _ time.Duration) error {
	return nil
}

func (s *stubJobEnqueuer) Close() error { return nil }

// ═══════════════════════════════════════════════════════════════════════════════
// Test Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func testScope() *shared.FamilyScope {
	auth := testAuth()
	s := shared.NewFamilyScopeFromAuth(auth)
	return &s
}

func testAuth() *shared.AuthContext {
	return &shared.AuthContext{
		ParentID:        uuid.Must(uuid.NewV7()),
		FamilyID:        uuid.Must(uuid.NewV7()),
		IdentityID:      uuid.Must(uuid.NewV7()),
		IsPrimaryParent: true,
	}
}

func testAuthWithSession(sessionID string) *shared.AuthContext {
	auth := testAuth()
	auth.SessionID = sessionID
	return auth
}

// newTestService creates a LifecycleService with the given dependencies.
func newTestService(
	exportRepo ExportRequestRepository,
	deletionRepo DeletionRequestRepository,
	recoveryRepo RecoveryRequestRepository,
	iamSvc IamServiceForLifecycle,
	billingSvc BillingServiceForLifecycle,
	jobs shared.JobEnqueuer,
	exportHandlers []ExportHandler,
	deletionHandlers []DeletionHandler,
) LifecycleService {
	return NewLifecycleService(
		exportRepo,
		deletionRepo,
		recoveryRepo,
		iamSvc,
		billingSvc,
		shared.NewEventBus(),
		jobs,
		exportHandlers,
		deletionHandlers,
	)
}
