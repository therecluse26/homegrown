package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/homegrown-academy/homegrown-academy/internal/admin"
	"github.com/homegrown-academy/homegrown-academy/internal/billing"
	"github.com/homegrown-academy/homegrown-academy/internal/comply"
	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"github.com/homegrown-academy/homegrown-academy/internal/discover"
	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/learn"
	"github.com/homegrown-academy/homegrown-academy/internal/lifecycle"
	"github.com/homegrown-academy/homegrown-academy/internal/media"
	"github.com/homegrown-academy/homegrown-academy/internal/method"
	"github.com/homegrown-academy/homegrown-academy/internal/middleware"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/homegrown-academy/homegrown-academy/internal/notify"
	"github.com/homegrown-academy/homegrown-academy/internal/onboard"
	"github.com/homegrown-academy/homegrown-academy/internal/plan"
	"github.com/homegrown-academy/homegrown-academy/internal/recs"
	"github.com/homegrown-academy/homegrown-academy/internal/safety"
	"github.com/homegrown-academy/homegrown-academy/internal/search"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

// AppState holds shared infrastructure and domain service interfaces.
// All third-party vendor types are hidden behind generic ports defined in shared/.
// All domain service fields must be non-nil; NewApp will panic if any are missing. [§5.1]
type AppState struct {
	// ─── Infrastructure ─────────────────────────────────────────────
	DB       *gorm.DB
	Cache    shared.Cache
	Auth     shared.SessionValidator
	Errors   shared.ErrorReporter
	Jobs     shared.JobEnqueuer // background job enqueuer (asynq-backed)
	EventBus *shared.EventBus
	Config   *config.AppConfig
	Version  string // Set via -ldflags at build time

	// ─── Domain Services ────────────────────────────────────────────
	IAM         iam.IamService
	Method      method.MethodologyService
	Discover    discover.DiscoveryService
	Onboard     onboard.OnboardingService
	Social      social.SocialService
	Learn       learn.LearningService
	Marketplace mkt.MarketplaceService
	Media       media.MediaService
	Notify      notify.NotificationService
	Billing     billing.BillingService
	Safety      safety.SafetyService
	Search      search.SearchService
	Recs        recs.RecsService
	Comply      comply.ComplianceService
	Lifecycle   lifecycle.LifecycleService
	Admin       admin.AdminService
	Plan        plan.PlanningService
	PubSub      shared.PubSub // needed by social handler for WebSocket
}

// ─── authDeps and rateLimitDeps interface satisfaction ──────────────────────
// AppState satisfies the unexported interfaces defined in internal/middleware/.
// This avoids a circular import (middleware cannot import app).

// GetAuthValidator satisfies middleware.authDeps.
func (s *AppState) GetAuthValidator() shared.SessionValidator {
	return s.Auth
}

// GetCache satisfies middleware.rateLimitDeps.
func (s *AppState) GetCache() shared.Cache {
	return s.Cache
}

// GetDB satisfies any future middleware interface requiring database access.
func (s *AppState) GetDB() *gorm.DB {
	return s.DB
}

// GetConfig returns the application config (for middleware and other consumers).
func (s *AppState) GetConfig() *config.AppConfig {
	return s.Config
}

// GetJobs returns the background job enqueuer (satisfies future middleware/domain deps interfaces).
func (s *AppState) GetJobs() shared.JobEnqueuer {
	return s.Jobs
}

// ─── Echo Validator ────────────────────────────────────────────────────────

// customValidator adapts go-playground/validator to Echo's Validator interface.
type customValidator struct {
	v *validator.Validate
}

func (cv *customValidator) Validate(i interface{}) error {
	return cv.v.Struct(i)
}

// ─── Router ───────────────────────────────────────────────────────────────────

