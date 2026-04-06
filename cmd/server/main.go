// @title Homegrown Academy API
// @version 0.1.0
// @description API for the Homegrown Academy homeschooling platform
// @host localhost:3500
// @BasePath /v1
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pressly/goose/v3"

	"github.com/homegrown-academy/homegrown-academy/internal/admin"
	"github.com/homegrown-academy/homegrown-academy/internal/app"
	"github.com/homegrown-academy/homegrown-academy/internal/billing"
	billingadapters "github.com/homegrown-academy/homegrown-academy/internal/billing/adapters"
	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"github.com/homegrown-academy/homegrown-academy/internal/discover"
	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	iamadapters "github.com/homegrown-academy/homegrown-academy/internal/iam/adapters"
	"github.com/homegrown-academy/homegrown-academy/internal/learn"
	"github.com/homegrown-academy/homegrown-academy/internal/media"
	mediaadapters "github.com/homegrown-academy/homegrown-academy/internal/media/adapters"
	"github.com/homegrown-academy/homegrown-academy/internal/method"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
	"github.com/homegrown-academy/homegrown-academy/internal/notify"
	notifyadapters "github.com/homegrown-academy/homegrown-academy/internal/notify/adapters"
	mktadapters "github.com/homegrown-academy/homegrown-academy/internal/mkt/adapters"
	"github.com/homegrown-academy/homegrown-academy/internal/comply"
	"github.com/homegrown-academy/homegrown-academy/internal/lifecycle"
	"github.com/homegrown-academy/homegrown-academy/internal/onboard"
	"github.com/homegrown-academy/homegrown-academy/internal/plan"
	"github.com/homegrown-academy/homegrown-academy/internal/recs"
	"github.com/homegrown-academy/homegrown-academy/internal/safety"
	safetyadapters "github.com/homegrown-academy/homegrown-academy/internal/safety/adapters"
	"github.com/homegrown-academy/homegrown-academy/internal/search"
	searchadapters "github.com/homegrown-academy/homegrown-academy/internal/search/adapters"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social"
	"gorm.io/gorm"
)

// version is set at build time via -ldflags '-X main.version=x.y.z'.
var version string

