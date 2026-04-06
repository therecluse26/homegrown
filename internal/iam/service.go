package iam

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// DefaultMethodologyResolver is a function that returns the default methodology slug.
// Injected by main.go after method:: is wired. Before injection, falls back to the
// charlotte-mason slug (display_order=1 in seed data).
type DefaultMethodologyResolver func(ctx context.Context) (string, error)

// fallbackMethodologySlug is the last-resort default if the resolver is not yet set.
// Matches Charlotte Mason (display_order=1) from the seed migration.
const fallbackMethodologySlug = "charlotte-mason"

// IamServiceImpl implements IamService.
// Holds references to repositories, adapters, the event bus, and the raw DB for transactions.
type IamServiceImpl struct {
	familyRepo                 FamilyRepository
	parentRepo                 ParentRepository
	studentRepo                StudentRepository
	inviteRepo                 CoParentInviteRepository
	sessionRepo                StudentSessionRepository
	kratosAdapter              KratosAdapter
	eventBus                   *shared.EventBus
	db                         *gorm.DB // for transaction management
	defaultMethodologyResolver DefaultMethodologyResolver
	billingSvc                 BillingServiceForIam
}

// NewIamService creates a new IamServiceImpl.
func NewIamService(
	familyRepo FamilyRepository,
	parentRepo ParentRepository,
	studentRepo StudentRepository,
	inviteRepo CoParentInviteRepository,
	sessionRepo StudentSessionRepository,
	kratosAdapter KratosAdapter,
	eventBus *shared.EventBus,
	db *gorm.DB,
) *IamServiceImpl {
	return &IamServiceImpl{
		familyRepo:    familyRepo,
		parentRepo:    parentRepo,
		studentRepo:   studentRepo,
		inviteRepo:    inviteRepo,
		sessionRepo:   sessionRepo,
		kratosAdapter: kratosAdapter,
		eventBus:      eventBus,
		db:            db,
	}
}

// SetDefaultMethodologyResolver injects the methodology resolver after method:: is wired.
// Called from cmd/server/main.go. [02-method Gap 5b]
func (s *IamServiceImpl) SetDefaultMethodologyResolver(resolver DefaultMethodologyResolver) {
	s.defaultMethodologyResolver = resolver
}

// SetBillingService injects the billing adapter after billing:: is wired.
// Called from cmd/server/main.go. Required for COPPA credit-card micro-charge. [§9.3]
func (s *IamServiceImpl) SetBillingService(svc BillingServiceForIam) {
	s.billingSvc = svc
}

// getDefaultMethodologySlug returns the default methodology slug, falling back to a
// hardcoded slug on error.
func (s *IamServiceImpl) getDefaultMethodologySlug(ctx context.Context) string {
	slug, err := s.defaultMethodologyResolver(ctx)
	if err != nil {
		slog.Error("failed to resolve default methodology slug, using fallback", "error", err)
		return fallbackMethodologySlug
	}
	return slug
}

// ─── Queries ──────────────────────────────────────────────────────────────────

func (s *IamServiceImpl) GetCurrentUser(ctx context.Context, auth *shared.AuthContext) (*CurrentUserResponse, error) {
	// Most fields come from AuthContext (already read by auth middleware from DB).
	// Only need to fetch family display_name via scoped query.
	scope := shared.NewFamilyScopeFromAuth(auth)

	var family *Family
	err := shared.ScopedTransaction(ctx, s.db, scope, func(tx *gorm.DB) error {
		var err error
		family, err = NewPgFamilyRepository(tx).FindByID(ctx, auth.FamilyID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return &CurrentUserResponse{
		ParentID:           auth.ParentID,
		FamilyID:           auth.FamilyID,
		DisplayName:        auth.DisplayName,
		Email:              auth.Email,
		IsPrimaryParent:    auth.IsPrimaryParent,
		IsPlatformAdmin:    auth.IsPlatformAdmin,
		SubscriptionTier:   string(auth.SubscriptionTier),
		CoppaConsentStatus: auth.CoppaConsentStatus,
		FamilyDisplayName:  family.DisplayName,
	}, nil
}

func (s *IamServiceImpl) GetFamilyProfile(ctx context.Context, scope *shared.FamilyScope) (*FamilyProfileResponse, error) {
	var family *Family
	var parents []Parent
	var students []Student

	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		familyRepo := NewPgFamilyRepository(tx)
		parentRepo := NewPgParentRepository(tx)
		studentRepo := NewPgStudentRepository(tx)

		var err error
		family, err = familyRepo.FindByID(ctx, scope.FamilyID())
		if err != nil {
			return err
		}
		parents, err = parentRepo.ListByFamily(ctx, scope)
		if err != nil {
			return err
		}
		students, err = studentRepo.ListByFamily(ctx, scope)
		return err
	})
	if err != nil {
		return nil, err
	}

	return buildFamilyProfileResponse(family, parents, students), nil
}

func (s *IamServiceImpl) ListStudents(ctx context.Context, scope *shared.FamilyScope) ([]StudentResponse, error) {
	var students []Student
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		var err error
		students, err = NewPgStudentRepository(tx).ListByFamily(ctx, scope)
		return err
	})
	if err != nil {
		return nil, err
	}

	result := make([]StudentResponse, len(students))
	for i, st := range students {
		result[i] = toStudentResponse(&st)
	}
	return result, nil
}

