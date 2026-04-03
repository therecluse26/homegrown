package safety

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Service Interface ──────────────────────────────────────────────────────────

// SafetyService defines all safety and moderation use cases. [11-safety §5]
type SafetyService interface {
	// ─── User-Facing Queries ────────────────────────────────────────────
	ListMyReports(ctx context.Context, scope shared.FamilyScope, pagination shared.PaginationParams) (*shared.PaginatedResponse[ReportResponse], error)
	GetMyReport(ctx context.Context, scope shared.FamilyScope, reportID uuid.UUID) (*ReportResponse, error)
	GetAccountStatus(ctx context.Context, scope shared.FamilyScope) (*AccountStatusResponse, error)
	GetMyAppeal(ctx context.Context, scope shared.FamilyScope, appealID uuid.UUID) (*AppealResponse, error)

	// ─── User-Facing Commands ───────────────────────────────────────────
	SubmitReport(ctx context.Context, scope shared.FamilyScope, auth *shared.AuthContext, cmd CreateReportCommand) (*ReportResponse, error)
	SubmitAppeal(ctx context.Context, scope shared.FamilyScope, cmd CreateAppealCommand) (*AppealResponse, error)

	// ─── Admin Queries ──────────────────────────────────────────────────
	AdminListReports(ctx context.Context, auth *shared.AuthContext, filter ReportFilter, pagination shared.PaginationParams) (*shared.PaginatedResponse[AdminReportResponse], error)
	AdminGetReport(ctx context.Context, auth *shared.AuthContext, reportID uuid.UUID) (*AdminReportResponse, error)
	AdminListFlags(ctx context.Context, auth *shared.AuthContext, filter FlagFilter, pagination shared.PaginationParams) (*shared.PaginatedResponse[ContentFlagResponse], error)
	AdminListActions(ctx context.Context, auth *shared.AuthContext, filter ActionFilter, pagination shared.PaginationParams) (*shared.PaginatedResponse[ModActionResponse], error)
	AdminGetAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID) (*AdminAccountStatusResponse, error)
	AdminListAppeals(ctx context.Context, auth *shared.AuthContext, filter AppealFilter, pagination shared.PaginationParams) (*shared.PaginatedResponse[AdminAppealResponse], error)
	AdminDashboard(ctx context.Context, auth *shared.AuthContext) (*DashboardStats, error)

	// ─── Admin Commands ─────────────────────────────────────────────────
	AdminUpdateReport(ctx context.Context, auth *shared.AuthContext, reportID uuid.UUID, cmd UpdateReportCommand) (*AdminReportResponse, error)
	AdminReviewFlag(ctx context.Context, auth *shared.AuthContext, flagID uuid.UUID, cmd ReviewFlagCommand) (*ContentFlagResponse, error)
	AdminTakeAction(ctx context.Context, auth *shared.AuthContext, cmd CreateModActionCommand) (*ModActionResponse, error)
	AdminSuspendAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd SuspendAccountCommand) (*AdminAccountStatusResponse, error)
	AdminBanAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd BanAccountCommand) (*AdminAccountStatusResponse, error)
	AdminLiftSuspension(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd LiftSuspensionCommand) (*AdminAccountStatusResponse, error)
	AdminResolveAppeal(ctx context.Context, auth *shared.AuthContext, appealID uuid.UUID, cmd ResolveAppealCommand) (*AdminAppealResponse, error)

	// ─── Internal (Cross-Domain) Methods ────────────────────────────────
	CheckAccountAccess(ctx context.Context, familyID uuid.UUID) error
	ScanText(ctx context.Context, text string) (*TextScanResult, error)
	RecordBotSignal(ctx context.Context, familyID uuid.UUID, parentID uuid.UUID, signal BotSignalType, details json.RawMessage) error
	HandleCsamDetection(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID, scanResult *CsamScanResult) error
	AdminEscalateToCsam(ctx context.Context, auth *shared.AuthContext, flagID uuid.UUID, cmd EscalateCsamCommand) error

	// ─── Phase 2: Expire Suspensions ────────────────────────────────
	ExpireSuspensions(ctx context.Context) error

	// ─── Phase 2: Parental Controls ─────────────────────────────────
	GetParentalControls(ctx context.Context, scope shared.FamilyScope) ([]ParentalControlResponse, error)
	UpsertParentalControl(ctx context.Context, scope shared.FamilyScope, cmd UpsertParentalControlCommand) (*ParentalControlResponse, error)
	DeleteParentalControl(ctx context.Context, scope shared.FamilyScope, controlID uuid.UUID) error

	// ─── Phase 2: Admin Roles ───────────────────────────────────────
	ListAdminRoles(ctx context.Context, auth *shared.AuthContext) ([]AdminRoleResponse, error)
	CreateAdminRole(ctx context.Context, auth *shared.AuthContext, cmd CreateAdminRoleCommand) (*AdminRoleResponse, error)
	AssignAdminRole(ctx context.Context, auth *shared.AuthContext, roleID uuid.UUID, cmd AssignAdminRoleCommand) (*AdminRoleAssignmentResponse, error)
	RevokeAdminRole(ctx context.Context, auth *shared.AuthContext, roleID uuid.UUID, parentID uuid.UUID) error
	ListAdminRoleAssignments(ctx context.Context, auth *shared.AuthContext, roleID uuid.UUID) ([]AdminRoleAssignmentResponse, error)
	GetParentPermissions(ctx context.Context, parentID uuid.UUID) ([]string, error)

	// ─── Phase 2: Grooming Detection ────────────────────────────────
	AnalyzeTextForGrooming(ctx context.Context, contentType string, contentID uuid.UUID, authorFamilyID uuid.UUID, text string) (*GroomingAnalysisResult, error)
	AdminListGroomingScores(ctx context.Context, auth *shared.AuthContext, pagination shared.PaginationParams) (*shared.PaginatedResponse[GroomingScoreResponse], error)
	AdminReviewGroomingScore(ctx context.Context, auth *shared.AuthContext, scoreID uuid.UUID, cmd ReviewGroomingScoreCommand) (*GroomingScoreResponse, error)
}

