//go:build integration

package plan

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/plan/...

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
			`INSERT INTO families (id, name, coppa_consent_status, time_zone)
			 VALUES (?, ?, 'registered', 'UTC')`,
			familyID, name,
		).Error
	})
	if err != nil {
		t.Fatalf("insert family %q: %v", name, err)
	}
	return familyID
}

// TestPlanIntegration_ScheduleItemCRUD verifies schedule item CRUD is family-scoped. [17-plan §4.1]
func TestPlanIntegration_ScheduleItemCRUD(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "Plan Family")
	scope := shared.NewFamilyScopeFromID(familyID)
	repo := NewPgScheduleItemRepository(testDB)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	item := &ScheduleItem{
		Title:     "Math Session",
		StartDate: today,
		Category:  ScheduleCategoryCustom,
	}
	if err := repo.Create(ctx, &scope, item); err != nil {
		t.Fatalf("Create schedule item: %v", err)
	}
	if item.ID == uuid.Nil {
		t.Fatal("expected non-nil ID after create")
	}

	found, err := repo.FindByID(ctx, &scope, item.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found == nil {
		t.Fatal("expected schedule item, got nil")
	}
	if found.Title != "Math Session" {
		t.Errorf("want title=%q, got %q", "Math Session", found.Title)
	}
}

// TestPlanIntegration_ScheduleItemFamilyIsolation verifies family B cannot see family A's items. [17-plan §4.1]
func TestPlanIntegration_ScheduleItemFamilyIsolation(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyA := insertTestFamily(ctx, t, "Plan Family A")
	familyB := insertTestFamily(ctx, t, "Plan Family B")
	scopeA := shared.NewFamilyScopeFromID(familyA)
	scopeB := shared.NewFamilyScopeFromID(familyB)
	repo := NewPgScheduleItemRepository(testDB)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	item := &ScheduleItem{
		Title:     "Secret Lesson",
		StartDate: today,
		Category:  ScheduleCategoryCustom,
	}
	if err := repo.Create(ctx, &scopeA, item); err != nil {
		t.Fatalf("Create schedule item: %v", err)
	}

	found, err := repo.FindByID(ctx, &scopeB, item.ID)
	if err != nil {
		t.Fatalf("FindByID (family B): %v", err)
	}
	if found != nil {
		t.Error("expected nil when reading family A item as family B")
	}
}
