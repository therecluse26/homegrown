//go:build integration

package onboard

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/onboard/...
//
// Tests spin up a postgis/postgis Docker container via testcontainers-go,
// run all goose migrations (including seed data), and verify invariants that
// can only be checked against a real database (RLS, FamilyScope, transactions).
//
// Skipped automatically if Docker is unavailable.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDB holds the GORM connection shared across all integration tests.
// Populated by TestMain; nil when Docker is unavailable.
var testDB *gorm.DB

// TestMain runs once per package invocation under the integration build tag.
// It starts the PostGIS container, runs all goose migrations, and cleans up
// after all tests finish.
func TestMain(m *testing.M) {
	ctx := context.Background()

	db, teardown, err := startTestDB(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: skipping db setup: %v\n", err)
		os.Exit(m.Run())
	}
	testDB = db

	code := m.Run()
	teardown()
	os.Exit(code)
}

// startTestDB launches a postgis/postgis container and returns a GORM DB
// connection after all goose migrations have been applied.
func startTestDB(ctx context.Context) (*gorm.DB, func(), error) {
	req := testcontainers.ContainerRequest{
		Image: "postgis/postgis:16-3.4",
		Env: map[string]string{
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
			wait.ForListeningPort("5432/tcp"),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("testcontainers: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("get host: %w", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("get port: %w", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=postgres password=testpass dbname=testdb sslmode=disable",
		host, port.Port(),
	)

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("gorm open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("get sql.DB: %w", err)
	}

	wd, _ := os.Getwd()
	migrationsDir := filepath.Join(wd, "..", "..", "migrations")

	goose.SetDialect("postgres") //nolint:errcheck
	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		_ = sqlDB.Close()
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("goose up: %w", err)
	}

	teardown := func() {
		_ = sqlDB.Close()
		_ = container.Terminate(ctx)
	}
	return db, teardown, nil
}

// ─── Test Fixtures ─────────────────────────────────────────────────────────────

// testFamily creates an iam_families row directly in the DB (bypassing iam service).
// Returns the family UUID. The family uses charlotte-mason as primary methodology.
func testFamily(t *testing.T, db *gorm.DB, displayName string) uuid.UUID {
	t.Helper()
	familyID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid: %v", err)
	}
	err = db.Exec(
		`INSERT INTO iam_families (id, display_name, primary_methodology_slug)
		 VALUES (?, ?, 'charlotte-mason')`, familyID, displayName,
	).Error
	if err != nil {
		t.Fatalf("insert family: %v", err)
	}
	return familyID
}

// cleanupFamily deletes a family and all cascade-dependent rows.
func cleanupFamily(t *testing.T, db *gorm.DB, familyID uuid.UUID) {
	t.Helper()
	if err := db.Exec("DELETE FROM iam_families WHERE id = ?", familyID).Error; err != nil {
		t.Logf("cleanup family %s: %v", familyID, err)
	}
}

// ─── Integration Setup ─────────────────────────────────────────────────────────

// integrationSetup holds the real repos + stub cross-domain adapters for one test.
type integrationSetup struct {
	svc OnboardingService
	db  *gorm.DB
}

