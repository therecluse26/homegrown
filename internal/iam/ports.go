package iam

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Service Interface ────────────────────────────────────────────────────────

// IamService defines all use cases exposed to handlers and other domains.
// Defined here per [CODING §8.2]. Implementation: IamServiceImpl in service.go.
type IamService interface {
	// ─── Queries ──────────────────────────────────────────────────────────────

	// GetCurrentUser returns the current user's info (parent + family summary).
	// Used by GET /v1/auth/me. Reads from AuthContext + family display_name.
	GetCurrentUser(ctx context.Context, auth *shared.AuthContext) (*CurrentUserResponse, error)

	// GetFamilyProfile returns the full family profile including parents and student count.
	// Used by GET /v1/families/profile.
	GetFamilyProfile(ctx context.Context, scope *shared.FamilyScope) (*FamilyProfileResponse, error)

	// ListStudents lists all students in the family.
	// Used by GET /v1/families/students.
	ListStudents(ctx context.Context, scope *shared.FamilyScope) ([]StudentResponse, error)

	// GetConsentStatus returns COPPA consent status from the family record.
	// Used by GET /v1/families/consent.
	GetConsentStatus(ctx context.Context, scope *shared.FamilyScope) (*ConsentStatusResponse, error)

	// ─── Commands ─────────────────────────────────────────────────────────────

	// HandlePostRegistration handles the Kratos post-registration webhook.
	// Creates family + parent atomically. Publishes FamilyCreated. [§10.1]
	HandlePostRegistration(ctx context.Context, payload KratosWebhookPayload) error

	// HandlePostLogin handles the Kratos post-login webhook.
	// Syncs Kratos identity traits (email, name) to local DB.
	HandlePostLogin(ctx context.Context, payload KratosWebhookPayload) error

	// UpdateFamilyProfile updates display_name, state_code, or location_region.
	// Does NOT update methodology (method:: domain) or subscription tier (billing:: domain).
	UpdateFamilyProfile(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamilyCommand) (*FamilyProfileResponse, error)

	// CreateStudent creates a student profile. COPPA consent is enforced by the handler
	// via RequireCoppaConsent middleware before calling this method. [§4.3]
	// Publishes StudentCreated.
	CreateStudent(ctx context.Context, scope *shared.FamilyScope, cmd CreateStudentCommand) (*StudentResponse, error)

	// UpdateStudent updates a student profile.
	UpdateStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd UpdateStudentCommand) (*StudentResponse, error)

	// DeleteStudent deletes a student profile. Publishes StudentDeleted.
	DeleteStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error

	// ─── Cross-Domain Methods (consumed by method::) ──────────────────────────

	// GetFamilyMethodologyIDs returns the family's primary and secondary methodology slugs.
	// Used by method:: for tool resolution. [02-method §11.2]
	GetFamilyMethodologyIDs(ctx context.Context, scope *shared.FamilyScope) (primarySlug string, secondarySlugs []string, err error)

	// GetStudent returns a single student by ID. Used by method:: for student tool resolution.
	// [02-method §10.2]
	GetStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*StudentResponse, error)

	// SetFamilyMethodology persists the family's methodology selection.
	// Called by method:: service after validation. [02-method §11.2, Appendix A]
	SetFamilyMethodology(ctx context.Context, scope *shared.FamilyScope, primarySlug string, secondarySlugs []string) error

	// SubmitCoppaConsent submits COPPA parental consent or acknowledges the COPPA notice.
	// Validates consent method. Publishes CoppaConsentGranted on Consented/ReVerified transition.
	// Phase 1: credit card micro-charge verification is stubbed (no Stripe call). [§9.3]
	SubmitCoppaConsent(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, cmd CoppaConsentCommand) (*ConsentStatusResponse, error)

	// RevokeFamilySessions revokes all Kratos sessions for every parent in a family.
	// Used by lifecycle:: during account deletion and safety:: on account suspension.
	// [15-data-lifecycle §12, 11-safety §7.3]
	RevokeFamilySessions(ctx context.Context, familyID uuid.UUID) error

	// GetStudentName returns the display_name for a student by ID.
	// Bypasses RLS — used by background jobs (comply PDF, plan calendar) that have no family scope.
	GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error)

	// ─── Phase 2: Co-parent Management ───────────────────────────────────────

	// InviteCoParent sends a co-parent invite email. Requires primary parent. [§5]
	InviteCoParent(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, cmd InviteCoParentCommand) (*CoParentInviteResponse, error)

	// CancelInvite cancels a pending invite. Requires primary parent. [§5]
	CancelInvite(ctx context.Context, scope *shared.FamilyScope, inviteID uuid.UUID) error

	// AcceptInvite accepts a co-parent invite by token. Requires auth (no family scope yet). [§5]
	AcceptInvite(ctx context.Context, auth *shared.AuthContext, token string) error

	// RemoveCoParent removes a co-parent from the family. Requires primary parent. [§5]
	RemoveCoParent(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, parentID uuid.UUID) error

	// TransferPrimaryParent atomically transfers primary ownership. Requires primary parent. [§5]
	TransferPrimaryParent(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, cmd TransferPrimaryCommand) error

	// ─── Phase 2: COPPA / Family Lifecycle ───────────────────────────────────

	// WithdrawCoppaConsent transitions consent status to "withdrawn". Requires primary. [§5, §9.2]
	WithdrawCoppaConsent(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext) error

	// RequestFamilyDeletion schedules family deletion. Requires primary parent. [§5]
	RequestFamilyDeletion(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext) error

	// CancelFamilyDeletion cancels a pending deletion request. Requires primary parent. [§5]
	CancelFamilyDeletion(ctx context.Context, scope *shared.FamilyScope) error

	// ─── Phase 2: Student Sessions ────────────────────────────────────────────

	// CreateStudentSession creates a time-limited student session token. [§5]
	CreateStudentSession(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, studentID uuid.UUID, cmd CreateStudentSessionCommand) (*StudentSessionResponse, error)

	// ListStudentSessions lists active sessions for a student. [§5]
	ListStudentSessions(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]StudentSessionSummaryResponse, error)

	// RevokeStudentSession revokes a student session by ID. [§5]
	RevokeStudentSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID) error

	// GetStudentSessionMe validates a student bearer token and returns the session identity. [§5]
	// Used by the student-session middleware — no family scope available.
	GetStudentSessionMe(ctx context.Context, token string) (*StudentSessionIdentityResponse, error)
}

