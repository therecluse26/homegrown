//go:build integration

package billing

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/billing/...
//
// Tests spin up a postgis/postgis Docker container via testcontainers-go,
// run all goose migrations, and verify invariants against a real database.
//
// Skipped automatically if Docker is unavailable.

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

// TestBillingIntegration_CustomerUpsertAndFind verifies that a Hyperswitch customer
// can be upserted and retrieved by family_id. [10-billing §3.2]
func TestBillingIntegration_CustomerUpsertAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgCustomerRepository(testDB)
	familyID := uuid.Must(uuid.NewV7())

	customer, err := repo.Upsert(ctx, familyID, UpsertCustomerRow{
		HyperswitchCustomerID: fmt.Sprintf("hs_cust_%s", familyID),
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if customer.FamilyID != familyID {
		t.Errorf("want FamilyID=%v, got %v", familyID, customer.FamilyID)
	}

	found, err := repo.FindByFamily(ctx, familyID)
	if err != nil {
		t.Fatalf("FindByFamily: %v", err)
	}
	if found.HyperswitchCustomerID != customer.HyperswitchCustomerID {
		t.Errorf("want customer_id=%q, got %q", customer.HyperswitchCustomerID, found.HyperswitchCustomerID)
	}
}

// TestBillingIntegration_SubscriptionCreateAndFind verifies subscription CRUD
// and that FindByFamily returns the correct record. [10-billing §3.2]
func TestBillingIntegration_SubscriptionCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	// We need a family to exist first — bypass RLS to insert directly.
	familyID := uuid.Must(uuid.NewV7())
	err := shared.BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO families (id, name, coppa_consent_status, time_zone)
			 VALUES (?, 'Test Family', 'registered', 'UTC')`,
			familyID,
		).Error
	})
	if err != nil {
		t.Fatalf("insert family: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	repo := NewPgSubscriptionRepository(testDB)

	sub, err := repo.Create(ctx, CreateSubscriptionRow{
		FamilyID:                  familyID,
		HyperswitchSubscriptionID: fmt.Sprintf("hs_sub_%s", familyID),
		HyperswitchCustomerID:     fmt.Sprintf("hs_cust_%s", familyID),
		Tier:                      "premium",
		Status:                    "active",
		BillingInterval:           "monthly",
		CurrentPeriodStart:        now,
		CurrentPeriodEnd:          now.AddDate(0, 1, 0),
		AmountCents:               999,
		Currency:                  "usd",
		HyperswitchPriceID:        "price_monthly_test",
	})
	if err != nil {
		t.Fatalf("Create subscription: %v", err)
	}

	// Create a minimal scope for the query.
	scope := shared.NewFamilyScopeFromID(familyID)
	found, err := repo.FindByFamily(ctx, scope)
	if err != nil {
		t.Fatalf("FindByFamily: %v", err)
	}
	if found.ID != sub.ID {
		t.Errorf("want subscription ID=%v, got %v", sub.ID, found.ID)
	}
	if found.Status != "active" {
		t.Errorf("want status=active, got %q", found.Status)
	}
}

// TestBillingIntegration_TransactionListEmpty verifies that ListByFamily returns an
// empty slice (not an error) when no transactions exist for a family. [10-billing §3.2]
func TestBillingIntegration_TransactionListEmpty(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgTransactionRepository(testDB)
	familyID := uuid.Must(uuid.NewV7())
	scope := shared.NewFamilyScopeFromID(familyID)

	txns, err := repo.ListByFamily(ctx, scope, &TransactionListParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListByFamily: %v", err)
	}
	if len(txns) != 0 {
		t.Errorf("expected 0 transactions for new family, got %d", len(txns))
	}
}
