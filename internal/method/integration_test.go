//go:build integration

package method

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/method/...
//
// Tests spin up a postgis/postgis Docker container via testcontainers-go,
// run all goose migrations (including seed data), and verify invariants that
// can only be checked against a real database.
//
// Skipped automatically if Docker is unavailable.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/method/domain"
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
		// Docker not available or container failed to start — skip gracefully.
		fmt.Fprintf(os.Stderr, "integration: skipping db setup: %v\n", err)
		os.Exit(m.Run())
	}
	testDB = db

	code := m.Run()
	teardown()
	os.Exit(code)
}

// startTestDB launches a postgis/postgis container and returns a GORM DB
// connection after migrations have been applied.
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

// ─── Test Helpers ─────────────────────────────────────────────────────────────

// integrationSetup holds the real repos + stub IAM + event bus for one test.
type integrationSetup struct {
	defRepo        MethodologyDefinitionRepository
	toolRepo       ToolRepository
	activationRepo ToolActivationRepository
	iamStub        *stubIntegrationIamSvc
	eventBus       *shared.EventBus
	svc            MethodologyService
}

// newIntegrationSetup wires the method service against the real testDB.
// Skips the test if testDB is nil (Docker unavailable).
func newIntegrationSetup(t *testing.T) *integrationSetup {
	t.Helper()
	if testDB == nil {
		t.Skip("testDB not available (Docker required for integration tests)")
	}
	bus := shared.NewEventBus()
	stub := &stubIntegrationIamSvc{}
	defRepo := NewPgMethodologyDefinitionRepository(testDB)
	toolRepo := NewPgToolRepository(testDB)
	activationRepo := NewPgToolActivationRepository(testDB)
	svc := NewMethodologyService(defRepo, toolRepo, activationRepo, stub, bus)
	return &integrationSetup{
		defRepo:        defRepo,
		toolRepo:       toolRepo,
		activationRepo: activationRepo,
		iamStub:        stub,
		eventBus:       bus,
		svc:            svc,
	}
}

// stubIntegrationIamSvc is a minimal IamServiceForMethod for integration tests.
// It records SetFamilyMethodology calls so tests can verify persistence was attempted.
type stubIntegrationIamSvc struct {
	mu                   sync.Mutex
	primaryID            uuid.UUID
	secondaryIDs         []uuid.UUID
	student              *StudentInfo
	setCalledCount       int
	capturedPrimaryID    uuid.UUID
	capturedSecondaryIDs []uuid.UUID
}

func (s *stubIntegrationIamSvc) GetFamilyMethodologyIDs(_ context.Context, _ *shared.FamilyScope) (uuid.UUID, []uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sec := s.secondaryIDs
	if sec == nil {
		sec = []uuid.UUID{}
	}
	return s.primaryID, sec, nil
}

func (s *stubIntegrationIamSvc) GetStudent(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*StudentInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.student != nil {
		return s.student, nil
	}
	return nil, &shared.AppError{Code: "student_not_found", Message: "Student not found", StatusCode: 404}
}

func (s *stubIntegrationIamSvc) SetFamilyMethodology(_ context.Context, _ *shared.FamilyScope, primaryID uuid.UUID, secondaryIDs []uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setCalledCount++
	s.capturedPrimaryID = primaryID
	s.capturedSecondaryIDs = secondaryIDs
	return nil
}

// captureHandler is a shared.DomainEventHandler that records all received events.
type captureHandler struct {
	mu       sync.Mutex
	received []shared.DomainEvent
}

func (h *captureHandler) Handle(_ context.Context, event shared.DomainEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.received = append(h.received, event)
	return nil
}

func (h *captureHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.received)
}

func (h *captureHandler) first() shared.DomainEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.received) == 0 {
		return nil
	}
	return h.received[0]
}

// ─── §13 Assertion 1: Seed data counts ───────────────────────────────────────

// TestSeedData_MethodologyCount verifies the seed migration inserts exactly 6 active
// methodology definitions. [§13 assertion 1, 02-method §3.3]
func TestSeedData_MethodologyCount(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	defs, err := s.defRepo.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(defs) != 6 {
		t.Errorf("want 6 active methodologies, got %d", len(defs))
	}
}

// TestSeedData_ToolCount verifies the seed migration inserts exactly 22 tools.
// [§13 assertion 1, 02-method §8.1]
func TestSeedData_ToolCount(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	tools, err := s.toolRepo.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive tools: %v", err)
	}
	if len(tools) != 22 {
		t.Errorf("want 22 active tools, got %d", len(tools))
	}
}

// ─── §13 Assertion 1: Display order ──────────────────────────────────────────