// newIntegrationSetup wires the onboarding service against the real testDB.
// Stub adapters provide predictable behavior for cross-domain calls.
func newIntegrationSetup(t *testing.T) *integrationSetup {
	t.Helper()
	if testDB == nil {
		t.Skip("testDB not available (Docker required for integration tests)")
	}

	wizardRepo := NewPgWizardProgressRepository(testDB)
	roadmapRepo := NewPgRoadmapItemRepository(testDB)
	recRepo := NewPgStarterRecommendationRepository(testDB)
	communityRepo := NewPgCommunitySuggestionRepository(testDB)
	eventBus := shared.NewEventBus()

	// Stub IAM adapter: profile/student ops are no-ops (DB rows created by testFamily/testStudent).
	// ListStudents returns actual students from the DB for age group resolution.
	iamAdapter := NewIamAdapter(
		func(_ context.Context, _ *shared.FamilyScope, _ UpdateFamilyProfileCommand) error {
			return nil // stub: profile already exists in test fixture
		},
		func(_ context.Context, _ *shared.FamilyScope, _ AddChildCommand) error {
			return nil // stub: student created by testStudent
		},
		func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
			return nil // stub: delete is a no-op in tests
		},
		func(_ context.Context, familyID uuid.UUID) ([]OnboardStudentInfo, error) {
			// Real DB lookup: query iam_students for age group resolution
			var rows []struct {
				ID        uuid.UUID
				BirthYear *int16
			}
			err := testDB.Raw(
				"SELECT id, birth_year FROM iam_students WHERE family_id = ?", familyID,
			).Scan(&rows).Error
			if err != nil {
				return nil, err
			}
			result := make([]OnboardStudentInfo, len(rows))
			for i, r := range rows {
				result[i] = OnboardStudentInfo{ID: r.ID, BirthYear: r.BirthYear}
			}
			return result, nil
		},
	)

	// Stub methodology adapter: returns canned methodology configs.
	methodAdapter := NewMethodAdapter(
		func(_ context.Context, slug string) (*OnboardMethodologyConfig, error) {
			// Return a minimal config that produces materialized data.
			onbConfig := onboardingConfigInternal{
				RoadmapSteps: []roadmapStepConfig{
					{Type: "task", Title: "Test roadmap: " + slug, Description: "Test desc", SortOrder: 1},
				},
				StarterRecs: []starterRecConfig{
					{Title: "Test rec: " + slug, Description: "Test rec desc", PriceType: "free", SortOrder: 1},
				},
			}
			commConfig := communityConfigInternal{
				DefaultGroups: []communityGroupConfig{
					{Type: "methodology_group", Title: "Test community: " + slug, Description: "Test community desc"},
				},
			}
			onbJSON, _ := json.Marshal(onbConfig)
			commJSON, _ := json.Marshal(commConfig)
			return &OnboardMethodologyConfig{
				Slug:             slug,
				DisplayName:      slug,
				OnboardingConfig: onbJSON,
				CommunityConfig:  commJSON,
			}, nil
		},
		func(_ context.Context) (string, error) {
			return "charlotte-mason", nil // default methodology
		},
		func(_ context.Context, _ []string) (bool, error) {
			return true, nil // all slugs are valid in test
		},
		func(_ context.Context, _ *shared.FamilyScope, _ string, _ []string) error {
			return nil // stub: methodology update is a no-op
		},
	)

	// Stub discovery adapter: returns a canned quiz result.
	discoverAdapter := NewDiscoverAdapter(
		func(_ context.Context, shareID string) (*OnboardQuizResult, error) {
			if shareID == "valid-share" {
				return &OnboardQuizResult{
					ShareID: shareID,
					Recommendations: []OnboardQuizRecommendation{
						{MethodologySlug: "charlotte-mason", MethodologyName: "Charlotte Mason", ScorePercentage: 85, Explanation: "Best match"},
						{MethodologySlug: "classical", MethodologyName: "Classical", ScorePercentage: 60, Explanation: "Good match"},
					},
				}, nil
			}
			return nil, fmt.Errorf("not found")
		},
		func(_ context.Context, _ string, _ uuid.UUID) error {
			return nil // stub: claim is a no-op
		},
	)

	svc := NewOnboardingService(
		wizardRepo, roadmapRepo, recRepo, communityRepo,
		iamAdapter, methodAdapter, discoverAdapter,
		eventBus, testDB,
	)

	return &integrationSetup{svc: svc, db: testDB}
}

// scopeForFamily builds a FamilyScope from a family UUID for testing.
func scopeForFamily(familyID uuid.UUID) *shared.FamilyScope {
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
	return &scope
}

// ─── Test 1: Wizard Happy Path ─────────────────────────────────────────────────

