package safety

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Test Helpers ────────────────────────────────────────────────────────────────

type testHarness struct {
	svc           SafetyService
	reportRepo    *mockReportRepo
	flagRepo      *mockFlagRepo
	actionRepo    *mockActionRepo
	accountRepo   *mockAccountStatusRepo
	appealRepo    *mockAppealRepo
	ncmecRepo     *mockNcmecRepo
	botSignalRepo *mockBotSignalRepo
	iamService    *mockIamService
	cache         *mockCache
	events        *mockEventBus
	jobs          *mockJobEnqueuer
}

func newTestHarness() *testHarness {
	h := &testHarness{
		reportRepo:    newMockReportRepo(),
		flagRepo:      newMockFlagRepo(),
		actionRepo:    newMockActionRepo(),
		accountRepo:   newMockAccountStatusRepo(),
		appealRepo:    newMockAppealRepo(),
		ncmecRepo:     newMockNcmecRepo(),
		botSignalRepo: newMockBotSignalRepo(),
		iamService:    newMockIamService(),
		cache:         newMockCache(),
		events:        newMockEventBus(),
		jobs:          newMockJobEnqueuer(),
	}

	// Sensible defaults for mocks used as internal dependencies.
	h.accountRepo.getOrCreateFn = func(_ context.Context, fid uuid.UUID) (*AccountStatusRow, error) {
		return &AccountStatusRow{FamilyID: fid, Status: "active"}, nil
	}
	h.accountRepo.updateFn = func(_ context.Context, fid uuid.UUID, u AccountStatusUpdate) (*AccountStatusRow, error) {
		s := "active"
		if u.Status != nil {
			s = *u.Status
		}
		return &AccountStatusRow{FamilyID: fid, Status: s}, nil
	}
	h.actionRepo.createFn = func(_ context.Context, input CreateModActionRow) (*ModAction, error) {
		return &ModAction{
			ID:             uuid.Must(uuid.NewV7()),
			AdminID:        input.AdminID,
			TargetFamilyID: input.TargetFamilyID,
			ActionType:     input.ActionType,
			Reason:         input.Reason,
			ReportID:       input.ReportID,
		}, nil
	}
	h.actionRepo.listByTargetFamilyFn = func(_ context.Context, _ uuid.UUID, _ shared.PaginationParams) ([]ModAction, error) {
		return nil, nil
	}
	h.flagRepo.createFn = func(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
		return &ContentFlag{ID: uuid.Must(uuid.NewV7()), Source: input.Source, FlagType: input.FlagType}, nil
	}
	h.ncmecRepo.createFn = func(_ context.Context, input CreateNcmecReportRow) (*NcmecReport, error) {
		return &NcmecReport{ID: uuid.Must(uuid.NewV7()), FamilyID: input.FamilyID, Status: "pending"}, nil
	}
	h.botSignalRepo.createFn = func(_ context.Context, _ CreateBotSignalRow) (*BotSignal, error) {
		return &BotSignal{ID: uuid.Must(uuid.NewV7())}, nil
	}
	h.botSignalRepo.countRecentFn = func(_ context.Context, _ uuid.UUID, _ uint32) (int64, error) {
		return 0, nil
	}
	h.reportRepo.updateFn = func(_ context.Context, id uuid.UUID, _ ReportUpdate) (*Report, error) {
		return &Report{ID: id, Status: "resolved_action_taken"}, nil
	}

	cfg := DefaultSafetyConfig()
	scanner := NewTextScanner(cfg)

	h.svc = NewSafetyService(
		h.reportRepo, h.flagRepo, h.actionRepo, h.accountRepo,
		h.appealRepo, h.ncmecRepo, h.botSignalRepo,
		h.iamService, h.cache, h.events, h.jobs, scanner, cfg,
		newMockParentalControlRepo(), newMockAdminRoleRepo(),
		newMockAdminRoleAssignRepo(), newMockGroomingScoreRepo(),
		newMockGroomingDetector(), nil,
	)

	return h
}

func testScope(familyID uuid.UUID) shared.FamilyScope {
	return shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
}

func testAuth(parentID, familyID uuid.UUID) *shared.AuthContext {
	return &shared.AuthContext{ParentID: parentID, FamilyID: familyID, IsPlatformAdmin: true}
}

func ptr[T any](v T) *T { return &v }

// ─── D1: SubmitReport ────────────────────────────────────────────────────────────