func (s *IamServiceImpl) GetConsentStatus(ctx context.Context, scope *shared.FamilyScope) (*ConsentStatusResponse, error) {
	var family *Family
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		var err error
		family, err = NewPgFamilyRepository(tx).FindByID(ctx, scope.FamilyID())
		return err
	})
	if err != nil {
		return nil, err
	}

	return toConsentStatusResponse(family), nil
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func (s *IamServiceImpl) HandlePostRegistration(ctx context.Context, payload KratosWebhookPayload) error {
	// Creates family + parent atomically. RLS bypassed — family does not exist yet. [§10.1]
	var familyID, parentID uuid.UUID

	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		familyRepo := NewPgFamilyRepository(tx)
		parentRepo := NewPgParentRepository(tx)

		family, err := familyRepo.Create(ctx, CreateFamily{
			DisplayName:          displayNameFromTraits(payload.Traits),
			PrimaryMethodologySlug: s.getDefaultMethodologySlug(ctx),
		})
		if err != nil {
			return err
		}
		familyID = family.ID

		parent, err := parentRepo.Create(ctx, CreateParent{
			FamilyID:    family.ID,
			IdentityID:  payload.IdentityID,
			DisplayName: payload.Traits.Name,
			Email:       payload.Traits.Email,
			IsPrimary:   true,
		})
		if err != nil {
			return err
		}
		parentID = parent.ID

		return familyRepo.SetPrimaryParent(ctx, family.ID, parent.ID)
	})
	if err != nil {
		return err
	}

	// Publish event after commit. Handler errors are logged, not propagated. [shared.EventBus]
	if err := s.eventBus.Publish(ctx, FamilyCreated{FamilyID: familyID, ParentID: parentID}); err != nil {
		slog.Error("failed to publish FamilyCreated", "family_id", familyID, "error", err)
	}
	return nil
}

func (s *IamServiceImpl) HandlePostLogin(ctx context.Context, payload KratosWebhookPayload) error {
	// Syncs Kratos traits to local DB. RLS bypassed — no family scope in webhook context. [§4.3]
	var orphaned bool
	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		parentRepo := NewPgParentRepository(tx)
		parent, err := parentRepo.FindByKratosID(ctx, payload.IdentityID)
		if err != nil {
			if errors.Is(err, ErrParentNotFound) {
				// Orphaned Kratos identity — registration webhook previously failed.
				// Signal recovery after this transaction closes (avoids nested transactions).
				orphaned = true
				return nil
			}
			return err
		}

		name := payload.Traits.Name
		email := payload.Traits.Email
		scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: parent.FamilyID})
		_, err = NewPgParentRepository(tx).Update(ctx, &scope, parent.ID, UpdateParent{
			DisplayName: &name,
			Email:       &email,
		})
		return err
	})
	if err != nil {
		return err
	}
	if orphaned {
		slog.Warn("post-login: orphaned identity detected, attempting recovery",
			"identity_id", payload.IdentityID)
		return s.HandlePostRegistration(ctx, payload)
	}
	return nil
}

func (s *IamServiceImpl) UpdateFamilyProfile(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamilyCommand) (*FamilyProfileResponse, error) {
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		_, err := NewPgFamilyRepository(tx).Update(ctx, scope, UpdateFamily(cmd))
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.GetFamilyProfile(ctx, scope)
}

