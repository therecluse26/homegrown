//go:build integration

package admin

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/admin/...

import (
	"context"
	"encoding/json"
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

// TestAdminIntegration_FeatureFlagCreateAndFind verifies feature flag CRUD. [16-admin §10]
func TestAdminIntegration_FeatureFlagCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgFeatureFlagRepository(testDB)
	adminID := uuid.Must(uuid.NewV7())

	flag, err := repo.Create(ctx, &CreateFlagInput{
		Key:         fmt.Sprintf("test-flag-%s", uuid.Must(uuid.NewV7())),
		Description: "Integration test flag",
		Enabled:     true,
	}, adminID)
	if err != nil {
		t.Fatalf("Create feature flag: %v", err)
	}
	if flag.Key == "" {
		t.Fatal("expected non-empty key")
	}
	if !flag.Enabled {
		t.Error("want Enabled=true")
	}

	found, err := repo.FindByKey(ctx, flag.Key)
	if err != nil {
		t.Fatalf("FindByKey: %v", err)
	}
	if found == nil {
		t.Fatal("expected flag, got nil")
	}
	if found.ID != flag.ID {
		t.Errorf("ID mismatch: want %v, got %v", flag.ID, found.ID)
	}
}

// TestAdminIntegration_FeatureFlagListAll verifies that ListAll returns flags. [16-admin §5]
func TestAdminIntegration_FeatureFlagListAll(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgFeatureFlagRepository(testDB)
	adminID := uuid.Must(uuid.NewV7())

	_, err := repo.Create(ctx, &CreateFlagInput{
		Key:         fmt.Sprintf("list-flag-%s", uuid.Must(uuid.NewV7())),
		Description: "List test flag",
		Enabled:     false,
	}, adminID)
	if err != nil {
		t.Fatalf("Create feature flag: %v", err)
	}

	flags, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(flags) == 0 {
		t.Error("expected at least one flag after creation")
	}
}

// TestAdminIntegration_AuditLogCreate verifies audit log entries can be created. [16-admin §11]
func TestAdminIntegration_AuditLogCreate(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgAuditLogRepository(testDB)
	adminID := uuid.Must(uuid.NewV7())

	entry, err := repo.Create(ctx, &CreateAuditLogEntry{
		AdminID:    adminID,
		Action:     "create_flag",
		TargetType: "feature_flag",
		Details:    json.RawMessage(`{"key":"test"}`),
		IPAddress:  "127.0.0.1",
		UserAgent:  "test-agent",
	})
	if err != nil {
		t.Fatalf("Create audit log: %v", err)
	}
	if entry.AdminID != adminID {
		t.Errorf("want AdminID=%v, got %v", adminID, entry.AdminID)
	}
	if entry.Action != "create_flag" {
		t.Errorf("want action=create_flag, got %q", entry.Action)
	}
}
