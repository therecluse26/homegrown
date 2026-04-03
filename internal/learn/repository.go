package learn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/learn/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 1: Activity Definition Repository (no FamilyScope)
// ═══════════════════════════════════════════════════════════════════════════════

// PgActivityDefRepository implements ActivityDefRepository using GORM.
type PgActivityDefRepository struct {
	db *gorm.DB
}

// NewPgActivityDefRepository creates a new PgActivityDefRepository.
func NewPgActivityDefRepository(db *gorm.DB) ActivityDefRepository {
	return &PgActivityDefRepository{db: db}
}

func (r *PgActivityDefRepository) Create(ctx context.Context, def *ActivityDefModel) error {
	if err := r.db.WithContext(ctx).Create(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgActivityDefRepository) FindByID(ctx context.Context, defID uuid.UUID) (*ActivityDefModel, error) {
	var def ActivityDefModel
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = true", defID).First(&def).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrActivityDefNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &def, nil
}

func (r *PgActivityDefRepository) List(ctx context.Context, query *ActivityDefQuery) ([]ActivityDefModel, error) {
	tx := r.db.WithContext(ctx).Where("is_active = true")

	if query.Subject != nil {
		tx = tx.Where("? = ANY(subject_tags)", *query.Subject)
	}
	if query.MethodologyID != nil {
		tx = tx.Where("methodology_id = ?", *query.MethodologyID)
	}
	if query.PublisherID != nil {
		tx = tx.Where("publisher_id = ?", *query.PublisherID)
	}
	if query.Search != nil {
		tx = tx.Where("title ILIKE ?", fmt.Sprintf("%%%s%%", *query.Search))
	}
	if query.Cursor != nil {
		tx = tx.Where("id < ?", *query.Cursor)
	}

	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var defs []ActivityDefModel
	err := tx.Order("id DESC").Limit(int(limit) + 1).Find(&defs).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return defs, nil
}

func (r *PgActivityDefRepository) Update(ctx context.Context, def *ActivityDefModel) error {
	if err := r.db.WithContext(ctx).Save(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgActivityDefRepository) SoftDelete(ctx context.Context, defID uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&ActivityDefModel{}).
		Where("id = ? AND is_active = true", defID).
		Update("is_active", false)
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return &LearningError{Err: domain.ErrActivityDefNotFound}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 3: Activity Log Repository (FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

// PgActivityLogRepository implements ActivityLogRepository using GORM.
type PgActivityLogRepository struct {
	db *gorm.DB
}

// NewPgActivityLogRepository creates a new PgActivityLogRepository.
func NewPgActivityLogRepository(db *gorm.DB) ActivityLogRepository {
	return &PgActivityLogRepository{db: db}
}

func (r *PgActivityLogRepository) Create(ctx context.Context, scope *shared.FamilyScope, log *ActivityLogModel) error {
	log.FamilyID = scope.FamilyID()
	var err error
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		err = tx.Create(log).Error
		return err
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgActivityLogRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, logID uuid.UUID) (*ActivityLogModel, error) {
	var log ActivityLogModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", logID).First(&log).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrActivityNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &log, nil
}

func (r *PgActivityLogRepository) ListByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query *ActivityLogQuery) ([]ActivityLogModel, error) {
	var logs []ActivityLogModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		q := tx.Where("student_id = ?", studentID)

		if query.Subject != nil {
			q = q.Where("? = ANY(subject_tags)", *query.Subject)
		}
		if query.DateFrom != nil {
			q = q.Where("activity_date >= ?", *query.DateFrom)
		}
		if query.DateTo != nil {
			q = q.Where("activity_date <= ?", *query.DateTo)
		}
		if query.Cursor != nil {
			q = q.Where("id < ?", *query.Cursor)
		}

		limit := query.Limit
		if limit <= 0 || limit > 50 {
			limit = 20
		}

		return q.Order("id DESC").Limit(int(limit) + 1).Find(&logs).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return logs, nil
}

func (r *PgActivityLogRepository) Update(ctx context.Context, scope *shared.FamilyScope, log *ActivityLogModel) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(log).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgActivityLogRepository) Delete(ctx context.Context, scope *shared.FamilyScope, logID uuid.UUID) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		result := tx.Delete(&ActivityLogModel{}, "id = ?", logID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return &LearningError{Err: domain.ErrActivityNotFound}
		}
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgActivityLogRepository) CountByStudentDateRange(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) (int64, error) {
	var count int64
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Model(&ActivityLogModel{}).
			Where("student_id = ? AND activity_date >= ? AND activity_date <= ?", studentID, dateFrom, dateTo).
			Count(&count).Error
	})
	if txErr != nil {
		return 0, shared.ErrDatabase(txErr)
	}
	return count, nil
}

func (r *PgActivityLogRepository) HoursBySubject(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) ([]SubjectHours, error) {
	var results []SubjectHours
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		// Unnest subject_tags and sum duration_minutes grouped by tag.
		return tx.Raw(`
			SELECT tag AS subject_slug, COALESCE(SUM(duration_minutes), 0) AS total_minutes
			FROM learn_activity_logs, unnest(subject_tags) AS tag
			WHERE student_id = ? AND activity_date >= ? AND activity_date <= ?
			  AND duration_minutes IS NOT NULL
			GROUP BY tag
			ORDER BY total_minutes DESC
		`, studentID, dateFrom, dateTo).Scan(&results).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return results, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 1: Reading Item Repository (no FamilyScope)
// ═══════════════════════════════════════════════════════════════════════════════

// PgReadingItemRepository implements ReadingItemRepository using GORM.
type PgReadingItemRepository struct {
	db *gorm.DB
}

// NewPgReadingItemRepository creates a new PgReadingItemRepository.
func NewPgReadingItemRepository(db *gorm.DB) ReadingItemRepository {
	return &PgReadingItemRepository{db: db}
}

func (r *PgReadingItemRepository) Create(ctx context.Context, item *ReadingItemModel) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgReadingItemRepository) FindByID(ctx context.Context, itemID uuid.UUID) (*ReadingItemModel, error) {
	var item ReadingItemModel
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = true", itemID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrReadingItemNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &item, nil
}

