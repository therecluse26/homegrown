package safety

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── GORM Models ────────────────────────────────────────────────────────────────

// Report maps to the safety_reports table. [11-safety §3.2]
type Report struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`
	ReporterFamilyID uuid.UUID  `gorm:"type:uuid;not null"`
	ReporterParentID uuid.UUID  `gorm:"type:uuid;not null"`
	TargetType       string     `gorm:"type:text;not null"`
	TargetID         uuid.UUID  `gorm:"type:uuid;not null"`
	TargetFamilyID   *uuid.UUID `gorm:"type:uuid"`
	Category         string     `gorm:"type:text;not null"`
	Description      *string    `gorm:"type:text"`
	Priority         string     `gorm:"type:text;not null;default:normal"`
	Status           string     `gorm:"type:text;not null;default:pending"`
	AssignedAdminID  *uuid.UUID `gorm:"type:uuid"`
	ResolvedAt       *time.Time `gorm:"type:timestamptz"`
	CreatedAt        time.Time  `gorm:"type:timestamptz;not null"`
	UpdatedAt        time.Time  `gorm:"type:timestamptz;not null"`
}

func (Report) TableName() string { return "safety_reports" }

func (r *Report) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

// ContentFlag maps to the safety_content_flags table. [11-safety §3.2]
type ContentFlag struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey"`
	Source         string           `gorm:"type:text;not null"`
	TargetType     string           `gorm:"type:text;not null"`
	TargetID       uuid.UUID        `gorm:"type:uuid;not null"`
	TargetFamilyID *uuid.UUID       `gorm:"type:uuid"`
	FlagType       string           `gorm:"type:text;not null"`
	Confidence     *float64         `gorm:"type:double precision"`
	Labels         *json.RawMessage `gorm:"type:jsonb"`
	ReportID       *uuid.UUID       `gorm:"type:uuid"`
	AutoRejected   bool             `gorm:"not null;default:false"`
	Reviewed       bool             `gorm:"not null;default:false"`
	ReviewedBy     *uuid.UUID       `gorm:"type:uuid"`
	ReviewedAt     *time.Time       `gorm:"type:timestamptz"`
	ActionTaken    *bool            `gorm:"type:boolean"`
	CreatedAt      time.Time        `gorm:"type:timestamptz;not null"`
}

func (ContentFlag) TableName() string { return "safety_content_flags" }

func (f *ContentFlag) BeforeCreate(_ *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

// ModAction maps to the safety_mod_actions table. [11-safety §3.2]
type ModAction struct {
	ID                  uuid.UUID        `gorm:"type:uuid;primaryKey"`
	AdminID             uuid.UUID        `gorm:"type:uuid;not null"`
	TargetFamilyID      uuid.UUID        `gorm:"type:uuid;not null"`
	TargetParentID      *uuid.UUID       `gorm:"type:uuid"`
	ActionType          string           `gorm:"type:text;not null"`
	Reason              string           `gorm:"type:text;not null"`
	ReportID            *uuid.UUID       `gorm:"type:uuid"`
	ContentSnapshot     *json.RawMessage `gorm:"type:jsonb"`
	SuspensionDays      *int32           `gorm:"type:integer"`
	SuspensionExpiresAt *time.Time       `gorm:"type:timestamptz"`
	CreatedAt           time.Time        `gorm:"type:timestamptz;not null"`
}

func (ModAction) TableName() string { return "safety_mod_actions" }

func (a *ModAction) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

// AccountStatusRow maps to the safety_account_status table. [11-safety §3.2]
type AccountStatusRow struct {
	FamilyID            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Status              string     `gorm:"type:text;not null;default:active"`
	SuspendedAt         *time.Time `gorm:"type:timestamptz"`
	SuspensionExpiresAt *time.Time `gorm:"type:timestamptz"`
	SuspensionReason    *string    `gorm:"type:text"`
	BannedAt            *time.Time `gorm:"type:timestamptz"`
	BanReason           *string    `gorm:"type:text"`
	LastActionID        *uuid.UUID `gorm:"type:uuid"`
	CreatedAt           time.Time  `gorm:"type:timestamptz;not null"`
	UpdatedAt           time.Time  `gorm:"type:timestamptz;not null"`
}

func (AccountStatusRow) TableName() string { return "safety_account_status" }

// Appeal maps to the safety_appeals table. [11-safety §3.2]
type Appeal struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	FamilyID        uuid.UUID  `gorm:"type:uuid;not null"`
	ActionID        uuid.UUID  `gorm:"type:uuid;not null"`
	AppealText      string     `gorm:"type:text;not null"`
	Status          string     `gorm:"type:text;not null;default:pending"`
	AssignedAdminID *uuid.UUID `gorm:"type:uuid"`
	ResolutionText  *string    `gorm:"type:text"`
	ResolvedAt      *time.Time `gorm:"type:timestamptz"`
	CreatedAt       time.Time  `gorm:"type:timestamptz;not null"`
	UpdatedAt       time.Time  `gorm:"type:timestamptz;not null"`
}