func TestSubmitReport_valid(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	parentID := uuid.Must(uuid.NewV7())
	targetID := uuid.Must(uuid.NewV7())

	h.reportRepo.existsRecentFn = func(_ context.Context, _ shared.FamilyScope, _ string, _ uuid.UUID, _ uint32) (bool, error) {
		return false, nil
	}
	h.reportRepo.createFn = func(_ context.Context, _ shared.FamilyScope, input CreateReportRow) (*Report, error) {
		return &Report{
			ID:               uuid.Must(uuid.NewV7()),
			ReporterFamilyID: input.ReporterFamilyID,
			Category:         input.Category,
			Status:           "pending",
			Priority:         input.Priority,
		}, nil
	}

	resp, err := h.svc.SubmitReport(context.Background(), testScope(familyID), testAuth(parentID, familyID), CreateReportCommand{
		TargetType: "post",
		TargetID:   targetID,
		Category:   "spam",
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "pending" {
		t.Errorf("status = %s, want pending", resp.Status)
	}
}

func TestSubmitReport_duplicate(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.reportRepo.existsRecentFn = func(_ context.Context, _ shared.FamilyScope, _ string, _ uuid.UUID, _ uint32) (bool, error) {
		return true, nil
	}

	_, err := h.svc.SubmitReport(context.Background(), testScope(familyID), testAuth(uuid.Must(uuid.NewV7()), familyID), CreateReportCommand{
		TargetType: "post",
		TargetID:   uuid.Must(uuid.NewV7()),
		Category:   "spam",
	})

	if !errors.Is(err, ErrDuplicateReport) {
		t.Errorf("err = %v, want ErrDuplicateReport", err)
	}
}

func TestSubmitReport_csam_priority_critical(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.reportRepo.existsRecentFn = func(_ context.Context, _ shared.FamilyScope, _ string, _ uuid.UUID, _ uint32) (bool, error) {
		return false, nil
	}
	var capturedPriority string
	h.reportRepo.createFn = func(_ context.Context, _ shared.FamilyScope, input CreateReportRow) (*Report, error) {
		capturedPriority = input.Priority
		return &Report{ID: uuid.Must(uuid.NewV7()), Status: "pending", Priority: input.Priority, Category: input.Category}, nil
	}

	_, err := h.svc.SubmitReport(context.Background(), testScope(familyID), testAuth(uuid.Must(uuid.NewV7()), familyID), CreateReportCommand{
		TargetType: "post",
		TargetID:   uuid.Must(uuid.NewV7()),
		Category:   "csam_child_safety",
	})

	if err != nil {
		t.Fatal(err)
	}
	if capturedPriority != "critical" {
		t.Errorf("priority = %s, want critical", capturedPriority)
	}
}

func TestSubmitReport_harassment_priority_high(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.reportRepo.existsRecentFn = func(_ context.Context, _ shared.FamilyScope, _ string, _ uuid.UUID, _ uint32) (bool, error) {
		return false, nil
	}
	var capturedPriority string
	h.reportRepo.createFn = func(_ context.Context, _ shared.FamilyScope, input CreateReportRow) (*Report, error) {
		capturedPriority = input.Priority
		return &Report{ID: uuid.Must(uuid.NewV7()), Status: "pending", Priority: input.Priority, Category: input.Category}, nil
	}

	_, err := h.svc.SubmitReport(context.Background(), testScope(familyID), testAuth(uuid.Must(uuid.NewV7()), familyID), CreateReportCommand{
		TargetType: "post",
		TargetID:   uuid.Must(uuid.NewV7()),
		Category:   "harassment",
	})

	if err != nil {
		t.Fatal(err)
	}
	if capturedPriority != "high" {
		t.Errorf("priority = %s, want high", capturedPriority)
	}
}

func TestSubmitReport_creates_content_flag(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.reportRepo.existsRecentFn = func(_ context.Context, _ shared.FamilyScope, _ string, _ uuid.UUID, _ uint32) (bool, error) {
		return false, nil
	}
	h.reportRepo.createFn = func(_ context.Context, _ shared.FamilyScope, input CreateReportRow) (*Report, error) {
		return &Report{ID: uuid.Must(uuid.NewV7()), Status: "pending", Priority: input.Priority}, nil
	}
	var capturedSource string
	h.flagRepo.createFn = func(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
		capturedSource = input.Source
		return &ContentFlag{ID: uuid.Must(uuid.NewV7())}, nil
	}

	_, err := h.svc.SubmitReport(context.Background(), testScope(familyID), testAuth(uuid.Must(uuid.NewV7()), familyID), CreateReportCommand{
		TargetType: "post",
		TargetID:   uuid.Must(uuid.NewV7()),
		Category:   "spam",
	})

	if err != nil {
		t.Fatal(err)
	}
	if capturedSource != "community_report" {
		t.Errorf("flag source = %s, want community_report", capturedSource)
	}
}

func TestSubmitReport_publishes_event(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.reportRepo.existsRecentFn = func(_ context.Context, _ shared.FamilyScope, _ string, _ uuid.UUID, _ uint32) (bool, error) {
		return false, nil
	}
	h.reportRepo.createFn = func(_ context.Context, _ shared.FamilyScope, input CreateReportRow) (*Report, error) {
		return &Report{ID: uuid.Must(uuid.NewV7()), Status: "pending", Priority: input.Priority, Category: input.Category}, nil
	}

	_, err := h.svc.SubmitReport(context.Background(), testScope(familyID), testAuth(uuid.Must(uuid.NewV7()), familyID), CreateReportCommand{
		TargetType: "post",
		TargetID:   uuid.Must(uuid.NewV7()),
		Category:   "spam",
	})

	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, evt := range h.events.published {
		if _, ok := evt.(ContentReported); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ContentReported event")
	}
}

// ─── D2: SubmitAppeal ────────────────────────────────────────────────────────────

func TestSubmitAppeal_valid(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	actionID := uuid.Must(uuid.NewV7())

	h.actionRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ModAction, error) {
		return &ModAction{ID: actionID, TargetFamilyID: familyID, ActionType: "warning_issued"}, nil
	}
	h.appealRepo.findByActionIDFn = func(_ context.Context, _ uuid.UUID) (*Appeal, error) {
		return nil, fmt.Errorf("not found")
	}
	h.appealRepo.createFn = func(_ context.Context, _ shared.FamilyScope, input CreateAppealRow) (*Appeal, error) {
		return &Appeal{
			ID:         uuid.Must(uuid.NewV7()),
			FamilyID:   familyID,
			ActionID:   input.ActionID,
			AppealText: input.AppealText,
			Status:     "pending",
		}, nil
	}

	resp, err := h.svc.SubmitAppeal(context.Background(), testScope(familyID), CreateAppealCommand{
		ActionID:   actionID,
		AppealText: "I believe this was a mistake",
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "pending" {
		t.Errorf("status = %s, want pending", resp.Status)
	}
}

func TestSubmitAppeal_action_not_found(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()

	h.actionRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ModAction, error) {
		return nil, fmt.Errorf("not found")
	}

	_, err := h.svc.SubmitAppeal(context.Background(), testScope(uuid.Must(uuid.NewV7())), CreateAppealCommand{
		ActionID:   uuid.Must(uuid.NewV7()),
		AppealText: "I believe this was a mistake",
	})

	if !errors.Is(err, ErrActionNotFound) {
		t.Errorf("err = %v, want ErrActionNotFound", err)
	}
}

func TestSubmitAppeal_wrong_family(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	callerFamilyID := uuid.Must(uuid.NewV7())
	otherFamilyID := uuid.Must(uuid.NewV7())

	h.actionRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ModAction, error) {
		return &ModAction{ID: uuid.Must(uuid.NewV7()), TargetFamilyID: otherFamilyID}, nil
	}

	_, err := h.svc.SubmitAppeal(context.Background(), testScope(callerFamilyID), CreateAppealCommand{
		ActionID:   uuid.Must(uuid.NewV7()),
		AppealText: "I believe this was a mistake",
	})

	if !errors.Is(err, ErrActionNotFound) {
		t.Errorf("err = %v, want ErrActionNotFound", err)
	}
}

func TestSubmitAppeal_csam_not_appealable(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.actionRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ModAction, error) {
		return &ModAction{ID: uuid.Must(uuid.NewV7()), TargetFamilyID: familyID, ActionType: "account_banned"}, nil
	}
	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		return &AccountStatusRow{FamilyID: familyID, Status: "banned", BanReason: ptr("csam_violation")}, nil
	}

	_, err := h.svc.SubmitAppeal(context.Background(), testScope(familyID), CreateAppealCommand{
		ActionID:   uuid.Must(uuid.NewV7()),
		AppealText: "I believe this was a mistake",
	})

	if !errors.Is(err, ErrCsamBanNotAppealable) {
		t.Errorf("err = %v, want ErrCsamBanNotAppealable", err)
	}
}

func TestSubmitAppeal_duplicate(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.actionRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ModAction, error) {
		return &ModAction{ID: uuid.Must(uuid.NewV7()), TargetFamilyID: familyID, ActionType: "warning_issued"}, nil
	}
	h.appealRepo.findByActionIDFn = func(_ context.Context, _ uuid.UUID) (*Appeal, error) {
		return &Appeal{ID: uuid.Must(uuid.NewV7())}, nil
	}

	_, err := h.svc.SubmitAppeal(context.Background(), testScope(familyID), CreateAppealCommand{
		ActionID:   uuid.Must(uuid.NewV7()),
		AppealText: "I believe this was a mistake",
	})

	if !errors.Is(err, ErrAppealAlreadyExists) {
		t.Errorf("err = %v, want ErrAppealAlreadyExists", err)
	}
}

// ─── E1: CheckAccountAccess ──────────────────────────────────────────────────────