func (r *PgReadingItemRepository) List(ctx context.Context, query *ReadingItemQuery) ([]ReadingItemModel, error) {
	tx := r.db.WithContext(ctx).Where("is_active = true")
	if query.Subject != nil {
		tx = tx.Where("? = ANY(subject_tags)", *query.Subject)
	}
	if query.Search != nil {
		tx = tx.Where("title ILIKE ?", fmt.Sprintf("%%%s%%", *query.Search))
	}
	if query.ISBN != nil {
		tx = tx.Where("isbn = ?", *query.ISBN)
	}
	if query.Cursor != nil {
		tx = tx.Where("id < ?", *query.Cursor)
	}
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var items []ReadingItemModel
	if err := tx.Order("id DESC").Limit(int(limit) + 1).Find(&items).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return items, nil
}

func (r *PgReadingItemRepository) Update(ctx context.Context, item *ReadingItemModel) error {
	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgReadingItemRepository) FindByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]ReadingItemModel, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	var items []ReadingItemModel
	if err := r.db.WithContext(ctx).Where("id IN ? AND is_active = true", itemIDs).Find(&items).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return items, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 3: Journal Entry Repository (FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

// PgJournalEntryRepository implements JournalEntryRepository using GORM.
type PgJournalEntryRepository struct {
	db *gorm.DB
}

// NewPgJournalEntryRepository creates a new PgJournalEntryRepository.
func NewPgJournalEntryRepository(db *gorm.DB) JournalEntryRepository {
	return &PgJournalEntryRepository{db: db}
}

func (r *PgJournalEntryRepository) Create(ctx context.Context, scope *shared.FamilyScope, entry *JournalEntryModel) error {
	entry.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(entry).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgJournalEntryRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, entryID uuid.UUID) (*JournalEntryModel, error) {
	var entry JournalEntryModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", entryID).First(&entry).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrJournalNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &entry, nil
}

func (r *PgJournalEntryRepository) ListByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query *JournalEntryQuery) ([]JournalEntryModel, error) {
	var entries []JournalEntryModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		q := tx.Where("student_id = ?", studentID)
		if query.EntryType != nil {
			q = q.Where("entry_type = ?", *query.EntryType)
		}
		if query.DateFrom != nil {
			q = q.Where("entry_date >= ?", *query.DateFrom)
		}
		if query.DateTo != nil {
			q = q.Where("entry_date <= ?", *query.DateTo)
		}
		if query.Search != nil {
			q = q.Where("content ILIKE ?", fmt.Sprintf("%%%s%%", *query.Search))
		}
		if query.Cursor != nil {
			q = q.Where("id < ?", *query.Cursor)
		}
		limit := query.Limit
		if limit <= 0 || limit > 50 {
			limit = 20
		}
		return q.Order("id DESC").Limit(int(limit) + 1).Find(&entries).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return entries, nil
}

func (r *PgJournalEntryRepository) Update(ctx context.Context, scope *shared.FamilyScope, entry *JournalEntryModel) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(entry).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgJournalEntryRepository) Delete(ctx context.Context, scope *shared.FamilyScope, entryID uuid.UUID) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		result := tx.Delete(&JournalEntryModel{}, "id = ?", entryID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return &LearningError{Err: domain.ErrJournalNotFound}
		}
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgJournalEntryRepository) CountByStudentDateRange(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) (int64, error) {
	var count int64
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Model(&JournalEntryModel{}).
			Where("student_id = ? AND entry_date >= ? AND entry_date <= ?", studentID, dateFrom, dateTo).
			Count(&count).Error
	})
	if txErr != nil {
		return 0, shared.ErrDatabase(txErr)
	}
	return count, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 3: Reading Progress Repository (FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

// PgReadingProgressRepository implements ReadingProgressRepository using GORM.
type PgReadingProgressRepository struct {
	db *gorm.DB
}

// NewPgReadingProgressRepository creates a new PgReadingProgressRepository.
func NewPgReadingProgressRepository(db *gorm.DB) ReadingProgressRepository {
	return &PgReadingProgressRepository{db: db}
}

func (r *PgReadingProgressRepository) Create(ctx context.Context, scope *shared.FamilyScope, progress *ReadingProgressModel) error {
	progress.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(progress).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgReadingProgressRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, progressID uuid.UUID) (*ReadingProgressModel, error) {
	var progress ReadingProgressModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", progressID).First(&progress).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrReadingProgressNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &progress, nil
}

func (r *PgReadingProgressRepository) ListByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query *ReadingProgressQuery) ([]ReadingProgressModel, error) {
	var progress []ReadingProgressModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		q := tx.Where("student_id = ?", studentID)
		if query.Status != nil {
			q = q.Where("status = ?", *query.Status)
		}
		if query.Cursor != nil {
			q = q.Where("id < ?", *query.Cursor)
		}
		limit := query.Limit
		if limit <= 0 || limit > 50 {
			limit = 20
		}
		return q.Order("id DESC").Limit(int(limit) + 1).Find(&progress).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return progress, nil
}