// ─── Repository Interfaces ────────────────────────────────────────────────────

// FamilyRepository defines persistence operations for family accounts.
// Implementations: PgFamilyRepository in repository.go. [CODING §8.2]
type FamilyRepository interface {
	// Create creates a new family. NOT family-scoped (family does not exist yet).
	Create(ctx context.Context, cmd CreateFamily) (*Family, error)

	// FindByID finds a family by ID. NOT family-scoped — used by auth middleware
	// and webhook handlers before FamilyScope is constructed.
	FindByID(ctx context.Context, id uuid.UUID) (*Family, error)

	// Update updates family profile fields. Family-scoped.
	Update(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamily) (*Family, error)

	// SetPrimaryParent sets the primary_parent_id on the family. NOT family-scoped —
	// used during registration before FamilyScope is available.
	SetPrimaryParent(ctx context.Context, familyID uuid.UUID, parentID uuid.UUID) error

	// UpdateConsentStatus sets the COPPA consent status and consent method. Family-scoped.
	UpdateConsentStatus(ctx context.Context, scope *shared.FamilyScope, status CoppaConsentStatus, method *string) (*Family, error)

	// SetMethodology sets methodology slugs on the family. Called by method:: service. Family-scoped.
	SetMethodology(ctx context.Context, scope *shared.FamilyScope, primarySlug string, secondarySlugs []string) error

	// SetDeletionRequested sets or clears deletion_requested_at. Family-scoped.
	SetDeletionRequested(ctx context.Context, scope *shared.FamilyScope, requestedAt *time.Time) error
}

// ParentRepository defines persistence operations for parent users.
// Implementations: PgParentRepository in repository.go. [CODING §8.2]
type ParentRepository interface {
	// Create creates a new parent. NOT family-scoped — used during registration
	// and co-parent invite acceptance.
	Create(ctx context.Context, cmd CreateParent) (*Parent, error)

	// FindByKratosID finds a parent by Kratos identity ID. NOT family-scoped — used by
	// auth middleware and login webhook before FamilyScope is constructed.
	FindByKratosID(ctx context.Context, kratosIdentityID uuid.UUID) (*Parent, error)

	// FindByID finds a specific parent by ID. Family-scoped.
	FindByID(ctx context.Context, scope *shared.FamilyScope, parentID uuid.UUID) (*Parent, error)

	// ListByFamily lists all parents in a family. Family-scoped.
	ListByFamily(ctx context.Context, scope *shared.FamilyScope) ([]Parent, error)

	// Update updates parent fields (display_name, email sync). Family-scoped.
	Update(ctx context.Context, scope *shared.FamilyScope, parentID uuid.UUID, cmd UpdateParent) (*Parent, error)

	// Delete removes a parent from the family. Family-scoped.
	Delete(ctx context.Context, scope *shared.FamilyScope, parentID uuid.UUID) error

	// SetPrimary updates is_primary flag. Family-scoped.
	SetPrimary(ctx context.Context, scope *shared.FamilyScope, parentID uuid.UUID, isPrimary bool) error
}

