package safety

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Mock Repositories ──────────────────────────────────────────────────────────

type mockReportRepo struct {
	createFn                    func(ctx context.Context, scope shared.FamilyScope, input CreateReportRow) (*Report, error)
	findByIDFn                  func(ctx context.Context, scope shared.FamilyScope, reportID uuid.UUID) (*Report, error)
	findByIDUnscopedFn          func(ctx context.Context, reportID uuid.UUID) (*Report, error)
	listByReporterFn            func(ctx context.Context, scope shared.FamilyScope, pagination shared.PaginationParams) ([]Report, error)
	listFilteredFn              func(ctx context.Context, filter ReportFilter, pagination shared.PaginationParams) ([]Report, error)
	updateFn                    func(ctx context.Context, reportID uuid.UUID, updates ReportUpdate) (*Report, error)
	existsRecentFn              func(ctx context.Context, scope shared.FamilyScope, targetType string, targetID uuid.UUID, withinHours uint32) (bool, error)
	countByStatusFn             func(ctx context.Context, status string) (int64, error)
	countByStatusAndPriorityFn  func(ctx context.Context, status string, priority string) (int64, error)
	countSinceFn                func(ctx context.Context, since string) (int64, error)
}

func newMockReportRepo() *mockReportRepo { return &mockReportRepo{} }

func (m *mockReportRepo) Create(ctx context.Context, scope shared.FamilyScope, input CreateReportRow) (*Report, error) {
	if m.createFn != nil {
		return m.createFn(ctx, scope, input)
	}
	panic("ReportRepo.Create not mocked")
}
func (m *mockReportRepo) FindByID(ctx context.Context, scope shared.FamilyScope, id uuid.UUID) (*Report, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, scope, id)
	}
	panic("ReportRepo.FindByID not mocked")
}
func (m *mockReportRepo) FindByIDUnscoped(ctx context.Context, id uuid.UUID) (*Report, error) {
	if m.findByIDUnscopedFn != nil {
		return m.findByIDUnscopedFn(ctx, id)
	}
	panic("ReportRepo.FindByIDUnscoped not mocked")
}
func (m *mockReportRepo) ListByReporter(ctx context.Context, scope shared.FamilyScope, p shared.PaginationParams) ([]Report, error) {
	if m.listByReporterFn != nil {
		return m.listByReporterFn(ctx, scope, p)
	}
	panic("ReportRepo.ListByReporter not mocked")
}
func (m *mockReportRepo) ListFiltered(ctx context.Context, f ReportFilter, p shared.PaginationParams) ([]Report, error) {
	if m.listFilteredFn != nil {
		return m.listFilteredFn(ctx, f, p)
	}
	panic("ReportRepo.ListFiltered not mocked")
}
func (m *mockReportRepo) Update(ctx context.Context, id uuid.UUID, u ReportUpdate) (*Report, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, u)
	}
	panic("ReportRepo.Update not mocked")
}
func (m *mockReportRepo) ExistsRecent(ctx context.Context, scope shared.FamilyScope, targetType string, targetID uuid.UUID, withinHours uint32) (bool, error) {
	if m.existsRecentFn != nil {
		return m.existsRecentFn(ctx, scope, targetType, targetID, withinHours)
	}
	panic("ReportRepo.ExistsRecent not mocked")
}
func (m *mockReportRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx, status)
	}
	return 0, nil
}
func (m *mockReportRepo) CountByStatusAndPriority(ctx context.Context, status string, priority string) (int64, error) {
	if m.countByStatusAndPriorityFn != nil {
		return m.countByStatusAndPriorityFn(ctx, status, priority)
	}
	return 0, nil
}
func (m *mockReportRepo) CountSince(ctx context.Context, since string) (int64, error) {
	if m.countSinceFn != nil {
		return m.countSinceFn(ctx, since)
	}
	return 0, nil
}

type mockFlagRepo struct {
	createFn          func(ctx context.Context, input CreateContentFlagRow) (*ContentFlag, error)
	findByIDFn        func(ctx context.Context, flagID uuid.UUID) (*ContentFlag, error)
	listFilteredFn    func(ctx context.Context, filter FlagFilter, pagination shared.PaginationParams) ([]ContentFlag, error)
	markReviewedFn    func(ctx context.Context, flagID uuid.UUID, reviewedBy uuid.UUID, actionTaken bool) (*ContentFlag, error)
	countUnreviewedFn func(ctx context.Context) (int64, error)
}

