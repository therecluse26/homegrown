// cmd/seed-full/main.go — Full-scale seeder simulating a real platform at ~1000 families.
//
// Designed to run AFTER the basic seeder (make seed). Adds 997 additional families
// plus rich social, learning, marketplace, compliance, and planning data so the
// application can be navigated as though it has a real user base.
//
// Usage:
//
//	make seed-full                              # basic seed + full scale data
//	go run ./cmd/seed-full/ --db homegrown_agent   # full seeder only
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
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

// ─── Scale configuration ──────────────────────────────────────────────────────

const (
	numNewFamilies = 997 // + 3 from basic seeder = 1000 total
	batchSize      = 500
	rngSeed        = 42

	// Published content
	numActivityDefs     = 80
	numReadingItems     = 150
	numAssessmentDefs   = 30
	numQuizDefs         = 15
	numQuestionsPerQuiz = 5
	numVideoDefs        = 25
	numProjectDefs      = 12
	numSequenceDefs     = 8

	// Social
	numGroups = 35
	numEvents = 120

	// Marketplace
	numPublishers = 8
	numCreators   = 35
	numListings   = 200
)

// ─── UUID domain codes ────────────────────────────────────────────────────────
// Full seeder UUIDs use prefix 01910000-{domain}-7000-8000-{seq} to avoid
// collision with basic seeder's 01900000 range.

const (
	dFamily    = 0x0001
	dParent    = 0x0002
	dStudent   = 0x0003
	dFriend    = 0x0005
	dGroup     = 0x0006
	dGroupMem  = 0x0007
	dPost      = 0x0008
	dComment   = 0x0009
	dLike      = 0x000A
	dConvo     = 0x000B
	dMessage   = 0x000C
	dEvent     = 0x000D
	dRSVP      = 0x000E
	dActDef    = 0x0010
	dReadItem  = 0x0011
	dAssDef    = 0x0012
	dQuizDef   = 0x0013
	dQuestion  = 0x0014
	dVideoDef  = 0x0015
	dProjDef   = 0x0016
	dSeqDef    = 0x0017
	dSeqItem   = 0x0018
	dActLog    = 0x0020
	dJournal   = 0x0021
	dReadProg  = 0x0022
	dAssResult = 0x0023
	dQuizSess  = 0x0024
	dAssign    = 0x0025
	dReadList  = 0x0026
	dPublisher = 0x0030
	dCreator   = 0x0031
	dListing   = 0x0032
	dPurchase  = 0x0033
	dReview    = 0x0034
	dCartItem  = 0x0035
	dListFile  = 0x0036
	dNotif     = 0x0040
	dNotifPref = 0x0041
	dSub       = 0x0050
	dTxn       = 0x0051
	dHSCust    = 0x0052
	dAttend    = 0x0060
	dComplyAss = 0x0061
	dCourse    = 0x0062
	dTranscr   = 0x0063
	dPortfolio = 0x0064
	dPortItem  = 0x0065
	dSchedItem = 0x0070
	dSchedTmpl = 0x0071
	dRecsSignl = 0x0080
	dRec       = 0x0081
	dRecsPref  = 0x0083
	dWizard    = 0x0090
	dCoppa     = 0x0091
	dCurated    = 0x00A0
	dPubMember  = 0x00A1
	dVideoProg  = 0x00B0
	dProjProg   = 0x00B1
	dSeqProg    = 0x00B2
	dGradScale  = 0x00B3
	dStdTest    = 0x00B4
	dMediaUp    = 0x00B5
	dStarterRec = 0x00B6
)

// Existing seed UUIDs from the basic seeder (for cross-referencing).
const (
	existSeedFamilyID   = "01900000-0000-7000-8000-000000000001"
	existFriendFamilyID = "01900000-0000-7000-8000-000000000002"
	existPlatformFamID  = "01900000-0000-7000-8000-000000000003"
	existSeedParentID   = "01900000-0000-7000-8000-000000000011"
	existFriendParentID = "01900000-0000-7000-8000-000000000012"
	existAdminParentID  = "01900000-0000-7000-8000-000000000013"
	existEmmaID         = "01900000-0000-7000-8000-000000000021"
	existJamesID        = "01900000-0000-7000-8000-000000000022"
	existSeedPublisher  = "01900000-0000-7000-8000-000000000201"
)

// Deterministic methodology UUIDs.
// learn_* tables use UUID methodology_id columns (no FK to method_definitions yet).
// We create stable UUIDs for each slug so content can be filtered by methodology.
var methodologyUUIDs = map[string]string{
	"charlotte-mason": "01910000-aa01-7000-8000-000000000001",
	"classical":       "01910000-aa01-7000-8000-000000000002",
	"traditional":     "01910000-aa01-7000-8000-000000000003",
	"montessori":      "01910000-aa01-7000-8000-000000000004",
	"waldorf":         "01910000-aa01-7000-8000-000000000005",
	"unschooling":     "01910000-aa01-7000-8000-000000000006",
}