func main() {
	// ── Step 1: Bootstrap logger with text defaults (before config is available) ──
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// ── Step 2: Load config ───────────────────────────────────────────────────────
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Re-initialize logger with config-specified format and level.
	initLogger(cfg)
	slog.Info("starting Homegrown Academy",
		"version", version,
		"environment", cfg.Environment,
	)

	// ── Step 3: Create database pool ─────────────────────────────────────────────
	db, err := shared.CreatePool(cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("failed to get underlying sql.DB", "error", err)
		os.Exit(1)
	}

	// ── Step 4: Run database migrations (goose) ───────────────────────────────────
	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("failed to set goose dialect", "error", err)
		os.Exit(1)
	}
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		slog.Error("database migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations applied")

	// ── Step 5: Create cache ──────────────────────────────────────────────────────
	ctx := context.Background()
	cache, err := shared.CreateCache(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to cache", "error", err)
		os.Exit(1)
	}

	// ── Step 5.6: Create job enqueuer ────────────────────────────────────────────
	jobs, err := shared.CreateJobEnqueuer(cfg)
	if err != nil {
		slog.Error("failed to create job enqueuer", "error", err)
		os.Exit(1)
	}

	// ── Step 5.5: Init Sentry error reporter ──────────────────────────────────────
	var errReporter shared.ErrorReporter = shared.NoopErrorReporter{}
	if cfg.ErrorReportingDSN != nil {
		if initErr := sentry.Init(sentry.ClientOptions{
			Dsn:              *cfg.ErrorReportingDSN,
			Release:          version,
			TracesSampleRate: 0.1,
		}); initErr != nil {
			slog.Error("sentry initialization failed", "error", initErr)
			// Non-fatal: continue with noop reporter.
		} else {
			errReporter = sentryReporter{}
		}
	}

	// ── Step 6: Init EventBus + register subscriptions ───────────────────────────
	eventBus := shared.NewEventBus()
	// Domain subscriptions registered after domains are wired (Step 7d for onboard::).
	// eventBus.Subscribe(reflect.TypeOf(iam.FamilyCreated{}), social.NewOnFamilyCreatedHandler(socialSvc))

	// ── Step 7: Wire IAM domain ───────────────────────────────────────────────────
	kratosAdapter := iamadapters.NewKratosAdapter(cfg.AuthAdminURL, cfg.AuthPublicURL)

	familyRepo := iam.NewPgFamilyRepository(db)
	parentRepo := iam.NewPgParentRepository(db)
	studentRepo := iam.NewPgStudentRepository(db)
	inviteRepo := iam.NewPgCoParentInviteRepository(db)
	sessionRepo := iam.NewPgStudentSessionRepository(db)

	iamSvc := iam.NewIamService(familyRepo, parentRepo, studentRepo, inviteRepo, sessionRepo, kratosAdapter, eventBus, db)

	// ── Step 7b: Wire method:: domain ────────────────────────────────────────────
	// method:: is constructed after iam:: because iam:: is a dependency of method::.
	// method:: validates methodology IDs; iam:: persists them. [02-method §11.2]
	defRepo := method.NewPgMethodologyDefinitionRepository(db)
	toolRepo := method.NewPgToolRepository(db)
	activationRepo := method.NewPgToolActivationRepository(db)

	// Adapter bridges iam.IamService → method.IamServiceForMethod without circular import.
	// Closures convert between iam's plain string slugs and method's typed MethodologyID. [02-method §11.2]
	iamForMethod := method.NewIamAdapter(
		func(ctx context.Context, scope *shared.FamilyScope) (method.MethodologyID, []method.MethodologyID, error) {
			primary, secondary, err := iamSvc.GetFamilyMethodologyIDs(ctx, scope)
			if err != nil {
				return "", nil, err
			}
			secondaryIDs := make([]method.MethodologyID, len(secondary))
			for i, s := range secondary {
				secondaryIDs[i] = method.MethodologyID(s)
			}
			return method.MethodologyID(primary), secondaryIDs, nil
		},
		func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*method.StudentInfo, error) {
			resp, err := iamSvc.GetStudent(ctx, scope, studentID)
			if err != nil {
				return nil, err
			}
			var overrideSlug *method.MethodologyID
			if resp.MethodologyOverrideSlug != nil {
				s := method.MethodologyID(*resp.MethodologyOverrideSlug)
				overrideSlug = &s
			}
			return &method.StudentInfo{
				ID:                      resp.ID,
				MethodologyOverrideSlug: overrideSlug,
			}, nil
		},
		func(ctx context.Context, scope *shared.FamilyScope, primarySlug method.MethodologyID, secondarySlugs []method.MethodologyID) error {
			secondary := make([]string, len(secondarySlugs))
			for i, s := range secondarySlugs {
				secondary[i] = string(s)
			}
			return iamSvc.SetFamilyMethodology(ctx, scope, string(primarySlug), secondary)
		},
		func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, override *method.MethodologyID) error {
			// UpdateStudentCommand.MethodologyOverrideSlug is **string:
			// - nil outer pointer = don't change
			// - non-nil outer pointing to nil inner = clear the override
			// - non-nil outer pointing to non-nil inner = set override to that value
			var slugPtr *string
			if override != nil {
				s := string(*override)
				slugPtr = &s
			}
			// Always pass a non-nil outer pointer so the repository applies the change.
			_, err := iamSvc.UpdateStudent(ctx, scope, studentID, iam.UpdateStudentCommand{
				MethodologyOverrideSlug: &slugPtr,
			})
			return err
		},
	)
	methodSvc := method.NewMethodologyService(defRepo, toolRepo, activationRepo, iamForMethod, eventBus)

	// Inject default methodology resolver into IAM now that method:: is wired. [02-method Gap 5b]
	// Wraps method.MethodologyID → string for iam.DefaultMethodologyResolver.
	iamSvc.SetDefaultMethodologyResolver(func(ctx context.Context) (string, error) {
		slug, err := methodSvc.GetDefaultMethodologySlug(ctx)
		if err != nil {
			return "", err
		}
		return string(slug), nil
	})

	// ── Step 7c: Wire discover:: domain ──────────────────────────────────────────
	// discover:: consumes method:: for methodology display names in quiz results.
	// The adapter bridges method::MethodologyService → discover::MethodologyServiceForDiscover
	// without importing the method package from discover::. [ARCH §4.2]
	quizDefRepo := discover.NewPgQuizDefinitionRepository(db)
	quizResRepo := discover.NewPgQuizResultRepository(db)
	discStateRepo := discover.NewPgStateGuideRepository(db)
	discContentRepo := discover.NewPgContentPageRepository(db)

	discoverMethod := discover.NewMethodAdapter(
		func(ctx context.Context, slug string) (string, error) {
			all, err := methodSvc.ListMethodologies(ctx)
			if err != nil {
				return slug, nil // graceful fallback to slug on error [03-discover §15.27]
			}
			for _, m := range all {
				if string(m.Slug) == slug {
					return m.DisplayName, nil
				}
			}
			return slug, nil
		},
	)
	discoverSvc := discover.NewDiscoveryService(quizDefRepo, quizResRepo, discStateRepo, discContentRepo, discoverMethod)

	// ── Step 7d: Wire onboard:: domain ──────────────────────────────────────────
	// onboard:: orchestrates iam::, method::, and discover:: through consumer-defined
	// interfaces. Closures bridge domain types at the composition root. [04-onboard §11]
	wizardRepo := onboard.NewPgWizardProgressRepository(db)
	roadmapRepo := onboard.NewPgRoadmapItemRepository(db)
	recRepo := onboard.NewPgStarterRecommendationRepository(db)
	communityRepo := onboard.NewPgCommunitySuggestionRepository(db)

	iamForOnboard := onboard.NewIamAdapter(
		func(ctx context.Context, scope *shared.FamilyScope, cmd onboard.UpdateFamilyProfileCommand) error {
			_, err := iamSvc.UpdateFamilyProfile(ctx, scope, iam.UpdateFamilyCommand{
				DisplayName:    &cmd.DisplayName,
				StateCode:      cmd.StateCode,
				LocationRegion: cmd.LocationRegion,
			})
			return err
		},
		func(ctx context.Context, scope *shared.FamilyScope, cmd onboard.AddChildCommand) error {
			_, err := iamSvc.CreateStudent(ctx, scope, iam.CreateStudentCommand{
				DisplayName: cmd.DisplayName,
				BirthYear:   cmd.BirthYear,
				GradeLevel:  cmd.GradeLevel,
			})
			return err
		},
		func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error {
			return iamSvc.DeleteStudent(ctx, scope, studentID)
		},
		func(ctx context.Context, familyID uuid.UUID) ([]onboard.OnboardStudentInfo, error) {
			// Construct FamilyScope from familyID for IAM service call.
			// Used by event handler path where no auth context is available.
			scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
			students, err := iamSvc.ListStudents(ctx, &scope)
			if err != nil {
				return nil, err
			}
			result := make([]onboard.OnboardStudentInfo, len(students))
			for i, s := range students {
				result[i] = onboard.OnboardStudentInfo{
					ID:        s.ID,
					BirthYear: s.BirthYear,
				}
			}
			return result, nil
		},
	)

	methodForOnboard := onboard.NewMethodAdapter(
		func(ctx context.Context, slug string) (*onboard.OnboardMethodologyConfig, error) {
			detail, err := methodSvc.GetMethodology(ctx, slug)
			if err != nil {
				return nil, err
			}
			return &onboard.OnboardMethodologyConfig{
				Slug:             string(detail.Slug),
				DisplayName:      detail.DisplayName,
				OnboardingConfig: detail.OnboardingConfig,
				CommunityConfig:  detail.CommunityConfig,
			}, nil
		},
		func(ctx context.Context) (string, error) {
			slug, err := methodSvc.GetDefaultMethodologySlug(ctx)
			if err != nil {
				return "", err
			}
			return string(slug), nil
		},
		func(ctx context.Context, slugs []string) (bool, error) {
			for _, s := range slugs {
				valid, err := methodSvc.ValidateMethodologySlug(ctx, method.MethodologyID(s))
				if err != nil {
					return false, err
				}
				if !valid {
					return false, nil
				}
			}
			return true, nil
		},
		func(ctx context.Context, scope *shared.FamilyScope, primarySlug string, secondarySlugs []string) error {
			secondaryIDs := make([]method.MethodologyID, len(secondarySlugs))
			for i, s := range secondarySlugs {
				secondaryIDs[i] = method.MethodologyID(s)
			}
			_, err := methodSvc.UpdateFamilyMethodology(ctx, scope, method.UpdateMethodologyCommand{
				PrimaryMethodologySlug:    method.MethodologyID(primarySlug),
				SecondaryMethodologySlugs: secondaryIDs,
			})
			return err
		},
	)

	discoverForOnboard := onboard.NewDiscoverAdapter(
		func(ctx context.Context, shareID string) (*onboard.OnboardQuizResult, error) {
			result, err := discoverSvc.GetQuizResult(ctx, shareID)
			if err != nil {
				return nil, err
			}
			recs := make([]onboard.OnboardQuizRecommendation, len(result.Recommendations))
			for i, r := range result.Recommendations {
				recs[i] = onboard.OnboardQuizRecommendation{
					MethodologySlug: r.MethodologySlug,
					MethodologyName: r.MethodologyName,
					ScorePercentage: r.ScorePercentage,
					Explanation:     r.Explanation,
				}
			}
			return &onboard.OnboardQuizResult{
				ShareID:         result.ShareID,
				Recommendations: recs,
			}, nil
		},
		func(ctx context.Context, shareID string, familyID uuid.UUID) error {
			return discoverSvc.ClaimQuizResult(ctx, shareID, familyID)
		},
	)

	onboardSvc := onboard.NewOnboardingService(
		wizardRepo, roadmapRepo, recRepo, communityRepo,
		iamForOnboard, methodForOnboard, discoverForOnboard,
		eventBus, db,
	)

	// Register onboard:: event subscriptions
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyCreated{}), onboard.NewFamilyCreatedHandler(onboardSvc))
	eventBus.Subscribe(reflect.TypeOf(method.FamilyMethodologyChanged{}), onboard.NewFamilyMethodologyChangedHandler(onboardSvc))

	// ── Step 7e: Wire social:: domain ───────────────────────────────────────────
	// social:: consumes iam:: for display names and method:: for methodology display names.
	// Cross-family lookups use FamilyScope constructed from the target family ID. [05-social §17.4]
	feedStore, err := shared.CreateFeedStore(ctx, cfg)
	if err != nil {
		slog.Error("failed to create feed store", "error", err)
		os.Exit(1)
	}
	pubsub, err := shared.CreatePubSub(ctx, cfg)
	if err != nil {
		slog.Error("failed to create pubsub", "error", err)
		os.Exit(1)
	}

	socProfileRepo := social.NewPgProfileRepository(db)
	socFriendshipRepo := social.NewPgFriendshipRepository(db)
	socBlockRepo := social.NewPgBlockRepository(db)
	socPostRepo := social.NewPgPostRepository(db)
	socCommentRepo := social.NewPgCommentRepository(db)
	socLikeRepo := social.NewPgPostLikeRepository(db)
	socConvRepo := social.NewPgConversationRepository(db)
	socConvPartRepo := social.NewPgConversationParticipantRepository(db)
	socMsgRepo := social.NewPgMessageRepository(db)
	socGroupRepo := social.NewPgGroupRepository(db)
	socGroupMemberRepo := social.NewPgGroupMemberRepository(db)
	socPinnedPostRepo := social.NewPgPinnedPostRepository(db)
	socEventRepo := social.NewPgEventRepository(db)
	socRSVPRepo := social.NewPgEventRSVPRepository(db)

	iamForSocial := social.NewIamAdapter(
		// GetFamilyDisplayName
		func(ctx context.Context, familyID uuid.UUID) (string, error) {
			scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
			profile, err := iamSvc.GetFamilyProfile(ctx, &scope)
			if err != nil {
				return "", err
			}
			return profile.DisplayName, nil
		},
		// GetParentDisplayName — cross-family lookup via RLS bypass.
		// Used by social:: for conversation participant display names. [05-social §8.2]
		func(ctx context.Context, parentID uuid.UUID) (string, error) {
			var displayName string
			err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				return tx.Table("iam_parents").Select("display_name").Where("id = ?", parentID).Scan(&displayName).Error
			})
			if err != nil || displayName == "" {
				return parentID.String(), nil // graceful fallback
			}
			return displayName, nil
		},
		// GetFamilyInfo
		func(ctx context.Context, familyID uuid.UUID) (*social.SocialFamilyInfo, error) {
			scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
			profile, err := iamSvc.GetFamilyProfile(ctx, &scope)
			if err != nil {
				return nil, err
			}
			parentNames := make([]string, len(profile.Parents))
			for i, p := range profile.Parents {
				parentNames[i] = p.DisplayName
			}
			return &social.SocialFamilyInfo{
				FamilyID:    familyID,
				DisplayName: profile.DisplayName,
				ParentNames: parentNames,
			}, nil
		},
		// GetParentInfo — cross-family lookup via RLS bypass. [05-social §8.2]
		func(ctx context.Context, parentID uuid.UUID) (*social.SocialParentInfo, error) {
			type parentRow struct {
				FamilyID    uuid.UUID
				DisplayName string
			}
			var row parentRow
			err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				return tx.Table("iam_parents").Select("family_id, display_name").Where("id = ?", parentID).Scan(&row).Error
			})
			if err != nil {
				return &social.SocialParentInfo{
					ParentID:    parentID,
					DisplayName: parentID.String(),
				}, nil // graceful fallback
			}
			return &social.SocialParentInfo{
				ParentID:    parentID,
				FamilyID:    row.FamilyID,
				DisplayName: row.DisplayName,
			}, nil
		},
		// DiscoverFamiliesByRegion — cross-family region-based discovery. [05-social §15, P2-4]
		// Looks up the requester's location_region, then queries iam_families for matches.
		// Uses region-based matching (not GPS/PostGIS) per privacy constraints [CODING §5].
		func(ctx context.Context, requesterFamilyID uuid.UUID, methodologySlug *string, limit int) ([]social.DiscoverableFamilyResponse, error) {
			type familyRow struct {
				ID                     uuid.UUID
				DisplayName            string
				LocationRegion         *string
				PrimaryMethodologySlug string
			}
			// First get requester's region.
			var requester familyRow
			if err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				return tx.Table("iam_families").
					Select("id, display_name, location_region").
					Where("id = ?", requesterFamilyID).
					Scan(&requester).Error
			}); err != nil || requester.LocationRegion == nil || *requester.LocationRegion == "" {
				return []social.DiscoverableFamilyResponse{}, nil
			}
			// Query families in the same region.
			var rows []familyRow
			if err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				q := tx.Table("iam_families").
					Select("id, display_name, location_region, primary_methodology_slug").
					Where("location_region = ? AND id != ?", *requester.LocationRegion, requesterFamilyID)
				if methodologySlug != nil {
					q = q.Where("(primary_methodology_slug = ? OR ? = ANY(secondary_methodology_slugs))", *methodologySlug, *methodologySlug)
				}
				return q.Limit(limit).Find(&rows).Error
			}); err != nil {
				return nil, err
			}
			result := make([]social.DiscoverableFamilyResponse, len(rows))
			for i, r := range rows {
				result[i] = social.DiscoverableFamilyResponse{
					FamilyID:         r.ID,
					DisplayName:      r.DisplayName,
					LocationRegion:   r.LocationRegion,
					MethodologyNames: []string{r.PrimaryMethodologySlug},
				}
			}
			return result, nil
		},
	)

	methodForSocial := social.NewMethodAdapter(
		func(ctx context.Context, slug string) (string, error) {
			all, err := methodSvc.ListMethodologies(ctx)
			if err != nil {
				return slug, nil // graceful fallback to slug on error
			}
			for _, m := range all {
				if string(m.Slug) == slug {
					return m.DisplayName, nil
				}
			}
			return slug, nil
		},
	)

	socialSvc := social.NewSocialService(
		socProfileRepo, socFriendshipRepo, socBlockRepo,
		socPostRepo, socCommentRepo, socLikeRepo,
		socConvRepo, socConvPartRepo, socMsgRepo,
		socGroupRepo, socGroupMemberRepo, socPinnedPostRepo,
		socEventRepo, socRSVPRepo,
		iamForSocial, methodForSocial,
		feedStore, pubsub, jobs, eventBus, db,
	)

	// Register social:: event subscriptions
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyCreated{}), social.NewFamilyCreatedHandler(socialSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.CoParentAdded{}), social.NewCoParentAddedHandler(socialSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.CoParentRemoved{}), social.NewCoParentRemovedHandler(socialSvc))
	eventBus.Subscribe(reflect.TypeOf(learn.MilestoneAchieved{}), social.NewMilestoneAchievedHandler(socialSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), social.NewFamilyDeletionScheduledHandler(socialSvc))

	// ── Step 7e½: Wire media:: domain ──────────────────────────────────────────
	// media:: is the shared file upload infrastructure — all domains that handle
	// user files delegate to media:: for presigned URLs, processing, and storage. [09-media §1]
	mediaUploadRepo := media.NewPgUploadRepository(db)
	mediaProcJobRepo := media.NewPgProcessingJobRepository(db)
	mediaTranscodeRepo := media.NewPgTranscodeJobRepository(db)

	// Storage adapter: use S3 when configured, noop in dev/test.
	var mediaStorage media.ObjectStorageAdapter = media.NoopStorageAdapter{}
	if cfg.ObjectStorageBucket != "" {
		s3Adapter, s3Err := mediaadapters.NewS3StorageAdapter(ctx, mediaadapters.S3Config{
			Endpoint:        cfg.ObjectStorageEndpoint,
			Region:          cfg.ObjectStorageRegion,
			Bucket:          cfg.ObjectStorageBucket,
			AccessKeyID:     cfg.ObjectStorageAccessKeyID,
			SecretAccessKey: cfg.ObjectStorageSecretAccessKey,
		})
		if s3Err != nil {
			slog.Error("failed to create S3 storage adapter", "error", s3Err)
			os.Exit(1)
		}
		mediaStorage = s3Adapter
	}

	// Safety scan adapter: noop until safety:: domain is implemented.
	var mediaSafety media.SafetyScanAdapter = media.NoopSafetyScanAdapter{}

	mediaCfg := &media.MediaConfig{
		PublicURLBase:           cfg.ObjectStoragePublicURL,
		PresignedUploadExpiry:   3600,
		PresignedDownloadExpiry: 3600,
	}

	mediaSvc := media.NewMediaService(
		mediaUploadRepo, mediaProcJobRepo,
		mediaStorage, mediaSafety,
		eventBus, jobs, mediaCfg, db,
	)

	// ── Step 7f: Wire mkt:: domain ──────────────────────────────────────────────
	// mkt:: is the marketplace domain — creator onboarding, publisher management,
	// listing lifecycle, cart/checkout, reviews, and payouts. [07-mkt §7]
	// Wired before learn:: because learn:: depends on mkt:: for publisher verification.
	mktCreatorRepo := mkt.NewPgCreatorRepository(db)
	mktPublisherRepo := mkt.NewPgPublisherRepository(db)
	mktListingRepo := mkt.NewPgListingRepository(db)
	mktListingFileRepo := mkt.NewPgListingFileRepository(db)
	mktCartRepo := mkt.NewPgCartRepository(db)
	mktPurchaseRepo := mkt.NewPgPurchaseRepository(db)
	mktReviewRepo := mkt.NewPgReviewRepository(db)
	mktCuratedSectionRepo := mkt.NewPgCuratedSectionRepository(db)

	paymentAdapter := mktadapters.NewHyperswitchPaymentAdapter(
		cfg.HyperswitchBaseURL, cfg.HyperswitchAPIKey, cfg.HyperswitchWebhookKey,
	)
	mediaAdapter := mktadapters.NewNoopMediaAdapter()

	mktSvc := mkt.NewMarketplaceService(
		mktCreatorRepo, mktPublisherRepo, mktListingRepo, mktListingFileRepo,
		mktCartRepo, mktPurchaseRepo, mktReviewRepo, mktCuratedSectionRepo,
		paymentAdapter, mediaAdapter,
		eventBus, db,
	)

	// Register mkt:: event subscriptions
	eventBus.Subscribe(reflect.TypeOf(method.MethodologyConfigUpdated{}), mkt.NewMethodologyConfigUpdatedHandler(cache))
	eventBus.Subscribe(reflect.TypeOf(safety.ContentFlagged{}), mkt.NewContentFlaggedHandler(mktSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), mkt.NewFamilyDeletionScheduledHandler(mktSvc))

	// ── Step 7g: Wire learn:: domain ───────────────────────────────────────────
	// learn:: consumes iam:: (student verification), method:: (tool resolution),
	// and mkt:: (publisher membership verification). [06-learn §7]
	learnActivityDefRepo := learn.NewPgActivityDefRepository(db)
	learnActivityLogRepo := learn.NewPgActivityLogRepository(db)
	learnReadingItemRepo := learn.NewPgReadingItemRepository(db)
	learnReadingProgressRepo := learn.NewPgReadingProgressRepository(db)
	learnReadingListRepo := learn.NewPgReadingListRepository(db)
	learnJournalEntryRepo := learn.NewPgJournalEntryRepository(db)
	learnArtifactLinkRepo := learn.NewPgArtifactLinkRepository(db)
	learnProgressRepo := learn.NewPgProgressRepository(db)
	learnTaxonomyRepo := learn.NewPgSubjectTaxonomyRepository(db)
	learnExportRepo := learn.NewPgExportRepository(db)
	learnQuestionRepo := learn.NewPgQuestionRepository(db)
	learnQuizDefRepo := learn.NewPgQuizDefRepository(db)
	learnQuizSessionRepo := learn.NewPgQuizSessionRepository(db)
	learnSequenceDefRepo := learn.NewPgSequenceDefRepository(db)
	learnSequenceProgressRepo := learn.NewPgSequenceProgressRepository(db)
	learnAssignmentRepo := learn.NewPgAssignmentRepository(db)
	learnVideoDefRepo := learn.NewPgVideoDefRepository(db)
	learnVideoProgressRepo := learn.NewPgVideoProgressRepository(db)
	learnAssessmentDefRepo := learn.NewPgAssessmentDefRepository(db)
	learnProjectDefRepo := learn.NewPgProjectDefRepository(db)
	learnAssessmentResultRepo := learn.NewPgAssessmentResultRepository(db)
	learnProjectProgressRepo := learn.NewPgProjectProgressRepository(db)
	learnGradingScaleRepo := learn.NewPgGradingScaleRepository(db)

	iamForLearn := learn.NewIamAdapter(
		// StudentBelongsToFamily — construct FamilyScope and try GetStudent
		func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error) {
			scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})
			student, err := iamSvc.GetStudent(ctx, &scope, studentID)
			if err != nil {
				return false, nil // student not found or not in family
			}
			return student != nil, nil
		},
		iamSvc.GetStudentName,
	)

	methodForLearn := learn.NewMethodAdapter(
		// ResolveFamilyTools — bridge method.ActiveToolResponse → learn.ActiveToolResponse
		func(ctx context.Context, scope *shared.FamilyScope) ([]learn.ActiveToolResponse, error) {
			tools, err := methodSvc.ResolveFamilyTools(ctx, scope)
			if err != nil {
				return nil, err
			}
			result := make([]learn.ActiveToolResponse, len(tools))
			for i, t := range tools {
				configOverrides := make(map[string]any)
				if t.ConfigOverrides != nil {
					_ = json.Unmarshal(t.ConfigOverrides, &configOverrides)
				}
				result[i] = learn.ActiveToolResponse{
					Slug:            string(t.Slug),
					DisplayName:     t.DisplayName,
					Label:           t.Label,
					Description:     t.Description,
					Tier:            t.Tier,
					Guidance:        t.Guidance,
					ConfigOverrides: configOverrides,
					SortOrder:       t.SortOrder,
				}
			}
			return result, nil
		},
		// ResolveStudentTools — bridge method.ActiveToolResponse → learn.ActiveToolResponse
		func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]learn.ActiveToolResponse, error) {
			tools, err := methodSvc.ResolveStudentTools(ctx, scope, studentID)
			if err != nil {
				return nil, err
			}
			result := make([]learn.ActiveToolResponse, len(tools))
			for i, t := range tools {
				configOverrides := make(map[string]any)
				if t.ConfigOverrides != nil {
					_ = json.Unmarshal(t.ConfigOverrides, &configOverrides)
				}
				result[i] = learn.ActiveToolResponse{
					Slug:            string(t.Slug),
					DisplayName:     t.DisplayName,
					Label:           t.Label,
					Description:     t.Description,
					Tier:            t.Tier,
					Guidance:        t.Guidance,
					ConfigOverrides: configOverrides,
					SortOrder:       t.SortOrder,
				}
			}
			return result, nil
		},
	)

	learnSvc := learn.NewLearningService(
		learnActivityDefRepo, learnActivityLogRepo,
		learnReadingItemRepo, learnReadingProgressRepo,
		learnReadingListRepo, learnJournalEntryRepo,
		learnArtifactLinkRepo, learnProgressRepo,
		learnTaxonomyRepo, learnExportRepo,
		learnQuestionRepo, learnQuizDefRepo,
		learnQuizSessionRepo, learnSequenceDefRepo,
		learnSequenceProgressRepo, learnAssignmentRepo,
		learnVideoDefRepo, learnVideoProgressRepo,
		learnAssessmentDefRepo, learnProjectDefRepo,
		learnAssessmentResultRepo, learnProjectProgressRepo,
		learnGradingScaleRepo,
		iamForLearn, methodForLearn,
		learn.NewMktAdapter(
			func(ctx context.Context, callerID, publisherID uuid.UUID) (bool, error) {
				return mktSvc.VerifyPublisherMembership(ctx, publisherID, callerID)
			},
		),
		eventBus, db,
	)

	// Register learn:: event subscriptions
	eventBus.Subscribe(reflect.TypeOf(iam.StudentCreated{}), learn.NewStudentCreatedHandler(learnSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.StudentDeleted{}), learn.NewStudentDeletedHandler(learnSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), learn.NewFamilyDeletionScheduledHandler(learnSvc))
	eventBus.Subscribe(reflect.TypeOf(mkt.PurchaseCompleted{}), learn.NewPurchaseCompletedHandler(learnSvc))

	// ── Step 7h: Wire notify:: domain ───────────────────────────────────────────
	// notify:: is the cross-cutting event-driven dispatch system. It receives domain
	// events from every other module and delivers in-app notifications, transactional
	// email, and streak detection. [08-notify §1]
	notifRepo := notify.NewPgNotificationRepository(db)
	notifPrefRepo := notify.NewPgPreferenceRepository(db)
	notifDigestRepo := notify.NewPgDigestRepository(db)

	// Email adapter: use Postmark when token is set, noop otherwise (dev/test). [08-notify §7]
	var emailAdapter notify.EmailAdapter = notifyadapters.NoopEmailAdapter{}
	if cfg.PostmarkServerToken != "" {
		emailAdapter = notifyadapters.NewPostmarkEmailAdapter(cfg.PostmarkServerToken)
	}

	// IAM adapter for notify: bridges iam.IamService → notify.IamServiceForNotify.
	iamForNotify := notify.NewIamAdapter(
		// GetFamilyPrimaryEmail — fetch family's primary parent email and family display name.
		// Uses RLS bypass since event handlers don't have auth context. [08-notify §7]
		func(ctx context.Context, familyID uuid.UUID) (string, string, error) {
			type emailRow struct {
				Email       string
				DisplayName string
			}
			var row emailRow
			err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				return tx.Table("iam_parents").
					Select("iam_parents.email, iam_families.display_name").
					Joins("JOIN iam_families ON iam_families.id = iam_parents.family_id").
					Where("iam_parents.family_id = ? AND iam_parents.is_primary = true", familyID).
					Limit(1).Scan(&row).Error
			})
			if err != nil {
				return "", "", err
			}
			return row.Email, row.DisplayName, nil
		},
		// GetFamilyIDForParent — reverse lookup from parent ID to family ID
		func(ctx context.Context, parentID uuid.UUID) (uuid.UUID, error) {
			var familyID uuid.UUID
			err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				return tx.Table("iam_parents").Select("family_id").Where("id = ?", parentID).Scan(&familyID).Error
			})
			return familyID, err
		},
		// GetFamilyIDForCreator — resolves creator → parent → family for PayoutCompleted handler
		func(ctx context.Context, creatorID uuid.UUID) (uuid.UUID, error) {
			var familyID uuid.UUID
			err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				return tx.Raw(`
					SELECT p.family_id FROM mkt_creators c
					JOIN iam_parents p ON p.id = c.parent_id
					WHERE c.id = ?
				`, creatorID).Scan(&familyID).Error
			})
			return familyID, err
		},
	)

	notifySvc := notify.NewNotificationService(
		notifRepo, notifPrefRepo, notifDigestRepo,
		emailAdapter, iamForNotify,
		cache, pubsub, jobs,
		cfg.UnsubscribeSecret,
	)

	// Register notify:: event subscriptions (Phase 1 — 14 handlers)
	// social:: events
	eventBus.Subscribe(reflect.TypeOf(social.FriendRequestSent{}), notify.NewFriendRequestSentHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(social.FriendRequestAccepted{}), notify.NewFriendRequestAcceptedHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(social.MessageSent{}), notify.NewMessageSentHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(social.EventCancelled{}), notify.NewEventCancelledHandler(notifySvc))
	// method:: events
	eventBus.Subscribe(reflect.TypeOf(method.FamilyMethodologyChanged{}), notify.NewFamilyMethodologyChangedHandler(notifySvc))
	// onboard:: events
	eventBus.Subscribe(reflect.TypeOf(onboard.OnboardingCompleted{}), notify.NewOnboardingCompletedHandler(notifySvc))
	// learn:: events
	eventBus.Subscribe(reflect.TypeOf(learn.ActivityLogged{}), notify.NewActivityLoggedHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(learn.MilestoneAchieved{}), notify.NewMilestoneAchievedHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(learn.BookCompleted{}), notify.NewBookCompletedHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(learn.DataExportReady{}), notify.NewDataExportReadyHandler(notifySvc))
	// mkt:: events
	eventBus.Subscribe(reflect.TypeOf(mkt.PurchaseCompleted{}), notify.NewPurchaseCompletedHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(mkt.PurchaseRefunded{}), notify.NewPurchaseRefundedHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(mkt.CreatorOnboarded{}), notify.NewCreatorOnboardedHandler(notifySvc))
	// safety:: events → notify + mkt
	eventBus.Subscribe(reflect.TypeOf(safety.ContentFlagged{}), notify.NewContentFlaggedHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(safety.ContentFlagged{}), mkt.NewContentFlaggedHandler(mktSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.CoParentAdded{}), notify.NewCoParentAddedHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), notify.NewFamilyDeletionScheduledHandler(notifySvc))

	// ── Step 7i: Wire billing:: domain ──────────────────────────────────────────
	// billing:: is the subscription lifecycle and tier-gating engine. Hyperswitch is
	// authoritative for subscription state; local DB mirrors via webhooks. [10-billing §1]
	billSubRepo := billing.NewPgSubscriptionRepository(db)
	billTxRepo := billing.NewPgTransactionRepository(db)
	billCustRepo := billing.NewPgCustomerRepository(db)
	billPayoutRepo := billing.NewPgPayoutRepository(db)

	// Hyperswitch billing adapter: uses billing-specific profile, separate from mkt.
	billAdapter := billingadapters.NewHyperswitchSubscriptionAdapter(
		cfg.HyperswitchBaseURL, cfg.HyperswitchAPIKey,
		cfg.HyperswitchBillingProfileID, cfg.BillingWebhookSecret,
	)

	// IAM adapter for billing: bridges iam:: → billing::IamServiceForBilling
	iamForBilling := billing.NewIamAdapter(
		// GetFamilyPrimaryEmail — uses RLS bypass since event handlers lack auth context. [10-billing §12]
		func(ctx context.Context, familyID uuid.UUID) (string, string, error) {
			type emailRow struct {
				Email       string
				DisplayName string
			}
			var row emailRow
			err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				return tx.Table("iam_parents").
					Select("iam_parents.email, iam_families.display_name").
					Joins("JOIN iam_families ON iam_families.id = iam_parents.family_id").
					Where("iam_parents.family_id = ? AND iam_parents.is_primary = true", familyID).
					Limit(1).Scan(&row).Error
			})
			if err != nil {
				return "", "", err
			}
			return row.Email, row.DisplayName, nil
		},
	)

	billCfg := billing.BillingConfig{
		HyperswitchAPIKey:    cfg.HyperswitchAPIKey,
		HyperswitchProfileID: cfg.HyperswitchBillingProfileID,
		HyperswitchBaseURL:   cfg.HyperswitchBaseURL,
		MonthlyPriceID:       cfg.HyperswitchMonthlyPriceID,
		AnnualPriceID:        cfg.HyperswitchAnnualPriceID,
		CoppaChargeCents:     cfg.CoppaChargeCents,
		WebhookSigningSecret: cfg.BillingWebhookSecret,
	}

	billingSvc := billing.NewBillingService(
		billSubRepo, billTxRepo, billCustRepo, billPayoutRepo,
		billAdapter, iamForBilling,
		eventBus, billCfg,
	)

	// Inject billing adapter into iam:: for COPPA credit-card micro-charge. [§9.3]
	iamSvc.SetBillingService(iamBillingAdapter{svc: billingSvc})

	// Register billing:: event subscriptions
	eventBus.Subscribe(reflect.TypeOf(mkt.PurchaseCompleted{}), billing.NewPurchaseCompletedHandler(billingSvc))
	eventBus.Subscribe(reflect.TypeOf(mkt.PurchaseRefunded{}), billing.NewPurchaseRefundedHandler(billingSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), billing.NewFamilyDeletionScheduledHandler(billingSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.PrimaryParentTransferred{}), billing.NewPrimaryParentTransferredHandler(billingSvc))

	// Now that billing:: is wired, register billing event handlers in notify::
	eventBus.Subscribe(reflect.TypeOf(billing.SubscriptionCreated{}), notify.NewSubscriptionCreatedHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(billing.SubscriptionChanged{}), notify.NewSubscriptionChangedHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(billing.SubscriptionCancelled{}), notify.NewSubscriptionCancelledHandler(notifySvc))
	eventBus.Subscribe(reflect.TypeOf(billing.PayoutCompleted{}), notify.NewPayoutCompletedHandler(notifySvc))

	// ── Step 7j: Wire safety:: domain ──────────────────────────────────────────
	// safety:: is the trust & safety domain — content moderation, CSAM detection,
	// account enforcement, user reporting, appeals, and bot detection. [11-safety §1]
	safetyReportRepo := safety.NewPgReportRepository(db)
	safetyFlagRepo := safety.NewPgContentFlagRepository(db)
	safetyActionRepo := safety.NewPgModActionRepository(db)
	safetyAccountRepo := safety.NewPgAccountStatusRepository(db)
	safetyAppealRepo := safety.NewPgAppealRepository(db)
	safetyNcmecRepo := safety.NewPgNcmecReportRepository(db)
	safetyBotRepo := safety.NewPgBotSignalRepository(db)

	// IAM adapter for safety: bridges iam:: → safety::IamServiceForSafety. [CRIT-SAFETY]
	iamForSafety := safety.NewIamAdapter(func(ctx context.Context, familyID uuid.UUID) error {
		return iamSvc.RevokeFamilySessions(ctx, familyID)
	})

	safetyCfg := safety.DefaultSafetyConfig()
	textScanner := safety.NewTextScanner(safetyCfg)

	// Phase 2 repos + adapters
	safetyParentalControlRepo := safety.NewPgParentalControlRepository(db)
	safetyAdminRoleRepo := safety.NewPgAdminRoleRepository(db)
	safetyAdminRoleAssignRepo := safety.NewPgAdminRoleAssignmentRepository(db)
	safetyGroomingScoreRepo := safety.NewPgGroomingScoreRepository(db)
	groomingDetector := safetyadapters.NewKeywordGroomingDetector()

	safetySvc := safety.NewSafetyService(
		safetyReportRepo, safetyFlagRepo, safetyActionRepo, safetyAccountRepo,
		safetyAppealRepo, safetyNcmecRepo, safetyBotRepo,
		iamForSafety, cache, eventBus, jobs, textScanner, safetyCfg,
		safetyParentalControlRepo, safetyAdminRoleRepo,
		safetyAdminRoleAssignRepo, safetyGroomingScoreRepo,
		groomingDetector,
	)

	// Register safety:: event subscriptions
	eventBus.Subscribe(reflect.TypeOf(media.UploadQuarantined{}), safety.NewUploadQuarantinedHandler(safetySvc))
	eventBus.Subscribe(reflect.TypeOf(media.UploadRejected{}), safety.NewUploadRejectedHandler(safetyFlagRepo, eventBus))
	eventBus.Subscribe(reflect.TypeOf(media.UploadFlagged{}), safety.NewUploadFlaggedHandler(safetyFlagRepo))
	eventBus.Subscribe(reflect.TypeOf(social.PostCreated{}), safety.NewPostCreatedHandler(safetySvc, safetyFlagRepo))
	eventBus.Subscribe(reflect.TypeOf(mkt.ReviewCreated{}), safety.NewReviewCreatedHandler(safetySvc, safetyFlagRepo))
	eventBus.Subscribe(reflect.TypeOf(social.MessageReported{}), safety.NewMessageReportedHandler(safetyFlagRepo))

	// ── Step 7k: Wire search:: domain ──────────────────────────────────────────
	// search:: is a read-only cross-cutting query system. It reads search_vector
	// columns on domain tables via PostgreSQL FTS. No cross-domain adapters needed —
	// privacy enforcement is pure SQL (JOINs on soc_friendships, soc_blocks, etc.). [12-search §1]
	searchSocialRepo := search.NewPgSocialSearchRepository(db)
	searchMktRepo := search.NewPgMarketplaceSearchRepository(db)
	searchLearnRepo := search.NewPgLearningSearchRepository(db)
	searchAutoRepo := search.NewPgAutocompleteRepository(db)

	// Typesense adapter: use real HTTP adapter if TYPESENSE_URL is configured, otherwise noop.
	var typesenseAdapter search.TypesenseAdapter
	if tsURL := os.Getenv("TYPESENSE_URL"); tsURL != "" {
		tsKey := os.Getenv("TYPESENSE_API_KEY")
		typesenseAdapter = searchadapters.NewHttpTypesenseAdapter(tsURL, tsKey)
		slog.Info("search: Typesense adapter configured", "url", tsURL)
	} else {
		typesenseAdapter = &searchadapters.NoopTypesenseAdapter{}
	}

	searchSvc := search.NewSearchService(searchSocialRepo, searchMktRepo, searchLearnRepo, searchAutoRepo, typesenseAdapter)

	// Register search:: event subscriptions (Phase 1 — all no-ops)
	eventBus.Subscribe(reflect.TypeOf(social.PostCreated{}), search.NewPostCreatedHandler(searchSvc))
	eventBus.Subscribe(reflect.TypeOf(mkt.ListingPublished{}), search.NewListingPublishedHandler(searchSvc))
	eventBus.Subscribe(reflect.TypeOf(mkt.ListingArchived{}), search.NewListingArchivedHandler(searchSvc))
	eventBus.Subscribe(reflect.TypeOf(media.UploadPublished{}), search.NewUploadPublishedHandler(searchSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), search.NewFamilyDeletionScheduledHandler(searchSvc))

	// ── Step 7l: Wire recs:: domain ──────────────────────────────────────────────
	// recs:: is the recommendations & signals domain. It records learning signals from
	// domain events and (Phase 2) serves methodology-constrained recommendations to
	// premium families. No external adapter — rule-based engine only. [13-recs §1]
	recSignalRepo     := recs.NewPgSignalRepository(db)
	recRecRepo        := recs.NewPgRecommendationRepository(db)
	recFeedbackRepo   := recs.NewPgFeedbackRepository(db)
	recPopularityRepo := recs.NewPgPopularityRepository(db)
	recPrefRepo       := recs.NewPgPreferenceRepository(db)
	recAnonRepo       := recs.NewPgAnonymizedInteractionRepository(db)

	// IAM adapter for recs: bridges iam.IamService → recs.IamServiceForRecs.
	// StudentBelongsToFamily reuses the GetStudent approach from learn::. [13-recs §16.2]
	iamForRecs := recs.NewIamAdapter(
		func(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error) {
			scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID.UUID})
			student, err := iamSvc.GetStudent(ctx, &scope, studentID)
			if err != nil {
				return false, nil // student not found or not in family
			}
			return student != nil, nil
		},
		func(ctx context.Context, familyID shared.FamilyID) (string, error) {
			scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID.UUID})
			primary, _, err := iamSvc.GetFamilyMethodologyIDs(ctx, &scope)
			return primary, err
		},
	)

	recsSvc := recs.NewRecsService(
		recSignalRepo, recRecRepo, recFeedbackRepo,
		recPopularityRepo, recPrefRepo, recAnonRepo,
		iamForRecs,
	)

	// Register recs:: event subscriptions (Phase 1 — signal recording + lifecycle)
	eventBus.Subscribe(reflect.TypeOf(learn.ActivityLogged{}), recs.NewActivityLoggedHandler(recsSvc))
	eventBus.Subscribe(reflect.TypeOf(learn.BookCompleted{}), recs.NewBookCompletedHandler(recsSvc))
	eventBus.Subscribe(reflect.TypeOf(mkt.PurchaseCompleted{}), recs.NewPurchaseCompletedHandler(recsSvc))
	eventBus.Subscribe(reflect.TypeOf(mkt.ListingPublished{}), recs.NewListingPublishedHandler(recsSvc))
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), recs.NewFamilyDeletionScheduledHandler(recsSvc))
	eventBus.Subscribe(reflect.TypeOf(method.MethodologyConfigUpdated{}), recs.NewMethodologyConfigUpdatedHandler(recsSvc))
	eventBus.Subscribe(reflect.TypeOf(recs.RecommendationsGenerated{}), notify.NewRecommendationsGeneratedHandler(notifySvc))

	// ── Step 7m: Wire comply:: domain ──────────────────────────────────────────────
	// comply:: is the compliance & reporting domain. It tracks attendance, assessments,
	// portfolios, transcripts, courses, and GPA for state homeschool compliance. [14-comply §1]
	complyStateConfigRepo   := comply.NewPgStateConfigRepository(db)
	complyFamilyConfigRepo  := comply.NewPgFamilyConfigRepository(db)
	complyScheduleRepo      := comply.NewPgScheduleRepository(db)
	complyAttendanceRepo    := comply.NewPgAttendanceRepository(db)
	complyAssessmentRepo    := comply.NewPgAssessmentRepository(db)
	complyTestRepo          := comply.NewPgTestScoreRepository(db)
	complyPortfolioRepo     := comply.NewPgPortfolioRepository(db)
	complyPortfolioItemRepo := comply.NewPgPortfolioItemRepository(db)
	complyTranscriptRepo    := comply.NewPgTranscriptRepository(db)
	complyCourseRepo        := comply.NewPgCourseRepository(db)

	// IAM adapter for comply: bridges iam.IamService → comply.IamServiceForComply.
	iamForComply := comply.NewIamAdapter(
		func(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error) {
			scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID.UUID})
			student, err := iamSvc.GetStudent(ctx, &scope, studentID)
			if err != nil {
				return false, nil
			}
			return student != nil, nil
		},
		iamSvc.GetStudentName,
	)

	// Learn adapter for comply: bridges learn:: activity data into portfolio items. [14-comply §9.2]
	// Delegates to learn service instead of raw SQL — respects bounded context. [SS2.7, SS8.1]
	learnForComply := comply.NewLearnAdapter(
		func(ctx context.Context, familyID uuid.UUID, sourceType string, sourceID uuid.UUID) (*comply.PortfolioItemData, error) {
			summary, err := learnSvc.GetPortfolioItemSummary(ctx, familyID, sourceType, sourceID)
			if err != nil {
				return nil, err
			}
			return &comply.PortfolioItemData{
				Title:       summary.Title,
				Description: summary.Description,
				Subject:     summary.Subject,
				Date:        summary.Date,
			}, nil
		},
	)

	// Discovery adapter for comply: bridges discover.DiscoveryService → comply.DiscoveryServiceForComply.
	discoverForComply := comply.NewDiscoveryAdapter(
		func(ctx context.Context, stateCode string) (*comply.StateRequirementsData, error) {
			reqs, err := discoverSvc.GetStateRequirements(ctx, stateCode)
			if err != nil {
				return nil, err
			}
			if reqs == nil {
				return nil, nil
			}
			// Get state name from guide list (best-effort).
			stateName := stateCode
			guides, listErr := discoverSvc.ListStateGuides(ctx)
			if listErr == nil {
				for _, g := range guides {
					if g.StateCode == stateCode {
						stateName = g.StateName
						break
					}
				}
			}
			// Convert *uint16 → *int16 for attendance days.
			var attendanceDays *int16
			if reqs.AttendanceDays != nil {
				v := int16(*reqs.AttendanceDays)
				attendanceDays = &v
			}
			return &comply.StateRequirementsData{
				StateCode:               stateCode,
				StateName:               stateName,
				NotificationRequired:    reqs.NotificationRequired,
				NotificationDetails:     reqs.NotificationDetails,
				RequiredSubjects:        reqs.RequiredSubjects,
				AssessmentRequired:      reqs.AssessmentRequired,
				AssessmentDetails:       reqs.AssessmentDetails,
				RecordKeepingRequired:   reqs.RecordKeepingRequired,
				RecordKeepingDetails:    reqs.RecordKeepingDetails,
				AttendanceRequired:      reqs.AttendanceRequired,
				AttendanceDays:          attendanceDays,
				AttendanceDetails:       reqs.AttendanceDetails,
				UmbrellaSchoolAvailable: reqs.UmbrellaSchoolAvailable,
				UmbrellaSchoolDetails:   reqs.UmbrellaSchoolDetails,
				RegulationLevel:         reqs.RegulationLevel,
			}, nil
		},
		func(ctx context.Context) ([]comply.StateGuideSummary, error) {
			guides, err := discoverSvc.ListStateGuides(ctx)
			if err != nil {
				return nil, err
			}
			out := make([]comply.StateGuideSummary, len(guides))
			for i, g := range guides {
				out[i] = comply.StateGuideSummary{
					StateCode: g.StateCode,
					StateName: g.StateName,
				}
			}
			return out, nil
		},
	)

	// Media adapter for comply: bridges media:: for server-side PDF upload + presigned download. [14-comply §9.2]
	mediaForComply := comply.NewMediaAdapter(
		// RequestUpload: server-side upload via object storage + create upload record.
		func(ctx context.Context, familyID uuid.UUID, uploadContext string, filename string, contentType string, data []byte) (*uuid.UUID, error) {
			scope := shared.NewFamilyScopeFromID(familyID)
			uploadID := uuid.New()
			storageKey := fmt.Sprintf("comply/%s/%s/%s", familyID, uploadID, filename)
			if err := mediaStorage.PutObject(ctx, storageKey, data, contentType); err != nil {
				return nil, fmt.Errorf("comply: media upload put: %w", err)
			}
			// Create upload record so the file is tracked and discoverable.
			upload, err := mediaUploadRepo.Create(ctx, scope, &media.CreateUploadRow{
				ID:               uploadID,
				OriginalFilename: filename,
				ContentType:      contentType,
				StorageKey:       storageKey,
				Context:          media.UploadContext(uploadContext),
				ExpiresAt:        time.Now().Add(365 * 24 * time.Hour), // PDFs don't expire
			})
			if err != nil {
				return nil, fmt.Errorf("comply: media upload record: %w", err)
			}
			// Mark as published (server-side upload bypasses normal processing pipeline).
			if _, err := mediaUploadRepo.UpdateStatus(ctx, upload.ID, media.UploadStatusPublished, nil); err != nil {
				slog.Error("comply: media upload status update", "upload_id", upload.ID, "error", err)
			}
			return &uploadID, nil
		},
		// PresignedGet: look up storage key by upload ID, then generate presigned URL.
		func(ctx context.Context, uploadID uuid.UUID) (string, error) {
			upload, err := mediaUploadRepo.FindByIDUnscoped(ctx, uploadID)
			if err != nil {
				return "", fmt.Errorf("comply: media presigned lookup: %w", err)
			}
			return mediaSvc.PresignedGet(ctx, upload.StorageKey, 3600)
		},
	)

	complySvc := comply.NewComplianceService(
		complyStateConfigRepo, complyFamilyConfigRepo, complyScheduleRepo,
		complyAttendanceRepo, complyAssessmentRepo, complyTestRepo,
		complyPortfolioRepo, complyPortfolioItemRepo, complyTranscriptRepo, complyCourseRepo,
		iamForComply, learnForComply, discoverForComply, mediaForComply,
		eventBus,
	)

	// Register comply:: event subscriptions
	eventBus.Subscribe(reflect.TypeOf(learn.ActivityLogged{}), comply.NewActivityLoggedHandler(complySvc))
	eventBus.Subscribe(reflect.TypeOf(iam.StudentDeleted{}), comply.NewStudentDeletedHandler(complySvc))
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), comply.NewFamilyDeletionScheduledHandler(complySvc))
	eventBus.Subscribe(reflect.TypeOf(billing.SubscriptionCancelled{}), comply.NewSubscriptionCancelledHandler(complySvc))

	// ── Step 7m.5: Wire lifecycle:: domain ───────────────────────────────────────
	// lifecycle:: is the GDPR/COPPA data lifecycle domain (export, deletion, recovery,
	// session management). [15-data-lifecycle §1]
	lifecycleExportRepo := lifecycle.NewPgExportRequestRepository(db)
	lifecycleDeletionRepo := lifecycle.NewPgDeletionRequestRepository(db)
	lifecycleRecoveryRepo := lifecycle.NewPgRecoveryRequestRepository(db)

	// IamServiceForLifecycle: bridges lifecycle → IAM/Kratos via function adapter.
	iamForLifecycle := lifecycle.NewIamAdapter(
		// InitiateRecoveryFlow: trigger Kratos email recovery (enum-prevention: errors only logged).
		func(ctx context.Context, email string) error {
			return kratosAdapter.InitiateAccountRecovery(ctx, email)
		},
		// ListSessions: look up identity ID from parentID, then list Kratos sessions.
		func(ctx context.Context, parentID uuid.UUID) ([]lifecycle.SessionInfo, error) {
			var identityIDStr string
			if err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				return tx.Table("iam_parents").
					Select("kratos_identity_id").
					Where("id = ?", parentID).
					Scan(&identityIDStr).Error
			}); err != nil {
				return nil, err
			}
			identityID, err := uuid.Parse(identityIDStr)
			if err != nil {
				return nil, fmt.Errorf("lifecycle: invalid kratos identity ID for parent %s: %w", parentID, err)
			}
			kratosSessions, err := kratosAdapter.ListSessionsForIdentity(ctx, identityID)
			if err != nil {
				return nil, err
			}
			sessions := make([]lifecycle.SessionInfo, 0, len(kratosSessions))
			for _, ks := range kratosSessions {
				sessions = append(sessions, lifecycle.SessionInfo{
					SessionID:  ks.SessionID,
					UserAgent:  ks.UserAgent,
					IPAddress:  ks.IPAddress,
					LastActive: ks.LastActive,
				})
			}
			return sessions, nil
		},
		// RevokeSession: revoke a specific Kratos session by ID.
		func(ctx context.Context, sessionID string) error {
			return kratosAdapter.RevokeSpecificSession(ctx, sessionID)
		},
		// RevokeAllSessions: list sessions for identity, revoke all except current.
		func(ctx context.Context, parentID uuid.UUID, currentSessionID string) (uint32, error) {
			var identityIDStr string
			if err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				return tx.Table("iam_parents").
					Select("kratos_identity_id").
					Where("id = ?", parentID).
					Scan(&identityIDStr).Error
			}); err != nil {
				return 0, err
			}
			identityID, err := uuid.Parse(identityIDStr)
			if err != nil {
				return 0, fmt.Errorf("lifecycle: invalid kratos identity ID for parent %s: %w", parentID, err)
			}
			kratosSession, err := kratosAdapter.ListSessionsForIdentity(ctx, identityID)
			if err != nil {
				return 0, err
			}
			var count uint32
			for _, ks := range kratosSession {
				if ks.SessionID == currentSessionID {
					continue
				}
				if revokeErr := kratosAdapter.RevokeSpecificSession(ctx, ks.SessionID); revokeErr != nil {
					slog.Error("lifecycle: revoke session", "session_id", ks.SessionID, "error", revokeErr)
					continue
				}
				count++
			}
			return count, nil
		},
		// RevokeFamilySessions: delegate to iamSvc which iterates all parent identities.
		func(ctx context.Context, familyID uuid.UUID) error {
			return iamSvc.RevokeFamilySessions(ctx, familyID)
		},
	)

	// BillingServiceForLifecycle: cancels all active subscriptions for a family during deletion.
	billingForLifecycle := lifecycle.NewBillingAdapter(
		func(ctx context.Context, familyID uuid.UUID) error {
			scope := shared.NewFamilyScopeFromID(familyID)
			_, err := billingSvc.CancelSubscription(ctx, scope)
			return err
		},
	)

	// Declare plan adapters early; svc is set after plan:: is wired (Step 7o).
	planDeletion := &lifecyclePlanDeletion{}
	planExport := &lifecyclePlanExport{}

	lifecycleSvc := lifecycle.NewLifecycleService(
		lifecycleExportRepo, lifecycleDeletionRepo, lifecycleRecoveryRepo,
		iamForLifecycle, billingForLifecycle,
		eventBus, jobs,
		[]lifecycle.ExportHandler{
			&lifecycleLearnExport{svc: learnSvc},
			&lifecycleSocialExport{svc: socialSvc},
			&lifecycleMktExport{svc: mktSvc},
			&lifecycleMediaExport{svc: mediaSvc},
			&lifecycleNotifyExport{svc: notifySvc},
			&lifecycleBillingExport{svc: billingSvc},
			&lifecycleComplyExport{svc: complySvc},
			&lifecycleRecsExport{svc: recsSvc},
			planExport, // svc set after plan:: is wired (below)
		},
		[]lifecycle.DeletionHandler{
			&lifecycleLearnDeletion{svc: learnSvc},
			&lifecycleSocialDeletion{svc: socialSvc},
			&lifecycleMktDeletion{svc: mktSvc},
			&lifecycleNotifyDeletion{svc: notifySvc},
			&lifecycleBillingDeletion{svc: billingSvc},
			&lifecycleComplyDeletion{svc: complySvc},
			&lifecycleRecsDeletion{svc: recsSvc},
			&lifecycleMediaDeletion{svc: mediaSvc},
			planDeletion, // svc set after plan:: is wired (below)
		},
	)

	// Subscribe lifecycle to family deletion events.
	eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), lifecycle.NewFamilyDeletionScheduledHandler(lifecycleSvc))

	// ── Step 7n: Wire admin:: domain ────────────────────────────────────────────
	adminFlagRepo := admin.NewPgFeatureFlagRepository(db)
	adminAuditRepo := admin.NewPgAuditLogRepository(db)

	// Wire real cross-domain adapters for admin. [16-admin §14]
	iamForAdmin := &adminIamAdapter{db: db}
	safetyForAdmin := &adminSafetyAdapter{db: db, svc: safetySvc}
	billingForAdmin := &adminBillingAdapter{svc: billingSvc}
	methodForAdmin := &adminMethodAdapter{db: db}
	lifecycleForAdmin := &adminLifecycleAdapter{db: db}
	healthForAdmin := &appHealthChecker{
		db:                    db,
		cache:                 cache,
		kratosPublicURL:       cfg.AuthPublicURL,
		objectStorageEndpoint: cfg.ObjectStorageEndpoint,
	}
	queueInspector, err := admin.NewAsynqQueueInspector(cfg.RedisURL)
	if err != nil {
		slog.Warn("queue inspector unavailable; admin job status will be empty", "error", err)
	}
	jobsForAdmin := &adminJobInspectorAdapter{inspector: queueInspector}

	adminSvc := admin.NewAdminService(
		adminFlagRepo, adminAuditRepo, cache,
		iamForAdmin, safetyForAdmin, billingForAdmin,
		methodForAdmin, lifecycleForAdmin,
		healthForAdmin, jobsForAdmin,
	)

	// ── Step 7o: Wire plan:: domain ────────────────────────────────────────────
	// plan:: is the planning & scheduling domain. It aggregates schedule items,
	// activities, attendance, and events into a unified calendar view. [17-planning §1]
	planRepo := plan.NewPgScheduleItemRepository(db)
	planTemplateRepo := plan.NewPgScheduleTemplateRepository(db)

	iamForPlan := plan.NewIamAdapter(
		func(ctx context.Context, studentID, familyID uuid.UUID) (bool, error) {
			scope := shared.NewFamilyScopeFromID(familyID)
			student, err := iamSvc.GetStudent(ctx, &scope, studentID)
			if err != nil || student == nil {
				return false, err
			}
			return true, nil
		},
		iamSvc.GetStudentName,
	)
	learnForPlan := plan.NewLearnAdapter(
		// ListActivitiesForCalendar: query learn:: activity logs by date range.
		func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]plan.ActivitySummary, error) {
			if studentID == nil {
				return []plan.ActivitySummary{}, nil // plan calendar requires a student filter
			}
			resp, err := learnSvc.ListActivityLogs(ctx, scope, *studentID, learn.ActivityLogQuery{
				DateFrom: &start,
				DateTo:   &end,
				Limit:    500,
			})
			if err != nil {
				return nil, err
			}
			summaries := make([]plan.ActivitySummary, 0, len(resp.Data))
			for _, a := range resp.Data {
				sid := a.StudentID
				s := plan.ActivitySummary{
					ID:        a.ID,
					Title:     a.Title,
					Date:      a.ActivityDate,
					StudentID: &sid,
					Tags:      a.SubjectTags,
				}
				if len(a.SubjectTags) > 0 {
					s.Subject = &a.SubjectTags[0]
				}
				summaries = append(summaries, s)
			}
			return summaries, nil
		},
		// LogActivity: map plan's flat args to learn's LogActivityCommand.
		func(ctx context.Context, _ *shared.AuthContext, scope *shared.FamilyScope, title string, date time.Time, durationMinutes *int, studentID *uuid.UUID, description *string, tags []string) (uuid.UUID, error) {
			if studentID == nil {
				return uuid.Nil, fmt.Errorf("plan: studentID required for LogActivity")
			}
			var dur *int16
			if durationMinutes != nil {
				d := int16(*durationMinutes)
				dur = &d
			}
			resp, err := learnSvc.LogActivity(ctx, scope, *studentID, learn.LogActivityCommand{
				Title:           title,
				Description:     description,
				SubjectTags:     tags,
				DurationMinutes: dur,
				ActivityDate:    &date,
			})
			if err != nil {
				return uuid.Nil, err
			}
			return resp.ID, nil
		},
	)
	complyForPlan := &planComplyAdapter{complySvc: complySvc, iamSvc: iamSvc}
	socialForPlan := &planSocialAdapter{socialSvc: socialSvc}

	planSvc := plan.NewPlanningService(
		planRepo, planTemplateRepo,
		iamForPlan, learnForPlan, complyForPlan, socialForPlan,
	)

	// Complete deferred plan adapter wiring (declared before lifecycle service).
	planDeletion.svc = planSvc
	planExport.svc = planSvc

	// Register plan:: event subscriptions [17-planning §16]
	eventBus.Subscribe(reflect.TypeOf(social.EventCancelled{}), plan.NewEventCancelledHandler(planSvc))
	eventBus.Subscribe(reflect.TypeOf(learn.ActivityLogged{}), plan.NewActivityLoggedHandler(planSvc))

	// ── Step 8: Wire AppState ─────────────────────────────────────────────────────
	state := &app.AppState{
		DB:       db,
		Cache:    cache,
		Auth:     kratosAdapter, // KratosAdapterImpl implements shared.SessionValidator
		Errors:   errReporter,
		Jobs:     jobs,
		EventBus: eventBus,
		Config:   cfg,
		Version:  version,
		IAM:      iamSvc,
		Method:   methodSvc,
		Discover: discoverSvc,
		Onboard:  onboardSvc,
		Social:      socialSvc,
		Learn:       learnSvc,
		Marketplace: mktSvc,
		Media:       mediaSvc,
		Notify:      notifySvc,
		Billing:     billingSvc,
		Safety:      safetySvc,
		Search:      searchSvc,
		Recs:        recsSvc,
		Comply:    complySvc,
		Lifecycle: lifecycleSvc,
		Admin:     adminSvc,
		Plan:      planSvc,
		PubSub:    pubsub,
	}

	// ── Step 8: Build Echo router ─────────────────────────────────────────────────
	e := app.NewApp(state)

	// ── Step 8.5: Start background job worker ────────────────────────────────────
	worker, err := shared.CreateJobWorker(cfg)
	if err != nil {
		slog.Error("failed to create job worker", "error", err)
		os.Exit(1)
	}
	social.RegisterFeedWorkers(worker, feedStore, socFriendshipRepo, socPostRepo)
	media.RegisterMediaWorkers(worker, mediaUploadRepo, mediaProcJobRepo, mediaTranscodeRepo, mediaStorage, mediaSafety, eventBus, jobs)
	learn.RegisterLearnWorkers(worker, learnSvc)
	notify.RegisterTaskHandlers(worker, emailAdapter)
	safetyManualReviewRepo := safety.NewPgManualReviewRepository(db)
	safetyNcmecPendingRepo := safety.NewPgNcmecPendingReportRepository(db)
	manualReviewThornAdapter := safety.NewManualReviewThornAdapter(safetyManualReviewRepo, safetyNcmecPendingRepo)
	safety.RegisterSafetyWorkers(worker, safetyNcmecRepo, manualReviewThornAdapter, jobs, safetySvc)
	recs.RegisterTaskHandlers(worker, db,
		recSignalRepo, recRecRepo, recFeedbackRepo,
		recPopularityRepo, recPrefRepo, recAnonRepo,
		cfg.RecsAnonymizationSecret,
		eventBus,
	)
	comply.RegisterTaskHandlers(worker, db,
		complyStateConfigRepo, complyFamilyConfigRepo, complyAttendanceRepo,
		complyPortfolioRepo, complyPortfolioItemRepo, complyTranscriptRepo, complyCourseRepo,
		iamForComply, discoverForComply, mediaForComply, eventBus,
	)
	lifecycle.RegisterTaskHandlers(worker, lifecycleSvc)
	billingMktAdapter := billing.NewMktAdapter(func(ctx context.Context, from, to time.Time) ([]billing.CreatorEarningSummary, error) {
		rows, err := mktPurchaseRepo.GetAllCreatorSales(ctx, from, to)
		if err != nil {
			return nil, err
		}
		out := make([]billing.CreatorEarningSummary, len(rows))
		for i, r := range rows {
			out[i] = billing.CreatorEarningSummary{
				CreatorID:            r.CreatorID,
				TotalPayoutCents:     r.TotalPayoutCents,
				PurchaseCount:        r.PurchaseCount,
				RefundDeductionCents: r.RefundDeductionCents,
			}
		}
		return out, nil
	})
	billing.RegisterTaskHandlers(worker, billPayoutRepo, billAdapter, billingMktAdapter)
	go func() {
		if startErr := worker.Start(); startErr != nil {
			slog.Error("job worker error", "error", startErr)
		}
	}()

	// ── Step 8.6: Start periodic job scheduler ────────────────────────────────────
	scheduler, err := shared.CreateJobScheduler(cfg)
	if err != nil {
		slog.Warn("job scheduler unavailable (Redis not configured); periodic tasks disabled", "error", err)
		scheduler = shared.NoopJobScheduler{}
	}
	// Weekly progress snapshots for all students — Sunday midnight UTC. [06-learn §12.3]
	if schedErr := scheduler.Register("0 0 * * 0", learn.SnapshotProgressPayload{}); schedErr != nil {
		slog.Error("failed to register learn:snapshot_progress schedule", "error", schedErr)
	}
	if schedErr := scheduler.Start(); schedErr != nil {
		slog.Error("job scheduler failed to start", "error", schedErr)
	}

	// ── Step 9: Start server (non-blocking) ───────────────────────────────────────
	addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	go func() {
		slog.Info("server listening", "address", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	// ── Step 10: Graceful shutdown ────────────────────────────────────────────────
	gracefulShutdown(ctx, e, func() {
		errReporter.Flush(5 * time.Second)
		worker.Stop()
		scheduler.Stop()
		if closeErr := jobs.Close(); closeErr != nil {
			slog.Error("job enqueuer close error", "error", closeErr)
		}
		if closeErr := pubsub.Close(); closeErr != nil {
			slog.Error("pubsub close error", "error", closeErr)
		}
		if closeErr := cache.Close(); closeErr != nil {
			slog.Error("cache close error", "error", closeErr)
		}
		if closeErr := sqlDB.Close(); closeErr != nil {
			slog.Error("database close error", "error", closeErr)
		}
		slog.Info("server stopped")
	})
}

// sentryReporter wraps the Sentry SDK behind the shared.ErrorReporter port.
// The sentry package is isolated here and in initSentry — it MUST NOT appear elsewhere.
type sentryReporter struct{}

func (sentryReporter) CaptureException(err error) { sentry.CaptureException(err) }
func (sentryReporter) Flush(d time.Duration) bool  { return sentry.Flush(d) }

// ─── Admin Domain Adapters ───────────────────────────────────────────────────
// Real implementations of admin:: consumer-defined cross-domain interfaces.
// All use BypassRLSTransaction — admin operations are platform-wide, not family-scoped.
// [16-admin §14, CODING §BypassRLS]

// ─── adminIamAdapter ─────────────────────────────────────────────────────────

type adminIamAdapter struct{ db *gorm.DB }

func (a *adminIamAdapter) SearchUsers(ctx context.Context, query *admin.UserSearchQuery, p *shared.PaginationParams) (*shared.PaginatedResponse[admin.AdminUserSummary], error) {
	type row struct {
		FamilyID           uuid.UUID  `gorm:"column:family_id"`
		FamilyName         string     `gorm:"column:family_name"`
		PrimaryParentEmail string     `gorm:"column:primary_parent_email"`
		ParentCount        int32      `gorm:"column:parent_count"`
		StudentCount       int32      `gorm:"column:student_count"`
		SubscriptionTier   string     `gorm:"column:subscription_tier"`
		AccountStatus      string     `gorm:"column:account_status"`
		CreatedAt          time.Time  `gorm:"column:created_at"`
	}
	limit := p.EffectiveLimit()
	var rows []row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin cross-family query — RLS bypass required.
		q := tx.Raw(`
			SELECT
				f.id AS family_id,
				f.display_name AS family_name,
				pp.email AS primary_parent_email,
				(SELECT COUNT(*) FROM iam_parents WHERE family_id = f.id) AS parent_count,
				(SELECT COUNT(*) FROM iam_students WHERE family_id = f.id) AS student_count,
				f.subscription_tier,
				COALESCE(sas.status, 'active') AS account_status,
				f.created_at
			FROM iam_families f
			JOIN iam_parents pp ON pp.family_id = f.id AND pp.is_primary = true
			LEFT JOIN safety_account_status sas ON sas.family_id = f.id
			WHERE
				(? IS NULL OR pp.email ILIKE '%' || ? || '%' OR f.display_name ILIKE '%' || ? || '%')
				AND (? IS NULL OR f.id = ?::uuid)
				AND (? IS NULL OR COALESCE(sas.status, 'active') = ?)
				AND (? IS NULL OR f.subscription_tier = ?)
			ORDER BY f.created_at DESC
			LIMIT ?`,
			query.Q, query.Q, query.Q,
			query.FamilyID, query.FamilyID,
			query.Status, query.Status,
			query.Subscription, query.Subscription,
			limit,
		)
		return q.Scan(&rows).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.SearchUsers: %w", err)
	}
	summaries := make([]admin.AdminUserSummary, len(rows))
	for i, r := range rows {
		summaries[i] = admin.AdminUserSummary{
			FamilyID:           r.FamilyID,
			FamilyName:         r.FamilyName,
			PrimaryParentEmail: r.PrimaryParentEmail,
			ParentCount:        r.ParentCount,
			StudentCount:       r.StudentCount,
			SubscriptionTier:   r.SubscriptionTier,
			AccountStatus:      r.AccountStatus,
			CreatedAt:          r.CreatedAt,
		}
	}
	return &shared.PaginatedResponse[admin.AdminUserSummary]{Data: summaries}, nil
}

