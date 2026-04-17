package notify

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// PgNotificationRepository [08-notify §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgNotificationRepository implements NotificationRepository using GORM.
type PgNotificationRepository struct{ db *gorm.DB }

func NewPgNotificationRepository(db *gorm.DB) NotificationRepository {
	return &PgNotificationRepository{db: db}
}

func (r *PgNotificationRepository) Create(ctx context.Context, cmd CreateNotification) (*NotifyNotification, error) {
	n := NotifyNotification{
		ID:               uuid.Must(uuid.NewV7()),
		FamilyID:         cmd.FamilyID,
		NotificationType: cmd.NotificationType,
		Category:         cmd.Category,
		Title:            cmd.Title,
		Body:             cmd.Body,
		ActionURL:        cmd.ActionURL,
		Metadata:         cmd.Metadata,
	}
	if err := r.db.WithContext(ctx).Create(&n).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &n, nil
}

func (r *PgNotificationRepository) GetByID(ctx context.Context, id uuid.UUID, scope *shared.FamilyScope) (*NotifyNotification, error) {
	var n NotifyNotification
	err := r.db.WithContext(ctx).
		Where("id = ? AND family_id = ?", id, scope.FamilyID()).
		First(&n).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &NotifyError{Err: ErrNotificationNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &n, nil
}

func (r *PgNotificationRepository) List(ctx context.Context, params *NotificationListParams, scope *shared.FamilyScope) ([]NotifyNotification, error) {
	tx := r.db.WithContext(ctx).Where("family_id = ?", scope.FamilyID())

	if params.Category != nil {
		tx = tx.Where("category = ?", *params.Category)
	}
	if params.UnreadOnly != nil && *params.UnreadOnly {
		tx = tx.Where("is_read = false")
	}
	if params.Cursor != nil {
		cursorID, cursorTS, err := shared.DecodeCursor(*params.Cursor)
		if err != nil {
			return nil, err
		}
		tx = tx.Where("(created_at, id) < (?, ?)", cursorTS, cursorID)
	}

	limit := 20
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = int(*params.Limit)
	}

	// Fetch limit+1 to detect whether more pages exist.
	var notifications []NotifyNotification
	if err := tx.Order("created_at DESC, id DESC").Limit(limit + 1).Find(&notifications).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return notifications, nil
}

// CountAll returns the total number of notifications matching the filter, ignoring
// pagination. Used to populate NotificationListResponse.Total so the UI can render
// accurate counts without guessing from the current page.
func (r *PgNotificationRepository) CountAll(ctx context.Context, params *NotificationListParams, scope *shared.FamilyScope) (int64, error) {
	tx := r.db.WithContext(ctx).Model(&NotifyNotification{}).
		Where("family_id = ?", scope.FamilyID())

	if params.Category != nil {
		tx = tx.Where("category = ?", *params.Category)
	}
	if params.UnreadOnly != nil && *params.UnreadOnly {
		tx = tx.Where("is_read = false")
	}

	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}

func (r *PgNotificationRepository) CountUnread(ctx context.Context, scope *shared.FamilyScope) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&NotifyNotification{}).
		Where("family_id = ? AND is_read = false", scope.FamilyID()).
		Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}

func (r *PgNotificationRepository) MarkRead(ctx context.Context, id uuid.UUID, scope *shared.FamilyScope) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(&NotifyNotification{}).
		Where("id = ? AND family_id = ?", id, scope.FamilyID()).
		Update("is_read", true)
	if result.Error != nil {
		return false, shared.ErrDatabase(result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *PgNotificationRepository) MarkAllRead(ctx context.Context, scope *shared.FamilyScope, category *string) (int64, error) {
	tx := r.db.WithContext(ctx).
		Model(&NotifyNotification{}).
		Where("family_id = ? AND is_read = false", scope.FamilyID())
	if category != nil {
		tx = tx.Where("category = ?", *category)
	}
	result := tx.Update("is_read", true)
	if result.Error != nil {
		return 0, shared.ErrDatabase(result.Error)
	}
	return result.RowsAffected, nil
}

func (r *PgNotificationRepository) ExistsBySourceEvent(ctx context.Context, familyID uuid.UUID, notificationType string, sourceEventID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&NotifyNotification{}).
		Where("family_id = ? AND notification_type = ? AND metadata->>'source_event_id' = ?",
			familyID, notificationType, sourceEventID).
		Count(&count).Error
	if err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count > 0, nil
}

