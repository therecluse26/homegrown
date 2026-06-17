// cmd/seed/main.go — Agent database seeder.
//
// Targets the "homegrown_agent" database by default (override with --db <name>).
// Reads connection parameters from the .env FILE (godotenv.Read, not Load) so
// shell-exported DATABASE_URL values are never used.
//
// Usage:
//
//	go run ./cmd/seed/          # seeds homegrown_agent
//	go run ./cmd/seed/ --db homegrown   # seeds dev database
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ─── Deterministic seed UUIDs ─────────────────────────────────────────────────
// All seed row IDs are hardcoded so the seeder is idempotent across runs.
const (
	// IAM
	seedFamilyID     = "01900000-0000-7000-8000-000000000001"
	friendFamilyID   = "01900000-0000-7000-8000-000000000002"
	platformFamilyID = "01900000-0000-7000-8000-000000000003"
	seedParentID     = "01900000-0000-7000-8000-000000000011"
	friendParentID   = "01900000-0000-7000-8000-000000000012"
	adminParentID    = "01900000-0000-7000-8000-000000000013"
	emmaStudentID    = "01900000-0000-7000-8000-000000000021"
	jamesStudentID   = "01900000-0000-7000-8000-000000000022"
	coppaAuditID     = "01900000-0000-7000-8000-000000000031"

	// IAM Extended
	coParentInviteID  = "01900000-0000-7000-8000-000000000032"
	studentSession1ID = "01900000-0000-7000-8000-000000000033"

	// Onboard
	seedWizardProgressID  = "01900000-0000-7000-8000-000000000131"
	adminWizardProgressID = "01900000-0000-7000-8000-000000000132"

	// Social
	friendshipID     = "01900000-0000-7000-8000-000000000041"
	groupID          = "01900000-0000-7000-8000-000000000051"
	seedGroupMemID   = "01900000-0000-7000-8000-000000000052"
	friendGroupMemID = "01900000-0000-7000-8000-000000000053"
	post1ID          = "01900000-0000-7000-8000-000000000061"
	post2ID          = "01900000-0000-7000-8000-000000000062"
	post3ID          = "01900000-0000-7000-8000-000000000063"
	comment1ID       = "01900000-0000-7000-8000-000000000071"
	comment2ID       = "01900000-0000-7000-8000-000000000072"
	postLike1ID      = "01900000-0000-7000-8000-000000000081"
	conversationID   = "01900000-0000-7000-8000-000000000091"
	msg1ID           = "01900000-0000-7000-8000-000000000101"
	msg2ID           = "01900000-0000-7000-8000-000000000102"
	msg3ID           = "01900000-0000-7000-8000-000000000103"
	eventID          = "01900000-0000-7000-8000-000000000111"
	rsvpID           = "01900000-0000-7000-8000-000000000112"

	// Social Extended
	pinnedPost1ID = "01900000-0000-7000-8000-000000000121"

	// Marketplace
	seedPublisherID = "01900000-0000-7000-8000-000000000201"
	seedCreatorID   = "01900000-0000-7000-8000-000000000202"
	listing1ID      = "01900000-0000-7000-8000-000000000211"
	listing2ID      = "01900000-0000-7000-8000-000000000212"
	listing3ID      = "01900000-0000-7000-8000-000000000213"
	listing4ID      = "01900000-0000-7000-8000-000000000214"
	listing5ID      = "01900000-0000-7000-8000-000000000215"
	purchase1ID     = "01900000-0000-7000-8000-000000000221"
	purchase2ID     = "01900000-0000-7000-8000-000000000222"
	review1ID       = "01900000-0000-7000-8000-000000000231"
	cartItem1ID     = "01900000-0000-7000-8000-000000000241"

	friendCreatorID  = "01900000-0000-7000-8000-000000000203"
	friendPublisherID = "01900000-0000-7000-8000-000000000204"
	friendListingID  = "01900000-0000-7000-8000-000000000216"
	seedPurchaseID   = "01900000-0000-7000-8000-000000000223"

	// Platform Content (deterministic; used by content ingestion CLI)
	// PlatformCreatorID MUST match sources.PlatformCreatorID in cmd/seed-content.
	platformContentCreatorID   = "01900000-0000-7000-8000-000000000206"
	platformContentPublisherID = "018f1234-0000-7000-8000-000000000002"

	// Marketplace Extended
	mktCuratedItem1ID = "01900000-0000-7000-8000-000000000242"
	mktCuratedItem2ID = "01900000-0000-7000-8000-000000000243"
	mktCuratedItem3ID = "01900000-0000-7000-8000-000000000244"
	mktCuratedItem4ID = "01900000-0000-7000-8000-000000000245"
	mktListingFile1ID = "01900000-0000-7000-8000-000000000246"

	// Learn
	activityDef1ID  = "01900000-0000-7000-8000-000000000301"
	activityDef2ID  = "01900000-0000-7000-8000-000000000302"
	activityDef3ID  = "01900000-0000-7000-8000-000000000303"
	readingItem1ID  = "01900000-0000-7000-8000-000000000311"
	readingItem2ID  = "01900000-0000-7000-8000-000000000312"
	readingItem3ID  = "01900000-0000-7000-8000-000000000313"
	actLog1ID       = "01900000-0000-7000-8000-000000000321"
	actLog2ID       = "01900000-0000-7000-8000-000000000322"
	actLog3ID       = "01900000-0000-7000-8000-000000000323"
	actLog4ID       = "01900000-0000-7000-8000-000000000324"
	actLog5ID       = "01900000-0000-7000-8000-000000000325"
	actLog6ID       = "01900000-0000-7000-8000-000000000326"
	actLog7ID       = "01900000-0000-7000-8000-000000000327"
	actLog8ID       = "01900000-0000-7000-8000-000000000328"
	actLog9ID       = "01900000-0000-7000-8000-000000000329"
	actLog10ID      = "01900000-0000-7000-8000-000000000330"
	readProg1ID     = "01900000-0000-7000-8000-000000000341"
	readProg2ID     = "01900000-0000-7000-8000-000000000342"
	readProg3ID     = "01900000-0000-7000-8000-000000000343"
	journal1ID      = "01900000-0000-7000-8000-000000000351"
	journal2ID      = "01900000-0000-7000-8000-000000000352"
	readingList1ID  = "01900000-0000-7000-8000-000000000361"
	gradingScale1ID = "01900000-0000-7000-8000-000000000371"

	// Learn Extended
	assessmentDef1ID    = "01900000-0000-7000-8000-000000000304"
	assessmentDef2ID    = "01900000-0000-7000-8000-000000000305"
	videoDef1ID         = "01900000-0000-7000-8000-000000000306"
	question1ID         = "01900000-0000-7000-8000-000000000372"
	question2ID         = "01900000-0000-7000-8000-000000000373"
	question3ID         = "01900000-0000-7000-8000-000000000374"
	question4ID         = "01900000-0000-7000-8000-000000000375"
	question5ID         = "01900000-0000-7000-8000-000000000376"
	quizDef1ID          = "01900000-0000-7000-8000-000000000381"
	sequenceDef1ID      = "01900000-0000-7000-8000-000000000385"
	sequenceItem1ID     = "01900000-0000-7000-8000-000000000386"
	sequenceItem2ID     = "01900000-0000-7000-8000-000000000387"
	sequenceItem3ID     = "01900000-0000-7000-8000-000000000388"
	studentAssign1ID    = "01900000-0000-7000-8000-000000000394"
	studentAssign2ID    = "01900000-0000-7000-8000-000000000395"
	progressSnapshot1ID = "01900000-0000-7000-8000-000000000396"
	progressSnapshot2ID = "01900000-0000-7000-8000-000000000397"

	// Notifications
	notif1ID = "01900000-0000-7000-8000-000000000401"
	notif2ID = "01900000-0000-7000-8000-000000000402"
	notif3ID = "01900000-0000-7000-8000-000000000403"
	notif4ID = "01900000-0000-7000-8000-000000000404"
	notif5ID = "01900000-0000-7000-8000-000000000405"
	pref1ID  = "01900000-0000-7000-8000-000000000411"
	pref2ID  = "01900000-0000-7000-8000-000000000412"

	// Billing
	subID = "01900000-0000-7000-8000-000000000501"
	txnID = "01900000-0000-7000-8000-000000000502"

	// Media
	media1ID = "01900000-0000-7000-8000-000000000601"
	media2ID = "01900000-0000-7000-8000-000000000602"

	// Safety Full
	safetyBotSignal1ID = "01900000-0000-7000-8000-000000000704"
	safetyBotSignal2ID = "01900000-0000-7000-8000-000000000705"
	safetyFlag1ID      = "01900000-0000-7000-8000-000000000701"
	safetyReport1ID    = "01900000-0000-7000-8000-000000000700"
	safetyModAction1ID = "01900000-0000-7000-8000-000000000702"
	safetyAppeal1ID    = "01900000-0000-7000-8000-000000000703"

	// Recommendations
	rec1ID    = "01900000-0000-7000-8000-000000000701" //nolint:deadcode // shadows safety IDs intentionally in different tables
	rec2ID    = "01900000-0000-7000-8000-000000000702"
	rec3ID    = "01900000-0000-7000-8000-000000000703"
	recPrefID = "01900000-0000-7000-8000-000000000711"

	// Learner Profile (domain 18)
	learnerProfile1ID = "01900000-0000-7000-8000-000000000720"
	rec4ID            = "01900000-0000-7000-8000-000000000721"

	// Recommendations Extended
	recsSignal1ID   = "01900000-0000-7000-8000-000000000520"
	recsSignal2ID   = "01900000-0000-7000-8000-000000000521"
	recsSignal3ID   = "01900000-0000-7000-8000-000000000522"
	recsFeedback1ID = "01900000-0000-7000-8000-000000000514"
	recsFeedback2ID = "01900000-0000-7000-8000-000000000515"

	// Admin Domain
	adminFlag1ID     = "01900000-0000-7000-8000-000000000900"
	adminFlag2ID     = "01900000-0000-7000-8000-000000000901"
	adminFlag3ID     = "01900000-0000-7000-8000-000000000902"
	adminFlag4ID     = "01900000-0000-7000-8000-000000000903"
	adminFlag5ID     = "01900000-0000-7000-8000-000000000904"
	adminAuditLog1ID = "01900000-0000-7000-8000-000000000910"
	adminAuditLog2ID = "01900000-0000-7000-8000-000000000911"
	adminAuditLog3ID = "01900000-0000-7000-8000-000000000912"

	// Comply Domain
	complySchedule1ID      = "01900000-0000-7000-8000-000000000800"
	complyAttend1ID        = "01900000-0000-7000-8000-000000000801"
	complyAttend2ID        = "01900000-0000-7000-8000-000000000802"
	complyAttend3ID        = "01900000-0000-7000-8000-000000000803"
	complyAttend4ID        = "01900000-0000-7000-8000-000000000804"
	complyAttend5ID        = "01900000-0000-7000-8000-000000000805"
	complyAttend6ID        = "01900000-0000-7000-8000-000000000806"
	complyAttend7ID        = "01900000-0000-7000-8000-000000000807"
	complyAttend8ID        = "01900000-0000-7000-8000-000000000808"
	complyAttend9ID        = "01900000-0000-7000-8000-000000000809"
	complyAttend10ID       = "01900000-0000-7000-8000-000000000810"
	complyAssess1ID        = "01900000-0000-7000-8000-000000000820"
	complyAssess2ID        = "01900000-0000-7000-8000-000000000821"
	complyAssess3ID        = "01900000-0000-7000-8000-000000000822"
	complyAssess4ID        = "01900000-0000-7000-8000-000000000823"
	complyCourse1ID        = "01900000-0000-7000-8000-000000000830"
	complyCourse2ID        = "01900000-0000-7000-8000-000000000831"
	complyCourse3ID        = "01900000-0000-7000-8000-000000000832"
	complyTranscript1ID    = "01900000-0000-7000-8000-000000000840"
	complyPortfolio1ID     = "01900000-0000-7000-8000-000000000850"
	complyPortfolioItem1ID = "01900000-0000-7000-8000-000000000851"
	complyPortfolioItem2ID = "01900000-0000-7000-8000-000000000852"
	complyPortfolioItem3ID = "01900000-0000-7000-8000-000000000853"
	complyStdTest1ID       = "01900000-0000-7000-8000-000000000860"

	// Discovery
	discQuizDefID     = "01900000-0000-7000-8000-000000000960"
	discQuizResult1ID = "01900000-0000-7000-8000-000000000961"

	// Planning
	schedItem1ID = "01900000-0000-7000-8000-000000000801" //nolint:deadcode // ranges don't overlap with comply
	schedItem2ID = "01900000-0000-7000-8000-000000000802"
	schedItem3ID = "01900000-0000-7000-8000-000000000803"
	schedItem4ID = "01900000-0000-7000-8000-000000000804"
	schedItem5ID = "01900000-0000-7000-8000-000000000805"
	schedTmpl1ID = "01900000-0000-7000-8000-000000000811"

	// Lifecycle
	lifecycleExport1ID = "01900000-0000-7000-8000-000000000970"

	// Fallback Kratos identity UUIDs when Kratos is unreachable
	fallbackKratosID       = "01900000-0000-7000-8000-000000000999" // seed parent
	fallbackFriendKratosID = "01900000-0000-7000-8000-000000000998" // friend parent
	fallbackAdminKratosID  = "01900000-0000-7000-8000-000000000997" // platform admin

	// Kratos admin API base URLs (host ports, not container-internal)
	kratosAdminURL      = "http://localhost:4934" // dev Kratos (homegrown DB)
	kratosAgentAdminURL = "http://localhost:4936" // agent Kratos (kratos_agent DB)

	// Stress Family — adversarial layout/overflow test data (range: 02...)
	// Login: stress@example.com / SeedPassword123!
	// Purpose: surfaces layout bugs hidden by polite happy-path seed data.
	stressFamilyID         = "02000000-0000-7000-8000-000000000001"
	stressParentID         = "02000000-0000-7000-8000-000000000002"
	stressStudent1ID       = "02000000-0000-7000-8000-000000000011" // ~195-char Latin+diacritics name
	stressStudent2ID       = "02000000-0000-7000-8000-000000000012" // emoji + mixed unicode name
	stressStudent3ID       = "02000000-0000-7000-8000-000000000013" // Arabic RTL name
	stressStudent4ID       = "02000000-0000-7000-8000-000000000014" // Greek script name
	stressStudent5ID       = "02000000-0000-7000-8000-000000000015" // normal (bulk to stress many-students UI)
	stressStudent6ID       = "02000000-0000-7000-8000-000000000016"
	stressStudent7ID       = "02000000-0000-7000-8000-000000000017"
	stressStudent8ID       = "02000000-0000-7000-8000-000000000018"
	stressWizardProgressID = "02000000-0000-7000-8000-000000000021"
	stressCoppaAuditID     = "02000000-0000-7000-8000-000000000022"
	stressPublisherID      = "02000000-0000-7000-8000-000000000031"
	stressCreatorID        = "02000000-0000-7000-8000-000000000032"
	stressJournal1ID       = "02000000-0000-7000-8000-000000000041" // very long content
	stressJournal2ID       = "02000000-0000-7000-8000-000000000042" // emoji + unicode + RTL mix
	stressJournal3ID       = "02000000-0000-7000-8000-000000000043" // minimal content (missing optional richness)
	stressReadingListID    = "02000000-0000-7000-8000-000000000051" // empty reading list
	fallbackStressKratosID = "02000000-0000-7000-8000-000000000999"
)

func main() {
	var dbName string
	flag.StringVar(&dbName, "db", "homegrown_agent", "target database name")
	flag.Parse()

	// ── Logger ────────────────────────────────────────────────────────────
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("seeder starting", "target_db", dbName)

	// ── Connection URL ────────────────────────────────────────────────────
	targetURL, adminURL, err := deriveURLs(dbName)
	if err != nil {
		slog.Error("failed to derive connection URLs", "err", err)
		os.Exit(1)
	}

	// ── Ensure database exists ────────────────────────────────────────────
	if err := ensureDatabase(targetURL, adminURL, dbName); err != nil {
		slog.Error("failed to ensure database exists", "err", err)
		os.Exit(1)
	}

	// ── Open GORM connection ──────────────────────────────────────────────
	db, err := openDB(targetURL)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("failed to get underlying sql.DB", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Warn("close db", "err", err)
		}
	}()

	// ── Migrations ────────────────────────────────────────────────────────
	if err := runMigrations(sqlDB); err != nil {
		slog.Error("migrations failed", "err", err)
		os.Exit(1)
	}

	// ── Kratos identities ─────────────────────────────────────────────────
	seedKratosID := ensureKratosIdentity(dbName)
	adminKratosID := ensureAdminKratosIdentity(dbName)
	stressKratosID := ensureStressKratosIdentity(dbName)

	// ── Seed all domains ──────────────────────────────────────────────────
	if err := seedAll(db, seedKratosID, adminKratosID, stressKratosID); err != nil {
		slog.Error("seeding failed", "err", err)
		os.Exit(1)
	}

	slog.Info("seeding complete", "target_db", dbName)
}