func (a *adminIamAdapter) GetFamilyDetail(ctx context.Context, familyID uuid.UUID) (*admin.AdminFamilyInfo, error) {
	type row struct {
		ID            uuid.UUID  `gorm:"column:id"`
		Name          string     `gorm:"column:name"`
		AccountStatus string     `gorm:"column:account_status"`
		CreatedAt     time.Time  `gorm:"column:created_at"`
	}
	var r row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin cross-family query — RLS bypass required.
		return tx.Raw(`
			SELECT f.id, f.display_name AS name, COALESCE(sas.status, 'active') AS account_status, f.created_at
			FROM iam_families f
			LEFT JOIN safety_account_status sas ON sas.family_id = f.id
			WHERE f.id = ?`, familyID).Scan(&r).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.GetFamilyDetail: %w", err)
	}
	if r.ID == uuid.Nil {
		return nil, admin.ErrUserNotFound
	}
	return &admin.AdminFamilyInfo{
		ID:            r.ID,
		Name:          r.Name,
		AccountStatus: r.AccountStatus,
		CreatedAt:     r.CreatedAt,
	}, nil
}

func (a *adminIamAdapter) GetParents(ctx context.Context, familyID uuid.UUID) ([]admin.AdminParentInfo, error) {
	type row struct {
		ID          uuid.UUID `gorm:"column:id"`
		DisplayName string    `gorm:"column:display_name"`
		Email       string    `gorm:"column:email"`
		IsPrimary   bool      `gorm:"column:is_primary"`
	}
	var rows []row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin cross-family query — RLS bypass required.
		return tx.Raw(`SELECT id, display_name, email, is_primary FROM iam_parents WHERE family_id = ? ORDER BY is_primary DESC, created_at ASC`, familyID).Scan(&rows).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.GetParents: %w", err)
	}
	result := make([]admin.AdminParentInfo, len(rows))
	for i, r := range rows {
		result[i] = admin.AdminParentInfo{ID: r.ID, DisplayName: r.DisplayName, Email: r.Email, IsPrimary: r.IsPrimary}
	}
	return result, nil
}

