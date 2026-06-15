package learner_profile

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ─── Repository Implementation ───────────────────────────────────────────────

// PgProfileRepository implements ProfileRepository using GORM + PostgreSQL.
type PgProfileRepository struct {
	db *gorm.DB
}

// NewPgProfileRepository creates a new repository.
func NewPgProfileRepository(db *gorm.DB) *PgProfileRepository {
	return &PgProfileRepository{db: db}
}

// Upsert creates or replaces the learner profile for a student.
// Uses ON CONFLICT (student_id) DO UPDATE — retake overwrites all fields.
func (r *PgProfileRepository) Upsert(ctx context.Context, scope *shared.FamilyScope, profile *LearnerProfile) error {
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		result := tx.WithContext(ctx).Exec(`
			INSERT INTO learner_profiles (
				id, family_id, student_id,
				activity_format, session_length, motivation, solo_collaborative,
				structure, outdoor_kinesthetic,
				interests, answered_count, confidence, source, respondent,
				created_at, updated_at
			) VALUES (
				uuidv7(), ?, ?,
				?, ?, ?, ?,
				?, ?,
				?, ?, ?, ?, ?,
				now(), now()
			)
			ON CONFLICT (student_id) DO UPDATE SET
				activity_format     = EXCLUDED.activity_format,
				session_length      = EXCLUDED.session_length,
				motivation          = EXCLUDED.motivation,
				solo_collaborative  = EXCLUDED.solo_collaborative,
				structure           = EXCLUDED.structure,
				outdoor_kinesthetic = EXCLUDED.outdoor_kinesthetic,
				interests           = EXCLUDED.interests,
				answered_count      = EXCLUDED.answered_count,
				confidence          = EXCLUDED.confidence,
				source              = EXCLUDED.source,
				respondent          = EXCLUDED.respondent,
				updated_at          = now()`,
			profile.FamilyID, profile.StudentID,
			profile.ActivityFormat, profile.SessionLength, profile.Motivation, profile.SoloCollaborative,
			profile.Structure, profile.OutdoorKinesthetic,
			profile.Interests, profile.AnsweredCount, profile.Confidence, profile.Source, profile.Respondent,
		)
		if result.Error != nil {
			return fmt.Errorf("learner_profile: upsert profile: %w", result.Error)
		}
		// Reload to get the id / timestamps from DB
		return tx.WithContext(ctx).Where("student_id = ?", profile.StudentID).First(profile).Error
	})
}

// FindByStudent returns the profile for one student, or nil if none exists.
func (r *PgProfileRepository) FindByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*LearnerProfile, error) {
	var profile LearnerProfile
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.WithContext(ctx).Where("student_id = ?", studentID).First(&profile).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("learner_profile: find by student: %w", err)
	}
	return &profile, nil
}

// FindByFamily returns all profiles for a family.
func (r *PgProfileRepository) FindByFamily(ctx context.Context, scope *shared.FamilyScope) ([]LearnerProfile, error) {
	var profiles []LearnerProfile
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.WithContext(ctx).Where("family_id = ?", scope.FamilyID()).Find(&profiles).Error
	})
	if err != nil {
		return nil, fmt.Errorf("learner_profile: find by family: %w", err)
	}
	return profiles, nil
}

// DeleteByStudent removes a student's profile. Called on student deletion.
func (r *PgProfileRepository) DeleteByStudent(ctx context.Context, studentID uuid.UUID) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		Delete(&LearnerProfile{})
	if result.Error != nil {
		return 0, fmt.Errorf("learner_profile: delete by student: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// DeleteByFamily removes all profiles for a family. Called on family deletion.
func (r *PgProfileRepository) DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("family_id = ?", familyID.UUID).
		Delete(&LearnerProfile{})
	if result.Error != nil {
		return 0, fmt.Errorf("learner_profile: delete by family: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// Compile-time interface check.
var _ ProfileRepository = (*PgProfileRepository)(nil)