func (s *IamServiceImpl) CreateStudent(ctx context.Context, scope *shared.FamilyScope, cmd CreateStudentCommand) (*StudentResponse, error) {
	var student *Student
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		var err error
		student, err = NewPgStudentRepository(tx).Create(ctx, scope, CreateStudent(cmd))
		return err
	})
	if err != nil {
		return nil, err
	}

	if err := s.eventBus.Publish(ctx, StudentCreated{FamilyID: scope.FamilyID(), StudentID: student.ID}); err != nil {
		slog.Error("failed to publish StudentCreated", "student_id", student.ID, "error", err)
	}
	result := toStudentResponse(student)
	return &result, nil
}

func (s *IamServiceImpl) UpdateStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd UpdateStudentCommand) (*StudentResponse, error) {
	var student *Student
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		var err error
		student, err = NewPgStudentRepository(tx).Update(ctx, scope, studentID, UpdateStudent(cmd))
		return err
	})
	if err != nil {
		return nil, err
	}
	result := toStudentResponse(student)
	return &result, nil
}

func (s *IamServiceImpl) DeleteStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error {
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		return NewPgStudentRepository(tx).Delete(ctx, scope, studentID)
	})
	if err != nil {
		return err
	}

	if err := s.eventBus.Publish(ctx, StudentDeleted{FamilyID: scope.FamilyID(), StudentID: studentID}); err != nil {
		slog.Error("failed to publish StudentDeleted", "student_id", studentID, "error", err)
	}
	return nil
}

func (s *IamServiceImpl) SubmitCoppaConsent(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, cmd CoppaConsentCommand) (*ConsentStatusResponse, error) {
	currentStatus := CoppaConsentStatus(auth.CoppaConsentStatus)
	newStatus, action, err := resolveConsentTransition(currentStatus, cmd)
	if err != nil {
		return nil, err
	}

	// Verify consent token. For credit_card_verification, run Hyperswitch micro-charge.
	// For other methods (e.g. knowledge_questions), token presence is sufficient for now. [§9.3]
	if newStatus == CoppaConsentConsented || newStatus == CoppaConsentReVerified {
		if cmd.Method == "" || cmd.VerificationToken == "" {
			return nil, ErrConsentVerificationFailed
		}
		if cmd.Method == "credit_card_verification" {
			if err := s.billingSvc.VerifyCreditCardMicroCharge(ctx, scope, cmd.VerificationToken); err != nil {
				slog.Error("iam: COPPA credit card micro-charge failed", "family_id", scope.FamilyID(), "error", err)
				return nil, ErrConsentVerificationFailed
			}
		}
	}

	var family *Family
	err = shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		var txErr error
		family, txErr = NewPgFamilyRepository(tx).UpdateConsentStatus(ctx, scope, newStatus, &cmd.Method)
		if txErr != nil {
			return txErr
		}

		// Create audit log entry atomically with status update. [§9.2]
		return tx.Create(&CoppaAuditLogModel{
			FamilyID:       scope.FamilyID(),
			Action:         action,
			Method:         &cmd.Method,
			PreviousStatus: string(currentStatus),
			NewStatus:      string(newStatus),
			PerformedBy:    auth.ParentID,
		}).Error
	})
	if err != nil {
		return nil, err
	}

	if newStatus == CoppaConsentConsented || newStatus == CoppaConsentReVerified {
		if pubErr := s.eventBus.Publish(ctx, CoppaConsentGranted{FamilyID: scope.FamilyID()}); pubErr != nil {
			slog.Error("failed to publish CoppaConsentGranted", "family_id", scope.FamilyID(), "error", pubErr)
		}
	}

	return toConsentStatusResponse(family), nil
}

// ─── Cross-Domain Methods (consumed by method::) ─────────────────────────────

func (s *IamServiceImpl) GetFamilyMethodologyIDs(ctx context.Context, scope *shared.FamilyScope) (string, []string, error) {
	var family *Family
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		var err error
		family, err = NewPgFamilyRepository(tx).FindByID(ctx, scope.FamilyID())
		return err
	})
	if err != nil {
		return "", nil, err
	}
	secondary := family.SecondaryMethodologySlugs
	if secondary == nil {
		secondary = []string{}
	}
	return family.PrimaryMethodologySlug, secondary, nil
}