func (a *adminIamAdapter) GetStudents(ctx context.Context, familyID uuid.UUID) ([]admin.AdminStudentInfo, error) {
	type row struct {
		ID          uuid.UUID `gorm:"column:id"`
		DisplayName string    `gorm:"column:display_name"`
		GradeLevel  *string   `gorm:"column:grade_level"`
	}
	var rows []row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin cross-family query — RLS bypass required.
		return tx.Raw(`SELECT id, display_name, grade_level FROM iam_students WHERE family_id = ? ORDER BY created_at ASC`, familyID).Scan(&rows).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.GetStudents: %w", err)
	}
	result := make([]admin.AdminStudentInfo, len(rows))
	for i, r := range rows {
		result[i] = admin.AdminStudentInfo{ID: r.ID, DisplayName: r.DisplayName, GradeLevel: r.GradeLevel}
	}
	return result, nil
}

// ─── adminSafetyAdapter ──────────────────────────────────────────────────────

type adminSafetyAdapter struct {
	db  *gorm.DB
	svc safety.SafetyService
}

func (a *adminSafetyAdapter) GetModerationHistory(ctx context.Context, familyID uuid.UUID) ([]admin.ModerationActionSummary, error) {
	type row struct {
		ActionType string    `gorm:"column:action_type"`
		Reason     string    `gorm:"column:reason"`
		CreatedAt  time.Time `gorm:"column:created_at"`
	}
	var rows []row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin cross-family moderation history — RLS bypass required.
		return tx.Raw(`SELECT action_type, reason, created_at FROM safety_mod_actions WHERE target_family_id = ? ORDER BY created_at DESC LIMIT 50`, familyID).Scan(&rows).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.GetModerationHistory: %w", err)
	}
	result := make([]admin.ModerationActionSummary, len(rows))
	for i, r := range rows {
		result[i] = admin.ModerationActionSummary{Action: r.ActionType, Reason: r.Reason, CreatedAt: r.CreatedAt}
	}
	return result, nil
}

