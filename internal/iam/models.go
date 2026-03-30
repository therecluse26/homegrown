package iam

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── Enums ────────────────────────────────────────────────────────────────────

// CoppaConsentStatus represents the COPPA consent state machine values. [S§17.2, ARCH §6.3]
type CoppaConsentStatus string

const (
	CoppaConsentRegistered CoppaConsentStatus = "registered"
	CoppaConsentNoticed    CoppaConsentStatus = "noticed"
	CoppaConsentConsented  CoppaConsentStatus = "consented"
	CoppaConsentReVerified CoppaConsentStatus = "re_verified"
	CoppaConsentWithdrawn  CoppaConsentStatus = "withdrawn"
)

// CanCreateStudents returns true when consent status allows student profile creation. [S§17.2]
func (s CoppaConsentStatus) CanCreateStudents() bool {
	return s == CoppaConsentConsented || s == CoppaConsentReVerified
}

// ─── Custom DB Types ──────────────────────────────────────────────────────────

// SlugArray is a custom type for PostgreSQL TEXT[] columns holding methodology slugs.
// Implements database/sql.Scanner and driver.Valuer without requiring lib/pq. [Plan §2]
type SlugArray []string

// Value implements driver.Valuer. Serializes to PostgreSQL array literal.
func (a SlugArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	return "{" + strings.Join(a, ",") + "}", nil
}

// Scan implements sql.Scanner. Parses PostgreSQL array literal {slug1,slug2,...}.
func (a *SlugArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}
	var str string
	switch v := src.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("SlugArray.Scan: unsupported type %T", src)
	}
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*a = SlugArray{}
		return nil
	}
	parts := strings.Split(str, ",")
	result := make(SlugArray, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	*a = result
	return nil
}

// ─── GORM Models ──────────────────────────────────────────────────────────────

// FamilyModel is the GORM model for the iam_families table.
type FamilyModel struct {
	ID                       uuid.UUID  `gorm:"type:uuid;primaryKey"`
	DisplayName              string     `gorm:"not null"`
	StateCode                *string    `gorm:"type:char(2)"`
	LocationRegion           *string
	PrimaryParentID          *uuid.UUID `gorm:"type:uuid"`
	PrimaryMethodologySlug   string     `gorm:"not null"`
	SecondaryMethodologySlugs SlugArray  `gorm:"type:text[]"`
	SubscriptionTier         string     `gorm:"not null;default:free"`
	CoppaConsentStatus       string     `gorm:"not null;default:registered"`
	CoppaConsentedAt         *time.Time
	CoppaConsentMethod       *string
	DeletionRequestedAt      *time.Time
	CreatedAt                time.Time  `gorm:"not null"`
	UpdatedAt                time.Time  `gorm:"not null"`
}

func (FamilyModel) TableName() string { return "iam_families" }

func (m *FamilyModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

func (m *FamilyModel) toDomain() *Family {
	var secondary []string
	if m.SecondaryMethodologySlugs != nil {
		secondary = []string(m.SecondaryMethodologySlugs)
	}
	return &Family{
		ID:                       m.ID,
		DisplayName:              m.DisplayName,
		StateCode:                m.StateCode,
		LocationRegion:           m.LocationRegion,
		PrimaryParentID:          m.PrimaryParentID,
		PrimaryMethodologySlug:   m.PrimaryMethodologySlug,
		SecondaryMethodologySlugs: secondary,
		SubscriptionTier:         m.SubscriptionTier,
		CoppaConsentStatus:       CoppaConsentStatus(m.CoppaConsentStatus),
		CoppaConsentedAt:         m.CoppaConsentedAt,
		CoppaConsentMethod:       m.CoppaConsentMethod,
		DeletionRequestedAt:      m.DeletionRequestedAt,
		CreatedAt:                m.CreatedAt,
		UpdatedAt:                m.UpdatedAt,
	}
}

// ParentModel is the GORM model for the iam_parents table.
type ParentModel struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	FamilyID         uuid.UUID `gorm:"type:uuid;not null"`
	KratosIdentityID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	DisplayName      string    `gorm:"not null"`
	Email            string    `gorm:"not null"`
	IsPrimary        bool      `gorm:"not null;default:false"`
	IsPlatformAdmin  bool      `gorm:"not null;default:false"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

func (ParentModel) TableName() string { return "iam_parents" }

func (m *ParentModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

func (m *ParentModel) toDomain() *Parent {
	return &Parent{
		ID:              m.ID,
		FamilyID:        m.FamilyID,
		IdentityID:      m.KratosIdentityID,
		DisplayName:     m.DisplayName,
		Email:           m.Email,
		IsPrimary:       m.IsPrimary,
		IsPlatformAdmin: m.IsPlatformAdmin,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

// StudentModel is the GORM model for the iam_students table.
type StudentModel struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey"`
	FamilyID                uuid.UUID `gorm:"type:uuid;not null"`
	DisplayName             string    `gorm:"not null"`
	BirthYear               *int16
	GradeLevel              *string
	MethodologyOverrideSlug *string
	CreatedAt               time.Time `gorm:"not null"`
	UpdatedAt               time.Time `gorm:"not null"`
}

