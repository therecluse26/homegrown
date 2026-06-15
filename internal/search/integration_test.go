//go:build integration

package search

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/search/...

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

// TestSearchIntegration_SearchFamiliesEmpty verifies SearchFamilies returns empty (not error)
// when no matching profiles exist. [12-search §6.1]
func TestSearchIntegration_SearchFamiliesEmpty(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgSocialSearchRepository(testDB)
	searcher := uuid.Must(uuid.NewV7())

	results, err := repo.SearchFamilies(ctx, searcher, "nonexistent-query-xyz", 10, nil)
	if err != nil {
		t.Fatalf("SearchFamilies: %v", err)
	}
	if results == nil {
		t.Error("expected non-nil slice, got nil")
	}
}

// TestSearchIntegration_SearchListingsEmpty verifies SearchListings returns empty (not error)
// when no listings exist. [12-search §6.2]
func TestSearchIntegration_SearchListingsEmpty(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgMarketplaceSearchRepository(testDB)

	results, err := repo.SearchListings(ctx, "homeschool", nil, SearchSortRelevance, 10, nil)
	if err != nil {
		t.Fatalf("SearchListings: %v", err)
	}
	if results == nil {
		t.Error("expected non-nil results, got nil")
	}
}

// TestSearchIntegration_SearchLearningEmpty verifies SearchLearning returns empty (not error)
// when no learning data exists for a family. [12-search §6.3]
func TestSearchIntegration_SearchLearningEmpty(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgLearningSearchRepository(testDB)
	familyID := uuid.Must(uuid.NewV7())
	scope := shared.NewFamilyScopeFromID(familyID)

	results, err := repo.SearchLearning(ctx, &scope, "math", nil, 10, nil)
	if err != nil {
		t.Fatalf("SearchLearning: %v", err)
	}
	if results == nil {
		t.Error("expected non-nil slice, got nil")
	}
}