// TestIntegration_WizardHappyPath tests the full wizard lifecycle:
// create family → initialize wizard → complete all steps → complete wizard.
// [04-onboard §15 assertion: wizard happy path]
func TestIntegration_WizardHappyPath(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	// Create test family
	familyID := testFamily(t, s.db, "Happy Path Family")
	t.Cleanup(func() { cleanupFamily(t, s.db, familyID) })
	scope := scopeForFamily(familyID)

	// Initialize wizard (simulates FamilyCreated event handler)
	if err := s.svc.InitializeWizard(ctx, familyID); err != nil {
		t.Fatalf("InitializeWizard: %v", err)
	}

	// Verify initial state
	progress, err := s.svc.GetProgress(ctx, scope)
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if progress.Status != StatusInProgress {
		t.Errorf("want status=in_progress, got %s", progress.Status)
	}
	if progress.CurrentStep != StepFamilyProfile {
		t.Errorf("want current_step=family_profile, got %s", progress.CurrentStep)
	}

	// Step 1: Update family profile (required)
	progress, err = s.svc.UpdateFamilyProfile(ctx, scope, UpdateFamilyProfileCommand{
		DisplayName: "Happy Path Family Updated",
	})
	if err != nil {
		t.Fatalf("UpdateFamilyProfile: %v", err)
	}
	if !containsStep(progress.CompletedSteps, StepFamilyProfile) {
		t.Error("family_profile step not in completed_steps")
	}

	// Step 2: Add child (optional)
	progress, err = s.svc.AddChild(ctx, scope, AddChildCommand{
		DisplayName: "Test Child",
	})
	if err != nil {
		t.Fatalf("AddChild: %v", err)
	}
	if !containsStep(progress.CompletedSteps, StepChildren) {
		t.Error("children step not in completed_steps")
	}

	// Step 3: Select methodology (required)
	progress, err = s.svc.SelectMethodology(ctx, scope, SelectMethodologyCommand{
		PrimaryMethodologySlug: "charlotte-mason",
		MethodologyPath:        "exploration",
	})
	if err != nil {
		t.Fatalf("SelectMethodology: %v", err)
	}
	if !containsStep(progress.CompletedSteps, StepMethodology) {
		t.Error("methodology step not in completed_steps")
	}

	// Verify materialization produced roadmap items
	roadmap, err := s.svc.GetRoadmap(ctx, scope)
	if err != nil {
		t.Fatalf("GetRoadmap: %v", err)
	}
	if len(roadmap.Groups) == 0 {
		t.Error("want non-empty roadmap groups after methodology selection")
	}

	// Verify recommendations were materialized
	recs, err := s.svc.GetRecommendations(ctx, scope)
	if err != nil {
		t.Fatalf("GetRecommendations: %v", err)
	}
	if len(recs.Groups) == 0 {
		t.Error("want non-empty recommendation groups after methodology selection")
	}

	// Verify community suggestions were materialized
	community, err := s.svc.GetCommunity(ctx, scope)
	if err != nil {
		t.Fatalf("GetCommunity: %v", err)
	}
	if len(community.Items) == 0 {
		t.Error("want non-empty community items after methodology selection")
	}

	// Complete wizard
	progress, err = s.svc.CompleteWizard(ctx, scope)
	if err != nil {
		t.Fatalf("CompleteWizard: %v", err)
	}
	if progress.Status != StatusCompleted {
		t.Errorf("want status=completed, got %s", progress.Status)
	}
	if progress.CompletedAt == nil {
		t.Error("want non-nil completed_at")
	}

	// Verify double-complete returns 409
	_, err = s.svc.CompleteWizard(ctx, scope)
	if err == nil {
		t.Fatal("expected error on second CompleteWizard call")
	}
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) || !errors.Is(onbErr.Err, ErrWizardAlreadyCompleted) {
		t.Errorf("want ErrWizardAlreadyCompleted, got %v", err)
	}
}

// ─── Test 2: Wizard Skip Path ──────────────────────────────────────────────────

