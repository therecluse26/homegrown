package safety

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/safety/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// safetyServiceImpl implements SafetyService. [11-safety §5.1]
type safetyServiceImpl struct {
	reportRepo    ReportRepository
	flagRepo      ContentFlagRepository
	actionRepo    ModActionRepository
	accountRepo   AccountStatusRepository
	appealRepo    AppealRepository
	ncmecRepo     NcmecReportRepository
	botSignalRepo BotSignalRepository
	iamService    IamServiceForSafety
	cache         shared.Cache
	events        eventPublisher
	jobs          shared.JobEnqueuer
	textScanner   *TextScanner
	config        SafetyConfig
}

// NewSafetyService creates a new SafetyService.
func NewSafetyService(
	reportRepo ReportRepository,
	flagRepo ContentFlagRepository,
	actionRepo ModActionRepository,
	accountRepo AccountStatusRepository,
	appealRepo AppealRepository,
	ncmecRepo NcmecReportRepository,
	botSignalRepo BotSignalRepository,
	iamService IamServiceForSafety,
	cache shared.Cache,
	events eventPublisher,
	jobs shared.JobEnqueuer,
	textScanner *TextScanner,
	config SafetyConfig,
) SafetyService {
	return &safetyServiceImpl{
		reportRepo:    reportRepo,
		flagRepo:      flagRepo,
		actionRepo:    actionRepo,
		accountRepo:   accountRepo,
		appealRepo:    appealRepo,
		ncmecRepo:     ncmecRepo,
		botSignalRepo: botSignalRepo,
		iamService:    iamService,
		cache:         cache,
		events:        events,
		jobs:          jobs,
		textScanner:   textScanner,
		config:        config,
	}
}

// ─── User-Facing Commands ───────────────────────────────────────────────────────

// SubmitReport creates a content report. [11-safety §4.3]
func (s *safetyServiceImpl) SubmitReport(ctx context.Context, scope shared.FamilyScope, auth *shared.AuthContext, cmd CreateReportCommand) (*ReportResponse, error) {
	// Check duplicate.
	exists, err := s.reportRepo.ExistsRecent(ctx, scope, cmd.TargetType, cmd.TargetID, 24)
	if err != nil {
		return nil, fmt.Errorf("check duplicate: %w", err)
	}
	if exists {
		return nil, &SafetyError{Err: ErrDuplicateReport}
	}

	priority := string(domain.DerivePriority(cmd.Category))

	report, err := s.reportRepo.Create(ctx, scope, CreateReportRow{
		ReporterFamilyID: auth.FamilyID,
		ReporterParentID: auth.ParentID,
		TargetType:       cmd.TargetType,
		TargetID:         cmd.TargetID,
		Category:         cmd.Category,
		Description:      cmd.Description,
		Priority:         priority,
	})
	if err != nil {
		return nil, fmt.Errorf("create report: %w", err)
	}

	// Create content flag with source=community_report.
	if _, err := s.flagRepo.Create(ctx, CreateContentFlagRow{
		Source:     "community_report",
		TargetType: cmd.TargetType,
		TargetID:   cmd.TargetID,
		FlagType:   flagTypeFromCategory(cmd.Category),
		ReportID:   &report.ID,
	}); err != nil {
		slog.Error("failed to create content flag for report", "report_id", report.ID, "error", err)
	}

	// Publish event.
	_ = s.events.Publish(ctx, ContentReported{
		ReportID:   report.ID,
		FamilyID:   auth.FamilyID,
		TargetType: cmd.TargetType,
		TargetID:   cmd.TargetID,
		Category:   cmd.Category,
		Priority:   priority,
	})

	return reportToResponse(report), nil
}

// SubmitAppeal submits an appeal against a moderation action. [11-safety §4.3]
func (s *safetyServiceImpl) SubmitAppeal(ctx context.Context, scope shared.FamilyScope, cmd CreateAppealCommand) (*AppealResponse, error) {
	// Find the action.
	action, err := s.actionRepo.FindByID(ctx, cmd.ActionID)
	if err != nil {
		return nil, &SafetyError{Err: ErrActionNotFound}
	}

	// Verify the action targets the caller's family.
	if action.TargetFamilyID != scope.FamilyID() {
		return nil, &SafetyError{Err: ErrActionNotFound}
	}

	// CSAM bans are not appealable.
	if action.ActionType == "account_banned" {
		acct, err := s.accountRepo.GetOrCreate(ctx, action.TargetFamilyID)
		if err == nil && acct.BanReason != nil && *acct.BanReason == "csam_violation" {
			return nil, &SafetyError{Err: ErrCsamBanNotAppealable}
		}
	}

	// Check for existing appeal.
	existing, err := s.appealRepo.FindByActionID(ctx, cmd.ActionID)
	if err == nil && existing != nil {
		return nil, &SafetyError{Err: ErrAppealAlreadyExists}
	}

	appeal, err := s.appealRepo.Create(ctx, scope, CreateAppealRow(cmd))
	if err != nil {
		return nil, fmt.Errorf("create appeal: %w", err)
	}

	return appealToResponse(appeal), nil
}