func (r *PgReadingProgressRepository) Update(ctx context.Context, scope *shared.FamilyScope, progress *ReadingProgressModel) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(progress).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgReadingProgressRepository) Exists(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, readingItemID uuid.UUID) (bool, error) {
	var count int64
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Model(&ReadingProgressModel{}).
			Where("student_id = ? AND reading_item_id = ?", studentID, readingItemID).
			Count(&count).Error
	})
	if txErr != nil {
		return false, shared.ErrDatabase(txErr)
	}
	return count > 0, nil
}

func (r *PgReadingProgressRepository) CountCompleted(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) (int64, error) {
	var count int64
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Model(&ReadingProgressModel{}).
			Where("student_id = ? AND status = 'completed' AND completed_at >= ? AND completed_at <= ?", studentID, dateFrom, dateTo).
			Count(&count).Error
	})
	if txErr != nil {
		return 0, shared.ErrDatabase(txErr)
	}
	return count, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 3: Reading List Repository (FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

// PgReadingListRepository implements ReadingListRepository using GORM.
type PgReadingListRepository struct {
	db *gorm.DB
}

// NewPgReadingListRepository creates a new PgReadingListRepository.
func NewPgReadingListRepository(db *gorm.DB) ReadingListRepository {
	return &PgReadingListRepository{db: db}
}

func (r *PgReadingListRepository) Create(ctx context.Context, scope *shared.FamilyScope, list *ReadingListModel) error {
	list.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(list).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgReadingListRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) (*ReadingListModel, error) {
	var list ReadingListModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", listID).First(&list).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrReadingListNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &list, nil
}

func (r *PgReadingListRepository) ListByFamily(ctx context.Context, scope *shared.FamilyScope) ([]ReadingListModel, error) {
	var lists []ReadingListModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Order("name ASC").Find(&lists).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return lists, nil
}

func (r *PgReadingListRepository) Update(ctx context.Context, scope *shared.FamilyScope, list *ReadingListModel) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(list).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgReadingListRepository) Delete(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		result := tx.Delete(&ReadingListModel{}, "id = ?", listID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return &LearningError{Err: domain.ErrReadingListNotFound}
		}
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgReadingListRepository) AddItems(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID, itemIDs []uuid.UUID) error {
	if len(itemIDs) == 0 {
		return nil
	}
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		// Get current max sort order.
		var maxOrder int16
		tx.Model(&ReadingListItemModel{}).Where("reading_list_id = ?", listID).
			Select("COALESCE(MAX(sort_order), 0)").Scan(&maxOrder)

		items := make([]ReadingListItemModel, len(itemIDs))
		for i, id := range itemIDs {
			maxOrder++
			items[i] = ReadingListItemModel{
				ReadingListID: listID,
				ReadingItemID: id,
				SortOrder:     maxOrder,
			}
		}
		return tx.Create(&items).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgReadingListRepository) RemoveItems(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID, itemIDs []uuid.UUID) error {
	if len(itemIDs) == 0 {
		return nil
	}
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("reading_list_id = ? AND reading_item_id IN ?", listID, itemIDs).
			Delete(&ReadingListItemModel{}).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgReadingListRepository) ListItems(ctx context.Context, listID uuid.UUID) ([]ReadingListItemModel, error) {
	var items []ReadingListItemModel
	if err := r.db.WithContext(ctx).Where("reading_list_id = ?", listID).
		Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return items, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Platform: Subject Taxonomy Repository (no FamilyScope for platform data)
// ═══════════════════════════════════════════════════════════════════════════════

// PgSubjectTaxonomyRepository implements SubjectTaxonomyRepository using GORM.
type PgSubjectTaxonomyRepository struct {
	db *gorm.DB
}

// NewPgSubjectTaxonomyRepository creates a new PgSubjectTaxonomyRepository.
func NewPgSubjectTaxonomyRepository(db *gorm.DB) SubjectTaxonomyRepository {
	return &PgSubjectTaxonomyRepository{db: db}
}

func (r *PgSubjectTaxonomyRepository) List(ctx context.Context, query *TaxonomyQuery) ([]SubjectTaxonomyModel, error) {
	tx := r.db.WithContext(ctx).Where("is_active = true")
	if query != nil {
		if query.Level != nil {
			tx = tx.Where("level = ?", *query.Level)
		}
		if query.ParentID != nil {
			tx = tx.Where("parent_id = ?", *query.ParentID)
		}
	}
	var nodes []SubjectTaxonomyModel
	if err := tx.Order("display_order ASC").Find(&nodes).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return nodes, nil
}

func (r *PgSubjectTaxonomyRepository) FindBySlug(ctx context.Context, slug string) (*SubjectTaxonomyModel, error) {
	var node SubjectTaxonomyModel
	err := r.db.WithContext(ctx).Where("slug = ? AND is_active = true", slug).First(&node).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, shared.ErrDatabase(err)
	}
	return &node, nil
}

func (r *PgSubjectTaxonomyRepository) ValidateSlugs(ctx context.Context, slugs []string) (bool, error) {
	if len(slugs) == 0 {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&SubjectTaxonomyModel{}).
		Where("slug IN ? AND is_active = true", slugs).
		Count(&count).Error
	if err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count == int64(len(slugs)), nil
}

func (r *PgSubjectTaxonomyRepository) ListCustomSubjects(ctx context.Context, scope *shared.FamilyScope) ([]CustomSubjectModel, error) {
	var subjects []CustomSubjectModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Order("name ASC").Find(&subjects).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return subjects, nil
}