func newMockFlagRepo() *mockFlagRepo { return &mockFlagRepo{} }

func (m *mockFlagRepo) Create(ctx context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	panic("FlagRepo.Create not mocked")
}
func (m *mockFlagRepo) FindByID(ctx context.Context, id uuid.UUID) (*ContentFlag, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	panic("FlagRepo.FindByID not mocked")
}
func (m *mockFlagRepo) ListFiltered(ctx context.Context, f FlagFilter, p shared.PaginationParams) ([]ContentFlag, error) {
	if m.listFilteredFn != nil {
		return m.listFilteredFn(ctx, f, p)
	}
	panic("FlagRepo.ListFiltered not mocked")
}
func (m *mockFlagRepo) MarkReviewed(ctx context.Context, id uuid.UUID, reviewedBy uuid.UUID, actionTaken bool) (*ContentFlag, error) {
	if m.markReviewedFn != nil {
		return m.markReviewedFn(ctx, id, reviewedBy, actionTaken)
	}
	panic("FlagRepo.MarkReviewed not mocked")
}
func (m *mockFlagRepo) CountUnreviewed(ctx context.Context) (int64, error) {
	if m.countUnreviewedFn != nil {
		return m.countUnreviewedFn(ctx)
	}
	return 0, nil
}

type mockActionRepo struct {
	createFn             func(ctx context.Context, input CreateModActionRow) (*ModAction, error)
	findByIDFn           func(ctx context.Context, actionID uuid.UUID) (*ModAction, error)
	listFilteredFn       func(ctx context.Context, filter ActionFilter, pagination shared.PaginationParams) ([]ModAction, error)
	listByTargetFamilyFn func(ctx context.Context, familyID uuid.UUID, pagination shared.PaginationParams) ([]ModAction, error)
	countSinceFn         func(ctx context.Context, since string) (int64, error)
}

func newMockActionRepo() *mockActionRepo { return &mockActionRepo{} }

func (m *mockActionRepo) Create(ctx context.Context, input CreateModActionRow) (*ModAction, error) {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	panic("ActionRepo.Create not mocked")
}
func (m *mockActionRepo) FindByID(ctx context.Context, id uuid.UUID) (*ModAction, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	panic("ActionRepo.FindByID not mocked")
}
func (m *mockActionRepo) ListFiltered(ctx context.Context, f ActionFilter, p shared.PaginationParams) ([]ModAction, error) {
	if m.listFilteredFn != nil {
		return m.listFilteredFn(ctx, f, p)
	}
	panic("ActionRepo.ListFiltered not mocked")
}
func (m *mockActionRepo) ListByTargetFamily(ctx context.Context, familyID uuid.UUID, p shared.PaginationParams) ([]ModAction, error) {
	if m.listByTargetFamilyFn != nil {
		return m.listByTargetFamilyFn(ctx, familyID, p)
	}
	panic("ActionRepo.ListByTargetFamily not mocked")
}
func (m *mockActionRepo) CountSince(ctx context.Context, since string) (int64, error) {
	if m.countSinceFn != nil {
		return m.countSinceFn(ctx, since)
	}
	return 0, nil
}

type mockAccountStatusRepo struct {
	getOrCreateFn   func(ctx context.Context, familyID uuid.UUID) (*AccountStatusRow, error)
	updateFn        func(ctx context.Context, familyID uuid.UUID, updates AccountStatusUpdate) (*AccountStatusRow, error)
	countByStatusFn func(ctx context.Context, status string) (int64, error)
}

func newMockAccountStatusRepo() *mockAccountStatusRepo { return &mockAccountStatusRepo{} }

func (m *mockAccountStatusRepo) GetOrCreate(ctx context.Context, familyID uuid.UUID) (*AccountStatusRow, error) {
	if m.getOrCreateFn != nil {
		return m.getOrCreateFn(ctx, familyID)
	}
	panic("AccountStatusRepo.GetOrCreate not mocked")
}
func (m *mockAccountStatusRepo) Update(ctx context.Context, familyID uuid.UUID, updates AccountStatusUpdate) (*AccountStatusRow, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, familyID, updates)
	}
	panic("AccountStatusRepo.Update not mocked")
}
func (m *mockAccountStatusRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx, status)
	}
	return 0, nil
}