func (a *adminSafetyAdapter) SuspendAccount(ctx context.Context, familyID uuid.UUID, reason string) error {
	// Platform admin action — use a system auth context.
	systemAuth := &shared.AuthContext{ParentID: uuid.Nil, FamilyID: uuid.Nil, IsPlatformAdmin: true}
	_, err := a.svc.AdminSuspendAccount(ctx, systemAuth, familyID, safety.SuspendAccountCommand{
		Reason:         reason,
		SuspensionDays: 30,
	})
	return err
}

func (a *adminSafetyAdapter) UnsuspendAccount(ctx context.Context, familyID uuid.UUID) error {
	systemAuth := &shared.AuthContext{ParentID: uuid.Nil, FamilyID: uuid.Nil, IsPlatformAdmin: true}
	_, err := a.svc.AdminLiftSuspension(ctx, systemAuth, familyID, safety.LiftSuspensionCommand{
		Reason: "Admin lift via admin panel",
	})
	return err
}

func (a *adminSafetyAdapter) BanAccount(ctx context.Context, familyID uuid.UUID, reason string) error {
	systemAuth := &shared.AuthContext{ParentID: uuid.Nil, FamilyID: uuid.Nil, IsPlatformAdmin: true}
	_, err := a.svc.AdminBanAccount(ctx, systemAuth, familyID, safety.BanAccountCommand{Reason: reason})
	return err
}