func TestCheckAccountAccess_active_cached(t *testing.T) { // [11-safety §12.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	h.cache.data[accountCacheKey(familyID)] = "active"

	err := h.svc.CheckAccountAccess(context.Background(), familyID)

	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

func TestCheckAccountAccess_suspended_cached(t *testing.T) { // [11-safety §12.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	h.cache.data[accountCacheKey(familyID)] = "suspended"

	err := h.svc.CheckAccountAccess(context.Background(), familyID)

	if !errors.Is(err, ErrAccountSuspended) {
		t.Errorf("err = %v, want ErrAccountSuspended", err)
	}
}

func TestCheckAccountAccess_banned_cached(t *testing.T) { // [11-safety §12.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	h.cache.data[accountCacheKey(familyID)] = "banned"

	err := h.svc.CheckAccountAccess(context.Background(), familyID)

	if !errors.Is(err, ErrAccountBanned) {
		t.Errorf("err = %v, want ErrAccountBanned", err)
	}
}

func TestCheckAccountAccess_cache_miss_queries_db(t *testing.T) { // [11-safety §12.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	dbQueried := false
	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		dbQueried = true
		return &AccountStatusRow{FamilyID: familyID, Status: "active"}, nil
	}

	err := h.svc.CheckAccountAccess(context.Background(), familyID)

	if err != nil {
		t.Fatal(err)
	}
	if !dbQueried {
		t.Error("expected DB query on cache miss")
	}
	// Verify it was cached.
	cached := h.cache.data[accountCacheKey(familyID)]
	if cached != "active" {
		t.Errorf("cached = %s, want active", cached)
	}
}

func TestCheckAccountAccess_expired_suspension(t *testing.T) { // [11-safety §12.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	pastTime := time.Now().Add(-1 * time.Hour)

	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		return &AccountStatusRow{
			FamilyID:            familyID,
			Status:              "suspended",
			SuspensionExpiresAt: &pastTime,
		}, nil
	}

	var capturedStatus *string
	h.accountRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AccountStatusUpdate) (*AccountStatusRow, error) {
		capturedStatus = u.Status
		return &AccountStatusRow{FamilyID: familyID, Status: "active"}, nil
	}

	err := h.svc.CheckAccountAccess(context.Background(), familyID)

	if err != nil {
		t.Errorf("err = %v, want nil (expired suspension)", err)
	}
	if capturedStatus == nil || *capturedStatus != "active" {
		t.Error("expected account status transitioned to active")
	}
}

func TestCheckAccountAccess_db_error_defaults_active(t *testing.T) { // [11-safety §12.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		return nil, fmt.Errorf("db error")
	}

	err := h.svc.CheckAccountAccess(context.Background(), familyID)

	if err != nil {
		t.Errorf("err = %v, want nil (fail open)", err)
	}
}

// ─── E2: AdminSuspendAccount ─────────────────────────────────────────────────────

func TestAdminSuspendAccount_active(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())

	var capturedActionType string
	h.actionRepo.createFn = func(_ context.Context, input CreateModActionRow) (*ModAction, error) {
		capturedActionType = input.ActionType
		return &ModAction{ID: uuid.Must(uuid.NewV7()), AdminID: input.AdminID, TargetFamilyID: familyID, ActionType: input.ActionType}, nil
	}

	var capturedStatus *string
	h.accountRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AccountStatusUpdate) (*AccountStatusRow, error) {
		capturedStatus = u.Status
		return &AccountStatusRow{FamilyID: familyID, Status: "suspended"}, nil
	}

	resp, err := h.svc.AdminSuspendAccount(context.Background(), testAuth(adminID, uuid.Must(uuid.NewV7())), familyID, SuspendAccountCommand{
		Reason:         "Harassment violations",
		SuspensionDays: 7,
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if capturedActionType != "account_suspended" {
		t.Errorf("action type = %s, want account_suspended", capturedActionType)
	}
	if capturedStatus == nil || *capturedStatus != "suspended" {
		t.Error("expected account status updated to suspended")
	}
	// Verify cache was invalidated.
	if _, exists := h.cache.data[accountCacheKey(familyID)]; exists {
		t.Error("expected cache invalidated")
	}
}

func TestAdminSuspendAccount_banned(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		return &AccountStatusRow{FamilyID: familyID, Status: "banned", BanReason: ptr("policy_violation")}, nil
	}

	_, err := h.svc.AdminSuspendAccount(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID, SuspendAccountCommand{
		Reason:         "Harassment violations",
		SuspensionDays: 7,
	})

	if !errors.Is(err, ErrAccountBanned) {
		t.Errorf("err = %v, want ErrAccountBanned", err)
	}
}

func TestAdminSuspendAccount_publishes_event(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	_, err := h.svc.AdminSuspendAccount(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID, SuspendAccountCommand{
		Reason:         "Spam violations",
		SuspensionDays: 3,
	})

	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, evt := range h.events.published {
		if e, ok := evt.(AccountSuspended); ok {
			found = true
			if e.FamilyID != familyID {
				t.Errorf("event family_id = %v, want %v", e.FamilyID, familyID)
			}
			if e.SuspensionDays != 3 {
				t.Errorf("suspension_days = %d, want 3", e.SuspensionDays)
			}
			if e.ExpiresAt.IsZero() {
				t.Error("expected non-zero expires_at")
			}
		}
	}
	if !found {
		t.Error("expected AccountSuspended event")
	}
}

// ─── E3: AdminBanAccount ─────────────────────────────────────────────────────────

func TestAdminBanAccount_active(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	var capturedActionType string
	h.actionRepo.createFn = func(_ context.Context, input CreateModActionRow) (*ModAction, error) {
		capturedActionType = input.ActionType
		return &ModAction{ID: uuid.Must(uuid.NewV7()), AdminID: input.AdminID, TargetFamilyID: familyID, ActionType: input.ActionType}, nil
	}

	sessionsRevoked := false
	h.iamService.revokeSessionsFn = func(_ context.Context, _ uuid.UUID) error {
		sessionsRevoked = true
		return nil
	}

	resp, err := h.svc.AdminBanAccount(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID, BanAccountCommand{
		Reason: "Severe policy violation",
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if capturedActionType != "account_banned" {
		t.Errorf("action type = %s, want account_banned", capturedActionType)
	}
	if !sessionsRevoked {
		t.Error("expected sessions revoked")
	}
}

func TestAdminBanAccount_suspended_clears_suspension(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	suspendedAt := time.Now().Add(-24 * time.Hour)
	expiresAt := time.Now().Add(6 * 24 * time.Hour)

	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		return &AccountStatusRow{
			FamilyID:            familyID,
			Status:              "suspended",
			SuspendedAt:         &suspendedAt,
			SuspensionExpiresAt: &expiresAt,
			SuspensionReason:    ptr("prior violation"),
		}, nil
	}

	var capturedUpdate AccountStatusUpdate
	h.accountRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AccountStatusUpdate) (*AccountStatusRow, error) {
		capturedUpdate = u
		return &AccountStatusRow{FamilyID: familyID, Status: "banned"}, nil
	}

	_, err := h.svc.AdminBanAccount(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID, BanAccountCommand{
		Reason: "Escalated to ban",
	})

	if err != nil {
		t.Fatal(err)
	}
	if capturedUpdate.Status == nil || *capturedUpdate.Status != "banned" {
		t.Error("expected status updated to banned")
	}
}

func TestAdminBanAccount_already_banned(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		return &AccountStatusRow{FamilyID: familyID, Status: "banned", BanReason: ptr("prior ban")}, nil
	}

	_, err := h.svc.AdminBanAccount(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID, BanAccountCommand{
		Reason: "Another violation",
	})

	if !errors.Is(err, ErrAccountBanned) {
		t.Errorf("err = %v, want ErrAccountBanned", err)
	}
}