// ─── Connection URL helpers ────────────────────────────────────────────────────

// deriveURLs reads the .env FILE (not os.Getenv), extracts DATABASE_URL,
// replaces the database name, and returns the target and admin (base) URLs.
func deriveURLs(dbName string) (targetURL, adminURL string, err error) {
	envMap, readErr := godotenv.Read(".env")
	if readErr != nil {
		slog.Warn("could not read .env file, using default connection string", "err", readErr)
	}

	base := envMap["DATABASE_URL"]
	if base == "" {
		base = "postgres://homegrown:homegrown@localhost:5932/homegrown"
	}

	u, err := url.Parse(base)
	if err != nil {
		return "", "", fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	// adminURL connects to the base "homegrown" DB (for CREATE DATABASE)
	adminU := *u
	adminU.Path = "/" + strings.TrimPrefix(u.Path, "/")
	adminURL = adminU.String()

	// targetURL replaces the database name
	u.Path = "/" + dbName
	targetURL = u.String()

	return targetURL, adminURL, nil
}

// ensureDatabase creates the target database if it doesn't already exist.
func ensureDatabase(targetURL, adminURL, dbName string) error {
	// Try connecting to the target first — if it works, nothing to do.
	testDB, err := sql.Open("pgx", targetURL)
	if err == nil {
		if pingErr := testDB.Ping(); pingErr == nil {
			if closeErr := testDB.Close(); closeErr != nil {
				slog.Warn("close test db", "err", closeErr)
			}
			slog.Info("target database already exists", "db", dbName)
			return nil
		}
		if closeErr := testDB.Close(); closeErr != nil {
			slog.Warn("close test db", "err", closeErr)
		}
	}

	slog.Info("creating target database", "db", dbName)

	adminDB, err := sql.Open("pgx", adminURL)
	if err != nil {
		return fmt.Errorf("open admin db: %w", err)
	}
	defer func() {
		if err := adminDB.Close(); err != nil {
			slog.Warn("close admin db", "err", err)
		}
	}()

	// dbName is controlled by the --db flag (developer input only), not user data.
	// Safe to interpolate here; the flag is not user-facing in production.
	if _, execErr := adminDB.Exec("CREATE DATABASE " + dbName); execErr != nil {
		// Ignore "already exists" errors.
		if !strings.Contains(execErr.Error(), "already exists") {
			return fmt.Errorf("create database: %w", execErr)
		}
	}

	return nil
}

func openDB(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

// ─── Migrations ───────────────────────────────────────────────────────────────

func runMigrations(db *sql.DB) error {
	slog.Info("running migrations")
	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	slog.Info("migrations complete")
	return nil
}

// ─── Kratos identity ──────────────────────────────────────────────────────────

type kratosIdentityBody struct {
	SchemaID    string         `json:"schema_id"`
	Traits      map[string]any `json:"traits"`
	Credentials map[string]any `json:"credentials"`
}

// kratosLookupOrCreate attempts to find or create a Kratos identity at baseURL.
// Returns (id, true) on success, ("", false) if the server is unreachable.
func kratosLookupOrCreate(baseURL, email string) (string, bool) {
	resp, err := http.Get(baseURL + "/admin/identities?credentials_identifier=" + url.QueryEscape(email))
	if err != nil {
		return "", false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("close resp body", "err", err)
		}
	}()

	body, _ := io.ReadAll(resp.Body)
	var identities []map[string]any
	if jsonErr := json.Unmarshal(body, &identities); jsonErr == nil && len(identities) > 0 {
		if id, ok := identities[0]["id"].(string); ok && id != "" {
			slog.Info("found existing Kratos identity", "email", email, "id", id)
			return id, true
		}
	}

	// Create new identity
	payload := kratosIdentityBody{
		SchemaID: "user",
		Traits:   map[string]any{"email": email, "name": email},
		Credentials: map[string]any{
			"password": map[string]any{
				"config": map[string]any{
					"password": "SeedPassword123!",
				},
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	createResp, createErr := http.Post(
		baseURL+"/admin/identities",
		"application/json",
		strings.NewReader(string(payloadBytes)),
	)
	if createErr != nil {
		return "", false
	}
	defer func() {
		if err := createResp.Body.Close(); err != nil {
			slog.Warn("close create resp body", "err", err)
		}
	}()

	createBody, _ := io.ReadAll(createResp.Body)
	var created map[string]any
	if jsonErr := json.Unmarshal(createBody, &created); jsonErr == nil {
		if id, ok := created["id"].(string); ok && id != "" {
			slog.Info("created Kratos identity", "email", email, "id", id)
			return id, true
		}
	}
	return "", false
}

// kratosURLOrder returns Kratos admin base URLs in priority order for the given
// database name. The agent database uses agent Kratos first; everything else
// (including the dev "homegrown" database) tries dev Kratos first so that
// identities land in the correct instance and dev-app login works.
func kratosURLOrder(dbName string) []string {
	if dbName == "homegrown_agent" {
		return []string{kratosAgentAdminURL, kratosAdminURL}
	}
	return []string{kratosAdminURL, kratosAgentAdminURL}
}

// ensureKratosIdentity creates or retrieves the seed@example.com identity.
// Tries the appropriate Kratos instance first based on the target database.
func ensureKratosIdentity(dbName string) string {
	for _, baseURL := range kratosURLOrder(dbName) {
		if id, ok := kratosLookupOrCreate(baseURL, "seed@example.com"); ok {
			return id
		}
	}
	slog.Warn("Kratos unreachable, using fallback identity UUID for seed parent")
	return fallbackKratosID
}

// ensureAdminKratosIdentity creates or retrieves the admin@example.com identity.
func ensureAdminKratosIdentity(dbName string) string {
	for _, baseURL := range kratosURLOrder(dbName) {
		if id, ok := kratosLookupOrCreate(baseURL, "admin@example.com"); ok {
			return id
		}
	}
	slog.Warn("Kratos unreachable, using fallback identity UUID for admin parent")
	return fallbackAdminKratosID
}

// ─── Seed orchestrator ────────────────────────────────────────────────────────

func seedAll(db *gorm.DB, seedKratosID, adminKratosID, stressKratosID string) error {
	// Look up platform publisher ID (seeded by migration, not by seeder)
	var platformPublisherID string
	if err := db.Raw("SELECT id FROM mkt_publishers WHERE slug = 'homegrown-academy'").
		Scan(&platformPublisherID).Error; err != nil || platformPublisherID == "" {
		slog.Warn("platform publisher not found — using seed publisher only")
		platformPublisherID = seedPublisherID
	}

	type seedStep struct {
		name string
		fn   func(*gorm.DB) error
	}

	steps := []seedStep{
		{"PlatformSetup", func(db *gorm.DB) error { return seedPlatformSetup(db, adminKratosID) }},
		{"IAM", func(db *gorm.DB) error { return seedIAM(db, seedKratosID) }},
		{"IAMExtended", seedIAMExtended},
		{"Onboard", seedOnboard},
		{"Social", seedSocial},
		{"SocialExtended", seedSocialExtended},
		{"Marketplace", seedMarketplace},
		{"MarketplaceExtended", seedMarketplaceExtended},
		{"PlatformContent", seedPlatformContent},
		{"Learn", func(db *gorm.DB) error { return seedLearn(db, platformPublisherID) }},
		{"LearnExtended", func(db *gorm.DB) error { return seedLearnExtended(db, platformPublisherID) }},
		{"Discovery", seedDiscovery},
		{"Notifications", seedNotifications},
		{"Billing", seedBilling},
		{"Media", seedMedia},
		{"Safety", seedSafetyFull},
		{"AdminDomain", seedAdminDomain},
		{"Comply", seedComply},
		{"Recommendations", seedRecs},
		{"RecsExtended", seedRecsExtended},
		{"LearnerProfile", seedLearnerProfile},
		{"Planning", seedPlan},
		{"Lifecycle", seedLifecycle},
		{"StressFamily", func(db *gorm.DB) error { return seedStressFamily(db, stressKratosID) }},
	}

	for _, step := range steps {
		slog.Info("seeding domain", "domain", step.name)
		if err := step.fn(db); err != nil {
			return fmt.Errorf("seed %s: %w", step.name, err)
		}
	}
	return nil
}

// bypassRLS executes fn inside a transaction with row_security disabled.
func bypassRLS(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL row_security = off").Error; err != nil {
			return err
		}
		return fn(tx)
	})
}

// ─── Platform setup seed ──────────────────────────────────────────────────────

func seedPlatformSetup(db *gorm.DB, adminKratosID string) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Platform family
		if err := tx.Exec(`
			INSERT INTO iam_families
				(id, display_name, state_code, primary_methodology_slug,
				 subscription_tier, coppa_consent_status)
			VALUES (?, 'Platform Team', 'TX', 'classical', 'premium', 'consented')
			ON CONFLICT (id) DO NOTHING`,
			platformFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert platform family: %w", err)
		}

		// Platform admin parent (is_platform_admin=true)
		if err := tx.Exec(`
			INSERT INTO iam_parents
				(id, family_id, kratos_identity_id, display_name, email,
				 is_primary, is_platform_admin)
			VALUES (?, ?, ?, 'Platform Admin', 'admin@example.com', true, true)
			ON CONFLICT (id) DO NOTHING`,
			adminParentID, platformFamilyID, adminKratosID,
		).Error; err != nil {
			return fmt.Errorf("insert admin parent: %w", err)
		}

		// Set primary parent (idempotent: only updates when NULL)
		if err := tx.Exec(`
			UPDATE iam_families SET primary_parent_id = ?
			WHERE id = ? AND primary_parent_id IS NULL`,
			adminParentID, platformFamilyID,
		).Error; err != nil {
			return fmt.Errorf("update platform family primary parent: %w", err)
		}

		return nil
	})
}

// ─── IAM seed ─────────────────────────────────────────────────────────────────

func seedIAM(db *gorm.DB, kratosID string) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Families (primary_parent_id set after parents are inserted)
		if err := tx.Exec(`
			INSERT INTO iam_families
				(id, display_name, state_code, primary_methodology_slug, subscription_tier,
				 coppa_consent_status, coppa_consented_at, coppa_consent_method)
			VALUES
				(?, 'The Seed Family',   'TX', 'charlotte-mason', 'premium',
				 'consented', NOW(), 'credit_card_verification'),
				(?, 'The Friend Family', 'TX', 'classical',       'free',
				 'registered', NULL, NULL)
			ON CONFLICT (id) DO NOTHING`,
			seedFamilyID, friendFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert families: %w", err)
		}

		// Parents
		if err := tx.Exec(`
			INSERT INTO iam_parents
				(id, family_id, kratos_identity_id, display_name, email, is_primary)
			VALUES
				(?, ?, ?, 'Seed Parent',   'seed@example.com',   true),
				(?, ?, ?, 'Friend Parent', 'friend@example.com', true)
			ON CONFLICT (id) DO NOTHING`,
			seedParentID, seedFamilyID, kratosID,
			friendParentID, friendFamilyID, fallbackFriendKratosID,
		).Error; err != nil {
			return fmt.Errorf("insert parents: %w", err)
		}

		// Update primary_parent_id on families
		if err := tx.Exec(`
			UPDATE iam_families SET primary_parent_id = ? WHERE id = ?`,
			seedParentID, seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("update seed family primary parent: %w", err)
		}
		if err := tx.Exec(`
			UPDATE iam_families SET primary_parent_id = ? WHERE id = ?`,
			friendParentID, friendFamilyID,
		).Error; err != nil {
			return fmt.Errorf("update friend family primary parent: %w", err)
		}

		// Students
		if err := tx.Exec(`
			INSERT INTO iam_students
				(id, family_id, display_name, birth_year, grade_level)
			VALUES
				(?, ?, 'Emma',  2014, '5th'),
				(?, ?, 'James', 2017, '2nd')
			ON CONFLICT (id) DO NOTHING`,
			emmaStudentID, seedFamilyID,
			jamesStudentID, seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert students: %w", err)
		}

		// COPPA audit log
		if err := tx.Exec(`
			INSERT INTO iam_coppa_audit_log
				(id, family_id, action, method,
				 previous_status, new_status, performed_by)
			VALUES
				(?, ?, 'consent_granted', 'credit_card_verification',
				 'noticed', 'consented', ?)
			ON CONFLICT (id) DO NOTHING`,
			coppaAuditID, seedFamilyID, seedParentID,
		).Error; err != nil {
			return fmt.Errorf("insert coppa audit log: %w", err)
		}

		return nil
	})
}

// ─── IAM Extended seed ────────────────────────────────────────────────────────

func seedIAMExtended(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Co-parent invite (pending, expires in 7 days)
		if err := tx.Exec(`
			INSERT INTO iam_co_parent_invites
				(id, family_id, invited_by, email, token, status, expires_at)
			VALUES (?, ?, ?, 'coparent@example.com',
				'seed-coparent-invite-token-001', 'pending',
				NOW() + INTERVAL '7 days')
			ON CONFLICT (id) DO NOTHING`,
			coParentInviteID, seedFamilyID, seedParentID,
		).Error; err != nil {
			return fmt.Errorf("insert co_parent_invite: %w", err)
		}

		// Student session (Emma's tablet session, active)
		// token_hash stores a bcrypt-like placeholder (real hash done by app)
		if err := tx.Exec(`
			INSERT INTO iam_student_sessions
				(id, family_id, student_id, created_by, token_hash,
				 expires_at, is_active, permissions)
			VALUES (?, ?, ?, ?,
				'$seed$emma$session$hash$placeholder$001',
				NOW() + INTERVAL '8 hours', true,
				'["reading-lists","activity-logger"]'::JSONB)
			ON CONFLICT (id) DO NOTHING`,
			studentSession1ID, seedFamilyID, emmaStudentID, seedParentID,
		).Error; err != nil {
			return fmt.Errorf("insert student session: %w", err)
		}

		return nil
	})
}

// ─── Onboard seed ─────────────────────────────────────────────────────────────

func seedOnboard(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Wizard progress for seed family — in_progress so the UI flow can be tested.
		if err := tx.Exec(`
			INSERT INTO onb_wizard_progress
				(id, family_id, status, current_step, completed_steps)
			VALUES (?, ?, 'in_progress', 'family_profile', '{}')
			ON CONFLICT (family_id) DO NOTHING`,
			seedWizardProgressID, seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert wizard progress: %w", err)
		}

		// Admin family wizard progress — skipped so admins bypass the onboarding guard.
		if err := tx.Exec(`
			INSERT INTO onb_wizard_progress
				(id, family_id, status, current_step, completed_steps)
			VALUES (?, ?, 'skipped', 'family_profile', '{}')
			ON CONFLICT (family_id) DO NOTHING`,
			adminWizardProgressID, platformFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert admin wizard progress: %w", err)
		}
		return nil
	})
}

// ─── Social seed ──────────────────────────────────────────────────────────────