func (s *IamServiceImpl) GetStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*StudentResponse, error) {
	var student *Student
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		var err error
		student, err = NewPgStudentRepository(tx).FindByID(ctx, scope, studentID)
		return err
	})
	if err != nil {
		return nil, err
	}
	result := toStudentResponse(student)
	return &result, nil
}

func (s *IamServiceImpl) SetFamilyMethodology(ctx context.Context, scope *shared.FamilyScope, primarySlug string, secondarySlugs []string) error {
	return shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		return NewPgFamilyRepository(tx).SetMethodology(ctx, scope, primarySlug, secondarySlugs)
	})
}

// ─── COPPA State Machine ──────────────────────────────────────────────────────

// resolveConsentTransition determines the target COPPA status from the current status and command.
// Returns InvalidConsentTransitionError for disallowed transitions. [§9.2]
func resolveConsentTransition(current CoppaConsentStatus, cmd CoppaConsentCommand) (CoppaConsentStatus, string, error) {
	// Determine whether this is "noticed" or "consented" based on the command payload.
	// If CoppaNoticeAcknowledged is true and no verification token → "noticed"
	// If CoppaNoticeAcknowledged is true and verification token provided → "consented"
	if !cmd.CoppaNoticeAcknowledged {
		return "", "", &InvalidConsentTransitionError{From: string(current), To: "?"}
	}

	switch current {
	case CoppaConsentRegistered:
		if cmd.VerificationToken != "" {
			// Combined flow: acknowledge + consent [§9.2]
			return CoppaConsentConsented, "consent_granted", nil
		}
		return CoppaConsentNoticed, "notice_acknowledged", nil

	case CoppaConsentNoticed:
		if cmd.VerificationToken != "" {
			return CoppaConsentConsented, "consent_granted", nil
		}
		// Already noticed, can't re-notice
		return "", "", &InvalidConsentTransitionError{From: string(current), To: string(CoppaConsentNoticed)}

	case CoppaConsentConsented, CoppaConsentReVerified:
		if cmd.VerificationToken != "" {
			return CoppaConsentReVerified, "consent_reverified", nil
		}
		return "", "", &InvalidConsentTransitionError{From: string(current), To: "?"}

	case CoppaConsentWithdrawn:
		// Cannot re-consent from withdrawn — must create a new account. [§9.2]
		return "", "", &InvalidConsentTransitionError{From: string(current), To: string(CoppaConsentConsented)}

	default:
		return "", "", &InvalidConsentTransitionError{From: string(current), To: "?"}
	}
}

// ─── Response Builders ────────────────────────────────────────────────────────

func buildFamilyProfileResponse(family *Family, parents []Parent, students []Student) *FamilyProfileResponse {
	parentSummaries := make([]ParentSummary, len(parents))
	for i, p := range parents {
		parentSummaries[i] = ParentSummary{
			ID:          p.ID,
			DisplayName: p.DisplayName,
			IsPrimary:   p.IsPrimary,
		}
	}
	secondary := family.SecondaryMethodologySlugs
	if secondary == nil {
		secondary = []string{}
	}
	return &FamilyProfileResponse{
		ID:                       family.ID,
		DisplayName:              family.DisplayName,
		StateCode:                family.StateCode,
		LocationRegion:           family.LocationRegion,
		PrimaryMethodologySlug:   family.PrimaryMethodologySlug,
		SecondaryMethodologySlugs: secondary,
		SubscriptionTier:         family.SubscriptionTier,
		CoppaConsentStatus:       string(family.CoppaConsentStatus),
		Parents:                  parentSummaries,
		StudentCount:             len(students),
		CreatedAt:                family.CreatedAt,
	}
}

func toStudentResponse(s *Student) StudentResponse {
	return StudentResponse{
		ID:                      s.ID,
		DisplayName:             s.DisplayName,
		BirthYear:               s.BirthYear,
		GradeLevel:              s.GradeLevel,
		MethodologyOverrideSlug: s.MethodologyOverrideSlug,
		CreatedAt:               s.CreatedAt,
		UpdatedAt:               s.UpdatedAt,
	}
}

func toConsentStatusResponse(family *Family) *ConsentStatusResponse {
	return &ConsentStatusResponse{
		Status:            string(family.CoppaConsentStatus),
		ConsentedAt:       family.CoppaConsentedAt,
		ConsentMethod:     family.CoppaConsentMethod,
		CanCreateStudents: family.CoppaConsentStatus.CanCreateStudents(),
	}
}