func (r *PgNotificationRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("family_id = ?", familyID).Delete(&NotifyNotification{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgPreferenceRepository [08-notify §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgPreferenceRepository implements PreferenceRepository using GORM.
type PgPreferenceRepository struct{ db *gorm.DB }

func NewPgPreferenceRepository(db *gorm.DB) PreferenceRepository {
	return &PgPreferenceRepository{db: db}
}

func (r *PgPreferenceRepository) GetAll(ctx context.Context, scope *shared.FamilyScope) ([]NotifyPreference, error) {
	var prefs []NotifyPreference
	if err := r.db.WithContext(ctx).Where("family_id = ?", scope.FamilyID()).Find(&prefs).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return prefs, nil
}

func (r *PgPreferenceRepository) UpsertBatch(ctx context.Context, scope *shared.FamilyScope, updates []PreferenceUpsert) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, u := range updates {
			pref := NotifyPreference{
				FamilyID:         scope.FamilyID(),
				NotificationType: u.NotificationType,
				Channel:          u.Channel,
				Enabled:          u.Enabled,
				DigestFrequency:  u.DigestFrequency,
			}
			result := tx.Where("family_id = ? AND notification_type = ? AND channel = ?",
				scope.FamilyID(), u.NotificationType, u.Channel).
				Assign(map[string]any{
					"enabled":          u.Enabled,
					"digest_frequency": u.DigestFrequency,
				}).
				FirstOrCreate(&pref)
			if result.Error != nil {
				return shared.ErrDatabase(result.Error)
			}
		}
		return nil
	})
}

func (r *PgPreferenceRepository) IsEnabled(ctx context.Context, familyID uuid.UUID, notificationType string, channel string) (bool, error) {
	var pref NotifyPreference
	err := r.db.WithContext(ctx).
		Where("family_id = ? AND notification_type = ? AND channel = ?", familyID, notificationType, channel).
		First(&pref).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil // Default-enabled semantics: missing row = enabled
		}
		return false, shared.ErrDatabase(err)
	}
	return pref.Enabled, nil
}

func (r *PgPreferenceRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("family_id = ?", familyID).Delete(&NotifyPreference{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgDigestRepository (Phase 2) [08-notify §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgDigestRepository implements DigestRepository using GORM.
type PgDigestRepository struct{ db *gorm.DB }

func NewPgDigestRepository(db *gorm.DB) DigestRepository {
	return &PgDigestRepository{db: db}
}

func (r *PgDigestRepository) Create(ctx context.Context, cmd CreateDigest) (*NotifyDigest, error) {
	d := NotifyDigest{
		ID:          uuid.Must(uuid.NewV7()),
		FamilyID:    cmd.FamilyID,
		DigestType:  cmd.DigestType,
		PeriodStart: cmd.PeriodStart,
		PeriodEnd:   cmd.PeriodEnd,
		ContentJSON: cmd.ContentJSON,
	}
	if len(d.ContentJSON) == 0 {
		d.ContentJSON = json.RawMessage(`{}`)
	}
	if err := r.db.WithContext(ctx).Create(&d).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &d, nil
}

func (r *PgDigestRepository) GetUnsent(ctx context.Context, limit int64) ([]NotifyDigest, error) {
	var digests []NotifyDigest
	if err := r.db.WithContext(ctx).
		Where("sent_at IS NULL").
		Order("created_at ASC").
		Limit(int(limit)).
		Find(&digests).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return digests, nil
}

func (r *PgDigestRepository) MarkSent(ctx context.Context, digestID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Model(&NotifyDigest{}).
		Where("id = ?", digestID).
		Update("sent_at", gorm.Expr("now()")).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}