func seedSocial(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Profiles (1:1 with families, PK = family_id)
		if err := tx.Exec(`
			INSERT INTO soc_profiles (family_id, bio)
			VALUES
				(?, 'A Charlotte Mason homeschool family in Texas. Narration, nature journals, and living books.'),
				(?, 'Classical education family. Trivium all the way!')
			ON CONFLICT (family_id) DO NOTHING`,
			seedFamilyID, friendFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert soc_profiles: %w", err)
		}

		// Friendship (accepted)
		if err := tx.Exec(`
			INSERT INTO soc_friendships
				(id, requester_family_id, accepter_family_id, status)
			VALUES (?, ?, ?, 'accepted')
			ON CONFLICT (id) DO NOTHING`,
			friendshipID, seedFamilyID, friendFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert friendship: %w", err)
		}

		// Group
		if err := tx.Exec(`
			INSERT INTO soc_groups
				(id, group_type, name, description, creator_family_id,
				 methodology_slug, join_policy, member_count)
			VALUES (?, 'user_created', 'Charlotte Mason Co-op',
				'A local co-op for Charlotte Mason families. Nature walks, handicrafts, and poetry tea times.',
				?, 'charlotte-mason', 'open', 2)
			ON CONFLICT (id) DO NOTHING`,
			groupID, seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert group: %w", err)
		}

		// Group members
		if err := tx.Exec(`
			INSERT INTO soc_group_members
				(id, group_id, family_id, role, status, joined_at)
			VALUES
				(?, ?, ?, 'owner',  'active', NOW()),
				(?, ?, ?, 'member', 'active', NOW())
			ON CONFLICT (id) DO NOTHING`,
			seedGroupMemID, groupID, seedFamilyID,
			friendGroupMemID, groupID, friendFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert group members: %w", err)
		}

		// Posts (2 from seed, 1 from friend — all visibility=friends, no group)
		if err := tx.Exec(`
			INSERT INTO soc_posts
				(id, family_id, author_parent_id, post_type, content, visibility)
			VALUES
				(?, ?, ?, 'text',
				 'Had a wonderful nature walk today! Emma found three different types of lichen and James spotted a red-tailed hawk.',
				 'friends'),
				(?, ?, ?, 'text',
				 'Starting Charlotte''s Web this week for our read-aloud. The kids are already begging for ''just one more chapter.''',
				 'friends'),
				(?, ?, ?, 'text',
				 'Finished our grammar stage memory work for the week. Latin is coming along beautifully!',
				 'friends')
			ON CONFLICT (id) DO NOTHING`,
			post1ID, seedFamilyID, seedParentID,
			post2ID, seedFamilyID, seedParentID,
			post3ID, friendFamilyID, friendParentID,
		).Error; err != nil {
			return fmt.Errorf("insert posts: %w", err)
		}

		// Comments
		if err := tx.Exec(`
			INSERT INTO soc_comments
				(id, post_id, family_id, author_parent_id, content)
			VALUES
				(?, ?, ?, ?, 'Love this! Which trail did you take'),
				(?, ?, ?, ?, 'We use a lichen field guide from the library — so helpful!')
			ON CONFLICT (id) DO NOTHING`,
			comment1ID, post1ID, friendFamilyID, friendParentID,
			comment2ID, post1ID, seedFamilyID, seedParentID,
		).Error; err != nil {
			return fmt.Errorf("insert comments: %w", err)
		}

		// Post likes (friend likes seed post1)
		if err := tx.Exec(`
			INSERT INTO soc_post_likes (id, post_id, family_id)
			VALUES (?, ?, ?)
			ON CONFLICT (id) DO NOTHING`,
			postLike1ID, post1ID, friendFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert post likes: %w", err)
		}

		// Conversation + participants + messages
		if err := tx.Exec(`
			INSERT INTO soc_conversations (id)
			VALUES (?)
			ON CONFLICT (id) DO NOTHING`,
			conversationID,
		).Error; err != nil {
			return fmt.Errorf("insert conversation: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO soc_conversation_participants
				(conversation_id, parent_id, family_id)
			VALUES
				(?, ?, ?),
				(?, ?, ?)
			ON CONFLICT (conversation_id, parent_id) DO NOTHING`,
			conversationID, seedParentID, seedFamilyID,
			conversationID, friendParentID, friendFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert conversation participants: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO soc_messages
				(id, conversation_id, sender_parent_id, sender_family_id, content)
			VALUES
				(?, ?, ?, ?, 'Hey! Are you coming to the co-op meetup next week'),
				(?, ?, ?, ?, 'Yes! We''re so excited. Should we bring anything'),
				(?, ?, ?, ?, 'Just bring yourselves! We''ll have snacks covered.')
			ON CONFLICT (id) DO NOTHING`,
			msg1ID, conversationID, seedParentID, seedFamilyID,
			msg2ID, conversationID, friendParentID, friendFamilyID,
			msg3ID, conversationID, seedParentID, seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert messages: %w", err)
		}

		// Event
		if err := tx.Exec(`
			INSERT INTO soc_events
				(id, creator_family_id, creator_parent_id, title, description,
				 event_date, location_name, location_region, visibility, status, methodology_slug)
			VALUES (?, ?, ?,
				'Spring Nature Walk & Sketching',
				'Join us for a guided nature walk followed by outdoor sketching. All ages welcome!',
				NOW() + INTERVAL '14 days',
				'Barton Creek Greenbelt', 'Austin, TX',
				'friends', 'active', 'charlotte-mason')
			ON CONFLICT (id) DO NOTHING`,
			eventID, seedFamilyID, seedParentID,
		).Error; err != nil {
			return fmt.Errorf("insert event: %w", err)
		}

		// RSVP
		if err := tx.Exec(`
			INSERT INTO soc_event_rsvps (id, event_id, family_id, status)
			VALUES (?, ?, ?, 'going')
			ON CONFLICT (id) DO NOTHING`,
			rsvpID, eventID, friendFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert event rsvp: %w", err)
		}

		return nil
	})
}

// ─── Social Extended seed ─────────────────────────────────────────────────────

func seedSocialExtended(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Pin post1 to the Charlotte Mason Co-op group
		if err := tx.Exec(`
			INSERT INTO soc_pinned_posts (id, group_id, post_id, pinned_by)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (id) DO NOTHING`,
			pinnedPost1ID, groupID, post1ID, seedParentID,
		).Error; err != nil {
			return fmt.Errorf("insert pinned post: %w", err)
		}
		return nil
	})
}

// ─── Marketplace seed ─────────────────────────────────────────────────────────

func seedMarketplace(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Seed publisher (creator-owned)
		if err := tx.Exec(`
			INSERT INTO mkt_publishers
				(id, name, slug, description, is_platform, is_verified)
			VALUES (?, 'Seed Homeschool Press', 'seed-homeschool-press',
				'Curriculum and resources by homeschooling parents, for homeschooling parents.',
				false, true)
			ON CONFLICT (id) DO NOTHING`,
			seedPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert seed publisher: %w", err)
		}

		// Creator
		if err := tx.Exec(`
			INSERT INTO mkt_creators
				(id, parent_id, onboarding_status, store_name, store_bio, tos_accepted_at)
			VALUES (?, ?, 'active', 'Seed Homeschool Press',
				'Creating gentle, literature-rich resources for Charlotte Mason families.',
				NOW() - INTERVAL '60 days')
			ON CONFLICT (id) DO NOTHING`,
			seedCreatorID, seedParentID,
		).Error; err != nil {
			return fmt.Errorf("insert creator: %w", err)
		}

		// 5 listings using seed creator + publisher
		listings := []struct {
			id           string
			title        string
			description  string
			priceCents   int
			contentType  string
			thumbnailURL string
		}{
			{listing1ID, "Charlotte Mason Year 1 Curriculum Guide",
				"A complete first-year guide with book lists, nature study schedules, and narration prompts.",
				2999, "curriculum",
				"https://picsum.photos/seed/charlotte-mason/800/450"},
			{listing2ID, "Nature Journal Starter Pack",
				"Illustrated worksheets for nature journaling: trees, insects, birds, and seasons.",
				999, "worksheet",
				"https://picsum.photos/seed/nature-journal/800/450"},
			{listing3ID, "Living Books Read-Aloud Video Series",
				"Twelve read-aloud video lessons featuring classic literature with narration guides.",
				1999, "video",
				"https://picsum.photos/seed/reading-books/800/450"},
			{listing4ID, "Charlotte Mason Book List: K-5",
				"Curated living books list organized by subject and grade level, with library links.",
				499, "book_list",
				"https://picsum.photos/seed/library-books/800/450"},
			{listing5ID, "Narration Assessment Rubric",
				"Age-appropriate narration assessment tools for oral and written narration.",
				299, "assessment",
				"https://picsum.photos/seed/writing-desk/800/450"},
		}

		for _, l := range listings {
			if err := tx.Exec(`
				INSERT INTO mkt_listings
					(id, creator_id, publisher_id, title, description, price_cents,
					 methodology_tags, subject_tags, content_type, status, published_at,
					 thumbnail_url)
				VALUES (?, ?, ?, ?, ?, ?,
					'{}', ARRAY['language-arts.reading-comprehension','language-arts'], ?, 'published', NOW() - INTERVAL '30 days',
					?)
				ON CONFLICT (id) DO NOTHING`,
				l.id, seedCreatorID, seedPublisherID,
				l.title, l.description, l.priceCents, l.contentType, l.thumbnailURL,
			).Error; err != nil {
				return fmt.Errorf("insert listing %s: %w", l.id, err)
			}
			// Backfill thumbnail for already-seeded rows (idempotent).
			if err := tx.Exec(
				`UPDATE mkt_listings SET thumbnail_url = ? WHERE id = ? AND thumbnail_url IS NULL`,
				l.thumbnailURL, l.id,
			).Error; err != nil {
				return fmt.Errorf("backfill thumbnail %s: %w", l.id, err)
			}
		}

		// Purchases (friend family bought listing1 and listing2)
		if err := tx.Exec(`
			INSERT INTO mkt_purchases
				(id, family_id, listing_id, creator_id,
				 amount_cents, platform_fee_cents, creator_payout_cents)
			VALUES
				(?, ?, ?, ?, 2999, 300, 2699),
				(?, ?, ?, ?,  999, 100,  899)
			ON CONFLICT (id) DO NOTHING`,
			purchase1ID, friendFamilyID, listing1ID, seedCreatorID,
			purchase2ID, friendFamilyID, listing2ID, seedCreatorID,
		).Error; err != nil {
			return fmt.Errorf("insert purchases: %w", err)
		}

		// Review (friend family reviews listing1)
		if err := tx.Exec(`
			INSERT INTO mkt_reviews
				(id, listing_id, purchase_id, family_id, rating,
				 review_text, is_anonymous, moderation_status)
			VALUES (?, ?, ?, ?, 5,
				'Absolutely wonderful curriculum guide. Saved us so much planning time!',
				false, 'approved')
			ON CONFLICT (id) DO NOTHING`,
			review1ID, listing1ID, purchase1ID, friendFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert review: %w", err)
		}

		// Cart item (seed family has listing3 in cart)
		if err := tx.Exec(`
			INSERT INTO mkt_cart_items
				(id, family_id, listing_id, added_by_parent_id)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (id) DO NOTHING`,
			cartItem1ID, seedFamilyID, listing3ID, seedParentID,
		).Error; err != nil {
			return fmt.Errorf("insert cart item: %w", err)
		}

		// ─── Friend creator + listing so seed family can purchase & review ────
		// Friend publisher (creator-owned)
		if err := tx.Exec(`
			INSERT INTO mkt_publishers
				(id, name, slug, description, is_platform, is_verified)
			VALUES (?, 'Friend Family Press', 'friend-family-press',
				'Resources from the Friend homeschooling family.',
				false, true)
			ON CONFLICT (id) DO NOTHING`,
			friendPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert friend publisher: %w", err)
		}

		// Friend creator
		if err := tx.Exec(`
			INSERT INTO mkt_creators
				(id, parent_id, onboarding_status, store_name, store_bio, tos_accepted_at)
			VALUES (?, ?, 'active', 'Friend Family Press',
				'Classical education resources from our family to yours.',
				NOW() - INTERVAL '45 days')
			ON CONFLICT (id) DO NOTHING`,
			friendCreatorID, friendParentID,
		).Error; err != nil {
			return fmt.Errorf("insert friend creator: %w", err)
		}

		// Publisher member: friend creator owns friend publisher
		if err := tx.Exec(`
			INSERT INTO mkt_publisher_members (publisher_id, creator_id, role)
			VALUES (?, ?, 'owner')
			ON CONFLICT (publisher_id, creator_id) DO NOTHING`,
			friendPublisherID, friendCreatorID,
		).Error; err != nil {
			return fmt.Errorf("insert friend publisher member: %w", err)
		}

		// Friend listing (something seed family can buy)
		if err := tx.Exec(`
			INSERT INTO mkt_listings
				(id, creator_id, publisher_id, title, description, price_cents,
				 methodology_tags, subject_tags, content_type, status, published_at,
				 thumbnail_url)
			VALUES (?, ?, ?, 'Classical Trivium Workbook', 'A comprehensive grammar-stage workbook.', 1999,
				'{}', ARRAY['language-arts','foreign-languages.latin'], 'worksheet', 'published', NOW() - INTERVAL '20 days',
				'https://picsum.photos/seed/classical-trivium/800/450')
			ON CONFLICT (id) DO NOTHING`,
			friendListingID, friendCreatorID, friendPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert friend listing: %w", err)
		}
		// Backfill thumbnail for already-seeded friend listing (idempotent).
		if err := tx.Exec(
			`UPDATE mkt_listings SET thumbnail_url = ? WHERE id = ? AND thumbnail_url IS NULL`,
			"https://picsum.photos/seed/classical-trivium/800/450", friendListingID,
		).Error; err != nil {
			return fmt.Errorf("backfill friend listing thumbnail: %w", err)
		}

		// Seed family purchase of friend listing (enables review creation in E2E)
		if err := tx.Exec(`
			INSERT INTO mkt_purchases
				(id, family_id, listing_id, creator_id,
				 amount_cents, platform_fee_cents, creator_payout_cents)
			VALUES (?, ?, ?, ?, 1999, 200, 1799)
			ON CONFLICT (id) DO NOTHING`,
			seedPurchaseID, seedFamilyID, friendListingID, friendCreatorID,
		).Error; err != nil {
			return fmt.Errorf("insert seed purchase: %w", err)
		}

		return nil
	})
}

// ─── Marketplace Extended seed ────────────────────────────────────────────────