// RevokeFamilySessions revokes all Kratos sessions for every parent in the family.
// Uses BypassRLSTransaction to list parent identity IDs without a family scope
// (background job and cross-domain context). [15-data-lifecycle §12, 11-safety §7.3]
func (s *IamServiceImpl) RevokeFamilySessions(ctx context.Context, familyID uuid.UUID) error {
	var identityIDs []uuid.UUID
	if err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		return tx.Table("iam_parents").
			Select("kratos_identity_id").
			Where("family_id = ?", familyID).
			Scan(&identityIDs).Error
	}); err != nil {
		return fmt.Errorf("iam: list family parent identities: %w", err)
	}

	var lastErr error
	for _, identityID := range identityIDs {
		if err := s.kratosAdapter.RevokeSessions(ctx, identityID); err != nil {
			slog.Error("iam: revoke sessions for identity", "identity_id", identityID, "error", err)
			lastErr = err
		}
	}
	return lastErr
}

// GetStudentName returns the display_name for a student by ID.
// Uses BypassRLSTransaction — called by background jobs without a family scope.
func (s *IamServiceImpl) GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error) {
	type nameRow struct {
		DisplayName string `gorm:"column:display_name"`
	}
	var row nameRow
	if err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		return tx.Raw(`SELECT display_name FROM iam_students WHERE id = ?`, studentID).Scan(&row).Error
	}); err != nil {
		return "", fmt.Errorf("iam: get student name: %w", err)
	}
	return row.DisplayName, nil
}

func displayNameFromTraits(traits KratosTraits) string {
	if traits.Name != "" {
		return traits.Name + " Family"
	}
	return "My Family"
}

// ─── Phase 2: Co-parent Management ───────────────────────────────────────────

func (s *IamServiceImpl) InviteCoParent(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, cmd InviteCoParentCommand) (*CoParentInviteResponse, error) {
	if !auth.IsPrimaryParent {
		return nil, ErrNotPrimaryParent
	}

	plaintext, tokenHash, err := generateToken(rand.Read)
	if err != nil {
		return nil, fmt.Errorf("iam: generate invite token: %w", err)
	}

	expiresAt := time.Now().Add(72 * time.Hour)

	var invite *CoParentInvite
	err = shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		var txErr error
		invite, txErr = NewPgCoParentInviteRepository(tx).Create(ctx, scope.FamilyID(), auth.ParentID, cmd.Email, tokenHash, expiresAt)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	// Notify domain sends the invite email via InviteCreated event.
	// Token included in event so notify can build the accept URL. [§5]
	if pubErr := s.eventBus.Publish(ctx, InviteCreated{
		FamilyID:  scope.FamilyID(),
		InviteID:  invite.ID,
		Email:     cmd.Email,
		Token:     plaintext, // plaintext — notify builds link; not stored in DB [CODING §5.2]
		ExpiresAt: expiresAt,
	}); pubErr != nil {
		slog.Error("failed to publish InviteCreated", "invite_id", invite.ID, "error", pubErr)
	}

	return &CoParentInviteResponse{
		ID:        invite.ID,
		Email:     invite.Email,
		Status:    invite.Status,
		ExpiresAt: invite.ExpiresAt,
		CreatedAt: invite.CreatedAt,
	}, nil
}

func (s *IamServiceImpl) CancelInvite(ctx context.Context, scope *shared.FamilyScope, inviteID uuid.UUID) error {
	return shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := NewPgCoParentInviteRepository(tx)
		invite, err := repo.FindByID(ctx, scope, inviteID)
		if err != nil {
			return err
		}
		if invite.Status != "pending" {
			return ErrInviteAlreadyAccepted
		}
		return repo.UpdateStatus(ctx, scope, inviteID, "cancelled")
	})
}

