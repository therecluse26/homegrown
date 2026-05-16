//go:build integration

package media

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/media/...

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

// TestMediaIntegration_UploadCreateAndFind verifies upload creation and family-scoped retrieval. [09-media §3.2]
func TestMediaIntegration_UploadCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "Media Family")
	scope := shared.NewFamilyScopeFromID(familyID)
	uploader := uuid.Must(uuid.NewV7())
	repo := NewPgUploadRepository(testDB)

	uploadID := uuid.Must(uuid.NewV7())
	upload, err := repo.Create(ctx, scope, &CreateUploadRow{
		ID:               uploadID,
		FamilyID:         familyID,
		UploadedBy:       uploader,
		Context:          UploadContextProfilePhoto,
		ContentType:      "image/jpeg",
		OriginalFilename: "avatar.jpg",
		StorageKey:       fmt.Sprintf("uploads/%s/avatar.jpg", uploadID),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Create upload: %v", err)
	}
	if upload.Status != UploadStatusPending {
		t.Errorf("want status=pending, got %q", upload.Status)
	}

	found, err := repo.FindByID(ctx, scope, uploadID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found == nil {
		t.Fatal("expected upload, got nil")
	}
	if found.OriginalFilename != "avatar.jpg" {
		t.Errorf("want filename=avatar.jpg, got %q", found.OriginalFilename)
	}
}

// TestMediaIntegration_UploadUpdateStatus verifies status transitions. [09-media §3.2]
func TestMediaIntegration_UploadUpdateStatus(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "Media Family 2")
	scope := shared.NewFamilyScopeFromID(familyID)
	uploader := uuid.Must(uuid.NewV7())
	repo := NewPgUploadRepository(testDB)

	uploadID := uuid.Must(uuid.NewV7())
	_, err := repo.Create(ctx, scope, &CreateUploadRow{
		ID:               uploadID,
		FamilyID:         familyID,
		UploadedBy:       uploader,
		Context:          UploadContextPostAttachment,
		ContentType:      "image/png",
		OriginalFilename: "photo.png",
		StorageKey:       fmt.Sprintf("uploads/%s/photo.png", uploadID),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Create upload: %v", err)
	}

	updated, err := repo.UpdateStatus(ctx, uploadID, UploadStatusUploaded, nil)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if updated.Status != UploadStatusUploaded {
		t.Errorf("want status=uploaded, got %q", updated.Status)
	}
}

// TestMediaIntegration_UploadFamilyIsolation verifies family B cannot read family A's upload. [09-media §3.2]
func TestMediaIntegration_UploadFamilyIsolation(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyA := insertTestFamily(ctx, t, "Media Family A")
	familyB := insertTestFamily(ctx, t, "Media Family B")
	scopeA := shared.NewFamilyScopeFromID(familyA)
	scopeB := shared.NewFamilyScopeFromID(familyB)
	repo := NewPgUploadRepository(testDB)

	uploadID := uuid.Must(uuid.NewV7())
	_, err := repo.Create(ctx, scopeA, &CreateUploadRow{
		ID:               uploadID,
		FamilyID:         familyA,
		UploadedBy:       uuid.Must(uuid.NewV7()),
		Context:          UploadContextProfilePhoto,
		ContentType:      "image/jpeg",
		OriginalFilename: "secret.jpg",
		StorageKey:       fmt.Sprintf("uploads/%s/secret.jpg", uploadID),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Create upload: %v", err)
	}

	found, err := repo.FindByID(ctx, scopeB, uploadID)
	if err != nil {
		t.Fatalf("FindByID (family B): %v", err)
	}
	if found != nil {
		t.Error("expected nil when reading family A upload as family B")
	}
}