// ─── User-Facing Queries ────────────────────────────────────────────────────────

func (s *safetyServiceImpl) ListMyReports(ctx context.Context, scope shared.FamilyScope, pagination shared.PaginationParams) (*shared.PaginatedResponse[ReportResponse], error) {
	reports, err := s.reportRepo.ListByReporter(ctx, scope, pagination)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}

	data := make([]ReportResponse, len(reports))
	for i, r := range reports {
		data[i] = *reportToResponse(&r)
	}

	return &shared.PaginatedResponse[ReportResponse]{Data: data}, nil
}

func (s *safetyServiceImpl) GetMyReport(ctx context.Context, scope shared.FamilyScope, reportID uuid.UUID) (*ReportResponse, error) {
	report, err := s.reportRepo.FindByID(ctx, scope, reportID)
	if err != nil {
		return nil, &SafetyError{Err: ErrReportNotFound}
	}
	return reportToResponse(report), nil
}

func (s *safetyServiceImpl) GetAccountStatus(ctx context.Context, scope shared.FamilyScope) (*AccountStatusResponse, error) {
	acct, err := s.accountRepo.GetOrCreate(ctx, scope.FamilyID())
	if err != nil {
		return nil, fmt.Errorf("get account status: %w", err)
	}
	return &AccountStatusResponse{
		Status:              acct.Status,
		SuspendedAt:         acct.SuspendedAt,
		SuspensionExpiresAt: acct.SuspensionExpiresAt,
		SuspensionReason:    acct.SuspensionReason,
	}, nil
}

func (s *safetyServiceImpl) GetMyAppeal(ctx context.Context, scope shared.FamilyScope, appealID uuid.UUID) (*AppealResponse, error) {
	appeal, err := s.appealRepo.FindByID(ctx, scope, appealID)
	if err != nil {
		return nil, &SafetyError{Err: ErrAppealNotFound}
	}
	return appealToResponse(appeal), nil
}

// ─── Admin Queries ──────────────────────────────────────────────────────────────

func (s *safetyServiceImpl) AdminListReports(ctx context.Context, _ *shared.AuthContext, filter ReportFilter, pagination shared.PaginationParams) (*shared.PaginatedResponse[AdminReportResponse], error) {
	reports, err := s.reportRepo.ListFiltered(ctx, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}

	data := make([]AdminReportResponse, len(reports))
	for i, r := range reports {
		data[i] = *adminReportToResponse(&r)
	}

	return &shared.PaginatedResponse[AdminReportResponse]{Data: data}, nil
}

func (s *safetyServiceImpl) AdminGetReport(ctx context.Context, _ *shared.AuthContext, reportID uuid.UUID) (*AdminReportResponse, error) {
	report, err := s.reportRepo.FindByIDUnscoped(ctx, reportID)
	if err != nil {
		return nil, &SafetyError{Err: ErrReportNotFound}
	}
	return adminReportToResponse(report), nil
}

func (s *safetyServiceImpl) AdminListFlags(ctx context.Context, _ *shared.AuthContext, filter FlagFilter, pagination shared.PaginationParams) (*shared.PaginatedResponse[ContentFlagResponse], error) {
	flags, err := s.flagRepo.ListFiltered(ctx, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("list flags: %w", err)
	}

	data := make([]ContentFlagResponse, len(flags))
	for i, f := range flags {
		data[i] = *flagToResponse(&f)
	}

	return &shared.PaginatedResponse[ContentFlagResponse]{Data: data}, nil
}

func (s *safetyServiceImpl) AdminListActions(ctx context.Context, _ *shared.AuthContext, filter ActionFilter, pagination shared.PaginationParams) (*shared.PaginatedResponse[ModActionResponse], error) {
	actions, err := s.actionRepo.ListFiltered(ctx, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("list actions: %w", err)
	}

	data := make([]ModActionResponse, len(actions))
	for i, a := range actions {
		data[i] = *actionToResponse(&a)
	}

	return &shared.PaginatedResponse[ModActionResponse]{Data: data}, nil
}