// TestListActive_OrderedByDisplayOrder verifies ListActive returns methodologies
// sorted by display_order ascending, starting with charlotte-mason. [§13.1]
func TestListActive_OrderedByDisplayOrder(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	defs, err := s.defRepo.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(defs) == 0 {
		t.Fatal("want non-empty methodology list")
	}
	if defs[0].Slug != "charlotte-mason" {
		t.Errorf("want first methodology 'charlotte-mason' (display_order=1), got %q", defs[0].Slug)
	}
	if defs[len(defs)-1].Slug != "unschooling" {
		t.Errorf("want last methodology 'unschooling' (display_order=6), got %q", defs[len(defs)-1].Slug)
	}
	// Verify monotonically increasing display_order.
	for i := 1; i < len(defs); i++ {
		if defs[i].DisplayOrder <= defs[i-1].DisplayOrder {
			t.Errorf("display_order not ascending at index %d: %d <= %d",
				i, defs[i].DisplayOrder, defs[i-1].DisplayOrder)
		}
	}
}

// ─── §13 Assertion 2: Full philosophy JSON ───────────────────────────────────

// TestGetMethodologyBySlug_PhilosophyJSON verifies that FindBySlug returns the
// charlotte-mason record with a fully populated philosophy JSONB field. [§13.2]
func TestGetMethodologyBySlug_PhilosophyJSON(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	def, err := s.defRepo.FindBySlug(ctx, "charlotte-mason")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if def.Slug != "charlotte-mason" {
		t.Errorf("want slug 'charlotte-mason', got %q", def.Slug)
	}
	// Philosophy must be non-empty JSON, not null or '{}'.
	if len(def.Philosophy) == 0 || string(def.Philosophy) == "{}" || string(def.Philosophy) == "null" {
		t.Errorf("philosophy should be populated JSON, got %q", def.Philosophy)
	}
	// Service layer: GetMethodology returns full detail.
	detail, err := s.svc.GetMethodology(ctx, "charlotte-mason")
	if err != nil {
		t.Fatalf("GetMethodology: %v", err)
	}
	if len(detail.Philosophy) == 0 {
		t.Error("GetMethodology philosophy field is empty")
	}
}

// ─── §13 Assertion 4: Tools for a methodology ────────────────────────────────

// TestGetCharlotteMasonTools_CorrectCount verifies that the charlotte-mason
// methodology has the expected number of activated tools (9 specific + 3 universal +
// 1 lesson-sequences = 13). [§13.4]
func TestGetCharlotteMasonTools_CorrectCount(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	tools, err := s.svc.GetMethodologyTools(ctx, "charlotte-mason")
	if err != nil {
		t.Fatalf("GetMethodologyTools: %v", err)
	}
	// 9 cm-specific + video-lessons + content-viewer + video-player + lesson-sequences = 13
	const wantCount = 13
	if len(tools) != wantCount {
		t.Errorf("want %d tools for charlotte-mason, got %d", wantCount, len(tools))
	}
	// Verify the universal video-lessons tool is present.
	found := false
	for _, tool := range tools {
		if tool.Slug == "video-lessons" {
			found = true
			break
		}
	}
	if !found {
		t.Error("want 'video-lessons' in charlotte-mason tool set (universal tool)")
	}
}

// ─── §13 Assertion 9+11: UpdateFamilyMethodology validation + event ──────────

// TestUpdateFamilyMethodology_ValidIDs_PublishesEvent verifies that:
//  1. UpdateFamilyMethodology succeeds when given valid active methodology IDs.
//  2. The FamilyMethodologyChanged event is published after persistence. [§13.11]
//  3. SetFamilyMethodology on the IAM adapter is called exactly once. [§13.11]
func TestUpdateFamilyMethodology_ValidIDs_PublishesEvent(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	// Load real methodology IDs from DB.
	defs, err := s.defRepo.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	cmDef := defs[0] // charlotte-mason (display_order=1)

	// Register an event capture handler.
	capture := &captureHandler{}
	s.eventBus.Subscribe(reflect.TypeOf(FamilyMethodologyChanged{}), capture)

	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.New()})
	resp, err := s.svc.UpdateFamilyMethodology(ctx, &scope, UpdateMethodologyCommand{
		PrimaryMethodologyID:    cmDef.ID,
		SecondaryMethodologyIDs: []uuid.UUID{},
	})
	if err != nil {
		t.Fatalf("UpdateFamilyMethodology: %v", err)
	}

	// Response should name charlotte-mason as primary.
	if resp.Primary.Slug != "charlotte-mason" {
		t.Errorf("want primary 'charlotte-mason', got %q", resp.Primary.Slug)
	}
	// Active tool count must be > 0.
	if resp.ActiveToolCount == 0 {
		t.Error("active_tool_count should be > 0 after valid update")
	}

	// IAM adapter should have been called exactly once.
	if s.iamStub.setCalledCount != 1 {
		t.Errorf("want SetFamilyMethodology called 1 time, got %d", s.iamStub.setCalledCount)
	}
	if s.iamStub.capturedPrimaryID != cmDef.ID {
		t.Errorf("SetFamilyMethodology captured wrong primaryID: got %v, want %v",
			s.iamStub.capturedPrimaryID, cmDef.ID)
	}

	// Event must have been published.
	if capture.count() != 1 {
		t.Errorf("want 1 FamilyMethodologyChanged event, got %d", capture.count())
	}
	evt, ok := capture.first().(FamilyMethodologyChanged)
	if !ok {
		t.Fatalf("want FamilyMethodologyChanged event, got %T", capture.first())
	}
	if evt.PrimaryMethodologyID != cmDef.ID {
		t.Errorf("event PrimaryMethodologyID = %v, want %v", evt.PrimaryMethodologyID, cmDef.ID)
	}
}