type mockAppealRepo struct {
	createFn          func(ctx context.Context, scope shared.FamilyScope, input CreateAppealRow) (*Appeal, error)
	findByIDFn        func(ctx context.Context, scope shared.FamilyScope, appealID uuid.UUID) (*Appeal, error)
	findByIDUnscopedFn func(ctx context.Context, appealID uuid.UUID) (*Appeal, error)
	findByActionIDFn  func(ctx context.Context, actionID uuid.UUID) (*Appeal, error)
	listFilteredFn    func(ctx context.Context, filter AppealFilter, pagination shared.PaginationParams) ([]Appeal, error)
	updateFn          func(ctx context.Context, appealID uuid.UUID, updates AppealUpdate) (*Appeal, error)
	countByStatusFn   func(ctx context.Context, status string) (int64, error)
}

func newMockAppealRepo() *mockAppealRepo { return &mockAppealRepo{} }

func (m *mockAppealRepo) Create(ctx context.Context, scope shared.FamilyScope, input CreateAppealRow) (*Appeal, error) {
	if m.createFn != nil {
		return m.createFn(ctx, scope, input)
	}
	panic("AppealRepo.Create not mocked")
}
func (m *mockAppealRepo) FindByID(ctx context.Context, scope shared.FamilyScope, id uuid.UUID) (*Appeal, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, scope, id)
	}
	panic("AppealRepo.FindByID not mocked")
}
func (m *mockAppealRepo) FindByIDUnscoped(ctx context.Context, id uuid.UUID) (*Appeal, error) {
	if m.findByIDUnscopedFn != nil {
		return m.findByIDUnscopedFn(ctx, id)
	}
	panic("AppealRepo.FindByIDUnscoped not mocked")
}
func (m *mockAppealRepo) FindByActionID(ctx context.Context, actionID uuid.UUID) (*Appeal, error) {
	if m.findByActionIDFn != nil {
		return m.findByActionIDFn(ctx, actionID)
	}
	panic("AppealRepo.FindByActionID not mocked")
}
func (m *mockAppealRepo) ListFiltered(ctx context.Context, f AppealFilter, p shared.PaginationParams) ([]Appeal, error) {
	if m.listFilteredFn != nil {
		return m.listFilteredFn(ctx, f, p)
	}
	panic("AppealRepo.ListFiltered not mocked")
}
func (m *mockAppealRepo) Update(ctx context.Context, id uuid.UUID, u AppealUpdate) (*Appeal, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, u)
	}
	panic("AppealRepo.Update not mocked")
}
func (m *mockAppealRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx, status)
	}
	return 0, nil
}

type mockNcmecRepo struct {
	createFn       func(ctx context.Context, input CreateNcmecReportRow) (*NcmecReport, error)
	findByIDFn     func(ctx context.Context, reportID uuid.UUID) (*NcmecReport, error)
	updateStatusFn func(ctx context.Context, reportID uuid.UUID, status string, ncmecReportID *string, errMsg *string) (*NcmecReport, error)
	findPendingFn  func(ctx context.Context) ([]NcmecReport, error)
}

func newMockNcmecRepo() *mockNcmecRepo { return &mockNcmecRepo{} }

func (m *mockNcmecRepo) Create(ctx context.Context, input CreateNcmecReportRow) (*NcmecReport, error) {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	panic("NcmecRepo.Create not mocked")
}
func (m *mockNcmecRepo) FindByID(ctx context.Context, id uuid.UUID) (*NcmecReport, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	panic("NcmecRepo.FindByID not mocked")
}
func (m *mockNcmecRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, ncmecReportID *string, errMsg *string) (*NcmecReport, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, ncmecReportID, errMsg)
	}
	panic("NcmecRepo.UpdateStatus not mocked")
}
func (m *mockNcmecRepo) FindPending(ctx context.Context) ([]NcmecReport, error) {
	if m.findPendingFn != nil {
		return m.findPendingFn(ctx)
	}
	panic("NcmecRepo.FindPending not mocked")
}