func (s *safetyServiceImpl) AdminGetAccount(ctx context.Context, _ *shared.AuthContext, familyID uuid.UUID) (*AdminAccountStatusResponse, error) {
	acct, err := s.accountRepo.GetOrCreate(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	actions, err := s.actionRepo.ListByTargetFamily(ctx, familyID, shared.PaginationParams{})
	if err != nil {
		return nil, fmt.Errorf("list actions: %w", err)
	}

	history := make([]ModActionResponse, len(actions))
	for i, a := range actions {
		history[i] = *actionToResponse(&a)
	}

	return &AdminAccountStatusResponse{
		FamilyID:            acct.FamilyID,
		Status:              acct.Status,
		SuspendedAt:         acct.SuspendedAt,
		SuspensionExpiresAt: acct.SuspensionExpiresAt,
		SuspensionReason:    acct.SuspensionReason,
		BannedAt:            acct.BannedAt,
		BanReason:           acct.BanReason,
		ActionHistory:       history,
	}, nil
}

func (s *safetyServiceImpl) AdminListAppeals(ctx context.Context, _ *shared.AuthContext, filter AppealFilter, pagination shared.PaginationParams) (*shared.PaginatedResponse[AdminAppealResponse], error) {
	appeals, err := s.appealRepo.ListFiltered(ctx, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("list appeals: %w", err)
	}

	data := make([]AdminAppealResponse, len(appeals))
	for i, a := range appeals {
		action, _ := s.actionRepo.FindByID(ctx, a.ActionID)
		data[i] = *adminAppealToResponse(&a, action)
	}

	return &shared.PaginatedResponse[AdminAppealResponse]{Data: data}, nil
}

func (s *safetyServiceImpl) AdminDashboard(ctx context.Context, _ *shared.AuthContext) (*DashboardStats, error) {
	pending, _ := s.reportRepo.CountByStatus(ctx, "pending")
	critical, _ := s.reportRepo.CountByStatusAndPriority(ctx, "pending", "critical")
	unreviewed, _ := s.flagRepo.CountUnreviewed(ctx)
	pendingAppeals, _ := s.appealRepo.CountByStatus(ctx, "pending")
	suspensions, _ := s.accountRepo.CountByStatus(ctx, "suspended")
	bans, _ := s.accountRepo.CountByStatus(ctx, "banned")
	reports24h, _ := s.reportRepo.CountSince(ctx, "24h")
	actions24h, _ := s.actionRepo.CountSince(ctx, "24h")

	return &DashboardStats{
		PendingReports:    pending,
		CriticalReports:   critical,
		UnreviewedFlags:   unreviewed,
		PendingAppeals:    pendingAppeals,
		ActiveSuspensions: suspensions,
		ActiveBans:        bans,
		ReportsLast24h:    reports24h,
		ActionsLast24h:    actions24h,
	}, nil
}

// ─── Admin Commands ─────────────────────────────────────────────────────────────

// AdminUpdateReport updates a report status or assignment. [11-safety §4.4]
func (s *safetyServiceImpl) AdminUpdateReport(ctx context.Context, auth *shared.AuthContext, reportID uuid.UUID, cmd UpdateReportCommand) (*AdminReportResponse, error) {
	report, err := s.reportRepo.FindByIDUnscoped(ctx, reportID)
	if err != nil {
		return nil, &SafetyError{Err: ErrReportNotFound}
	}

	// Use domain aggregate for state transitions.
	agg := domain.ReportFromPersistence(
		report.ID, report.ReporterFamilyID, report.ReporterParentID,
		report.TargetType, report.TargetID, report.TargetFamilyID,
		report.Category, report.Description,
		domain.ReportPriority(report.Priority), domain.ReportStatus(report.Status),
		report.AssignedAdminID, report.ResolvedAt,
		report.CreatedAt, report.UpdatedAt,
	)

	var updates ReportUpdate

	if cmd.AssignedAdminID != nil {
		if err := agg.Assign(*cmd.AssignedAdminID); err != nil {
			return nil, &SafetyError{Err: err}
		}
		status := string(agg.Status())
		updates.Status = &status
		updates.AssignedAdminID = cmd.AssignedAdminID
	}

	if cmd.Status != nil {
		switch *cmd.Status {
		case "resolved_action_taken":
			if err := agg.ResolveActionTaken(); err != nil {
				return nil, &SafetyError{Err: err}
			}
		case "resolved_no_action":
			if err := agg.ResolveNoAction(); err != nil {
				return nil, &SafetyError{Err: err}
			}
		case "dismissed":
			if err := agg.Dismiss(); err != nil {
				return nil, &SafetyError{Err: err}
			}
		case "in_review":
			if err := agg.Assign(auth.ParentID); err != nil {
				return nil, &SafetyError{Err: err}
			}
			updates.AssignedAdminID = &auth.ParentID
		default:
			return nil, &SafetyError{Err: ErrInvalidReportTransition}
		}
		status := string(agg.Status())
		updates.Status = &status
		if agg.ResolvedAt() != nil {
			updates.ResolvedAt = agg.ResolvedAt()
		}
	}

	updated, err := s.reportRepo.Update(ctx, reportID, updates)
	if err != nil {
		return nil, fmt.Errorf("update report: %w", err)
	}

	return adminReportToResponse(updated), nil
}

// AdminReviewFlag marks a flag as reviewed. [11-safety §4.4]
func (s *safetyServiceImpl) AdminReviewFlag(ctx context.Context, auth *shared.AuthContext, flagID uuid.UUID, cmd ReviewFlagCommand) (*ContentFlagResponse, error) {
	flag, err := s.flagRepo.FindByID(ctx, flagID)
	if err != nil {
		return nil, &SafetyError{Err: ErrFlagNotFound}
	}

	if flag.Reviewed {
		return nil, &SafetyError{Err: ErrFlagAlreadyReviewed}
	}

	updated, err := s.flagRepo.MarkReviewed(ctx, flagID, auth.ParentID, cmd.ActionTaken)
	if err != nil {
		return nil, fmt.Errorf("mark reviewed: %w", err)
	}

	return flagToResponse(updated), nil
}

// AdminTakeAction creates a moderation action. [11-safety §4.4]
func (s *safetyServiceImpl) AdminTakeAction(ctx context.Context, auth *shared.AuthContext, cmd CreateModActionCommand) (*ModActionResponse, error) {
	switch cmd.ActionType {
	case "content_removed", "warning_issued", "content_restored":
		// Simple actions — just create the record.
	case "account_suspended":
		days := int32(7)
		if cmd.SuspensionDays != nil {
			days = *cmd.SuspensionDays
		}
		if _, err := s.AdminSuspendAccount(ctx, auth, cmd.TargetFamilyID, SuspendAccountCommand{
			Reason:         cmd.Reason,
			SuspensionDays: days,
			ReportID:       cmd.ReportID,
		}); err != nil {
			return nil, err
		}
		// The suspend action creates its own record, so return.
		return s.findLastAction(ctx, cmd.TargetFamilyID)
	case "account_banned":
		if _, err := s.AdminBanAccount(ctx, auth, cmd.TargetFamilyID, BanAccountCommand{
			Reason:   cmd.Reason,
			ReportID: cmd.ReportID,
		}); err != nil {
			return nil, err
		}
		return s.findLastAction(ctx, cmd.TargetFamilyID)
	default:
		return nil, &SafetyError{Err: ErrInvalidActionType}
	}

	action, err := s.actionRepo.Create(ctx, CreateModActionRow{
		AdminID:        auth.ParentID,
		TargetFamilyID: cmd.TargetFamilyID,
		TargetParentID: cmd.TargetParentID,
		ActionType:     cmd.ActionType,
		Reason:         cmd.Reason,
		ReportID:       cmd.ReportID,
	})
	if err != nil {
		return nil, fmt.Errorf("create action: %w", err)
	}

	// Resolve associated report if provided.
	if cmd.ReportID != nil {
		now := time.Now().UTC()
		status := "resolved_action_taken"
		if _, err := s.reportRepo.Update(ctx, *cmd.ReportID, ReportUpdate{
			Status:     &status,
			ResolvedAt: &now,
		}); err != nil {
			slog.Error("failed to resolve report", "report_id", cmd.ReportID, "error", err)
		}
	}

	// Publish ModerationActionTaken event. [11-safety §4.4, §16.3]
	_ = s.events.Publish(ctx, ModerationActionTaken{
		ActionID:       action.ID,
		ActionType:     action.ActionType,
		TargetFamilyID: action.TargetFamilyID,
	})

	return actionToResponse(action), nil
}

// AdminSuspendAccount suspends an account. [11-safety §4.4]
func (s *safetyServiceImpl) AdminSuspendAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd SuspendAccountCommand) (*AdminAccountStatusResponse, error) {
	acct, err := s.accountRepo.GetOrCreate(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	// Use domain aggregate for validation.
	state := domain.AccountStateFromPersistence(
		acct.FamilyID, domain.AccountStatus(acct.Status),
		acct.SuspendedAt, acct.SuspensionExpiresAt, acct.SuspensionReason,
		acct.BannedAt, acct.BanReason, acct.LastActionID,
		acct.CreatedAt, acct.UpdatedAt,
	)

	evt, err := state.Suspend(auth.ParentID, cmd.Reason, cmd.SuspensionDays)
	if err != nil {
		return nil, &SafetyError{Err: err}
	}

	// Create mod action.
	now := time.Now().UTC()
	expiresAt := now.AddDate(0, 0, int(cmd.SuspensionDays))
	action, err := s.actionRepo.Create(ctx, CreateModActionRow{
		AdminID:             auth.ParentID,
		TargetFamilyID:      familyID,
		ActionType:          "account_suspended",
		Reason:              cmd.Reason,
		ReportID:            cmd.ReportID,
		SuspensionDays:      &cmd.SuspensionDays,
		SuspensionExpiresAt: &expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create action: %w", err)
	}

	// Update account status.
	status := "suspended"
	if _, err := s.accountRepo.Update(ctx, familyID, AccountStatusUpdate{
		Status:              &status,
		SuspendedAt:         &now,
		SuspensionExpiresAt: &expiresAt,
		SuspensionReason:    &cmd.Reason,
		LastActionID:        &action.ID,
	}); err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}

	// Invalidate cache.
	_ = s.cache.Delete(ctx, accountCacheKey(familyID))

	// Publish event. [11-safety §16.3]
	_ = s.events.Publish(ctx, AccountSuspended{
		FamilyID:       evt.FamilyID,
		SuspensionDays: cmd.SuspensionDays,
		ExpiresAt:      evt.ExpiresAt,
	})

	return s.AdminGetAccount(ctx, auth, familyID)
}

// AdminBanAccount bans an account. [11-safety §4.4]
func (s *safetyServiceImpl) AdminBanAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd BanAccountCommand) (*AdminAccountStatusResponse, error) {
	acct, err := s.accountRepo.GetOrCreate(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	state := domain.AccountStateFromPersistence(
		acct.FamilyID, domain.AccountStatus(acct.Status),
		acct.SuspendedAt, acct.SuspensionExpiresAt, acct.SuspensionReason,
		acct.BannedAt, acct.BanReason, acct.LastActionID,
		acct.CreatedAt, acct.UpdatedAt,
	)

	if err := state.Ban(cmd.Reason); err != nil {
		return nil, &SafetyError{Err: err}
	}

	// Create mod action.
	now := time.Now().UTC()
	action, err := s.actionRepo.Create(ctx, CreateModActionRow{
		AdminID:        auth.ParentID,
		TargetFamilyID: familyID,
		ActionType:     "account_banned",
		Reason:         cmd.Reason,
		ReportID:       cmd.ReportID,
	})
	if err != nil {
		return nil, fmt.Errorf("create action: %w", err)
	}

	// Update account status.
	status := "banned"
	if _, err := s.accountRepo.Update(ctx, familyID, AccountStatusUpdate{
		Status:              &status,
		BannedAt:            &now,
		BanReason:           &cmd.Reason,
		SuspendedAt:         nil,
		SuspensionExpiresAt: nil,
		SuspensionReason:    nil,
		LastActionID:        &action.ID,
	}); err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}

	// Invalidate cache.
	_ = s.cache.Delete(ctx, accountCacheKey(familyID))

	// Revoke all sessions.
	if err := s.iamService.RevokeSessions(ctx, familyID); err != nil {
		slog.Error("failed to revoke sessions", "family_id", familyID, "error", err)
	}

	// Publish event only for non-CSAM bans.
	if cmd.Reason != "csam_violation" {
		_ = s.events.Publish(ctx, AccountBanned{
			FamilyID: familyID,
			AdminID:  auth.ParentID,
			Reason:   cmd.Reason,
		})
	}

	return s.AdminGetAccount(ctx, auth, familyID)
}

