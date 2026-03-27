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
	seedFamilyID   = "01900000-0000-7000-8000-000000000001"
	friendFamilyID = "01900000-0000-7000-8000-000000000002"
	seedParentID   = "01900000-0000-7000-8000-000000000011"
	friendParentID = "01900000-0000-7000-8000-000000000012"
	emmaStudentID  = "01900000-0000-7000-8000-000000000021"
	jamesStudentID = "01900000-0000-7000-8000-000000000022"
	coppaAuditID   = "01900000-0000-7000-8000-000000000031"

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

	// Marketplace
	seedPublisherID  = "01900000-0000-7000-8000-000000000201"
	seedCreatorID    = "01900000-0000-7000-8000-000000000202"
	listing1ID       = "01900000-0000-7000-8000-000000000211"
	listing2ID       = "01900000-0000-7000-8000-000000000212"
	listing3ID       = "01900000-0000-7000-8000-000000000213"
	listing4ID       = "01900000-0000-7000-8000-000000000214"
	listing5ID       = "01900000-0000-7000-8000-000000000215"
	purchase1ID      = "01900000-0000-7000-8000-000000000221"
	purchase2ID      = "01900000-0000-7000-8000-000000000222"
	review1ID        = "01900000-0000-7000-8000-000000000231"
	cartItem1ID      = "01900000-0000-7000-8000-000000000241"

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

	// Notifications
	notif1ID = "01900000-0000-7000-8000-000000000401"
	notif2ID = "01900000-0000-7000-8000-000000000402"
	notif3ID = "01900000-0000-7000-8000-000000000403"
	notif4ID = "01900000-0000-7000-8000-000000000404"
	notif5ID = "01900000-0000-7000-8000-000000000405"
	pref1ID  = "01900000-0000-7000-8000-000000000411"
	pref2ID  = "01900000-0000-7000-8000-000000000412"

	// Billing
	subID   = "01900000-0000-7000-8000-000000000501"
	txnID   = "01900000-0000-7000-8000-000000000502"

	// Media
	media1ID = "01900000-0000-7000-8000-000000000601"
	media2ID = "01900000-0000-7000-8000-000000000602"

	// Recommendations
	rec1ID   = "01900000-0000-7000-8000-000000000701"
	rec2ID   = "01900000-0000-7000-8000-000000000702"
	rec3ID   = "01900000-0000-7000-8000-000000000703"
	recPrefID = "01900000-0000-7000-8000-000000000711"

	// Planning
	schedItem1ID = "01900000-0000-7000-8000-000000000801"
	schedItem2ID = "01900000-0000-7000-8000-000000000802"
	schedItem3ID = "01900000-0000-7000-8000-000000000803"
	schedItem4ID = "01900000-0000-7000-8000-000000000804"
	schedItem5ID = "01900000-0000-7000-8000-000000000805"
	schedTmpl1ID = "01900000-0000-7000-8000-000000000811"

	// Fallback Kratos identity UUIDs when Kratos is unreachable
	fallbackKratosID       = "01900000-0000-7000-8000-000000000999" // seed parent
	fallbackFriendKratosID = "01900000-0000-7000-8000-000000000998" // friend parent

	// Kratos admin API base URL
	kratosAdminURL = "http://localhost:4434"
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

	// ── Kratos identity ───────────────────────────────────────────────────
	kratosID := ensureKratosIdentity()

	// ── Seed all domains ──────────────────────────────────────────────────
	if err := seedAll(db, kratosID); err != nil {
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
	// Keep adminURL pointing to the existing homegrown db for CREATE DATABASE
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
	SchemaID string         `json:"schema_id"`
	Traits   map[string]any `json:"traits"`
	Credentials map[string]any `json:"credentials"`
}

// ensureKratosIdentity creates or retrieves the seed identity from Kratos.
// Returns a deterministic fallback UUID if Kratos is unreachable.
func ensureKratosIdentity() string {
	const email = "seed@example.com"

	// Try to find existing identity
	resp, err := http.Get(kratosAdminURL + "/admin/identities?credentials_identifier=" + url.QueryEscape(email))
	if err != nil {
		slog.Warn("Kratos unreachable, using fallback identity UUID", "err", err)
		return fallbackKratosID
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
			slog.Info("found existing Kratos identity", "id", id)
			return id
		}
	}

	// Create new identity
	payload := kratosIdentityBody{
		SchemaID: "default",
		Traits: map[string]any{
			"email": email,
		},
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
		kratosAdminURL+"/admin/identities",
		"application/json",
		strings.NewReader(string(payloadBytes)),
	)
	if createErr != nil {
		slog.Warn("failed to create Kratos identity, using fallback", "err", createErr)
		return fallbackKratosID
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
			slog.Info("created Kratos identity", "id", id)
			return id
		}
	}

	slog.Warn("could not parse Kratos response, using fallback identity UUID")
	return fallbackKratosID
}

// ─── Seed orchestrator ────────────────────────────────────────────────────────

