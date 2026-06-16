//go:build integration

package social

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/social/...

import (
	"context"
	"encoding/json"
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

// TestSocialIntegration_ProfileCreateAndFind verifies profile creation and retrieval. [05-social §3.2]
func TestSocialIntegration_ProfileCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "Social Family")
	repo := NewPgProfileRepository(testDB)

	profile := &Profile{
		FamilyID:        familyID,
		PrivacySettings: json.RawMessage(`{}`),
		LocationVisible: false,
	}
	if err := repo.Create(ctx, profile); err != nil {
		t.Fatalf("Create profile: %v", err)
	}

	found, err := repo.FindByFamilyID(ctx, familyID)
	if err != nil {
		t.Fatalf("FindByFamilyID: %v", err)
	}
	if found == nil {
		t.Fatal("expected profile, got nil")
	}
	if found.FamilyID != familyID {
		t.Errorf("want FamilyID=%v, got %v", familyID, found.FamilyID)
	}
}

// TestSocialIntegration_FriendshipCreateAndFind verifies friendship creation and bidirectional lookup. [05-social §3.2]
func TestSocialIntegration_FriendshipCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyA := insertTestFamily(ctx, t, "Social Family A")
	familyB := insertTestFamily(ctx, t, "Social Family B")
	repo := NewPgFriendshipRepository(testDB)

	friendship := &Friendship{
		RequesterFamilyID: familyA,
		AccepterFamilyID:  familyB,
		Status:            "pending",
	}
	if err := repo.Create(ctx, friendship); err != nil {
		t.Fatalf("Create friendship: %v", err)
	}
	if friendship.ID == uuid.Nil {
		t.Fatal("expected non-nil ID after create")
	}

	found, err := repo.FindBetween(ctx, familyA, familyB)
	if err != nil {
		t.Fatalf("FindBetween: %v", err)
	}
	if found == nil {
		t.Fatal("expected friendship, got nil")
	}
	if found.Status != "pending" {
		t.Errorf("want status=pending, got %q", found.Status)
	}
}
