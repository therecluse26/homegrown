package iam

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ─── Family Repository ────────────────────────────────────────────────────────

// PgFamilyRepository implements FamilyRepository using PostgreSQL via GORM.
type PgFamilyRepository struct {
	db *gorm.DB
}

// NewPgFamilyRepository creates a new PgFamilyRepository.
// Pass a *gorm.DB transaction to create a tx-scoped instance inside transactions.
func NewPgFamilyRepository(db *gorm.DB) *PgFamilyRepository {
	return &PgFamilyRepository{db: db}
}

func (r *PgFamilyRepository) Create(ctx context.Context, cmd CreateFamily) (*Family, error) {
	model := &FamilyModel{
		DisplayName:               cmd.DisplayName,
		PrimaryMethodologySlug:    cmd.PrimaryMethodologySlug,
		SecondaryMethodologySlugs: SlugArray{},
		SubscriptionTier:          "free",
		CoppaConsentStatus:        string(CoppaConsentRegistered),
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgFamilyRepository) FindByID(ctx context.Context, id uuid.UUID) (*Family, error) {
	// NOT family-scoped — used by auth middleware and registration webhooks
	// before FamilyScope is constructed. Caller MUST ensure RLS is handled. [§6]
	var model FamilyModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFamilyNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgFamilyRepository) Update(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamily) (*Family, error) {
	updates := make(map[string]interface{})
	if cmd.DisplayName != nil {
		updates["display_name"] = *cmd.DisplayName
	}
	if cmd.StateCode != nil {
		updates["state_code"] = *cmd.StateCode
	}
	if cmd.LocationRegion != nil {
		updates["location_region"] = *cmd.LocationRegion
	}
	if len(updates) == 0 {
		return r.FindByID(ctx, scope.FamilyID())
	}
	updates["updated_at"] = time.Now()

	if err := r.db.WithContext(ctx).Model(&FamilyModel{}).
		Where("id = ?", scope.FamilyID()).
		Updates(updates).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return r.FindByID(ctx, scope.FamilyID())
}

func (r *PgFamilyRepository) SetPrimaryParent(ctx context.Context, familyID uuid.UUID, parentID uuid.UUID) error {
	// NOT family-scoped — used during registration before FamilyScope exists.
	// Caller MUST ensure RLS is handled (BypassRLSTransaction). [§6]
	if err := r.db.WithContext(ctx).Model(&FamilyModel{}).
		Where("id = ?", familyID).
		Updates(map[string]interface{}{
			"primary_parent_id": parentID,
			"updated_at":        time.Now(),
		}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgFamilyRepository) UpdateConsentStatus(ctx context.Context, scope *shared.FamilyScope, status CoppaConsentStatus, method *string) (*Family, error) {
	updates := map[string]interface{}{
		"coppa_consent_status": string(status),
		"updated_at":           time.Now(),
	}
	if status == CoppaConsentConsented || status == CoppaConsentReVerified {
		now := time.Now()
		updates["coppa_consented_at"] = now
	}
	if method != nil {
		updates["coppa_consent_method"] = *method
	}

	if err := r.db.WithContext(ctx).Model(&FamilyModel{}).
		Where("id = ?", scope.FamilyID()).
		Updates(updates).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return r.FindByID(ctx, scope.FamilyID())
}

func (r *PgFamilyRepository) SetMethodology(ctx context.Context, scope *shared.FamilyScope, primarySlug string, secondarySlugs []string) error {
	arr := SlugArray(secondarySlugs)
	val, err := arr.Value()
	if err != nil {
		return shared.ErrDatabase(err)
	}
	if err := r.db.WithContext(ctx).Model(&FamilyModel{}).
		Where("id = ?", scope.FamilyID()).
		Updates(map[string]interface{}{
			"primary_methodology_slug":    primarySlug,
			"secondary_methodology_slugs": val,
			"updated_at":                  time.Now(),
		}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgFamilyRepository) SetDeletionRequested(ctx context.Context, scope *shared.FamilyScope, requestedAt *time.Time) error {
	if err := r.db.WithContext(ctx).Model(&FamilyModel{}).
		Where("id = ?", scope.FamilyID()).
		Updates(map[string]interface{}{
			"deletion_requested_at": requestedAt,
			"updated_at":            time.Now(),
		}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── Parent Repository ────────────────────────────────────────────────────────

// PgParentRepository implements ParentRepository using PostgreSQL via GORM.
type PgParentRepository struct {
	db *gorm.DB
}

// NewPgParentRepository creates a new PgParentRepository.
func NewPgParentRepository(db *gorm.DB) *PgParentRepository {
	return &PgParentRepository{db: db}
}

func (r *PgParentRepository) Create(ctx context.Context, cmd CreateParent) (*Parent, error) {
	// NOT family-scoped — used during registration. [§6]
	model := &ParentModel{
		FamilyID:         cmd.FamilyID,
		KratosIdentityID: cmd.IdentityID,
		DisplayName:      cmd.DisplayName,
		Email:            cmd.Email,
		IsPrimary:        cmd.IsPrimary,
		IsPlatformAdmin:  false,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgParentRepository) FindByKratosID(ctx context.Context, kratosIdentityID uuid.UUID) (*Parent, error) {
	// NOT family-scoped — used by auth middleware before FamilyScope is constructed. [§6]
	// Caller MUST ensure RLS is handled (BypassRLSTransaction). [01-iam §11.1]
	var model ParentModel
	err := r.db.WithContext(ctx).Where("kratos_identity_id = ?", kratosIdentityID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgParentRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, parentID uuid.UUID) (*Parent, error) {
	var model ParentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND family_id = ?", parentID, scope.FamilyID()).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgParentRepository) ListByFamily(ctx context.Context, scope *shared.FamilyScope) ([]Parent, error) {
	var models []ParentModel
	if err := r.db.WithContext(ctx).Where("family_id = ?", scope.FamilyID()).Find(&models).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	result := make([]Parent, len(models))
	for i, m := range models {
		result[i] = *m.toDomain()
	}
	return result, nil
}

func (r *PgParentRepository) Update(ctx context.Context, scope *shared.FamilyScope, parentID uuid.UUID, cmd UpdateParent) (*Parent, error) {
	updates := make(map[string]interface{})
	if cmd.DisplayName != nil {
		updates["display_name"] = *cmd.DisplayName
	}
	if cmd.Email != nil {
		updates["email"] = *cmd.Email
	}
	if len(updates) == 0 {
		return r.FindByID(ctx, scope, parentID)
	}
	updates["updated_at"] = time.Now()

	if err := r.db.WithContext(ctx).Model(&ParentModel{}).
		Where("id = ? AND family_id = ?", parentID, scope.FamilyID()).
		Updates(updates).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return r.FindByID(ctx, scope, parentID)
}

func (r *PgParentRepository) Delete(ctx context.Context, scope *shared.FamilyScope, parentID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Where("id = ? AND family_id = ?", parentID, scope.FamilyID()).
		Delete(&ParentModel{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgParentRepository) SetPrimary(ctx context.Context, scope *shared.FamilyScope, parentID uuid.UUID, isPrimary bool) error {
	if err := r.db.WithContext(ctx).Model(&ParentModel{}).
		Where("id = ? AND family_id = ?", parentID, scope.FamilyID()).
		Updates(map[string]interface{}{
			"is_primary": isPrimary,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── Student Repository ───────────────────────────────────────────────────────

// PgStudentRepository implements StudentRepository using PostgreSQL via GORM.
type PgStudentRepository struct {
	db *gorm.DB
}

// NewPgStudentRepository creates a new PgStudentRepository.
func NewPgStudentRepository(db *gorm.DB) *PgStudentRepository {
	return &PgStudentRepository{db: db}
}

func (r *PgStudentRepository) Create(ctx context.Context, scope *shared.FamilyScope, cmd CreateStudent) (*Student, error) {
	model := &StudentModel{
		FamilyID:                scope.FamilyID(),
		DisplayName:             cmd.DisplayName,
		BirthYear:               cmd.BirthYear,
		GradeLevel:              cmd.GradeLevel,
		MethodologyOverrideSlug: cmd.MethodologyOverrideSlug,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgStudentRepository) ListByFamily(ctx context.Context, scope *shared.FamilyScope) ([]Student, error) {
	var models []StudentModel
	if err := r.db.WithContext(ctx).Where("family_id = ?", scope.FamilyID()).Find(&models).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	result := make([]Student, len(models))
	for i, m := range models {
		result[i] = *m.toDomain()
	}
	return result, nil
}

func (r *PgStudentRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*Student, error) {
	var model StudentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND family_id = ?", studentID, scope.FamilyID()).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStudentNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgStudentRepository) Update(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd UpdateStudent) (*Student, error) {
	updates := make(map[string]interface{})
	if cmd.DisplayName != nil {
		updates["display_name"] = *cmd.DisplayName
	}
	if cmd.BirthYear != nil {
		updates["birth_year"] = *cmd.BirthYear
	}
	if cmd.GradeLevel != nil {
		updates["grade_level"] = *cmd.GradeLevel
	}
	if cmd.MethodologyOverrideSlug != nil {
		// **string: outer nil = don't change; non-nil pointing to nil = clear
		updates["methodology_override_slug"] = *cmd.MethodologyOverrideSlug
	}
	if len(updates) == 0 {
		return r.FindByID(ctx, scope, studentID)
	}
	updates["updated_at"] = time.Now()

	if err := r.db.WithContext(ctx).Model(&StudentModel{}).
		Where("id = ? AND family_id = ?", studentID, scope.FamilyID()).
		Updates(updates).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return r.FindByID(ctx, scope, studentID)
}

func (r *PgStudentRepository) Delete(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND family_id = ?", studentID, scope.FamilyID()).
		Delete(&StudentModel{})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrStudentNotFound
	}
	return nil
}

// ─── Co-Parent Invite Repository ─────────────────────────────────────────────

// PgCoParentInviteRepository implements CoParentInviteRepository using PostgreSQL via GORM.
type PgCoParentInviteRepository struct {
	db *gorm.DB
}

// NewPgCoParentInviteRepository creates a new PgCoParentInviteRepository.
func NewPgCoParentInviteRepository(db *gorm.DB) *PgCoParentInviteRepository {
	return &PgCoParentInviteRepository{db: db}
}

func (r *PgCoParentInviteRepository) Create(ctx context.Context, familyID uuid.UUID, email, tokenHash string, expiresAt time.Time) (*CoParentInvite, error) {
	model := &CoParentInviteModel{
		FamilyID:  familyID,
		Email:     email,
		TokenHash: tokenHash,
		Status:    "pending",
		ExpiresAt: expiresAt,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgCoParentInviteRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*CoParentInvite, error) {
	var model CoParentInviteModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND family_id = ?", id, scope.FamilyID()).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgCoParentInviteRepository) FindByToken(ctx context.Context, tokenHash string) (*CoParentInvite, error) {
	// NOT family-scoped — called in AcceptInvite before requester has a family scope.
	// Caller MUST run inside BypassRLSTransaction. [§6]
	var model CoParentInviteModel
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgCoParentInviteRepository) UpdateStatus(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, status string) error {
	if err := r.db.WithContext(ctx).Model(&CoParentInviteModel{}).
		Where("id = ? AND family_id = ?", id, scope.FamilyID()).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── Student Session Repository ───────────────────────────────────────────────

// PgStudentSessionRepository implements StudentSessionRepository using PostgreSQL via GORM.
type PgStudentSessionRepository struct {
	db *gorm.DB
}

// NewPgStudentSessionRepository creates a new PgStudentSessionRepository.
func NewPgStudentSessionRepository(db *gorm.DB) *PgStudentSessionRepository {
	return &PgStudentSessionRepository{db: db}
}

func (r *PgStudentSessionRepository) Create(ctx context.Context, scope *shared.FamilyScope, studentID, createdBy uuid.UUID, tokenHash string, expiresAt time.Time, permissions []string) (*StudentSession, error) {
	model := &StudentSessionModel{
		FamilyID:    scope.FamilyID(),
		StudentID:   studentID,
		CreatedBy:   createdBy,
		TokenHash:   tokenHash,
		IsActive:    true,
		ExpiresAt:   expiresAt,
		Permissions: SlugArray(permissions),
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgStudentSessionRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*StudentSession, error) {
	var model StudentSessionModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND family_id = ?", id, scope.FamilyID()).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStudentSessionNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgStudentSessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*StudentSession, error) {
	// NOT family-scoped — called in GetStudentSessionMe before family scope is available.
	// Caller MUST run inside BypassRLSTransaction. [§6]
	var model StudentSessionModel
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStudentSessionNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return model.toDomain(), nil
}

func (r *PgStudentSessionRepository) ListActiveByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]StudentSession, error) {
	var models []StudentSessionModel
	if err := r.db.WithContext(ctx).
		Where("family_id = ? AND student_id = ? AND is_active = true AND expires_at > NOW()", scope.FamilyID(), studentID).
		Find(&models).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	result := make([]StudentSession, len(models))
	for i, m := range models {
		result[i] = *m.toDomain()
	}
	return result, nil
}

func (r *PgStudentSessionRepository) Revoke(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&StudentSessionModel{}).
		Where("id = ? AND family_id = ?", id, scope.FamilyID()).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrStudentSessionNotFound
	}
	return nil
}

// generateToken creates a 32-byte cryptographically random token and returns
// the hex-encoded plaintext and its bcrypt hash. [§5, CODING §5.2]
func generateToken(randReader func([]byte) (int, error)) (plaintext, hash string, err error) {
	raw := make([]byte, 32)
	if _, err = randReader(raw); err != nil {
		return "", "", err
	}
	plaintext = hex.EncodeToString(raw)
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return plaintext, string(hashBytes), nil
}
