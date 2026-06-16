//go:build integration

package learn

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/learn/...

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

// seedTestPublisher inserts a minimal mkt_publishers row and returns its UUID.
// learn_*_defs tables FK to mkt_publishers(id). [06-learn §3.2]
func seedTestPublisher(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	slug := fmt.Sprintf("learn-test-pub-%s", id)
	if err := testDB.Exec(
		`INSERT INTO mkt_publishers (id, name, slug) VALUES (?, 'Learn Test Publisher', ?)`,
		id, slug,
	).Error; err != nil {
		t.Fatalf("seed mkt_publishers: %v", err)
	}
	t.Cleanup(func() { testDB.Exec(`DELETE FROM mkt_publishers WHERE id = ?`, id) })
	return id
}

// insertTestStudent creates a minimal student row bypassing RLS.
func insertTestStudent(ctx context.Context, t *testing.T, familyID uuid.UUID) uuid.UUID {
	t.Helper()
	studentID := uuid.Must(uuid.NewV7())
	err := shared.BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO iam_students (id, family_id, display_name)
			 VALUES (?, ?, 'Test Student')`,
			studentID, familyID,
		).Error
	})
	if err != nil {
		t.Fatalf("insert student: %v", err)
	}
	return studentID
}

// TestLearnIntegration_ActivityDefCRUD verifies activity definition can be created
// and retrieved globally (not family-scoped). [06-learn §4.1]
func TestLearnIntegration_ActivityDefCRUD(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgActivityDefRepository(testDB)
	publisherID := seedTestPublisher(t)

	def := &ActivityDefModel{
		PublisherID:  publisherID,
		Title:        "Nature Journaling Basics",
		SubjectTags:  StringArray{"science", "art"},
		Attachments:  json.RawMessage(`[]`),
		IsActive:     true,
	}
	if err := repo.Create(ctx, def); err != nil {
		t.Fatalf("Create activity def: %v", err)
	}
	if def.ID == uuid.Nil {
		t.Fatal("expected non-nil ID after create")
	}

	found, err := repo.FindByID(ctx, def.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Title != "Nature Journaling Basics" {
		t.Errorf("want title=%q, got %q", "Nature Journaling Basics", found.Title)
	}
}

// TestLearnIntegration_JournalEntryCRUD verifies journal entries are family-scoped. [06-learn §4.3]
func TestLearnIntegration_JournalEntryCRUD(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "Learn Family")
	studentID := insertTestStudent(ctx, t, familyID)
	scope := shared.NewFamilyScopeFromID(familyID)

	repo := NewPgJournalEntryRepository(testDB)
	today := time.Now().UTC()

	entry := &JournalEntryModel{
		StudentID:   studentID,
		EntryType:   "daily",
		Content:     "Today we learned about prime numbers.",
		SubjectTags: StringArray{"math"},
		Attachments: json.RawMessage(`[]`),
		EntryDate:   today,
	}
	if err := repo.Create(ctx, &scope, entry); err != nil {
		t.Fatalf("Create journal entry: %v", err)
	}
	if entry.ID == uuid.Nil {
		t.Fatal("expected non-nil ID after create")
	}

	found, err := repo.FindByID(ctx, &scope, entry.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Content != "Today we learned about prime numbers." {
		t.Errorf("content mismatch: got %q", found.Content)
	}
}

// TestLearnIntegration_JournalEntryCrossFamily verifies family B cannot read family A's entries. [06-learn §4.3]
func TestLearnIntegration_JournalEntryCrossFamily(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyA := insertTestFamily(ctx, t, "Learn Family A")
	familyB := insertTestFamily(ctx, t, "Learn Family B")
	studentA := insertTestStudent(ctx, t, familyA)
	scopeA := shared.NewFamilyScopeFromID(familyA)
	scopeB := shared.NewFamilyScopeFromID(familyB)

	repo := NewPgJournalEntryRepository(testDB)

	entry := &JournalEntryModel{
		StudentID:   studentA,
		EntryType:   "daily",
		Content:     "Secret family A entry.",
		SubjectTags: StringArray{},
		Attachments: json.RawMessage(`[]`),
		EntryDate:   time.Now().UTC(),
	}
	if err := repo.Create(ctx, &scopeA, entry); err != nil {
		t.Fatalf("Create journal entry: %v", err)
	}

	// Family B should not be able to read family A's entry.
	_, err := repo.FindByID(ctx, &scopeB, entry.ID)
	if err == nil {
		t.Error("expected error reading family A entry as family B, got nil")
	}
}