func (s *IamServiceImpl) AcceptInvite(ctx context.Context, auth *shared.AuthContext, token string) error {
	// Hash the token to look up the invite row.
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("iam: hash accept token: %w", err)
	}
	tokenHash := string(hashBytes)
	_ = tokenHash // We actually need to find by the stored hash — use bcrypt.CompareHashAndPassword below.

	// BypassRLS: requester is not yet a member of the invite's family. [§6]
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		inviteRepo := NewPgCoParentInviteRepository(tx)
		parentRepo := NewPgParentRepository(tx)

		// Scan all pending, non-expired invites and compare with bcrypt.
		// This is a linear scan over the invite table — acceptable for low-volume operation.
		var models []CoParentInviteModel
		if err := tx.WithContext(ctx).
			Where("status = 'pending' AND expires_at > NOW()").
			Find(&models).Error; err != nil {
			return shared.ErrDatabase(err)
		}

		// Compare all entries to avoid leaking match position via timing. [P3-6]
		var matched *CoParentInviteModel
		for i := range models {
			if err := bcrypt.CompareHashAndPassword([]byte(models[i].TokenHash), []byte(token)); err == nil {
				matched = &models[i]
				// Continue iterating all entries — constant-time over the set.
			}
		}
		if matched == nil {
			return ErrInviteNotFound
		}

		// Check invite is still valid.
		if matched.Status != "pending" {
			return ErrInviteAlreadyAccepted
		}
		if time.Now().After(matched.ExpiresAt) {
			return ErrInviteExpired
		}

		// Check requester is not already in the family.
		var count int64
		if err := tx.WithContext(ctx).Model(&ParentModel{}).
			Where("family_id = ? AND kratos_identity_id = ?", matched.FamilyID, auth.IdentityID).
			Count(&count).Error; err != nil {
			return shared.ErrDatabase(err)
		}
		if count > 0 {
			return ErrParentAlreadyInFamily
		}

		// Create parent row.
		scope := shared.NewFamilyScopeFromID(matched.FamilyID)
		if _, err := parentRepo.Create(ctx, CreateParent{
			FamilyID:    matched.FamilyID,
			IdentityID:  auth.IdentityID,
			DisplayName: auth.DisplayName,
			Email:       auth.Email,
			IsPrimary:   false,
		}); err != nil {
			return err
		}

		// Mark invite accepted.
		if err := inviteRepo.UpdateStatus(ctx, &scope, matched.ID, "accepted"); err != nil {
			return err
		}

		// Publish event after all DB work succeeds.
		// Event is published inside the transaction closure; bus.Publish logs on error. [shared.EventBus]
		if pubErr := s.eventBus.Publish(ctx, CoParentAdded{
			FamilyID:     matched.FamilyID,
			CoParentID:   auth.ParentID,
			CoParentEmail: auth.Email,
			CoParentName:  auth.DisplayName,
		}); pubErr != nil {
			slog.Error("failed to publish CoParentAdded", "family_id", matched.FamilyID, "error", pubErr)
		}
		return nil
	})
}

func (s *IamServiceImpl) RemoveCoParent(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, parentID uuid.UUID) error {
	if !auth.IsPrimaryParent {
		return ErrNotPrimaryParent
	}

	var identityID uuid.UUID
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := NewPgParentRepository(tx)
		parent, err := repo.FindByID(ctx, scope, parentID)
		if err != nil {
			return err
		}
		if parent.IsPrimary {
			return ErrCannotRemovePrimaryParent
		}
		identityID = parent.IdentityID
		return repo.Delete(ctx, scope, parentID)
	})
	if err != nil {
		return err
	}

	// Revoke Kratos sessions after DB delete commits.
	if err := s.kratosAdapter.RevokeSessions(ctx, identityID); err != nil {
		slog.Error("iam: revoke sessions on co-parent removal", "identity_id", identityID, "error", err)
	}

	if pubErr := s.eventBus.Publish(ctx, CoParentRemoved{FamilyID: scope.FamilyID(), CoParentID: parentID}); pubErr != nil {
		slog.Error("failed to publish CoParentRemoved", "family_id", scope.FamilyID(), "error", pubErr)
	}
	return nil
}

func (s *IamServiceImpl) TransferPrimaryParent(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, cmd TransferPrimaryCommand) error {
	if !auth.IsPrimaryParent {
		return ErrNotPrimaryParent
	}
	if cmd.NewPrimaryParentID == auth.ParentID {
		return ErrCannotTransferToSelf
	}

	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := NewPgParentRepository(tx)
		// Verify target parent is in the family.
		if _, err := repo.FindByID(ctx, scope, cmd.NewPrimaryParentID); err != nil {
			return err
		}
		// Atomic swap: clear current primary, set new primary.
		if err := repo.SetPrimary(ctx, scope, auth.ParentID, false); err != nil {
			return err
		}
		return repo.SetPrimary(ctx, scope, cmd.NewPrimaryParentID, true)
	})
	if err != nil {
		return err
	}

	if pubErr := s.eventBus.Publish(ctx, PrimaryParentTransferred{
		FamilyID:      scope.FamilyID(),
		NewPrimaryID:  cmd.NewPrimaryParentID,
		PrevPrimaryID: auth.ParentID,
	}); pubErr != nil {
		slog.Error("failed to publish PrimaryParentTransferred", "family_id", scope.FamilyID(), "error", pubErr)
	}
	return nil
}

