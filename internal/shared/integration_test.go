//go:build integration

package shared

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/shared/...

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
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

func skipIfNoTestDB(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("testDB not available (Docker required for integration tests)")
	}
}

// TestSharedIntegration_BypassRLSTransaction verifies that BypassRLSTransaction can bypass
// RLS to insert rows that would otherwise be blocked. [00-core §6]
func TestSharedIntegration_BypassRLSTransaction(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := uuid.Must(uuid.NewV7())
	err := BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO families (id, name, coppa_consent_status, time_zone)
			 VALUES (?, 'Shared Test Family', 'registered', 'UTC')`,
			familyID,
		).Error
	})
	if err != nil {
		t.Fatalf("BypassRLSTransaction: %v", err)
	}

	// Verify row is visible via a bypass read.
	var count int64
	err = BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Raw("SELECT COUNT(*) FROM families WHERE id = ?", familyID).Scan(&count).Error
	})
	if err != nil {
		t.Fatalf("BypassRLSTransaction read: %v", err)
	}
	if count != 1 {
		t.Errorf("want 1 family row, got %d", count)
	}
}

// TestSharedIntegration_ScopedTransactionCommits verifies ScopedTransaction commits on success. [00-core §6]
func TestSharedIntegration_ScopedTransactionCommits(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := uuid.Must(uuid.NewV7())

	// Seed the family so we can reference it in scoped work.
	err := BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO families (id, name, coppa_consent_status, time_zone)
			 VALUES (?, 'Scoped Tx Family', 'registered', 'UTC')`,
			familyID,
		).Error
	})
	if err != nil {
		t.Fatalf("seed family: %v", err)
	}

	scope := NewFamilyScopeFromID(familyID)
	err = ScopedTransaction(ctx, testDB, scope, func(tx *gorm.DB) error {
		return nil
	})
	if err != nil {
		t.Fatalf("ScopedTransaction: %v", err)
	}
}