// methodologySlugs lists all slugs in a stable order for round-robin assignment.
var methodologySlugs = []string{
	"charlotte-mason", "classical", "traditional", "montessori", "waldorf", "unschooling",
}

// ─── Core types ───────────────────────────────────────────────────────────────

type student struct {
	ID         string
	Name       string
	BirthYear  int
	GradeLevel string
}

type family struct {
	ID                 string
	DisplayName        string
	StateCode          string
	PrimaryMethodology string
	SubscriptionTier   string
	PrimaryParentID    string
	PrimaryParentName  string
	PrimaryParentEmail string
	CoParentID         string // empty if none
	CoParentName       string
	Students           []student
	ActivityLevel      int // 1–5, determines data volume
	CreatedAt          time.Time
}

type seeder struct {
	db  *gorm.DB
	rng *rand.Rand
	now time.Time

	// Families: index 0–2 = existing seed families, 3+ = new families.
	families []family

	// Published content IDs populated during content seeding.
	activityDefIDs   []string
	readingItemIDs   []string
	assessmentDefIDs []string
	quizDefIDs       []string
	videoDefIDs      []string
	projectDefIDs    []string
	sequenceDefIDs   []string
	publisherIDs     []string
	creatorIDs       []string
	creatorParentIDs []string // parent IDs that are creators (for purchases to exclude)
	listingIDs       []string
	listingCreators  []string // parallel to listingIDs: creator ID for each listing
	listingPrices    []int    // parallel to listingIDs: price in cents
	groupIDs         []string
	eventIDs         []string
	eventDates       []time.Time // parallel to eventIDs

	// Purchase tracking: purchaseID → {listingID, familyID}.
	purchaseListingMap map[string]string // purchaseID → listingID
	purchaseFamilyMap  map[string]string // purchaseID → familyID

	// Friendship graph: familyIndex → set of friend familyIndexes.
	friendships map[int]map[int]bool

	// Posts for comment/like generation.
	postIDs       []string
	postFamilyIdx []int // family index for each post

	// Conversations
	convoIDs []string
}

// ─── UUID generation ──────────────────────────────────────────────────────────

func uid(domain, seq int) string {
	return fmt.Sprintf("01910000-%04x-7000-8000-%012x", domain, seq)
}

// ─── Batch insert helper ──────────────────────────────────────────────────────

// execBatch executes batch INSERT statements.
//
//	prefix:  "INSERT INTO table (col1, col2) VALUES "
//	tmpl:    "(?, ?, ?::JSONB)"
//	suffix:  "ON CONFLICT (id) DO NOTHING"
//	rows:    each element is one row's arguments
func execBatch(tx *gorm.DB, prefix, tmpl, suffix string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[i:end]

		vals := make([]string, len(batch))
		args := make([]any, 0, len(batch)*len(batch[0]))
		for j, row := range batch {
			vals[j] = tmpl
			args = append(args, row...)
		}

		query := prefix + strings.Join(vals, ", ") + " " + suffix
		if err := tx.Exec(query, args...).Error; err != nil {
			return fmt.Errorf("batch insert at offset %d: %w", i, err)
		}
	}
	return nil
}

// ─── RLS bypass (same pattern as basic seeder) ────────────────────────────────

func bypassRLS(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL row_security = off").Error; err != nil {
			return err
		}
		return fn(tx)
	})
}

// ─── Database connection (mirrors basic seeder) ──────────────────────────────

