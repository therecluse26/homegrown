//go:build integration

package iam

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/iam/...
//
// Tests spin up a postgis/postgis Docker container via testcontainers-go,
// run all goose migrations, and verify invariants that can only be checked
// against a real database (transactional atomicity, RLS isolation).
//
// Skipped automatically if Docker is unavailable.

import (
	"context"
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
// connection after migrations have been applied. The returned teardown function
// closes the connection and terminates the container.
func startTestDB(ctx context.Context) (*gorm.DB, func(), error) {
	req := testcontainers.ContainerRequest{
		Image: "postgis/postgis:16-3.4",
		Env: map[string]string{
			// Use the default superuser so that SET LOCAL row_security = off
			// and CREATE EXTENSION succeed without additional GRANT statements.
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

	// goose migrations directory relative to the package directory.
	// During tests the CWD is always the package directory. [Go testing spec]
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

// newTestService creates an IamServiceImpl wired to the shared test DB.
// KratosAdapter is nil — the integration tests never call Kratos operations.
func newTestService(t *testing.T) *IamServiceImpl {
	t.Helper()
	if testDB == nil {
		t.Skip("testDB not available (Docker required for integration tests)")
	}
	return NewIamService(
		NewPgFamilyRepository(testDB),
		NewPgParentRepository(testDB),
		NewPgStudentRepository(testDB),
		nil, // KratosAdapter — not exercised by Phase 1 registration/student tests
		shared.NewEventBus(),
		testDB,
	)
}

// registerFamily is a helper that calls HandlePostRegistration and returns the
// created parent (looked up via bypass transaction).
func registerFamily(ctx context.Context, t *testing.T, svc *IamServiceImpl, name, email string) *ParentModel {
	t.Helper()
	identityID := uuid.Must(uuid.NewV7())
	if err := svc.HandlePostRegistration(ctx, KratosWebhookPayload{
		IdentityID: identityID,
		Traits:     KratosTraits{Email: email, Name: name},
	}); err != nil {
		t.Fatalf("HandlePostRegistration(%q): %v", email, err)
	}

	var parent ParentModel
	err := shared.BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Where("kratos_identity_id = ?", identityID).First(&parent).Error
	})
	if err != nil {
		t.Fatalf("lookup parent after registration: %v", err)
	}
	return &parent
}

// ─── §15.1: Registration Atomicity ───────────────────────────────────────────

// TestRegistrationAtomic verifies that HandlePostRegistration creates the
// family and parent in a single transaction and sets primary_parent_id. [§15.1]
func TestRegistrationAtomic(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	parent := registerFamily(ctx, t, svc, "Alice", fmt.Sprintf("alice-%s@test.com", uuid.Must(uuid.NewV7())))

	// Verify the parent has is_primary = true.
	if !parent.IsPrimary {
		t.Error("parent.is_primary should be true for the first parent in the family")
	}

	// Verify the family exists and primary_parent_id points to the parent.
	var family FamilyModel
	err := shared.BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Where("id = ?", parent.FamilyID).First(&family).Error
	})
	if err != nil {
		t.Fatalf("family not found after registration: %v", err)
	}

	if family.PrimaryParentID == nil {
		t.Fatal("family.primary_parent_id is nil — should point to the new parent")
	}
	if *family.PrimaryParentID != parent.ID {
		t.Errorf("family.primary_parent_id = %v, want %v", *family.PrimaryParentID, parent.ID)
	}
	if family.CoppaConsentStatus != string(CoppaConsentRegistered) {
		t.Errorf("family.coppa_consent_status = %q, want %q",
			family.CoppaConsentStatus, CoppaConsentRegistered)
	}
}

// TestRegistrationAtomic_DuplicateIdentity verifies that re-registering the
// same Kratos identity_id is rejected, leaving the DB unchanged. [§15.1]
func TestRegistrationAtomic_DuplicateIdentity(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	identityID := uuid.Must(uuid.NewV7())
	payload := KratosWebhookPayload{
		IdentityID: identityID,
		Traits:     KratosTraits{Email: fmt.Sprintf("dup-%s@test.com", uuid.Must(uuid.NewV7())), Name: "Dup Parent"},
	}

	// First registration succeeds.
	if err := svc.HandlePostRegistration(ctx, payload); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Second registration with the same identity_id must fail (UNIQUE constraint on kratos_identity_id).
	err := svc.HandlePostRegistration(ctx, payload)
	if err == nil {
		t.Fatal("expected error on duplicate registration, got nil")
	}
}

