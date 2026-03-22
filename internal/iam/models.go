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
	CoppaConsentRegistered  CoppaConsentStatus = "registered"
	CoppaConsentNoticed     CoppaConsentStatus = "noticed"
	CoppaConsentConsented   CoppaConsentStatus = "consented"
	CoppaConsentReVerified  CoppaConsentStatus = "re_verified"
	CoppaConsentWithdrawn   CoppaConsentStatus = "withdrawn"
)

// CanCreateStudents returns true when consent status allows student profile creation. [S§17.2]
func (s CoppaConsentStatus) CanCreateStudents() bool {
	return s == CoppaConsentConsented || s == CoppaConsentReVerified
}

// ─── Custom DB Types ──────────────────────────────────────────────────────────

// UUIDArray is a custom type for PostgreSQL UUID arrays.
// Implements database/sql.Scanner and driver.Valuer without requiring lib/pq. [Plan §2]
type UUIDArray []uuid.UUID

// Value implements driver.Valuer. Serializes to PostgreSQL array literal.
func (a UUIDArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	elements := make([]string, len(a))
	for i, id := range a {
		elements[i] = id.String()
	}
	return "{" + strings.Join(elements, ",") + "}", nil
}

// Scan implements sql.Scanner. Parses PostgreSQL array literal {uuid1,uuid2,...}.
func (a *UUIDArray) Scan(src interface{}) error {
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
		return fmt.Errorf("UUIDArray.Scan: unsupported type %T", src)
	}
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*a = UUIDArray{}
		return nil
	}
	parts := strings.Split(str, ",")
	result := make(UUIDArray, len(parts))
	for i, p := range parts {
		id, err := uuid.Parse(strings.TrimSpace(p))
		if err != nil {
			return fmt.Errorf("UUIDArray.Scan: invalid UUID %q: %w", p, err)
		}
		result[i] = id
	}
	*a = result
	return nil
}

// ─── GORM Models ──────────────────────────────────────────────────────────────

// FamilyModel is the GORM model for the iam_families table.
type FamilyModel struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey"`
	DisplayName            string     `gorm:"not null"`
	StateCode              *string    `gorm:"type:char(2)"`
	LocationRegion         *string
	PrimaryParentID        *uuid.UUID `gorm:"type:uuid"`
	PrimaryMethodologyID   uuid.UUID  `gorm:"type:uuid;not null"`
	SecondaryMethodologyIDs UUIDArray  `gorm:"type:uuid[]"`
	SubscriptionTier       string     `gorm:"not null;default:free"`
	CoppaConsentStatus     string     `gorm:"not null;default:registered"`
	CoppaConsentedAt       *time.Time
	CoppaConsentMethod     *string
	DeletionRequestedAt    *time.Time
	CreatedAt              time.Time  `gorm:"not null"`
	UpdatedAt              time.Time  `gorm:"not null"`
}

func (FamilyModel) TableName() string { return "iam_families" }