func (r *PgSubjectTaxonomyRepository) CreateCustomSubject(ctx context.Context, scope *shared.FamilyScope, subject *CustomSubjectModel) error {
	subject.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(subject).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 1: Artifact Link Repository (no FamilyScope — polymorphic)
// ═══════════════════════════════════════════════════════════════════════════════

// PgArtifactLinkRepository implements ArtifactLinkRepository using GORM.
type PgArtifactLinkRepository struct {
	db *gorm.DB
}

// NewPgArtifactLinkRepository creates a new PgArtifactLinkRepository.
func NewPgArtifactLinkRepository(db *gorm.DB) ArtifactLinkRepository {
	return &PgArtifactLinkRepository{db: db}
}

func (r *PgArtifactLinkRepository) Create(ctx context.Context, link *ArtifactLinkModel) error {
	if err := r.db.WithContext(ctx).Create(link).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgArtifactLinkRepository) FindByID(ctx context.Context, linkID uuid.UUID) (*ArtifactLinkModel, error) {
	var link ArtifactLinkModel
	err := r.db.WithContext(ctx).Where("id = ?", linkID).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrLinkNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &link, nil
}

func (r *PgArtifactLinkRepository) FindByContent(ctx context.Context, contentType string, contentID uuid.UUID, direction LinkDirection) ([]ArtifactLinkModel, error) {
	var links []ArtifactLinkModel
	tx := r.db.WithContext(ctx)
	switch direction {
	case LinkDirectionSource:
		tx = tx.Where("source_type = ? AND source_id = ?", contentType, contentID)
	case LinkDirectionTarget:
		tx = tx.Where("target_type = ? AND target_id = ?", contentType, contentID)
	case LinkDirectionBoth:
		tx = tx.Where("(source_type = ? AND source_id = ?) OR (target_type = ? AND target_id = ?)",
			contentType, contentID, contentType, contentID)
	}
	if err := tx.Order("created_at DESC").Find(&links).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return links, nil
}

func (r *PgArtifactLinkRepository) Delete(ctx context.Context, linkID uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&ArtifactLinkModel{}, "id = ?", linkID)
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return &LearningError{Err: domain.ErrLinkNotFound}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 3: Progress Repository (FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

// PgProgressRepository implements ProgressRepository using GORM.
type PgProgressRepository struct {
	db *gorm.DB
}

// NewPgProgressRepository creates a new PgProgressRepository.
func NewPgProgressRepository(db *gorm.DB) ProgressRepository {
	return &PgProgressRepository{db: db}
}

func (r *PgProgressRepository) CreateSnapshot(ctx context.Context, scope *shared.FamilyScope, snapshot *ProgressSnapshotModel) error {
	snapshot.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(snapshot).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgProgressRepository) GetLatestSnapshot(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*ProgressSnapshotModel, error) {
	var snap ProgressSnapshotModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("student_id = ?", studentID).Order("snapshot_date DESC").First(&snap).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &snap, nil
}

func (r *PgProgressRepository) ListSnapshots(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) ([]ProgressSnapshotModel, error) {
	var snaps []ProgressSnapshotModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("student_id = ? AND snapshot_date >= ? AND snapshot_date <= ?", studentID, dateFrom, dateTo).
			Order("snapshot_date ASC").Find(&snaps).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return snaps, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 3: Export Repository (FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

// PgExportRepository implements ExportRepository using GORM.
type PgExportRepository struct {
	db *gorm.DB
}

// NewPgExportRepository creates a new PgExportRepository.
func NewPgExportRepository(db *gorm.DB) ExportRepository {
	return &PgExportRepository{db: db}
}

func (r *PgExportRepository) Create(ctx context.Context, scope *shared.FamilyScope, request *ExportRequestModel) error {
	request.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(request).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgExportRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, exportID uuid.UUID) (*ExportRequestModel, error) {
	var req ExportRequestModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", exportID).First(&req).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrExportNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &req, nil
}

func (r *PgExportRepository) HasActiveExport(ctx context.Context, scope *shared.FamilyScope) (bool, error) {
	var count int64
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Model(&ExportRequestModel{}).
			Where("status IN ?", []string{"pending", "processing"}).
			Count(&count).Error
	})
	if txErr != nil {
		return false, shared.ErrDatabase(txErr)
	}
	return count > 0, nil
}

func (r *PgExportRepository) UpdateStatus(ctx context.Context, exportID uuid.UUID, status string, fileURL *string, expiresAt *time.Time, errorMessage *string) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	if fileURL != nil {
		updates["file_url"] = *fileURL
	}
	if expiresAt != nil {
		updates["expires_at"] = *expiresAt
	}
	if errorMessage != nil {
		updates["error_message"] = *errorMessage
	}
	if err := r.db.Model(&ExportRequestModel{}).Where("id = ?", exportID).Updates(updates).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 1: Question Repository (no FamilyScope)
// ═══════════════════════════════════════════════════════════════════════════════

// PgQuestionRepository implements QuestionRepository using GORM.
type PgQuestionRepository struct {
	db *gorm.DB
}

// NewPgQuestionRepository creates a new PgQuestionRepository.
func NewPgQuestionRepository(db *gorm.DB) QuestionRepository {
	return &PgQuestionRepository{db: db}
}

func (r *PgQuestionRepository) Create(ctx context.Context, q *QuestionModel) error {
	if err := r.db.WithContext(ctx).Create(q).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgQuestionRepository) FindByID(ctx context.Context, questionID uuid.UUID) (*QuestionModel, error) {
	var q QuestionModel
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = true", questionID).First(&q).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrQuestionNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &q, nil
}

func (r *PgQuestionRepository) List(ctx context.Context, query *QuestionQuery) ([]QuestionModel, error) {
	tx := r.db.WithContext(ctx).Where("is_active = true")
	if query.PublisherID != nil {
		tx = tx.Where("publisher_id = ?", *query.PublisherID)
	}
	if query.QuestionType != nil {
		tx = tx.Where("question_type = ?", *query.QuestionType)
	}
	if query.Subject != nil {
		tx = tx.Where("? = ANY(subject_tags)", *query.Subject)
	}
	if query.MethodologyID != nil {
		tx = tx.Where("methodology_id = ?", *query.MethodologyID)
	}
	if query.Search != nil {
		tx = tx.Where("content ILIKE ?", fmt.Sprintf("%%%s%%", *query.Search))
	}
	if query.Cursor != nil {
		tx = tx.Where("id < ?", *query.Cursor)
	}
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var questions []QuestionModel
	if err := tx.Order("id DESC").Limit(int(limit) + 1).Find(&questions).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return questions, nil
}

func (r *PgQuestionRepository) Update(ctx context.Context, q *QuestionModel) error {
	if err := r.db.WithContext(ctx).Save(q).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgQuestionRepository) FindByIDs(ctx context.Context, questionIDs []uuid.UUID) ([]QuestionModel, error) {
	if len(questionIDs) == 0 {
		return nil, nil
	}
	var questions []QuestionModel
	if err := r.db.WithContext(ctx).Where("id IN ? AND is_active = true", questionIDs).Find(&questions).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return questions, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 1: Quiz Definition Repository (no FamilyScope)
// ═══════════════════════════════════════════════════════════════════════════════

// PgQuizDefRepository implements QuizDefRepository using GORM.
type PgQuizDefRepository struct {
	db *gorm.DB
}

// NewPgQuizDefRepository creates a new PgQuizDefRepository.
func NewPgQuizDefRepository(db *gorm.DB) QuizDefRepository {
	return &PgQuizDefRepository{db: db}
}

func (r *PgQuizDefRepository) Create(ctx context.Context, def *QuizDefModel) error {
	if err := r.db.WithContext(ctx).Create(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgQuizDefRepository) FindByID(ctx context.Context, quizDefID uuid.UUID) (*QuizDefModel, error) {
	var def QuizDefModel
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = true", quizDefID).First(&def).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrQuizDefNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &def, nil
}

func (r *PgQuizDefRepository) Update(ctx context.Context, def *QuizDefModel) error {
	if err := r.db.WithContext(ctx).Save(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgQuizDefRepository) SetQuestions(ctx context.Context, quizDefID uuid.UUID, questions []QuizQuestionModel) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Remove existing questions.
		if err := tx.Where("quiz_def_id = ?", quizDefID).Delete(&QuizQuestionModel{}).Error; err != nil {
			return err
		}
		if len(questions) == 0 {
			return nil
		}
		return tx.Create(&questions).Error
	})
}

func (r *PgQuizDefRepository) ListQuestions(ctx context.Context, quizDefID uuid.UUID) ([]QuizQuestionModel, error) {
	var questions []QuizQuestionModel
	if err := r.db.WithContext(ctx).Where("quiz_def_id = ?", quizDefID).
		Order("sort_order ASC").Find(&questions).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return questions, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 3: Quiz Session Repository (FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

// PgQuizSessionRepository implements QuizSessionRepository using GORM.
type PgQuizSessionRepository struct {
	db *gorm.DB
}

// NewPgQuizSessionRepository creates a new PgQuizSessionRepository.
func NewPgQuizSessionRepository(db *gorm.DB) QuizSessionRepository {
	return &PgQuizSessionRepository{db: db}
}

func (r *PgQuizSessionRepository) Create(ctx context.Context, scope *shared.FamilyScope, session *QuizSessionModel) error {
	session.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(session).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgQuizSessionRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, sessionID uuid.UUID) (*QuizSessionModel, error) {
	var session QuizSessionModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", sessionID).First(&session).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrQuizSessionNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &session, nil
}

func (r *PgQuizSessionRepository) Update(ctx context.Context, scope *shared.FamilyScope, session *QuizSessionModel) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(session).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 1: Sequence Definition Repository (no FamilyScope)
// ═══════════════════════════════════════════════════════════════════════════════

// PgSequenceDefRepository implements SequenceDefRepository using GORM.
type PgSequenceDefRepository struct {
	db *gorm.DB
}

// NewPgSequenceDefRepository creates a new PgSequenceDefRepository.
func NewPgSequenceDefRepository(db *gorm.DB) SequenceDefRepository {
	return &PgSequenceDefRepository{db: db}
}

func (r *PgSequenceDefRepository) Create(ctx context.Context, def *SequenceDefModel) error {
	if err := r.db.WithContext(ctx).Create(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgSequenceDefRepository) FindByID(ctx context.Context, sequenceDefID uuid.UUID) (*SequenceDefModel, error) {
	var def SequenceDefModel
	if err := r.db.WithContext(ctx).Where("id = ? AND is_active = true", sequenceDefID).First(&def).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrSequenceDefNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &def, nil
}

func (r *PgSequenceDefRepository) Update(ctx context.Context, def *SequenceDefModel) error {
	def.UpdatedAt = time.Now()
	if err := r.db.WithContext(ctx).Save(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgSequenceDefRepository) SetItems(ctx context.Context, sequenceDefID uuid.UUID, items []SequenceItemModel) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing items
		if err := tx.Where("sequence_def_id = ?", sequenceDefID).Delete(&SequenceItemModel{}).Error; err != nil {
			return shared.ErrDatabase(err)
		}
		// Insert new items
		if len(items) > 0 {
			for i := range items {
				items[i].SequenceDefID = sequenceDefID
			}
			if err := tx.Create(&items).Error; err != nil {
				return shared.ErrDatabase(err)
			}
		}
		return nil
	})
}

func (r *PgSequenceDefRepository) ListItems(ctx context.Context, sequenceDefID uuid.UUID) ([]SequenceItemModel, error) {
	var items []SequenceItemModel
	if err := r.db.WithContext(ctx).Where("sequence_def_id = ?", sequenceDefID).
		Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return items, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 1: Video Definition Repository (no FamilyScope)
// ═══════════════════════════════════════════════════════════════════════════════

// PgVideoDefRepository implements VideoDefRepository using GORM.
type PgVideoDefRepository struct {
	db *gorm.DB
}

// NewPgVideoDefRepository creates a new PgVideoDefRepository.
func NewPgVideoDefRepository(db *gorm.DB) VideoDefRepository {
	return &PgVideoDefRepository{db: db}
}

func (r *PgVideoDefRepository) Create(ctx context.Context, def *VideoDefModel) error {
	if err := r.db.WithContext(ctx).Create(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgVideoDefRepository) FindByID(ctx context.Context, defID uuid.UUID) (*VideoDefModel, error) {
	var def VideoDefModel
	if err := r.db.WithContext(ctx).Where("id = ? AND is_active = true", defID).First(&def).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrVideoDefNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &def, nil
}

func (r *PgVideoDefRepository) List(ctx context.Context, query *VideoDefQuery) ([]VideoDefModel, error) {
	tx := r.db.WithContext(ctx).Where("is_active = true")
	if query.Subject != nil {
		tx = tx.Where("? = ANY(subject_tags)", *query.Subject)
	}
	if query.MethodologyID != nil {
		tx = tx.Where("methodology_id = ?", *query.MethodologyID)
	}
	if query.PublisherID != nil {
		tx = tx.Where("publisher_id = ?", *query.PublisherID)
	}
	if query.Search != nil {
		tx = tx.Where("title ILIKE ?", fmt.Sprintf("%%%s%%", *query.Search))
	}
	if query.Cursor != nil {
		tx = tx.Where("id < ?", *query.Cursor)
	}
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var defs []VideoDefModel
	if err := tx.Order("id DESC").Limit(int(limit) + 1).Find(&defs).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return defs, nil
}

func (r *PgVideoDefRepository) Update(ctx context.Context, def *VideoDefModel) error {
	def.UpdatedAt = time.Now()
	if err := r.db.WithContext(ctx).Save(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 3: Sequence Progress Repository (FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

// PgSequenceProgressRepository implements SequenceProgressRepository using GORM.
type PgSequenceProgressRepository struct {
	db *gorm.DB
}

// NewPgSequenceProgressRepository creates a new PgSequenceProgressRepository.
func NewPgSequenceProgressRepository(db *gorm.DB) SequenceProgressRepository {
	return &PgSequenceProgressRepository{db: db}
}

func (r *PgSequenceProgressRepository) Create(ctx context.Context, scope *shared.FamilyScope, progress *SequenceProgressModel) error {
	progress.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(progress).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgSequenceProgressRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, progressID uuid.UUID) (*SequenceProgressModel, error) {
	var progress SequenceProgressModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", progressID).First(&progress).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrSequenceProgressNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &progress, nil
}

func (r *PgSequenceProgressRepository) Update(ctx context.Context, scope *shared.FamilyScope, progress *SequenceProgressModel) error {
	progress.UpdatedAt = time.Now()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(progress).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 3: Assignment Repository (FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

// PgAssignmentRepository implements AssignmentRepository using GORM.
type PgAssignmentRepository struct {
	db *gorm.DB
}

// NewPgAssignmentRepository creates a new PgAssignmentRepository.
func NewPgAssignmentRepository(db *gorm.DB) AssignmentRepository {
	return &PgAssignmentRepository{db: db}
}

func (r *PgAssignmentRepository) Create(ctx context.Context, scope *shared.FamilyScope, assignment *StudentAssignmentModel) error {
	assignment.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(assignment).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgAssignmentRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, assignmentID uuid.UUID) (*StudentAssignmentModel, error) {
	var assignment StudentAssignmentModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", assignmentID).First(&assignment).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrAssignmentNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &assignment, nil
}

func (r *PgAssignmentRepository) ListByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query *AssignmentQuery) ([]StudentAssignmentModel, error) {
	var assignments []StudentAssignmentModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		tx = tx.Where("student_id = ?", studentID)
		if query.Status != nil {
			tx = tx.Where("status = ?", *query.Status)
		}
		if query.DueBefore != nil {
			tx = tx.Where("due_date IS NOT NULL AND due_date <= ?", *query.DueBefore)
		}
		if query.Cursor != nil {
			tx = tx.Where("id < ?", *query.Cursor)
		}
		limit := query.Limit
		if limit <= 0 || limit > 50 {
			limit = 20
		}
		return tx.Order("id DESC").Limit(int(limit) + 1).Find(&assignments).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return assignments, nil
}

func (r *PgAssignmentRepository) Update(ctx context.Context, scope *shared.FamilyScope, assignment *StudentAssignmentModel) error {
	assignment.UpdatedAt = time.Now()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(assignment).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgAssignmentRepository) Delete(ctx context.Context, scope *shared.FamilyScope, assignmentID uuid.UUID) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		result := tx.Where("id = ?", assignmentID).Delete(&StudentAssignmentModel{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return &LearningError{Err: domain.ErrAssignmentNotFound}
		}
		return shared.ErrDatabase(txErr)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Layer 3: Video Progress Repository (FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

// PgVideoProgressRepository implements VideoProgressRepository using GORM.
type PgVideoProgressRepository struct {
	db *gorm.DB
}

// NewPgVideoProgressRepository creates a new PgVideoProgressRepository.
func NewPgVideoProgressRepository(db *gorm.DB) VideoProgressRepository {
	return &PgVideoProgressRepository{db: db}
}

func (r *PgVideoProgressRepository) Upsert(ctx context.Context, scope *shared.FamilyScope, progress *VideoProgressModel) error {
	progress.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(progress).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgVideoProgressRepository) FindByStudentAndVideo(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, videoDefID uuid.UUID) (*VideoProgressModel, error) {
	var progress VideoProgressModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("student_id = ? AND video_def_id = ?", studentID, videoDefID).First(&progress).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrVideoProgressNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &progress, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Assessment Def Repository (Layer 1 — no FamilyScope)
// ═══════════════════════════════════════════════════════════════════════════════

type PgAssessmentDefRepository struct {
	db *gorm.DB
}

func NewPgAssessmentDefRepository(db *gorm.DB) AssessmentDefRepository {
	return &PgAssessmentDefRepository{db: db}
}

func (r *PgAssessmentDefRepository) Create(ctx context.Context, def *AssessmentDefModel) error {
	if err := r.db.WithContext(ctx).Create(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgAssessmentDefRepository) FindByID(ctx context.Context, defID uuid.UUID) (*AssessmentDefModel, error) {
	var def AssessmentDefModel
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = true", defID).First(&def).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrAssessmentDefNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &def, nil
}

func (r *PgAssessmentDefRepository) List(ctx context.Context, query *AssessmentDefQuery) ([]AssessmentDefModel, error) {
	tx := r.db.WithContext(ctx).Where("is_active = true")

	if query.Subject != nil {
		tx = tx.Where("? = ANY(subject_tags)", *query.Subject)
	}
	if query.ScoringType != nil {
		tx = tx.Where("scoring_type = ?", *query.ScoringType)
	}
	if query.PublisherID != nil {
		tx = tx.Where("publisher_id = ?", *query.PublisherID)
	}
	if query.Search != nil {
		tx = tx.Where("title ILIKE ?", "%"+*query.Search+"%")
	}
	if query.Cursor != nil {
		tx = tx.Where("id > ?", *query.Cursor)
	}

	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	tx = tx.Order("id ASC").Limit(int(limit))

	var defs []AssessmentDefModel
	if err := tx.Find(&defs).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return defs, nil
}

func (r *PgAssessmentDefRepository) Update(ctx context.Context, def *AssessmentDefModel) error {
	if err := r.db.WithContext(ctx).Save(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgAssessmentDefRepository) SoftDelete(ctx context.Context, defID uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&AssessmentDefModel{}).Where("id = ?", defID).Update("is_active", false)
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return &LearningError{Err: domain.ErrAssessmentDefNotFound}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Project Def Repository (Layer 1 — no FamilyScope)
// ═══════════════════════════════════════════════════════════════════════════════

type PgProjectDefRepository struct {
	db *gorm.DB
}

func NewPgProjectDefRepository(db *gorm.DB) ProjectDefRepository {
	return &PgProjectDefRepository{db: db}
}

func (r *PgProjectDefRepository) Create(ctx context.Context, def *ProjectDefModel) error {
	if err := r.db.WithContext(ctx).Create(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgProjectDefRepository) FindByID(ctx context.Context, defID uuid.UUID) (*ProjectDefModel, error) {
	var def ProjectDefModel
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = true", defID).First(&def).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrProjectDefNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &def, nil
}

func (r *PgProjectDefRepository) List(ctx context.Context, query *ProjectDefQuery) ([]ProjectDefModel, error) {
	tx := r.db.WithContext(ctx).Where("is_active = true")

	if query.Subject != nil {
		tx = tx.Where("? = ANY(subject_tags)", *query.Subject)
	}
	if query.PublisherID != nil {
		tx = tx.Where("publisher_id = ?", *query.PublisherID)
	}
	if query.Search != nil {
		tx = tx.Where("title ILIKE ?", "%"+*query.Search+"%")
	}
	if query.Cursor != nil {
		tx = tx.Where("id > ?", *query.Cursor)
	}

	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	tx = tx.Order("id ASC").Limit(int(limit))

	var defs []ProjectDefModel
	if err := tx.Find(&defs).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return defs, nil
}

func (r *PgProjectDefRepository) Update(ctx context.Context, def *ProjectDefModel) error {
	if err := r.db.WithContext(ctx).Save(def).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgProjectDefRepository) SoftDelete(ctx context.Context, defID uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&ProjectDefModel{}).Where("id = ?", defID).Update("is_active", false)
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return &LearningError{Err: domain.ErrProjectDefNotFound}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Assessment Result Repository (Layer 3 — FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

type PgAssessmentResultRepository struct {
	db *gorm.DB
}

func NewPgAssessmentResultRepository(db *gorm.DB) AssessmentResultRepository {
	return &PgAssessmentResultRepository{db: db}
}

func (r *PgAssessmentResultRepository) Create(ctx context.Context, scope *shared.FamilyScope, result *AssessmentResultModel) error {
	result.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(result).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgAssessmentResultRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, resultID uuid.UUID) (*AssessmentResultModel, error) {
	var result AssessmentResultModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", resultID).First(&result).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrAssessmentResultNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &result, nil
}

func (r *PgAssessmentResultRepository) ListByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query *AssessmentResultQuery) ([]AssessmentResultModel, error) {
	var results []AssessmentResultModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		tx = tx.Where("student_id = ?", studentID)

		if query.AssessmentDefID != nil {
			tx = tx.Where("assessment_def_id = ?", *query.AssessmentDefID)
		}
		if query.DateFrom != nil {
			tx = tx.Where("assessment_date >= ?", *query.DateFrom)
		}
		if query.DateTo != nil {
			tx = tx.Where("assessment_date <= ?", *query.DateTo)
		}
		if query.Cursor != nil {
			tx = tx.Where("id > ?", *query.Cursor)
		}

		limit := query.Limit
		if limit <= 0 || limit > 100 {
			limit = 25
		}

		return tx.Order("assessment_date DESC, id ASC").Limit(int(limit)).Find(&results).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return results, nil
}

func (r *PgAssessmentResultRepository) Update(ctx context.Context, scope *shared.FamilyScope, result *AssessmentResultModel) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(result).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgAssessmentResultRepository) Delete(ctx context.Context, scope *shared.FamilyScope, resultID uuid.UUID) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		result := tx.Where("id = ?", resultID).Delete(&AssessmentResultModel{})
		if result.RowsAffected == 0 {
			return domain.ErrAssessmentResultNotFound
		}
		return result.Error
	})
	if txErr != nil {
		if errors.Is(txErr, domain.ErrAssessmentResultNotFound) {
			return &LearningError{Err: domain.ErrAssessmentResultNotFound}
		}
		return shared.ErrDatabase(txErr)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Project Progress Repository (Layer 3 — FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

type PgProjectProgressRepository struct {
	db *gorm.DB
}

func NewPgProjectProgressRepository(db *gorm.DB) ProjectProgressRepository {
	return &PgProjectProgressRepository{db: db}
}

func (r *PgProjectProgressRepository) Create(ctx context.Context, scope *shared.FamilyScope, progress *ProjectProgressModel) error {
	progress.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(progress).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgProjectProgressRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, progressID uuid.UUID) (*ProjectProgressModel, error) {
	var progress ProjectProgressModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", progressID).First(&progress).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrProjectProgressNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &progress, nil
}

func (r *PgProjectProgressRepository) ListByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query *ProjectProgressQuery) ([]ProjectProgressModel, error) {
	var results []ProjectProgressModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		tx = tx.Where("student_id = ?", studentID)

		if query.Status != nil {
			tx = tx.Where("status = ?", *query.Status)
		}
		if query.ProjectDefID != nil {
			tx = tx.Where("project_def_id = ?", *query.ProjectDefID)
		}
		if query.Cursor != nil {
			tx = tx.Where("id > ?", *query.Cursor)
		}

		limit := query.Limit
		if limit <= 0 || limit > 100 {
			limit = 25
		}

		return tx.Order("created_at DESC, id ASC").Limit(int(limit)).Find(&results).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return results, nil
}

func (r *PgProjectProgressRepository) Update(ctx context.Context, scope *shared.FamilyScope, progress *ProjectProgressModel) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(progress).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgProjectProgressRepository) Delete(ctx context.Context, scope *shared.FamilyScope, progressID uuid.UUID) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		result := tx.Where("id = ?", progressID).Delete(&ProjectProgressModel{})
		if result.RowsAffected == 0 {
			return domain.ErrProjectProgressNotFound
		}
		return result.Error
	})
	if txErr != nil {
		if errors.Is(txErr, domain.ErrProjectProgressNotFound) {
			return &LearningError{Err: domain.ErrProjectProgressNotFound}
		}
		return shared.ErrDatabase(txErr)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Grading Scale Repository (Layer 3 — FamilyScope required)
// ═══════════════════════════════════════════════════════════════════════════════

type PgGradingScaleRepository struct {
	db *gorm.DB
}

func NewPgGradingScaleRepository(db *gorm.DB) GradingScaleRepository {
	return &PgGradingScaleRepository{db: db}
}

func (r *PgGradingScaleRepository) Create(ctx context.Context, scope *shared.FamilyScope, scale *GradingScaleModel) error {
	scale.FamilyID = scope.FamilyID()
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(scale).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgGradingScaleRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, scaleID uuid.UUID) (*GradingScaleModel, error) {
	var scale GradingScaleModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", scaleID).First(&scale).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, &LearningError{Err: domain.ErrGradingScaleNotFound}
		}
		return nil, shared.ErrDatabase(txErr)
	}
	return &scale, nil
}

func (r *PgGradingScaleRepository) ListByFamily(ctx context.Context, scope *shared.FamilyScope) ([]GradingScaleModel, error) {
	var scales []GradingScaleModel
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Order("is_default DESC, name ASC").Find(&scales).Error
	})
	if txErr != nil {
		return nil, shared.ErrDatabase(txErr)
	}
	return scales, nil
}

func (r *PgGradingScaleRepository) Update(ctx context.Context, scope *shared.FamilyScope, scale *GradingScaleModel) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Save(scale).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgGradingScaleRepository) Delete(ctx context.Context, scope *shared.FamilyScope, scaleID uuid.UUID) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		result := tx.Where("id = ?", scaleID).Delete(&GradingScaleModel{})
		if result.RowsAffected == 0 {
			return domain.ErrGradingScaleNotFound
		}
		return result.Error
	})
	if txErr != nil {
		if errors.Is(txErr, domain.ErrGradingScaleNotFound) {
			return &LearningError{Err: domain.ErrGradingScaleNotFound}
		}
		return shared.ErrDatabase(txErr)
	}
	return nil
}

func (r *PgGradingScaleRepository) ClearDefault(ctx context.Context, scope *shared.FamilyScope) error {
	txErr := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Model(&GradingScaleModel{}).Where("is_default = true").Update("is_default", false).Error
	})
	if txErr != nil {
		return shared.ErrDatabase(txErr)
	}
	return nil
}
