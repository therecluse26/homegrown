package config

import (
	"fmt"
	"log/slog"
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

	// ─── Auth Provider (Hearth) ──────────────────────────────────────
	// Hearth public URL — OIDC / token endpoints. Example: "http://localhost:4433"
	AuthPublicURL string

	// Hearth admin URL — internal admin API. Example: "http://localhost:4934"
	AuthAdminURL string

	// Hearth realm ID. Example: "homegrown"
	HearthRealmID string

	// Hearth SPA client ID (public client, no secret). Example: "homegrown-spa"
	HearthClientID string

	// Shared secret for Hearth signed webhook signature validation.
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

	// OTLP exporter endpoint (gRPC). Optional — omit to disable OTLP export.
	// In development, traces are emitted to stdout when this is unset.
	// Example: "grpc://localhost:4317" or "https://api.honeycomb.io"
	OTELEndpoint *string

	// Fraction of requests to sample for tracing. Default: 1.0 (dev), 0.1 (prod/staging).
	OTELSampleRate float64

	// ─── Payments (Hyperswitch) ─────────────────────────────────────
	// Hyperswitch base URL. Optional — omit to disable payment processing.
	// Example: "http://hyperswitch:8080"
	HyperswitchBaseURL string

	// Hyperswitch API key. Required when HyperswitchBaseURL is set.
	HyperswitchAPIKey string

	// Hyperswitch webhook signing key. Required when HyperswitchBaseURL is set.
	HyperswitchWebhookKey string

	// ─── Billing (Hyperswitch Subscriptions) ──────────────────────
	// Hyperswitch business profile ID for the billing/subscription domain.
	// Separate from the marketplace payment profile. [10-billing §7]
	HyperswitchBillingProfileID string

	// ─── Marketplace (Hyperswitch Payments) ───────────────────────
	// Hyperswitch business profile ID for marketplace payments (listing purchases,
	// split payments, creator payouts). Separate from the billing profile. [07-mkt §18.5]
	HyperswitchMktProfileID string

	// Hyperswitch price ID for the monthly subscription plan.
	HyperswitchMonthlyPriceID string

	// Hyperswitch price ID for the annual subscription plan.
	HyperswitchAnnualPriceID string

	// COPPA micro-charge amount in cents. Default: 50 ($0.50). [10-billing §3]
	CoppaChargeCents int64

	// Webhook signing secret for billing-specific Hyperswitch webhooks.
	BillingWebhookSecret string

	// ─── Application ────────────────────────────────────────────────
	// Public base URL of the app (e.g. "https://app.homegrown.academy").
	// Used to construct absolute URLs in emails (unsubscribe links, etc.).
	// Required in production; defaults to "http://localhost:5173" in development.
	AppPublicURL string

	// ─── Notifications (Postmark) ──────────────────────────────────
	// Postmark server API token. Optional — omit to use NoopEmailAdapter.
	PostmarkServerToken string

	// From address for all outbound emails, e.g. "Homegrown Academy <hello@example.com>".
	// Required when PostmarkServerToken is set.
	PostmarkFromAddress string

	// HMAC secret for generating one-click email unsubscribe tokens. [08-notify §13]
	UnsubscribeSecret string

	// ─── Object Storage (S3-compatible) ──────────────────────────────
	// S3-compatible object storage endpoint URL (e.g., R2, MinIO).
	// Optional — omit to use NoopStorageAdapter (dev mode). [ARCH §2.10]
	ObjectStorageEndpoint string

	// S3 region or "auto" for R2.
	ObjectStorageRegion string

	// S3 bucket name.
	ObjectStorageBucket string

	// S3 access key ID.
	ObjectStorageAccessKeyID string

	// S3 secret access key.
	ObjectStorageSecretAccessKey string

	// Public base URL for CDN-served media (e.g., "https://media.homegrown.academy").
	ObjectStoragePublicURL string

	// ─���─ Safety Scanning (Thorn Safer) ───────────────────────────────
	// Thorn Safer API key. Optional — omit to use NoopSafetyScanAdapter.
	// When set, CSAM scanning uses Thorn Safer PhotoDNA hash matching. [09-media §7.2]
	ThornAPIKey string

	// Thorn Safer API base URL. Default: "https://safer.thorn.org".
	ThornBaseURL string

	// ─── Session Encryption ──────────────────────────────────────────
	// AES-256 key (32 bytes) for encrypting access and refresh tokens at rest in iam_sessions.
	// Loaded from HEARTH_SESSION_KEY (hex-encoded). A dev default is used when not set. [ADR-D]
	HearthSessionKey []byte

	// ─── Recommendations ─────────────────────────────────────────────
	// HMAC secret for anonymizing family IDs in recs_anonymized_interactions.
	// Optional — empty value disables the anonymization task (dev mode). [13-recs §14.3]
	RecsAnonymizationSecret string

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

	authPublicURL, err := requiredEnv("AUTH_PUBLIC_URL")
	if err != nil {
		return nil, err
	}

	authAdminURL, err := requiredEnv("AUTH_ADMIN_URL")
	if err != nil {
		return nil, err
	}

	hearthRealmID := envOrDefault("HEARTH_REALM_ID", "homegrown")
	hearthClientID := envOrDefault("HEARTH_CLIENT_ID", "homegrown-spa")

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

	var otelEndpoint *string
	if v, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); ok {
		otelEndpoint = &v
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

	// OTEL sample rate: default 1.0 in dev, 0.1 in prod/staging. Override with OTEL_SAMPLE_RATE.
	otelSampleRate := 1.0
	if env == EnvironmentProduction || env == EnvironmentStaging {
		otelSampleRate = 0.1
	}
	if v, ok := os.LookupEnv("OTEL_SAMPLE_RATE"); ok {
		parsed, parseErr := strconv.ParseFloat(v, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid OTEL_SAMPLE_RATE: %w", parseErr)
		}
		otelSampleRate = parsed
	}

	// Optional Hyperswitch config (omit to disable payments)
	hyperswitchBaseURL := envOrDefault("HYPERSWITCH_BASE_URL", "")
	hyperswitchAPIKey := envOrDefault("HYPERSWITCH_API_KEY", "")
	hyperswitchWebhookKey := envOrDefault("HYPERSWITCH_WEBHOOK_KEY", "")

	// Billing-specific Hyperswitch config [10-billing §7]
	hyperswitchBillingProfileID := envOrDefault("HYPERSWITCH_BILLING_PROFILE_ID", "")
	// Marketplace-specific Hyperswitch config [07-mkt §18.5]
	hyperswitchMktProfileID := envOrDefault("HYPERSWITCH_MKT_PROFILE_ID", "")
	hyperswitchMonthlyPriceID := envOrDefault("HYPERSWITCH_MONTHLY_PRICE_ID", "")
	hyperswitchAnnualPriceID := envOrDefault("HYPERSWITCH_ANNUAL_PRICE_ID", "")
	coppaChargeCents := int64(50) // default $0.50
	if v, ok := os.LookupEnv("COPPA_CHARGE_CENTS"); ok {
		parsed, parseErr := strconv.ParseInt(v, 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid COPPA_CHARGE_CENTS: %w", parseErr)
		}
		coppaChargeCents = parsed
	}
	billingWebhookSecret := envOrDefault("BILLING_WEBHOOK_SECRET", "")

	// Validate critical secrets at startup in non-development environments. [P3-8]
	if env != EnvironmentDevelopment {
		if billingWebhookSecret == "" {
			slog.Warn("BILLING_WEBHOOK_SECRET not set — billing webhooks will fail signature verification")
		}
		// Reject known dev-default placeholder values — they appear in .env.example and must
		// never reach a deployed environment. If AUTH_WEBHOOK_SECRET or UNSUBSCRIBE_SECRET are
		// still set to these strings it means the operator copied .env.example without changing them.
		knownDevDefaults := map[string]string{
			"AUTH_WEBHOOK_SECRET": "dev-webhook-secret-change-in-production",
			"UNSUBSCRIBE_SECRET":  "dev-unsubscribe-secret-change-in-production",
		}
		for name, badVal := range knownDevDefaults {
			if os.Getenv(name) == badVal {
				return nil, fmt.Errorf("environment variable %s is set to a known dev-default placeholder; replace it with a real secret before deploying", name) //nolint:goerr113
			}
		}
	}

	// Optional object storage config (omit to use noop adapter)
	objectStorageEndpoint := envOrDefault("OBJECT_STORAGE_ENDPOINT", "")
	objectStorageRegion := envOrDefault("OBJECT_STORAGE_REGION", "auto")
	objectStorageBucket := envOrDefault("OBJECT_STORAGE_BUCKET", "")
	objectStorageAccessKeyID := envOrDefault("OBJECT_STORAGE_ACCESS_KEY_ID", "")
	objectStorageSecretAccessKey := envOrDefault("OBJECT_STORAGE_SECRET_ACCESS_KEY", "")
	objectStoragePublicURL := os.Getenv("OBJECT_STORAGE_PUBLIC_URL")
	if objectStoragePublicURL == "" {
		if env == EnvironmentProduction {
			return nil, fmt.Errorf("required environment variable OBJECT_STORAGE_PUBLIC_URL is not set") //nolint:goerr113
		}
		slog.Warn("OBJECT_STORAGE_PUBLIC_URL not set; media URLs will be relative — set this env var in production")
		objectStoragePublicURL = "https://media.localhost"
	}

	// Public base URL (used for absolute URLs in emails)
	appPublicURL := os.Getenv("APP_PUBLIC_URL")
	if appPublicURL == "" {
		if env == EnvironmentProduction {
			return nil, fmt.Errorf("required environment variable APP_PUBLIC_URL is not set") //nolint:goerr113
		}
		appPublicURL = "http://localhost:5173"
		slog.Warn("APP_PUBLIC_URL not set; using localhost default — set this env var in production")
	}

	// Optional Postmark config (omit to disable email)
	postmarkServerToken := envOrDefault("POSTMARK_SERVER_TOKEN", "")
	postmarkFromAddress := os.Getenv("POSTMARK_FROM_ADDRESS")
	if postmarkServerToken != "" && postmarkFromAddress == "" {
		if env == EnvironmentProduction {
			return nil, fmt.Errorf("required environment variable POSTMARK_FROM_ADDRESS is not set") //nolint:goerr113
		}
		slog.Warn("POSTMARK_FROM_ADDRESS not set; Postmark will use server-level default sender — set this env var in production")
	}
	unsubscribeSecret := os.Getenv("UNSUBSCRIBE_SECRET")
	if unsubscribeSecret == "" {
		if env == EnvironmentProduction {
			return nil, fmt.Errorf("required environment variable UNSUBSCRIBE_SECRET is not set") //nolint:goerr113
		}
		slog.Warn("UNSUBSCRIBE_SECRET not set; email unsubscribe tokens will fail — set this env var in production")
	}

	// Optional Thorn Safer config (omit to use noop safety scanner)
	thornAPIKey := envOrDefault("THORN_API_KEY", "")
	thornBaseURL := envOrDefault("THORN_BASE_URL", "https://safer.thorn.org")

	// Optional recs anonymization secret (omit to disable anonymization task)
	recsAnonymizationSecret := envOrDefault("RECS_ANONYMIZATION_SECRET", "")

	// Session token encryption key: 32-byte AES-256 key, hex-encoded in HEARTH_SESSION_KEY.
	// An insecure dev default is used when the variable is absent; production requires it. [ADR-D]
	hearthSessionKey, err := loadHearthSessionKey(env)
	if err != nil {
		return nil, err
	}

	origins := strings.Split(corsOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	return &AppConfig{
		DatabaseURL:            databaseURL,
		DatabaseMaxConnections: maxConns,
		RedisURL:               redisURL,
		AuthPublicURL:          authPublicURL,
		AuthAdminURL:           authAdminURL,
		HearthRealmID:          hearthRealmID,
		HearthClientID:         hearthClientID,
		HearthSessionKey:       hearthSessionKey,
		AuthWebhookSecret:      authWebhookSecret,
		CORSAllowedOrigins:     origins,
		ServerHost:             serverHost,
		ServerPort:             serverPort,
		LogLevel:               logLevel,
		ErrorReportingDSN:      errorReportingDSN,
		OTELEndpoint:           otelEndpoint,
		OTELSampleRate:         otelSampleRate,
		HyperswitchBaseURL:          hyperswitchBaseURL,
		HyperswitchAPIKey:           hyperswitchAPIKey,
		HyperswitchWebhookKey:       hyperswitchWebhookKey,
		HyperswitchBillingProfileID: hyperswitchBillingProfileID,
		HyperswitchMktProfileID:     hyperswitchMktProfileID,
		HyperswitchMonthlyPriceID:   hyperswitchMonthlyPriceID,
		HyperswitchAnnualPriceID:    hyperswitchAnnualPriceID,
		CoppaChargeCents:            coppaChargeCents,
		BillingWebhookSecret:        billingWebhookSecret,
		ObjectStorageEndpoint:       objectStorageEndpoint,
		ObjectStorageRegion:         objectStorageRegion,
		ObjectStorageBucket:         objectStorageBucket,
		ObjectStorageAccessKeyID:    objectStorageAccessKeyID,
		ObjectStorageSecretAccessKey: objectStorageSecretAccessKey,
		ObjectStoragePublicURL:      objectStoragePublicURL,
		AppPublicURL:            appPublicURL,
		PostmarkServerToken:     postmarkServerToken,
		PostmarkFromAddress:     postmarkFromAddress,
		UnsubscribeSecret:       unsubscribeSecret,
		ThornAPIKey:             thornAPIKey,
		ThornBaseURL:            thornBaseURL,
		RecsAnonymizationSecret: recsAnonymizationSecret,
		Environment:              env,
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

// loadHearthSessionKey loads the 32-byte AES-256 session encryption key.
// In production the key MUST be set via HEARTH_SESSION_KEY (64 hex chars = 32 bytes).
// A fixed insecure dev default is used otherwise; warning logged for non-dev environments.
func loadHearthSessionKey(env Environment) ([]byte, error) {
	const devDefault = "0000000000000000000000000000000000000000000000000000000000000000"
	raw := envOrDefault("HEARTH_SESSION_KEY", "")
	if raw == "" {
		if env != EnvironmentDevelopment {
			slog.Warn("HEARTH_SESSION_KEY not set; using insecure dev default — set a real key in production")
		}
		raw = devDefault
	}
	if len(raw) != 64 {
		return nil, fmt.Errorf("HEARTH_SESSION_KEY must be 64 hex characters (32 bytes), got %d", len(raw))
	}
	key := make([]byte, 32)
	for i := range 32 {
		b, err := strconv.ParseUint(raw[i*2:i*2+2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("HEARTH_SESSION_KEY contains invalid hex at position %d: %w", i*2, err)
		}
		key[i] = byte(b)
	}
	return key, nil
}
