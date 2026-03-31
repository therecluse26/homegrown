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
// Adds `WHERE family_id = ?` to every GORM query within the transaction,
// ensuring all reads and writes are restricted to the given family. [ARCH §1.5, §5.2, ADR-008]
func ScopedTransaction(ctx context.Context, db *gorm.DB, scope FamilyScope, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		scopedTx := tx.Where("family_id = ?", scope.FamilyID())
		return fn(scopedTx)
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

// BypassRLSTransaction executes fn in a transaction WITHOUT family-scoped WHERE clauses.
//
// Functionally identical to UnscopedTransaction, but the distinct name documents intent:
// the caller explicitly needs cross-family access. ONLY for exceptional cases:
//   - Auth middleware: finding parent by Kratos identity_id before FamilyScope exists
//   - Registration webhooks: family does not exist yet
//   - Post-login webhooks: looking up parent by Kratos ID, no family scope in context
//
// Every call site MUST have a comment explaining why unscoped access is required. [01-iam §11.1]
func BypassRLSTransaction(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}