func (m *FamilyModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

func (m *FamilyModel) toDomain() *Family {
	var secondary []uuid.UUID
	if m.SecondaryMethodologyIDs != nil {
		secondary = []uuid.UUID(m.SecondaryMethodologyIDs)
	}
	return &Family{
		ID:                     m.ID,
		DisplayName:            m.DisplayName,
		StateCode:              m.StateCode,
		LocationRegion:         m.LocationRegion,
		PrimaryParentID:        m.PrimaryParentID,
		PrimaryMethodologyID:   m.PrimaryMethodologyID,
		SecondaryMethodologyIDs: secondary,
		SubscriptionTier:       m.SubscriptionTier,
		CoppaConsentStatus:     CoppaConsentStatus(m.CoppaConsentStatus),
		CoppaConsentedAt:       m.CoppaConsentedAt,
		CoppaConsentMethod:     m.CoppaConsentMethod,
		DeletionRequestedAt:    m.DeletionRequestedAt,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
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
		m.ID = uuid.New()
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
	ID                    uuid.UUID  `gorm:"type:uuid;primaryKey"`
	FamilyID              uuid.UUID  `gorm:"type:uuid;not null"`
	DisplayName           string     `gorm:"not null"`
	BirthYear             *int16
	GradeLevel            *string
	MethodologyOverrideID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt             time.Time  `gorm:"not null"`
	UpdatedAt             time.Time  `gorm:"not null"`
}

func (StudentModel) TableName() string { return "iam_students" }

func (m *StudentModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

func (m *StudentModel) toDomain() *Student {
	return &Student{
		ID:                    m.ID,
		FamilyID:              m.FamilyID,
		DisplayName:           m.DisplayName,
		BirthYear:             m.BirthYear,
		GradeLevel:            m.GradeLevel,
		MethodologyOverrideID: m.MethodologyOverrideID,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
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
		m.ID = uuid.New()
	}
	return nil
}

// ─── Internal Domain Types ────────────────────────────────────────────────────

// Family is the internal domain representation of a family account. [S§3.1.1]
type Family struct {
	ID                     uuid.UUID
	DisplayName            string
	StateCode              *string
	LocationRegion         *string
	PrimaryParentID        *uuid.UUID
	PrimaryMethodologyID   uuid.UUID
	SecondaryMethodologyIDs []uuid.UUID
	SubscriptionTier       string
	CoppaConsentStatus     CoppaConsentStatus
	CoppaConsentedAt       *time.Time
	CoppaConsentMethod     *string
	DeletionRequestedAt    *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
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
	ID                    uuid.UUID
	FamilyID              uuid.UUID
	DisplayName           string
	BirthYear             *int16
	GradeLevel            *string
	MethodologyOverrideID *uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// KratosSession holds session data returned by the Kratos public API.
type KratosSession struct {
	IdentityID      uuid.UUID
	Active          bool
	AuthenticatedAt time.Time
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
	DisplayName          string
	PrimaryMethodologyID uuid.UUID
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
	DisplayName           string
	BirthYear             *int16
	GradeLevel            *string
	MethodologyOverrideID *uuid.UUID
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
	DisplayName           *string
	BirthYear             *int16
	GradeLevel            *string
	MethodologyOverrideID **uuid.UUID // nil = don't change; non-nil pointing to nil = clear
}

// ─── API Request Types ────────────────────────────────────────────────────────

// CreateStudentCommand is the request body for POST /v1/families/students. [§4.3]
type CreateStudentCommand struct {
	DisplayName           string     `json:"display_name"            validate:"required,min=1,max=100"`
	BirthYear             *int16     `json:"birth_year,omitempty"    validate:"omitempty,min=2000,max=2030"`
	GradeLevel            *string    `json:"grade_level,omitempty"   validate:"omitempty,max=20"`
	MethodologyOverrideID *uuid.UUID `json:"methodology_override_id,omitempty"`
}

// UpdateStudentCommand is the request body for PATCH /v1/families/students/:id. [§4.3]
type UpdateStudentCommand struct {
	DisplayName           *string    `json:"display_name,omitempty"           validate:"omitempty,min=1,max=100"`
	BirthYear             *int16     `json:"birth_year,omitempty"             validate:"omitempty,min=2000,max=2030"`
	GradeLevel            *string    `json:"grade_level,omitempty"            validate:"omitempty,max=20"`
	MethodologyOverrideID **uuid.UUID `json:"methodology_override_id,omitempty"`
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
	VerificationToken       string `json:"verification_token"        validate:"required"`
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
	SubscriptionTier   string    `json:"subscription_tier"`
	CoppaConsentStatus string    `json:"coppa_consent_status"`
	FamilyDisplayName  string    `json:"family_display_name"`
}

// FamilyProfileResponse is returned by GET/PATCH /v1/families/profile. [§4.3]
type FamilyProfileResponse struct {
	ID                      uuid.UUID       `json:"id"`
	DisplayName             string          `json:"display_name"`
	StateCode               *string         `json:"state_code,omitempty"`
	LocationRegion          *string         `json:"location_region,omitempty"`
	PrimaryMethodologyID    uuid.UUID       `json:"primary_methodology_id"`
	SecondaryMethodologyIDs []uuid.UUID     `json:"secondary_methodology_ids"`
	SubscriptionTier        string          `json:"subscription_tier"`
	CoppaConsentStatus      string          `json:"coppa_consent_status"`
	Parents                 []ParentSummary `json:"parents"`
	StudentCount            int             `json:"student_count"`
	CreatedAt               time.Time       `json:"created_at"`
}

// ParentSummary is embedded in FamilyProfileResponse.
type ParentSummary struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	IsPrimary   bool      `json:"is_primary"`
}

// StudentResponse is returned by student CRUD endpoints. [§4.3]
type StudentResponse struct {
	ID                    uuid.UUID  `json:"id"`
	DisplayName           string     `json:"display_name"`
	BirthYear             *int16     `json:"birth_year,omitempty"`
	GradeLevel            *string    `json:"grade_level,omitempty"`
	MethodologyOverrideID *uuid.UUID `json:"methodology_override_id,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// ConsentStatusResponse is returned by GET/POST /v1/families/consent. [§4.3]
type ConsentStatusResponse struct {
	Status            string     `json:"status"`
	ConsentedAt       *time.Time `json:"consented_at,omitempty"`
	ConsentMethod     *string    `json:"consent_method,omitempty"`
	CanCreateStudents bool       `json:"can_create_students"`
}