func seedMarketplaceExtended(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Publisher member: seed creator is an owner of seed publisher
		if err := tx.Exec(`
			INSERT INTO mkt_publisher_members (publisher_id, creator_id, role)
			VALUES (?, ?, 'owner')
			ON CONFLICT (publisher_id, creator_id) DO NOTHING`,
			seedPublisherID, seedCreatorID,
		).Error; err != nil {
			return fmt.Errorf("insert publisher member: %w", err)
		}

		// Look up curated section IDs by slug (seeded by migration)
		var featuredSectionID, staffPicksSectionID string
		if err := tx.Raw("SELECT id FROM mkt_curated_sections WHERE slug = 'featured'").
			Scan(&featuredSectionID).Error; err != nil || featuredSectionID == "" {
			slog.Warn("featured curated section not found, skipping curated items")
			return nil
		}
		if err := tx.Raw("SELECT id FROM mkt_curated_sections WHERE slug = 'staff-picks'").
			Scan(&staffPicksSectionID).Error; err != nil || staffPicksSectionID == "" {
			slog.Warn("staff-picks curated section not found, skipping curated items")
			return nil
		}

		// Curated section items
		curatedItems := []struct {
			id        string
			sectionID string
			listingID string
			order     int
		}{
			{mktCuratedItem1ID, featuredSectionID, listing1ID, 1},
			{mktCuratedItem2ID, featuredSectionID, listing3ID, 2},
			{mktCuratedItem3ID, staffPicksSectionID, listing2ID, 1},
			{mktCuratedItem4ID, staffPicksSectionID, listing4ID, 2},
		}
		for _, item := range curatedItems {
			if err := tx.Exec(`
				INSERT INTO mkt_curated_section_items
					(id, section_id, listing_id, sort_order)
				VALUES (?, ?, ?, ?)
				ON CONFLICT (section_id, listing_id) DO NOTHING`,
				item.id, item.sectionID, item.listingID, item.order,
			).Error; err != nil {
				return fmt.Errorf("insert curated item %s: %w", item.id, err)
			}
		}

		// Listing file for listing1
		if err := tx.Exec(`
			INSERT INTO mkt_listing_files
				(id, listing_id, file_name, file_size_bytes, mime_type, storage_key, sort_order)
			VALUES (?, ?, 'curriculum-guide.pdf', 2048000, 'application/pdf',
				'listings/listing1/curriculum-guide.pdf', 1)
			ON CONFLICT (id) DO NOTHING`,
			mktListingFile1ID, listing1ID,
		).Error; err != nil {
			return fmt.Errorf("insert listing file: %w", err)
		}

		// Populate auto curated sections (trending + new-arrivals).
		// These sections are normally refreshed by a scheduled job; seed them here
		// so the marketplace home page is never blank after a fresh seed.
		var trendingSectionID, newArrivalsSectionID string
		if err := tx.Raw("SELECT id FROM mkt_curated_sections WHERE slug = 'trending'").
			Scan(&trendingSectionID).Error; err != nil || trendingSectionID == "" {
			slog.Warn("trending curated section not found, skipping auto-populate")
			return nil
		}
		if err := tx.Raw("SELECT id FROM mkt_curated_sections WHERE slug = 'new-arrivals'").
			Scan(&newArrivalsSectionID).Error; err != nil || newArrivalsSectionID == "" {
			slog.Warn("new-arrivals curated section not found, skipping auto-populate")
			return nil
		}

		// Trending: rank by recent purchases, fall back to rating when sparse.
		if err := tx.Exec(`DELETE FROM mkt_curated_section_items WHERE section_id = ?`,
			trendingSectionID,
		).Error; err != nil {
			return fmt.Errorf("clear trending items: %w", err)
		}
		if err := tx.Exec(`
			INSERT INTO mkt_curated_section_items (section_id, listing_id, sort_order)
			SELECT ?, l.id, ROW_NUMBER() OVER (
				ORDER BY COALESCE(recent.purchase_count, 0) DESC,
				         l.rating_count DESC,
				         l.rating_avg DESC,
				         l.published_at DESC NULLS LAST
			)
			FROM mkt_listings l
			LEFT JOIN (
				SELECT listing_id, COUNT(*) AS purchase_count
				FROM mkt_purchases
				WHERE created_at > NOW() - INTERVAL '30 days'
				GROUP BY listing_id
			) recent ON recent.listing_id = l.id
			WHERE l.status = 'published'
			ORDER BY COALESCE(recent.purchase_count, 0) DESC,
			         l.rating_count DESC,
			         l.rating_avg DESC,
			         l.published_at DESC NULLS LAST
			LIMIT 20`,
			trendingSectionID,
		).Error; err != nil {
			return fmt.Errorf("populate trending items: %w", err)
		}

		// New Arrivals: most recently published listings.
		if err := tx.Exec(`DELETE FROM mkt_curated_section_items WHERE section_id = ?`,
			newArrivalsSectionID,
		).Error; err != nil {
			return fmt.Errorf("clear new-arrivals items: %w", err)
		}
		if err := tx.Exec(`
			INSERT INTO mkt_curated_section_items (section_id, listing_id, sort_order)
			SELECT ?, id, ROW_NUMBER() OVER (ORDER BY published_at DESC)
			FROM mkt_listings
			WHERE status = 'published'
			ORDER BY published_at DESC
			LIMIT 20`,
			newArrivalsSectionID,
		).Error; err != nil {
			return fmt.Errorf("populate new-arrivals items: %w", err)
		}

		return nil
	})
}

// ─── Platform content seed ────────────────────────────────────────────────────
// Inserts deterministic creator + publisher rows used by the content ingestion
// CLI (cmd/seed-content). parent_id is the admin parent so the DB FK is
// satisfied; no Kratos identity or payout account is needed.

func seedPlatformContent(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO mkt_creators
				(id, parent_id, onboarding_status, store_name, store_bio, tos_accepted_at)
			VALUES (?, ?, 'active', 'Homegrown Academy',
				'Curated free and public domain educational resources.',
				NOW() - INTERVAL '365 days')
			ON CONFLICT (parent_id) DO NOTHING`,
			platformContentCreatorID, adminParentID,
		).Error; err != nil {
			return fmt.Errorf("insert platform content creator: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO mkt_publishers
				(id, name, slug, description, is_platform, is_verified)
			VALUES (?, 'Homegrown Academy', 'homegrown-academy-mkt',
				'Platform-curated free content library.', true, true)
			ON CONFLICT (id) DO NOTHING`,
			platformContentPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert platform content publisher: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO mkt_publisher_members (publisher_id, creator_id, role)
			VALUES (?, ?, 'owner')
			ON CONFLICT (publisher_id, creator_id) DO NOTHING`,
			platformContentPublisherID, platformContentCreatorID,
		).Error; err != nil {
			return fmt.Errorf("insert platform content publisher member: %w", err)
		}

		return nil
	})
}

// ─── Learn seed ───────────────────────────────────────────────────────────────

func seedLearn(db *gorm.DB, platformPublisherID string) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Activity definitions (Layer 1 — publisher-owned, not family-scoped)
		if err := tx.Exec(`
			INSERT INTO learn_activity_defs
				(id, publisher_id, title, description, subject_tags, est_duration_minutes)
			VALUES
				(?, ?, 'Nature Walk',
				 'Outdoor nature observation and sketching session.',
				 ARRAY['science'], 60),
				(?, ?, 'Math Games',
				 'Hands-on math games using manipulatives and real-world objects.',
				 ARRAY['math'], 30),
				(?, ?, 'Read Aloud',
				 'Parent-led read-aloud session with narration.',
				 ARRAY['language-arts.reading-comprehension','language-arts'], 45)
			ON CONFLICT (id) DO NOTHING`,
			activityDef1ID, platformPublisherID,
			activityDef2ID, platformPublisherID,
			activityDef3ID, platformPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert activity defs: %w", err)
		}

		// Reading items (Layer 1)
		if err := tx.Exec(`
			INSERT INTO learn_reading_items
				(id, publisher_id, title, author, subject_tags, description, page_count)
			VALUES
				(?, ?, 'Charlotte''s Web',    'E.B. White',    ARRAY['language-arts.reading-comprehension','language-arts'], 'A classic tale of friendship between a pig and a spider.', 192),
				(?, ?, 'A Bear Called Paddington', 'Michael Bond', ARRAY['language-arts.reading-comprehension','language-arts'], 'The adventures of a bear from Peru.', 144),
				(?, ?, 'The Lion, the Witch and the Wardrobe', 'C.S. Lewis', ARRAY['language-arts.reading-comprehension','language-arts'], 'Four siblings discover a magical world through a wardrobe.', 208)
			ON CONFLICT (id) DO NOTHING`,
			readingItem1ID, platformPublisherID,
			readingItem2ID, platformPublisherID,
			readingItem3ID, platformPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert reading items: %w", err)
		}

		// Activity logs (Layer 3 — family-scoped), 10 entries over past 14 days
		now := time.Now()
		type actLog struct {
			id         string
			studentID  string
			actDefID   string
			title      string
			daysAgo    int
			durationMn int
			subjects   string
		}
		logs := []actLog{
			{actLog1ID, emmaStudentID, activityDef1ID, "Nature Walk — Creek Trail", 1, 65, "ARRAY['science']"},
			{actLog2ID, emmaStudentID, activityDef3ID, "Read Aloud: Charlotte's Web Ch 3", 1, 45, "ARRAY['language-arts.reading-comprehension']"},
			{actLog3ID, jamesStudentID, activityDef2ID, "Math Games: Counting Bears", 2, 30, "ARRAY['math']"},
			{actLog4ID, emmaStudentID, activityDef2ID, "Math: Fraction Circles", 3, 35, "ARRAY['math']"},
			{actLog5ID, jamesStudentID, activityDef3ID, "Read Aloud: Paddington Ch 1", 3, 40, "ARRAY['language-arts.reading-comprehension']"},
			{actLog6ID, emmaStudentID, activityDef1ID, "Nature Walk — Backyard Birds", 5, 50, "ARRAY['science']"},
			{actLog7ID, jamesStudentID, activityDef1ID, "Nature Walk — Bug Hunt", 7, 45, "ARRAY['science']"},
			{actLog8ID, emmaStudentID, activityDef3ID, "Read Aloud: Narnia Ch 1", 8, 50, "ARRAY['language-arts.reading-comprehension']"},
			{actLog9ID, jamesStudentID, activityDef2ID, "Math Games: Skip Counting", 10, 25, "ARRAY['math']"},
			{actLog10ID, emmaStudentID, activityDef3ID, "Read Aloud: Charlotte's Web Ch 7", 14, 45, "ARRAY['language-arts.reading-comprehension']"},
		}
		for _, l := range logs {
			date := now.AddDate(0, 0, -l.daysAgo).Format("2006-01-02")
			if err := tx.Exec(fmt.Sprintf(`
				INSERT INTO learn_activity_logs
					(id, family_id, student_id, title, subject_tags,
					 content_id, duration_minutes, activity_date)
				VALUES (?, ?, ?, ?, %s, ?, ?, ?)
				ON CONFLICT (id) DO NOTHING`, l.subjects),
				l.id, seedFamilyID, l.studentID, l.title,
				l.actDefID, l.durationMn, date,
			).Error; err != nil {
				return fmt.Errorf("insert activity log %s: %w", l.id, err)
			}
		}

		// Reading progress
		if err := tx.Exec(`
			INSERT INTO learn_reading_progress
				(id, family_id, student_id, reading_item_id, status, started_at)
			VALUES
				(?, ?, ?, ?, 'in_progress', NOW() - INTERVAL '7 days'),
				(?, ?, ?, ?, 'completed',   NOW() - INTERVAL '21 days'),
				(?, ?, ?, ?, 'in_progress', NOW() - INTERVAL '5 days')
			ON CONFLICT (id) DO NOTHING`,
			readProg1ID, seedFamilyID, emmaStudentID, readingItem1ID,
			readProg2ID, seedFamilyID, emmaStudentID, readingItem3ID,
			readProg3ID, seedFamilyID, jamesStudentID, readingItem2ID,
		).Error; err != nil {
			return fmt.Errorf("insert reading progress: %w", err)
		}

		// Journal entries (for Emma)
		if err := tx.Exec(`
			INSERT INTO learn_journal_entries
				(id, family_id, student_id, entry_type, title, content, subject_tags, entry_date)
			VALUES
				(?, ?, ?, 'narration', 'Charlotte''s Web Chapter 3',
				 'Wilbur was very sad because he had no friends. Then Charlotte spoke to him from her web and said she would be his friend. I think Charlotte is kind because she noticed Wilbur was lonely.',
				 ARRAY['language-arts.reading-comprehension','language-arts'], CURRENT_DATE - 1),
				(?, ?, ?, 'freeform', 'Birds I Saw Today',
				 'I saw a blue jay and two mourning doves in the backyard. The blue jay chased away the smaller birds. I drew them in my nature journal.',
				 ARRAY['science'], CURRENT_DATE - 3)
			ON CONFLICT (id) DO NOTHING`,
			journal1ID, seedFamilyID, emmaStudentID,
			journal2ID, seedFamilyID, emmaStudentID,
		).Error; err != nil {
			return fmt.Errorf("insert journal entries: %w", err)
		}

		// Reading list
		if err := tx.Exec(`
			INSERT INTO learn_reading_lists
				(id, family_id, name, description, student_id)
			VALUES (?, ?, 'Emma''s Books', 'Emma''s current and upcoming reading list', ?)
			ON CONFLICT (id) DO NOTHING`,
			readingList1ID, seedFamilyID, emmaStudentID,
		).Error; err != nil {
			return fmt.Errorf("insert reading list: %w", err)
		}

		// Grading scale
		if err := tx.Exec(`
			INSERT INTO learn_grading_scales
				(id, family_id, name, scale_type, grades, is_default)
			VALUES (?, ?, 'Standard Letter Grades', 'letter',
				'[{"grade":"A","min":90,"label":"Excellent"},{"grade":"B","min":80,"label":"Good"},{"grade":"C","min":70,"label":"Satisfactory"},{"grade":"D","min":60,"label":"Needs Improvement"},{"grade":"F","min":0,"label":"Unsatisfactory"}]',
				true)
			ON CONFLICT (id) DO NOTHING`,
			gradingScale1ID, seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert grading scale: %w", err)
		}

		return nil
	})
}

// ─── Learn Extended seed ──────────────────────────────────────────────────────

func seedLearnExtended(db *gorm.DB, platformPublisherID string) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Assessment definitions
		if err := tx.Exec(`
			INSERT INTO learn_assessment_defs
				(id, publisher_id, title, description, subject_tags, scoring_type, max_score)
			VALUES
				(?, ?, 'Narration Rubric',
				 'Assessment rubric for oral and written narration quality.',
				 ARRAY['language-arts.reading-comprehension','language-arts'], 'percentage', 100),
				(?, ?, 'Spelling Assessment',
				 'Weekly spelling test with progressive difficulty levels.',
				 ARRAY['language-arts'], 'percentage', 100)
			ON CONFLICT (id) DO NOTHING`,
			assessmentDef1ID, platformPublisherID,
			assessmentDef2ID, platformPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert assessment defs: %w", err)
		}

		// Video definition
		if err := tx.Exec(`
			INSERT INTO learn_video_defs
				(id, publisher_id, title, description, subject_tags,
				 duration_seconds, video_url, video_source)
			VALUES (?, ?, 'Introduction to Nature Journaling',
				'A guided introduction to keeping a nature journal in the Charlotte Mason tradition.',
				ARRAY['science'],
				1080, 'https://cdn.example.com/videos/intro-nature-journaling.mp4',
				'self_hosted')
			ON CONFLICT (id) DO NOTHING`,
			videoDef1ID, platformPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert video def: %w", err)
		}

		// Questions for Charlotte's Web comprehension quiz
		type questionRow struct {
			id         string
			qtype      string
			content    string
			answerData string
		}
		questions := []questionRow{
			{question1ID, "multiple_choice",
				"Who is the main character in Charlotte's Web",
				`{"correct_answer":"wilbur","choices":[{"id":"wilbur","text":"Wilbur the pig"},{"id":"charlotte","text":"Charlotte the spider"},{"id":"templeton","text":"Templeton the rat"},{"id":"fern","text":"Fern the girl"}]}`},
			{question2ID, "multiple_choice",
				"What does Charlotte write in her web to save Wilbur",
				`{"correct_answer":"some_pig","choices":[{"id":"some_pig","text":"Some Pig"},{"id":"terrific","text":"Terrific"},{"id":"humble","text":"Humble"},{"id":"radiant","text":"Radiant"}]}`},
			{question3ID, "true_false",
				"Wilbur and Charlotte become close friends",
				`{"correct_answer":"true","choices":[{"id":"true","text":"True"},{"id":"false","text":"False"}]}`},
			{question4ID, "short_answer",
				"Describe in your own words why Charlotte helps Wilbur",
				`{"correct_answer":"","sample_answer":"Charlotte helps Wilbur because she cares about him and does not want him to be killed."}`},
			{question5ID, "multiple_choice",
				"Where does the story take place",
				`{"correct_answer":"farm","choices":[{"id":"farm","text":"On a farm"},{"id":"city","text":"In a city"},{"id":"forest","text":"In a forest"},{"id":"barn","text":"In a barn only"}]}`},
		}
		for _, q := range questions {
			if err := tx.Exec(`
				INSERT INTO learn_questions
					(id, publisher_id, question_type, content, answer_data,
					 subject_tags, difficulty_level, auto_scorable, points)
				VALUES (?, ?, ?, ?, ?::JSONB, ARRAY['language-arts.reading-comprehension','language-arts'], 2, true, 1)
				ON CONFLICT (id) DO NOTHING`,
				q.id, platformPublisherID, q.qtype, q.content, q.answerData,
			).Error; err != nil {
				return fmt.Errorf("insert question %s: %w", q.id, err)
			}
		}

		// Quiz definition
		if err := tx.Exec(`
			INSERT INTO learn_quiz_defs
				(id, publisher_id, title, description, subject_tags,
				 passing_score_percent, question_count, show_correct_after)
			VALUES (?, ?, 'Charlotte''s Web Comprehension Quiz',
				'Five comprehension questions covering the key events and themes of Charlotte''s Web.',
				ARRAY['language-arts.reading-comprehension','language-arts'], 70, 5, true)
			ON CONFLICT (id) DO NOTHING`,
			quizDef1ID, platformPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert quiz def: %w", err)
		}

		// Quiz questions (join table, composite PK)
		quizQs := []struct {
			qID   string
			order int
		}{
			{question1ID, 1},
			{question2ID, 2},
			{question3ID, 3},
			{question4ID, 4},
			{question5ID, 5},
		}
		for _, qq := range quizQs {
			if err := tx.Exec(`
				INSERT INTO learn_quiz_questions (quiz_def_id, question_id, sort_order)
				VALUES (?, ?, ?)
				ON CONFLICT (quiz_def_id, question_id) DO NOTHING`,
				quizDef1ID, qq.qID, qq.order,
			).Error; err != nil {
				return fmt.Errorf("insert quiz question %s: %w", qq.qID, err)
			}
		}

		// Sequence definition
		if err := tx.Exec(`
			INSERT INTO learn_sequence_defs
				(id, publisher_id, title, description, subject_tags, is_linear)
			VALUES (?, ?, 'Beginning Reading Sequence',
				'A progressive three-book reading sequence for early readers.',
				ARRAY['language-arts.reading-comprehension','language-arts'], true)
			ON CONFLICT (id) DO NOTHING`,
			sequenceDef1ID, platformPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert sequence def: %w", err)
		}

		// Sequence items (own UUID PK)
		seqItems := []struct {
			id          string
			order       int
			contentType string
			contentID   string
		}{
			{sequenceItem1ID, 1, "reading_item", readingItem2ID},
			{sequenceItem2ID, 2, "reading_item", readingItem1ID},
			{sequenceItem3ID, 3, "reading_item", readingItem3ID},
		}
		for _, si := range seqItems {
			if err := tx.Exec(`
				INSERT INTO learn_sequence_items
					(id, sequence_def_id, sort_order, content_type, content_id, is_required)
				VALUES (?, ?, ?, ?, ?, true)
				ON CONFLICT (id) DO NOTHING`,
				si.id, sequenceDef1ID, si.order, si.contentType, si.contentID,
			).Error; err != nil {
				return fmt.Errorf("insert sequence item %s: %w", si.id, err)
			}
		}

		// Reading list items: add all 3 reading items to Emma's list
		listItems := []struct {
			itemID string
			order  int
		}{
			{readingItem1ID, 1},
			{readingItem2ID, 2},
			{readingItem3ID, 3},
		}
		for _, li := range listItems {
			if err := tx.Exec(`
				INSERT INTO learn_reading_list_items
					(reading_list_id, reading_item_id, sort_order)
				VALUES (?, ?, ?)
				ON CONFLICT (reading_list_id, reading_item_id) DO NOTHING`,
				readingList1ID, li.itemID, li.order,
			).Error; err != nil {
				return fmt.Errorf("insert reading list item %s: %w", li.itemID, err)
			}
		}

		// Student assignments for Emma
		now := time.Now()
		dueDate1 := now.AddDate(0, 0, 7).Format("2006-01-02")
		dueDate2 := now.AddDate(0, 0, 14).Format("2006-01-02")
		if err := tx.Exec(`
			INSERT INTO learn_student_assignments
				(id, family_id, student_id, assigned_by, content_type, content_id,
				 due_date, status)
			VALUES
				(?, ?, ?, ?, 'quiz_def', ?, ?, 'assigned'),
				(?, ?, ?, ?, 'activity_def', ?, ?, 'assigned')
			ON CONFLICT (id) DO NOTHING`,
			studentAssign1ID, seedFamilyID, emmaStudentID, seedParentID,
			quizDef1ID, dueDate1,
			studentAssign2ID, seedFamilyID, emmaStudentID, seedParentID,
			activityDef3ID, dueDate2,
		).Error; err != nil {
			return fmt.Errorf("insert student assignments: %w", err)
		}

		// Progress snapshots for Emma (2 monthly snapshots)
		snap1Date := now.AddDate(0, -2, 0).Format("2006-01-02")
		snap2Date := now.AddDate(0, -1, 0).Format("2006-01-02")
		snap1Data := `{"activities_logged":18,"books_completed":1,"journal_entries":6,"total_minutes":480}`
		snap2Data := `{"activities_logged":22,"books_completed":2,"journal_entries":8,"total_minutes":610}`
		if err := tx.Exec(`
			INSERT INTO learn_progress_snapshots
				(id, family_id, student_id, snapshot_date, data)
			VALUES
				(?, ?, ?, ?, ?::JSONB),
				(?, ?, ?, ?, ?::JSONB)
			ON CONFLICT DO NOTHING`,
			progressSnapshot1ID, seedFamilyID, emmaStudentID, snap1Date, snap1Data,
			progressSnapshot2ID, seedFamilyID, emmaStudentID, snap2Date, snap2Data,
		).Error; err != nil {
			return fmt.Errorf("insert progress snapshots: %w", err)
		}

		return nil
	})
}

// ─── Discovery seed ───────────────────────────────────────────────────────────

// discAnswer holds an answer option with methodology weight scores.
type discAnswer struct {
	ID      string             `json:"id"`
	Text    string             `json:"text"`
	Weights map[string]float64 `json:"weights"`
}

// discQuestion holds a quiz question with its answer options.
type discQuestion struct {
	ID       string       `json:"id"`
	Category string       `json:"category"`
	Text     string       `json:"text"`
	HelpText string       `json:"help_text"`
	Answers  []discAnswer `json:"answers"`
}

func seedDiscovery(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Build questions JSONB in Go to avoid any ? substitution issues
		questions := []discQuestion{
			{
				ID: "q1", Category: "scheduling",
				Text:     "How do you approach daily learning time",
				HelpText: "Think about your typical school day.",
				Answers: []discAnswer{
					{ID: "q1a1", Text: "Highly structured with set times for each subject",
						Weights: map[string]float64{"traditional": 1.0, "classical": 0.8}},
					{ID: "q1a2", Text: "Loosely structured around themes and living books",
						Weights: map[string]float64{"charlotte-mason": 0.9, "waldorf": 0.8}},
					{ID: "q1a3", Text: "Entirely child-led and interest-driven",
						Weights: map[string]float64{"unschooling": 1.0, "montessori": 0.7}},
				},
			},
			{
				ID: "q2", Category: "outdoor",
				Text:     "How important is nature-based outdoor time in your approach",
				HelpText: "Consider how often you incorporate outdoor learning.",
				Answers: []discAnswer{
					{ID: "q2a1", Text: "Central — nature study is a daily part of learning",
						Weights: map[string]float64{"charlotte-mason": 1.0, "waldorf": 0.7, "unschooling": 0.5}},
					{ID: "q2a2", Text: "Occasional — we do nature activities when convenient",
						Weights: map[string]float64{"traditional": 0.4, "classical": 0.3}},
					{ID: "q2a3", Text: "Not a priority — we focus on indoor academics",
						Weights: map[string]float64{"traditional": 0.2}},
				},
			},
			{
				ID: "q3", Category: "assessment",
				Text:     "How do you view formal tests and grades",
				HelpText: "Think about how you measure your child's progress.",
				Answers: []discAnswer{
					{ID: "q3a1", Text: "Important for college preparation and accountability",
						Weights: map[string]float64{"classical": 0.9, "traditional": 1.0}},
					{ID: "q3a2", Text: "I prefer portfolios, narration, and project work",
						Weights: map[string]float64{"charlotte-mason": 0.8, "waldorf": 0.7}},
					{ID: "q3a3", Text: "We do not use grades — learning speaks for itself",
						Weights: map[string]float64{"unschooling": 1.0, "montessori": 0.6}},
				},
			},
			{
				ID: "q4", Category: "arts",
				Text:     "What role should arts play in academics",
				HelpText: "Consider how arts are woven into your learning.",
				Answers: []discAnswer{
					{ID: "q4a1", Text: "Deeply integrated — arts enhance all subjects",
						Weights: map[string]float64{"waldorf": 1.0, "charlotte-mason": 0.8, "montessori": 0.6}},
					{ID: "q4a2", Text: "A separate subject with dedicated time",
						Weights: map[string]float64{"traditional": 0.7, "classical": 0.5}},
					{ID: "q4a3", Text: "Child-directed — arts when children choose",
						Weights: map[string]float64{"unschooling": 0.9}},
				},
			},
			{
				ID: "q5", Category: "curriculum",
				Text:     "How do you feel about textbooks as learning tools",
				HelpText: "Think about the materials you prefer to use.",
				Answers: []discAnswer{
					{ID: "q5a1", Text: "They provide structure and a solid academic foundation",
						Weights: map[string]float64{"traditional": 1.0, "classical": 0.7}},
					{ID: "q5a2", Text: "I prefer living books and real literature",
						Weights: map[string]float64{"charlotte-mason": 1.0}},
					{ID: "q5a3", Text: "Real-world learning and experience over textbooks",
						Weights: map[string]float64{"unschooling": 0.9, "montessori": 0.7}},
				},
			},
		}

		questionsJSON, err := json.Marshal(questions)
		if err != nil {
			return fmt.Errorf("marshal questions: %w", err)
		}

		explanations := map[string]map[string]string{
			"charlotte-mason": {
				"match_text":    "Charlotte Mason methodology aligns beautifully with your approach — living books, nature study, and narration.",
				"mismatch_text": "Charlotte Mason emphasizes living books and nature study, which differs from your current approach.",
			},
			"classical": {
				"match_text":    "Classical education matches your structured, academically rigorous approach.",
				"mismatch_text": "Classical education uses the trivium (grammar, logic, rhetoric) which may differ from your preferences.",
			},
			"unschooling": {
				"match_text":    "Unschooling aligns with your child-led, interest-driven philosophy.",
				"mismatch_text": "Unschooling removes all formal structure, which may differ from your preferences.",
			},
			"waldorf": {
				"match_text":    "Waldorf education's arts-integrated, seasonal approach resonates with your values.",
				"mismatch_text": "Waldorf emphasizes arts and imagination over early academics.",
			},
			"montessori": {
				"match_text":    "Montessori's child-paced, hands-on environment fits your learning style.",
				"mismatch_text": "Montessori uses prepared environments and child-paced progression.",
			},
			"traditional": {
				"match_text":    "Traditional education's structured curriculum and clear standards match your approach.",
				"mismatch_text": "Traditional education uses textbooks and standardized grading.",
			},
		}
		explanationsJSON, err := json.Marshal(explanations)
		if err != nil {
			return fmt.Errorf("marshal explanations: %w", err)
		}

		// Insert quiz definition (active — only one allowed at a time)
		if err := tx.Exec(`
			INSERT INTO disc_quiz_definitions
				(id, version, title, description, status, questions, explanations)
			VALUES (?, 1,
				'Homeschool Methodology Quiz',
				'Discover which homeschooling methodology aligns best with your family values and goals.',
				'active', ?::JSONB, ?::JSONB)
			ON CONFLICT (id) DO NOTHING`,
			discQuizDefID, string(questionsJSON), string(explanationsJSON),
		).Error; err != nil {
			return fmt.Errorf("insert quiz definition: %w", err)
		}

		// Quiz result for seed family (answers favoring charlotte-mason)
		answers := map[string]string{
			"q1": "q1a2", // loosely structured → charlotte-mason
			"q2": "q2a1", // central outdoor → charlotte-mason
			"q3": "q3a2", // portfolios/narration → charlotte-mason
			"q4": "q4a1", // arts integrated → waldorf/charlotte-mason
			"q5": "q5a2", // living books → charlotte-mason
		}
		scores := map[string]float64{
			"charlotte-mason": 0.90,
			"waldorf":         0.64,
			"unschooling":     0.10,
			"classical":       0.00,
			"traditional":     0.00,
			"montessori":      0.12,
		}
		recommendations := []map[string]any{
			{
				"slug":             "charlotte-mason",
				"display_name":     "Charlotte Mason",
				"score_percentage": 90,
				"explanation":      "Charlotte Mason methodology aligns beautifully with your approach — living books, nature study, and narration.",
			},
		}

		answersJSON, err := json.Marshal(answers)
		if err != nil {
			return fmt.Errorf("marshal answers: %w", err)
		}
		scoresJSON, err := json.Marshal(scores)
		if err != nil {
			return fmt.Errorf("marshal scores: %w", err)
		}
		recsJSON, err := json.Marshal(recommendations)
		if err != nil {
			return fmt.Errorf("marshal recommendations: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO disc_quiz_results
				(id, quiz_definition_id, share_id, answers, scores, recommendations, family_id)
			VALUES (?, ?, 'SdF3kQ9mPxL7', ?::JSONB, ?::JSONB, ?::JSONB, ?)
			ON CONFLICT (id) DO NOTHING`,
			discQuizResult1ID, discQuizDefID,
			string(answersJSON), string(scoresJSON), string(recsJSON),
			seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert quiz result: %w", err)
		}

		return nil
	})
}