type mockBotSignalRepo struct {
	createFn      func(ctx context.Context, input CreateBotSignalRow) (*BotSignal, error)
	countRecentFn func(ctx context.Context, parentID uuid.UUID, withinMinutes uint32) (int64, error)
}

func newMockBotSignalRepo() *mockBotSignalRepo { return &mockBotSignalRepo{} }

func (m *mockBotSignalRepo) Create(ctx context.Context, input CreateBotSignalRow) (*BotSignal, error) {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	panic("BotSignalRepo.Create not mocked")
}
func (m *mockBotSignalRepo) CountRecent(ctx context.Context, parentID uuid.UUID, withinMinutes uint32) (int64, error) {
	if m.countRecentFn != nil {
		return m.countRecentFn(ctx, parentID, withinMinutes)
	}
	panic("BotSignalRepo.CountRecent not mocked")
}

// ─── Mock Adapters ──────────────────────────────────────────────────────────────

type mockIamService struct {
	revokeSessionsFn func(ctx context.Context, familyID uuid.UUID) error
}

func newMockIamService() *mockIamService { return &mockIamService{} }

func (m *mockIamService) RevokeSessions(ctx context.Context, familyID uuid.UUID) error {
	if m.revokeSessionsFn != nil {
		return m.revokeSessionsFn(ctx, familyID)
	}
	return nil
}

// ─── Mock Cache ─────────────────────────────────────────────────────────────────

type mockCache struct {
	data map[string]string
}

func newMockCache() *mockCache {
	return &mockCache{data: make(map[string]string)}
}

func (m *mockCache) Get(_ context.Context, key string) (string, error) {
	return m.data[key], nil
}