func TestAdminBanAccount_revokes_sessions(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	var revokedFamilyID uuid.UUID
	h.iamService.revokeSessionsFn = func(_ context.Context, fid uuid.UUID) error {
		revokedFamilyID = fid
		return nil
	}

	_, err := h.svc.AdminBanAccount(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID, BanAccountCommand{
		Reason: "Policy violation",
	})

	if err != nil {
		t.Fatal(err)
	}
	if revokedFamilyID != familyID {
		t.Errorf("revoked family = %v, want %v", revokedFamilyID, familyID)
	}
}

func TestAdminBanAccount_csam_no_notification(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	_, err := h.svc.AdminBanAccount(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID, BanAccountCommand{
		Reason: "csam_violation",
	})

	if err != nil {
		t.Fatal(err)
	}
	for _, evt := range h.events.published {
		if _, ok := evt.(AccountBanned); ok {
			t.Error("expected NO AccountBanned event for CSAM ban")
		}
	}
}

// ─── E4: AdminLiftSuspension ─────────────────────────────────────────────────────

func TestAdminLiftSuspension_suspended(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	suspendedAt := time.Now().Add(-24 * time.Hour)
	expiresAt := time.Now().Add(6 * 24 * time.Hour)

	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		return &AccountStatusRow{
			FamilyID:            familyID,
			Status:              "suspended",
			SuspendedAt:         &suspendedAt,
			SuspensionExpiresAt: &expiresAt,
			SuspensionReason:    ptr("policy violation"),
		}, nil
	}

	var capturedStatus *string
	h.accountRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AccountStatusUpdate) (*AccountStatusRow, error) {
		capturedStatus = u.Status
		return &AccountStatusRow{FamilyID: familyID, Status: "active"}, nil
	}

	resp, err := h.svc.AdminLiftSuspension(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID, LiftSuspensionCommand{
		Reason: "Appeal granted",
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if capturedStatus == nil || *capturedStatus != "active" {
		t.Error("expected status transitioned to active")
	}
}

func TestAdminLiftSuspension_active(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	// Default getOrCreate returns active, which is what we want.

	_, err := h.svc.AdminLiftSuspension(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID, LiftSuspensionCommand{
		Reason: "Appeal granted",
	})

	if !errors.Is(err, ErrInvalidActionType) {
		t.Errorf("err = %v, want ErrInvalidActionType", err)
	}
}

func TestAdminLiftSuspension_banned(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		return &AccountStatusRow{FamilyID: familyID, Status: "banned", BanReason: ptr("some reason")}, nil
	}

	_, err := h.svc.AdminLiftSuspension(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID, LiftSuspensionCommand{
		Reason: "Appeal granted",
	})

	if !errors.Is(err, ErrInvalidActionType) {
		t.Errorf("err = %v, want ErrInvalidActionType", err)
	}
}

// ─── F1: AdminUpdateReport ───────────────────────────────────────────────────────

func TestAdminUpdateReport_assign(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	reportID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())

	h.reportRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Report, error) {
		return &Report{ID: reportID, Status: "pending", Priority: "normal", Category: "spam"}, nil
	}
	h.reportRepo.updateFn = func(_ context.Context, _ uuid.UUID, u ReportUpdate) (*Report, error) {
		return &Report{
			ID:              reportID,
			Status:          *u.Status,
			AssignedAdminID: u.AssignedAdminID,
			Priority:        "normal",
		}, nil
	}

	resp, err := h.svc.AdminUpdateReport(context.Background(), testAuth(adminID, uuid.Must(uuid.NewV7())), reportID, UpdateReportCommand{
		AssignedAdminID: &adminID,
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "in_review" {
		t.Errorf("status = %s, want in_review", resp.Status)
	}
}

func TestAdminUpdateReport_resolve(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	reportID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())

	h.reportRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Report, error) {
		return &Report{
			ID:              reportID,
			Status:          "in_review",
			Priority:        "normal",
			Category:        "spam",
			AssignedAdminID: &adminID,
		}, nil
	}
	h.reportRepo.updateFn = func(_ context.Context, _ uuid.UUID, u ReportUpdate) (*Report, error) {
		return &Report{ID: reportID, Status: *u.Status, Priority: "normal"}, nil
	}

	status := "resolved_action_taken"
	resp, err := h.svc.AdminUpdateReport(context.Background(), testAuth(adminID, uuid.Must(uuid.NewV7())), reportID, UpdateReportCommand{
		Status: &status,
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "resolved_action_taken" {
		t.Errorf("status = %s, want resolved_action_taken", resp.Status)
	}
}

func TestAdminUpdateReport_invalid_transition(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	reportID := uuid.Must(uuid.NewV7())

	h.reportRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Report, error) {
		return &Report{ID: reportID, Status: "pending", Priority: "normal", Category: "spam"}, nil
	}

	status := "resolved_action_taken"
	_, err := h.svc.AdminUpdateReport(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), reportID, UpdateReportCommand{
		Status: &status,
	})

	if !errors.Is(err, ErrInvalidReportTransition) {
		t.Errorf("err = %v, want ErrInvalidReportTransition", err)
	}
}

func TestAdminUpdateReport_not_found(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()

	h.reportRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Report, error) {
		return nil, fmt.Errorf("not found")
	}

	_, err := h.svc.AdminUpdateReport(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), uuid.Must(uuid.NewV7()), UpdateReportCommand{})

	if !errors.Is(err, ErrReportNotFound) {
		t.Errorf("err = %v, want ErrReportNotFound", err)
	}
}

// ─── F2: AdminReviewFlag ─────────────────────────────────────────────────────────

func TestAdminReviewFlag_success(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	flagID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())

	h.flagRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ContentFlag, error) {
		return &ContentFlag{ID: flagID, Reviewed: false}, nil
	}
	h.flagRepo.markReviewedFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, actionTaken bool) (*ContentFlag, error) {
		return &ContentFlag{ID: flagID, Reviewed: true, ActionTaken: &actionTaken}, nil
	}

	resp, err := h.svc.AdminReviewFlag(context.Background(), testAuth(adminID, uuid.Must(uuid.NewV7())), flagID, ReviewFlagCommand{ActionTaken: true})

	if err != nil {
		t.Fatal(err)
	}
	if !resp.Reviewed {
		t.Error("expected reviewed=true")
	}
}

func TestAdminReviewFlag_already_reviewed(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	flagID := uuid.Must(uuid.NewV7())

	h.flagRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ContentFlag, error) {
		return &ContentFlag{ID: flagID, Reviewed: true}, nil
	}

	_, err := h.svc.AdminReviewFlag(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), flagID, ReviewFlagCommand{})

	if !errors.Is(err, ErrFlagAlreadyReviewed) {
		t.Errorf("err = %v, want ErrFlagAlreadyReviewed", err)
	}
}

func TestAdminReviewFlag_not_found(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()

	h.flagRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ContentFlag, error) {
		return nil, fmt.Errorf("not found")
	}

	_, err := h.svc.AdminReviewFlag(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), uuid.Must(uuid.NewV7()), ReviewFlagCommand{})

	if !errors.Is(err, ErrFlagNotFound) {
		t.Errorf("err = %v, want ErrFlagNotFound", err)
	}
}

