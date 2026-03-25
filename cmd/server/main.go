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
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pressly/goose/v3"

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
	"github.com/homegrown-academy/homegrown-academy/internal/onboard"
	"github.com/homegrown-academy/homegrown-academy/internal/recs"
	"github.com/homegrown-academy/homegrown-academy/internal/safety"
	"github.com/homegrown-academy/homegrown-academy/internal/search"
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

	iamSvc := iam.NewIamService(familyRepo, parentRepo, studentRepo, kratosAdapter, eventBus, db)

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
	discoverSvc := discover.NewDiscoveryService(quizDefRepo, quizResRepo, discStateRepo, discoverMethod)

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
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(iam.CoParentRemoved{}), social.NewCoParentRemovedHandler(socialSvc))
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(learn.MilestoneAchieved{}), social.NewMilestoneAchievedHandler(socialSvc))
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), social.NewFamilyDeletionScheduledHandler(socialSvc))

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
		eventBus, jobs, mediaCfg,
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
	// Handler structs exist; subscriptions deferred until source event types are defined:
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(safety.ContentFlagged{}), mkt.NewContentFlaggedHandler(mktSvc))
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), mkt.NewFamilyDeletionScheduledHandler(mktSvc))

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
		// GetStudentName — construct FamilyScope via RLS bypass to look up display name
		func(ctx context.Context, studentID uuid.UUID) (string, error) {
			var displayName string
			err := shared.BypassRLSTransaction(ctx, db, func(tx *gorm.DB) error {
				return tx.Table("iam_students").Select("display_name").Where("id = ?", studentID).Scan(&displayName).Error
			})
			if err != nil || displayName == "" {
				return studentID.String(), nil // graceful fallback
			}
			return displayName, nil
		},
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
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), learn.NewFamilyDeletionScheduledHandler(learnSvc))
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(mkt.PurchaseCompleted{}), learn.NewPurchaseCompletedHandler(learnSvc))

	// ── Step 7h: Wire notify:: domain ───────────────────────────────────────────
	// notify:: is the cross-cutting event-driven dispatch system. It receives domain
	// events from every other module and delivers in-app notifications, transactional
	// email, and streak detection. [08-notify §1]
	notifRepo := notify.NewPgNotificationRepository(db)
	notifPrefRepo := notify.NewPgPreferenceRepository(db)
	notifDigestRepo := notify.NewPgDigestRepository(db)

	// Email adapter: use Postmark in production, noop in dev/test.
	var emailAdapter notify.EmailAdapter = notifyadapters.NoopEmailAdapter{}
	// TODO: Wire PostmarkEmailAdapter when cfg.PostmarkServerToken != ""

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
	// DEFERRED: safety::ContentFlagged — safety:: domain not implemented
	// DEFERRED: iam::CoParentAdded, iam::FamilyDeletionScheduled — events not defined yet

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

	// Register billing:: event subscriptions
	eventBus.Subscribe(reflect.TypeOf(mkt.PurchaseCompleted{}), billing.NewPurchaseCompletedHandler(billingSvc))
	eventBus.Subscribe(reflect.TypeOf(mkt.PurchaseRefunded{}), billing.NewPurchaseRefundedHandler(billingSvc))
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), billing.NewFamilyDeletionScheduledHandler(billingSvc))
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(iam.PrimaryParentTransferred{}), billing.NewPrimaryParentTransferredHandler(billingSvc))

	// Now that billing:: is wired, register billing event handlers in notify::
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(billing.SubscriptionCreated{}), notify.NewSubscriptionCreatedHandler(notifySvc))
	// DEFERRED: eventBus.Subscribe(reflect.TypeOf(billing.SubscriptionCancelled{}), notify.NewSubscriptionCancelledHandler(notifySvc))

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

	// IAM adapter for safety: bridges iam:: → safety::IamServiceForSafety
	var iamForSafety safety.IamServiceForSafety = safety.NoopIamServiceForSafety{}
	// FUTURE: wire real IamServiceForSafety when iam:: exposes RevokeSessions

	safetyCfg := safety.DefaultSafetyConfig()
	textScanner := safety.NewTextScanner(safetyCfg)

	safetySvc := safety.NewSafetyService(
		safetyReportRepo, safetyFlagRepo, safetyActionRepo, safetyAccountRepo,
		safetyAppealRepo, safetyNcmecRepo, safetyBotRepo,
		iamForSafety, cache, eventBus, jobs, textScanner, safetyCfg,
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

	searchSvc := search.NewSearchService(searchSocialRepo, searchMktRepo, searchLearnRepo, searchAutoRepo)

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
		PubSub:      pubsub,
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
	notify.RegisterTaskHandlers(worker, emailAdapter)
	safety.RegisterSafetyWorkers(worker, safetyNcmecRepo, safety.NoopThornAdapter{}, jobs)
	recs.RegisterTaskHandlers(worker, db,
		recSignalRepo, recRecRepo, recFeedbackRepo,
		recPopularityRepo, recPrefRepo, recAnonRepo,
		cfg.RecsAnonymizationSecret,
	)
	go func() {
		if startErr := worker.Start(); startErr != nil {
			slog.Error("job worker error", "error", startErr)
		}
	}()

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