func (m *mockCache) Set(_ context.Context, key string, value string, _ time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(_ context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache) IncrementWithExpiry(_ context.Context, _ string, _ time.Duration) (int64, error) {
	return 0, nil
}

func (m *mockCache) Close() error { return nil }

// ─── Mock EventBus ──────────────────────────────────────────────────────────────

type mockEventBus struct {
	published []shared.DomainEvent
}

func newMockEventBus() *mockEventBus { return &mockEventBus{} }

func (m *mockEventBus) Publish(_ context.Context, event shared.DomainEvent) error {
	m.published = append(m.published, event)
	return nil
}

// ─── Mock JobEnqueuer ────────────────────────────────────────────────────────────

type mockJobEnqueuer struct {
	enqueued []shared.JobPayload
}

func newMockJobEnqueuer() *mockJobEnqueuer { return &mockJobEnqueuer{} }

func (m *mockJobEnqueuer) Enqueue(_ context.Context, payload shared.JobPayload) error {
	m.enqueued = append(m.enqueued, payload)
	return nil
}

func (m *mockJobEnqueuer) EnqueueIn(_ context.Context, payload shared.JobPayload, _ time.Duration) error {
	m.enqueued = append(m.enqueued, payload)
	return nil
}

func (m *mockJobEnqueuer) Close() error { return nil }

// ─── Mock SafetyService ─────────────────────────────────────────────────────────

type mockSafetyService struct {
	// User-facing queries
	listMyReportsFn  func(ctx context.Context, scope shared.FamilyScope, p shared.PaginationParams) (*shared.PaginatedResponse[ReportResponse], error)
	getMyReportFn    func(ctx context.Context, scope shared.FamilyScope, id uuid.UUID) (*ReportResponse, error)
	getAccountStatusFn func(ctx context.Context, scope shared.FamilyScope) (*AccountStatusResponse, error)
	getMyAppealFn    func(ctx context.Context, scope shared.FamilyScope, id uuid.UUID) (*AppealResponse, error)

	// User-facing commands
	submitReportFn func(ctx context.Context, scope shared.FamilyScope, auth *shared.AuthContext, cmd CreateReportCommand) (*ReportResponse, error)
	submitAppealFn func(ctx context.Context, scope shared.FamilyScope, cmd CreateAppealCommand) (*AppealResponse, error)

	// Admin queries
	adminListReportsFn  func(ctx context.Context, auth *shared.AuthContext, f ReportFilter, p shared.PaginationParams) (*shared.PaginatedResponse[AdminReportResponse], error)
	adminGetReportFn    func(ctx context.Context, auth *shared.AuthContext, id uuid.UUID) (*AdminReportResponse, error)
	adminListFlagsFn    func(ctx context.Context, auth *shared.AuthContext, f FlagFilter, p shared.PaginationParams) (*shared.PaginatedResponse[ContentFlagResponse], error)
	adminListActionsFn  func(ctx context.Context, auth *shared.AuthContext, f ActionFilter, p shared.PaginationParams) (*shared.PaginatedResponse[ModActionResponse], error)
	adminGetAccountFn   func(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID) (*AdminAccountStatusResponse, error)
	adminListAppealsFn  func(ctx context.Context, auth *shared.AuthContext, f AppealFilter, p shared.PaginationParams) (*shared.PaginatedResponse[AdminAppealResponse], error)
	adminDashboardFn    func(ctx context.Context, auth *shared.AuthContext) (*DashboardStats, error)

	// Admin commands
	adminUpdateReportFn    func(ctx context.Context, auth *shared.AuthContext, id uuid.UUID, cmd UpdateReportCommand) (*AdminReportResponse, error)
	adminReviewFlagFn      func(ctx context.Context, auth *shared.AuthContext, id uuid.UUID, cmd ReviewFlagCommand) (*ContentFlagResponse, error)
	adminTakeActionFn      func(ctx context.Context, auth *shared.AuthContext, cmd CreateModActionCommand) (*ModActionResponse, error)
	adminSuspendAccountFn  func(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd SuspendAccountCommand) (*AdminAccountStatusResponse, error)
	adminBanAccountFn      func(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd BanAccountCommand) (*AdminAccountStatusResponse, error)
	adminLiftSuspensionFn  func(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd LiftSuspensionCommand) (*AdminAccountStatusResponse, error)
	adminResolveAppealFn   func(ctx context.Context, auth *shared.AuthContext, appealID uuid.UUID, cmd ResolveAppealCommand) (*AdminAppealResponse, error)

	// Internal
	checkAccountAccessFn    func(ctx context.Context, familyID uuid.UUID) error
	scanTextFn              func(ctx context.Context, text string) (*TextScanResult, error)
	recordBotSignalFn       func(ctx context.Context, familyID, parentID uuid.UUID, signal BotSignalType, details json.RawMessage) error
	handleCsamDetectionFn   func(ctx context.Context, uploadID, familyID uuid.UUID, result *CsamScanResult) error
	adminEscalateToCsamFn   func(ctx context.Context, auth *shared.AuthContext, flagID uuid.UUID, cmd EscalateCsamCommand) error
}

func (m *mockSafetyService) ListMyReports(ctx context.Context, scope shared.FamilyScope, p shared.PaginationParams) (*shared.PaginatedResponse[ReportResponse], error) {
	if m.listMyReportsFn != nil {
		return m.listMyReportsFn(ctx, scope, p)
	}
	return &shared.PaginatedResponse[ReportResponse]{Data: []ReportResponse{}}, nil
}
func (m *mockSafetyService) GetMyReport(ctx context.Context, scope shared.FamilyScope, id uuid.UUID) (*ReportResponse, error) {
	if m.getMyReportFn != nil {
		return m.getMyReportFn(ctx, scope, id)
	}
	panic("GetMyReport not mocked")
}
func (m *mockSafetyService) GetAccountStatus(ctx context.Context, scope shared.FamilyScope) (*AccountStatusResponse, error) {
	if m.getAccountStatusFn != nil {
		return m.getAccountStatusFn(ctx, scope)
	}
	return &AccountStatusResponse{Status: "active"}, nil
}
func (m *mockSafetyService) GetMyAppeal(ctx context.Context, scope shared.FamilyScope, id uuid.UUID) (*AppealResponse, error) {
	if m.getMyAppealFn != nil {
		return m.getMyAppealFn(ctx, scope, id)
	}
	panic("GetMyAppeal not mocked")
}
func (m *mockSafetyService) SubmitReport(ctx context.Context, scope shared.FamilyScope, auth *shared.AuthContext, cmd CreateReportCommand) (*ReportResponse, error) {
	if m.submitReportFn != nil {
		return m.submitReportFn(ctx, scope, auth, cmd)
	}
	panic("SubmitReport not mocked")
}
func (m *mockSafetyService) SubmitAppeal(ctx context.Context, scope shared.FamilyScope, cmd CreateAppealCommand) (*AppealResponse, error) {
	if m.submitAppealFn != nil {
		return m.submitAppealFn(ctx, scope, cmd)
	}
	panic("SubmitAppeal not mocked")
}
func (m *mockSafetyService) AdminListReports(ctx context.Context, auth *shared.AuthContext, f ReportFilter, p shared.PaginationParams) (*shared.PaginatedResponse[AdminReportResponse], error) {
	if m.adminListReportsFn != nil {
		return m.adminListReportsFn(ctx, auth, f, p)
	}
	return &shared.PaginatedResponse[AdminReportResponse]{Data: []AdminReportResponse{}}, nil
}
func (m *mockSafetyService) AdminGetReport(ctx context.Context, auth *shared.AuthContext, id uuid.UUID) (*AdminReportResponse, error) {
	if m.adminGetReportFn != nil {
		return m.adminGetReportFn(ctx, auth, id)
	}
	panic("AdminGetReport not mocked")
}
func (m *mockSafetyService) AdminListFlags(ctx context.Context, auth *shared.AuthContext, f FlagFilter, p shared.PaginationParams) (*shared.PaginatedResponse[ContentFlagResponse], error) {
	if m.adminListFlagsFn != nil {
		return m.adminListFlagsFn(ctx, auth, f, p)
	}
	return &shared.PaginatedResponse[ContentFlagResponse]{Data: []ContentFlagResponse{}}, nil
}
func (m *mockSafetyService) AdminListActions(ctx context.Context, auth *shared.AuthContext, f ActionFilter, p shared.PaginationParams) (*shared.PaginatedResponse[ModActionResponse], error) {
	if m.adminListActionsFn != nil {
		return m.adminListActionsFn(ctx, auth, f, p)
	}
	return &shared.PaginatedResponse[ModActionResponse]{Data: []ModActionResponse{}}, nil
}
func (m *mockSafetyService) AdminGetAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID) (*AdminAccountStatusResponse, error) {
	if m.adminGetAccountFn != nil {
		return m.adminGetAccountFn(ctx, auth, familyID)
	}
	panic("AdminGetAccount not mocked")
}
func (m *mockSafetyService) AdminListAppeals(ctx context.Context, auth *shared.AuthContext, f AppealFilter, p shared.PaginationParams) (*shared.PaginatedResponse[AdminAppealResponse], error) {
	if m.adminListAppealsFn != nil {
		return m.adminListAppealsFn(ctx, auth, f, p)
	}
	return &shared.PaginatedResponse[AdminAppealResponse]{Data: []AdminAppealResponse{}}, nil
}
func (m *mockSafetyService) AdminDashboard(ctx context.Context, auth *shared.AuthContext) (*DashboardStats, error) {
	if m.adminDashboardFn != nil {
		return m.adminDashboardFn(ctx, auth)
	}
	return &DashboardStats{}, nil
}
func (m *mockSafetyService) AdminUpdateReport(ctx context.Context, auth *shared.AuthContext, id uuid.UUID, cmd UpdateReportCommand) (*AdminReportResponse, error) {
	if m.adminUpdateReportFn != nil {
		return m.adminUpdateReportFn(ctx, auth, id, cmd)
	}
	panic("AdminUpdateReport not mocked")
}
func (m *mockSafetyService) AdminReviewFlag(ctx context.Context, auth *shared.AuthContext, id uuid.UUID, cmd ReviewFlagCommand) (*ContentFlagResponse, error) {
	if m.adminReviewFlagFn != nil {
		return m.adminReviewFlagFn(ctx, auth, id, cmd)
	}
	panic("AdminReviewFlag not mocked")
}
func (m *mockSafetyService) AdminTakeAction(ctx context.Context, auth *shared.AuthContext, cmd CreateModActionCommand) (*ModActionResponse, error) {
	if m.adminTakeActionFn != nil {
		return m.adminTakeActionFn(ctx, auth, cmd)
	}
	panic("AdminTakeAction not mocked")
}
func (m *mockSafetyService) AdminSuspendAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd SuspendAccountCommand) (*AdminAccountStatusResponse, error) {
	if m.adminSuspendAccountFn != nil {
		return m.adminSuspendAccountFn(ctx, auth, familyID, cmd)
	}
	panic("AdminSuspendAccount not mocked")
}
func (m *mockSafetyService) AdminBanAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd BanAccountCommand) (*AdminAccountStatusResponse, error) {
	if m.adminBanAccountFn != nil {
		return m.adminBanAccountFn(ctx, auth, familyID, cmd)
	}
	panic("AdminBanAccount not mocked")
}
func (m *mockSafetyService) AdminLiftSuspension(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd LiftSuspensionCommand) (*AdminAccountStatusResponse, error) {
	if m.adminLiftSuspensionFn != nil {
		return m.adminLiftSuspensionFn(ctx, auth, familyID, cmd)
	}
	panic("AdminLiftSuspension not mocked")
}
func (m *mockSafetyService) AdminResolveAppeal(ctx context.Context, auth *shared.AuthContext, appealID uuid.UUID, cmd ResolveAppealCommand) (*AdminAppealResponse, error) {
	if m.adminResolveAppealFn != nil {
		return m.adminResolveAppealFn(ctx, auth, appealID, cmd)
	}
	panic("AdminResolveAppeal not mocked")
}
func (m *mockSafetyService) CheckAccountAccess(ctx context.Context, familyID uuid.UUID) error {
	if m.checkAccountAccessFn != nil {
		return m.checkAccountAccessFn(ctx, familyID)
	}
	return nil
}
func (m *mockSafetyService) ScanText(ctx context.Context, text string) (*TextScanResult, error) {
	if m.scanTextFn != nil {
		return m.scanTextFn(ctx, text)
	}
	return &TextScanResult{Severity: "none"}, nil
}
func (m *mockSafetyService) RecordBotSignal(ctx context.Context, familyID, parentID uuid.UUID, signal BotSignalType, details json.RawMessage) error {
	if m.recordBotSignalFn != nil {
		return m.recordBotSignalFn(ctx, familyID, parentID, signal, details)
	}
	return nil
}
func (m *mockSafetyService) HandleCsamDetection(ctx context.Context, uploadID, familyID uuid.UUID, result *CsamScanResult) error {
	if m.handleCsamDetectionFn != nil {
		return m.handleCsamDetectionFn(ctx, uploadID, familyID, result)
	}
	return nil
}
func (m *mockSafetyService) AdminEscalateToCsam(ctx context.Context, auth *shared.AuthContext, flagID uuid.UUID, cmd EscalateCsamCommand) error {
	if m.adminEscalateToCsamFn != nil {
		return m.adminEscalateToCsamFn(ctx, auth, flagID, cmd)
	}
	return nil
}