// TestIntegration_WizardSkipPath tests the skip workflow:
// create family → initialize wizard → skip wizard.
// [04-onboard §15 assertion: wizard skip path]
func TestIntegration_WizardSkipPath(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	familyID := testFamily(t, s.db, "Skip Path Family")
	t.Cleanup(func() { cleanupFamily(t, s.db, familyID) })
	scope := scopeForFamily(familyID)

	if err := s.svc.InitializeWizard(ctx, familyID); err != nil {
		t.Fatalf("InitializeWizard: %v", err)
	}

	// Skip wizard immediately
	progress, err := s.svc.SkipWizard(ctx, scope)
	if err != nil {
		t.Fatalf("SkipWizard: %v", err)
	}
	if progress.Status != StatusSkipped {
		t.Errorf("want status=skipped, got %s", progress.Status)
	}
	if progress.CompletedAt == nil {
		t.Error("want non-nil completed_at after skip")
	}

	// Verify double-skip returns 409
	_, err = s.svc.SkipWizard(ctx, scope)
	if err == nil {
		t.Fatal("expected error on second SkipWizard call")
	}
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) || !errors.Is(onbErr.Err, ErrWizardAlreadySkipped) {
		t.Errorf("want ErrWizardAlreadySkipped, got %v", err)
	}

	// Verify UpdateFamilyProfile returns 409 on skipped wizard
	_, err = s.svc.UpdateFamilyProfile(ctx, scope, UpdateFamilyProfileCommand{
		DisplayName: "Should Fail",
	})
	if err == nil {
		t.Fatal("expected error updating profile on skipped wizard")
	}
	if !errors.As(err, &onbErr) || !errors.Is(onbErr.Err, ErrWizardNotInProgress) {
		t.Errorf("want ErrWizardNotInProgress, got %v", err)
	}
}

// ─── Test 3: Methodology Selection Triggers Materialization ─────────────────────

// TestIntegration_MethodologyTriggersMaterialization verifies that selecting a
// methodology populates roadmap, recommendations, and community tables.
// [04-onboard §15 assertion: methodology selection triggers materialization]
func TestIntegration_MethodologyTriggersMaterialization(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	familyID := testFamily(t, s.db, "Materialization Family")
	t.Cleanup(func() { cleanupFamily(t, s.db, familyID) })
	scope := scopeForFamily(familyID)

	if err := s.svc.InitializeWizard(ctx, familyID); err != nil {
		t.Fatalf("InitializeWizard: %v", err)
	}

	// Before methodology: no materialized data
	_, err := s.svc.GetRoadmap(ctx, scope)
	if err == nil {
		t.Error("expected error (no roadmap) before methodology selection")
	}

	// Select methodology
	_, err = s.svc.SelectMethodology(ctx, scope, SelectMethodologyCommand{
		PrimaryMethodologySlug:    "classical",
		SecondaryMethodologySlugs: []string{"charlotte-mason"},
		MethodologyPath:           "exploration",
		ExplanationAcknowledged:   true,
	})
	if err != nil {
		t.Fatalf("SelectMethodology: %v", err)
	}

	// After methodology: verify materialized data exists
	roadmap, err := s.svc.GetRoadmap(ctx, scope)
	if err != nil {
		t.Fatalf("GetRoadmap after methodology: %v", err)
	}
	// Should have items from both primary (classical) and secondary (charlotte-mason)
	totalItems := 0
	for _, g := range roadmap.Groups {
		totalItems += len(g.Items)
	}
	if totalItems < 2 {
		t.Errorf("want roadmap items from both methodologies, got %d total items", totalItems)
	}

	recs, err := s.svc.GetRecommendations(ctx, scope)
	if err != nil {
		t.Fatalf("GetRecommendations after methodology: %v", err)
	}
	totalRecs := 0
	for _, g := range recs.Groups {
		totalRecs += len(g.Items)
	}
	if totalRecs < 2 {
		t.Errorf("want recommendation items from both methodologies, got %d", totalRecs)
	}

	community, err := s.svc.GetCommunity(ctx, scope)
	if err != nil {
		t.Fatalf("GetCommunity after methodology: %v", err)
	}
	if len(community.Items) < 2 {
		t.Errorf("want community items from both methodologies, got %d", len(community.Items))
	}

	// Verify re-materialization: select a different methodology
	_, err = s.svc.SelectMethodology(ctx, scope, SelectMethodologyCommand{
		PrimaryMethodologySlug: "waldorf",
		MethodologyPath:        "exploration",
	})
	if err != nil {
		t.Fatalf("SelectMethodology (re-select): %v", err)
	}

	roadmap2, err := s.svc.GetRoadmap(ctx, scope)
	if err != nil {
		t.Fatalf("GetRoadmap after re-select: %v", err)
	}
	// After re-materialization: all items should be from waldorf only
	for _, g := range roadmap2.Groups {
		for _, item := range g.Items {
			if item.MethodologySlug != "waldorf" {
				t.Errorf("want methodology_slug=waldorf after re-select, got %s", item.MethodologySlug)
			}
		}
	}
}