func (a *adminSafetyAdapter) GetReviewQueue(ctx context.Context, p *shared.PaginationParams) ([]admin.ModerationQueueItem, error) {
	type row struct {
		ID            uuid.UUID       `gorm:"column:id"`
		TargetType    string          `gorm:"column:target_type"`
		TargetID      uuid.UUID       `gorm:"column:target_id"`
		TargetFamilyID *uuid.UUID     `gorm:"column:target_family_id"`
		FlagType      string          `gorm:"column:flag_type"`
		Labels        []byte          `gorm:"column:labels"`
		CreatedAt     time.Time       `gorm:"column:created_at"`
	}
	limit := p.EffectiveLimit()
	var rows []row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin moderation queue — RLS bypass required (cross-family view).
		return tx.Raw(`
			SELECT id, target_type, target_id, target_family_id, flag_type, labels, created_at
			FROM safety_content_flags
			WHERE reviewed = false AND auto_rejected = false
			ORDER BY created_at ASC
			LIMIT ?`, limit).Scan(&rows).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.GetReviewQueue: %w", err)
	}
	result := make([]admin.ModerationQueueItem, len(rows))
	for i, r := range rows {
		familyID := uuid.Nil
		if r.TargetFamilyID != nil {
			familyID = *r.TargetFamilyID
		}
		result[i] = admin.ModerationQueueItem{
			ID:          r.ID,
			ContentType: r.TargetType,
			ContentID:   r.TargetID,
			FamilyID:    familyID,
			Reason:      r.FlagType,
			Status:      "pending",
			Details:     r.Labels,
			CreatedAt:   r.CreatedAt,
		}
	}
	return result, nil
}

func (a *adminSafetyAdapter) GetReviewQueueItem(ctx context.Context, itemID uuid.UUID) (*admin.ModerationQueueItem, error) {
	type row struct {
		ID             uuid.UUID  `gorm:"column:id"`
		TargetType     string     `gorm:"column:target_type"`
		TargetID       uuid.UUID  `gorm:"column:target_id"`
		TargetFamilyID *uuid.UUID `gorm:"column:target_family_id"`
		FlagType       string     `gorm:"column:flag_type"`
		Labels         []byte     `gorm:"column:labels"`
		CreatedAt      time.Time  `gorm:"column:created_at"`
	}
	var r row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin queue item lookup — RLS bypass required.
		return tx.Raw(`SELECT id, target_type, target_id, target_family_id, flag_type, labels, created_at FROM safety_content_flags WHERE id = ?`, itemID).Scan(&r).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.GetReviewQueueItem: %w", err)
	}
	if r.ID == uuid.Nil {
		return nil, nil
	}
	familyID := uuid.Nil
	if r.TargetFamilyID != nil {
		familyID = *r.TargetFamilyID
	}
	return &admin.ModerationQueueItem{
		ID:          r.ID,
		ContentType: r.TargetType,
		ContentID:   r.TargetID,
		FamilyID:    familyID,
		Reason:      r.FlagType,
		Status:      "pending",
		Details:     r.Labels,
		CreatedAt:   r.CreatedAt,
	}, nil
}

func (a *adminSafetyAdapter) TakeModerationAction(ctx context.Context, itemID uuid.UUID, action string, reason string) error {
	// Look up the target family from the content flag, then delegate to safety service.
	var targetFamilyID uuid.UUID
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin action — RLS bypass required to read across families.
		return tx.Raw(`SELECT COALESCE(target_family_id, '00000000-0000-0000-0000-000000000000'::uuid) FROM safety_content_flags WHERE id = ?`, itemID).Scan(&targetFamilyID).Error
	})
	if err != nil {
		return fmt.Errorf("admin.TakeModerationAction: lookup target: %w", err)
	}
	systemAuth := &shared.AuthContext{ParentID: uuid.Nil, FamilyID: uuid.Nil, IsPlatformAdmin: true}
	_, err = a.svc.AdminTakeAction(ctx, systemAuth, safety.CreateModActionCommand{
		TargetFamilyID: targetFamilyID,
		ActionType:     action,
		Reason:         reason,
	})
	return err
}

// ─── adminBillingAdapter ─────────────────────────────────────────────────────

type adminBillingAdapter struct{ svc billing.BillingService }