func seedAll(db *gorm.DB, kratosID string) error {
	// Look up platform publisher ID (seeded by migration)
	var platformPublisherID string
	if err := db.Raw("SELECT id FROM mkt_publishers WHERE slug = 'homegrown-academy'").
		Scan(&platformPublisherID).Error; err != nil || platformPublisherID == "" {
		slog.Warn("platform publisher not found — using seed publisher only")
		platformPublisherID = seedPublisherID
	}

	steps := []struct {
		name string
		fn   func(*gorm.DB, string, string) error
	}{
		{"IAM", func(db *gorm.DB, kratosID, pubID string) error { return seedIAM(db, kratosID) }},
		{"Social", func(db *gorm.DB, _ string, pubID string) error { return seedSocial(db) }},
		{"Marketplace", func(db *gorm.DB, _ string, _ string) error { return seedMarketplace(db) }},
		{"Learn", func(db *gorm.DB, _ string, pubID string) error { return seedLearn(db, pubID) }},
		{"Notifications", func(db *gorm.DB, _ string, pubID string) error { return seedNotifications(db) }},
		{"Billing", func(db *gorm.DB, _ string, pubID string) error { return seedBilling(db) }},
		{"Media", func(db *gorm.DB, _ string, pubID string) error { return seedMedia(db) }},
		{"Safety", func(db *gorm.DB, _ string, pubID string) error { return seedSafety(db) }},
		{"Recommendations", func(db *gorm.DB, _ string, pubID string) error { return seedRecs(db) }},
		{"Planning", func(db *gorm.DB, _ string, pubID string) error { return seedPlan(db) }},
	}

	for _, step := range steps {
		slog.Info("seeding domain", "domain", step.name)
		if err := step.fn(db, kratosID, platformPublisherID); err != nil {
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
			id          string
			title       string
			description string
			priceCents  int
			contentType string
		}{
			{listing1ID, "Charlotte Mason Year 1 Curriculum Guide",
				"A complete first-year guide with book lists, nature study schedules, and narration prompts.",
				2999, "curriculum"},
			{listing2ID, "Nature Journal Starter Pack",
				"Illustrated worksheets for nature journaling: trees, insects, birds, and seasons.",
				999, "worksheet"},
			{listing3ID, "Living Books Read-Aloud Video Series",
				"Twelve read-aloud video lessons featuring classic literature with narration guides.",
				1999, "video"},
			{listing4ID, "Charlotte Mason Book List: K-5",
				"Curated living books list organized by subject and grade level, with library links.",
				499, "book_list"},
			{listing5ID, "Narration Assessment Rubric",
				"Age-appropriate narration assessment tools for oral and written narration.",
				299, "assessment"},
		}

		for _, l := range listings {
			if err := tx.Exec(`
				INSERT INTO mkt_listings
					(id, creator_id, publisher_id, title, description, price_cents,
					 methodology_tags, subject_tags, content_type, status, published_at)
				VALUES (?, ?, ?, ?, ?, ?,
					'{}', ARRAY['reading','language_arts'], ?, 'published', NOW() - INTERVAL '30 days')
				ON CONFLICT (id) DO NOTHING`,
				l.id, seedCreatorID, seedPublisherID,
				l.title, l.description, l.priceCents, l.contentType,
			).Error; err != nil {
				return fmt.Errorf("insert listing %s: %w", l.id, err)
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
				 ARRAY['nature_study','science'], 60),
				(?, ?, 'Math Games',
				 'Hands-on math games using manipulatives and real-world objects.',
				 ARRAY['mathematics'], 30),
				(?, ?, 'Read Aloud',
				 'Parent-led read-aloud session with narration.',
				 ARRAY['reading','language_arts'], 45)
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
				(?, ?, 'Charlotte''s Web',    'E.B. White',    ARRAY['reading','language_arts'], 'A classic tale of friendship between a pig and a spider.', 192),
				(?, ?, 'A Bear Called Paddington', 'Michael Bond', ARRAY['reading','language_arts'], 'The adventures of a bear from Peru.', 144),
				(?, ?, 'The Lion, the Witch and the Wardrobe', 'C.S. Lewis', ARRAY['reading','language_arts'], 'Four siblings discover a magical world through a wardrobe.', 208)
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
			{actLog1ID, emmaStudentID, activityDef1ID, "Nature Walk — Creek Trail", 1, 65, "ARRAY['nature_study','science']"},
			{actLog2ID, emmaStudentID, activityDef3ID, "Read Aloud: Charlotte's Web Ch 3", 1, 45, "ARRAY['reading']"},
			{actLog3ID, jamesStudentID, activityDef2ID, "Math Games: Counting Bears", 2, 30, "ARRAY['mathematics']"},
			{actLog4ID, emmaStudentID, activityDef2ID, "Math: Fraction Circles", 3, 35, "ARRAY['mathematics']"},
			{actLog5ID, jamesStudentID, activityDef3ID, "Read Aloud: Paddington Ch 1", 3, 40, "ARRAY['reading']"},
			{actLog6ID, emmaStudentID, activityDef1ID, "Nature Walk — Backyard Birds", 5, 50, "ARRAY['nature_study']"},
			{actLog7ID, jamesStudentID, activityDef1ID, "Nature Walk — Bug Hunt", 7, 45, "ARRAY['nature_study','science']"},
			{actLog8ID, emmaStudentID, activityDef3ID, "Read Aloud: Narnia Ch 1", 8, 50, "ARRAY['reading']"},
			{actLog9ID, jamesStudentID, activityDef2ID, "Math Games: Skip Counting", 10, 25, "ARRAY['mathematics']"},
			{actLog10ID, emmaStudentID, activityDef3ID, "Read Aloud: Charlotte's Web Ch 7", 14, 45, "ARRAY['reading']"},
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
				 ARRAY['reading','language_arts'], CURRENT_DATE - 1),
				(?, ?, ?, 'freeform', 'Birds I Saw Today',
				 'I saw a blue jay and two mourning doves in the backyard. The blue jay chased away the smaller birds. I drew them in my nature journal.',
				 ARRAY['nature_study'], CURRENT_DATE - 3)
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

// ─── Safety seed ──────────────────────────────────────────────────────────────

func seedSafety(db *gorm.DB) error {
	return bypassRLS(db, func(tx *gorm.DB) error {
		// account_status PK = family_id
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