// AdminLiftSuspension lifts a suspension. [11-safety §4.4]
func (s *safetyServiceImpl) AdminLiftSuspension(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd LiftSuspensionCommand) (*AdminAccountStatusResponse, error) {
	acct, err := s.accountRepo.GetOrCreate(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	state := domain.AccountStateFromPersistence(
		acct.FamilyID, domain.AccountStatus(acct.Status),
		acct.SuspendedAt, acct.SuspensionExpiresAt, acct.SuspensionReason,
		acct.BannedAt, acct.BanReason, acct.LastActionID,
		acct.CreatedAt, acct.UpdatedAt,
	)

	if err := state.LiftSuspension(); err != nil {
		return nil, &SafetyError{Err: err}
	}

	// Create mod action.
	if _, err := s.actionRepo.Create(ctx, CreateModActionRow{
		AdminID:        auth.ParentID,
		TargetFamilyID: familyID,
		ActionType:     "suspension_lifted",
		Reason:         cmd.Reason,
	}); err != nil {
		return nil, fmt.Errorf("create action: %w", err)
	}

	// Update account status.
	status := "active"
	if _, err := s.accountRepo.Update(ctx, familyID, AccountStatusUpdate{
		Status:              &status,
		SuspendedAt:         nil,
		SuspensionExpiresAt: nil,
		SuspensionReason:    nil,
	}); err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}

	// Invalidate cache.
	_ = s.cache.Delete(ctx, accountCacheKey(familyID))

	return s.AdminGetAccount(ctx, auth, familyID)
}