func (Appeal) TableName() string { return "safety_appeals" }

func (a *Appeal) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

// NcmecReport maps to the safety_ncmec_reports table. [11-safety §3.2]
type NcmecReport struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UploadID           uuid.UUID  `gorm:"type:uuid;not null"`
	FamilyID           uuid.UUID  `gorm:"type:uuid;not null"`
	ParentID           uuid.UUID  `gorm:"type:uuid;not null"`
	CsamHash           *string    `gorm:"type:text"`
	Confidence         *float64   `gorm:"type:double precision"`
	MatchedDatabase    *string    `gorm:"type:text"`
	NcmecReportID      *string    `gorm:"type:text"`
	Status             string     `gorm:"type:text;not null;default:pending"`
	SubmittedAt        *time.Time `gorm:"type:timestamptz"`
	ErrorMessage       *string    `gorm:"type:text"`
	EvidenceStorageKey string     `gorm:"type:text;not null"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null"`
}

func (NcmecReport) TableName() string { return "safety_ncmec_reports" }

func (r *NcmecReport) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

// BotSignal maps to the safety_bot_signals table. [11-safety §3.2]
type BotSignal struct {
	ID         uuid.UUID       `gorm:"type:uuid;primaryKey"`
	FamilyID   uuid.UUID       `gorm:"type:uuid;not null"`
	ParentID   uuid.UUID       `gorm:"type:uuid;not null"`
	SignalType string          `gorm:"type:text;not null"`
	Details    json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt  time.Time       `gorm:"type:timestamptz;not null"`
}

func (BotSignal) TableName() string { return "safety_bot_signals" }

func (s *BotSignal) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

// ─── Request Types ──────────────────────────────────────────────────────────────

// CreateReportCommand is the HTTP request body for POST /v1/safety/reports. [11-safety §8.1]
type CreateReportCommand struct {
	TargetType  string    `json:"target_type" validate:"required"`
	TargetID    uuid.UUID `json:"target_id" validate:"required"`
	Description *string   `json:"description,omitempty" validate:"omitempty,max=2000"`
	Category    string    `json:"category" validate:"required"`
}

// CreateAppealCommand is the HTTP request body for POST /v1/safety/appeals. [11-safety §8.1]
type CreateAppealCommand struct {
	ActionID   uuid.UUID `json:"action_id" validate:"required"`
	AppealText string    `json:"appeal_text" validate:"required,min=10,max=5000"`
}

// CreateModActionCommand is the HTTP request body for POST /v1/admin/safety/actions. [11-safety §8.1]
type CreateModActionCommand struct {
	TargetFamilyID uuid.UUID  `json:"target_family_id" validate:"required"`
	TargetParentID *uuid.UUID `json:"target_parent_id,omitempty"`
	ActionType     string     `json:"action_type" validate:"required"`
	Reason         string     `json:"reason" validate:"required,min=5,max=2000"`
	ReportID       *uuid.UUID `json:"report_id,omitempty"`
	SuspensionDays *int32     `json:"suspension_days,omitempty"`
}

// SuspendAccountCommand is the HTTP request body for POST /v1/admin/safety/accounts/:family_id/suspend.
type SuspendAccountCommand struct {
	Reason         string     `json:"reason" validate:"required,min=5,max=2000"`
	SuspensionDays int32      `json:"suspension_days" validate:"required,min=1,max=365"`
	ReportID       *uuid.UUID `json:"report_id,omitempty"`
}

// BanAccountCommand is the HTTP request body for POST /v1/admin/safety/accounts/:family_id/ban.
type BanAccountCommand struct {
	Reason   string     `json:"reason" validate:"required,min=5,max=2000"`
	ReportID *uuid.UUID `json:"report_id,omitempty"`
}