// StudentRepository defines persistence operations for student profiles.
// Implementations: PgStudentRepository in repository.go. [CODING §8.2]
type StudentRepository interface {
	// Create creates a student profile. Family-scoped.
	Create(ctx context.Context, scope *shared.FamilyScope, cmd CreateStudent) (*Student, error)

	// ListByFamily lists all students in the family. Family-scoped.
	ListByFamily(ctx context.Context, scope *shared.FamilyScope) ([]Student, error)

	// FindByID finds a specific student by ID. Family-scoped.
	FindByID(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*Student, error)

	// Update updates a student profile. Family-scoped.
	Update(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd UpdateStudent) (*Student, error)

	// Delete deletes a student profile. Family-scoped.
	Delete(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error
}

// CoParentInviteRepository defines persistence for co-parent invites. [CODING §8.2]
type CoParentInviteRepository interface {
	// Create inserts a new invite record. NOT family-scoped (token generated server-side). [§5]
	Create(ctx context.Context, familyID, invitedBy uuid.UUID, email, tokenHash string, expiresAt time.Time) (*CoParentInvite, error)

	// FindByID finds an invite by ID. Family-scoped.
	FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*CoParentInvite, error)

	// FindByToken finds an invite by token hash. NOT family-scoped — used in AcceptInvite
	// before the requester has a family scope. Caller MUST use BypassRLSTransaction. [§5, §6]
	FindByToken(ctx context.Context, tokenHash string) (*CoParentInvite, error)

	// UpdateStatus updates the invite status (pending → accepted|cancelled). Family-scoped.
	UpdateStatus(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, status string) error
}

// StudentSessionRepository defines persistence for student session tokens. [CODING §8.2]
type StudentSessionRepository interface {
	// Create inserts a new session. Family-scoped.
	Create(ctx context.Context, scope *shared.FamilyScope, studentID, createdBy uuid.UUID, tokenHash string, expiresAt time.Time, permissions []string) (*StudentSession, error)

	// FindByID finds a session by ID. Family-scoped.
	FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*StudentSession, error)

	// FindByTokenHash finds a session by token hash. NOT family-scoped — used in
	// student session auth before family scope is available. Caller MUST use BypassRLSTransaction. [§5, §6]
	FindByTokenHash(ctx context.Context, tokenHash string) (*StudentSession, error)

	// ListActiveByStudent lists active, non-expired sessions for a student. Family-scoped.
	ListActiveByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]StudentSession, error)

	// Revoke marks a session as inactive. Family-scoped.
	Revoke(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error
}

// ─── Kratos Adapter Interface ─────────────────────────────────────────────────

// KratosAdapter defines the admin-level Kratos API operations used by the IAM service.
// The adapter also implements shared.SessionValidator (ValidateSession) for auth middleware.
// KratosAdapterImpl is in internal/iam/adapters/kratos.go. [CODING §8.1, ARCH §4.2]
//
// Note: ValidateSession is on shared.SessionValidator, not here, to avoid conflicting
// method signatures between the two interfaces.
type KratosAdapter interface {
	// GetIdentity retrieves identity traits (email, name) from the Kratos Admin API.
	// Used by service layer when webhook payload is not sufficient.
	GetIdentity(ctx context.Context, identityID uuid.UUID) (*KratosIdentity, error)

	// DeleteIdentity deletes a Kratos identity. Used during family deletion (Phase 2).
	DeleteIdentity(ctx context.Context, identityID uuid.UUID) error

	// RevokeSessions revokes all active sessions for an identity.
	// Used when removing a co-parent (Phase 2).
	RevokeSessions(ctx context.Context, identityID uuid.UUID) error

	// ListSessionsForIdentity returns all active sessions for a Kratos identity.
	// Used by the lifecycle domain for session management. [15-data-lifecycle §12]
	ListSessionsForIdentity(ctx context.Context, identityID uuid.UUID) ([]KratosAdminSession, error)

	// RevokeSpecificSession revokes a single Kratos session by session ID.
	// Used by the lifecycle domain when a parent revokes a specific session. [15-data-lifecycle §12]
	RevokeSpecificSession(ctx context.Context, sessionID string) error

	// InitiateAccountRecovery sends a Kratos recovery email to the given address.
	// Email enumeration is prevented by the caller. [15-data-lifecycle §13]
	InitiateAccountRecovery(ctx context.Context, email string) error
}

// ─── Cross-Domain Consumer Interfaces ────────────────────────────────────────

// BillingServiceForIam is the narrow billing capability consumed by iam::.
// Defined here per consumer-interface pattern [ARCH §4.3]. Implemented by a
// function adapter in cmd/server/main.go over billing.BillingService.
type BillingServiceForIam interface {
	// VerifyCreditCardMicroCharge charges $0.50 and immediately refunds it to verify
	// parental identity for COPPA credit-card consent. [§9.3, 10-billing §13]
	VerifyCreditCardMicroCharge(ctx context.Context, scope *shared.FamilyScope, paymentMethodID string) error
}