// AdminResolveAppeal resolves an appeal. [11-safety §4.4]
func (s *safetyServiceImpl) AdminResolveAppeal(ctx context.Context, auth *shared.AuthContext, appealID uuid.UUID, cmd ResolveAppealCommand) (*AdminAppealResponse, error) {
	appeal, err := s.appealRepo.FindByIDUnscoped(ctx, appealID)
	if err != nil {
		return nil, &SafetyError{Err: ErrAppealNotFound}
	}

	// Assigned admin must differ from original action admin.
	action, err := s.actionRepo.FindByID(ctx, appeal.ActionID)
	if err != nil {
		return nil, fmt.Errorf("find action: %w", err)
	}
	if action.AdminID == auth.ParentID {
		return nil, &SafetyError{Err: ErrSameAdminAppeal}
	}

	now := time.Now().UTC()
	updated, err := s.appealRepo.Update(ctx, appealID, AppealUpdate{
		Status:          &cmd.Status,
		AssignedAdminID: &auth.ParentID,
		ResolutionText:  &cmd.ResolutionText,
		ResolvedAt:      &now,
	})
	if err != nil {
		return nil, fmt.Errorf("update appeal: %w", err)
	}

	// If granted, reverse the original action.
	if cmd.Status == "granted" {
		switch action.ActionType {
		case "account_suspended":
			if _, err := s.AdminLiftSuspension(ctx, auth, action.TargetFamilyID, LiftSuspensionCommand{
				Reason: "Appeal granted: " + cmd.ResolutionText,
			}); err != nil {
				slog.Error("failed to lift suspension on appeal", "appeal_id", appealID, "error", err)
			}
		case "account_banned":
			// Reverse ban via domain aggregate. [11-safety §12.4]
			acct, acctErr := s.accountRepo.GetOrCreate(ctx, action.TargetFamilyID)
			if acctErr != nil {
				slog.Error("failed to get account for ban reversal", "appeal_id", appealID, "error", acctErr)
			} else {
				state := domain.AccountStateFromPersistence(
					acct.FamilyID, domain.AccountStatus(acct.Status),
					acct.SuspendedAt, acct.SuspensionExpiresAt, acct.SuspensionReason,
					acct.BannedAt, acct.BanReason, acct.LastActionID,
					acct.CreatedAt, acct.UpdatedAt,
				)
				if unbanErr := state.Unban(); unbanErr != nil {
					slog.Error("failed to unban on appeal", "appeal_id", appealID, "error", unbanErr)
				} else {
					// Create appeal_granted mod action.
					if _, actErr := s.actionRepo.Create(ctx, CreateModActionRow{
						AdminID:        auth.ParentID,
						TargetFamilyID: action.TargetFamilyID,
						ActionType:     "appeal_granted",
						Reason:         "Appeal granted: " + cmd.ResolutionText,
					}); actErr != nil {
						slog.Error("failed to create appeal_granted action", "appeal_id", appealID, "error", actErr)
					}
					// Update account status to active.
					activeStatus := "active"
					if _, updErr := s.accountRepo.Update(ctx, action.TargetFamilyID, AccountStatusUpdate{
						Status:    &activeStatus,
						BannedAt:  nil,
						BanReason: nil,
					}); updErr != nil {
						slog.Error("failed to update account after unban", "appeal_id", appealID, "error", updErr)
					}
					// Invalidate cache.
					_ = s.cache.Delete(ctx, accountCacheKey(action.TargetFamilyID))
				}
			}
		}
	}

	// Publish event.
	_ = s.events.Publish(ctx, AppealResolved{
		AppealID: appealID,
		FamilyID: appeal.FamilyID,
		Status:   cmd.Status,
	})

	return adminAppealToResponse(updated, action), nil
}