// LiftSuspensionCommand is the HTTP request body for POST /v1/admin/safety/accounts/:family_id/lift.
type LiftSuspensionCommand struct {
	Reason string `json:"reason" validate:"required,min=5,max=2000"`
}

// UpdateReportCommand is the HTTP request body for PATCH /v1/admin/safety/reports/:id.
type UpdateReportCommand struct {
	AssignedAdminID *uuid.UUID `json:"assigned_admin_id,omitempty"`
	Status          *string    `json:"status,omitempty"`
}

// ReviewFlagCommand is the HTTP request body for PATCH /v1/admin/safety/flags/:id.
type ReviewFlagCommand struct {
	ActionTaken bool `json:"action_taken"`
}

// ResolveAppealCommand is the HTTP request body for PATCH /v1/admin/safety/appeals/:id.
type ResolveAppealCommand struct {
	Status         string `json:"status" validate:"required"`
	ResolutionText string `json:"resolution_text" validate:"required,min=5,max=2000"`
}

// EscalateCsamCommand is the HTTP request body for PATCH /v1/admin/safety/flags/:id/escalate-csam. [11-safety §11.4.1]
type EscalateCsamCommand struct {
	AdminNotes string `json:"admin_notes" validate:"required,min=5,max=2000"`
}

// ─── Response Types ─────────────────────────────────────────────────────────────

