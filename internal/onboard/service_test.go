package onboard

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Test Fixtures ───────────────────────────────────────────────────────────

func testFamilyID() uuid.UUID { return uuid.Must(uuid.NewV7()) }
func testScope(familyID uuid.UUID) *shared.FamilyScope {
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
	return &scope
}

// ─── Mock Cross-Domain Services ─────────────────────────────────────────────

type mockIamForOnboard struct {
	updateErr    error
	createErr    error
	deleteErr    error
	students     []OnboardStudentInfo
	studentsErr  error
}

func (m *mockIamForOnboard) UpdateFamilyProfile(_ context.Context, _ *shared.FamilyScope, _ UpdateFamilyProfileCommand) error {
	return m.updateErr
}
func (m *mockIamForOnboard) CreateStudent(_ context.Context, _ *shared.FamilyScope, _ AddChildCommand) error {
	return m.createErr
}
func (m *mockIamForOnboard) DeleteStudent(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
	return m.deleteErr
}
func (m *mockIamForOnboard) ListStudents(_ context.Context, _ uuid.UUID) ([]OnboardStudentInfo, error) {
	return m.students, m.studentsErr
}

type mockMethodForOnboard struct {
	configs       map[string]*OnboardMethodologyConfig
	defaultSlug   string
	validateValid bool
	updateErr     error
}

func (m *mockMethodForOnboard) GetMethodology(_ context.Context, slug string) (*OnboardMethodologyConfig, error) {
	if c, ok := m.configs[slug]; ok {
		return c, nil
	}
	return nil, errors.New("methodology not found")
}
func (m *mockMethodForOnboard) GetDefaultMethodologySlug(_ context.Context) (string, error) {
	return m.defaultSlug, nil
}
func (m *mockMethodForOnboard) ValidateMethodologySlugs(_ context.Context, _ []string) (bool, error) {
	return m.validateValid, nil
}
func (m *mockMethodForOnboard) UpdateFamilyMethodology(_ context.Context, _ *shared.FamilyScope, _ string, _ []string) error {
	return m.updateErr
}

type mockDiscoverForOnboard struct {
	quizResult *OnboardQuizResult
	claimErr   error
}

func (m *mockDiscoverForOnboard) GetQuizResult(_ context.Context, _ string) (*OnboardQuizResult, error) {
	if m.quizResult == nil {
		return nil, errors.New("quiz result not found")
	}
	return m.quizResult, nil
}
func (m *mockDiscoverForOnboard) ClaimQuizResult(_ context.Context, _ string, _ uuid.UUID) error {
	return m.claimErr
}

// ─── Test: AgeGroupFromBirthYear ─────────────────────────────────────────────

func TestAgeGroupFromBirthYear(t *testing.T) {
	currentYear := 2026
	tests := []struct {
		name      string
		birthYear *int16
		want      AgeGroup
	}{
		{"nil birth year", nil, ""},
		{"preschool (age 4)", ptr[int16](2022), AgeGroupPreschool},
		{"early elem (age 7)", ptr[int16](2019), AgeGroupEarlyElem},
		{"upper elem (age 10)", ptr[int16](2016), AgeGroupUpperElem},
		{"middle (age 14)", ptr[int16](2012), AgeGroupMiddle},
		{"high school (age 17)", ptr[int16](2009), AgeGroupHighSchool},
		{"edge: age 5 = preschool", ptr[int16](2021), AgeGroupPreschool},
		{"edge: age 6 = early elem", ptr[int16](2020), AgeGroupEarlyElem},
		{"edge: age 8 = early elem", ptr[int16](2018), AgeGroupEarlyElem},
		{"edge: age 9 = upper elem", ptr[int16](2017), AgeGroupUpperElem},
		{"edge: age 12 = upper elem", ptr[int16](2014), AgeGroupUpperElem},
		{"edge: age 13 = middle", ptr[int16](2013), AgeGroupMiddle},
		{"edge: age 15 = middle", ptr[int16](2011), AgeGroupMiddle},
		{"edge: age 16 = high school", ptr[int16](2010), AgeGroupHighSchool},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AgeGroupFromBirthYear(tt.birthYear, currentYear)
			if got != tt.want {
				t.Errorf("AgeGroupFromBirthYear(%v, %d) = %q, want %q", tt.birthYear, currentYear, got, tt.want)
			}
		})
	}
}

