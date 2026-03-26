package admin

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Custom DB Types
// ═══════════════════════════════════════════════════════════════════════════════

// uuidArray is a custom type for PostgreSQL UUID[] columns.
// Implements database/sql.Scanner and driver.Valuer without requiring lib/pq.
// Follows the SlugArray pattern from iam/models.go.
type uuidArray []uuid.UUID

// Value implements driver.Valuer. Serializes to PostgreSQL array literal.
func (a uuidArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}
	parts := make([]string, len(a))
	for i, id := range a {
		parts[i] = id.String()
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

// Scan implements sql.Scanner. Parses PostgreSQL array literal {uuid1,uuid2,...}.
func (a *uuidArray) Scan(src any) error {
	if src == nil {
		*a = nil
		return nil
	}
	var str string
	switch v := src.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("uuidArray.Scan: unsupported type %T", src)
	}
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*a = uuidArray{}
		return nil
	}
	parts := strings.Split(str, ",")
	result := make(uuidArray, len(parts))
	for i, p := range parts {
		id, err := uuid.Parse(strings.TrimSpace(p))
		if err != nil {
			return fmt.Errorf("uuidArray.Scan: invalid UUID %q: %w", p, err)
		}
		result[i] = id
	}
	*a = result
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Feature Flag Repository [16-admin §6]
// ═══════════════════════════════════════════════════════════════════════════════