// ─── Phase 2: COPPA / Family Lifecycle ───────────────────────────────────────

func (s *IamServiceImpl) WithdrawCoppaConsent(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext) error {
	if !auth.IsPrimaryParent {
		return ErrNotPrimaryParent
	}
	current := CoppaConsentStatus(auth.CoppaConsentStatus)
	if current != CoppaConsentConsented && current != CoppaConsentReVerified {
		return &InvalidConsentTransitionError{From: string(current), To: string(CoppaConsentWithdrawn)}
	}

	return shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		familyRepo := NewPgFamilyRepository(tx)
		method := "withdrawn_by_parent"
		if _, err := familyRepo.UpdateConsentStatus(ctx, scope, CoppaConsentWithdrawn, &method); err != nil {
			return err
		}
		return tx.Create(&CoppaAuditLogModel{
			FamilyID:       scope.FamilyID(),
			Action:         "consent_withdrawn",
			Method:         &method,
			PreviousStatus: string(current),
			NewStatus:      string(CoppaConsentWithdrawn),
			PerformedBy:    auth.ParentID,
		}).Error
	})
}

func (s *IamServiceImpl) RequestFamilyDeletion(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext) error {
	if !auth.IsPrimaryParent {
		return ErrNotPrimaryParent
	}

	var deleteAfter time.Time
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		family, err := NewPgFamilyRepository(tx).FindByID(ctx, scope.FamilyID())
		if err != nil {
			return err
		}
		if family.DeletionRequestedAt != nil {
			return ErrDeletionAlreadyRequested
		}
		now := time.Now()
		deleteAfter = now.Add(30 * 24 * time.Hour)
		return NewPgFamilyRepository(tx).SetDeletionRequested(ctx, scope, &now)
	})
	if err != nil {
		return err
	}

	if pubErr := s.eventBus.Publish(ctx, FamilyDeletionScheduled{
		FamilyID:    scope.FamilyID(),
		DeleteAfter: deleteAfter,
	}); pubErr != nil {
		slog.Error("failed to publish FamilyDeletionScheduled", "family_id", scope.FamilyID(), "error", pubErr)
	}
	return nil
}

func (s *IamServiceImpl) CancelFamilyDeletion(ctx context.Context, scope *shared.FamilyScope) error {
	return shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		family, err := NewPgFamilyRepository(tx).FindByID(ctx, scope.FamilyID())
		if err != nil {
			return err
		}
		if family.DeletionRequestedAt == nil {
			return ErrNoPendingDeletion
		}
		return NewPgFamilyRepository(tx).SetDeletionRequested(ctx, scope, nil)
	})
}

// ─── Phase 2: Student Sessions ────────────────────────────────────────────────

func (s *IamServiceImpl) CreateStudentSession(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, studentID uuid.UUID, cmd CreateStudentSessionCommand) (*StudentSessionResponse, error) {
	// Verify student belongs to this family.
	if _, err := s.GetStudent(ctx, scope, studentID); err != nil {
		return nil, err
	}

	plaintext, tokenHash, err := generateToken(rand.Read)
	if err != nil {
		return nil, fmt.Errorf("iam: generate session token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(cmd.ExpiresInHours) * time.Hour)

	var session *StudentSession
	err = shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		var txErr error
		session, txErr = NewPgStudentSessionRepository(tx).Create(ctx, scope, studentID, auth.ParentID, tokenHash, expiresAt, cmd.AllowedToolSlugs)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return &StudentSessionResponse{
		ID:          session.ID,
		StudentID:   session.StudentID,
		Token:       plaintext, // returned once only [CODING §5.2]
		ExpiresAt:   session.ExpiresAt,
		Permissions: session.Permissions,
		CreatedAt:   session.CreatedAt,
	}, nil
}

func (s *IamServiceImpl) ListStudentSessions(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]StudentSessionSummaryResponse, error) {
	var sessions []StudentSession
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		var txErr error
		sessions, txErr = NewPgStudentSessionRepository(tx).ListActiveByStudent(ctx, scope, studentID)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	result := make([]StudentSessionSummaryResponse, len(sessions))
	for i, ss := range sessions {
		result[i] = StudentSessionSummaryResponse{
			ID:          ss.ID,
			StudentID:   ss.StudentID,
			ExpiresAt:   ss.ExpiresAt,
			IsActive:    ss.IsActive,
			Permissions: ss.Permissions,
			CreatedAt:   ss.CreatedAt,
		}
	}
	return result, nil
}

func (s *IamServiceImpl) RevokeStudentSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID) error {
	return shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := NewPgStudentSessionRepository(tx)
		session, err := repo.FindByID(ctx, scope, sessionID)
		if err != nil {
			return err
		}
		// Ensure session belongs to the specified student.
		if session.StudentID != studentID {
			return ErrStudentSessionNotFound
		}
		return repo.Revoke(ctx, scope, sessionID)
	})
}