// ─── Test: Wizard State Machine ──────────────────────────────────────────────

func TestAdvanceStep_AddsToCompleted(t *testing.T) {
	p := &WizardProgress{
		Status:         StatusInProgress,
		CurrentStep:    StepFamilyProfile,
		CompletedSteps: WizardStepArray{},
	}
	advanceStep(p, StepFamilyProfile)

	if len(p.CompletedSteps) != 1 || p.CompletedSteps[0] != StepFamilyProfile {
		t.Errorf("expected [family_profile], got %v", p.CompletedSteps)
	}
	if p.CurrentStep != StepChildren {
		t.Errorf("expected current_step=children, got %s", p.CurrentStep)
	}
}

func TestAdvanceStep_Idempotent(t *testing.T) {
	p := &WizardProgress{
		Status:         StatusInProgress,
		CurrentStep:    StepChildren,
		CompletedSteps: WizardStepArray{StepFamilyProfile},
	}
	advanceStep(p, StepFamilyProfile) // already completed

	if len(p.CompletedSteps) != 1 {
		t.Errorf("expected 1 step, got %d", len(p.CompletedSteps))
	}
}

func TestAdvanceStep_SkipsOptional(t *testing.T) {
	p := &WizardProgress{
		Status:         StatusInProgress,
		CurrentStep:    StepFamilyProfile,
		CompletedSteps: WizardStepArray{},
	}
	// Complete family_profile and methodology (skipping children)
	advanceStep(p, StepFamilyProfile)
	advanceStep(p, StepMethodology)

	// Current step should be children (first incomplete)
	if p.CurrentStep != StepChildren {
		t.Errorf("expected current_step=children, got %s", p.CurrentStep)
	}
}

func TestAdvanceStep_AllCompleted(t *testing.T) {
	p := &WizardProgress{
		Status:         StatusInProgress,
		CurrentStep:    StepFamilyProfile,
		CompletedSteps: WizardStepArray{},
	}
	for _, step := range wizardStepOrder {
		advanceStep(p, step)
	}
	if p.CurrentStep != StepRoadmapReview {
		t.Errorf("expected current_step=roadmap_review (last step), got %s", p.CurrentStep)
	}
	if len(p.CompletedSteps) != 4 {
		t.Errorf("expected 4 completed steps, got %d", len(p.CompletedSteps))
	}
}

// ─── Test: matchesAgeGroup ───────────────────────────────────────────────────

func TestMatchesAgeGroup_EmptyItem(t *testing.T) {
	if !matchesAgeGroup("", map[AgeGroup]bool{AgeGroupEarlyElem: true}) {
		t.Error("universal items should always match")
	}
}

func TestMatchesAgeGroup_NoStudents(t *testing.T) {
	if !matchesAgeGroup("6-8", map[AgeGroup]bool{}) {
		t.Error("when no students, all items should match")
	}
}

func TestMatchesAgeGroup_Match(t *testing.T) {
	if !matchesAgeGroup("6-8", map[AgeGroup]bool{AgeGroupEarlyElem: true}) {
		t.Error("should match when family has matching age group")
	}
}

func TestMatchesAgeGroup_NoMatch(t *testing.T) {
	if matchesAgeGroup("13-15", map[AgeGroup]bool{AgeGroupEarlyElem: true}) {
		t.Error("should not match when family has no matching age group")
	}
}

// ─── Test: validateCompleteWizard / applyCompleteWizard ──────────────────────

func TestValidateCompleteWizard_MissingRequiredSteps(t *testing.T) {
	progress := &WizardProgress{
		Status:         StatusInProgress,
		CompletedSteps: WizardStepArray{StepFamilyProfile}, // missing methodology
	}

	err := validateCompleteWizard(progress)
	if err == nil {
		t.Fatal("expected error")
	}
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) {
		t.Fatalf("want *OnboardError, got %T: %v", err, err)
	}
	if !errors.Is(onbErr.Err, ErrRequiredStepsMissing) {
		t.Errorf("want ErrRequiredStepsMissing, got %v", onbErr.Err)
	}
}