// ─── Internal Methods ───────────────────────────────────────────────────────────

// CheckAccountAccess checks whether a family's account is active. [11-safety §12.3]
func (s *safetyServiceImpl) CheckAccountAccess(ctx context.Context, familyID uuid.UUID) error {
	// Check cache first.
	cached, err := s.cache.Get(ctx, accountCacheKey(familyID))
	if err == nil && cached != "" {
		return statusToError(cached)
	}

	// Cache miss — query DB.
	acct, err := s.accountRepo.GetOrCreate(ctx, familyID)
	if err != nil {
		// Default to active on DB error (fail open).
		return nil
	}

	// Check lazy expiry.
	if acct.Status == "suspended" && acct.SuspensionExpiresAt != nil && time.Now().After(*acct.SuspensionExpiresAt) {
		status := "active"
		if _, err := s.accountRepo.Update(ctx, familyID, AccountStatusUpdate{
			Status:              &status,
			SuspendedAt:         nil,
			SuspensionExpiresAt: nil,
			SuspensionReason:    nil,
		}); err != nil {
			slog.Error("failed to expire suspension", "family_id", familyID, "error", err)
		}
		_ = s.cache.Delete(ctx, accountCacheKey(familyID))
		return nil
	}

	// Cache result.
	ttl := time.Duration(s.config.AccountStatusCacheTTLSeconds) * time.Second
	_ = s.cache.Set(ctx, accountCacheKey(familyID), acct.Status, ttl)

	return statusToError(acct.Status)
}

