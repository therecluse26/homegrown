// @title Homegrown Academy API
// @version 0.1.0
// @description API for the Homegrown Academy homeschooling platform
// @host localhost:3500
// @BasePath /v1
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pressly/goose/v3"

	"github.com/homegrown-academy/homegrown-academy/internal/app"
	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"github.com/homegrown-academy/homegrown-academy/internal/discover"
	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	iamadapters "github.com/homegrown-academy/homegrown-academy/internal/iam/adapters"
	"github.com/homegrown-academy/homegrown-academy/internal/method"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
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
	// Domain subscriptions are registered here as domains are built:
	// eventBus.Subscribe(reflect.TypeOf(iam.FamilyCreated{}), social.NewOnFamilyCreatedHandler(socialSvc))
	// eventBus.Subscribe(reflect.TypeOf(iam.FamilyCreated{}), onboard.NewOnFamilyCreatedHandler(onboardSvc))

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
	}

	// ── Step 8: Build Echo router ─────────────────────────────────────────────────
	e := app.NewApp(state)

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
		if closeErr := jobs.Close(); closeErr != nil {
			slog.Error("job enqueuer close error", "error", closeErr)
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