// ─── Mock ThornAdapter ──────────────────────────────────────────────────────────

type mockThornAdapter struct {
	scanCsamFn          func(ctx context.Context, storageKey string) (*CsamScanResult, error)
	submitNcmecReportFn func(ctx context.Context, report NcmecReportPayload) (*NcmecSubmissionResult, error)
	checkHashUpdateFn   func(ctx context.Context) (bool, error)
}

func newMockThornAdapter() *mockThornAdapter { return &mockThornAdapter{} }

func (m *mockThornAdapter) ScanCsam(ctx context.Context, storageKey string) (*CsamScanResult, error) {
	if m.scanCsamFn != nil {
		return m.scanCsamFn(ctx, storageKey)
	}
	return &CsamScanResult{}, nil
}
func (m *mockThornAdapter) SubmitNcmecReport(ctx context.Context, report NcmecReportPayload) (*NcmecSubmissionResult, error) {
	if m.submitNcmecReportFn != nil {
		return m.submitNcmecReportFn(ctx, report)
	}
	panic("SubmitNcmecReport not mocked")
}
func (m *mockThornAdapter) CheckHashUpdate(ctx context.Context) (bool, error) {
	if m.checkHashUpdateFn != nil {
		return m.checkHashUpdateFn(ctx)
	}
	return false, nil
}