// ─── F3: AdminTakeAction ─────────────────────────────────────────────────────────

func TestAdminTakeAction_content_removed(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())

	var capturedType string
	h.actionRepo.createFn = func(_ context.Context, input CreateModActionRow) (*ModAction, error) {
		capturedType = input.ActionType
		return &ModAction{ID: uuid.Must(uuid.NewV7()), AdminID: input.AdminID, ActionType: input.ActionType, TargetFamilyID: familyID}, nil
	}

	resp, err := h.svc.AdminTakeAction(context.Background(), testAuth(adminID, uuid.Must(uuid.NewV7())), CreateModActionCommand{
		TargetFamilyID: familyID,
		ActionType:     "content_removed",
		Reason:         "Violates community guidelines",
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp.ActionType != "content_removed" {
		t.Errorf("action_type = %s, want content_removed", resp.ActionType)
	}
	if capturedType != "content_removed" {
		t.Errorf("captured type = %s, want content_removed", capturedType)
	}
}

func TestAdminTakeAction_account_suspended_delegates(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())

	// Set up action to be returned by ListByTargetFamily (for findLastAction).
	h.actionRepo.listByTargetFamilyFn = func(_ context.Context, _ uuid.UUID, _ shared.PaginationParams) ([]ModAction, error) {
		return []ModAction{{
			ID:             uuid.Must(uuid.NewV7()),
			AdminID:        adminID,
			TargetFamilyID: familyID,
			ActionType:     "account_suspended",
			Reason:         "Spam violations",
		}}, nil
	}

	resp, err := h.svc.AdminTakeAction(context.Background(), testAuth(adminID, uuid.Must(uuid.NewV7())), CreateModActionCommand{
		TargetFamilyID: familyID,
		ActionType:     "account_suspended",
		Reason:         "Spam violations",
		SuspensionDays: ptr(int32(7)),
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
}

func TestAdminTakeAction_account_banned_delegates(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())

	h.actionRepo.listByTargetFamilyFn = func(_ context.Context, _ uuid.UUID, _ shared.PaginationParams) ([]ModAction, error) {
		return []ModAction{{
			ID:             uuid.Must(uuid.NewV7()),
			AdminID:        adminID,
			TargetFamilyID: familyID,
			ActionType:     "account_banned",
			Reason:         "Severe violation",
		}}, nil
	}

	resp, err := h.svc.AdminTakeAction(context.Background(), testAuth(adminID, uuid.Must(uuid.NewV7())), CreateModActionCommand{
		TargetFamilyID: familyID,
		ActionType:     "account_banned",
		Reason:         "Severe violation",
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
}

func TestAdminTakeAction_invalid_type(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()

	_, err := h.svc.AdminTakeAction(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), CreateModActionCommand{
		TargetFamilyID: uuid.Must(uuid.NewV7()),
		ActionType:     "destroy_everything",
		Reason:         "Because I can",
	})

	if !errors.Is(err, ErrInvalidActionType) {
		t.Errorf("err = %v, want ErrInvalidActionType", err)
	}
}

func TestAdminTakeAction_resolves_report(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	reportID := uuid.Must(uuid.NewV7())
	familyID := uuid.Must(uuid.NewV7())

	reportResolved := false
	h.reportRepo.updateFn = func(_ context.Context, id uuid.UUID, u ReportUpdate) (*Report, error) {
		if id == reportID && u.Status != nil && *u.Status == "resolved_action_taken" {
			reportResolved = true
		}
		return &Report{ID: id, Status: "resolved_action_taken"}, nil
	}

	_, err := h.svc.AdminTakeAction(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), CreateModActionCommand{
		TargetFamilyID: familyID,
		ActionType:     "content_removed",
		Reason:         "Violates guidelines",
		ReportID:       &reportID,
	})

	if err != nil {
		t.Fatal(err)
	}
	if !reportResolved {
		t.Error("expected report resolved")
	}
}

func TestAdminTakeAction_publishes_moderation_event(t *testing.T) { // [11-safety §4.4, §16.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())

	_, err := h.svc.AdminTakeAction(context.Background(), testAuth(adminID, uuid.Must(uuid.NewV7())), CreateModActionCommand{
		TargetFamilyID: familyID,
		ActionType:     "content_removed",
		Reason:         "Violates guidelines",
	})

	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, evt := range h.events.published {
		if e, ok := evt.(ModerationActionTaken); ok {
			found = true
			if e.ActionType != "content_removed" {
				t.Errorf("action_type = %s, want content_removed", e.ActionType)
			}
			if e.TargetFamilyID != familyID {
				t.Errorf("target_family_id = %v, want %v", e.TargetFamilyID, familyID)
			}
		}
	}
	if !found {
		t.Error("expected ModerationActionTaken event")
	}
}

// ─── F4: AdminResolveAppeal ──────────────────────────────────────────────────────

func TestAdminResolveAppeal_grant_lifts_suspension(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	appealID := uuid.Must(uuid.NewV7())
	familyID := uuid.Must(uuid.NewV7())
	originalAdminID := uuid.Must(uuid.NewV7())
	reviewerAdminID := uuid.Must(uuid.NewV7())
	actionID := uuid.Must(uuid.NewV7())
	suspendedAt := time.Now().Add(-24 * time.Hour)
	expiresAt := time.Now().Add(6 * 24 * time.Hour)

	h.appealRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Appeal, error) {
		return &Appeal{ID: appealID, FamilyID: familyID, ActionID: actionID, Status: "pending"}, nil
	}
	h.actionRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ModAction, error) {
		return &ModAction{ID: actionID, AdminID: originalAdminID, TargetFamilyID: familyID, ActionType: "account_suspended"}, nil
	}
	h.appealRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AppealUpdate) (*Appeal, error) {
		return &Appeal{ID: appealID, FamilyID: familyID, ActionID: actionID, Status: *u.Status}, nil
	}

	// For AdminLiftSuspension to work, account must be suspended.
	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		return &AccountStatusRow{
			FamilyID:            familyID,
			Status:              "suspended",
			SuspendedAt:         &suspendedAt,
			SuspensionExpiresAt: &expiresAt,
			SuspensionReason:    ptr("policy violation"),
		}, nil
	}

	resp, err := h.svc.AdminResolveAppeal(context.Background(), testAuth(reviewerAdminID, uuid.Must(uuid.NewV7())), appealID, ResolveAppealCommand{
		Status:         "granted",
		ResolutionText: "Appeal has merit",
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "granted" {
		t.Errorf("status = %s, want granted", resp.Status)
	}
}

func TestAdminResolveAppeal_grant_reverses_ban(t *testing.T) { // [11-safety §12.4]
	h := newTestHarness()
	appealID := uuid.Must(uuid.NewV7())
	familyID := uuid.Must(uuid.NewV7())
	originalAdminID := uuid.Must(uuid.NewV7())
	reviewerAdminID := uuid.Must(uuid.NewV7())
	actionID := uuid.Must(uuid.NewV7())
	bannedAt := time.Now().Add(-24 * time.Hour)

	h.appealRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Appeal, error) {
		return &Appeal{ID: appealID, FamilyID: familyID, ActionID: actionID, Status: "pending"}, nil
	}
	h.actionRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ModAction, error) {
		return &ModAction{ID: actionID, AdminID: originalAdminID, TargetFamilyID: familyID, ActionType: "account_banned"}, nil
	}
	h.appealRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AppealUpdate) (*Appeal, error) {
		return &Appeal{ID: appealID, FamilyID: familyID, ActionID: actionID, Status: *u.Status}, nil
	}

	// Account must be banned for Unban() to work.
	h.accountRepo.getOrCreateFn = func(_ context.Context, _ uuid.UUID) (*AccountStatusRow, error) {
		return &AccountStatusRow{
			FamilyID:  familyID,
			Status:    "banned",
			BannedAt:  &bannedAt,
			BanReason: ptr("policy_violation"),
		}, nil
	}

	var updatedStatus *string
	h.accountRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AccountStatusUpdate) (*AccountStatusRow, error) {
		updatedStatus = u.Status
		return &AccountStatusRow{FamilyID: familyID, Status: "active"}, nil
	}

	// Track appeal_granted action creation.
	var appealGrantedCreated bool
	h.actionRepo.createFn = func(_ context.Context, input CreateModActionRow) (*ModAction, error) {
		if input.ActionType == "appeal_granted" {
			appealGrantedCreated = true
		}
		return &ModAction{ID: uuid.Must(uuid.NewV7()), AdminID: input.AdminID, TargetFamilyID: familyID, ActionType: input.ActionType}, nil
	}

	resp, err := h.svc.AdminResolveAppeal(context.Background(), testAuth(reviewerAdminID, uuid.Must(uuid.NewV7())), appealID, ResolveAppealCommand{
		Status:         "granted",
		ResolutionText: "Ban was disproportionate",
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "granted" {
		t.Errorf("status = %s, want granted", resp.Status)
	}
	if updatedStatus == nil || *updatedStatus != "active" {
		t.Error("expected account status updated to active")
	}
	if !appealGrantedCreated {
		t.Error("expected appeal_granted mod action created")
	}
}

func TestAdminResolveAppeal_deny(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	appealID := uuid.Must(uuid.NewV7())
	familyID := uuid.Must(uuid.NewV7())
	originalAdminID := uuid.Must(uuid.NewV7())
	reviewerAdminID := uuid.Must(uuid.NewV7())
	actionID := uuid.Must(uuid.NewV7())

	h.appealRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Appeal, error) {
		return &Appeal{ID: appealID, FamilyID: familyID, ActionID: actionID, Status: "pending"}, nil
	}
	h.actionRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ModAction, error) {
		return &ModAction{ID: actionID, AdminID: originalAdminID, TargetFamilyID: familyID, ActionType: "account_suspended"}, nil
	}
	h.appealRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AppealUpdate) (*Appeal, error) {
		return &Appeal{ID: appealID, FamilyID: familyID, ActionID: actionID, Status: *u.Status}, nil
	}

	resp, err := h.svc.AdminResolveAppeal(context.Background(), testAuth(reviewerAdminID, uuid.Must(uuid.NewV7())), appealID, ResolveAppealCommand{
		Status:         "denied",
		ResolutionText: "Violation clearly documented",
	})

	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "denied" {
		t.Errorf("status = %s, want denied", resp.Status)
	}
}

