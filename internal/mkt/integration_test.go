//go:build integration

package mkt

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/mkt/...

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

// seedPublisher creates a minimal publisher row for FK satisfaction.
func seedPublisher(t *testing.T, ctx context.Context, slug string) uuid.UUID {
	t.Helper()
	repo := NewPgPublisherRepository(testDB)
	pub, err := repo.Create(ctx, CreatePublisher{
		Name: "Test Publisher",
		Slug: slug,
	})
	if err != nil {
		t.Fatalf("seed publisher: %v", err)
	}
	return pub.ID
}

// TestMktIntegration_CreatorCreateAndFind verifies creator CRUD. [07-mkt §3.2]
func TestMktIntegration_CreatorCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	parentID := uuid.Must(uuid.NewV7())
	repo := NewPgCreatorRepository(testDB)

	creator, err := repo.Create(ctx, CreateCreator{
		ParentID:  parentID,
		StoreName: "Math & More",
	})
	if err != nil {
		t.Fatalf("Create creator: %v", err)
	}
	if creator.ID == uuid.Nil {
		t.Fatal("expected non-nil ID after create")
	}
	if creator.OnboardingStatus != "pending" {
		t.Errorf("want onboarding_status=pending, got %q", creator.OnboardingStatus)
	}

	found, err := repo.GetByParentID(ctx, parentID)
	if err != nil {
		t.Fatalf("GetByParentID: %v", err)
	}
	if found.ID != creator.ID {
		t.Errorf("want ID=%v, got %v", creator.ID, found.ID)
	}
}

// TestMktIntegration_ListingCreateAndFind verifies listing creation and retrieval. [07-mkt §3.2]
func TestMktIntegration_ListingCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	parentID := uuid.Must(uuid.NewV7())
	creatorRepo := NewPgCreatorRepository(testDB)
	creator, err := creatorRepo.Create(ctx, CreateCreator{
		ParentID:  parentID,
		StoreName: "Listing Test Store",
	})
	if err != nil {
		t.Fatalf("Create creator: %v", err)
	}

	publisherID := seedPublisher(t, ctx, fmt.Sprintf("test-pub-%s", uuid.Must(uuid.NewV7())))

	listingRepo := NewPgListingRepository(testDB)
	listing, err := listingRepo.Create(ctx, CreateListing{
		CreatorID:   creator.ID,
		PublisherID: publisherID,
		Title:       "Introduction to Latin",
		Description: "A classical Latin curriculum for all ages.",
		PriceCents:  1999,
		ContentType: "curriculum",
	})
	if err != nil {
		t.Fatalf("Create listing: %v", err)
	}
	if listing.ID == uuid.Nil {
		t.Fatal("expected non-nil listing ID")
	}
	if listing.Status != "draft" {
		t.Errorf("want status=draft, got %q", listing.Status)
	}

	found, err := listingRepo.GetByID(ctx, listing.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if found.Title != "Introduction to Latin" {
		t.Errorf("want title=%q, got %q", "Introduction to Latin", found.Title)
	}
}

// TestMktIntegration_CartAddAndRetrieve verifies cart item management. [07-mkt §3.2]
func TestMktIntegration_CartAddAndRetrieve(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	// Cart is family-scoped; use a fresh family UUID (no FK to families in cart table required).
	familyID := uuid.Must(uuid.NewV7())
	scope := shared.NewFamilyScopeFromID(familyID)

	parentID := uuid.Must(uuid.NewV7())
	creatorRepo := NewPgCreatorRepository(testDB)
	creator, err := creatorRepo.Create(ctx, CreateCreator{
		ParentID:  parentID,
		StoreName: "Cart Test Store",
	})
	if err != nil {
		t.Fatalf("Create creator: %v", err)
	}

	publisherID := seedPublisher(t, ctx, fmt.Sprintf("cart-pub-%s", uuid.Must(uuid.NewV7())))
	listingRepo := NewPgListingRepository(testDB)
	listing, err := listingRepo.Create(ctx, CreateListing{
		CreatorID:   creator.ID,
		PublisherID: publisherID,
		Title:       "Cart Listing",
		Description: "Test.",
		PriceCents:  500,
		ContentType: "worksheet",
	})
	if err != nil {
		t.Fatalf("Create listing: %v", err)
	}

	cartRepo := NewPgCartRepository(testDB)
	if err := cartRepo.AddItem(ctx, listing.ID, scope); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	items, err := cartRepo.GetItems(ctx, scope)
	if err != nil {
		t.Fatalf("GetItems: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected at least 1 cart item after AddItem")
	}
}

