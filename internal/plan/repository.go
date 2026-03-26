package plan

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// PostgreSQL Repository Implementation [17-planning §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgScheduleItemRepository implements ScheduleItemRepository using GORM.
type PgScheduleItemRepository struct {
	db *gorm.DB
}

// NewPgScheduleItemRepository creates a new PgScheduleItemRepository.
func NewPgScheduleItemRepository(db *gorm.DB) ScheduleItemRepository {
	return &PgScheduleItemRepository{db: db}
}

func (r *PgScheduleItemRepository) Create(ctx context.Context, scope *shared.FamilyScope, item *ScheduleItem) error {
	item.FamilyID = scope.FamilyID()
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(item).Error
	})
}

func (r *PgScheduleItemRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ScheduleItem, error) {
	var item ScheduleItem
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).First(&item).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *PgScheduleItemRepository) FindByLinkedEventID(ctx context.Context, eventID uuid.UUID) ([]ScheduleItem, error) {
	var items []ScheduleItem
	// Bypass family scope — event handlers query across families. [17-planning §16]
	err := shared.BypassRLSTransaction(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Where("linked_event_id = ?", eventID).Find(&items).Error
	})
	return items, err
}

func (r *PgScheduleItemRepository) FindByStudentAndDate(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, date time.Time) ([]ScheduleItem, error) {
	var items []ScheduleItem
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("student_id = ? AND start_date = ?", studentID, date).
			Order("start_time").Find(&items).Error
	})
	return items, err
}

func (r *PgScheduleItemRepository) ListByDateRange(
	ctx context.Context,
	scope *shared.FamilyScope,
	start time.Time,
	end time.Time,
	studentID *uuid.UUID,
) ([]ScheduleItem, error) {
	var items []ScheduleItem
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		q := tx.Where("start_date >= ? AND start_date < ?", start, end)
		if studentID != nil {
			q = q.Where("student_id = ?", *studentID)
		}
		return q.Order("start_date, start_time").Find(&items).Error
	})
	return items, err
}

func (r *PgScheduleItemRepository) ListFiltered(
	ctx context.Context,
	scope *shared.FamilyScope,
	query *ScheduleItemQuery,
	pagination *shared.PaginationParams,
) ([]ScheduleItem, error) {
	var items []ScheduleItem
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		q := tx.Model(&ScheduleItem{})
		if query.StartDate != nil {
			q = q.Where("start_date >= ?", *query.StartDate)
		}
		if query.EndDate != nil {
			q = q.Where("start_date <= ?", *query.EndDate)
		}
		if query.StudentID != nil {
			q = q.Where("student_id = ?", *query.StudentID)
		}
		if query.Category != nil {
			q = q.Where("category = ?", *query.Category)
		}
		if query.IsCompleted != nil {
			q = q.Where("is_completed = ?", *query.IsCompleted)
		}
		limit := pagination.EffectiveLimit()
		q = q.Limit(limit + 1).Order("start_date DESC, created_at DESC")
		return q.Find(&items).Error
	})
	return items, err
}

func (r *PgScheduleItemRepository) Update(
	ctx context.Context,
	scope *shared.FamilyScope,
	id uuid.UUID,
	input *UpdateScheduleItemInput,
) error {
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		updates := make(map[string]any)
		if input.Title != nil {
			updates["title"] = *input.Title
		}
		if input.Description != nil {
			updates["description"] = *input.Description
		}
		if input.StudentID != nil {
			updates["student_id"] = *input.StudentID
		}
		if input.StartDate != nil {
			updates["start_date"] = *input.StartDate
		}
		if input.StartTime != nil {
			updates["start_time"] = *input.StartTime
		}
		if input.EndTime != nil {
			updates["end_time"] = *input.EndTime
		}
		if input.DurationMinutes != nil {
			updates["duration_minutes"] = *input.DurationMinutes
		}
		if input.Category != nil {
			updates["category"] = *input.Category
		}
		if input.SubjectID != nil {
			updates["subject_id"] = *input.SubjectID
		}
		if input.Color != nil {
			updates["color"] = *input.Color
		}
		if input.Notes != nil {
			updates["notes"] = *input.Notes
		}
		updates["updated_at"] = time.Now()
		return tx.Model(&ScheduleItem{}).Where("id = ?", id).Updates(updates).Error
	})
}

