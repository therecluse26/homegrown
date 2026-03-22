module github.com/homegrown-academy/homegrown-academy

go 1.23

require (
	// Web framework
	github.com/labstack/echo/v4 v4.12.0

	// Database
	gorm.io/gorm v1.25.12
	gorm.io/driver/postgres v1.5.9

	// Migrations
	github.com/pressly/goose/v3 v3.21.1

	// Validation
	github.com/go-playground/validator/v10 v10.22.0

	// OpenAPI
	github.com/swaggo/swag v1.16.3
	github.com/swaggo/echo-swagger v1.4.1

	// Types
	github.com/google/uuid v1.6.0

	// Redis
	github.com/redis/go-redis/v9 v9.7.0

	// Configuration
	github.com/joho/godotenv v1.5.1

	// HTML sanitization [CODING §5.2]
	github.com/microcosm-cc/bluemonday v1.0.27

	// Background jobs
	github.com/hibiken/asynq v0.24.1

	// WebSocket
	github.com/gorilla/websocket v1.5.3

	// Sentry (optional error tracking) [ARCH §2.14]
	github.com/getsentry/sentry-go v0.28.0
)