func deriveURL(dbName string) (string, error) {
	envMap, _ := godotenv.Read(".env")
	base := envMap["DATABASE_URL"]
	if base == "" {
		base = "postgres://homegrown:homegrown@localhost:5932/homegrown"
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	u.Path = "/" + dbName
	return u.String(), nil
}

func openDB(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	return goose.Up(db, "migrations")
}

// ─── Family generation ────────────────────────────────────────────────────────

func (s *seeder) generateFamilies() {
	schoolYearStart := time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC)

	// Slot 0–2: existing seed families (we reference but don't re-create their IAM rows).
	s.families = make([]family, 0, 3+numNewFamilies)
	s.families = append(s.families, family{
		ID: existSeedFamilyID, DisplayName: "The Seed Family",
		StateCode: "TX", PrimaryMethodology: "charlotte-mason",
		SubscriptionTier: "premium", PrimaryParentID: existSeedParentID,
		PrimaryParentName: "Seed Parent", PrimaryParentEmail: "seed@example.com",
		ActivityLevel: 5, CreatedAt: schoolYearStart.AddDate(0, -6, 0),
		Students: []student{
			{ID: existEmmaID, Name: "Emma", BirthYear: 2014, GradeLevel: "5th"},
			{ID: existJamesID, Name: "James", BirthYear: 2017, GradeLevel: "2nd"},
		},
	})
	s.families = append(s.families, family{
		ID: existFriendFamilyID, DisplayName: "The Friend Family",
		StateCode: "TX", PrimaryMethodology: "classical",
		SubscriptionTier: "free", PrimaryParentID: existFriendParentID,
		PrimaryParentName: "Friend Parent", PrimaryParentEmail: "friend@example.com",
		ActivityLevel: 3, CreatedAt: schoolYearStart.AddDate(0, -4, 0),
	})
	s.families = append(s.families, family{
		ID: existPlatformFamID, DisplayName: "Platform Team",
		StateCode: "TX", PrimaryMethodology: "classical",
		SubscriptionTier: "premium", PrimaryParentID: existAdminParentID,
		PrimaryParentName: "Platform Admin", PrimaryParentEmail: "admin@example.com",
		ActivityLevel: 1, CreatedAt: schoolYearStart.AddDate(-1, 0, 0),
	})

	// Generate 997 new families.
	usedEmails := map[string]bool{
		"seed@example.com": true, "friend@example.com": true, "admin@example.com": true,
	}

	for i := 0; i < numNewFamilies; i++ {
		lastName := lastNames[s.rng.Intn(len(lastNames))]
		stateCode := weightedPick(s.rng, stateWeights)
		methodology := weightedPick(s.rng, methodologyWeights)

		tier := "free"
		if s.rng.Float64() < 0.30 {
			tier = "premium"
		}

		// Activity level: weighted toward middle (bell curve-ish).
		actLevel := s.rng.Intn(5) + 1
		if actLevel > 3 && s.rng.Float64() < 0.4 {
			actLevel = 3
		}

		// Stagger creation dates over 18 months.
		daysBack := s.rng.Intn(540) // up to ~18 months
		createdAt := s.now.AddDate(0, 0, -daysBack)

		famID := uid(dFamily, i+1)
		parentID := uid(dParent, i+1)

		firstName := femaleNames[s.rng.Intn(len(femaleNames))]
		if s.rng.Float64() < 0.5 {
			firstName = maleNames[s.rng.Intn(len(maleNames))]
		}
		parentName := firstName + " " + lastName
		email := strings.ToLower(firstName + "." + lastName + fmt.Sprintf("%d", i+1) + "@example.com")
		for usedEmails[email] {
			email = strings.ToLower(fmt.Sprintf("%s.%s%d@example.com", firstName, lastName, s.rng.Intn(9000)+1000))
		}
		usedEmails[email] = true

		fam := family{
			ID: famID, DisplayName: "The " + lastName + " Family",
			StateCode: stateCode, PrimaryMethodology: methodology,
			SubscriptionTier: tier, PrimaryParentID: parentID,
			PrimaryParentName: parentName, PrimaryParentEmail: email,
			ActivityLevel: actLevel, CreatedAt: createdAt,
		}

		// ~25% get a co-parent.
		if s.rng.Float64() < 0.25 {
			cpFirst := maleNames[s.rng.Intn(len(maleNames))]
			if s.rng.Float64() < 0.5 {
				cpFirst = femaleNames[s.rng.Intn(len(femaleNames))]
			}
			fam.CoParentID = uid(dParent, numNewFamilies+i+1)
			fam.CoParentName = cpFirst + " " + lastName
		}

		// 1–4 students per family.
		numStudents := pickWeighted(s.rng, []int{1, 2, 3, 4}, []float64{0.15, 0.40, 0.30, 0.15})
		for j := 0; j < numStudents; j++ {
			birthYear := 2008 + s.rng.Intn(13) // 2008–2020
			age := 2026 - birthYear
			grade := ageToGrade(age)
			sName := femaleNames[s.rng.Intn(len(femaleNames))]
			if s.rng.Float64() < 0.5 {
				sName = maleNames[s.rng.Intn(len(maleNames))]
			}
			fam.Students = append(fam.Students, student{
				ID:         uid(dStudent, i*4+j+1),
				Name:       sName,
				BirthYear:  birthYear,
				GradeLevel: grade,
			})
		}

		s.families = append(s.families, fam)
	}
}

func gradeToNum(g string) int {
	m := map[string]int{
		"Pre-K": 0, "K": 0, "1st": 1, "2nd": 2, "3rd": 3,
		"4th": 4, "5th": 5, "6th": 6, "7th": 7, "8th": 8,
		"9th": 9, "10th": 10, "11th": 11, "12th": 12,
	}
	if n, ok := m[g]; ok {
		return n
	}
	return 0
}

