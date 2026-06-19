//go:build integration

package safety

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/safety/...

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time" // used by startTestDB wait timeout

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDB *gorm.DB

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

func startTestDB(ctx context.Context) (*gorm.DB, func(), error) {
	req := testcontainers.ContainerRequest{
		Image: "postgis/postgis:18-3.6",
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

func skipIfNoTestDB(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("testDB not available (Docker required for integration tests)")
	}
}

// insertTestFamily creates a minimal family row bypassing RLS.
func insertTestFamily(ctx context.Context, t *testing.T, name string) uuid.UUID {
	t.Helper()
	familyID := uuid.Must(uuid.NewV7())
	err := shared.BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO iam_families (id, display_name, primary_methodology_slug)
			 VALUES (?, ?, 'charlotte-mason')`,
			familyID, name,
		).Error
	})
	if err != nil {
		t.Fatalf("insert family %q: %v", name, err)
	}
	return familyID
}

// TestSafetyIntegration_ReportCreateAndFind verifies safety report CRUD. [11-safety §4.1]
func TestSafetyIntegration_ReportCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	reporterFamily := insertTestFamily(ctx, t, "Reporter Family")
	targetFamily := insertTestFamily(ctx, t, "Target Family")
	parentID := uuid.Must(uuid.NewV7())
	scope := shared.NewFamilyScopeFromID(reporterFamily)

	repo := NewPgReportRepository(testDB)
	targetID := uuid.Must(uuid.NewV7())

	report, err := repo.Create(ctx, scope, CreateReportRow{
		ReporterParentID: parentID,
		TargetType:       "post",
		TargetID:         targetID,
		TargetFamilyID:   &targetFamily,
		Category:         "harassment",
		Priority:         "normal",
	})
	if err != nil {
		t.Fatalf("Create report: %v", err)
	}
	if report.Status != "pending" {
		t.Errorf("want status=pending, got %q", report.Status)
	}

	found, err := repo.FindByID(ctx, scope, report.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Category != "harassment" {
		t.Errorf("want category=harassment, got %q", found.Category)
	}
}

// TestSafetyIntegration_ContentFlagCreateAndList verifies content flag repository. [11-safety §7]
func TestSafetyIntegration_ContentFlagCreateAndList(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgContentFlagRepository(testDB)
	targetID := uuid.Must(uuid.NewV7())
	targetFamily := insertTestFamily(ctx, t, "Flag Target Family")

	flag, err := repo.Create(ctx, CreateContentFlagRow{
		Source:         "automated",
		TargetType:     "upload",
		TargetID:       targetID,
		TargetFamilyID: &targetFamily,
		FlagType:       "explicit_content",
		AutoRejected:   true,
	})
	if err != nil {
		t.Fatalf("Create content flag: %v", err)
	}
	if flag.FlagType != "explicit_content" {
		t.Errorf("want flag_type=explicit_content, got %q", flag.FlagType)
	}
	if !flag.AutoRejected {
		t.Error("want auto_rejected=true")
	}
}

// TestSafetyIntegration_AccountStatusGetOrCreate verifies account status is created
// on demand and defaults to "active". [11-safety §9]
func TestSafetyIntegration_AccountStatusGetOrCreate(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "Status Family")
	repo := NewPgAccountStatusRepository(testDB)

	status, err := repo.GetOrCreate(ctx, familyID)
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if status.Status != "active" {
		t.Errorf("want status=active, got %q", status.Status)
	}
	if status.FamilyID != familyID {
		t.Errorf("want FamilyID=%v, got %v", familyID, status.FamilyID)
	}

	// Idempotent: second call should not error.
	status2, err := repo.GetOrCreate(ctx, familyID)
	if err != nil {
		t.Fatalf("GetOrCreate (2nd): %v", err)
	}
	if status2.FamilyID != familyID {
		t.Errorf("want same FamilyID on 2nd call, got %v", status2.FamilyID)
	}
}