// ─── Test 4: Required Step Validation on Complete ──────────────────────────────

// TestIntegration_RequiredStepValidation verifies that CompleteWizard fails
// when required steps (family_profile, methodology) are not completed.
// [04-onboard §15 assertion: required step validation on complete]
func TestIntegration_RequiredStepValidation(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	familyID := testFamily(t, s.db, "Validation Family")
	t.Cleanup(func() { cleanupFamily(t, s.db, familyID) })
	scope := scopeForFamily(familyID)

	if err := s.svc.InitializeWizard(ctx, familyID); err != nil {
		t.Fatalf("InitializeWizard: %v", err)
	}

	// Try to complete with no steps done — should fail
	_, err := s.svc.CompleteWizard(ctx, scope)
	if err == nil {
		t.Fatal("expected error completing wizard with no steps done")
	}
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) || !errors.Is(onbErr.Err, ErrRequiredStepsMissing) {
		t.Errorf("want ErrRequiredStepsMissing, got %v", err)
	}

	// Complete only family_profile (still missing methodology)
	_, err = s.svc.UpdateFamilyProfile(ctx, scope, UpdateFamilyProfileCommand{
		DisplayName: "Validation Family Updated",
	})
	if err != nil {
		t.Fatalf("UpdateFamilyProfile: %v", err)
	}

	_, err = s.svc.CompleteWizard(ctx, scope)
	if err == nil {
		t.Fatal("expected error completing wizard with only family_profile done")
	}
	if !errors.As(err, &onbErr) || !errors.Is(onbErr.Err, ErrRequiredStepsMissing) {
		t.Errorf("want ErrRequiredStepsMissing (methodology not done), got %v", err)
	}

	// Now complete methodology — wizard should be completable
	_, err = s.svc.SelectMethodology(ctx, scope, SelectMethodologyCommand{
		PrimaryMethodologySlug: "charlotte-mason",
		MethodologyPath:        "exploration",
	})
	if err != nil {
		t.Fatalf("SelectMethodology: %v", err)
	}

	progress, err := s.svc.CompleteWizard(ctx, scope)
	if err != nil {
		t.Fatalf("CompleteWizard (should succeed): %v", err)
	}
	if progress.Status != StatusCompleted {
		t.Errorf("want status=completed, got %s", progress.Status)
	}
}

// ─── Test 5: Quiz Import with Valid Share ID ────────────────────────────────────

// TestIntegration_QuizImport verifies the quiz import flow:
// import quiz result → quiz_share_id recorded → import same quiz again returns 409.
// [04-onboard §15 assertion: quiz import with valid share_id]
func TestIntegration_QuizImport(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	familyID := testFamily(t, s.db, "Quiz Import Family")
	t.Cleanup(func() { cleanupFamily(t, s.db, familyID) })
	scope := scopeForFamily(familyID)

	if err := s.svc.InitializeWizard(ctx, familyID); err != nil {
		t.Fatalf("InitializeWizard: %v", err)
	}

	// Import quiz with valid share_id
	resp, err := s.svc.ImportQuiz(ctx, scope, ImportQuizCommand{ShareID: "valid-share"})
	if err != nil {
		t.Fatalf("ImportQuiz: %v", err)
	}
	if resp.ShareID != "valid-share" {
		t.Errorf("want share_id=valid-share, got %s", resp.ShareID)
	}
	if resp.SuggestedPrimarySlug != "charlotte-mason" {
		t.Errorf("want suggested_primary_slug=charlotte-mason, got %s", resp.SuggestedPrimarySlug)
	}
	if len(resp.MethodologyRecommendations) != 2 {
		t.Errorf("want 2 methodology_recommendations, got %d", len(resp.MethodologyRecommendations))
	}
	// Verify explanation is populated
	for _, rec := range resp.MethodologyRecommendations {
		if rec.Explanation == "" {
			t.Errorf("want non-empty explanation for %s", rec.MethodologySlug)
		}
	}

	// Verify quiz_share_id is recorded in progress
	progress, err := s.svc.GetProgress(ctx, scope)
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if progress.QuizShareID == nil || *progress.QuizShareID != "valid-share" {
		t.Errorf("want quiz_share_id=valid-share in progress, got %v", progress.QuizShareID)
	}

	// Import same quiz again — should return 409
	_, err = s.svc.ImportQuiz(ctx, scope, ImportQuizCommand{ShareID: "valid-share"})
	if err == nil {
		t.Fatal("expected error on second ImportQuiz call")
	}
	var onbErr *OnboardError
	if !errors.As(err, &onbErr) || !errors.Is(onbErr.Err, ErrQuizAlreadyImported) {
		t.Errorf("want ErrQuizAlreadyImported, got %v", err)
	}

	// Import invalid quiz — should return error
	_, err = s.svc.ImportQuiz(ctx, scope, ImportQuizCommand{ShareID: "invalid-share"})
	if err == nil {
		t.Fatal("expected error for invalid share_id")
	}
}