// ─── Repository Interfaces ──────────────────────────────────────────────────────

// ReportRepository defines persistence operations for safety_reports. [11-safety §6.1]
type ReportRepository interface {
	Create(ctx context.Context, scope shared.FamilyScope, input CreateReportRow) (*Report, error)
	FindByID(ctx context.Context, scope shared.FamilyScope, reportID uuid.UUID) (*Report, error)
	FindByIDUnscoped(ctx context.Context, reportID uuid.UUID) (*Report, error)
	ListByReporter(ctx context.Context, scope shared.FamilyScope, pagination shared.PaginationParams) ([]Report, error)
	ListFiltered(ctx context.Context, filter ReportFilter, pagination shared.PaginationParams) ([]Report, error)
	Update(ctx context.Context, reportID uuid.UUID, updates ReportUpdate) (*Report, error)
	ExistsRecent(ctx context.Context, scope shared.FamilyScope, targetType string, targetID uuid.UUID, withinHours uint32) (bool, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	CountByStatusAndPriority(ctx context.Context, status string, priority string) (int64, error)
	CountSince(ctx context.Context, since string) (int64, error)
}

// ContentFlagRepository defines persistence operations for safety_content_flags. [11-safety §6.2]
type ContentFlagRepository interface {
	Create(ctx context.Context, input CreateContentFlagRow) (*ContentFlag, error)
	FindByID(ctx context.Context, flagID uuid.UUID) (*ContentFlag, error)
	ListFiltered(ctx context.Context, filter FlagFilter, pagination shared.PaginationParams) ([]ContentFlag, error)
	MarkReviewed(ctx context.Context, flagID uuid.UUID, reviewedBy uuid.UUID, actionTaken bool) (*ContentFlag, error)
	CountUnreviewed(ctx context.Context) (int64, error)
}

// ModActionRepository defines persistence operations for safety_mod_actions. [11-safety §6.3]
type ModActionRepository interface {
	Create(ctx context.Context, input CreateModActionRow) (*ModAction, error)
	FindByID(ctx context.Context, actionID uuid.UUID) (*ModAction, error)
	ListFiltered(ctx context.Context, filter ActionFilter, pagination shared.PaginationParams) ([]ModAction, error)
	ListByTargetFamily(ctx context.Context, familyID uuid.UUID, pagination shared.PaginationParams) ([]ModAction, error)
	CountSince(ctx context.Context, since string) (int64, error)
}

// AccountStatusRepository defines persistence operations for safety_account_status. [11-safety §6.4]
type AccountStatusRepository interface {
	GetOrCreate(ctx context.Context, familyID uuid.UUID) (*AccountStatusRow, error)
	Update(ctx context.Context, familyID uuid.UUID, updates AccountStatusUpdate) (*AccountStatusRow, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	FindExpiredSuspensions(ctx context.Context) ([]AccountStatusRow, error)
}

// AppealRepository defines persistence operations for safety_appeals. [11-safety §6.5]
type AppealRepository interface {
	Create(ctx context.Context, scope shared.FamilyScope, input CreateAppealRow) (*Appeal, error)
	FindByID(ctx context.Context, scope shared.FamilyScope, appealID uuid.UUID) (*Appeal, error)
	FindByIDUnscoped(ctx context.Context, appealID uuid.UUID) (*Appeal, error)
	FindByActionID(ctx context.Context, actionID uuid.UUID) (*Appeal, error)
	ListFiltered(ctx context.Context, filter AppealFilter, pagination shared.PaginationParams) ([]Appeal, error)
	Update(ctx context.Context, appealID uuid.UUID, updates AppealUpdate) (*Appeal, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
}

// NcmecReportRepository defines persistence operations for safety_ncmec_reports. [11-safety §6.6]
type NcmecReportRepository interface {
	Create(ctx context.Context, input CreateNcmecReportRow) (*NcmecReport, error)
	FindByID(ctx context.Context, reportID uuid.UUID) (*NcmecReport, error)
	UpdateStatus(ctx context.Context, reportID uuid.UUID, status string, ncmecReportID *string, errMsg *string) (*NcmecReport, error)
	FindPending(ctx context.Context) ([]NcmecReport, error)
}

// BotSignalRepository defines persistence operations for safety_bot_signals. [11-safety §6.7]
type BotSignalRepository interface {
	Create(ctx context.Context, input CreateBotSignalRow) (*BotSignal, error)
	CountRecent(ctx context.Context, parentID uuid.UUID, withinMinutes uint32) (int64, error)
}

// ManualReviewRepository defines persistence operations for safety_manual_review_queue. [11-safety §7.1, CRIT-1]
type ManualReviewRepository interface {
	Create(ctx context.Context, item *ManualReviewItem) error
	FindPending(ctx context.Context, limit int) ([]ManualReviewItem, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, reviewerNotes *string, reviewedBy *uuid.UUID) error
}

// NcmecPendingReportRepository defines persistence operations for safety_ncmec_pending_reports. [11-safety §7.1, CRIT-1]
type NcmecPendingReportRepository interface {
	Create(ctx context.Context, report *NcmecPendingReport) error
	FindQueued(ctx context.Context) ([]NcmecPendingReport, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}

// ParentalControlRepository defines persistence operations for safety_parental_controls. [11-safety §14.3]
type ParentalControlRepository interface {
	ListByFamily(ctx context.Context, familyID uuid.UUID) ([]ParentalControl, error)
	Upsert(ctx context.Context, control *ParentalControl) error
	Delete(ctx context.Context, familyID uuid.UUID, controlID uuid.UUID) error
}

// AdminRoleRepository defines persistence operations for safety_admin_roles. [11-safety §9.3]
type AdminRoleRepository interface {
	List(ctx context.Context) ([]AdminRole, error)
	FindByID(ctx context.Context, roleID uuid.UUID) (*AdminRole, error)
	Create(ctx context.Context, role *AdminRole) error
}

// AdminRoleAssignmentRepository defines persistence operations for safety_admin_role_assignments. [11-safety §9.3]
type AdminRoleAssignmentRepository interface {
	ListByRole(ctx context.Context, roleID uuid.UUID) ([]AdminRoleAssignment, error)
	ListByParent(ctx context.Context, parentID uuid.UUID) ([]AdminRoleAssignment, error)
	Create(ctx context.Context, assignment *AdminRoleAssignment) error
	Delete(ctx context.Context, roleID uuid.UUID, parentID uuid.UUID) error
}

// GroomingScoreRepository defines persistence operations for safety_grooming_scores. [11-safety §14.2]
type GroomingScoreRepository interface {
	Create(ctx context.Context, score *GroomingScore) error
	FindByID(ctx context.Context, scoreID uuid.UUID) (*GroomingScore, error)
	ListFlagged(ctx context.Context, pagination shared.PaginationParams) ([]GroomingScore, error)
	MarkReviewed(ctx context.Context, scoreID uuid.UUID, reviewedBy uuid.UUID) error
}

// ─── Adapter Interfaces ─────────────────────────────────────────────────────────

// ThornAdapter wraps the Thorn Safer API for CSAM detection and NCMEC reporting. [11-safety §7.1]
type ThornAdapter interface {
	ScanCsam(ctx context.Context, storageKey string) (*CsamScanResult, error)
	SubmitNcmecReport(ctx context.Context, report NcmecReportPayload) (*NcmecSubmissionResult, error)
	CheckHashUpdate(ctx context.Context) (bool, error)
}

// RekognitionAdapter wraps AWS Rekognition's DetectModerationLabels API. [11-safety §7.2]
type RekognitionAdapter interface {
	DetectModerationLabels(ctx context.Context, storageKey string) (*ModerationResult, error)
}

// GroomingDetector wraps ML-based grooming behavior detection. [11-safety §14.2]
// Phase 2: pluggable adapter (noop in dev, Comprehend/Perspective in production).
type GroomingDetector interface {
	Analyze(ctx context.Context, text string) (*GroomingAnalysisResult, error)
}

// IamServiceForSafety is the consumer-defined interface for iam:: methods needed by safety::. [CODING §8.2]
type IamServiceForSafety interface {
	RevokeSessions(ctx context.Context, familyID uuid.UUID) error
}

// ─── Internal Interfaces ────────────────────────────────────────────────────────

// eventPublisher is the internal event publishing interface.
type eventPublisher interface {
	Publish(ctx context.Context, event shared.DomainEvent) error
}