// TestUpdateFamilyMethodology_InvalidID_ReturnsError verifies that passing a
// non-existent methodology ID returns ErrInvalidMethodologyIDs. [§13.9]
func TestUpdateFamilyMethodology_InvalidID_ReturnsError(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.New()})
	_, err := s.svc.UpdateFamilyMethodology(ctx, &scope, UpdateMethodologyCommand{
		PrimaryMethodologyID:    uuid.New(), // random UUID — not in DB
		SecondaryMethodologyIDs: []uuid.UUID{},
	})
	if err == nil {
		t.Fatal("expected error for invalid methodology ID, got nil")
	}
	var methodErr *domain.MethodError
	if !errors.As(err, &methodErr) {
		t.Fatalf("want *MethodError, got %T: %v", err, err)
	}
	if !errors.Is(methodErr.Err, domain.ErrInvalidMethodologyIDs) {
		t.Errorf("want ErrInvalidMethodologyIDs, got %v", methodErr.Err)
	}
}

// ─── §13 Assertions 13–14: Student tool resolution ───────────────────────────

// TestStudentTools_WithOverride verifies that ResolveStudentTools uses the
// student's methodology override when one is set. [§13.13]
func TestStudentTools_WithOverride(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	defs, err := s.defRepo.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	// Use two different methodologies: CM as family primary, Traditional as override.
	cmDef := defs[0]     // charlotte-mason
	tradDef := defs[1]   // traditional

	studentID := uuid.New()
	s.iamStub.primaryID = cmDef.ID
	s.iamStub.student = &StudentInfo{
		ID:                    studentID,
		MethodologyOverrideID: &tradDef.ID,
	}

	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.New()})
	tools, err := s.svc.ResolveStudentTools(ctx, &scope, studentID)
	if err != nil {
		t.Fatalf("ResolveStudentTools: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("want non-empty tool set for student with override")
	}
	// All returned tools must come from traditional (not charlotte-mason).
	// nature-journals is CM-only; tests-grades is Traditional-only.
	// Verify tests-grades is present (Traditional-specific tool).
	found := false
	for _, tool := range tools {
		if tool.Slug == "tests-grades" {
			found = true
			break
		}
	}
	if !found {
		t.Error("want 'tests-grades' in traditional tool set")
	}
}

// TestStudentTools_NoOverrideFallsBackToFamily verifies that ResolveStudentTools
// falls back to the family methodology when no student override is set. [§13.14]
func TestStudentTools_NoOverrideFallsBackToFamily(t *testing.T) {
	s := newIntegrationSetup(t)
	ctx := context.Background()

	defs, err := s.defRepo.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	cmDef := defs[0] // charlotte-mason

	studentID := uuid.New()
	s.iamStub.primaryID = cmDef.ID
	s.iamStub.student = &StudentInfo{
		ID:                    studentID,
		MethodologyOverrideID: nil, // no override — falls back to family
	}

	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.New()})
	tools, err := s.svc.ResolveStudentTools(ctx, &scope, studentID)
	if err != nil {
		t.Fatalf("ResolveStudentTools: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("want non-empty tool set for student without override")
	}
	// nature-journals is charlotte-mason-specific; should be present in family fallback.
	found := false
	for _, tool := range tools {
		if tool.Slug == "nature-journals" {
			found = true
			break
		}
	}
	if !found {
		t.Error("want 'nature-journals' from charlotte-mason fallback tools")
	}
}
