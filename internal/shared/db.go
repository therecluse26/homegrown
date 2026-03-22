package shared

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// CreatePool creates a GORM database connection pool.
// Uses slog for logging (GORM's built-in logger is silenced). [§9.1]
func CreatePool(cfg *config.AppConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Use slog, not GORM's built-in logging
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.DatabaseMaxConnections)
	sqlDB.SetMaxIdleConns(1)

	return db, nil
}

// ScopedTransaction executes fn within a family-scoped transaction.
//
// Sets `SET LOCAL app.current_family_id` for RLS enforcement. The local setting
// is automatically cleared on commit or rollback. [ARCH §1.5, §5.2]
func ScopedTransaction(ctx context.Context, db *gorm.DB, scope FamilyScope, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// SET LOCAL is transaction-scoped — automatically reset on commit/rollback.
		// The UUID type guarantees no SQL injection here.
		setSQL := fmt.Sprintf("SET LOCAL app.current_family_id = '%s'", scope.FamilyID().String())
		if err := tx.Exec(setSQL).Error; err != nil {
			return fmt.Errorf("failed to set family scope: %w", err)
		}
		return fn(tx)
	})
}

// UnscopedTransaction executes fn in a transaction WITHOUT family scope.
//
// ONLY for:
//   - Auth middleware lookups (FamilyScope not yet constructed)
//   - Registration webhooks (family does not exist yet)
//   - Background cleanup jobs (cross-family by design)
//
// Every call site MUST have a comment explaining why unscoped access is required.
func UnscopedTransaction(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}