func (r *PgScheduleItemRepository) MarkCompleted(
	ctx context.Context,
	scope *shared.FamilyScope,
	id uuid.UUID,
	completedAt time.Time,
) error {
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Model(&ScheduleItem{}).Where("id = ?", id).Updates(map[string]any{
			"is_completed": true,
			"completed_at": completedAt,
			"updated_at":   time.Now(),
		}).Error
	})
}

func (r *PgScheduleItemRepository) SetLinkedActivity(
	ctx context.Context,
	scope *shared.FamilyScope,
	id uuid.UUID,
	activityID uuid.UUID,
) error {
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Model(&ScheduleItem{}).Where("id = ?", id).Updates(map[string]any{
			"linked_activity_id": activityID,
			"updated_at":         time.Now(),
		}).Error
	})
}

func (r *PgScheduleItemRepository) Delete(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error {
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).Delete(&ScheduleItem{}).Error
	})
}

func (r *PgScheduleItemRepository) DeleteAllByFamily(ctx context.Context, scope *shared.FamilyScope) error {
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("1=1").Delete(&ScheduleItem{}).Error
	})
}

func (r *PgScheduleItemRepository) ListAllByFamily(ctx context.Context, scope *shared.FamilyScope) ([]ScheduleItem, error) {
	var items []ScheduleItem
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Order("start_date, created_at").Find(&items).Error
	})
	return items, err
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgScheduleTemplateRepository [17-planning §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgScheduleTemplateRepository implements ScheduleTemplateRepository using GORM.
type PgScheduleTemplateRepository struct {
	db *gorm.DB
}

// NewPgScheduleTemplateRepository creates a new PgScheduleTemplateRepository.
func NewPgScheduleTemplateRepository(db *gorm.DB) ScheduleTemplateRepository {
	return &PgScheduleTemplateRepository{db: db}
}

func (r *PgScheduleTemplateRepository) Create(ctx context.Context, scope *shared.FamilyScope, tmpl *ScheduleTemplate) error {
	tmpl.FamilyID = scope.FamilyID()
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(tmpl).Error
	})
}

func (r *PgScheduleTemplateRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ScheduleTemplate, error) {
	var tmpl ScheduleTemplate
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).First(&tmpl).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tmpl, nil
}

func (r *PgScheduleTemplateRepository) ListByFamily(ctx context.Context, scope *shared.FamilyScope) ([]ScheduleTemplate, error) {
	var templates []ScheduleTemplate
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Order("created_at DESC").Find(&templates).Error
	})
	return templates, err
}

func (r *PgScheduleTemplateRepository) Update(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, input *UpdateTemplateInput, itemsJSON []byte) error {
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		updates := make(map[string]any)
		if input.Name != nil {
			updates["name"] = *input.Name
		}
		if input.Description != nil {
			updates["description"] = *input.Description
		}
		if itemsJSON != nil {
			updates["items"] = itemsJSON
		}
		if input.IsActive != nil {
			updates["is_active"] = *input.IsActive
		}
		updates["updated_at"] = time.Now()
		return tx.Model(&ScheduleTemplate{}).Where("id = ?", id).Updates(updates).Error
	})
}

func (r *PgScheduleTemplateRepository) Delete(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error {
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).Delete(&ScheduleTemplate{}).Error
	})
}

func (r *PgScheduleTemplateRepository) DeleteAllByFamily(ctx context.Context, scope *shared.FamilyScope) error {
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("1=1").Delete(&ScheduleTemplate{}).Error
	})
}