func (StudentModel) TableName() string { return "iam_students" }

func (m *StudentModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

func (m *StudentModel) toDomain() *Student {
	return &Student{
		ID:                      m.ID,
		FamilyID:                m.FamilyID,
		DisplayName:             m.DisplayName,
		BirthYear:               m.BirthYear,
		GradeLevel:              m.GradeLevel,
		MethodologyOverrideSlug: m.MethodologyOverrideSlug,
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
	}
}

// CoppaAuditLogModel is the GORM model for iam_coppa_audit_log.
type CoppaAuditLogModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	FamilyID       uuid.UUID `gorm:"type:uuid;not null"`
	Action         string    `gorm:"not null"`
	Method         *string
	PreviousStatus string    `gorm:"type:iam_coppa_consent_enum;not null"`
	NewStatus      string    `gorm:"type:iam_coppa_consent_enum;not null"`
	PerformedBy    uuid.UUID `gorm:"type:uuid;not null"`
	IPHash         *string   `gorm:"column:ip_hash"`
	CreatedAt      time.Time `gorm:"not null"`
}

func (CoppaAuditLogModel) TableName() string { return "iam_coppa_audit_log" }

func (m *CoppaAuditLogModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ─── Internal Domain Types ────────────────────────────────────────────────────

// Family is the internal domain representation of a family account. [S§3.1.1]
type Family struct {
	ID                       uuid.UUID
	DisplayName              string
	StateCode                *string
	LocationRegion           *string
	PrimaryParentID          *uuid.UUID
	PrimaryMethodologySlug   string
	SecondaryMethodologySlugs []string
	SubscriptionTier         string
	CoppaConsentStatus       CoppaConsentStatus
	CoppaConsentedAt         *time.Time
	CoppaConsentMethod       *string
	DeletionRequestedAt      *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// Parent is the internal domain representation of a parent user. [S§3.1.2]
type Parent struct {
	ID              uuid.UUID
	FamilyID        uuid.UUID
	IdentityID      uuid.UUID
	DisplayName     string
	Email           string // PII — never log [CODING §5.2]
	IsPrimary       bool
	IsPlatformAdmin bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Student is the internal domain representation of a student profile. [S§3.1.3]
type Student struct {
	ID                      uuid.UUID
	FamilyID                uuid.UUID
	DisplayName             string
	BirthYear               *int16
	GradeLevel              *string
	MethodologyOverrideSlug *string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// KratosSession holds session data returned by the Kratos public API.
type KratosSession struct {
	IdentityID      uuid.UUID
	Active          bool
	AuthenticatedAt time.Time
}

// KratosAdminSession holds session data returned by the Kratos admin API.
// Used by the lifecycle domain for session listing. [15-data-lifecycle §12]
type KratosAdminSession struct {
	SessionID  string
	UserAgent  *string
	IPAddress  *string
	LastActive time.Time
}

// KratosIdentity holds identity traits returned by the Kratos admin API.
type KratosIdentity struct {
	ID    uuid.UUID
	Email string // PII — never log [CODING §5.2]
	Name  string
}

// ─── Repository Command Types ─────────────────────────────────────────────────

// CreateFamily is the command type for FamilyRepository.Create.
type CreateFamily struct {
	DisplayName            string
	PrimaryMethodologySlug string
}

// CreateParent is the command type for ParentRepository.Create.
type CreateParent struct {
	FamilyID    uuid.UUID
	IdentityID  uuid.UUID
	DisplayName string
	Email       string // PII — never log [CODING §5.2]
	IsPrimary   bool
}

// CreateStudent is the command type for StudentRepository.Create.
type CreateStudent struct {
	DisplayName             string
	BirthYear               *int16
	GradeLevel              *string
	MethodologyOverrideSlug *string
}

// UpdateFamily is the command type for FamilyRepository.Update.
type UpdateFamily struct {
	DisplayName    *string
	StateCode      *string
	LocationRegion *string
}

// UpdateParent is the command type for ParentRepository.Update.
type UpdateParent struct {
	DisplayName *string
	Email       *string // PII — never log [CODING §5.2]
}

// UpdateStudent is the command type for StudentRepository.Update.
type UpdateStudent struct {
	DisplayName             *string
	BirthYear               *int16
	GradeLevel              *string
	MethodologyOverrideSlug **string // nil = don't change; non-nil pointing to nil = clear
}

// ─── API Request Types ────────────────────────────────────────────────────────

// CreateStudentCommand is the request body for POST /v1/families/students. [§4.3]
type CreateStudentCommand struct {
	DisplayName             string  `json:"display_name"                    validate:"required,min=1,max=100"`
	BirthYear               *int16  `json:"birth_year,omitempty"            validate:"omitempty,min=2000,max=2030"`
	GradeLevel              *string `json:"grade_level,omitempty"           validate:"omitempty,max=20"`
	MethodologyOverrideSlug *string `json:"methodology_override_slug,omitempty"`
}

// UpdateStudentCommand is the request body for PATCH /v1/families/students/:id. [§4.3]
type UpdateStudentCommand struct {
	DisplayName             *string  `json:"display_name,omitempty"              validate:"omitempty,min=1,max=100"`
	BirthYear               *int16   `json:"birth_year,omitempty"               validate:"omitempty,min=2000,max=2030"`
	GradeLevel              *string  `json:"grade_level,omitempty"              validate:"omitempty,max=20"`
	MethodologyOverrideSlug **string `json:"methodology_override_slug,omitempty"`
}

// UpdateFamilyCommand is the request body for PATCH /v1/families/profile. [§4.3]
type UpdateFamilyCommand struct {
	DisplayName    *string `json:"display_name,omitempty"     validate:"omitempty,min=1,max=100"`
	StateCode      *string `json:"state_code,omitempty"       validate:"omitempty,len=2"`
	LocationRegion *string `json:"location_region,omitempty"  validate:"omitempty,max=200"`
}

// CoppaConsentCommand is the request body for POST /v1/families/consent. [§4.3]
type CoppaConsentCommand struct {
	Method                  string `json:"method"                    validate:"required"`
	VerificationToken       string `json:"verification_token"        validate:"omitempty"`
	CoppaNoticeAcknowledged bool   `json:"coppa_notice_acknowledged" validate:"required"`
}

// ─── Webhook Types ────────────────────────────────────────────────────────────

// KratosWebhookPayload is the body sent by Kratos post-registration and post-login hooks.
type KratosWebhookPayload struct {
	IdentityID uuid.UUID    `json:"identity_id"`
	Traits     KratosTraits `json:"traits"`
}

// KratosTraits contains the identity fields synced from Kratos. [§10.1]
type KratosTraits struct {
	Email string `json:"email"` // PII — never log [CODING §5.2]
	Name  string `json:"name"`
}

// ─── API Response Types ───────────────────────────────────────────────────────

// CurrentUserResponse is returned by GET /v1/auth/me. [§4.3]
type CurrentUserResponse struct {
	ParentID           uuid.UUID `json:"parent_id"`
	FamilyID           uuid.UUID `json:"family_id"`
	DisplayName        string    `json:"display_name"`
	Email              string    `json:"email"`
	IsPrimaryParent    bool      `json:"is_primary_parent"`
	IsPlatformAdmin    bool      `json:"is_platform_admin"`
	SubscriptionTier   string    `json:"subscription_tier"`
	CoppaConsentStatus string    `json:"coppa_consent_status"`
	FamilyDisplayName  string    `json:"family_display_name"`
}

// FamilyProfileResponse is returned by GET/PATCH /v1/families/profile. [§4.3]
type FamilyProfileResponse struct {
	ID                       uuid.UUID       `json:"id"`
	DisplayName              string          `json:"display_name"`
	StateCode                *string         `json:"state_code,omitempty"`
	LocationRegion           *string         `json:"location_region,omitempty"`
	PrimaryMethodologySlug   string          `json:"primary_methodology_slug"`
	SecondaryMethodologySlugs []string        `json:"secondary_methodology_slugs"`
	SubscriptionTier         string          `json:"subscription_tier"`
	CoppaConsentStatus       string          `json:"coppa_consent_status"`
	Parents                  []ParentSummary `json:"parents"`
	StudentCount             int             `json:"student_count"`
	CreatedAt                time.Time       `json:"created_at"`
}

// ParentSummary is embedded in FamilyProfileResponse.
type ParentSummary struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	IsPrimary   bool      `json:"is_primary"`
}

// StudentResponse is returned by student CRUD endpoints. [§4.3]
type StudentResponse struct {
	ID                      uuid.UUID `json:"id"`
	DisplayName             string    `json:"display_name"`
	BirthYear               *int16    `json:"birth_year,omitempty"`
	GradeLevel              *string   `json:"grade_level,omitempty"`
	MethodologyOverrideSlug *string   `json:"methodology_override_slug,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// ConsentStatusResponse is returned by GET/POST /v1/families/consent. [§4.3]
type ConsentStatusResponse struct {
	Status            string     `json:"status"`
	ConsentedAt       *time.Time `json:"consented_at,omitempty"`
	ConsentMethod     *string    `json:"consent_method,omitempty"`
	CanCreateStudents bool       `json:"can_create_students"`
}

// ─── Phase 2 GORM Models ──────────────────────────────────────────────────────

// CoParentInviteModel is the GORM model for the iam_co_parent_invites table. [§5]
type CoParentInviteModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	FamilyID  uuid.UUID `gorm:"type:uuid;not null"`
	Email     string    `gorm:"not null"`
	TokenHash string    `gorm:"not null"`
	Status    string    `gorm:"not null;default:pending"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (CoParentInviteModel) TableName() string { return "iam_co_parent_invites" }

func (m *CoParentInviteModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

func (m *CoParentInviteModel) toDomain() *CoParentInvite {
	return &CoParentInvite{
		ID:        m.ID,
		FamilyID:  m.FamilyID,
		Email:     m.Email,
		Status:    m.Status,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}
}

// StudentSessionModel is the GORM model for the iam_student_sessions table. [§5]
type StudentSessionModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	FamilyID    uuid.UUID `gorm:"type:uuid;not null"`
	StudentID   uuid.UUID `gorm:"type:uuid;not null"`
	CreatedBy   uuid.UUID `gorm:"type:uuid;not null"`
	TokenHash   string    `gorm:"not null"`
	IsActive    bool      `gorm:"not null;default:true"`
	ExpiresAt   time.Time `gorm:"not null"`
	Permissions SlugArray `gorm:"type:text[]"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (StudentSessionModel) TableName() string { return "iam_student_sessions" }

func (m *StudentSessionModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

func (m *StudentSessionModel) toDomain() *StudentSession {
	var perms []string
	if m.Permissions != nil {
		perms = []string(m.Permissions)
	}
	return &StudentSession{
		ID:          m.ID,
		FamilyID:    m.FamilyID,
		StudentID:   m.StudentID,
		CreatedBy:   m.CreatedBy,
		IsActive:    m.IsActive,
		ExpiresAt:   m.ExpiresAt,
		Permissions: perms,
		CreatedAt:   m.CreatedAt,
	}
}

// ─── Phase 2 Domain Types ─────────────────────────────────────────────────────

// CoParentInvite is the internal domain type for a co-parent invite. [§5]
type CoParentInvite struct {
	ID        uuid.UUID
	FamilyID  uuid.UUID
	Email     string // PII — never log [CODING §5.2]
	Status    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// StudentSession is the internal domain type for a student session token. [§5]
type StudentSession struct {
	ID          uuid.UUID
	FamilyID    uuid.UUID
	StudentID   uuid.UUID
	CreatedBy   uuid.UUID
	IsActive    bool
	ExpiresAt   time.Time
	Permissions []string
	CreatedAt   time.Time
}

// ─── Phase 2 API Request/Response Types ──────────────────────────────────────

// InviteCoParentCommand is the request body for POST /v1/families/invites. [§5]
type InviteCoParentCommand struct {
	Email string `json:"email" validate:"required,email"`
}

// CoParentInviteResponse is returned by invite endpoints. [§5]
type CoParentInviteResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateStudentSessionCommand is the request body for POST /v1/families/students/:id/sessions. [§5]
type CreateStudentSessionCommand struct {
	ExpiresInHours   int      `json:"expires_in_hours"    validate:"required,min=1,max=720"`
	AllowedToolSlugs []string `json:"allowed_tool_slugs"  validate:"required,min=1"`
}

// StudentSessionResponse is returned after session creation (includes plaintext token). [§5]
// Token is returned once only; not stored.
type StudentSessionResponse struct {
	ID          uuid.UUID `json:"id"`
	StudentID   uuid.UUID `json:"student_id"`
	Token       string    `json:"token"` // plaintext — returned once only [CODING §5.2]
	ExpiresAt   time.Time `json:"expires_at"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// StudentSessionSummaryResponse is returned by list/revoke session endpoints. [§5]
type StudentSessionSummaryResponse struct {
	ID          uuid.UUID `json:"id"`
	StudentID   uuid.UUID `json:"student_id"`
	ExpiresAt   time.Time `json:"expires_at"`
	IsActive    bool      `json:"is_active"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// StudentSessionIdentityResponse is returned by GET /v1/student-session/me. [§5]
type StudentSessionIdentityResponse struct {
	StudentID        uuid.UUID `json:"student_id"`
	FamilyID         uuid.UUID `json:"family_id"`
	AllowedToolSlugs []string  `json:"allowed_tool_slugs"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// TransferPrimaryCommand is the request body for POST /v1/families/primary-parent. [§5]
type TransferPrimaryCommand struct {
	NewPrimaryParentID uuid.UUID `json:"new_primary_parent_id" validate:"required"`
}
