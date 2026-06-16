//go:build integration

package recs

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/recs/...

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

// seedIntTestFamily inserts a minimal iam_families row (bypassing RLS) for FK satisfaction.
func seedIntTestFamily(t *testing.T) shared.FamilyID {
	t.Helper()
	familyID := shared.NewFamilyID(uuid.Must(uuid.NewV7()))
	err := shared.BypassRLSTransaction(context.Background(), testDB, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO iam_families (id, display_name, primary_methodology_slug)
			 VALUES (?, 'Recs Test Family', 'charlotte-mason')`,
			familyID.UUID,
		).Error
	})
	if err != nil {
		t.Fatalf("seed iam_families: %v", err)
	}
	t.Cleanup(func() {
		testDB.Exec(`DELETE FROM recs_signals WHERE family_id = ?`, familyID.UUID)
		testDB.Exec(`DELETE FROM iam_families WHERE id = ?`, familyID.UUID)
	})
	return familyID
}

// TestRecsIntegration_SignalCreateAndFind verifies signals are family-scoped. [13-recs §4.1]
func TestRecsIntegration_SignalCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := seedIntTestFamily(t)
	repo := NewPgSignalRepository(testDB)
	scope := shared.NewFamilyScopeFromID(familyID.UUID)
	now := time.Now().UTC()

	err := repo.Create(ctx, NewSignal{
		FamilyID:        familyID,
		SignalType:      SignalActivityLogged,
		MethodologySlug: "charlotte-mason",
		Payload:         map[string]any{"activity_id": uuid.Must(uuid.NewV7()).String()},
		SignalDate:      now,
	})
	if err != nil {
		t.Fatalf("Create signal: %v", err)
	}

	signals, err := repo.FindByFamily(ctx, &scope, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("FindByFamily: %v", err)
	}
	if len(signals) == 0 {
		t.Error("expected at least 1 signal after creation")
	}
	if signals[0].FamilyID != familyID {
		t.Errorf("want FamilyID=%v, got %v", familyID, signals[0].FamilyID)
	}
}

// TestRecsIntegration_SignalDeleteByFamily verifies signals are deleted per family. [13-recs §4.1]
func TestRecsIntegration_SignalDeleteByFamily(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := seedIntTestFamily(t)
	repo := NewPgSignalRepository(testDB)
	scope := shared.NewFamilyScopeFromID(familyID.UUID)
	now := time.Now().UTC()

	if err := repo.Create(ctx, NewSignal{
		FamilyID:        familyID,
		SignalType:      SignalActivityLogged,
		MethodologySlug: "classical",
		Payload:         map[string]any{},
		SignalDate:      now,
	}); err != nil {
		t.Fatalf("Create signal: %v", err)
	}

	deleted, err := repo.DeleteByFamily(ctx, familyID)
	if err != nil {
		t.Fatalf("DeleteByFamily: %v", err)
	}
	if deleted == 0 {
		t.Error("expected at least 1 row deleted")
	}

	signals, err := repo.FindByFamily(ctx, &scope, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("FindByFamily after delete: %v", err)
	}
	if len(signals) != 0 {
		t.Errorf("expected 0 signals after delete, got %d", len(signals))
	}
}