func (a *adminBillingAdapter) GetSubscriptionInfo(ctx context.Context, familyID uuid.UUID) (*admin.AdminSubscriptionInfo, error) {
	scope := shared.NewFamilyScopeFromID(familyID)
	sub, err := a.svc.GetSubscription(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("admin.GetSubscriptionInfo: %w", err)
	}
	status := "active"
	if sub.Status != nil {
		status = *sub.Status
	}
	return &admin.AdminSubscriptionInfo{
		Tier:      sub.Tier,
		Status:    status,
		ExpiresAt: sub.CurrentPeriodEnd,
	}, nil
}

// ─── adminMethodAdapter ──────────────────────────────────────────────────────

type adminMethodAdapter struct{ db *gorm.DB }

func (a *adminMethodAdapter) ListMethodologies(ctx context.Context) ([]admin.MethodologyConfig, error) {
	type row struct {
		Slug        string          `gorm:"column:slug"`
		DisplayName string          `gorm:"column:display_name"`
		IsActive    bool            `gorm:"column:is_active"`
		Philosophy  json.RawMessage `gorm:"column:philosophy"`
		UpdatedAt   time.Time       `gorm:"column:updated_at"`
	}
	var rows []row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// method_definitions has no RLS; bypass used for consistency with other admin queries.
		return tx.Raw(`SELECT slug, display_name, is_active, philosophy, updated_at FROM method_definitions ORDER BY display_order ASC`).Scan(&rows).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.ListMethodologies: %w", err)
	}
	result := make([]admin.MethodologyConfig, len(rows))
	for i, r := range rows {
		result[i] = admin.MethodologyConfig{
			Slug:        r.Slug,
			DisplayName: r.DisplayName,
			Enabled:     r.IsActive,
			Settings:    r.Philosophy,
			UpdatedAt:   r.UpdatedAt,
		}
	}
	return result, nil
}

func (a *adminMethodAdapter) UpdateMethodologyConfig(ctx context.Context, slug string, input *admin.UpdateMethodologyInput) (*admin.MethodologyConfig, error) {
	if input.Enabled == nil && input.Settings == nil {
		return nil, admin.ErrMethodologyNotFound // nothing to update
	}
	type row struct {
		Slug        string          `gorm:"column:slug"`
		DisplayName string          `gorm:"column:display_name"`
		IsActive    bool            `gorm:"column:is_active"`
		Philosophy  json.RawMessage `gorm:"column:philosophy"`
		UpdatedAt   time.Time       `gorm:"column:updated_at"`
	}
	var r row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin methodology update — bypass for consistency.
		updates := map[string]any{"updated_at": time.Now()}
		if input.Enabled != nil {
			updates["is_active"] = *input.Enabled
		}
		if input.Settings != nil {
			updates["philosophy"] = *input.Settings
		}
		if err := tx.Table("method_definitions").Where("slug = ?", slug).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Raw(`SELECT slug, display_name, is_active, philosophy, updated_at FROM method_definitions WHERE slug = ?`, slug).Scan(&r).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.UpdateMethodologyConfig: %w", err)
	}
	if r.Slug == "" {
		return nil, admin.ErrMethodologyNotFound
	}
	return &admin.MethodologyConfig{
		Slug:        r.Slug,
		DisplayName: r.DisplayName,
		Enabled:     r.IsActive,
		Settings:    r.Philosophy,
		UpdatedAt:   r.UpdatedAt,
	}, nil
}

// ─── adminLifecycleAdapter ───────────────────────────────────────────────────

type adminLifecycleAdapter struct{ db *gorm.DB }

func (a *adminLifecycleAdapter) GetPendingDeletions(ctx context.Context, p *shared.PaginationParams) ([]admin.DeletionSummary, error) {
	type row struct {
		FamilyID    uuid.UUID `gorm:"column:family_id"`
		FamilyName  string    `gorm:"column:family_name"`
		RequestedAt time.Time `gorm:"column:requested_at"`
		ScheduledAt time.Time `gorm:"column:scheduled_at"`
	}
	limit := p.EffectiveLimit()
	var rows []row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin lifecycle view — RLS bypass required (cross-family).
		return tx.Raw(`
			SELECT d.family_id, f.display_name AS family_name, d.created_at AS requested_at, d.grace_period_ends_at AS scheduled_at
			FROM lifecycle_deletion_requests d
			JOIN iam_families f ON f.id = d.family_id
			WHERE d.status IN ('grace_period', 'processing')
			ORDER BY d.grace_period_ends_at ASC
			LIMIT ?`, limit).Scan(&rows).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.GetPendingDeletions: %w", err)
	}
	result := make([]admin.DeletionSummary, len(rows))
	for i, r := range rows {
		result[i] = admin.DeletionSummary{FamilyID: r.FamilyID, FamilyName: r.FamilyName, RequestedAt: r.RequestedAt, ScheduledAt: r.ScheduledAt}
	}
	return result, nil
}

func (a *adminLifecycleAdapter) GetRecoveryRequests(ctx context.Context, p *shared.PaginationParams) ([]admin.RecoverySummary, error) {
	type row struct {
		ID          uuid.UUID `gorm:"column:id"`
		Email       string    `gorm:"column:email"`
		RequestedAt time.Time `gorm:"column:requested_at"`
		Status      string    `gorm:"column:status"`
	}
	limit := p.EffectiveLimit()
	var rows []row
	err := shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin recovery view — no RLS on recovery table, bypass for consistency.
		return tx.Raw(`
			SELECT id, email, created_at AS requested_at, status
			FROM lifecycle_recovery_requests
			WHERE status IN ('pending', 'escalated')
			ORDER BY created_at ASC
			LIMIT ?`, limit).Scan(&rows).Error
	})
	if err != nil {
		return nil, fmt.Errorf("admin.GetRecoveryRequests: %w", err)
	}
	result := make([]admin.RecoverySummary, len(rows))
	for i, r := range rows {
		result[i] = admin.RecoverySummary{
			ID:          r.ID,
			FamilyName:  r.Email, // email is used as identifier (no family yet)
			RequestedAt: r.RequestedAt,
			Reason:      r.Status,
		}
	}
	return result, nil
}

func (a *adminLifecycleAdapter) ResolveRecoveryRequest(ctx context.Context, requestID uuid.UUID, approved bool) error {
	status := "denied"
	if approved {
		status = "completed"
	}
	return shared.BypassRLSTransaction(ctx, a.db, func(tx *gorm.DB) error {
		// Admin recovery resolution — RLS bypass for consistency.
		return tx.Exec(`UPDATE lifecycle_recovery_requests SET status = ?, resolved_at = now() WHERE id = ?`, status, requestID).Error
	})
}

// appHealthChecker implements admin.HealthChecker by probing each critical dependency. [16-admin §11.1]
type appHealthChecker struct {
	db                    *gorm.DB
	cache                 shared.Cache
	kratosPublicURL       string
	objectStorageEndpoint string
}

func (h *appHealthChecker) CheckAll(ctx context.Context) []admin.ComponentHealth {
	results := make([]admin.ComponentHealth, 0, 4)
	results = append(results, pingDatabase(ctx, h.db))
	results = append(results, pingCache(ctx, h.cache))
	results = append(results, pingKratos(ctx, h.kratosPublicURL))
	if h.objectStorageEndpoint != "" {
		results = append(results, pingObjectStorage(ctx, h.objectStorageEndpoint))
	}
	return results
}

func pingDatabase(ctx context.Context, db *gorm.DB) admin.ComponentHealth {
	start := time.Now()
	err := db.WithContext(ctx).Exec("SELECT 1").Error
	latency := time.Since(start).Milliseconds()
	if err != nil {
		msg := err.Error()
		return admin.ComponentHealth{Name: "database", Status: "unhealthy", LatencyMs: &latency, Details: &msg}
	}
	return admin.ComponentHealth{Name: "database", Status: "healthy", LatencyMs: &latency}
}

func pingCache(ctx context.Context, c shared.Cache) admin.ComponentHealth {
	start := time.Now()
	_, err := c.Get(ctx, "__hc__")
	latency := time.Since(start).Milliseconds()
	if err != nil {
		msg := err.Error()
		return admin.ComponentHealth{Name: "redis", Status: "unhealthy", LatencyMs: &latency, Details: &msg}
	}
	return admin.ComponentHealth{Name: "redis", Status: "healthy", LatencyMs: &latency}
}

func pingKratos(ctx context.Context, baseURL string) admin.ComponentHealth {
	start := time.Now()
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, baseURL+"/health/ready", nil)
	if err != nil {
		msg := err.Error()
		latency := time.Since(start).Milliseconds()
		return admin.ComponentHealth{Name: "kratos", Status: "unhealthy", LatencyMs: &latency, Details: &msg}
	}
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		msg := err.Error()
		return admin.ComponentHealth{Name: "kratos", Status: "unhealthy", LatencyMs: &latency, Details: &msg}
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return admin.ComponentHealth{Name: "kratos", Status: "healthy", LatencyMs: &latency}
	}
	msg := fmt.Sprintf("unexpected status %d", resp.StatusCode)
	return admin.ComponentHealth{Name: "kratos", Status: "degraded", LatencyMs: &latency, Details: &msg}
}

func pingObjectStorage(ctx context.Context, endpoint string) admin.ComponentHealth {
	start := time.Now()
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, endpoint, nil)
	if err != nil {
		msg := err.Error()
		latency := time.Since(start).Milliseconds()
		return admin.ComponentHealth{Name: "r2", Status: "unhealthy", LatencyMs: &latency, Details: &msg}
	}
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		msg := err.Error()
		return admin.ComponentHealth{Name: "r2", Status: "unhealthy", LatencyMs: &latency, Details: &msg}
	}
	_ = resp.Body.Close()
	// Any HTTP response means the endpoint is reachable (even 4xx/5xx from auth/routing)
	return admin.ComponentHealth{Name: "r2", Status: "healthy", LatencyMs: &latency}
}

// adminJobInspectorAdapter bridges admin.QueueInspector → admin.JobInspector. [16-admin §11.2]
type adminJobInspectorAdapter struct {
	inspector *admin.AsynqQueueInspector
}

func (a *adminJobInspectorAdapter) GetQueueStatus(ctx context.Context) (*admin.JobStatusResponse, error) {
	if a.inspector == nil {
		return &admin.JobStatusResponse{Queues: []admin.QueueStatus{}}, nil
	}
	queues, deadCount, err := a.inspector.GetQueues(ctx)
	if err != nil {
		return nil, err
	}
	result := &admin.JobStatusResponse{
		DeadLetterCount: deadCount,
		Queues:          make([]admin.QueueStatus, len(queues)),
	}
	for i, q := range queues {
		result.Queues[i] = admin.QueueStatus(q)
	}
	return result, nil
}