func ageToGrade(age int) string {
	grades := map[int]string{
		4: "Pre-K", 5: "K", 6: "1st", 7: "2nd", 8: "3rd",
		9: "4th", 10: "5th", 11: "6th", 12: "7th", 13: "8th",
		14: "9th", 15: "10th", 16: "11th", 17: "12th", 18: "12th",
	}
	if g, ok := grades[age]; ok {
		return g
	}
	if age < 4 {
		return "Pre-K"
	}
	return "12th"
}

// ─── Weighted random helpers ──────────────────────────────────────────────────

type weightedOption struct {
	Value  string
	Weight float64
}

func weightedPick(rng *rand.Rand, opts []weightedOption) string {
	total := 0.0
	for _, o := range opts {
		total += o.Weight
	}
	r := rng.Float64() * total
	for _, o := range opts {
		r -= o.Weight
		if r <= 0 {
			return o.Value
		}
	}
	return opts[len(opts)-1].Value
}

func pickWeighted(rng *rand.Rand, vals []int, weights []float64) int {
	total := 0.0
	for _, w := range weights {
		total += w
	}
	r := rng.Float64() * total
	for i, w := range weights {
		r -= w
		if r <= 0 {
			return vals[i]
		}
	}
	return vals[len(vals)-1]
}

// pickN picks n unique random indices from [0, max).
func pickN(rng *rand.Rand, max, n int) []int {
	if n >= max {
		result := make([]int, max)
		for i := range result {
			result[i] = i
		}
		return result
	}
	seen := make(map[int]bool, n)
	result := make([]int, 0, n)
	for len(result) < n {
		v := rng.Intn(max)
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// randomDate returns a random date between start and end.
func randomDate(rng *rand.Rand, start, end time.Time) time.Time {
	diff := end.Sub(start)
	if diff <= 0 {
		return start
	}
	return start.Add(time.Duration(rng.Int63n(int64(diff))))
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	var dbName string
	flag.StringVar(&dbName, "db", "homegrown_agent", "target database name")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("full-scale seeder starting", "target_db", dbName, "families", 3+numNewFamilies)

	dsn, err := deriveURL(dbName)
	if err != nil {
		slog.Error("derive URL", "err", err)
		os.Exit(1)
	}

	db, err := openDB(dsn)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("get sql.DB", "err", err)
		os.Exit(1)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := runMigrations(sqlDB); err != nil {
		slog.Error("migrations", "err", err)
		os.Exit(1)
	}

	s := &seeder{
		db:          db,
		rng:         rand.New(rand.NewSource(rngSeed)),
		now:         time.Now().UTC().Truncate(24 * time.Hour),
		friendships: make(map[int]map[int]bool),
	}

	s.generateFamilies()
	slog.Info("generated family data",
		"families", len(s.families),
		"new", numNewFamilies,
	)

	type step struct {
		name string
		fn   func() error
	}

	steps := []step{
		{"IAM", s.seedIAM},
		{"GradingScales", s.seedGradingScales},
		{"MediaUploads", s.seedMediaUploads},
		{"Onboarding", s.seedOnboarding},
		{"NotificationPreferences", s.seedNotificationPreferences},
		{"StarterRecommendations", s.seedStarterRecommendations},
		{"SocialProfiles", s.seedSocialProfiles},
		{"Friendships", s.seedFriendships},
		{"Groups", s.seedGroups},
		{"Posts", s.seedPosts},
		{"Comments", s.seedComments},
		{"PostLikes", s.seedPostLikes},
		{"Conversations", s.seedConversations},
		{"Events", s.seedEvents},
		{"Publishers", s.seedPublishers},
		{"PublishedContent", s.seedPublishedContent},
		{"Listings", s.seedListings},
		{"Purchases", s.seedPurchases},
		{"Reviews", s.seedReviews},
		{"ActivityLogs", s.seedActivityLogs},
		{"JournalEntries", s.seedJournalEntries},
		{"ReadingProgress", s.seedReadingProgress},
		{"AssessmentResults", s.seedAssessmentResults},
		{"Compliance", s.seedCompliance},
		{"Transcripts", s.seedTranscripts},
		{"Portfolios", s.seedPortfolios},
		{"StandardizedTests", s.seedStandardizedTests},
		{"ScheduleItems", s.seedScheduleItems},
		{"ScheduleTemplates", s.seedScheduleTemplates},
		{"LearningProgress", s.seedLearningProgress},
		{"Notifications", s.seedNotifications},
		{"Billing", s.seedBilling},
		{"Recommendations", s.seedRecommendations},
	}

	for _, st := range steps {
		slog.Info("seeding", "domain", st.name)
		if err := st.fn(); err != nil {
			slog.Error("seed failed", "domain", st.name, "err", err)
			os.Exit(1)
		}
	}

	slog.Info("full-scale seeding complete", "target_db", dbName)
}