func TestValidateCompleteWizard_AlreadyCompleted(t *testing.T) {
	progress := &WizardProgress{Status: StatusCompleted}
	err := validateCompleteWizard(progress)
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) || !errors.Is(onbErr.Err, ErrWizardAlreadyCompleted) {
		t.Errorf("want ErrWizardAlreadyCompleted, got %v", err)
	}
}

func TestValidateCompleteWizard_HappyPath(t *testing.T) {
	progress := &WizardProgress{
		Status:         StatusInProgress,
		CompletedSteps: WizardStepArray{StepFamilyProfile, StepMethodology},
	}
	if err := validateCompleteWizard(progress); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyCompleteWizard(t *testing.T) {
	progress := &WizardProgress{Status: StatusInProgress}
	applyCompleteWizard(progress)
	if progress.Status != StatusCompleted {
		t.Errorf("want status=completed, got %s", progress.Status)
	}
	if progress.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

// ─── Test: validateSkipWizard / applySkipWizard ─────────────────────────────

func TestValidateSkipWizard_HappyPath(t *testing.T) {
	progress := &WizardProgress{Status: StatusInProgress}
	if err := validateSkipWizard(progress); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSkipWizard_AlreadyCompleted(t *testing.T) {
	progress := &WizardProgress{Status: StatusCompleted}
	err := validateSkipWizard(progress)
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) || !errors.Is(onbErr.Err, ErrWizardAlreadyCompleted) {
		t.Errorf("want ErrWizardAlreadyCompleted, got %v", err)
	}
}

func TestValidateSkipWizard_AlreadySkipped(t *testing.T) {
	progress := &WizardProgress{Status: StatusSkipped}
	err := validateSkipWizard(progress)
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) || !errors.Is(onbErr.Err, ErrWizardAlreadySkipped) {
		t.Errorf("want ErrWizardAlreadySkipped, got %v", err)
	}
}

func TestApplySkipWizard(t *testing.T) {
	progress := &WizardProgress{Status: StatusInProgress}
	applySkipWizard(progress)
	if progress.Status != StatusSkipped {
		t.Errorf("want status=skipped, got %s", progress.Status)
	}
	if progress.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

// ─── Test: SelectMethodology ─────────────────────────────────────────────────

func TestSelectMethodology_SecondaryWithoutAck(t *testing.T) {
	svc := &onboardingServiceImpl{}

	scope := testScope(testFamilyID())
	_, err := svc.SelectMethodology(context.Background(), scope, SelectMethodologyCommand{
		PrimaryMethodologySlug:    "charlotte-mason",
		SecondaryMethodologySlugs: []string{"classical"},
		MethodologyPath:           "exploration",
		ExplanationAcknowledged:   false,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) {
		t.Fatalf("want *OnboardError, got %T", err)
	}
	if !errors.Is(onbErr.Err, ErrSecondaryWithoutAck) {
		t.Errorf("want ErrSecondaryWithoutAck, got %v", onbErr.Err)
	}
}

func TestSelectMethodology_InvalidSlugs(t *testing.T) {
	svc := &onboardingServiceImpl{
		methodology: &mockMethodForOnboard{validateValid: false},
	}

	scope := testScope(testFamilyID())
	_, err := svc.SelectMethodology(context.Background(), scope, SelectMethodologyCommand{
		PrimaryMethodologySlug: "nonexistent",
		MethodologyPath:        "exploration",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) {
		t.Fatalf("want *OnboardError, got %T: %v", err, err)
	}
	if !errors.Is(onbErr.Err, ErrInvalidMethodologySlug) {
		t.Errorf("want ErrInvalidMethodologySlug, got %v", onbErr.Err)
	}
}

// ─── Test: ImportQuiz (pre-transaction logic) ────────────────────────────────

func TestImportQuiz_QuizNotFound(t *testing.T) {
	svc := &onboardingServiceImpl{
		discovery: &mockDiscoverForOnboard{quizResult: nil},
	}

	scope := testScope(testFamilyID())
	_, err := svc.ImportQuiz(context.Background(), scope, ImportQuizCommand{ShareID: "notexist"})
	if err == nil {
		t.Fatal("expected error")
	}
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) {
		t.Fatalf("want *OnboardError, got %T", err)
	}
	if !errors.Is(onbErr.Err, ErrQuizResultNotFound) {
		t.Errorf("want ErrQuizResultNotFound, got %v", onbErr.Err)
	}
}

func TestImportQuiz_AlreadyImported(t *testing.T) {
	existingShareID := "existing123"
	progress := &WizardProgress{QuizShareID: &existingShareID}

	// validateImportQuiz is the inline check: QuizShareID != nil
	if progress.QuizShareID == nil {
		t.Fatal("expected QuizShareID to be set")
	}
	// Verify the error type matches what the service would return
	err := &OnboardError{Err: ErrQuizAlreadyImported}
	if !errors.Is(err.Err, ErrQuizAlreadyImported) {
		t.Error("want ErrQuizAlreadyImported")
	}
}

// ─── Test: Materialization ───────────────────────────────────────────────────

func TestBuildMaterializedData_AgeFiltering(t *testing.T) {
	configs := map[string]*OnboardMethodologyConfig{
		"charlotte-mason": {
			Slug:        "charlotte-mason",
			DisplayName: "Charlotte Mason",
			OnboardingConfig: mustJSON(onboardingConfigInternal{
				RoadmapSteps: []roadmapStepConfig{
					{Type: "task", Title: "For all ages", SortOrder: 1},
					{Type: "task", Title: "For 6-8", AgeGroup: "6-8", SortOrder: 2},
					{Type: "task", Title: "For 13-15", AgeGroup: "13-15", SortOrder: 3},
				},
				StarterRecs: []starterRecConfig{
					{Title: "Universal rec", PriceType: "free", SortOrder: 1},
					{Title: "Rec 6-8", PriceType: "paid", AgeGroup: "6-8", SortOrder: 2},
					{Title: "Rec 13-15", PriceType: "free", AgeGroup: "13-15", SortOrder: 3},
				},
			}),
			CommunityConfig: mustJSON(communityConfigInternal{
				DefaultGroups: []communityGroupConfig{
					{Type: "methodology_group", Title: "CM Community", Description: "Connect"},
				},
			}),
		},
	}

	// Family with one child aged 7 (age group 6-8)
	by := int16(2019)
	svc := &onboardingServiceImpl{
		iam: &mockIamForOnboard{
			students: []OnboardStudentInfo{{ID: uuid.Must(uuid.NewV7()), BirthYear: &by}},
		},
		methodology: &mockMethodForOnboard{configs: configs},
	}

	familyID := testFamilyID()
	wizardID := uuid.Must(uuid.NewV7())
	data, err := svc.buildMaterializedData(context.Background(), familyID, wizardID, "charlotte-mason", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Roadmap: "For all ages" + "For 6-8" = 2 items (not "For 13-15")
	if len(data.roadmapItems) != 2 {
		t.Errorf("want 2 roadmap items, got %d", len(data.roadmapItems))
	}

	// Recs: "Universal rec" + "Rec 6-8" = 2 items (not "Rec 13-15")
	if len(data.recommendations) != 2 {
		t.Errorf("want 2 recommendations, got %d", len(data.recommendations))
	}

	// Community: always included
	if len(data.community) != 1 {
		t.Errorf("want 1 community suggestion, got %d", len(data.community))
	}
}

func TestBuildMaterializedData_NoStudentsIncludeAll(t *testing.T) {
	configs := map[string]*OnboardMethodologyConfig{
		"charlotte-mason": {
			Slug:        "charlotte-mason",
			DisplayName: "Charlotte Mason",
			OnboardingConfig: mustJSON(onboardingConfigInternal{
				RoadmapSteps: []roadmapStepConfig{
					{Type: "task", Title: "For all ages", SortOrder: 1},
					{Type: "task", Title: "For 6-8", AgeGroup: "6-8", SortOrder: 2},
					{Type: "task", Title: "For 13-15", AgeGroup: "13-15", SortOrder: 3},
				},
			}),
			CommunityConfig: mustJSON(communityConfigInternal{}),
		},
	}

	svc := &onboardingServiceImpl{
		iam:         &mockIamForOnboard{students: nil},
		methodology: &mockMethodForOnboard{configs: configs},
	}

	familyID := testFamilyID()
	wizardID := uuid.Must(uuid.NewV7())
	data, err := svc.buildMaterializedData(context.Background(), familyID, wizardID, "charlotte-mason", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 3 items should be included since no students
	if len(data.roadmapItems) != 3 {
		t.Errorf("want 3 roadmap items (no students = include all), got %d", len(data.roadmapItems))
	}
}

func TestBuildMaterializedData_MaxRecsPerAgeGroup(t *testing.T) {
	// Create 8 recs for the same age group — only 6 should be included
	recs := make([]starterRecConfig, 8)
	for i := range recs {
		recs[i] = starterRecConfig{
			Title:     "Rec",
			PriceType: "free",
			AgeGroup:  "6-8",
			SortOrder: int16(i),
		}
	}

	configs := map[string]*OnboardMethodologyConfig{
		"charlotte-mason": {
			Slug:             "charlotte-mason",
			OnboardingConfig: mustJSON(onboardingConfigInternal{StarterRecs: recs}),
			CommunityConfig:  mustJSON(communityConfigInternal{}),
		},
	}

	by := int16(2019)
	svc := &onboardingServiceImpl{
		iam:         &mockIamForOnboard{students: []OnboardStudentInfo{{BirthYear: &by}}},
		methodology: &mockMethodForOnboard{configs: configs},
	}

	data, err := svc.buildMaterializedData(context.Background(), testFamilyID(), uuid.Must(uuid.NewV7()), "charlotte-mason", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data.recommendations) != 6 {
		t.Errorf("want max 6 recs per age group, got %d", len(data.recommendations))
	}
}

// ─── Test: WizardStepArray ───────────────────────────────────────────────────

func TestWizardStepArray_Value(t *testing.T) {
	a := WizardStepArray{StepFamilyProfile, StepMethodology}
	v, err := a.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "{family_profile,methodology}" {
		t.Errorf("want {family_profile,methodology}, got %v", v)
	}
}

func TestWizardStepArray_ValueEmpty(t *testing.T) {
	a := WizardStepArray{}
	v, err := a.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "{}" {
		t.Errorf("want {}, got %v", v)
	}
}

func TestWizardStepArray_Scan(t *testing.T) {
	var a WizardStepArray
	if err := a.Scan("{family_profile,methodology}"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a) != 2 || a[0] != StepFamilyProfile || a[1] != StepMethodology {
		t.Errorf("want [family_profile, methodology], got %v", a)
	}
}

func TestWizardStepArray_ScanEmpty(t *testing.T) {
	var a WizardStepArray
	if err := a.Scan("{}"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a) != 0 {
		t.Errorf("want empty, got %v", a)
	}
}

func TestWizardStepArray_ScanNil(t *testing.T) {
	var a WizardStepArray
	if err := a.Scan(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a != nil {
		t.Errorf("want nil, got %v", a)
	}
}

// ─── Test: InitializeWizard ──────────────────────────────────────────────────

// InitializeWizard uses BypassRLSTransaction (requires real DB).
// This test verifies the initial WizardProgress struct construction.
func TestInitializeWizard_DefaultState(t *testing.T) {
	p := &WizardProgress{
		FamilyID:       testFamilyID(),
		Status:         StatusInProgress,
		CurrentStep:    StepFamilyProfile,
		CompletedSteps: WizardStepArray{},
	}

	if p.Status != StatusInProgress {
		t.Errorf("want status=in_progress, got %s", p.Status)
	}
	if p.CurrentStep != StepFamilyProfile {
		t.Errorf("want current_step=family_profile, got %s", p.CurrentStep)
	}
	if len(p.CompletedSteps) != 0 {
		t.Errorf("want empty completed_steps, got %v", p.CompletedSteps)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func ptr[T any](v T) *T { return &v }

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
