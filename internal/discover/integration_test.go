//go:build integration

package discover

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/discover/...
//
// Tests spin up a postgis/postgis Docker container via testcontainers-go,
// run all goose migrations (including seed data), and verify invariants that
// can only be checked against a real database.
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

// ─── Integration Setup ────────────────────────────────────────────────────────

// integrationSetup holds the real repos + stub method adapter for one test.
type integrationSetup struct {
	quizDefRepo QuizDefinitionRepository
	quizResRepo QuizResultRepository
	stateRepo   StateGuideRepository
	svc         DiscoveryService
}

// newIntegrationSetup wires the discovery service against the real testDB.
// Skips the test if testDB is nil (Docker unavailable).
func newIntegrationSetup(t *testing.T) *integrationSetup {
	t.Helper()
	if testDB == nil {
		t.Skip("testDB not available (Docker required for integration tests)")
	}

	quizDefRepo := NewPgQuizDefinitionRepository(testDB)
	quizResRepo := NewPgQuizResultRepository(testDB)
	stateRepo := NewPgStateGuideRepository(testDB)

	// Stub method adapter returns the slug as the display name.
	methodAdapter := NewMethodAdapter(func(_ context.Context, slug string) (string, error) {
		return slug, nil
	})

	svc := NewDiscoveryService(quizDefRepo, quizResRepo, stateRepo, methodAdapter)

	return &integrationSetup{
		quizDefRepo: quizDefRepo,
		quizResRepo: quizResRepo,
		stateRepo:   stateRepo,
		svc:         svc,
	}
}

// ─── §15 Assertion 13: 51 state guides after seed ────────────────────────────

// TestSeedData_StateGuideCount verifies that exactly 51 state guides are seeded
// (50 states + DC). [03-discover §15.13]
func TestSeedData_StateGuideCount(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	guides, err := s.stateRepo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(guides) != 51 {
		t.Errorf("want 51 state guides, got %d", len(guides))
	}
}

// TestSeedData_StateGuidesDraft verifies all seeded guides are draft status.
func TestSeedData_StateGuidesDraft(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	guides, err := s.stateRepo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	for _, g := range guides {
		if g.Status != "draft" {
			t.Errorf("want status='draft' for seeded guide %q, got %q", g.StateCode, g.Status)
		}
	}
}

// TestListStateGuides_AllDraftShowsUnavailable verifies that all seeded guides
// have IsAvailable=false in the list response (all seeded as draft). [03-discover §8.3]
func TestListStateGuides_AllDraftShowsUnavailable(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	resp, err := s.svc.ListStateGuides(ctx)
	if err != nil {
		t.Fatalf("ListStateGuides: %v", err)
	}
	for _, guide := range resp {
		if guide.IsAvailable {
			t.Errorf("want IsAvailable=false for draft guide %q, got true", guide.StateCode)
		}
	}
}

// ─── §15 Assertion 16: Draft state guide returns 404 ─────────────────────────

// TestGetStateGuide_DraftReturns404 verifies that a draft guide returns the
// not-published error, which maps to HTTP 404. [03-discover §15.16]
func TestGetStateGuide_DraftReturns404(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	_, err := s.svc.GetStateGuide(ctx, "CA")
	if err == nil {
		t.Fatal("expected error for draft CA guide")
	}

	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T: %v", err, err)
	}
	if !errors.Is(discErr.Err, ErrStateGuideNotPublished) {
		t.Errorf("want ErrStateGuideNotPublished, got %v", discErr.Err)
	}
}

// TestGetStateGuide_InvalidShareID returns not found for unknown state code.
func TestGetStateGuide_UnknownStateCodeReturnsError(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	_, err := s.svc.GetStateGuide(ctx, "XX")
	if err == nil {
		t.Fatal("expected error for unknown state code XX")
	}
	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T: %v", err, err)
	}
	if !errors.Is(discErr.Err, ErrStateGuideNotFound) {
		t.Errorf("want ErrStateGuideNotFound, got %v", discErr.Err)
	}
}

// ─── §15 Assertions 2, 3: No active quiz → 404; Get active quiz when present ─

// TestGetActiveQuiz_NoActiveQuiz verifies that ErrNoActiveQuiz is returned when
// no quiz has been inserted. [03-discover §15.2]
func TestIntegration_GetActiveQuiz_NoActiveQuiz(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	_, err := s.svc.GetActiveQuiz(ctx)
	if err == nil {
		t.Fatal("expected error (no active quiz seeded)")
	}
	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T", err)
	}
	if !errors.Is(discErr.Err, ErrNoActiveQuiz) {
		t.Errorf("want ErrNoActiveQuiz, got %v", discErr.Err)
	}
}

