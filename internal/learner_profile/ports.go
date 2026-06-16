package learner_profile

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Consumer-Defined Cross-Domain Interfaces [18-learner-profile §9, MEMORY patterns]
// ═══════════════════════════════════════════════════════════════════════════════

// IamServiceForLearnerProfile is a consumer-defined interface for cross-domain
// reads from iam::. Wired in main.go via a function adapter over iam.Service.
// familyID uses uuid.UUID (not shared.FamilyID) because FamilyScope.FamilyID() returns uuid.UUID.
type IamServiceForLearnerProfile interface {
	StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error)
	GetStudentDisplayName(ctx context.Context, studentID uuid.UUID) (string, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interface [18-learner-profile §9]
// ═══════════════════════════════════════════════════════════════════════════════

// ProfileRepository defines the data-access boundary for learner_profiles.
type ProfileRepository interface {
	// Upsert creates or replaces the profile for a student (retake semantics).
	Upsert(ctx context.Context, scope *shared.FamilyScope, profile *LearnerProfile) error
	// FindByStudent returns the profile for a student, or nil if none exists.
	FindByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*LearnerProfile, error)
	// FindByFamily returns all profiles for a family, keyed by student ID.
	// Used by the recs LearnerProfilePort adapter.
	FindByFamily(ctx context.Context, scope *shared.FamilyScope) ([]LearnerProfile, error)
	// DeleteByStudent deletes the profile for a student (belt-and-suspenders on CASCADE).
	DeleteByStudent(ctx context.Context, studentID uuid.UUID) (int64, error)
	// DeleteByFamily deletes all profiles for a family (belt-and-suspenders on CASCADE).
	DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface [18-learner-profile §9]
// ═══════════════════════════════════════════════════════════════════════════════

// LearnerProfileService is the primary service interface for the learner profile domain.
type LearnerProfileService interface {
	// SubmitProfile processes a quiz submission and upserts the learner profile.
	SubmitProfile(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd SubmitProfileCommand) (*LearnerProfileResponse, error)

	// GetProfile returns the current learner profile for a student.
	// Returns ErrProfileNotFound if no profile exists yet.
	GetProfile(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*LearnerProfileResponse, error)

	// GetStudentInterestsByFamily returns declared interests keyed by student ID.
	// Used by the recs:: LearnerProfilePort adapter (cross-domain bridge).
	GetStudentInterestsByFamily(ctx context.Context, familyID shared.FamilyID) (map[uuid.UUID][]string, error)

	// HandleStudentDeletion cleans up learner profile data on student deletion.
	HandleStudentDeletion(ctx context.Context, studentID uuid.UUID) error

	// HandleFamilyDeletion cleans up all learner profile data on family deletion.
	HandleFamilyDeletion(ctx context.Context, familyID shared.FamilyID) error
}