// ScanText delegates to the TextScanner. [11-safety §11.1]
func (s *safetyServiceImpl) ScanText(ctx context.Context, text string) (*TextScanResult, error) {
	return s.textScanner.Scan(ctx, text)
}

// RecordBotSignal records a bot signal and auto-suspends if threshold is reached. [11-safety §13.2]
func (s *safetyServiceImpl) RecordBotSignal(ctx context.Context, familyID uuid.UUID, parentID uuid.UUID, signal BotSignalType, details json.RawMessage) error {
	if _, err := s.botSignalRepo.Create(ctx, CreateBotSignalRow{
		FamilyID:   familyID,
		ParentID:   parentID,
		SignalType:  string(signal),
		Details:     details,
	}); err != nil {
		return fmt.Errorf("create bot signal: %w", err)
	}

	count, err := s.botSignalRepo.CountRecent(ctx, parentID, s.config.BotSignalWindowMinutes)
	if err != nil {
		return fmt.Errorf("count recent signals: %w", err)
	}

	if count >= s.config.BotSignalThreshold {
		systemAuth := &shared.AuthContext{
			ParentID:        uuid.Nil,
			FamilyID:        uuid.Nil,
			IsPlatformAdmin: true,
		}
		if _, err := s.AdminSuspendAccount(ctx, systemAuth, familyID, SuspendAccountCommand{
			Reason:         "Automated suspension: bot-like behavior detected",
			SuspensionDays: 1,
		}); err != nil {
			return fmt.Errorf("auto-suspend: %w", err)
		}
	}

	return nil
}

// HandleCsamDetection processes CSAM detection. [11-safety §10]
func (s *safetyServiceImpl) HandleCsamDetection(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID, scanResult *CsamScanResult) error {
	// 1. Create NCMEC report record.
	evidenceKey := fmt.Sprintf("evidence/csam/%s/%s", uuid.Must(uuid.NewV7()), uploadID)
	ncmecReport, err := s.ncmecRepo.Create(ctx, CreateNcmecReportRow{
		UploadID:           uploadID,
		FamilyID:           familyID,
		ParentID:           uuid.Nil, // Will be looked up by job
		CsamHash:           scanResult.Hash,
		Confidence:         scanResult.Confidence,
		MatchedDatabase:    scanResult.MatchedDatabase,
		EvidenceStorageKey: evidenceKey,
	})
	if err != nil {
		return fmt.Errorf("create ncmec report: %w", err)
	}

	// 2. Enqueue NCMEC report job.
	if err := s.jobs.Enqueue(ctx, &CsamReportPayload{NcmecReportID: ncmecReport.ID}); err != nil {
		slog.Error("failed to enqueue CSAM report job", "ncmec_report_id", ncmecReport.ID, "error", err)
	}

	// 3. Ban account immediately.
	systemAuth := &shared.AuthContext{
		ParentID:        uuid.Nil,
		FamilyID:        uuid.Nil,
		IsPlatformAdmin: true,
	}
	if _, err := s.AdminBanAccount(ctx, systemAuth, familyID, BanAccountCommand{
		Reason: "csam_violation",
	}); err != nil {
		slog.Error("failed to ban account for CSAM", "family_id", familyID, "error", err)
	}

	// 4. Invalidate cache (already done by AdminBanAccount).
	// 5. Revoke sessions (already done by AdminBanAccount).
	// 6. Do NOT publish notification event — zero user notification.

	return nil
}

// AdminEscalateToCsam escalates flagged content to CSAM. [11-safety §11.4.1]
func (s *safetyServiceImpl) AdminEscalateToCsam(ctx context.Context, auth *shared.AuthContext, flagID uuid.UUID, cmd EscalateCsamCommand) error {
	flag, err := s.flagRepo.FindByID(ctx, flagID)
	if err != nil {
		return &SafetyError{Err: ErrFlagNotFound}
	}

	if flag.Reviewed {
		return &SafetyError{Err: ErrFlagAlreadyReviewed}
	}

	// Mark flag as reviewed.
	if _, err := s.flagRepo.MarkReviewed(ctx, flagID, auth.ParentID, true); err != nil {
		return fmt.Errorf("mark reviewed: %w", err)
	}

	// Create escalate_to_csam mod action.
	targetFamilyID := uuid.Nil
	if flag.TargetFamilyID != nil {
		targetFamilyID = *flag.TargetFamilyID
	}
	if _, err := s.actionRepo.Create(ctx, CreateModActionRow{
		AdminID:        auth.ParentID,
		TargetFamilyID: targetFamilyID,
		ActionType:     "escalate_to_csam",
		Reason:         cmd.AdminNotes,
	}); err != nil {
		return fmt.Errorf("create escalation action: %w", err)
	}

	// Delegate to HandleCsamDetection with nil hash fields (human-identified).
	return s.HandleCsamDetection(ctx, flag.TargetID, targetFamilyID, &CsamScanResult{
		IsCSAM: true,
	})
}