// ─── Notifications seed ────────────────────────────────────────────────────────

func seedNotifications(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO notify_notifications
				(id, family_id, notification_type, category, title, body, is_read)
			VALUES
				(?, ?, 'onboarding_completed',      'system',      'Welcome to Homegrown Academy!',      'Your account setup is complete. Explore the platform and start your first lesson.',           true),
				(?, ?, 'purchase_completed',         'marketplace', 'Purchase Complete',                  'Charlotte Mason Year 1 Curriculum Guide is now in your library.',                           true),
				(?, ?, 'friend_request_accepted',    'social',      'Friend Request Accepted',            'The Friend Family accepted your friend request.',                                           true),
				(?, ?, 'activity_streak',            'learning',    '7-Day Learning Streak!',             'Amazing! You''ve logged activities for 7 days in a row. Keep it up!',                      false),
				(?, ?, 'book_completed',             'learning',    'Book Completed',                     'Emma has finished reading The Lion, the Witch and the Wardrobe. Great job!',                false)
			ON CONFLICT (id) DO NOTHING`,
			notif1ID, seedFamilyID,
			notif2ID, seedFamilyID,
			notif3ID, seedFamilyID,
			notif4ID, seedFamilyID,
			notif5ID, seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert notifications: %w", err)
		}

		// Preferences (2 overrides for seed family)
		if err := tx.Exec(`
			INSERT INTO notify_preferences
				(id, family_id, notification_type, channel, enabled, digest_frequency)
			VALUES
				(?, ?, 'message_received', 'email',  false, 'off'),
				(?, ?, 'activity_streak',  'in_app', true,  'weekly')
			ON CONFLICT (id) DO NOTHING`,
			pref1ID, seedFamilyID,
			pref2ID, seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert notification preferences: %w", err)
		}

		return nil
	})
}

// ─── Billing seed ─────────────────────────────────────────────────────────────

func seedBilling(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		now := time.Now()
		periodStart := now.Format(time.RFC3339)
		periodEnd := now.AddDate(0, 1, 0).Format(time.RFC3339)

		// Hyperswitch customer
		if err := tx.Exec(`
			INSERT INTO bill_hyperswitch_customers
				(family_id, hyperswitch_customer_id)
			VALUES (?, 'cus_seed_family_001')
			ON CONFLICT (family_id) DO NOTHING`,
			seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert billing customer: %w", err)
		}

		// Subscription
		if err := tx.Exec(`
			INSERT INTO bill_subscriptions
				(id, family_id, hyperswitch_subscription_id, hyperswitch_customer_id,
				 tier, status, billing_interval,
				 current_period_start, current_period_end,
				 amount_cents, hyperswitch_price_id)
			VALUES (?, ?, 'sub_seed_premium_001', 'cus_seed_family_001',
				'premium', 'active', 'monthly',
				?, ?,
				999, 'price_premium_monthly_001')
			ON CONFLICT (id) DO NOTHING`,
			subID, seedFamilyID, periodStart, periodEnd,
		).Error; err != nil {
			return fmt.Errorf("insert subscription: %w", err)
		}

		// Transaction
		if err := tx.Exec(`
			INSERT INTO bill_transactions
				(id, family_id, transaction_type, status,
				 amount_cents, hyperswitch_payment_id, description)
			VALUES (?, ?, 'subscription_payment', 'succeeded',
				999, 'pay_seed_001', 'Monthly premium subscription')
			ON CONFLICT (id) DO NOTHING`,
			txnID, seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}

		return nil
	})
}

// ─── Media seed ───────────────────────────────────────────────────────────────

func seedMedia(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO media_uploads
				(id, family_id, uploaded_by, context, status,
				 original_filename, content_type, storage_key, published_at)
			VALUES
				(?, ?, ?, 'profile_photo', 'published',
				 'family-photo.jpg', 'image/jpeg', 'seed/profile/family-photo.jpg', NOW() - INTERVAL '30 days'),
				(?, ?, ?, 'post_attachment', 'published',
				 'nature-walk.jpg', 'image/jpeg', 'seed/posts/nature-walk.jpg',    NOW() - INTERVAL '7 days')
			ON CONFLICT (storage_key) DO NOTHING`,
			media1ID, seedFamilyID, seedParentID,
			media2ID, seedFamilyID, seedParentID,
		).Error; err != nil {
			return fmt.Errorf("insert media uploads: %w", err)
		}

		return nil
	})
}

