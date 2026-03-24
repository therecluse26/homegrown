package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"github.com/homegrown-academy/homegrown-academy/internal/discover"
	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/learn"
	"github.com/homegrown-academy/homegrown-academy/internal/media"
	"github.com/homegrown-academy/homegrown-academy/internal/method"
	"github.com/homegrown-academy/homegrown-academy/internal/middleware"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
	"github.com/homegrown-academy/homegrown-academy/internal/notify"
	"github.com/homegrown-academy/homegrown-academy/internal/onboard"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

// AppState holds shared infrastructure and domain service interfaces.
// All third-party vendor types are hidden behind generic ports defined in shared/.
// Domain service fields are uncommented as each domain is implemented. [§5.1]
type AppState struct {
	// ─── Infrastructure ─────────────────────────────────────────────
	DB       *gorm.DB
	Cache    shared.Cache
	Auth     shared.SessionValidator // nil until 01-iam wires KratosSessionValidator
	Errors   shared.ErrorReporter
	Jobs     shared.JobEnqueuer // background job enqueuer (asynq-backed)
	EventBus *shared.EventBus
	Config   *config.AppConfig
	Version  string // Set via -ldflags at build time

	// ─── Domain Services (added incrementally as domains are built) ─
	IAM      iam.IamService
	Method   method.MethodologyService
	Discover discover.DiscoveryService
	Onboard  onboard.OnboardingService
	Social      social.SocialService
	Learn       learn.LearningService
	Marketplace mkt.MarketplaceService
	Media       media.MediaService
	Notify      notify.NotificationService
	PubSub      shared.PubSub // needed by social handler for WebSocket
}

// ─── authDeps and rateLimitDeps interface satisfaction ──────────────────────
// AppState satisfies the unexported interfaces defined in internal/middleware/.
// This avoids a circular import (middleware cannot import app).

// GetAuthValidator satisfies middleware.authDeps.
// Returns nil until 01-iam wires a concrete KratosSessionValidator.
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
	// Order: RequestLogger → SecurityHeaders → CORS → (rate limit added per-group/route)
	e.Use(echomw.RequestLoggerWithConfig(requestLoggerConfig()))
	e.Use(middleware.SecurityHeaders())
	e.Use(echomw.CORSWithConfig(corsConfig(state.Config)))

	// ─── Public Routes ────────────────────────────────────────────────
	// GET /health — unauthenticated, used by ALB health checks and UptimeRobot. [§5.4]
	e.GET("/health", healthHandler(state))

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

	// ─── Domain Route Registration ────────────────────────────────────
	if state.IAM != nil {
		iam.NewHandler(state.IAM, state.Config.AuthWebhookSecret).Register(auth, hooks)
	}
	if state.Method != nil {
		method.NewHandler(state.Method).Register(pub, auth)
	}
	if state.Discover != nil {
		discover.NewHandler(state.Discover).Register(pub, auth)
	}
	if state.Onboard != nil {
		onboard.NewHandler(state.Onboard).Register(auth)
	}
	if state.Social != nil {
		social.NewHandler(state.Social, state.PubSub).Register(auth)
	}
	if state.Learn != nil {
		learn.NewHandler(state.Learn).Register(auth)
	}
	if state.Media != nil {
		media.NewHandler(state.Media).Register(auth)
	}
	if state.Marketplace != nil {
		mkt.NewHandler(state.Marketplace, state.Cache).Register(auth, hooks, pub)
	}
	if state.Notify != nil {
		notify.NewHandler(state.Notify, state.Config.UnsubscribeSecret).Register(auth, pub)
	}

	return e
}

// ─── Health Endpoint ─────────────────────────────────────────────────────────

// HealthResponse is the JSON response for GET /health.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// healthHandler returns the health check handler.
// No database connectivity check — DB is validated at startup. [§5.4]
//
// HealthCheck godoc
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func healthHandler(state *AppState) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, HealthResponse{
			Status:  "ok",
			Version: state.Version,
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