func (s *IamServiceImpl) GetStudentSessionMe(ctx context.Context, token string) (*StudentSessionIdentityResponse, error) {
	// BypassRLS: no family scope available — auth via student bearer token. [§6]
	var result *StudentSessionIdentityResponse
	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		repo := NewPgStudentSessionRepository(tx)
		// Scan active, non-expired sessions and compare token with bcrypt.
		var models []StudentSessionModel
		if err := tx.WithContext(ctx).
			Where("is_active = true AND expires_at > NOW()").
			Find(&models).Error; err != nil {
			return shared.ErrDatabase(err)
		}

		// Compare all entries to avoid leaking match position via timing. [P3-6]
		var matched *StudentSessionModel
		for i := range models {
			if err := bcrypt.CompareHashAndPassword([]byte(models[i].TokenHash), []byte(token)); err == nil {
				matched = &models[i]
				// Continue iterating all entries — constant-time over the set.
			}
		}
		if matched == nil {
			return ErrStudentSessionExpired
		}
		_ = repo

		var perms []string
		if matched.Permissions != nil {
			perms = []string(matched.Permissions)
		}
		result = &StudentSessionIdentityResponse{
			StudentID:        matched.StudentID,
			FamilyID:         matched.FamilyID,
			AllowedToolSlugs: perms,
			ExpiresAt:        matched.ExpiresAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ─── HandleFamilyDeletionScheduled ────────────────────────────────────────────

// HandleFamilyDeletionScheduled deletes all IAM data for a family.
// Must run LAST among deletion handlers because other domains' tables reference IAM records.
// RETAINS: iam_coppa_audit_log (legal compliance requirement). [15-data-lifecycle §7]
func (s *IamServiceImpl) HandleFamilyDeletionScheduled(ctx context.Context, familyID uuid.UUID) error {
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		// Delete children-first to respect FK ordering.

		// 1. Student sessions (references iam_students).
		studentIDSubquery := tx.Model(&StudentModel{}).Select("id").Where("family_id = ?", familyID)
		if err := tx.Where("student_id IN (?)", studentIDSubquery).Delete(&StudentSessionModel{}).Error; err != nil {
			return fmt.Errorf("iam: delete student_sessions: %w", err)
		}

		// 2. Co-parent invites.
		if err := tx.Where("family_id = ?", familyID).Delete(&CoParentInviteModel{}).Error; err != nil {
			return fmt.Errorf("iam: delete co_parent_invites: %w", err)
		}

		// 3. Students (referenced by many domains — must be deleted after domain handlers).
		if err := tx.Where("family_id = ?", familyID).Delete(&StudentModel{}).Error; err != nil {
			return fmt.Errorf("iam: delete students: %w", err)
		}

		// 4. Parents (referenced by many domains — must be deleted after domain handlers).
		if err := tx.Where("family_id = ?", familyID).Delete(&ParentModel{}).Error; err != nil {
			return fmt.Errorf("iam: delete parents: %w", err)
		}

		// 5. Family record itself.
		if err := tx.Where("id = ?", familyID).Delete(&FamilyModel{}).Error; err != nil {
			return fmt.Errorf("iam: delete family: %w", err)
		}

		// NOTE: iam_coppa_audit_log RETAINED per legal compliance requirement.
		return nil
	})
}