// ─── Safety Full seed ──────────────────────────────────────────────────────────
// Replaces the minimal seedSafety with a full moderation scenario.

func seedSafetyFull(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Bot signals on friend family (rapid posting + suspicious registration)
		botDetails1 := `{"post_count":8,"window_minutes":5}`
		botDetails2 := `{"ip_reuse":true}`
		if err := tx.Exec(`
			INSERT INTO safety_bot_signals
				(id, family_id, parent_id, signal_type, details)
			VALUES
				(?, ?, ?, 'rapid_posting',           ?::JSONB),
				(?, ?, ?, 'suspicious_registration', ?::JSONB)
			ON CONFLICT (id) DO NOTHING`,
			safetyBotSignal1ID, friendFamilyID, friendParentID, botDetails1,
			safetyBotSignal2ID, friendFamilyID, friendParentID, botDetails2,
		).Error; err != nil {
			return fmt.Errorf("insert bot signals: %w", err)
		}

		// Content flag (automated) on friend's post3
		flagLabels := `{"spam_keywords":["buy now","limited offer"]}`
		if err := tx.Exec(`
			INSERT INTO safety_content_flags
				(id, source, target_type, target_id, target_family_id,
				 flag_type, confidence, labels, auto_rejected)
			VALUES (?, 'automated', 'post', ?, ?, 'spam', 0.85, ?::JSONB, false)
			ON CONFLICT (id) DO NOTHING`,
			safetyFlag1ID, post3ID, friendFamilyID, flagLabels,
		).Error; err != nil {
			return fmt.Errorf("insert content flag: %w", err)
		}

		// Report (seed family reports friend's post3)
		if err := tx.Exec(`
			INSERT INTO safety_reports
				(id, reporter_family_id, reporter_parent_id,
				 target_type, target_id, target_family_id,
				 category, description, priority, status, assigned_admin_id)
			VALUES (?, ?, ?, 'post', ?, ?,
				'spam', 'This post appears to be spam content promoting a product.',
				'normal', 'resolved_action_taken', ?)
			ON CONFLICT (id) DO NOTHING`,
			safetyReport1ID, seedFamilyID, seedParentID,
			post3ID, friendFamilyID, adminParentID,
		).Error; err != nil {
			return fmt.Errorf("insert safety report: %w", err)
		}

		// Mod action (admin issues warning to friend family)
		if err := tx.Exec(`
			INSERT INTO safety_mod_actions
				(id, admin_id, target_family_id, target_parent_id,
				 action_type, reason, report_id)
			VALUES (?, ?, ?, ?,
				'warning_issued',
				'Spam content posted to community group. First offense — warning issued.',
				?)
			ON CONFLICT (id) DO NOTHING`,
			safetyModAction1ID, adminParentID, friendFamilyID, friendParentID,
			safetyReport1ID,
		).Error; err != nil {
			return fmt.Errorf("insert mod action: %w", err)
		}

		// Appeal (friend family appeals the warning)
		if err := tx.Exec(`
			INSERT INTO safety_appeals
				(id, family_id, action_id, appeal_text, status)
			VALUES (?, ?, ?,
				'This was not spam — I was sharing a co-op supply list with our group members.',
				'pending')
			ON CONFLICT (id) DO NOTHING`,
			safetyAppeal1ID, friendFamilyID, safetyModAction1ID,
		).Error; err != nil {
			return fmt.Errorf("insert safety appeal: %w", err)
		}

		// Account statuses (both families remain active)
		if err := tx.Exec(`
			INSERT INTO safety_account_status (family_id, status)
			VALUES
				(?, 'active'),
				(?, 'active')
			ON CONFLICT (family_id) DO NOTHING`,
			seedFamilyID, friendFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert safety account status: %w", err)
		}

		return nil
	})
}

// ─── Admin Domain seed ─────────────────────────────────────────────────────────

func seedAdminDomain(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Feature flags (5 flags, all created by admin parent)
		type flagRow struct {
			id      string
			key     string
			enabled bool
			rollout *int
			desc    string
		}
		r50 := 50
		r100 := 100
		r10 := 10
		r75 := 75
		flags := []flagRow{
			{adminFlag1ID, "ai_lesson_generator", true, &r50,
				"AI-powered lesson plan generation"},
			{adminFlag2ID, "advanced_analytics", false, nil,
				"Family learning analytics dashboard"},
			{adminFlag3ID, "marketplace_creator_tools", true, &r100,
				"Enhanced creator dashboard and sales analytics"},
			{adminFlag4ID, "social_live_events", false, &r10,
				"Live video events for co-op groups (beta)"},
			{adminFlag5ID, "parent_co_op_tools", true, &r75,
				"Co-op scheduling and resource sharing tools"},
		}
		for _, f := range flags {
			var rolloutSQL string
			var args []any
			if f.rollout != nil {
				rolloutSQL = "?"
				args = []any{f.id, f.key, f.desc, f.enabled, *f.rollout, adminParentID}
			} else {
				rolloutSQL = "NULL"
				args = []any{f.id, f.key, f.desc, f.enabled, adminParentID}
			}
			query := fmt.Sprintf(`
				INSERT INTO admin_feature_flags
					(id, key, description, enabled, rollout_percentage, created_by)
				VALUES (?, ?, ?, ?, %s, ?)
				ON CONFLICT (id) DO NOTHING`, rolloutSQL)
			if err := tx.Exec(query, args...).Error; err != nil {
				return fmt.Errorf("insert feature flag %s: %w", f.key, err)
			}
		}

		// Audit log entries (3 actions by admin)
		type auditRow struct {
			id         string
			action     string
			targetType string
			targetID   string
			details    string
		}
		auditLogs := []auditRow{
			{
				adminAuditLog1ID, "flag_create", "feature_flag", adminFlag1ID,
				`{"key":"ai_lesson_generator","enabled":true}`,
			},
			{
				adminAuditLog2ID, "user_suspend", "family", friendFamilyID,
				`{"reason":"Spam content posted","duration_days":0,"action":"warning_issued"}`,
			},
			{
				adminAuditLog3ID, "content_remove", "post", post3ID,
				`{"reason":"Spam content removed after report"}`,
			},
		}
		for _, a := range auditLogs {
			if err := tx.Exec(`
				INSERT INTO admin_audit_log
					(id, admin_id, action, target_type, target_id, details)
				VALUES (?, ?, ?, ?, ?, ?::JSONB)
				ON CONFLICT (id) DO NOTHING`,
				a.id, adminParentID, a.action, a.targetType, a.targetID, a.details,
			).Error; err != nil {
				return fmt.Errorf("insert audit log %s: %w", a.id, err)
			}
		}

		return nil
	})
}

// ─── Comply seed ──────────────────────────────────────────────────────────────

func seedComply(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// 1. State configs (platform reference data, PK = state_code)
		if err := tx.Exec(`
			INSERT INTO comply_state_configs
				(state_code, state_name, regulation_level,
				 notification_required, required_subjects,
				 assessment_required, attendance_required, attendance_days)
			VALUES
				('TX', 'Texas',      'low',
				 false, '{}', false, false, NULL),
				('CA', 'California', 'moderate',
				 true,
				 ARRAY['english','mathematics','social_studies','science',
				       'art','music','health','physical_education'],
				 false, false, NULL),
				('NY', 'New York',   'high',
				 true,
				 ARRAY['english','mathematics','science','social_studies',
				       'history','geography','health','physical_education',
				       'art','music','library'],
				 true, true, 180)
			ON CONFLICT (state_code) DO NOTHING`,
		).Error; err != nil {
			return fmt.Errorf("insert state configs: %w", err)
		}

		// 2. Custom schedule for seed family
		exclusionPeriods := `[{"label":"Summer Break","start":"2026-06-01","end":"2026-08-15"}]`
		if err := tx.Exec(`
			INSERT INTO comply_custom_schedules
				(id, family_id, name, school_days, exclusion_periods)
			VALUES (?, ?, 'Standard Week',
				'{true,true,true,true,true,false,false}',
				?::JSONB)
			ON CONFLICT (id) DO NOTHING`,
			complySchedule1ID, seedFamilyID, exclusionPeriods,
		).Error; err != nil {
			return fmt.Errorf("insert custom schedule: %w", err)
		}

		// 3. Family config (seed family in TX)
		if err := tx.Exec(`
			INSERT INTO comply_family_configs
				(family_id, state_code, school_year_start, school_year_end,
				 total_school_days, custom_schedule_id, gpa_scale)
			VALUES (?, 'TX', '2025-08-25', '2026-05-22', 180, ?, 'standard_4')
			ON CONFLICT (family_id) DO NOTHING`,
			seedFamilyID, complySchedule1ID,
		).Error; err != nil {
			return fmt.Errorf("insert family config: %w", err)
		}

		// 4. Attendance (10 records: Emma x5, James x5, no date collisions)
		now := time.Now()
		type attendRec struct {
			id        string
			studentID string
			daysAgo   int
			status    string
		}
		attendance := []attendRec{
			{complyAttend1ID, emmaStudentID, 14, "present_full"},
			{complyAttend2ID, emmaStudentID, 11, "present_full"},
			{complyAttend3ID, emmaStudentID, 8, "present_full"},
			{complyAttend4ID, emmaStudentID, 5, "present_full"},
			{complyAttend5ID, emmaStudentID, 2, "absent"},
			{complyAttend6ID, jamesStudentID, 13, "present_full"},
			{complyAttend7ID, jamesStudentID, 10, "present_full"},
			{complyAttend8ID, jamesStudentID, 7, "present_full"},
			{complyAttend9ID, jamesStudentID, 4, "present_full"},
			{complyAttend10ID, jamesStudentID, 1, "absent"},
		}
		for _, a := range attendance {
			date := now.AddDate(0, 0, -a.daysAgo).Format("2006-01-02")
			if err := tx.Exec(`
				INSERT INTO comply_attendance
					(id, family_id, student_id, attendance_date, status)
				VALUES (?, ?, ?, ?, ?)
				ON CONFLICT DO NOTHING`,
				a.id, seedFamilyID, a.studentID, date, a.status,
			).Error; err != nil {
				return fmt.Errorf("insert attendance %s: %w", a.id, err)
			}
		}

		// 5. Assessment records for Emma (4 records)
		type assessRec struct {
			id      string
			title   string
			subject string
			atype   string
			score   float64
			max     float64
			grade   string
			daysAgo int
		}
		assessments := []assessRec{
			{complyAssess1ID, "Charlotte's Web Narration", "english", "assignment", 92, 100, "A", 30},
			{complyAssess2ID, "Fractions Quiz", "mathematics", "quiz", 85, 100, "B", 21},
			{complyAssess3ID, "Nature Journal Review", "science", "project", 95, 100, "A", 14},
			{complyAssess4ID, "Spelling Test Week 12", "english", "test", 78, 100, "C", 7},
		}
		for _, a := range assessments {
			date := now.AddDate(0, 0, -a.daysAgo).Format("2006-01-02")
			if err := tx.Exec(`
				INSERT INTO comply_assessment_records
					(id, family_id, student_id, title, subject, assessment_type,
					 score, max_score, grade_letter, assessment_date)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (id) DO NOTHING`,
				a.id, seedFamilyID, emmaStudentID,
				a.title, a.subject, a.atype, a.score, a.max, a.grade, date,
			).Error; err != nil {
				return fmt.Errorf("insert assessment %s: %w", a.id, err)
			}
		}

		// 6. Courses for Emma (transcript prep)
		type courseRec struct {
			id          string
			title       string
			subject     string
			gradeLetter string
			gradePoints float64
			credits     float64
		}
		courses := []courseRec{
			{complyCourse1ID, "Literature", "english", "A", 4.0, 1.0},
			{complyCourse2ID, "Mathematics", "mathematics", "B", 3.0, 1.0},
			{complyCourse3ID, "Nature Science", "science", "A", 4.0, 0.5},
		}
		for _, c := range courses {
			if err := tx.Exec(`
				INSERT INTO comply_courses
					(id, family_id, student_id, title, subject, grade_level,
					 credits, grade_letter, grade_points, level, school_year, semester)
				VALUES (?, ?, ?, ?, ?, 5, ?, ?, ?, 'regular', '2025-2026', 'full_year')
				ON CONFLICT (id) DO NOTHING`,
				c.id, seedFamilyID, emmaStudentID,
				c.title, c.subject, c.credits, c.gradeLetter, c.gradePoints,
			).Error; err != nil {
				return fmt.Errorf("insert course %s: %w", c.id, err)
			}
		}

		// 7. Transcript (draft, configuring status)
		if err := tx.Exec(`
			INSERT INTO comply_transcripts
				(id, family_id, student_id, title, student_name, grade_levels, status)
			VALUES (?, ?, ?, 'Emma — 2025-2026 Year-End Transcript',
				'Emma', ARRAY['5'], 'configuring')
			ON CONFLICT (id) DO NOTHING`,
			complyTranscript1ID, seedFamilyID, emmaStudentID,
		).Error; err != nil {
			return fmt.Errorf("insert transcript: %w", err)
		}

		// 8. Portfolio (configuring)
		if err := tx.Exec(`
			INSERT INTO comply_portfolios
				(id, family_id, student_id, title, organization,
				 date_range_start, date_range_end,
				 include_attendance, include_assessments, status)
			VALUES (?, ?, ?, 'Emma Spring 2026 Portfolio', 'by_subject',
				'2026-01-01', '2026-05-22', true, true, 'configuring')
			ON CONFLICT (id) DO NOTHING`,
			complyPortfolio1ID, seedFamilyID, emmaStudentID,
		).Error; err != nil {
			return fmt.Errorf("insert portfolio: %w", err)
		}

		// 9. Portfolio items (3 items from different source types)
		actDate := now.AddDate(0, 0, -1).Format("2006-01-02")     // actLog1 date
		journalDate := now.AddDate(0, 0, -3).Format("2006-01-02") // journal2 date
		assessDate := now.AddDate(0, 0, -30).Format("2006-01-02") // complyAssess1 date
		portfolioItems := []struct {
			id         string
			sourceType string
			sourceID   string
			title      string
			order      int
			cachedDate string
		}{
			{complyPortfolioItem1ID, "activity", actLog1ID,
				"Nature Walk — Creek Trail", 1, actDate},
			{complyPortfolioItem2ID, "journal", journal1ID,
				"Charlotte's Web Chapter 3", 2, journalDate},
			{complyPortfolioItem3ID, "assessment", complyAssess1ID,
				"Charlotte's Web Narration", 3, assessDate},
		}
		for _, item := range portfolioItems {
			if err := tx.Exec(`
				INSERT INTO comply_portfolio_items
					(id, portfolio_id, source_type, source_id,
					 display_order, cached_title, cached_date)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (id) DO NOTHING`,
				item.id, complyPortfolio1ID, item.sourceType, item.sourceID,
				item.order, item.title, item.cachedDate,
			).Error; err != nil {
				return fmt.Errorf("insert portfolio item %s: %w", item.id, err)
			}
		}

		// 10. Standardized test for Emma
		scoresJSON := `{"reading":85,"math":88,"language":82,"science":91}`
		if err := tx.Exec(`
			INSERT INTO comply_standardized_tests
				(id, family_id, student_id, test_name, test_date,
				 grade_level, scores, composite_score, percentile)
			VALUES (?, ?, ?, 'California Achievement Test (CAT)', '2025-11-15',
				5, ?::JSONB, 86.5, 73)
			ON CONFLICT (id) DO NOTHING`,
			complyStdTest1ID, seedFamilyID, emmaStudentID, scoresJSON,
		).Error; err != nil {
			return fmt.Errorf("insert standardized test: %w", err)
		}

		return nil
	})
}

// ─── Recommendations seed ─────────────────────────────────────────────────────