// NewApp builds the Echo router with middleware layering and route groups.
// Middleware ordering is outermost-first (see §5.3 for the full ordering table).
func NewApp(state *AppState) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	e.Validator = &customValidator{v: validator.New()}

	// ─── Global Middleware (outermost applied first) ──────────────────
	// Order: Metrics → RequestLogger → SecurityHeaders → CORS → CSRF → (rate limit added per-group/route)
	e.Use(middleware.Metrics())
	e.Use(echomw.RequestLoggerWithConfig(requestLoggerConfig()))
	e.Use(middleware.SecurityHeaders())
	e.Use(echomw.CORSWithConfig(corsConfig(state.Config)))
	e.Use(middleware.CSRF()) // Double-submit cookie CSRF protection [CRIT-2]

	// ─── Public Routes ────────────────────────────────────────────────
	// GET /health — unauthenticated, used by ALB health checks and UptimeRobot. [§5.4]
	e.GET("/health", healthHandler(state))
	// GET /metrics — unauthenticated Prometheus scrape endpoint. [P2-1]
	// Exposes default Go runtime metrics and HTTP request count/duration histograms.
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// ─── Webhook Routes ───────────────────────────────────────────────
	// Domain webhooks are registered here with rate limiting (10 req/60s per IP).
	// Webhook secret validation is done per-domain in the handler middleware.
	hooks := e.Group("/hooks")
	hooks.Use(middleware.RateLimit(state, 10, 60*time.Second))

	// ─── Authenticated Routes ─────────────────────────────────────────
	// All domain API routes live under /v1 and require authentication.
	auth := e.Group("/v1")
	auth.Use(middleware.RateLimit(state, 100, 60*time.Second))
	auth.Use(middleware.Auth(state))

	// ─── Public API Routes ─────────────────────────────────────────────
	// Some domain routes are public (no auth required), e.g. methodology exploration.
	pub := e.Group("/v1")
	pub.Use(middleware.RateLimit(state, 100, 60*time.Second))

	// ─── Student Session Routes (tighter rate limit) ─────────────────
	// Student token validation is separate: bearer token auth, stricter rate limit. [P2-6]
	studentPub := e.Group("/v1")
	studentPub.Use(middleware.RateLimit(state, 10, 60*time.Second)) // 10 req/60s per IP

	// ─── Domain Route Registration ────────────────────────────────────
	iam.NewHandler(state.IAM, state.Config.AuthWebhookSecret).RegisterWithStudentSession(auth, hooks, studentPub)
	method.NewHandler(state.Method).Register(pub, auth)
	discover.NewHandler(state.Discover).Register(pub, auth)
	onboard.NewHandler(state.Onboard).Register(auth)
	social.NewHandler(state.Social, state.PubSub, state.Config.CORSAllowedOrigins, state.Auth).Register(auth)
	learn.NewHandler(state.Learn).Register(auth)
	media.NewHandler(state.Media).Register(auth)
	mkt.NewHandler(state.Marketplace, state.Cache).Register(auth, hooks, pub)
	notify.NewHandler(state.Notify, state.Config.UnsubscribeSecret).Register(auth, pub)
	billing.NewHandler(state.Billing, state.Config.BillingWebhookSecret, state.DB).Register(auth, hooks)
	// Shared admin group — used by both safety and admin domains.
	adminGroup := auth.Group("/admin")
	safety.NewHandler(state.Safety).Register(auth, adminGroup)
	search.NewHandler(state.Search, state).Register(auth)
	recs.NewHandler(state.Recs).Register(auth)
	comply.NewHandler(state.Comply).Register(auth)
	lifecycle.NewHandler(state.Lifecycle).Register(auth, pub)
	admin.NewHandler(state.Admin).Register(auth, adminGroup)
	plan.NewHandler(state.Plan).Register(auth)

	return e
}

// ─── Health Endpoint ─────────────────────────────────────────────────────────

// HealthResponse is the JSON response for GET /health.
type HealthResponse struct {
	Status  string            `json:"status"`
	Version string            `json:"version"`
	Checks  map[string]string `json:"checks,omitempty"` // per-dependency health status [P2-2]
}

// healthHandler returns the health check handler.
// Verifies DB, Redis, and Kratos connectivity. Returns 503 if any dependency is unhealthy. [P2-2]
//
// HealthCheck godoc
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /health [get]
func healthHandler(state *AppState) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		checks := make(map[string]string)
		healthy := true

		// Check database connectivity
		sqlDB, err := state.DB.WithContext(ctx).DB()
		if err != nil {
			checks["database"] = "error: " + err.Error()
			healthy = false
		} else if err := sqlDB.PingContext(ctx); err != nil {
			checks["database"] = "error: " + err.Error()
			healthy = false
		} else {
			checks["database"] = "ok"
		}

		// Check Redis connectivity
		if err := state.Cache.Ping(ctx); err != nil {
			checks["redis"] = "error: " + err.Error()
			healthy = false
		} else {
			checks["redis"] = "ok"
		}

		status := "ok"
		statusCode := http.StatusOK
		if !healthy {
			status = "degraded"
			statusCode = http.StatusServiceUnavailable
		}

		return c.JSON(statusCode, HealthResponse{
			Status:  status,
			Version: state.Version,
			Checks:  checks,
		})
	}
}

// ─── Middleware Config Helpers ───────────────────────────────────────────────

// requestLoggerConfig configures Echo's request logger to emit slog records.
// Logs: method, URI, status, latency. Never logs: IP, headers, bodies, PII. [CODING §5.2]
func requestLoggerConfig() echomw.RequestLoggerConfig {
	return echomw.RequestLoggerConfig{
		LogMethod:  true,
		LogURI:     true,
		LogStatus:  true,
		LogLatency: true,
		LogValuesFunc: func(_ echo.Context, v echomw.RequestLoggerValues) error {
			slog.Info("request",
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency", v.Latency,
			)
			return nil
		},
	}
}

// corsConfig builds the CORS middleware config from application config.
func corsConfig(cfg *config.AppConfig) echomw.CORSConfig {
	return echomw.CORSConfig{
		AllowOrigins: cfg.CORSAllowedOrigins,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true, // Required for auth provider session cookies
	}
}