// ─── Helpers ────────────────────────────────────────────────────────────────────

func accountCacheKey(familyID uuid.UUID) string {
	return "safety:account:" + familyID.String()
}

func statusToError(status string) error {
	switch status {
	case "suspended":
		return &SafetyError{Err: ErrAccountSuspended}
	case "banned":
		return &SafetyError{Err: ErrAccountBanned}
	default:
		return nil
	}
}

func flagTypeFromCategory(category string) string {
	switch category {
	case "csam_child_safety":
		return "csam"
	case "harassment":
		return "harassment"
	case "spam":
		return "spam"
	default:
		return "prohibited_content"
	}
}

func (s *safetyServiceImpl) findLastAction(ctx context.Context, familyID uuid.UUID) (*ModActionResponse, error) {
	actions, err := s.actionRepo.ListByTargetFamily(ctx, familyID, shared.PaginationParams{})
	if err != nil || len(actions) == 0 {
		return nil, fmt.Errorf("find action: %w", err)
	}
	return actionToResponse(&actions[0]), nil
}

// ─── Mappers ────────────────────────────────────────────────────────────────────

func reportToResponse(r *Report) *ReportResponse {
	return &ReportResponse{
		ID:         r.ID,
		TargetType: r.TargetType,
		Category:   r.Category,
		Status:     r.Status,
		CreatedAt:  r.CreatedAt,
	}
}

func adminReportToResponse(r *Report) *AdminReportResponse {
	return &AdminReportResponse{
		ID:               r.ID,
		ReporterFamilyID: r.ReporterFamilyID,
		TargetType:       r.TargetType,
		TargetID:         r.TargetID,
		TargetFamilyID:   r.TargetFamilyID,
		Category:         r.Category,
		Description:      r.Description,
		Priority:         r.Priority,
		Status:           r.Status,
		AssignedAdminID:  r.AssignedAdminID,
		ResolvedAt:       r.ResolvedAt,
		CreatedAt:        r.CreatedAt,
	}
}

func flagToResponse(f *ContentFlag) *ContentFlagResponse {
	return &ContentFlagResponse{
		ID:          f.ID,
		Source:      f.Source,
		TargetType:  f.TargetType,
		TargetID:    f.TargetID,
		FlagType:    f.FlagType,
		Confidence:  f.Confidence,
		Labels:      f.Labels,
		Reviewed:    f.Reviewed,
		ReviewedBy:  f.ReviewedBy,
		ActionTaken: f.ActionTaken,
		CreatedAt:   f.CreatedAt,
	}
}

func actionToResponse(a *ModAction) *ModActionResponse {
	return &ModActionResponse{
		ID:                  a.ID,
		AdminID:             a.AdminID,
		TargetFamilyID:      a.TargetFamilyID,
		TargetParentID:      a.TargetParentID,
		ActionType:          a.ActionType,
		Reason:              a.Reason,
		ReportID:            a.ReportID,
		SuspensionDays:      a.SuspensionDays,
		SuspensionExpiresAt: a.SuspensionExpiresAt,
		CreatedAt:           a.CreatedAt,
	}
}

func appealToResponse(a *Appeal) *AppealResponse {
	return &AppealResponse{
		ID:             a.ID,
		ActionID:       a.ActionID,
		Status:         a.Status,
		AppealText:     a.AppealText,
		ResolutionText: a.ResolutionText,
		ResolvedAt:     a.ResolvedAt,
		CreatedAt:      a.CreatedAt,
	}
}

func adminAppealToResponse(a *Appeal, action *ModAction) *AdminAppealResponse {
	resp := &AdminAppealResponse{
		ID:              a.ID,
		FamilyID:        a.FamilyID,
		ActionID:        a.ActionID,
		AppealText:      a.AppealText,
		Status:          a.Status,
		AssignedAdminID: a.AssignedAdminID,
		ResolutionText:  a.ResolutionText,
		ResolvedAt:      a.ResolvedAt,
		CreatedAt:       a.CreatedAt,
	}
	if action != nil {
		resp.OriginalAction = *actionToResponse(action)
	}
	return resp
}

// CsamReportPayload is the job payload for NCMEC report submission. [11-safety §10.3]
type CsamReportPayload struct {
	NcmecReportID uuid.UUID `json:"ncmec_report_id"`
}

func (CsamReportPayload) TaskType() string { return "safety:csam_report" }

// CheckCsamHashUpdatePayload is the job payload for checking CSAM hash updates. [11-safety §10.7]
type CheckCsamHashUpdatePayload struct{}

func (CheckCsamHashUpdatePayload) TaskType() string { return "safety:check_csam_hash_update" }