func (a *adminJobInspectorAdapter) GetDeadLetterJobs(ctx context.Context, pagination *shared.PaginationParams) ([]admin.DeadLetterJob, error) {
	if a.inspector == nil {
		return []admin.DeadLetterJob{}, nil
	}
	offset := 0
	limit := 20
	if pagination != nil {
		limit = pagination.EffectiveLimit()
		// Dead-letter jobs use offset-based pagination; cursor encodes the offset.
		if pagination.Cursor != nil {
			if parsed, parseErr := strconv.Atoi(*pagination.Cursor); parseErr == nil && parsed >= 0 {
				offset = parsed
			}
		}
	}
	jobs, err := a.inspector.GetDeadLetterJobs(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	result := make([]admin.DeadLetterJob, len(jobs))
	for i, j := range jobs {
		result[i] = admin.DeadLetterJob(j)
	}
	return result, nil
}

func (a *adminJobInspectorAdapter) RetryDeadLetterJob(ctx context.Context, jobID string) error {
	if a.inspector == nil {
		return admin.ErrDeadLetterNotFound
	}
	// jobID format is "queue:id"; parse to find the correct queue.
	queue := "default"
	id := jobID
	if idx := strings.LastIndex(jobID, ":"); idx > 0 {
		queue = jobID[:idx]
		id = jobID[idx+1:]
	}
	return a.inspector.RetryDeadLetterJob(ctx, queue, id)
}

// ─── Lifecycle Export Handler Adapters ────────────────────────────────────────
// Each adapter bridges a domain service to the lifecycle.ExportHandler interface,
// serializing family data as JSON files for GDPR Art. 20 data portability. [15-data-lifecycle §7]

type lifecycleLearnExport struct{ svc learn.LearningService }

func (a *lifecycleLearnExport) DomainName() string { return "learning" }
func (a *lifecycleLearnExport) ExportFamilyData(ctx context.Context, familyID uuid.UUID, _ lifecycle.ExportFormat) ([]lifecycle.ExportFile, error) {
	scope := shared.NewFamilyScopeFromID(familyID)
	lists, _ := a.svc.ListReadingLists(ctx, &scope)
	export := struct {
		ReadingLists any `json:"reading_lists"`
	}{ReadingLists: lists}
	data, err := json.Marshal(export)
	if err != nil {
		return nil, fmt.Errorf("learn: marshal export: %w", err)
	}
	return []lifecycle.ExportFile{{Filename: "learning.json", Content: data}}, nil
}

type lifecycleSocialExport struct{ svc social.SocialService }

func (a *lifecycleSocialExport) DomainName() string { return "social" }
func (a *lifecycleSocialExport) ExportFamilyData(ctx context.Context, familyID uuid.UUID, _ lifecycle.ExportFormat) ([]lifecycle.ExportFile, error) {
	scope := shared.NewFamilyScopeFromID(familyID)
	friends, _ := a.svc.ListFriends(ctx, &scope, nil, 1000)
	groups, _ := a.svc.ListMyGroups(ctx, &scope)
	export := struct {
		Friends any `json:"friends"`
		Groups  any `json:"groups"`
	}{Friends: friends, Groups: groups}
	data, err := json.Marshal(export)
	if err != nil {
		return nil, fmt.Errorf("social: marshal export: %w", err)
	}
	return []lifecycle.ExportFile{{Filename: "social.json", Content: data}}, nil
}

type lifecycleMktExport struct{ svc mkt.MarketplaceService }

func (a *lifecycleMktExport) DomainName() string { return "marketplace" }
func (a *lifecycleMktExport) ExportFamilyData(ctx context.Context, familyID uuid.UUID, _ lifecycle.ExportFormat) ([]lifecycle.ExportFile, error) {
	scope := shared.NewFamilyScopeFromID(familyID)
	cart, _ := a.svc.GetCart(ctx, scope)
	purchases, _ := a.svc.GetPurchases(ctx, scope, mkt.PurchaseQueryParams{})
	export := struct {
		Cart      any `json:"cart"`
		Purchases any `json:"purchases"`
	}{Cart: cart, Purchases: purchases}
	data, err := json.Marshal(export)
	if err != nil {
		return nil, fmt.Errorf("mkt: marshal export: %w", err)
	}
	return []lifecycle.ExportFile{{Filename: "marketplace.json", Content: data}}, nil
}

type lifecycleMediaExport struct{ svc media.MediaService }

func (a *lifecycleMediaExport) DomainName() string { return "media" }
func (a *lifecycleMediaExport) ExportFamilyData(ctx context.Context, familyID uuid.UUID, _ lifecycle.ExportFormat) ([]lifecycle.ExportFile, error) {
	uploads, _ := a.svc.ListUploads(ctx, familyID, 10000, nil)
	data, err := json.Marshal(uploads)
	if err != nil {
		return nil, fmt.Errorf("media: marshal export: %w", err)
	}
	return []lifecycle.ExportFile{{Filename: "media.json", Content: data}}, nil
}

type lifecycleNotifyExport struct{ svc notify.NotificationService }

func (a *lifecycleNotifyExport) DomainName() string { return "notifications" }
func (a *lifecycleNotifyExport) ExportFamilyData(ctx context.Context, familyID uuid.UUID, _ lifecycle.ExportFormat) ([]lifecycle.ExportFile, error) {
	scope := shared.NewFamilyScopeFromID(familyID)
	prefs, _ := a.svc.GetPreferences(ctx, &scope)
	notifications, _ := a.svc.ListNotifications(ctx, notify.NotificationListParams{}, &scope)
	export := struct {
		Preferences   any `json:"preferences"`
		Notifications any `json:"notifications"`
	}{Preferences: prefs, Notifications: notifications}
	data, err := json.Marshal(export)
	if err != nil {
		return nil, fmt.Errorf("notify: marshal export: %w", err)
	}
	return []lifecycle.ExportFile{{Filename: "notifications.json", Content: data}}, nil
}

type lifecycleBillingExport struct{ svc billing.BillingService }

func (a *lifecycleBillingExport) DomainName() string { return "billing" }
func (a *lifecycleBillingExport) ExportFamilyData(ctx context.Context, familyID uuid.UUID, _ lifecycle.ExportFormat) ([]lifecycle.ExportFile, error) {
	scope := shared.NewFamilyScopeFromID(familyID)
	sub, _ := a.svc.GetSubscription(ctx, scope)
	txns, _ := a.svc.ListTransactions(ctx, billing.TransactionListParams{}, scope)
	export := struct {
		Subscription any `json:"subscription"`
		Transactions any `json:"transactions"`
	}{Subscription: sub, Transactions: txns}
	data, err := json.Marshal(export)
	if err != nil {
		return nil, fmt.Errorf("billing: marshal export: %w", err)
	}
	return []lifecycle.ExportFile{{Filename: "billing.json", Content: data}}, nil
}

type lifecycleComplyExport struct{ svc comply.ComplianceService }

func (a *lifecycleComplyExport) DomainName() string { return "compliance" }
func (a *lifecycleComplyExport) ExportFamilyData(ctx context.Context, familyID uuid.UUID, _ lifecycle.ExportFormat) ([]lifecycle.ExportFile, error) {
	scope := shared.NewFamilyScopeFromID(familyID)
	cfg, _ := a.svc.GetFamilyConfig(ctx, scope)
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("comply: marshal export: %w", err)
	}
	return []lifecycle.ExportFile{{Filename: "compliance.json", Content: data}}, nil
}

type lifecycleRecsExport struct{ svc recs.RecsService }

func (a *lifecycleRecsExport) DomainName() string { return "recommendations" }
func (a *lifecycleRecsExport) ExportFamilyData(ctx context.Context, familyID uuid.UUID, _ lifecycle.ExportFormat) ([]lifecycle.ExportFile, error) {
	scope := shared.NewFamilyScopeFromID(familyID)
	prefs, _ := a.svc.GetPreferences(ctx, &scope)
	data, err := json.Marshal(prefs)
	if err != nil {
		return nil, fmt.Errorf("recs: marshal export: %w", err)
	}
	return []lifecycle.ExportFile{{Filename: "recommendations.json", Content: data}}, nil
}

type lifecyclePlanExport struct{ svc plan.PlanningService }

func (a *lifecyclePlanExport) DomainName() string { return "planning" }
func (a *lifecyclePlanExport) ExportFamilyData(ctx context.Context, familyID uuid.UUID, _ lifecycle.ExportFormat) ([]lifecycle.ExportFile, error) {
	scope := shared.NewFamilyScopeFromID(familyID)
	data, err := a.svc.ExportData(ctx, &scope)
	if err != nil {
		return nil, fmt.Errorf("plan: export: %w", err)
	}
	return []lifecycle.ExportFile{{Filename: "planning.json", Content: data}}, nil
}

// ─── Lifecycle Deletion Handler Adapters ─────────────────────────────────────
// Each adapter bridges a domain service to the lifecycle.DeletionHandler interface,
// enabling coordinated family/student data deletion. [15-data-lifecycle §7]

type lifecycleLearnDeletion struct{ svc learn.LearningService }

func (a *lifecycleLearnDeletion) DomainName() string { return "learning" }
func (a *lifecycleLearnDeletion) DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error {
	return a.svc.HandleFamilyDeletionScheduled(ctx, familyID)
}
func (a *lifecycleLearnDeletion) DeleteStudentData(ctx context.Context, familyID uuid.UUID, studentID uuid.UUID) error {
	return a.svc.HandleStudentDeleted(ctx, familyID, studentID)
}

type lifecycleSocialDeletion struct{ svc social.SocialService }

func (a *lifecycleSocialDeletion) DomainName() string { return "social" }
func (a *lifecycleSocialDeletion) DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error {
	return a.svc.HandleFamilyDeletionScheduled(ctx, familyID)
}
func (a *lifecycleSocialDeletion) DeleteStudentData(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil // social has no student-specific data
}

type lifecycleMktDeletion struct{ svc mkt.MarketplaceService }

func (a *lifecycleMktDeletion) DomainName() string { return "marketplace" }
func (a *lifecycleMktDeletion) DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error {
	return a.svc.HandleFamilyDeletionScheduled(ctx, familyID)
}
func (a *lifecycleMktDeletion) DeleteStudentData(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil // marketplace has no student-specific data
}

type lifecycleNotifyDeletion struct{ svc notify.NotificationService }

func (a *lifecycleNotifyDeletion) DomainName() string { return "notifications" }
func (a *lifecycleNotifyDeletion) DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error {
	return a.svc.HandleFamilyDeletionScheduled(ctx, notify.FamilyDeletionScheduledEvent{FamilyID: familyID})
}
func (a *lifecycleNotifyDeletion) DeleteStudentData(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil // notifications has no student-specific data
}

type lifecycleBillingDeletion struct{ svc billing.BillingService }

func (a *lifecycleBillingDeletion) DomainName() string { return "billing" }
func (a *lifecycleBillingDeletion) DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error {
	return a.svc.HandleFamilyDeletionScheduled(ctx, billing.FamilyDeletionScheduledEvent{FamilyID: familyID})
}
func (a *lifecycleBillingDeletion) DeleteStudentData(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil // billing has no student-specific data
}

type lifecycleComplyDeletion struct{ svc comply.ComplianceService }

func (a *lifecycleComplyDeletion) DomainName() string { return "compliance" }
func (a *lifecycleComplyDeletion) DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error {
	return a.svc.HandleFamilyDeletionScheduled(ctx, &comply.FamilyDeletionScheduledEvent{FamilyID: familyID})
}
func (a *lifecycleComplyDeletion) DeleteStudentData(ctx context.Context, familyID uuid.UUID, studentID uuid.UUID) error {
	return a.svc.HandleStudentDeleted(ctx, &comply.StudentDeletedEvent{StudentID: studentID, FamilyID: familyID})
}

type lifecycleRecsDeletion struct{ svc recs.RecsService }

func (a *lifecycleRecsDeletion) DomainName() string { return "recommendations" }
func (a *lifecycleRecsDeletion) DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error {
	return a.svc.HandleFamilyDeletion(ctx, shared.NewFamilyID(familyID))
}
func (a *lifecycleRecsDeletion) DeleteStudentData(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil // recommendations are family-level, not student-specific
}

type lifecycleMediaDeletion struct{ svc media.MediaService }

func (a *lifecycleMediaDeletion) DomainName() string { return "media" }
func (a *lifecycleMediaDeletion) DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error {
	return a.svc.HandleFamilyDeletionScheduled(ctx, familyID)
}
func (a *lifecycleMediaDeletion) DeleteStudentData(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil // media uploads are family-scoped, not student-specific
}

// lifecyclePlanDeletion bridges plan:: into lifecycle deletion.
// The svc field is set after plan:: is wired (plan:: depends on comply:: and social::,
// which are wired after lifecycle::). The adapter is only called at deletion time,
// well after all services are fully wired.
type lifecyclePlanDeletion struct{ svc plan.PlanningService }

func (a *lifecyclePlanDeletion) DomainName() string { return "planning" }
func (a *lifecyclePlanDeletion) DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error {
	scope := shared.NewFamilyScopeFromID(familyID)
	return a.svc.DeleteData(ctx, &scope)
}
func (a *lifecyclePlanDeletion) DeleteStudentData(ctx context.Context, familyID uuid.UUID, studentID uuid.UUID) error {
	scope := shared.NewFamilyScopeFromID(familyID)
	return a.svc.DeleteStudentData(ctx, &scope, studentID)
}

// ─── Plan Domain Cross-Domain Adapters ──────────────────────────────────────
// Bridge comply:: and social:: into the plan domain's consumer-defined interfaces.
// [17-planning §8]

type planComplyAdapter struct {
	complySvc comply.ComplianceService
	iamSvc    iam.IamService
}

func (a *planComplyAdapter) GetAttendanceRange(ctx context.Context, _ *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]plan.AttendanceSummary, error) {
	params := comply.AttendanceListParams{
		StartDate: start,
		EndDate:   end,
	}

	var studentIDs []uuid.UUID
	if studentID != nil {
		studentIDs = []uuid.UUID{*studentID}
	} else {
		students, err := a.iamSvc.ListStudents(ctx, scope)
		if err != nil {
			return nil, fmt.Errorf("plan comply adapter: list students: %w", err)
		}
		for _, s := range students {
			studentIDs = append(studentIDs, s.ID)
		}
	}

	var results []plan.AttendanceSummary
	for _, sid := range studentIDs {
		resp, err := a.complySvc.ListAttendance(ctx, sid, params, *scope)
		if err != nil {
			continue // skip students with no attendance data
		}
		for _, r := range resp.Records {
			results = append(results, plan.AttendanceSummary{
				ID:        r.ID,
				Date:      r.AttendanceDate,
				StudentID: &sid,
				Status:    r.Status,
			})
		}
	}
	return results, nil
}

type planSocialAdapter struct {
	socialSvc social.SocialService
}

func (a *planSocialAdapter) GetEventsForCalendar(ctx context.Context, auth *shared.AuthContext, _ *shared.FamilyScope, start, end time.Time) ([]plan.EventSummary, error) {
	events, err := a.socialSvc.ListEventsForDateRange(ctx, auth, start, end)
	if err != nil {
		return nil, fmt.Errorf("plan social adapter: list events: %w", err)
	}
	results := make([]plan.EventSummary, 0, len(events))
	for _, e := range events {
		results = append(results, plan.EventSummary{
			ID:        e.ID,
			Title:     e.Title,
			Date:      e.EventDate,
			Location:  e.LocationName,
			RSVPStatus: e.MyRSVP,
		})
	}
	return results, nil
}

// gracefulShutdown listens for SIGINT/SIGTERM and shuts the server down cleanly.
// Waits up to 30 seconds for in-flight requests to complete. [§4.3]
func gracefulShutdown(ctx context.Context, e *echo.Echo, cleanup func()) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	slog.Info("received shutdown signal", "signal", sig.String())

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	cleanup()
}

// initLogger configures the global slog logger based on environment and config. [§14.1]
// JSON format in production (CloudWatch); text format in development.
func initLogger(cfg *config.AppConfig) {
	level := parseLogLevel(cfg.LogLevel)

	var handler slog.Handler
	if cfg.Environment == config.EnvironmentProduction {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	slog.SetDefault(slog.New(handler))
}

// ─── iamBillingAdapter ───────────────────────────────────────────────────────

// iamBillingAdapter bridges billing.BillingService → iam.BillingServiceForIam.
// Satisfies the consumer-defined interface pattern [ARCH §4.3].
type iamBillingAdapter struct{ svc billing.BillingService }

func (a iamBillingAdapter) VerifyCreditCardMicroCharge(ctx context.Context, scope *shared.FamilyScope, paymentMethodID string) error {
	_, err := a.svc.ProcessCoppaVerification(ctx, billing.CoppaVerificationCommand{
		PaymentMethodID: paymentMethodID,
	}, *scope)
	return err
}

// parseLogLevel converts a string log level to slog.Level.
func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