// ReportResponse is the user-facing report response. [11-safety §8.2]
type ReportResponse struct {
	ID         uuid.UUID `json:"id"`
	TargetType string    `json:"target_type"`
	Category   string    `json:"category"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// AdminReportResponse is the admin-facing report response. [11-safety §8.2]
type AdminReportResponse struct {
	ID               uuid.UUID  `json:"id"`
	ReporterFamilyID uuid.UUID  `json:"reporter_family_id"`
	TargetType       string     `json:"target_type"`
	TargetID         uuid.UUID  `json:"target_id"`
	TargetFamilyID   *uuid.UUID `json:"target_family_id,omitempty"`
	Category         string     `json:"category"`
	Description      *string    `json:"description,omitempty"`
	Priority         string     `json:"priority"`
	Status           string     `json:"status"`
	AssignedAdminID  *uuid.UUID `json:"assigned_admin_id,omitempty"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ContentFlagResponse is the admin-facing content flag response. [11-safety §8.2]
type ContentFlagResponse struct {
	ID          uuid.UUID        `json:"id"`
	Source      string           `json:"source"`
	TargetType  string           `json:"target_type"`
	TargetID    uuid.UUID        `json:"target_id"`
	FlagType    string           `json:"flag_type"`
	Confidence  *float64         `json:"confidence,omitempty"`
	Labels      *json.RawMessage `json:"labels,omitempty"`
	Reviewed    bool             `json:"reviewed"`
	ReviewedBy  *uuid.UUID       `json:"reviewed_by,omitempty"`
	ActionTaken *bool            `json:"action_taken,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
}

// ModActionResponse is the moderation action response. [11-safety §8.2]
type ModActionResponse struct {
	ID                  uuid.UUID  `json:"id"`
	AdminID             uuid.UUID  `json:"admin_id"`
	TargetFamilyID      uuid.UUID  `json:"target_family_id"`
	TargetParentID      *uuid.UUID `json:"target_parent_id,omitempty"`
	ActionType          string     `json:"action_type"`
	Reason              string     `json:"reason"`
	ReportID            *uuid.UUID `json:"report_id,omitempty"`
	SuspensionDays      *int32     `json:"suspension_days,omitempty"`
	SuspensionExpiresAt *time.Time `json:"suspension_expires_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// AccountStatusResponse is the user-facing account status response. [11-safety §8.2]
type AccountStatusResponse struct {
	Status              string     `json:"status"`
	SuspendedAt         *time.Time `json:"suspended_at,omitempty"`
	SuspensionExpiresAt *time.Time `json:"suspension_expires_at,omitempty"`
	SuspensionReason    *string    `json:"suspension_reason,omitempty"`
}

// AdminAccountStatusResponse is the admin-facing account status response. [11-safety §8.2]
type AdminAccountStatusResponse struct {
	FamilyID            uuid.UUID           `json:"family_id"`
	Status              string              `json:"status"`
	SuspendedAt         *time.Time          `json:"suspended_at,omitempty"`
	SuspensionExpiresAt *time.Time          `json:"suspension_expires_at,omitempty"`
	SuspensionReason    *string             `json:"suspension_reason,omitempty"`
	BannedAt            *time.Time          `json:"banned_at,omitempty"`
	BanReason           *string             `json:"ban_reason,omitempty"`
	ActionHistory       []ModActionResponse `json:"action_history"`
}

// AppealResponse is the user-facing appeal response. [11-safety §8.2]
type AppealResponse struct {
	ID             uuid.UUID  `json:"id"`
	ActionID       uuid.UUID  `json:"action_id"`
	Status         string     `json:"status"`
	AppealText     string     `json:"appeal_text"`
	ResolutionText *string    `json:"resolution_text,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// AdminAppealResponse is the admin-facing appeal response. [11-safety §8.2]
type AdminAppealResponse struct {
	ID              uuid.UUID         `json:"id"`
	FamilyID        uuid.UUID         `json:"family_id"`
	ActionID        uuid.UUID         `json:"action_id"`
	OriginalAction  ModActionResponse `json:"original_action"`
	AppealText      string            `json:"appeal_text"`
	Status          string            `json:"status"`
	AssignedAdminID *uuid.UUID        `json:"assigned_admin_id,omitempty"`
	ResolutionText  *string           `json:"resolution_text,omitempty"`
	ResolvedAt      *time.Time        `json:"resolved_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
}

// DashboardStats is the admin dashboard statistics response. [11-safety §8.2]
type DashboardStats struct {
	PendingReports    int64 `json:"pending_reports"`
	CriticalReports   int64 `json:"critical_reports"`
	UnreviewedFlags   int64 `json:"unreviewed_flags"`
	PendingAppeals    int64 `json:"pending_appeals"`
	ActiveSuspensions int64 `json:"active_suspensions"`
	ActiveBans        int64 `json:"active_bans"`
	ReportsLast24h    int64 `json:"reports_last_24h"`
	ActionsLast24h    int64 `json:"actions_last_24h"`
}

// TextScanResult is the result of text content scanning. [11-safety §8.2]
type TextScanResult struct {
	HasViolations bool     `json:"has_violations"`
	MatchedTerms  []string `json:"matched_terms"`
	Severity      string   `json:"severity"` // "none", "low", "high", "critical"
}

// ─── Filter Types ───────────────────────────────────────────────────────────────

// ReportFilter filters reports in admin queries. [11-safety §8.3]
type ReportFilter struct {
	Status          *string    `query:"status"`
	Priority        *string    `query:"priority"`
	Category        *string    `query:"category"`
	AssignedAdminID *uuid.UUID `query:"assigned_admin_id"`
}

// FlagFilter filters content flags in admin queries. [11-safety §8.3]
type FlagFilter struct {
	Reviewed   *bool   `query:"reviewed"`
	FlagType   *string `query:"flag_type"`
	TargetType *string `query:"target_type"`
}

// ActionFilter filters moderation actions in admin queries. [11-safety §8.3]
type ActionFilter struct {
	AdminID        *uuid.UUID `query:"admin_id"`
	TargetFamilyID *uuid.UUID `query:"target_family_id"`
	ActionType     *string    `query:"action_type"`
}

// AppealFilter filters appeals in admin queries. [11-safety §8.3]
type AppealFilter struct {
	Status *string `query:"status"`
}

// ─── Internal / Adapter Types ───────────────────────────────────────────────────

// CsamScanResult is the CSAM scan result from Thorn Safer. [11-safety §8.4]
type CsamScanResult struct {
	IsCSAM          bool
	Hash            *string
	Confidence      *float64
	MatchedDatabase *string
}

// ModerationResult is the content moderation result from Rekognition. [11-safety §8.4]
type ModerationResult struct {
	HasViolations bool
	AutoReject    bool
	Labels        []ModerationLabel
	Priority      *string
}

// ModerationLabel represents a single moderation label. [11-safety §8.4]
type ModerationLabel struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	ParentName *string `json:"parent_name,omitempty"`
}

// NcmecReportPayload is the payload for Thorn Safer NCMEC submission. [11-safety §8.4]
type NcmecReportPayload struct {
	UploadID           uuid.UUID `json:"upload_id"`
	CsamHash           *string   `json:"csam_hash,omitempty"`
	Confidence         *float64  `json:"confidence,omitempty"`
	MatchedDatabase    *string   `json:"matched_database,omitempty"`
	EvidenceStorageKey string    `json:"evidence_storage_key"`
	UploaderFamilyID   uuid.UUID `json:"uploader_family_id"`
	UploaderParentID   uuid.UUID `json:"uploader_parent_id"`
	UploadTimestamp    time.Time `json:"upload_timestamp"`
}

// NcmecSubmissionResult is the result from a Thorn Safer NCMEC submission. [11-safety §8.4]
type NcmecSubmissionResult struct {
	NcmecReportID string    `json:"ncmec_report_id"`
	SubmittedAt   time.Time `json:"submitted_at"`
}

// BotSignalType enumerates bot signal types. [11-safety §8.4]
type BotSignalType string

const (
	BotSignalRapidPosting           BotSignalType = "rapid_posting"
	BotSignalMassFriendRequests     BotSignalType = "mass_friend_requests"
	BotSignalRepetitiveContent      BotSignalType = "repetitive_content"
	BotSignalSuspiciousRegistration BotSignalType = "suspicious_registration"
	BotSignalRateLimitExceeded      BotSignalType = "rate_limit_exceeded"
)

// SafetyConfig holds safety domain configuration. [11-safety §8.4]
type SafetyConfig struct {
	RekognitionMinConfidence     float64
	NudityAutoRejectLabels       []string
	BotSignalThreshold           int64
	BotSignalWindowMinutes       uint32
	AccountStatusCacheTTLSeconds uint64
}

// DefaultSafetyConfig returns the default safety configuration.
func DefaultSafetyConfig() SafetyConfig {
	return SafetyConfig{
		RekognitionMinConfidence: 70.0,
		NudityAutoRejectLabels: []string{
			"Explicit Nudity", "Nudity",
			"Graphic Male Nudity", "Graphic Female Nudity",
		},
		BotSignalThreshold:           5,
		BotSignalWindowMinutes:       60,
		AccountStatusCacheTTLSeconds: 60,
	}
}

// ─── Repository Input Types ─────────────────────────────────────────────────────

// CreateReportRow is the input for creating a new report record.
type CreateReportRow struct {
	ReporterFamilyID uuid.UUID
	ReporterParentID uuid.UUID
	TargetType       string
	TargetID         uuid.UUID
	TargetFamilyID   *uuid.UUID
	Category         string
	Description      *string
	Priority         string
}

// ReportUpdate holds updatable fields for reports.
type ReportUpdate struct {
	Status          *string
	AssignedAdminID *uuid.UUID
	ResolvedAt      *time.Time
}

// CreateContentFlagRow is the input for creating a new content flag record.
type CreateContentFlagRow struct {
	Source         string
	TargetType     string
	TargetID       uuid.UUID
	TargetFamilyID *uuid.UUID
	FlagType       string
	Confidence     *float64
	Labels         *json.RawMessage
	ReportID       *uuid.UUID
	AutoRejected   bool
}

// CreateModActionRow is the input for creating a new moderation action record.
type CreateModActionRow struct {
	AdminID             uuid.UUID
	TargetFamilyID      uuid.UUID
	TargetParentID      *uuid.UUID
	ActionType          string
	Reason              string
	ReportID            *uuid.UUID
	ContentSnapshot     *json.RawMessage
	SuspensionDays      *int32
	SuspensionExpiresAt *time.Time
}

// AccountStatusUpdate holds updatable fields for account status.
type AccountStatusUpdate struct {
	Status              *string
	SuspendedAt         *time.Time
	SuspensionExpiresAt *time.Time
	SuspensionReason    *string
	BannedAt            *time.Time
	BanReason           *string
	LastActionID        *uuid.UUID
}

// CreateAppealRow is the input for creating a new appeal record.
type CreateAppealRow struct {
	ActionID   uuid.UUID
	AppealText string
}

// AppealUpdate holds updatable fields for appeals.
type AppealUpdate struct {
	Status          *string
	AssignedAdminID *uuid.UUID
	ResolutionText  *string
	ResolvedAt      *time.Time
}

// CreateNcmecReportRow is the input for creating a new NCMEC report record.
type CreateNcmecReportRow struct {
	UploadID           uuid.UUID
	FamilyID           uuid.UUID
	ParentID           uuid.UUID
	CsamHash           *string
	Confidence         *float64
	MatchedDatabase    *string
	EvidenceStorageKey string
}

// CreateBotSignalRow is the input for creating a new bot signal record.
type CreateBotSignalRow struct {
	FamilyID   uuid.UUID
	ParentID   uuid.UUID
	SignalType string
	Details    json.RawMessage
}
