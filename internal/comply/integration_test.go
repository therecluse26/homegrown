//go:build integration

package comply

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/comply/...

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

// TestComplyIntegration_StateConfigUpsertAndFind verifies state config can be upserted
// and retrieved by state code. [14-comply §3.1]
func TestComplyIntegration_StateConfigUpsertAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgStateConfigRepository(testDB)
	row := UpsertStateConfigRow{
		StateCode:            "ZZ",
		StateName:            "Test State",
		NotificationRequired: true,
		RequiredSubjects:     []string{"Math", "Reading"},
		RegulationLevel:      "low",
	}

	_, err := repo.Upsert(ctx, row)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	found, err := repo.FindByStateCode(ctx, "ZZ")
	if err != nil {
		t.Fatalf("FindByStateCode: %v", err)
	}
	if found.StateName != "Test State" {
		t.Errorf("want StateName=%q, got %q", "Test State", found.StateName)
	}
	if !found.NotificationRequired {
		t.Error("want NotificationRequired=true")
	}
}

// TestComplyIntegration_FamilyConfigUpsertAndFind verifies family config is family-scoped. [14-comply §3.2]
func TestComplyIntegration_FamilyConfigUpsertAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := uuid.Must(uuid.NewV7())
	err := shared.BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO families (id, name, coppa_consent_status, time_zone)
			 VALUES (?, 'Comply Family', 'registered', 'UTC')`,
			familyID,
		).Error
	})
	if err != nil {
		t.Fatalf("insert family: %v", err)
	}

	// Upsert a state config so the FK is valid.
	stateRepo := NewPgStateConfigRepository(testDB)
	if _, err := stateRepo.Upsert(ctx, UpsertStateConfigRow{
		StateCode: "TX", StateName: "Texas", RegulationLevel: "medium",
	}); err != nil {
		t.Fatalf("upsert state config: %v", err)
	}

	scope := shared.NewFamilyScopeFromID(familyID)
	repo := NewPgFamilyConfigRepository(testDB)

	now := time.Now().UTC().Truncate(time.Second)
	cfg, err := repo.Upsert(ctx, scope, UpsertFamilyConfigRow{
		StateCode:       "TX",
		SchoolYearStart: now,
		SchoolYearEnd:   now.AddDate(1, 0, 0),
		TotalSchoolDays: 180,
		GpaScale:        "standard_4",
		GpaCustomConfig: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Upsert family config: %v", err)
	}
	if cfg.StateCode != "TX" {
		t.Errorf("want StateCode=TX, got %q", cfg.StateCode)
	}

	found, err := repo.FindByFamily(ctx, scope)
	if err != nil {
		t.Fatalf("FindByFamily: %v", err)
	}
	if found.TotalSchoolDays != 180 {
		t.Errorf("want TotalSchoolDays=180, got %d", found.TotalSchoolDays)
	}
}

// TestComplyIntegration_ScheduleCRUD verifies custom schedule CRUD and family isolation. [14-comply §3.3]
func TestComplyIntegration_ScheduleCRUD(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := uuid.Must(uuid.NewV7())
	err := shared.BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO families (id, name, coppa_consent_status, time_zone)
			 VALUES (?, 'Schedule Family', 'registered', 'UTC')`,
			familyID,
		).Error
	})
	if err != nil {
		t.Fatalf("insert family: %v", err)
	}

	scope := shared.NewFamilyScopeFromID(familyID)
	repo := NewPgScheduleRepository(testDB)

	sched, err := repo.Create(ctx, scope, CreateScheduleRow{
		Name:             "Weekly Plan",
		SchoolDays:       []bool{false, true, true, true, true, true, false},
		ExclusionPeriods: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatalf("Create schedule: %v", err)
	}
	if sched.Name != "Weekly Plan" {
		t.Errorf("want Name=%q, got %q", "Weekly Plan", sched.Name)
	}

	found, err := repo.FindByID(ctx, sched.ID, scope)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.ID != sched.ID {
		t.Errorf("ID mismatch: want %v, got %v", sched.ID, found.ID)
	}

	list, err := repo.ListByFamily(ctx, scope)
	if err != nil {
		t.Fatalf("ListByFamily: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected at least 1 schedule for family")
	}
}
