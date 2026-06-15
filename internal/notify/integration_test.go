//go:build integration

package notify

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/notify/...

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

// TestNotifyIntegration_NotificationCreateAndList verifies notification CRUD and family scoping. [08-notify §3.2]
func TestNotifyIntegration_NotificationCreateAndList(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "Notify Family")
	scope := shared.NewFamilyScopeFromID(familyID)
	repo := NewPgNotificationRepository(testDB)

	notif, err := repo.Create(ctx, CreateNotification{
		FamilyID:         familyID,
		NotificationType: TypeFriendRequestSent,
		Category:         CategorySocial,
		Title:            "New Friend Request",
		Body:             "Someone wants to connect.",
		Metadata:         json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Create notification: %v", err)
	}
	if notif.ID == uuid.Nil {
		t.Fatal("expected non-nil ID after create")
	}
	if notif.IsRead {
		t.Error("want IsRead=false on creation")
	}

	list, err := repo.List(ctx, &NotificationListParams{}, &scope)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected at least 1 notification after creation")
	}
}

// TestNotifyIntegration_CountUnreadAndMarkRead verifies unread counting and mark-read. [08-notify §3.2]
func TestNotifyIntegration_CountUnreadAndMarkRead(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	familyID := insertTestFamily(ctx, t, "Notify Family 2")
	scope := shared.NewFamilyScopeFromID(familyID)
	repo := NewPgNotificationRepository(testDB)

	notif, err := repo.Create(ctx, CreateNotification{
		FamilyID:         familyID,
		NotificationType: TypeActivityStreak,
		Category:         CategoryLearning,
		Title:            "3-day streak!",
		Body:             "Keep it up.",
		Metadata:         json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Create notification: %v", err)
	}

	count, err := repo.CountUnread(ctx, &scope)
	if err != nil {
		t.Fatalf("CountUnread: %v", err)
	}
	if count == 0 {
		t.Error("expected at least 1 unread notification")
	}

	marked, err := repo.MarkRead(ctx, notif.ID, &scope)
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if !marked {
		t.Error("expected MarkRead to return true")
	}

	countAfter, err := repo.CountUnread(ctx, &scope)
	if err != nil {
		t.Fatalf("CountUnread after mark: %v", err)
	}
	if countAfter >= count {
		t.Errorf("expected unread count to decrease, got %d → %d", count, countAfter)
	}
}