func TestAdminResolveAppeal_same_admin(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	appealID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())
	actionID := uuid.Must(uuid.NewV7())

	h.appealRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Appeal, error) {
		return &Appeal{ID: appealID, ActionID: actionID, Status: "pending"}, nil
	}
	h.actionRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ModAction, error) {
		return &ModAction{ID: actionID, AdminID: adminID}, nil
	}

	_, err := h.svc.AdminResolveAppeal(context.Background(), testAuth(adminID, uuid.Must(uuid.NewV7())), appealID, ResolveAppealCommand{
		Status:         "granted",
		ResolutionText: "Overturning my own action",
	})

	if !errors.Is(err, ErrSameAdminAppeal) {
		t.Errorf("err = %v, want ErrSameAdminAppeal", err)
	}
}

func TestAdminResolveAppeal_not_found(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()

	h.appealRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Appeal, error) {
		return nil, fmt.Errorf("not found")
	}

	_, err := h.svc.AdminResolveAppeal(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), uuid.Must(uuid.NewV7()), ResolveAppealCommand{
		Status:         "granted",
		ResolutionText: "Does not matter",
	})

	if !errors.Is(err, ErrAppealNotFound) {
		t.Errorf("err = %v, want ErrAppealNotFound", err)
	}
}

func TestAdminResolveAppeal_publishes_event(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	appealID := uuid.Must(uuid.NewV7())
	familyID := uuid.Must(uuid.NewV7())
	originalAdminID := uuid.Must(uuid.NewV7())
	reviewerAdminID := uuid.Must(uuid.NewV7())
	actionID := uuid.Must(uuid.NewV7())

	h.appealRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Appeal, error) {
		return &Appeal{ID: appealID, FamilyID: familyID, ActionID: actionID, Status: "pending"}, nil
	}
	h.actionRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ModAction, error) {
		return &ModAction{ID: actionID, AdminID: originalAdminID, TargetFamilyID: familyID, ActionType: "warning_issued"}, nil
	}
	h.appealRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AppealUpdate) (*Appeal, error) {
		return &Appeal{ID: appealID, FamilyID: familyID, ActionID: actionID, Status: *u.Status}, nil
	}

	_, err := h.svc.AdminResolveAppeal(context.Background(), testAuth(reviewerAdminID, uuid.Must(uuid.NewV7())), appealID, ResolveAppealCommand{
		Status:         "denied",
		ResolutionText: "Decision stands",
	})

	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, evt := range h.events.published {
		if e, ok := evt.(AppealResolved); ok {
			found = true
			if e.AppealID != appealID {
				t.Errorf("event appeal_id = %v, want %v", e.AppealID, appealID)
			}
		}
	}
	if !found {
		t.Error("expected AppealResolved event")
	}
}

// ─── G1: HandleCsamDetection ─────────────────────────────────────────────────────

func TestHandleCsamDetection_creates_ncmec_report(t *testing.T) { // [11-safety §10]
	h := newTestHarness()
	uploadID := uuid.Must(uuid.NewV7())
	familyID := uuid.Must(uuid.NewV7())

	ncmecCreated := false
	h.ncmecRepo.createFn = func(_ context.Context, input CreateNcmecReportRow) (*NcmecReport, error) {
		ncmecCreated = true
		if input.FamilyID != familyID {
			t.Errorf("ncmec family = %v, want %v", input.FamilyID, familyID)
		}
		return &NcmecReport{ID: uuid.Must(uuid.NewV7()), FamilyID: familyID, Status: "pending"}, nil
	}

	err := h.svc.HandleCsamDetection(context.Background(), uploadID, familyID, &CsamScanResult{
		IsCSAM:     true,
		Hash:       ptr("abc123"),
		Confidence: ptr(99.5),
	})

	if err != nil {
		t.Fatal(err)
	}
	if !ncmecCreated {
		t.Error("expected NCMEC report created")
	}
}

