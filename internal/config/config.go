package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Environment represents the runtime environment.
type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

// AppConfig holds all application configuration loaded from environment variables.
type AppConfig struct {
	// ─── Database ───────────────────────────────────────────────────
	// PostgreSQL connection string.
	// Example: "postgres://user:pass@localhost:5932/homegrown"
	DatabaseURL string

	// Maximum connections in the GORM pool. Default: 10.
	DatabaseMaxConnections int

	// ─── Redis ──────────────────────────────────────────────────────
	// Redis connection string.
	// Example: "redis://localhost:6879"
	RedisURL string

	// ─── Auth Provider ───────────────────────────────────────────────
	// Auth provider admin API URL (internal sidecar, never public).
	// Example: "http://kratos:4434"
	AuthAdminURL string

	// Auth provider public API URL (browser-facing, session validation).
	// Example: "http://kratos:4433"
	AuthPublicURL string

	// Shared secret for auth provider webhook signature validation.
	AuthWebhookSecret string

	// ─── CORS ───────────────────────────────────────────────────────
	// Comma-separated list of allowed origins.
	// Example: "http://localhost:5673,https://app.homegrown.academy"
	CORSAllowedOrigins []string

	// ─── Server ─────────────────────────────────────────────────────
	// Host to bind to. Default: "0.0.0.0".
	ServerHost string

	// Port to bind to. Default: 3500.
	ServerPort int

	// ─── Logging ────────────────────────────────────────────────────
	// slog log level. Default: "info".
	// Example: "debug"
	LogLevel string

	// ─── Observability ──────────────────────────────────────────────
	// Error reporting DSN. Optional — omit to disable error reporting (e.g. Sentry). [ARCH §2.14]
	ErrorReportingDSN *string

	// ─── Payments (Hyperswitch) ─────────────────────────────────────
	// Hyperswitch base URL. Optional — omit to disable payment processing.
	// Example: "http://hyperswitch:8080"
	HyperswitchBaseURL string

	// Hyperswitch API key. Required when HyperswitchBaseURL is set.
	HyperswitchAPIKey string

	// Hyperswitch webhook signing key. Required when HyperswitchBaseURL is set.
	HyperswitchWebhookKey string

	// ─── Environment ────────────────────────────────────────────────
	// Runtime environment. Controls log format, debug features, etc.
	Environment Environment
}

// LoadConfig loads configuration from environment variables.
// Loads .env file if present (dev only, not required in production).
func LoadConfig() (*AppConfig, error) {
	// Load .env file if it exists (dev only, not required)
	_ = godotenv.Load()

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	redisURL, err := requiredEnv("REDIS_URL")
	if err != nil {
		return nil, err
	}

	authAdminURL, err := requiredEnv("AUTH_ADMIN_URL")
	if err != nil {
		return nil, err
	}

	authPublicURL, err := requiredEnv("AUTH_PUBLIC_URL")
	if err != nil {
		return nil, err
	}

	authWebhookSecret, err := requiredEnv("AUTH_WEBHOOK_SECRET")
	if err != nil {
		return nil, err
	}

	corsOrigins, err := requiredEnv("CORS_ALLOWED_ORIGINS")
	if err != nil {
		return nil, err
	}

	maxConns := 10
	if v, ok := os.LookupEnv("DATABASE_MAX_CONNECTIONS"); ok {
		parsed, parseErr := strconv.Atoi(v)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid DATABASE_MAX_CONNECTIONS: %w", parseErr)
		}
		maxConns = parsed
	}

	serverHost := envOrDefault("SERVER_HOST", "0.0.0.0")

	serverPort := 3500
	if v, ok := os.LookupEnv("SERVER_PORT"); ok {
		parsed, parseErr := strconv.Atoi(v)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid SERVER_PORT: %w", parseErr)
		}
		serverPort = parsed
	}

	logLevel := envOrDefault("LOG_LEVEL", "info")

	var errorReportingDSN *string
	if v, ok := os.LookupEnv("ERROR_REPORTING_DSN"); ok {
		errorReportingDSN = &v
	}

	envStr := envOrDefault("ENVIRONMENT", "development")
	var env Environment
	switch envStr {
	case "production":
		env = EnvironmentProduction
	case "staging":
		env = EnvironmentStaging
	default:
		env = EnvironmentDevelopment
	}

	// Optional Hyperswitch config (omit to disable payments)
	hyperswitchBaseURL := envOrDefault("HYPERSWITCH_BASE_URL", "")
	hyperswitchAPIKey := envOrDefault("HYPERSWITCH_API_KEY", "")
	hyperswitchWebhookKey := envOrDefault("HYPERSWITCH_WEBHOOK_KEY", "")

	origins := strings.Split(corsOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	return &AppConfig{
		DatabaseURL:            databaseURL,
		DatabaseMaxConnections: maxConns,
		RedisURL:               redisURL,
		AuthAdminURL:           authAdminURL,
		AuthPublicURL:          authPublicURL,
		AuthWebhookSecret:      authWebhookSecret,
		CORSAllowedOrigins:     origins,
		ServerHost:             serverHost,
		ServerPort:             serverPort,
		LogLevel:               logLevel,
		ErrorReportingDSN:      errorReportingDSN,
		HyperswitchBaseURL:    hyperswitchBaseURL,
		HyperswitchAPIKey:     hyperswitchAPIKey,
		HyperswitchWebhookKey: hyperswitchWebhookKey,
		Environment:            env,
	}, nil
}

// requiredEnv returns an error if the environment variable is absent or empty.
func requiredEnv(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return val, nil
}

// envOrDefault returns the env var value or defaultVal if absent.
func envOrDefault(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}
