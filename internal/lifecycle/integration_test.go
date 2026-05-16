//go:build integration

package lifecycle

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/lifecycle/...

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

// insertTestFamily is a helper that creates a minimal family row bypassing RLS.
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

// TestLifecycleIntegration_ExportRequestCreateAndFind verifies export request CRUD. [15-lifecycle §4.1]
func TestLifecycleIntegration_ExportRequestCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "Export Family")
	parentID := uuid.Must(uuid.NewV7())

	scope := shared.NewFamilyScopeFromID(familyID)
	repo := NewPgExportRequestRepository(testDB)

	req, err := repo.Create(ctx, &scope, &CreateExportRequest{
		RequestedBy:    parentID,
		Format:         ExportFormatJSON,
		IncludeDomains: []string{"learning", "social"},
	})
	if err != nil {
		t.Fatalf("Create export request: %v", err)
	}
	if req.Status != ExportStatusPending {
		t.Errorf("want status=pending, got %q", req.Status)
	}

	found, err := repo.FindByID(ctx, &scope, req.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.ID != req.ID {
		t.Errorf("ID mismatch: want %v, got %v", req.ID, found.ID)
	}
	if found.Format != ExportFormatJSON {
		t.Errorf("want format=json, got %q", found.Format)
	}
}

// TestLifecycleIntegration_DeletionRequestCreateAndCancel verifies deletion request
// lifecycle: create in grace_period, then cancel. [15-lifecycle §4.2]
func TestLifecycleIntegration_DeletionRequestCreateAndCancel(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "Deletion Family")
	parentID := uuid.Must(uuid.NewV7())

	scope := shared.NewFamilyScopeFromID(familyID)
	repo := NewPgDeletionRequestRepository(testDB)

	gracePeriodEnd := time.Now().Add(30 * 24 * time.Hour)
	req, err := repo.Create(ctx, &scope, &CreateDeletionRequest{
		RequestedBy:       parentID,
		DeletionType:      DeletionTypeFamily,
		GracePeriodEndsAt: gracePeriodEnd,
		Status:            DeletionStatusGracePeriod,
	})
	if err != nil {
		t.Fatalf("Create deletion request: %v", err)
	}
	if req.Status != DeletionStatusGracePeriod {
		t.Errorf("want status=grace_period, got %q", req.Status)
	}

	active, err := repo.FindActiveByFamily(ctx, &scope)
	if err != nil {
		t.Fatalf("FindActiveByFamily: %v", err)
	}
	if active == nil {
		t.Fatal("expected active deletion request, got nil")
	}

	if err := repo.Cancel(ctx, &scope, req.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	active2, err := repo.FindActiveByFamily(ctx, &scope)
	if err != nil {
		t.Fatalf("FindActiveByFamily after cancel: %v", err)
	}
	if active2 != nil {
		t.Error("expected no active deletion request after cancel")
	}
}

// TestLifecycleIntegration_ExportListByFamily verifies pagination on export requests. [15-lifecycle §5]
func TestLifecycleIntegration_ExportListByFamily(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "List Family")
	parentID := uuid.Must(uuid.NewV7())
	scope := shared.NewFamilyScopeFromID(familyID)
	repo := NewPgExportRequestRepository(testDB)

	for i := 0; i < 3; i++ {
		if _, err := repo.Create(ctx, &scope, &CreateExportRequest{
			RequestedBy:    parentID,
			Format:         ExportFormatCSV,
			IncludeDomains: []string{"billing"},
		}); err != nil {
			t.Fatalf("Create export %d: %v", i, err)
		}
	}

	exports, total, err := repo.ListByFamily(ctx, &scope, &PaginationParams{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListByFamily: %v", err)
	}
	if total < 3 {
		t.Errorf("want at least 3 exports, got total=%d", total)
	}
	if len(exports) < 3 {
		t.Errorf("want at least 3 exports in page, got %d", len(exports))
	}
}