func TestHandleCsamDetection_bans_account(t *testing.T) { // [11-safety §10]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	var capturedBanReason string
	h.accountRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AccountStatusUpdate) (*AccountStatusRow, error) {
		if u.BanReason != nil {
			capturedBanReason = *u.BanReason
		}
		return &AccountStatusRow{FamilyID: familyID, Status: "banned"}, nil
	}

	err := h.svc.HandleCsamDetection(context.Background(), uuid.Must(uuid.NewV7()), familyID, &CsamScanResult{IsCSAM: true})

	if err != nil {
		t.Fatal(err)
	}
	if capturedBanReason != "csam_violation" {
		t.Errorf("ban reason = %s, want csam_violation", capturedBanReason)
	}
}

func TestHandleCsamDetection_revokes_sessions(t *testing.T) { // [11-safety §10]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	sessionsRevoked := false
	h.iamService.revokeSessionsFn = func(_ context.Context, fid uuid.UUID) error {
		if fid == familyID {
			sessionsRevoked = true
		}
		return nil
	}

	err := h.svc.HandleCsamDetection(context.Background(), uuid.Must(uuid.NewV7()), familyID, &CsamScanResult{IsCSAM: true})

	if err != nil {
		t.Fatal(err)
	}
	if !sessionsRevoked {
		t.Error("expected sessions revoked")
	}
}

func TestHandleCsamDetection_no_notification(t *testing.T) { // [11-safety §10]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	err := h.svc.HandleCsamDetection(context.Background(), uuid.Must(uuid.NewV7()), familyID, &CsamScanResult{IsCSAM: true})

	if err != nil {
		t.Fatal(err)
	}
	for _, evt := range h.events.published {
		if _, ok := evt.(AccountBanned); ok {
			t.Error("expected NO AccountBanned event for CSAM detection")
		}
	}
}

func TestHandleCsamDetection_enqueues_job(t *testing.T) { // [11-safety §10]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	err := h.svc.HandleCsamDetection(context.Background(), uuid.Must(uuid.NewV7()), familyID, &CsamScanResult{IsCSAM: true})

	if err != nil {
		t.Fatal(err)
	}
	if len(h.jobs.enqueued) == 0 {
		t.Error("expected CSAM report job enqueued")
	}
	if h.jobs.enqueued[0].TaskType() != "safety:csam_report" {
		t.Errorf("job type = %s, want safety:csam_report", h.jobs.enqueued[0].TaskType())
	}
}

// ─── G2: AdminEscalateToCsam ─────────────────────────────────────────────────────

func TestAdminEscalateToCsam_success(t *testing.T) { // [11-safety §11.4.1]
	h := newTestHarness()
	flagID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())
	targetFamilyID := uuid.Must(uuid.NewV7())
	targetID := uuid.Must(uuid.NewV7())

	h.flagRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ContentFlag, error) {
		return &ContentFlag{ID: flagID, TargetID: targetID, TargetFamilyID: &targetFamilyID, Reviewed: false}, nil
	}

	flagReviewed := false
	h.flagRepo.markReviewedFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ bool) (*ContentFlag, error) {
		flagReviewed = true
		return &ContentFlag{ID: flagID, Reviewed: true}, nil
	}

	var escalationActionType string
	originalCreate := h.actionRepo.createFn
	callCount := 0
	h.actionRepo.createFn = func(ctx context.Context, input CreateModActionRow) (*ModAction, error) {
		callCount++
		if callCount == 1 {
			escalationActionType = input.ActionType
		}
		return originalCreate(ctx, input)
	}

	err := h.svc.AdminEscalateToCsam(context.Background(), testAuth(adminID, uuid.Must(uuid.NewV7())), flagID, EscalateCsamCommand{
		AdminNotes: "Confirmed CSAM content",
	})

	if err != nil {
		t.Fatal(err)
	}
	if !flagReviewed {
		t.Error("expected flag marked reviewed")
	}
	if escalationActionType != "escalate_to_csam" {
		t.Errorf("action type = %s, want escalate_to_csam", escalationActionType)
	}
}

func TestAdminEscalateToCsam_already_reviewed(t *testing.T) { // [11-safety §11.4.1]
	h := newTestHarness()
	flagID := uuid.Must(uuid.NewV7())

	h.flagRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ContentFlag, error) {
		return &ContentFlag{ID: flagID, Reviewed: true}, nil
	}

	err := h.svc.AdminEscalateToCsam(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), flagID, EscalateCsamCommand{
		AdminNotes: "Should fail",
	})

	if !errors.Is(err, ErrFlagAlreadyReviewed) {
		t.Errorf("err = %v, want ErrFlagAlreadyReviewed", err)
	}
}

func TestAdminEscalateToCsam_nil_hash_fields(t *testing.T) { // [11-safety §11.4.1]
	h := newTestHarness()
	flagID := uuid.Must(uuid.NewV7())
	targetFamilyID := uuid.Must(uuid.NewV7())

	h.flagRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*ContentFlag, error) {
		return &ContentFlag{ID: flagID, TargetID: uuid.Must(uuid.NewV7()), TargetFamilyID: &targetFamilyID, Reviewed: false}, nil
	}
	h.flagRepo.markReviewedFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ bool) (*ContentFlag, error) {
		return &ContentFlag{ID: flagID, Reviewed: true}, nil
	}

	var capturedResult *CsamScanResult
	originalNcmecCreate := h.ncmecRepo.createFn
	h.ncmecRepo.createFn = func(ctx context.Context, input CreateNcmecReportRow) (*NcmecReport, error) {
		capturedResult = &CsamScanResult{
			Hash:       input.CsamHash,
			Confidence: input.Confidence,
		}
		return originalNcmecCreate(ctx, input)
	}

	err := h.svc.AdminEscalateToCsam(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), flagID, EscalateCsamCommand{
		AdminNotes: "Human-identified CSAM",
	})

	if err != nil {
		t.Fatal(err)
	}
	if capturedResult != nil && capturedResult.Hash != nil {
		t.Error("expected nil hash for human-identified CSAM")
	}
}

// ─── H2: RecordBotSignal ─────────────────────────────────────────────────────────

func TestRecordBotSignal_records_signal(t *testing.T) { // [11-safety §13.2]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	parentID := uuid.Must(uuid.NewV7())

	signalRecorded := false
	h.botSignalRepo.createFn = func(_ context.Context, input CreateBotSignalRow) (*BotSignal, error) {
		signalRecorded = true
		if input.SignalType != string(BotSignalRapidPosting) {
			t.Errorf("signal type = %s, want rapid_posting", input.SignalType)
		}
		return &BotSignal{ID: uuid.Must(uuid.NewV7())}, nil
	}

	err := h.svc.RecordBotSignal(context.Background(), familyID, parentID, BotSignalRapidPosting, nil)

	if err != nil {
		t.Fatal(err)
	}
	if !signalRecorded {
		t.Error("expected bot signal recorded")
	}
}

func TestRecordBotSignal_below_threshold(t *testing.T) { // [11-safety §13.2]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	parentID := uuid.Must(uuid.NewV7())

	h.botSignalRepo.countRecentFn = func(_ context.Context, _ uuid.UUID, _ uint32) (int64, error) {
		return 2, nil // Below default threshold of 5
	}

	// If AdminSuspendAccount were called, accountRepo.update would be called.
	// Since we're below threshold, it should NOT be called for suspension.
	updateCalls := 0
	h.accountRepo.updateFn = func(_ context.Context, _ uuid.UUID, _ AccountStatusUpdate) (*AccountStatusRow, error) {
		updateCalls++
		return &AccountStatusRow{Status: "active"}, nil
	}

	err := h.svc.RecordBotSignal(context.Background(), familyID, parentID, BotSignalRapidPosting, nil)

	if err != nil {
		t.Fatal(err)
	}
	if updateCalls > 0 {
		t.Error("expected no suspension below threshold")
	}
}