func seedRecs(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		expiresAt := time.Now().AddDate(0, 0, 14).Format(time.RFC3339)

		if err := tx.Exec(`
			INSERT INTO recs_recommendations
				(id, family_id, recommendation_type, target_entity_id,
				 target_entity_label, source_signal, source_label, score, status, expires_at)
			VALUES
				(?, ?, 'marketplace_content', ?,
				 'Charlotte Mason Year 1 Curriculum Guide', 'methodology_match', 'Matches your charlotte-mason methodology', 0.95, 'active', ?),
				(?, ?, 'marketplace_content', ?,
				 'Nature Journal Starter Pack', 'methodology_match', 'Popular with charlotte-mason families', 0.88, 'active', ?),
				(?, ?, 'reading_suggestion', ?,
				 'Charlotte''s Web', 'reading_history', 'Based on Emma''s reading history', 0.82, 'active', ?)
			ON CONFLICT DO NOTHING`,
			rec1ID, seedFamilyID, listing1ID, expiresAt,
			rec2ID, seedFamilyID, listing2ID, expiresAt,
			rec3ID, seedFamilyID, readingItem1ID, expiresAt,
		).Error; err != nil {
			return fmt.Errorf("insert recommendations: %w", err)
		}

		// Preferences
		if err := tx.Exec(`
			INSERT INTO recs_preferences
				(id, family_id, enabled_types, exploration_frequency)
			VALUES (?, ?,
				ARRAY['marketplace_content','activity_idea','reading_suggestion','community_group'],
				'occasional')
			ON CONFLICT DO NOTHING`,
			recPrefID, seedFamilyID,
		).Error; err != nil {
			return fmt.Errorf("insert recs preferences: %w", err)
		}

		return nil
	})
}

// ─── Recommendations Extended seed ───────────────────────────────────────────

func seedRecsExtended(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		now := time.Now()

		// 3 learning signals for seed family
		// signal_type values: 'activity_logged', 'book_completed', 'purchase_completed'
		type signalRow struct {
			id         string
			signalType string
			entityID   string
			daysAgo    int
		}
		signals := []signalRow{
			{recsSignal1ID, "activity_logged", actLog1ID, 0},
			{recsSignal2ID, "purchase_completed", listing1ID, 7},
			{recsSignal3ID, "book_completed", readingItem3ID, 14},
		}
		for _, s := range signals {
			date := now.AddDate(0, 0, -s.daysAgo).Format("2006-01-02")
			payload := fmt.Sprintf(`{"entity_id":"%s"}`, s.entityID)
			if err := tx.Exec(`
				INSERT INTO recs_signals
					(id, family_id, signal_type, methodology_slug, payload, signal_date)
				VALUES (?, ?, ?, 'charlotte-mason', ?::JSONB, ?)
				ON CONFLICT (id) DO NOTHING`,
				s.id, seedFamilyID, s.signalType, payload, date,
			).Error; err != nil {
				return fmt.Errorf("insert recs signal %s: %w", s.id, err)
			}
		}

		// 2 feedback records: dismiss rec1, block rec2
		// action values: 'dismiss', 'block'
		if err := tx.Exec(`
			INSERT INTO recs_recommendation_feedback
				(id, family_id, recommendation_id, action)
			VALUES
				(?, ?, ?, 'dismiss'),
				(?, ?, ?, 'block')
			ON CONFLICT (recommendation_id) DO NOTHING`,
			recsFeedback1ID, seedFamilyID, rec1ID,
			recsFeedback2ID, seedFamilyID, rec2ID,
		).Error; err != nil {
			return fmt.Errorf("insert recs feedback: %w", err)
		}

		return nil
	})
}

// ─── Learner Profile seed ─────────────────────────────────────────────────────

func seedLearnerProfile(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Emma's learner profile — hands-on, outdoor, long sessions
		if err := tx.Exec(`
			INSERT INTO learner_profiles
				(id, family_id, student_id,
				 activity_format, session_length, motivation,
				 solo_collaborative, structure, outdoor_kinesthetic,
				 interests, answered_count, confidence, source, respondent)
			VALUES (?, ?, ?,
				0.85, 0.55, 0.70,
				0.30, 0.40, 0.90,
				ARRAY['science','building','outdoors'],
				12, 0.850, 'declared', 'parent')
			ON CONFLICT (student_id) DO NOTHING`,
			learnerProfile1ID, seedFamilyID, emmaStudentID,
		).Error; err != nil {
			return fmt.Errorf("insert learner profile: %w", err)
		}

		// A recommendation with fit_score so QA can verify the FitBadge component.
		expiresAt := time.Now().AddDate(0, 0, 14).Format(time.RFC3339)
		if err := tx.Exec(`
			INSERT INTO recs_recommendations
				(id, family_id, recommendation_type, target_entity_id,
				 target_entity_label, source_signal, source_label, score,
				 fit_score, fit_why, status, expires_at)
			VALUES (?, ?, 'marketplace_content', ?,
				'Living Books Read-Aloud Video Series',
				'learner_profile_fit', 'Matches Emma''s hands-on outdoor learner profile',
				0.88, 0.700, 'Great hands-on outdoor match for Emma', 'active', ?)
			ON CONFLICT DO NOTHING`,
			rec4ID, seedFamilyID, listing3ID, expiresAt,
		).Error; err != nil {
			return fmt.Errorf("insert fit-badge recommendation: %w", err)
		}

		return nil
	})
}

// ─── Planning seed ────────────────────────────────────────────────────────────

func seedPlan(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		now := time.Now()

		type schedItem struct {
			id        string
			studentID *string
			title     string
			daysOut   int
			startTime string
			endTime   string
			category  string
			color     string
		}
		emmaID := emmaStudentID
		jamesID := jamesStudentID
		items := []schedItem{
			{schedItem1ID, &emmaID, "Read Aloud: Charlotte's Web", 0, "09:00", "09:45", "reading", "#10B981"},
			{schedItem2ID, &emmaID, "Math: Fraction Practice", 0, "10:00", "10:35", "lesson", "#3B82F6"},
			{schedItem3ID, &jamesID, "Math Games", 1, "09:00", "09:30", "activity", "#3B82F6"},
			{schedItem4ID, nil, "Charlotte Mason Co-op", 2, "10:00", "13:00", "co_op", "#8B5CF6"},
			{schedItem5ID, nil, "Afternoon Free Time", 3, "14:00", "15:30", "break", "#F59E0B"},
		}

		for _, item := range items {
			date := now.AddDate(0, 0, item.daysOut).Format("2006-01-02")
			if item.studentID != nil {
				if err := tx.Exec(`
					INSERT INTO plan_schedule_items
						(id, family_id, student_id, title, start_date,
						 start_time, end_time, category, color)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
					ON CONFLICT (id) DO NOTHING`,
					item.id, seedFamilyID, *item.studentID,
					item.title, date, item.startTime, item.endTime,
					item.category, item.color,
				).Error; err != nil {
					return fmt.Errorf("insert schedule item %s: %w", item.id, err)
				}
			} else {
				if err := tx.Exec(`
					INSERT INTO plan_schedule_items
						(id, family_id, title, start_date,
						 start_time, end_time, category, color)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
					ON CONFLICT (id) DO NOTHING`,
					item.id, seedFamilyID,
					item.title, date, item.startTime, item.endTime,
					item.category, item.color,
				).Error; err != nil {
					return fmt.Errorf("insert schedule item %s: %w", item.id, err)
				}
			}
		}

		// Schedule template
		templateItems := `[
			{"day_of_week":1,"start_time":"09:00","end_time":"09:45","title":"Morning Read Aloud","category":"reading","color":"#10B981"},
			{"day_of_week":1,"start_time":"10:00","end_time":"10:35","title":"Math","category":"lesson","color":"#3B82F6"},
			{"day_of_week":1,"start_time":"11:00","end_time":"11:45","title":"Nature Study","category":"activity","color":"#F59E0B"},
			{"day_of_week":3,"start_time":"10:00","end_time":"13:00","title":"Co-op","category":"co_op","color":"#8B5CF6"},
			{"day_of_week":5,"start_time":"09:00","end_time":"12:00","title":"Free Learning Friday","category":"custom","color":"#EC4899"}
		]`
		if err := tx.Exec(`
			INSERT INTO plan_schedule_templates
				(id, family_id, name, description, items, is_active)
			VALUES (?, ?, 'Charlotte Mason Week',
				'Our standard Charlotte Mason weekly schedule with co-op on Wednesdays.',
				?::JSONB, true)
			ON CONFLICT (id) DO NOTHING`,
			schedTmpl1ID, seedFamilyID, templateItems,
		).Error; err != nil {
			return fmt.Errorf("insert schedule template: %w", err)
		}

		return nil
	})
}

// ─── Lifecycle seed ───────────────────────────────────────────────────────────

func seedLifecycle(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO lifecycle_export_requests
				(id, family_id, requested_by, status, format,
				 include_domains, archive_key, size_bytes,
				 download_expires_at, completed_at)
			VALUES (?, ?, ?, 'completed', 'json',
				ARRAY['iam','learn','social','billing'],
				'exports/seed-family/export-2026-03-01.json',
				524288,
				NOW() + INTERVAL '7 days',
				NOW() - INTERVAL '30 days')
			ON CONFLICT (id) DO NOTHING`,
			lifecycleExport1ID, seedFamilyID, seedParentID,
		).Error; err != nil {
			return fmt.Errorf("insert lifecycle export request: %w", err)
		}

		return nil
	})
}

// ─── Stress Family ID helpers ─────────────────────────────────────────────────
// Generate deterministic IDs for bulk stress-family rows. Idempotent across runs.

func stressListingID(n int) string   { return fmt.Sprintf("02000000-0000-7000-8000-0000000001%02d", n) }
func stressActLogID(n int) string    { return fmt.Sprintf("02000000-0000-7000-8000-0000000002%02d", n) }
func stressSchedItemID(n int) string { return fmt.Sprintf("02000000-0000-7000-8000-0000000003%02d", n) }

// ─── Stress Kratos identity ───────────────────────────────────────────────────

func ensureStressKratosIdentity(dbName string) string {
	for _, baseURL := range kratosURLOrder(dbName) {
		if id, ok := kratosLookupOrCreate(baseURL, "stress@example.com"); ok {
			return id
		}
	}
	slog.Warn("Kratos unreachable, using fallback identity UUID for stress parent")
	return fallbackStressKratosID
}

// ─── Stress family seed ───────────────────────────────────────────────────────
//
// Adversarial "stress family" for layout/overflow QA. All data is synthetic.
//
// Login:   stress@example.com / SeedPassword123!
// Purpose: surfaces layout bugs hidden by polite happy-path data.
//
// Covered failure modes:
//   - Long names:    family (~175 chars with diacritics), parent (~150 chars),
//                    student 1 (~195-char Latin+diacritics), student 2 (emoji+unicode),
//                    student 3 (Arabic RTL), student 4 (Greek script),
//                    listing 1 title ~200 chars.
//   - Many items:    8 students, 20 marketplace listings, 25 activity log entries,
//                    12 schedule items.
//   - Empty sets:    zero notifications, zero purchases, zero social posts,
//                    empty reading list (no items), students 5–8 have zero activity logs.
//   - Unicode/RTL:   Arabic in student 3 name and activity logs; Greek in student 4;
//                    emoji in student 2 name and journal entries.
//   - Missing opts:  empty bio on social profile; 5 draft listings (no published_at);
//                    journal entry 3 minimal content.
//   - Huge numbers:  listing 6 at price_cents=9_999_999 (~$100K);
//                    listing 5 at price_cents=0 (free);
//                    listing 7 at price_cents=1 (minimum);
//                    activity log 4 with duration_minutes=9999 (~7 days);
//                    activity log 10 with duration_minutes=1 (minimum).

func seedStressFamily(db *gorm.DB, stressKratosID string) error {
	if err := seedStressFamilyIAM(db, stressKratosID); err != nil {
		return err
	}
	if err := seedStressFamilyOnboard(db); err != nil {
		return err
	}
	if err := seedStressFamilySocial(db); err != nil {
		return err
	}
	if err := seedStressFamilyMarketplace(db); err != nil {
		return err
	}
	if err := seedStressFamilyLearn(db); err != nil {
		return err
	}
	return seedStressFamilyPlan(db)
}

func seedStressFamilyIAM(db *gorm.DB, stressKratosID string) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		const (
			longFamilyName = "Müller-García-Nakamura-Okonkwo-Johanssen-Papadopoulos International Cooperative Homeschooling Family of Greater Metropolitan Austin — Classical Trivium & Charlotte Mason"
			longParentName = "Administrateur Pédagogique Principal de la Grande Famille Müller-García-Nakamura-Okonkwo, Responsable de l'Éducation Maison Extraordinairement Dénommée"
		)

		if err := tx.Exec(`
			INSERT INTO iam_families
				(id, display_name, state_code, primary_methodology_slug,
				 subscription_tier, coppa_consent_status, coppa_consented_at, coppa_consent_method)
			VALUES (?, ?, 'TX', 'charlotte-mason', 'free',
				'consented', NOW(), 'credit_card_verification')
			ON CONFLICT (id) DO NOTHING`,
			stressFamilyID, longFamilyName,
		).Error; err != nil {
			return fmt.Errorf("insert stress family: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO iam_parents
				(id, family_id, kratos_identity_id, display_name, email, is_primary)
			VALUES (?, ?, ?, ?, 'stress@example.com', true)
			ON CONFLICT (id) DO NOTHING`,
			stressParentID, stressFamilyID, stressKratosID, longParentName,
		).Error; err != nil {
			return fmt.Errorf("insert stress parent: %w", err)
		}

		if err := tx.Exec(`
			UPDATE iam_families SET primary_parent_id = ?
			WHERE id = ? AND primary_parent_id IS NULL`,
			stressParentID, stressFamilyID,
		).Error; err != nil {
			return fmt.Errorf("update stress family primary parent: %w", err)
		}

		type stressStudent struct {
			id         string
			name       string
			birthYear  int
			gradeLevel string
		}
		students := []stressStudent{
			// #1: ~195-char Latin+diacritics name — tests max-width truncation
			{stressStudent1ID,
				"Reginald Bartholomew Constantine Fitzgerald-Montgomery von Schüttemeyer-Nakamura-García-Okonkwo the Third Extraordinarily Long Student Name For Automated Layout Stress Testing Of Text Truncation",
				2010, "8th"},
			// #2: emoji + mixed unicode
			{stressStudent2ID, "🦁 Émilie-Ångström Ñoño 📚🌿 Unicode Emoji Student", 2015, "4th"},
			// #3: Arabic RTL
			{stressStudent3ID, "أحمد محمد عبدالله النموذجي للاختبار", 2012, "6th"},
			// #4: Greek script
			{stressStudent4ID, "Αλεξάνδρα Νικολάου-Παπαδοπούλου", 2013, "7th"},
			// #5–#8: normal names, bulk count tests "many students" scroll behaviour
			{stressStudent5ID, "Alice Verylongsurname-Hyphenated-Compound", 2016, "3rd"},
			{stressStudent6ID, "Bob Short", 2018, "1st"},
			{stressStudent7ID, "Carol-Anne Middleton", 2017, "2nd"},
			{stressStudent8ID, "Dave", 2019, "K"},
		}
		for _, s := range students {
			if err := tx.Exec(`
				INSERT INTO iam_students
					(id, family_id, display_name, birth_year, grade_level)
				VALUES (?, ?, ?, ?, ?)
				ON CONFLICT (id) DO NOTHING`,
				s.id, stressFamilyID, s.name, s.birthYear, s.gradeLevel,
			).Error; err != nil {
				return fmt.Errorf("insert stress student %s: %w", s.id, err)
			}
		}

		if err := tx.Exec(`
			INSERT INTO iam_coppa_audit_log
				(id, family_id, action, method,
				 previous_status, new_status, performed_by)
			VALUES (?, ?, 'consent_granted', 'credit_card_verification',
				'noticed', 'consented', ?)
			ON CONFLICT (id) DO NOTHING`,
			stressCoppaAuditID, stressFamilyID, stressParentID,
		).Error; err != nil {
			return fmt.Errorf("insert stress coppa audit log: %w", err)
		}

		return nil
	})
}

func seedStressFamilyOnboard(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Wizard = skipped so the stress family bypasses the onboarding guard.
		return tx.Exec(`
			INSERT INTO onb_wizard_progress
				(id, family_id, status, current_step, completed_steps)
			VALUES (?, ?, 'skipped', 'family_profile', '{}')
			ON CONFLICT (family_id) DO NOTHING`,
			stressWizardProgressID, stressFamilyID,
		).Error
	})
}