// ─── Test 6: Cross-Family Access Prevention ────────────────────────────────────

// TestIntegration_CrossFamilyAccessPrevention verifies that FamilyScope enforces
// data isolation: family A cannot read family B's wizard progress.
// [04-onboard §15 assertion: cross-family access prevention]
func TestIntegration_CrossFamilyAccessPrevention(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	// Create two families
	familyA := testFamily(t, s.db, "Family Alpha")
	t.Cleanup(func() { cleanupFamily(t, s.db, familyA) })
	familyB := testFamily(t, s.db, "Family Beta")
	t.Cleanup(func() { cleanupFamily(t, s.db, familyB) })

	scopeA := scopeForFamily(familyA)
	scopeB := scopeForFamily(familyB)

	// Initialize wizards for both families
	if err := s.svc.InitializeWizard(ctx, familyA); err != nil {
		t.Fatalf("InitializeWizard A: %v", err)
	}
	if err := s.svc.InitializeWizard(ctx, familyB); err != nil {
		t.Fatalf("InitializeWizard B: %v", err)
	}

	// Family A completes profile
	_, err := s.svc.UpdateFamilyProfile(ctx, scopeA, UpdateFamilyProfileCommand{
		DisplayName: "Alpha Updated",
	})
	if err != nil {
		t.Fatalf("UpdateFamilyProfile A: %v", err)
	}

	// Family B's progress should NOT include family A's steps
	progressB, err := s.svc.GetProgress(ctx, scopeB)
	if err != nil {
		t.Fatalf("GetProgress B: %v", err)
	}
	if containsStep(progressB.CompletedSteps, StepFamilyProfile) {
		t.Error("family B should not have family A's completed family_profile step")
	}
	if progressB.FamilyID != familyB {
		t.Errorf("want family_id=%s for family B, got %s", familyB, progressB.FamilyID)
	}

	// Family A's progress should have the completed step
	progressA, err := s.svc.GetProgress(ctx, scopeA)
	if err != nil {
		t.Fatalf("GetProgress A: %v", err)
	}
	if !containsStep(progressA.CompletedSteps, StepFamilyProfile) {
		t.Error("family A should have completed family_profile step")
	}
	if progressA.FamilyID != familyA {
		t.Errorf("want family_id=%s for family A, got %s", familyA, progressA.FamilyID)
	}

	// Family A selects methodology → materializes guidance
	_, err = s.svc.SelectMethodology(ctx, scopeA, SelectMethodologyCommand{
		PrimaryMethodologySlug: "charlotte-mason",
		MethodologyPath:        "exploration",
	})
	if err != nil {
		t.Fatalf("SelectMethodology A: %v", err)
	}

	// Family A should have roadmap items
	_, err = s.svc.GetRoadmap(ctx, scopeA)
	if err != nil {
		t.Fatalf("GetRoadmap A: %v", err)
	}

	// Family B should NOT have roadmap items (no methodology selected)
	_, err = s.svc.GetRoadmap(ctx, scopeB)
	if err == nil {
		t.Error("family B should not have roadmap items (no methodology selected)")
	}
}

// ─── Helpers ────────────────────────────────────────────────────────────────────

func containsStep(steps []WizardStep, target WizardStep) bool {
	for _, s := range steps {
		if s == target {
			return true
		}
	}
	return false
}