// TestIntegration_FullQuizFlow tests the full quiz lifecycle:
// insert active quiz → get quiz → submit answers → retrieve by share_id.
// [03-discover §15.2, §15.3, §15.12]
func TestIntegration_FullQuizFlow(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	// ── 1. Insert an active quiz definition ───────────────────────────────────
	questions := []quizQuestionInternal{
		{
			ID:       "q1",
			Category: "style",
			Text:     "Test question?",
			Answers: []quizAnswerInternal{
				{ID: "a1", Text: "Option A", Weights: map[string]float64{"charlotte-mason": 0.8, "traditional": 0.2}},
				{ID: "a2", Text: "Option B", Weights: map[string]float64{"charlotte-mason": 0.2, "traditional": 0.8}},
			},
		},
	}
	explanations := quizExplanationsInternal{
		"charlotte-mason": {MatchText: "CM match", MismatchText: "CM mismatch"},
		"traditional":     {MatchText: "Trad match", MismatchText: "Trad mismatch"},
	}
	questionsJSON, _ := json.Marshal(questions)
	explanationsJSON, _ := json.Marshal(explanations)

	def := QuizDefinition{
		Version:      1,
		Title:        "Integration Test Quiz",
		Description:  "Test quiz for integration tests",
		Status:       "active",
		Questions:    json.RawMessage(questionsJSON),
		Explanations: json.RawMessage(explanationsJSON),
	}
	if err := testDB.WithContext(ctx).Create(&def).Error; err != nil {
		t.Fatalf("create quiz definition: %v", err)
	}
	t.Cleanup(func() {
		testDB.WithContext(ctx).Delete(&def)
	})

	// ── 2. Get active quiz (weights must be stripped) ─────────────────────────
	quizResp, err := s.svc.GetActiveQuiz(ctx)
	if err != nil {
		t.Fatalf("GetActiveQuiz: %v", err)
	}
	if quizResp.Title != "Integration Test Quiz" {
		t.Errorf("want title 'Integration Test Quiz', got %q", quizResp.Title)
	}
	// Verify weights are stripped. [03-discover §15.1]
	for _, q := range quizResp.Questions {
		for _, a := range q.Answers {
			raw, _ := json.Marshal(a)
			var m map[string]json.RawMessage
			_ = json.Unmarshal(raw, &m)
			if _, hasWeights := m["weights"]; hasWeights {
				t.Errorf("weights field leaked for answer %q", a.ID)
			}
		}
	}

	// ── 3. Submit quiz answers (full answer set) ──────────────────────────────
	submitResp, err := s.svc.SubmitQuiz(ctx, SubmitQuizCommand{
		Answers: map[string]string{"q1": "a1"},
	})
	if err != nil {
		t.Fatalf("SubmitQuiz: %v", err)
	}
	if submitResp.ShareID == "" {
		t.Fatal("share_id must not be empty")
	}
	// [03-discover §15.11] share_id is 12 chars.
	if len(submitResp.ShareID) != shareIDLength {
		t.Errorf("want share_id length %d, got %d", shareIDLength, len(submitResp.ShareID))
	}
	// [03-discover §8.2 gap fix] quiz_version, created_at, is_claimed must be present.
	if submitResp.QuizVersion != 1 {
		t.Errorf("want quiz_version 1, got %d", submitResp.QuizVersion)
	}
	if submitResp.CreatedAt.IsZero() {
		t.Error("want non-zero created_at in submit response")
	}
	if submitResp.IsClaimed {
		t.Error("want is_claimed=false for anonymous submission")
	}
	// [03-discover §15.26] rank must be 1-indexed.
	for i, rec := range submitResp.Recommendations {
		if rec.Rank != uint8(i+1) {
			t.Errorf("want rank %d at index %d, got %d (methodology_slug=%q)",
				i+1, i, rec.Rank, rec.MethodologySlug)
		}
		if rec.MethodologySlug == "" {
			t.Errorf("want non-empty methodology_slug at index %d", i)
		}
	}

	// ── 4. Retrieve result by share_id ────────────────────────────────────────
	getResp, err := s.svc.GetQuizResult(ctx, submitResp.ShareID)
	if err != nil {
		t.Fatalf("GetQuizResult: %v", err)
	}
	if getResp.ShareID != submitResp.ShareID {
		t.Errorf("share_id mismatch: got %q, want %q", getResp.ShareID, submitResp.ShareID)
	}
	if len(getResp.Recommendations) == 0 {
		t.Error("want non-empty recommendations from retrieved result")
	}
	if getResp.CreatedAt.IsZero() {
		t.Error("want non-zero created_at in retrieved result")
	}
	if getResp.IsClaimed {
		t.Error("want is_claimed=false for unclaimed retrieved result")
	}

	// ── 5. Invalid share_id returns 404 ──────────────────────────────────────
	_, err = s.svc.GetQuizResult(ctx, "notavalidid!")
	if err == nil {
		t.Fatal("expected error for invalid share_id")
	}
	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T", err)
	}
	if !errors.Is(discErr.Err, ErrQuizResultNotFound) {
		t.Errorf("want ErrQuizResultNotFound, got %v", discErr.Err)
	}
}

// TestIntegration_PartialQuizSubmission verifies that a partial answer set
// (not all questions answered) succeeds. [03-discover §15.6]
func TestIntegration_PartialQuizSubmission(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	questions := []quizQuestionInternal{
		{ID: "q1", Category: "style", Text: "Q1", Answers: []quizAnswerInternal{
			{ID: "a1", Text: "A1", Weights: map[string]float64{"charlotte-mason": 0.9}},
		}},
		{ID: "q2", Category: "schedule", Text: "Q2", Answers: []quizAnswerInternal{
			{ID: "b1", Text: "B1", Weights: map[string]float64{"traditional": 0.9}},
		}},
	}
	questionsJSON, _ := json.Marshal(questions)

	def := QuizDefinition{
		Version:      1,
		Title:        "Partial Test Quiz",
		Status:       "active",
		Questions:    json.RawMessage(questionsJSON),
		Explanations: json.RawMessage(`{}`),
	}
	if err := testDB.WithContext(ctx).Create(&def).Error; err != nil {
		t.Fatalf("create quiz definition: %v", err)
	}
	t.Cleanup(func() {
		testDB.WithContext(ctx).Delete(&def)
	})

	// Submit only q1 (omit q2) — should succeed.
	resp, err := s.svc.SubmitQuiz(ctx, SubmitQuizCommand{
		Answers: map[string]string{"q1": "a1"},
	})
	if err != nil {
		t.Fatalf("SubmitQuiz partial: %v", err)
	}
	if resp.ShareID == "" {
		t.Error("share_id must not be empty for partial submission")
	}
}