func TestRecordBotSignal_above_threshold_suspends(t *testing.T) { // [11-safety §13.2]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	parentID := uuid.Must(uuid.NewV7())

	h.botSignalRepo.countRecentFn = func(_ context.Context, _ uuid.UUID, _ uint32) (int64, error) {
		return 5, nil // At threshold
	}

	suspended := false
	h.accountRepo.updateFn = func(_ context.Context, _ uuid.UUID, u AccountStatusUpdate) (*AccountStatusRow, error) {
		if u.Status != nil && *u.Status == "suspended" {
			suspended = true
		}
		return &AccountStatusRow{FamilyID: familyID, Status: "suspended"}, nil
	}

	err := h.svc.RecordBotSignal(context.Background(), familyID, parentID, BotSignalRapidPosting, nil)

	if err != nil {
		t.Fatal(err)
	}
	if !suspended {
		t.Error("expected auto-suspension at threshold")
	}
}

// ─── J: Query Methods ────────────────────────────────────────────────────────────

func TestListMyReports(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.reportRepo.listByReporterFn = func(_ context.Context, _ shared.FamilyScope, _ shared.PaginationParams) ([]Report, error) {
		return []Report{
			{ID: uuid.Must(uuid.NewV7()), Status: "pending", Category: "spam"},
			{ID: uuid.Must(uuid.NewV7()), Status: "in_review", Category: "harassment"},
		}, nil
	}

	resp, err := h.svc.ListMyReports(context.Background(), testScope(familyID), shared.PaginationParams{})

	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Data))
	}
}

func TestGetMyReport_found(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	reportID := uuid.Must(uuid.NewV7())

	h.reportRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, _ uuid.UUID) (*Report, error) {
		return &Report{ID: reportID, Status: "pending", Category: "spam"}, nil
	}

	resp, err := h.svc.GetMyReport(context.Background(), testScope(familyID), reportID)

	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != reportID {
		t.Errorf("id = %v, want %v", resp.ID, reportID)
	}
}

func TestGetMyReport_not_found(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()

	h.reportRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, _ uuid.UUID) (*Report, error) {
		return nil, fmt.Errorf("not found")
	}

	_, err := h.svc.GetMyReport(context.Background(), testScope(uuid.Must(uuid.NewV7())), uuid.Must(uuid.NewV7()))

	if !errors.Is(err, ErrReportNotFound) {
		t.Errorf("err = %v, want ErrReportNotFound", err)
	}
}

func TestGetAccountStatus(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	resp, err := h.svc.GetAccountStatus(context.Background(), testScope(familyID))

	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "active" {
		t.Errorf("status = %s, want active", resp.Status)
	}
}

func TestGetMyAppeal_found(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())
	appealID := uuid.Must(uuid.NewV7())

	h.appealRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, _ uuid.UUID) (*Appeal, error) {
		return &Appeal{ID: appealID, FamilyID: familyID, Status: "pending", AppealText: "Please review"}, nil
	}

	resp, err := h.svc.GetMyAppeal(context.Background(), testScope(familyID), appealID)

	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != appealID {
		t.Errorf("id = %v, want %v", resp.ID, appealID)
	}
}

func TestGetMyAppeal_not_found(t *testing.T) { // [11-safety §4.3]
	h := newTestHarness()

	h.appealRepo.findByIDFn = func(_ context.Context, _ shared.FamilyScope, _ uuid.UUID) (*Appeal, error) {
		return nil, fmt.Errorf("not found")
	}

	_, err := h.svc.GetMyAppeal(context.Background(), testScope(uuid.Must(uuid.NewV7())), uuid.Must(uuid.NewV7()))

	if !errors.Is(err, ErrAppealNotFound) {
		t.Errorf("err = %v, want ErrAppealNotFound", err)
	}
}

func TestAdminListReports(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()

	h.reportRepo.listFilteredFn = func(_ context.Context, _ ReportFilter, _ shared.PaginationParams) ([]Report, error) {
		return []Report{{ID: uuid.Must(uuid.NewV7()), Status: "pending", Priority: "critical"}}, nil
	}

	resp, err := h.svc.AdminListReports(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), ReportFilter{}, shared.PaginationParams{})

	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("len = %d, want 1", len(resp.Data))
	}
}

func TestAdminGetReport_found(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	reportID := uuid.Must(uuid.NewV7())

	h.reportRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Report, error) {
		return &Report{ID: reportID, Status: "pending", Priority: "normal", Category: "spam"}, nil
	}

	resp, err := h.svc.AdminGetReport(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), reportID)

	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != reportID {
		t.Errorf("id = %v, want %v", resp.ID, reportID)
	}
}

func TestAdminGetReport_not_found(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()

	h.reportRepo.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Report, error) {
		return nil, fmt.Errorf("not found")
	}

	_, err := h.svc.AdminGetReport(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), uuid.Must(uuid.NewV7()))

	if !errors.Is(err, ErrReportNotFound) {
		t.Errorf("err = %v, want ErrReportNotFound", err)
	}
}

func TestAdminGetAccount(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()
	familyID := uuid.Must(uuid.NewV7())

	h.actionRepo.listByTargetFamilyFn = func(_ context.Context, _ uuid.UUID, _ shared.PaginationParams) ([]ModAction, error) {
		return []ModAction{
			{ID: uuid.Must(uuid.NewV7()), ActionType: "warning_issued", Reason: "First offense"},
		}, nil
	}

	resp, err := h.svc.AdminGetAccount(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), familyID)

	if err != nil {
		t.Fatal(err)
	}
	if resp.FamilyID != familyID {
		t.Errorf("family_id = %v, want %v", resp.FamilyID, familyID)
	}
	if len(resp.ActionHistory) != 1 {
		t.Errorf("action history len = %d, want 1", len(resp.ActionHistory))
	}
}

func TestAdminDashboard(t *testing.T) { // [11-safety §4.4]
	h := newTestHarness()

	h.reportRepo.countByStatusFn = func(_ context.Context, _ string) (int64, error) {
		return 5, nil
	}
	h.reportRepo.countByStatusAndPriorityFn = func(_ context.Context, _ string, _ string) (int64, error) {
		return 2, nil
	}
	h.flagRepo.countUnreviewedFn = func(_ context.Context) (int64, error) {
		return 10, nil
	}

	resp, err := h.svc.AdminDashboard(context.Background(), testAuth(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())))

	if err != nil {
		t.Fatal(err)
	}
	if resp.PendingReports != 5 {
		t.Errorf("pending = %d, want 5", resp.PendingReports)
	}
	if resp.CriticalReports != 2 {
		t.Errorf("critical = %d, want 2", resp.CriticalReports)
	}
	if resp.UnreviewedFlags != 10 {
		t.Errorf("unreviewed = %d, want 10", resp.UnreviewedFlags)
	}
}
