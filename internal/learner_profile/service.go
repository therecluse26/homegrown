package learner_profile

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Service Implementation ──────────────────────────────────────────────────

type learnerProfileServiceImpl struct {
	profileRepo ProfileRepository
	iam         IamServiceForLearnerProfile
}

// NewLearnerProfileService creates a new LearnerProfileService.
func NewLearnerProfileService(
	profileRepo ProfileRepository,
	iam IamServiceForLearnerProfile,
) LearnerProfileService {
	return &learnerProfileServiceImpl{
		profileRepo: profileRepo,
		iam:         iam,
	}
}

// ─── SubmitProfile ───────────────────────────────────────────────────────────

func (s *learnerProfileServiceImpl) SubmitProfile(
	ctx context.Context,
	scope *shared.FamilyScope,
	studentID uuid.UUID,
	cmd SubmitProfileCommand,
) (*LearnerProfileResponse, error) {
	// Verify student belongs to this family (cross-family data protection).
	belongs, err := s.iam.StudentBelongsToFamily(ctx, studentID, scope.FamilyID())
	if err != nil {
		return nil, fmt.Errorf("learner_profile: verify student ownership: %w", err)
	}
	if !belongs {
		return nil, ErrStudentNotInFamily
	}

	// Compute dimension vector from answers.
	vec := ComputeVector(cmd.Answers)

	profile := &LearnerProfile{
		FamilyID:           scope.FamilyID(),
		StudentID:          studentID,
		ActivityFormat:     vec.ActivityFormat,
		SessionLength:      vec.SessionLength,
		Motivation:         vec.Motivation,
		SoloCollaborative:  vec.SoloCollaborative,
		Structure:          vec.Structure,
		OutdoorKinesthetic: vec.OutdoorKinesthetic,
		Interests:          StringSlice(cmd.Interests),
		AnsweredCount:      vec.AnsweredCount,
		Confidence:         vec.Confidence,
		Source:             "declared",
		Respondent:         cmd.Respondent,
	}

	if err := s.profileRepo.Upsert(ctx, scope, profile); err != nil {
		return nil, fmt.Errorf("learner_profile: upsert: %w", err)
	}

	// Fetch student display name for summary text.
	studentName, err := s.iam.GetStudentDisplayName(ctx, studentID)
	if err != nil {
		slog.Warn("learner_profile: could not get student name for summary", "student_id", studentID, "error", err)
		studentName = "Your child"
	}

	return toResponse(profile, studentName), nil
}

// ─── GetProfile ──────────────────────────────────────────────────────────────

func (s *learnerProfileServiceImpl) GetProfile(
	ctx context.Context,
	scope *shared.FamilyScope,
	studentID uuid.UUID,
) (*LearnerProfileResponse, error) {
	// Verify student ownership.
	belongs, err := s.iam.StudentBelongsToFamily(ctx, studentID, scope.FamilyID())
	if err != nil {
		return nil, fmt.Errorf("learner_profile: verify student ownership: %w", err)
	}
	if !belongs {
		return nil, ErrStudentNotInFamily
	}

	profile, err := s.profileRepo.FindByStudent(ctx, scope, studentID)
	if err != nil {
		return nil, fmt.Errorf("learner_profile: find profile: %w", err)
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}

	studentName, err := s.iam.GetStudentDisplayName(ctx, studentID)
	if err != nil {
		slog.Warn("learner_profile: could not get student name", "student_id", studentID, "error", err)
		studentName = "Your child"
	}

	return toResponse(profile, studentName), nil
}

// ─── GetStudentInterestsByFamily ─────────────────────────────────────────────

// GetStudentInterestsByFamily returns declared interests keyed by student ID.
// Used by the recs:: LearnerProfilePort adapter to seed cold-start signal.
// This performs a BypassRLS read — interests are not PII and are read cross-family
// by the background task only; no scope is available in that context.
func (s *learnerProfileServiceImpl) GetStudentInterestsByFamily(
	ctx context.Context,
	familyID shared.FamilyID,
) (map[uuid.UUID][]string, error) {
	// Build a scope for the family (NewFamilyScopeFromID pattern — background task context).
	scope := shared.NewFamilyScopeFromID(familyID.UUID)
	profiles, err := s.profileRepo.FindByFamily(ctx, &scope)
	if err != nil {
		return nil, fmt.Errorf("learner_profile: get interests by family: %w", err)
	}

	result := make(map[uuid.UUID][]string, len(profiles))
	for _, p := range profiles {
		result[p.StudentID] = []string(p.Interests)
	}
	return result, nil
}

// ─── Deletion Event Handlers ─────────────────────────────────────────────────

func (s *learnerProfileServiceImpl) HandleStudentDeletion(ctx context.Context, studentID uuid.UUID) error {
	deleted, err := s.profileRepo.DeleteByStudent(ctx, studentID)
	if err != nil {
		return fmt.Errorf("learner_profile: handle student deletion: %w", err)
	}
	slog.Info("learner_profile: student deletion cleanup", "student_id", studentID, "deleted", deleted)
	return nil
}

func (s *learnerProfileServiceImpl) HandleFamilyDeletion(ctx context.Context, familyID shared.FamilyID) error {
	deleted, err := s.profileRepo.DeleteByFamily(ctx, familyID)
	if err != nil {
		return fmt.Errorf("learner_profile: handle family deletion: %w", err)
	}
	slog.Info("learner_profile: family deletion cleanup", "family_id", familyID.UUID, "deleted", deleted)
	return nil
}

// Compile-time interface check.
var _ LearnerProfileService = (*learnerProfileServiceImpl)(nil)