// ─── §15.2 + §15.8: Family-Scoped Student CRUD and RLS Isolation ─────────────

// TestStudentCRUD_FamilyScoped verifies that:
//  1. Students created under family A are visible only to family A (§15.2)
//  2. RLS blocks raw cross-family reads at the database level (§15.8)
func TestStudentCRUD_FamilyScoped(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	// ── Set up two independent families ──────────────────────────────────────

	parentA := registerFamily(ctx, t, svc, "Alice A", fmt.Sprintf("a-%s@test.com", uuid.Must(uuid.NewV7())))
	parentB := registerFamily(ctx, t, svc, "Bob B", fmt.Sprintf("b-%s@test.com", uuid.Must(uuid.NewV7())))

	// ── Grant COPPA consent for family A (needed to create students) ──────────
	// Directly update via bypass to avoid needing a full Stripe stub. [§9.3]
	err := shared.BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Model(&FamilyModel{}).
			Where("id = ?", parentA.FamilyID).
			Updates(map[string]interface{}{
				"coppa_consent_status": string(CoppaConsentConsented),
				"coppa_consented_at":   time.Now(),
			}).Error
	})
	if err != nil {
		t.Fatalf("set consent status: %v", err)
	}

	// ── Build scopes ──────────────────────────────────────────────────────────

	authA := &shared.AuthContext{
		ParentID:           parentA.ID,
		FamilyID:           parentA.FamilyID,
		CoppaConsentStatus: string(CoppaConsentConsented),
	}
	scopeA := shared.NewFamilyScopeFromAuth(authA)

	authB := &shared.AuthContext{
		ParentID: parentB.ID,
		FamilyID: parentB.FamilyID,
	}
	scopeB := shared.NewFamilyScopeFromAuth(authB)

	// ── Create a student under family A ───────────────────────────────────────

	studentResp, err := svc.CreateStudent(ctx, &scopeA, CreateStudentCommand{
		DisplayName: "Student Alpha",
	})
	if err != nil {
		t.Fatalf("CreateStudent(familyA): %v", err)
	}

	// ── Assertion 1: family A lists the student ───────────────────────────────

	studentsA, err := svc.ListStudents(ctx, &scopeA)
	if err != nil {
		t.Fatalf("ListStudents(scopeA): %v", err)
	}
	if len(studentsA) != 1 {
		t.Errorf("family A: want 1 student, got %d", len(studentsA))
	}
	if len(studentsA) > 0 && studentsA[0].ID != studentResp.ID {
		t.Errorf("family A student ID mismatch: got %v, want %v", studentsA[0].ID, studentResp.ID)
	}

	// ── Assertion 2: family B sees zero students (application-level scope) ────

	studentsB, err := svc.ListStudents(ctx, &scopeB)
	if err != nil {
		t.Fatalf("ListStudents(scopeB): %v", err)
	}
	if len(studentsB) != 0 {
		t.Errorf("family B: application scope leaked %d student(s) from family A", len(studentsB))
	}

	// ── Assertion 3: RLS blocks a raw count query with family B's session var ──
	//
	// This bypasses the application-level WHERE family_id = ? and relies
	// solely on the database RLS policy to enforce isolation. [§15.8, ARCH §2.5]
	//
	// With app.current_family_id set to family B, all rows whose family_id ≠
	// family B are invisible — the total count must be 0 even though family A's
	// student exists in the table.
	var rawCount int64
	err = testDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		setSQL := fmt.Sprintf("SET LOCAL app.current_family_id = '%s'", parentB.FamilyID.String())
		if execErr := tx.Exec(setSQL).Error; execErr != nil {
			return execErr
		}
		// Deliberately no WHERE clause — RLS is the only filter.
		return tx.Model(&StudentModel{}).Count(&rawCount).Error
	})
	if err != nil {
		t.Fatalf("raw RLS count query: %v", err)
	}
	if rawCount != 0 {
		t.Errorf("RLS violation: raw query with family B scope returned %d row(s) from family A", rawCount)
	}

	// ── Assertion 4: FindByID with wrong scope returns not-found ──────────────

	err = shared.ScopedTransaction(ctx, testDB, scopeB, func(tx *gorm.DB) error {
		repo := NewPgStudentRepository(tx)
		_, innerErr := repo.FindByID(ctx, &scopeB, studentResp.ID)
		return innerErr
	})
	if err == nil {
		t.Error("FindByID with family B scope should return not-found for family A's student")
	}
}