// featureFlagRow is the GORM model for admin_feature_flags.
type featureFlagRow struct {
	ID                uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	Key               string     `gorm:"column:key;uniqueIndex"`
	Description       string     `gorm:"column:description"`
	Enabled           bool       `gorm:"column:enabled"`
	RolloutPercentage *int16     `gorm:"column:rollout_percentage"`
	AllowedFamilyIDs  uuidArray  `gorm:"column:allowed_family_ids;type:uuid[]"`
	CreatedBy         uuid.UUID  `gorm:"column:created_by"`
	UpdatedBy         *uuid.UUID `gorm:"column:updated_by"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (featureFlagRow) TableName() string { return "admin_feature_flags" }

func (r featureFlagRow) toModel() FeatureFlag {
	return FeatureFlag{
		ID:                r.ID,
		Key:               r.Key,
		Description:       r.Description,
		Enabled:           r.Enabled,
		RolloutPercentage: r.RolloutPercentage,
		AllowedFamilyIDs:  []uuid.UUID(r.AllowedFamilyIDs),
		CreatedBy:         r.CreatedBy,
		UpdatedBy:         r.UpdatedBy,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

// PgFeatureFlagRepository is the PostgreSQL implementation of FeatureFlagRepository.
type PgFeatureFlagRepository struct {
	db *gorm.DB
}

// NewPgFeatureFlagRepository creates a new PgFeatureFlagRepository.
func NewPgFeatureFlagRepository(db *gorm.DB) *PgFeatureFlagRepository {
	return &PgFeatureFlagRepository{db: db}
}

func (r *PgFeatureFlagRepository) ListAll(ctx context.Context) ([]FeatureFlag, error) {
	var rows []featureFlagRow
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	flags := make([]FeatureFlag, len(rows))
	for i, row := range rows {
		flags[i] = row.toModel()
	}
	return flags, nil
}

func (r *PgFeatureFlagRepository) FindByKey(ctx context.Context, key string) (*FeatureFlag, error) {
	var row featureFlagRow
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, shared.ErrDatabase(err)
	}
	flag := row.toModel()
	return &flag, nil
}

func (r *PgFeatureFlagRepository) Create(ctx context.Context, input *CreateFlagInput, adminID uuid.UUID) (*FeatureFlag, error) {
	row := featureFlagRow{
		ID:                uuid.Must(uuid.NewV7()),
		Key:               input.Key,
		Description:       input.Description,
		Enabled:           input.Enabled,
		RolloutPercentage: input.RolloutPercentage,
		AllowedFamilyIDs:  uuidArray(input.AllowedFamilyIDs),
		CreatedBy:         adminID,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrFlagAlreadyExists
		}
		return nil, shared.ErrDatabase(err)
	}

	flag := row.toModel()
	return &flag, nil
}

func (r *PgFeatureFlagRepository) Update(ctx context.Context, key string, input *UpdateFlagInput, adminID uuid.UUID) (*FeatureFlag, error) {
	var row featureFlagRow
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFlagNotFound
		}
		return nil, shared.ErrDatabase(err)
	}

	updates := map[string]any{"updated_by": adminID}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.RolloutPercentage != nil {
		updates["rollout_percentage"] = *input.RolloutPercentage
	}
	if input.AllowedFamilyIDs != nil {
		updates["allowed_family_ids"] = uuidArray(*input.AllowedFamilyIDs)
	}

	if err := r.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}

	// Re-read to get updated_at from DB.
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&row).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}

	flag := row.toModel()
	return &flag, nil
}

func (r *PgFeatureFlagRepository) Delete(ctx context.Context, key string) error {
	result := r.db.WithContext(ctx).Where("key = ?", key).Delete(&featureFlagRow{})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrFlagNotFound
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Audit Log Repository [16-admin §6, §8]
// ═══════════════════════════════════════════════════════════════════════════════

// auditLogRow is the GORM model for admin_audit_log.
type auditLogRow struct {
	ID         uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	AdminID    uuid.UUID  `gorm:"column:admin_id"`
	Action     string     `gorm:"column:action"`
	TargetType string     `gorm:"column:target_type"`
	TargetID   *uuid.UUID `gorm:"column:target_id"`
	Details    []byte     `gorm:"column:details;type:jsonb"`
	IPAddress  *string    `gorm:"column:ip_address"`
	UserAgent  *string    `gorm:"column:user_agent"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (auditLogRow) TableName() string { return "admin_audit_log" }

func (r auditLogRow) toModel() AuditLogEntry {
	return AuditLogEntry{
		ID:         r.ID,
		AdminID:    r.AdminID,
		Action:     r.Action,
		TargetType: r.TargetType,
		TargetID:   r.TargetID,
		Details:    r.Details,
		CreatedAt:  r.CreatedAt,
	}
}

// PgAuditLogRepository is the PostgreSQL implementation of AuditLogRepository.
type PgAuditLogRepository struct {
	db *gorm.DB
}

// NewPgAuditLogRepository creates a new PgAuditLogRepository.
func NewPgAuditLogRepository(db *gorm.DB) *PgAuditLogRepository {
	return &PgAuditLogRepository{db: db}
}

func (r *PgAuditLogRepository) Create(ctx context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
	var ipAddr *string
	if entry.IPAddress != "" {
		ipAddr = &entry.IPAddress
	}
	var ua *string
	if entry.UserAgent != "" {
		ua = &entry.UserAgent
	}

	row := auditLogRow{
		ID:         uuid.Must(uuid.NewV7()),
		AdminID:    entry.AdminID,
		Action:     entry.Action,
		TargetType: entry.TargetType,
		TargetID:   entry.TargetID,
		Details:    entry.Details,
		IPAddress:  ipAddr,
		UserAgent:  ua,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}

	model := row.toModel()
	return &model, nil
}

func (r *PgAuditLogRepository) Search(ctx context.Context, query *AuditLogQuery, pagination *shared.PaginationParams) ([]AuditLogEntry, error) {
	q := r.db.WithContext(ctx).Model(&auditLogRow{}).Order("created_at DESC")

	if query.AdminID != nil {
		q = q.Where("admin_id = ?", *query.AdminID)
	}
	if query.Action != nil {
		q = q.Where("action = ?", *query.Action)
	}
	if query.TargetType != nil {
		q = q.Where("target_type = ?", *query.TargetType)
	}
	if query.TargetID != nil {
		q = q.Where("target_id = ?", *query.TargetID)
	}
	if query.FromDate != nil {
		q = q.Where("created_at >= ?", *query.FromDate)
	}
	if query.ToDate != nil {
		q = q.Where("created_at <= ?", *query.ToDate)
	}

	q = q.Limit(pagination.EffectiveLimit())

	var rows []auditLogRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}

	entries := make([]AuditLogEntry, len(rows))
	for i, row := range rows {
		entries[i] = row.toModel()
	}
	return entries, nil
}

func (r *PgAuditLogRepository) FindByTarget(ctx context.Context, targetType string, targetID uuid.UUID, pagination *shared.PaginationParams) ([]AuditLogEntry, error) {
	var rows []auditLogRow
	if err := r.db.WithContext(ctx).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Order("created_at DESC").
		Limit(pagination.EffectiveLimit()).
		Find(&rows).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}

	entries := make([]AuditLogEntry, len(rows))
	for i, row := range rows {
		entries[i] = row.toModel()
	}
	return entries, nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

// isDuplicateKeyError checks if a GORM error is a PostgreSQL unique violation.
func isDuplicateKeyError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "23505")
}
