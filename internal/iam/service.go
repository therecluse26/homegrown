package iam

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
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
	kratosAdapter              KratosAdapter
	eventBus                   *shared.EventBus
	db                         *gorm.DB // for transaction management
	defaultMethodologyResolver DefaultMethodologyResolver
	billingSvc                 BillingServiceForIam // optional; nil before billing:: is wired
}

// NewIamService creates a new IamServiceImpl.
func NewIamService(
	familyRepo FamilyRepository,
	parentRepo ParentRepository,
	studentRepo StudentRepository,
	kratosAdapter KratosAdapter,
	eventBus *shared.EventBus,
	db *gorm.DB,
) *IamServiceImpl {
	return &IamServiceImpl{
		familyRepo:    familyRepo,
		parentRepo:    parentRepo,
		studentRepo:   studentRepo,
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

// getDefaultMethodologySlug returns the default methodology slug using the injected resolver,
// or the fallback slug if the resolver is not set.
func (s *IamServiceImpl) getDefaultMethodologySlug(ctx context.Context) string {
	if s.defaultMethodologyResolver != nil {
		slug, err := s.defaultMethodologyResolver(ctx)
		if err != nil {
			slog.Error("failed to resolve default methodology slug, using fallback", "error", err)
			return fallbackMethodologySlug
		}
		return slug
	}
	return fallbackMethodologySlug
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
	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		parentRepo := NewPgParentRepository(tx)
		parent, err := parentRepo.FindByKratosID(ctx, payload.IdentityID)
		if err != nil {
			if errors.Is(err, ErrParentNotFound) {
				// Orphaned Kratos identity — log warning, do not fail webhook.
				// This can happen if registration webhook failed previously.
				slog.Warn("post-login: parent not found for kratos identity",
					"identity_id", payload.IdentityID)
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
	return err
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
		if cmd.Method == "credit_card_verification" && s.billingSvc != nil {
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