func seedStressFamilySocial(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Empty bio string — tests the missing-optional-field social profile state.
		// No posts, conversations, or events: exercises the empty social feed UI state.
		return tx.Exec(`
			INSERT INTO soc_profiles (family_id, bio)
			VALUES (?, '')
			ON CONFLICT (family_id) DO NOTHING`,
			stressFamilyID,
		).Error
	})
}

func seedStressFamilyMarketplace(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// Publisher name is itself a long stress string.
		if err := tx.Exec(`
			INSERT INTO mkt_publishers
				(id, name, slug, description, is_platform, is_verified)
			VALUES (?,
				'Stress Test Press — Müller-García-Nakamura-Okonkwo & Associates Educational Publishing House of Greater Metropolitan Austin International',
				'stress-test-press',
				NULL,
				false, false)
			ON CONFLICT (id) DO NOTHING`,
			stressPublisherID,
		).Error; err != nil {
			return fmt.Errorf("insert stress publisher: %w", err)
		}

		// store_bio = NULL — tests missing-optional creator bio display.
		if err := tx.Exec(`
			INSERT INTO mkt_creators
				(id, parent_id, onboarding_status, store_name, store_bio, tos_accepted_at)
			VALUES (?, ?, 'active', 'Stress Test Press', NULL, NOW() - INTERVAL '10 days')
			ON CONFLICT (id) DO NOTHING`,
			stressCreatorID, stressParentID,
		).Error; err != nil {
			return fmt.Errorf("insert stress creator: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO mkt_publisher_members (publisher_id, creator_id, role)
			VALUES (?, ?, 'owner')
			ON CONFLICT (publisher_id, creator_id) DO NOTHING`,
			stressPublisherID, stressCreatorID,
		).Error; err != nil {
			return fmt.Errorf("insert stress publisher member: %w", err)
		}

		// Long description for listing 2: ~3 000 chars of repetitive text.
		longDesc := strings.Repeat("This is a very long description paragraph that tests the layout when descriptions are excessively verbose and extend far beyond what a normal creator would write. The narration continues for many sentences to push layout boundaries. ", 14)

		type stressListing struct {
			n           int
			title       string
			description string
			priceCents  int
			contentType string
			status      string
			published   bool
		}
		listings := []stressListing{
			// 1: ~200-char title — tests listing-card title overflow
			{1,
				"The Extraordinarily Comprehensive and Extensively Detailed Charlotte Mason Year-One Curriculum Planning Guide for Ambitious Homeschooling Families Pursuing Excellence in Classical Education",
				"A complete first-year guide.", 2999, "curriculum", "published", true},
			// 2: normal title, very long description
			{2, "Nature Journal Starter Pack — Stress Edition", longDesc, 999, "worksheet", "published", true},
			// 3: emoji + unicode in title
			{3, "🌿 Émoji & Ünïcödé Cürrïcülüm Güïdé 📚 — Ångström Edition", "Resources with special characters in title.", 1499, "curriculum", "published", true},
			// 4: Arabic title — RTL text in a LTR listing card
			{4, "المواد التعليمية للتجربة — Educational Materials For Testing", "Bilingual Arabic-English listing.", 1799, "worksheet", "published", true},
			// 5: price = 0 (free listing edge case)
			{5, "Free Starter Guide — Zero Price Edge Case", "This listing is free. Tests zero-price display.", 0, "worksheet", "published", true},
			// 6: huge price (~$100 K) — tests currency formatting overflow
			{6, "Premium Platinum Deluxe Curriculum Suite — Enterprise Edition", "The most expensive listing for stress testing huge price display.", 9999999, "curriculum", "published", true},
			// 7: minimum price = 1 cent
			{7, "Minimal Priced Resource (One Cent)", "One cent listing for minimum price edge case.", 1, "worksheet", "published", true},
			// 8–12: draft listings (no published_at) — tests missing-optional field in listing UI
			{8, "Draft Listing Alpha — Unpublished", "Draft listing 1.", 500, "worksheet", "draft", false},
			{9, "Draft Listing Beta — Pending Review", "Draft listing 2.", 500, "worksheet", "draft", false},
			{10, "Draft Listing Gamma", "Draft listing 3.", 500, "worksheet", "draft", false},
			{11, "Draft Listing Delta", "Draft listing 4.", 500, "worksheet", "draft", false},
			{12, "Draft Listing Epsilon", "Draft listing 5.", 500, "worksheet", "draft", false},
			// 13–20: standard listings to reach 20 total (forces pagination at >10 items)
			{13, "Stress Listing 13 — Standard", "Standard listing.", 999, "worksheet", "published", true},
			{14, "Stress Listing 14 — Standard", "Standard listing.", 999, "worksheet", "published", true},
			{15, "Stress Listing 15 — Standard", "Standard listing.", 999, "worksheet", "published", true},
			{16, "Stress Listing 16 — Standard", "Standard listing.", 999, "worksheet", "published", true},
			{17, "Stress Listing 17 — Standard", "Standard listing.", 999, "worksheet", "published", true},
			{18, "Stress Listing 18 — Standard", "Standard listing.", 999, "worksheet", "published", true},
			{19, "Stress Listing 19 — Standard", "Standard listing.", 999, "worksheet", "published", true},
			{20, "Stress Listing 20 — Standard", "Standard listing.", 999, "worksheet", "published", true},
		}

		for _, l := range listings {
			id := stressListingID(l.n)
			if l.published {
				if err := tx.Exec(`
					INSERT INTO mkt_listings
						(id, creator_id, publisher_id, title, description, price_cents,
						 methodology_tags, subject_tags, content_type, status, published_at)
					VALUES (?, ?, ?, ?, ?, ?,
						'{}', ARRAY['science'], ?, ?, NOW() - INTERVAL '5 days')
					ON CONFLICT (id) DO NOTHING`,
					id, stressCreatorID, stressPublisherID,
					l.title, l.description, l.priceCents, l.contentType, l.status,
				).Error; err != nil {
					return fmt.Errorf("insert stress listing %d: %w", l.n, err)
				}
			} else {
				// Draft: no published_at — missing optional field
				if err := tx.Exec(`
					INSERT INTO mkt_listings
						(id, creator_id, publisher_id, title, description, price_cents,
						 methodology_tags, subject_tags, content_type, status)
					VALUES (?, ?, ?, ?, ?, ?,
						'{}', ARRAY['science'], ?, ?)
					ON CONFLICT (id) DO NOTHING`,
					id, stressCreatorID, stressPublisherID,
					l.title, l.description, l.priceCents, l.contentType, l.status,
				).Error; err != nil {
					return fmt.Errorf("insert stress draft listing %d: %w", l.n, err)
				}
			}
		}

		return nil
	})
}

func seedStressFamilyLearn(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		now := time.Now()

		// 25 activity logs across students 1–4.
		// Students 5–8 have zero activity logs — tests empty activity-history UI state.
		type actLog struct {
			n          int
			studentID  string
			actDefID   string
			title      string
			daysAgo    int
			durationMn int
		}
		logs := []actLog{
			// Student 1 (long name): many logs, including huge and minimal durations
			{1, stressStudent1ID, activityDef1ID, "Nature Walk — Stress Student Long Name Day 1", 0, 60},
			{2, stressStudent1ID, activityDef2ID, "Math Games — Stress Student Day 1", 0, 30},
			{3, stressStudent1ID, activityDef3ID, "Read Aloud: Charlotte's Web — Stress Student Day 1", 1, 45},
			{4, stressStudent1ID, activityDef1ID, "Nature Walk — Day 2 (9999 min huge duration)", 1, 9999},
			{5, stressStudent1ID, activityDef2ID, "Math — Day 3", 2, 35},
			{6, stressStudent1ID, activityDef3ID, "Read Aloud — Day 3", 2, 45},
			{7, stressStudent1ID, activityDef1ID, "Nature — Day 5", 4, 50},
			{8, stressStudent1ID, activityDef2ID, "Math — Day 7", 6, 30},
			{9, stressStudent1ID, activityDef3ID, "Read Aloud — Day 10", 9, 45},
			{10, stressStudent1ID, activityDef1ID, "Nature — Day 14 (1 min minimal duration)", 13, 1},
			// Student 2 (emoji name): logs with emoji/unicode in title
			{11, stressStudent2ID, activityDef1ID, "🌿 Nature Walk — Émoji Student 📚", 0, 45},
			{12, stressStudent2ID, activityDef3ID, "Read Aloud — Ångström Study Session 🦁", 1, 40},
			{13, stressStudent2ID, activityDef2ID, "Math Games — Unicode Émoji ñoño", 2, 30},
			{14, stressStudent2ID, activityDef1ID, "Nature — Day 5", 4, 55},
			{15, stressStudent2ID, activityDef3ID, "Read Aloud — Day 7", 6, 45},
			// Student 3 (Arabic RTL): logs with Arabic text
			{16, stressStudent3ID, activityDef1ID, "جولة الطبيعة — دراسة التلميذ النموذجي في العلوم", 0, 60},
			{17, stressStudent3ID, activityDef2ID, "ألعاب الرياضيات — اليوم الأول", 1, 30},
			{18, stressStudent3ID, activityDef3ID, "القراءة بصوت عالٍ — الفصل الأول", 2, 45},
			// Student 4 (Greek): logs with Greek text
			{19, stressStudent4ID, activityDef1ID, "Βόλτα στη Φύση — Ελληνικός Μαθητής", 0, 55},
			{20, stressStudent4ID, activityDef2ID, "Παιχνίδια Μαθηματικών — Ημέρα 1", 1, 30},
			{21, stressStudent4ID, activityDef3ID, "Ανάγνωση Δυνατά — Κεφάλαιο Πρώτο", 2, 40},
			// Additional logs to reach 25 total
			{22, stressStudent1ID, activityDef2ID, "Math — Week 4", 21, 35},
			{23, stressStudent2ID, activityDef1ID, "Nature — Week 4", 22, 50},
			{24, stressStudent3ID, activityDef3ID, "Read Aloud — Week 5", 28, 45},
			{25, stressStudent4ID, activityDef2ID, "Math — Week 5", 29, 30},
		}

		for _, l := range logs {
			id := stressActLogID(l.n)
			date := now.AddDate(0, 0, -l.daysAgo).Format("2006-01-02")
			if err := tx.Exec(`
				INSERT INTO learn_activity_logs
					(id, family_id, student_id, title, subject_tags,
					 content_id, duration_minutes, activity_date)
				VALUES (?, ?, ?, ?, ARRAY['science'], ?, ?, ?)
				ON CONFLICT (id) DO NOTHING`,
				id, stressFamilyID, l.studentID, l.title,
				l.actDefID, l.durationMn, date,
			).Error; err != nil {
				return fmt.Errorf("insert stress activity log %d: %w", l.n, err)
			}
		}

		// Journal entries: 3 for student 1 only.
		// Students 2–8 have zero journal entries — tests empty journal UI state.
		longContent := strings.Repeat("This is a very long journal entry paragraph that tests how the UI handles excessively verbose free-text content from a student or parent. The narration continues for many sentences to push layout boundaries and verify text overflow handling. ", 20)
		emojiContent := "🦁 Today we saw lions and 🌿 plants! The nature study was 📚 amazing. أحمد and Αλεξάνδρα both attended. 日本語テスト. Ångström units. Ñoño noño. ¡Hola! Ça va? Привет мир."

		journals := []struct {
			id        string
			entryType string
			title     string
			content   string
		}{
			// Long content: tests scroll/overflow in journal entry view
			{stressJournal1ID, "narration", "Very Long Narration Entry — Layout Stress Test For Maximum Content Overflow", longContent},
			// Emoji + unicode + RTL mixed: tests bidi rendering
			{stressJournal2ID, "freeform", "🌿 Emoji & Unicode Journal Entry 📚🦁 — Mixed Scripts", emojiContent},
			// Minimal content: tests missing-optional-richness display (no subjects inferred)
			{stressJournal3ID, "freeform", "Brief", "Brief."},
		}
		for _, j := range journals {
			if err := tx.Exec(`
				INSERT INTO learn_journal_entries
					(id, family_id, student_id, entry_type, title, content, subject_tags, entry_date)
				VALUES (?, ?, ?, ?, ?, ?, ARRAY['science'], CURRENT_DATE)
				ON CONFLICT (id) DO NOTHING`,
				j.id, stressFamilyID, stressStudent1ID,
				j.entryType, j.title, j.content,
			).Error; err != nil {
				return fmt.Errorf("insert stress journal %s: %w", j.id, err)
			}
		}

		// Empty reading list: no items added — tests the empty reading-list UI state.
		return tx.Exec(`
			INSERT INTO learn_reading_lists
				(id, family_id, name, description, student_id)
			VALUES (?, ?, 'Stress Empty Reading List — Tests Empty State UI', NULL, ?)
			ON CONFLICT (id) DO NOTHING`,
			stressReadingListID, stressFamilyID, stressStudent1ID,
		).Error
	})
}

func seedStressFamilyPlan(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		now := time.Now()

		type schedItem struct {
			n         int
			studentID *string
			title     string
			daysOut   int
			startTime string
			endTime   string
			category  string
			color     string
		}
		s1 := stressStudent1ID
		s2 := stressStudent2ID

		items := []schedItem{
			// Unicode/emoji title
			{1, &s1, "🌿 Nature Walk — Émoji Schedule Item 📚 Ångström Edition", 0, "09:00", "10:00", "activity", "#10B981"},
			// Long title — tests schedule-card title overflow
			{2, &s1, "Read Aloud: Charlotte's Web & Mathematics Cross-Subject Integrated Learning Block — Extraordinarily Long Schedule Item Title For Layout Stress", 0, "10:00", "11:30", "reading", "#3B82F6"},
			// 8-hour item (480 min) — tests huge-duration display
			{3, &s2, "Full-Day Arabic Language Immersion Session — أحمد محمد — 480 Minutes", 1, "08:00", "16:00", "lesson", "#8B5CF6"},
			// 5-minute item — tests minimum-duration display
			{4, &s1, "Quick Vocabulary Check", 1, "09:00", "09:05", "lesson", "#F59E0B"},
			// Two overlapping items for same student same day — tests schedule conflict display
			{5, &s1, "Overlapping Block A (10:00–11:00)", 2, "10:00", "11:00", "activity", "#EF4444"},
			{6, &s1, "Overlapping Block B (10:30–11:30)", 2, "10:30", "11:30", "lesson", "#EF4444"},
			// Family-wide items (no student)
			{7, nil, "Family Co-op — All Students Weekly Meeting", 2, "13:00", "16:00", "co_op", "#6366F1"},
			{8, nil, "Stress Co-op: Ångström Session & الاجتماع العائلي Ελληνικός", 3, "09:00", "12:00", "co_op", "#14B8A6"},
			// Additional items to force vertical scroll (12 total)
			{9, &s1, "Stress Item 9 — Standard", 4, "09:00", "09:30", "activity", "#10B981"},
			{10, &s2, "Stress Item 10 — Standard", 4, "10:00", "10:30", "lesson", "#3B82F6"},
			{11, &s1, "Stress Item 11 — Standard", 5, "09:00", "09:45", "reading", "#F59E0B"},
			{12, &s2, "Stress Item 12 — Standard", 5, "11:00", "12:00", "activity", "#8B5CF6"},
		}

		for _, item := range items {
			id := stressSchedItemID(item.n)
			date := now.AddDate(0, 0, item.daysOut).Format("2006-01-02")
			if item.studentID != nil {
				if err := tx.Exec(`
					INSERT INTO plan_schedule_items
						(id, family_id, student_id, title, start_date,
						 start_time, end_time, category, color)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
					ON CONFLICT (id) DO NOTHING`,
					id, stressFamilyID, *item.studentID,
					item.title, date, item.startTime, item.endTime,
					item.category, item.color,
				).Error; err != nil {
					return fmt.Errorf("insert stress schedule item %d: %w", item.n, err)
				}
			} else {
				if err := tx.Exec(`
					INSERT INTO plan_schedule_items
						(id, family_id, title, start_date,
						 start_time, end_time, category, color)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
					ON CONFLICT (id) DO NOTHING`,
					id, stressFamilyID,
					item.title, date, item.startTime, item.endTime,
					item.category, item.color,
				).Error; err != nil {
					return fmt.Errorf("insert stress schedule item %d: %w", item.n, err)
				}
			}
		}

		return nil
	})
}
