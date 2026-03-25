package safety

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── PgReportRepository ──────────────────────────────────────────────────────

// PgReportRepository implements ReportRepository using PostgreSQL/GORM. [11-safety §6.1]
type PgReportRepository struct {
	db *gorm.DB
}

// NewPgReportRepository constructs a ReportRepository.
func NewPgReportRepository(db *gorm.DB) ReportRepository {
	return &PgReportRepository{db: db}
}

func (r *PgReportRepository) Create(_ context.Context, scope shared.FamilyScope, input CreateReportRow) (*Report, error) {
	report := &Report{
		ReporterFamilyID: scope.FamilyID(),
		ReporterParentID: input.ReporterParentID,
		TargetType:       input.TargetType,
		TargetID:         input.TargetID,
		TargetFamilyID:   input.TargetFamilyID,
		Category:         input.Category,
		Description:      input.Description,
		Priority:         input.Priority,
		Status:           "pending",
	}
	if err := r.db.Create(report).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return report, nil
}

func (r *PgReportRepository) FindByID(_ context.Context, scope shared.FamilyScope, reportID uuid.UUID) (*Report, error) {
	var report Report
	if err := r.db.Where("id = ? AND reporter_family_id = ?", reportID, scope.FamilyID()).First(&report).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &SafetyError{Err: ErrReportNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &report, nil
}

func (r *PgReportRepository) FindByIDUnscoped(_ context.Context, reportID uuid.UUID) (*Report, error) {
	var report Report
	if err := r.db.Where("id = ?", reportID).First(&report).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &SafetyError{Err: ErrReportNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &report, nil
}

func (r *PgReportRepository) ListByReporter(_ context.Context, scope shared.FamilyScope, pagination shared.PaginationParams) ([]Report, error) {
	limit := pagination.EffectiveLimit()
	q := r.db.Where("reporter_family_id = ?", scope.FamilyID()).
		Order("created_at DESC, id DESC").
		Limit(limit + 1)

	if pagination.Cursor != nil {
		cursorID, cursorAt, err := shared.DecodeCursor(*pagination.Cursor)
		if err != nil {
			return nil, err
		}
		q = q.Where("(created_at, id) < (?, ?)", cursorAt, cursorID)
	}

	var reports []Report
	if err := q.Find(&reports).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return reports, nil
}

func (r *PgReportRepository) ListFiltered(_ context.Context, filter ReportFilter, pagination shared.PaginationParams) ([]Report, error) {
	limit := pagination.EffectiveLimit()
	q := r.db.Model(&Report{})
	if filter.Status != nil {
		q = q.Where("status = ?", *filter.Status)
	}
	if filter.Priority != nil {
		q = q.Where("priority = ?", *filter.Priority)
	}
	if filter.Category != nil {
		q = q.Where("category = ?", *filter.Category)
	}
	if filter.AssignedAdminID != nil {
		q = q.Where("assigned_admin_id = ?", *filter.AssignedAdminID)
	}

	q = q.Order("created_at DESC, id DESC").Limit(limit + 1)

	if pagination.Cursor != nil {
		cursorID, cursorAt, err := shared.DecodeCursor(*pagination.Cursor)
		if err != nil {
			return nil, err
		}
		q = q.Where("(created_at, id) < (?, ?)", cursorAt, cursorID)
	}

	var reports []Report
	if err := q.Find(&reports).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return reports, nil
}

func (r *PgReportRepository) Update(_ context.Context, reportID uuid.UUID, updates ReportUpdate) (*Report, error) {
	updateMap := map[string]any{
		"updated_at": time.Now(),
	}
	if updates.Status != nil {
		updateMap["status"] = *updates.Status
	}
	if updates.AssignedAdminID != nil {
		updateMap["assigned_admin_id"] = *updates.AssignedAdminID
	}
	if updates.ResolvedAt != nil {
		updateMap["resolved_at"] = *updates.ResolvedAt
	}

	result := r.db.Model(&Report{}).Where("id = ?", reportID).Updates(updateMap)
	if result.Error != nil {
		return nil, shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, &SafetyError{Err: ErrReportNotFound}
	}

	var report Report
	if err := r.db.Where("id = ?", reportID).First(&report).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &report, nil
}

func (r *PgReportRepository) ExistsRecent(_ context.Context, scope shared.FamilyScope, targetType string, targetID uuid.UUID, withinHours uint32) (bool, error) {
	var count int64
	cutoff := time.Now().Add(-time.Duration(withinHours) * time.Hour)
	if err := r.db.Model(&Report{}).
		Where("reporter_family_id = ? AND target_type = ? AND target_id = ? AND created_at > ?",
			scope.FamilyID(), targetType, targetID, cutoff).
		Count(&count).Error; err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count > 0, nil
}

func (r *PgReportRepository) CountByStatus(_ context.Context, status string) (int64, error) {
	var count int64
	if err := r.db.Model(&Report{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}

func (r *PgReportRepository) CountByStatusAndPriority(_ context.Context, status string, priority string) (int64, error) {
	var count int64
	if err := r.db.Model(&Report{}).Where("status = ? AND priority = ?", status, priority).Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}

func (r *PgReportRepository) CountSince(_ context.Context, since string) (int64, error) {
	var count int64
	if err := r.db.Model(&Report{}).Where("created_at > ?", since).Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}

// ─── PgContentFlagRepository ─────────────────────────────────────────────────

// PgContentFlagRepository implements ContentFlagRepository using PostgreSQL/GORM. [11-safety §6.2]
type PgContentFlagRepository struct {
	db *gorm.DB
}

// NewPgContentFlagRepository constructs a ContentFlagRepository.
func NewPgContentFlagRepository(db *gorm.DB) ContentFlagRepository {
	return &PgContentFlagRepository{db: db}
}

func (r *PgContentFlagRepository) Create(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
	flag := &ContentFlag{
		Source:         input.Source,
		TargetType:     input.TargetType,
		TargetID:       input.TargetID,
		TargetFamilyID: input.TargetFamilyID,
		FlagType:       input.FlagType,
		Confidence:     input.Confidence,
		Labels:         input.Labels,
		ReportID:       input.ReportID,
		AutoRejected:   input.AutoRejected,
	}
	if err := r.db.Create(flag).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return flag, nil
}

func (r *PgContentFlagRepository) FindByID(_ context.Context, flagID uuid.UUID) (*ContentFlag, error) {
	var flag ContentFlag
	if err := r.db.Where("id = ?", flagID).First(&flag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &SafetyError{Err: ErrFlagNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &flag, nil
}

func (r *PgContentFlagRepository) ListFiltered(_ context.Context, filter FlagFilter, pagination shared.PaginationParams) ([]ContentFlag, error) {
	limit := pagination.EffectiveLimit()
	q := r.db.Model(&ContentFlag{})
	if filter.Reviewed != nil {
		q = q.Where("reviewed = ?", *filter.Reviewed)
	}
	if filter.FlagType != nil {
		q = q.Where("flag_type = ?", *filter.FlagType)
	}
	if filter.TargetType != nil {
		q = q.Where("target_type = ?", *filter.TargetType)
	}

	q = q.Order("created_at DESC, id DESC").Limit(limit + 1)

	if pagination.Cursor != nil {
		cursorID, cursorAt, err := shared.DecodeCursor(*pagination.Cursor)
		if err != nil {
			return nil, err
		}
		q = q.Where("(created_at, id) < (?, ?)", cursorAt, cursorID)
	}

	var flags []ContentFlag
	if err := q.Find(&flags).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return flags, nil
}

func (r *PgContentFlagRepository) MarkReviewed(_ context.Context, flagID uuid.UUID, reviewedBy uuid.UUID, actionTaken bool) (*ContentFlag, error) {
	now := time.Now()
	result := r.db.Model(&ContentFlag{}).Where("id = ?", flagID).Updates(map[string]any{
		"reviewed":     true,
		"reviewed_by":  reviewedBy,
		"reviewed_at":  now,
		"action_taken": actionTaken,
	})
	if result.Error != nil {
		return nil, shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, &SafetyError{Err: ErrFlagNotFound}
	}

	var flag ContentFlag
	if err := r.db.Where("id = ?", flagID).First(&flag).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &flag, nil
}

func (r *PgContentFlagRepository) CountUnreviewed(_ context.Context) (int64, error) {
	var count int64
	if err := r.db.Model(&ContentFlag{}).Where("reviewed = false").Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}

// ─── PgModActionRepository ───────────────────────────────────────────────────

// PgModActionRepository implements ModActionRepository using PostgreSQL/GORM. [11-safety §6.3]
type PgModActionRepository struct {
	db *gorm.DB
}

// NewPgModActionRepository constructs a ModActionRepository.
func NewPgModActionRepository(db *gorm.DB) ModActionRepository {
	return &PgModActionRepository{db: db}
}

func (r *PgModActionRepository) Create(_ context.Context, input CreateModActionRow) (*ModAction, error) {
	action := &ModAction{
		AdminID:             input.AdminID,
		TargetFamilyID:      input.TargetFamilyID,
		TargetParentID:      input.TargetParentID,
		ActionType:          input.ActionType,
		Reason:              input.Reason,
		ReportID:            input.ReportID,
		ContentSnapshot:     input.ContentSnapshot,
		SuspensionDays:      input.SuspensionDays,
		SuspensionExpiresAt: input.SuspensionExpiresAt,
	}
	if err := r.db.Create(action).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return action, nil
}

func (r *PgModActionRepository) FindByID(_ context.Context, actionID uuid.UUID) (*ModAction, error) {
	var action ModAction
	if err := r.db.Where("id = ?", actionID).First(&action).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &SafetyError{Err: ErrActionNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &action, nil
}

func (r *PgModActionRepository) ListFiltered(_ context.Context, filter ActionFilter, pagination shared.PaginationParams) ([]ModAction, error) {
	limit := pagination.EffectiveLimit()
	q := r.db.Model(&ModAction{})
	if filter.AdminID != nil {
		q = q.Where("admin_id = ?", *filter.AdminID)
	}
	if filter.TargetFamilyID != nil {
		q = q.Where("target_family_id = ?", *filter.TargetFamilyID)
	}
	if filter.ActionType != nil {
		q = q.Where("action_type = ?", *filter.ActionType)
	}

	q = q.Order("created_at DESC, id DESC").Limit(limit + 1)

	if pagination.Cursor != nil {
		cursorID, cursorAt, err := shared.DecodeCursor(*pagination.Cursor)
		if err != nil {
			return nil, err
		}
		q = q.Where("(created_at, id) < (?, ?)", cursorAt, cursorID)
	}

	var actions []ModAction
	if err := q.Find(&actions).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return actions, nil
}

func (r *PgModActionRepository) ListByTargetFamily(_ context.Context, familyID uuid.UUID, pagination shared.PaginationParams) ([]ModAction, error) {
	limit := pagination.EffectiveLimit()
	q := r.db.Where("target_family_id = ?", familyID).
		Order("created_at DESC, id DESC").
		Limit(limit + 1)

	if pagination.Cursor != nil {
		cursorID, cursorAt, err := shared.DecodeCursor(*pagination.Cursor)
		if err != nil {
			return nil, err
		}
		q = q.Where("(created_at, id) < (?, ?)", cursorAt, cursorID)
	}

	var actions []ModAction
	if err := q.Find(&actions).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return actions, nil
}

func (r *PgModActionRepository) CountSince(_ context.Context, since string) (int64, error) {
	var count int64
	if err := r.db.Model(&ModAction{}).Where("created_at > ?", since).Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}

// ─── PgAccountStatusRepository ───────────────────────────────────────────────

// PgAccountStatusRepository implements AccountStatusRepository using PostgreSQL/GORM. [11-safety §6.4]
type PgAccountStatusRepository struct {
	db *gorm.DB
}

// NewPgAccountStatusRepository constructs an AccountStatusRepository.
func NewPgAccountStatusRepository(db *gorm.DB) AccountStatusRepository {
	return &PgAccountStatusRepository{db: db}
}

func (r *PgAccountStatusRepository) GetOrCreate(_ context.Context, familyID uuid.UUID) (*AccountStatusRow, error) {
	var row AccountStatusRow
	err := r.db.Where("family_id = ?", familyID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = AccountStatusRow{
			FamilyID:  familyID,
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if createErr := r.db.Create(&row).Error; createErr != nil {
			return nil, shared.ErrDatabase(createErr)
		}
		return &row, nil
	}
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &row, nil
}

func (r *PgAccountStatusRepository) Update(_ context.Context, familyID uuid.UUID, updates AccountStatusUpdate) (*AccountStatusRow, error) {
	updateMap := map[string]any{
		"updated_at": time.Now(),
	}
	if updates.Status != nil {
		updateMap["status"] = *updates.Status
	}
	if updates.SuspendedAt != nil {
		updateMap["suspended_at"] = *updates.SuspendedAt
	}
	if updates.SuspensionExpiresAt != nil {
		updateMap["suspension_expires_at"] = *updates.SuspensionExpiresAt
	}
	if updates.SuspensionReason != nil {
		updateMap["suspension_reason"] = *updates.SuspensionReason
	}
	if updates.BannedAt != nil {
		updateMap["banned_at"] = *updates.BannedAt
	}
	if updates.BanReason != nil {
		updateMap["ban_reason"] = *updates.BanReason
	}
	if updates.LastActionID != nil {
		updateMap["last_action_id"] = *updates.LastActionID
	}

	result := r.db.Model(&AccountStatusRow{}).Where("family_id = ?", familyID).Updates(updateMap)
	if result.Error != nil {
		return nil, shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("account status not found for family %s", familyID)
	}

	var row AccountStatusRow
	if err := r.db.Where("family_id = ?", familyID).First(&row).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &row, nil
}

func (r *PgAccountStatusRepository) CountByStatus(_ context.Context, status string) (int64, error) {
	var count int64
	if err := r.db.Model(&AccountStatusRow{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}

// ─── PgAppealRepository ──────────────────────────────────────────────────────

// PgAppealRepository implements AppealRepository using PostgreSQL/GORM. [11-safety §6.5]
type PgAppealRepository struct {
	db *gorm.DB
}

// NewPgAppealRepository constructs an AppealRepository.
func NewPgAppealRepository(db *gorm.DB) AppealRepository {
	return &PgAppealRepository{db: db}
}

func (r *PgAppealRepository) Create(_ context.Context, scope shared.FamilyScope, input CreateAppealRow) (*Appeal, error) {
	appeal := &Appeal{
		FamilyID:   scope.FamilyID(),
		ActionID:   input.ActionID,
		AppealText: input.AppealText,
		Status:     "pending",
	}
	if err := r.db.Create(appeal).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return appeal, nil
}

func (r *PgAppealRepository) FindByID(_ context.Context, scope shared.FamilyScope, appealID uuid.UUID) (*Appeal, error) {
	var appeal Appeal
	if err := r.db.Where("id = ? AND family_id = ?", appealID, scope.FamilyID()).First(&appeal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &SafetyError{Err: ErrAppealNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &appeal, nil
}

func (r *PgAppealRepository) FindByIDUnscoped(_ context.Context, appealID uuid.UUID) (*Appeal, error) {
	var appeal Appeal
	if err := r.db.Where("id = ?", appealID).First(&appeal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &SafetyError{Err: ErrAppealNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &appeal, nil
}

func (r *PgAppealRepository) FindByActionID(_ context.Context, actionID uuid.UUID) (*Appeal, error) {
	var appeal Appeal
	if err := r.db.Where("action_id = ?", actionID).First(&appeal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, shared.ErrDatabase(err)
	}
	return &appeal, nil
}

func (r *PgAppealRepository) ListFiltered(_ context.Context, filter AppealFilter, pagination shared.PaginationParams) ([]Appeal, error) {
	limit := pagination.EffectiveLimit()
	q := r.db.Model(&Appeal{})
	if filter.Status != nil {
		q = q.Where("status = ?", *filter.Status)
	}

	q = q.Order("created_at DESC, id DESC").Limit(limit + 1)

	if pagination.Cursor != nil {
		cursorID, cursorAt, err := shared.DecodeCursor(*pagination.Cursor)
		if err != nil {
			return nil, err
		}
		q = q.Where("(created_at, id) < (?, ?)", cursorAt, cursorID)
	}

	var appeals []Appeal
	if err := q.Find(&appeals).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return appeals, nil
}

func (r *PgAppealRepository) Update(_ context.Context, appealID uuid.UUID, updates AppealUpdate) (*Appeal, error) {
	updateMap := map[string]any{
		"updated_at": time.Now(),
	}
	if updates.Status != nil {
		updateMap["status"] = *updates.Status
	}
	if updates.AssignedAdminID != nil {
		updateMap["assigned_admin_id"] = *updates.AssignedAdminID
	}
	if updates.ResolutionText != nil {
		updateMap["resolution_text"] = *updates.ResolutionText
	}
	if updates.ResolvedAt != nil {
		updateMap["resolved_at"] = *updates.ResolvedAt
	}

	result := r.db.Model(&Appeal{}).Where("id = ?", appealID).Updates(updateMap)
	if result.Error != nil {
		return nil, shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, &SafetyError{Err: ErrAppealNotFound}
	}

	var appeal Appeal
	if err := r.db.Where("id = ?", appealID).First(&appeal).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &appeal, nil
}

func (r *PgAppealRepository) CountByStatus(_ context.Context, status string) (int64, error) {
	var count int64
	if err := r.db.Model(&Appeal{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}

// ─── PgNcmecReportRepository ─────────────────────────────────────────────────

// PgNcmecReportRepository implements NcmecReportRepository using PostgreSQL/GORM. [11-safety §6.6]
type PgNcmecReportRepository struct {
	db *gorm.DB
}

// NewPgNcmecReportRepository constructs a NcmecReportRepository.
func NewPgNcmecReportRepository(db *gorm.DB) NcmecReportRepository {
	return &PgNcmecReportRepository{db: db}
}

func (r *PgNcmecReportRepository) Create(_ context.Context, input CreateNcmecReportRow) (*NcmecReport, error) {
	report := &NcmecReport{
		UploadID:           input.UploadID,
		FamilyID:           input.FamilyID,
		ParentID:           input.ParentID,
		CsamHash:           input.CsamHash,
		Confidence:         input.Confidence,
		MatchedDatabase:    input.MatchedDatabase,
		EvidenceStorageKey: input.EvidenceStorageKey,
		Status:             "pending",
	}
	if err := r.db.Create(report).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return report, nil
}

func (r *PgNcmecReportRepository) FindByID(_ context.Context, reportID uuid.UUID) (*NcmecReport, error) {
	var report NcmecReport
	if err := r.db.Where("id = ?", reportID).First(&report).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &SafetyError{Err: ErrReportNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &report, nil
}

func (r *PgNcmecReportRepository) UpdateStatus(_ context.Context, reportID uuid.UUID, status string, ncmecReportID *string, errMsg *string) (*NcmecReport, error) {
	updateMap := map[string]any{
		"status": status,
	}
	if ncmecReportID != nil {
		updateMap["ncmec_report_id"] = *ncmecReportID
		now := time.Now()
		updateMap["submitted_at"] = now
	}
	if errMsg != nil {
		updateMap["error_message"] = *errMsg
	}

	result := r.db.Model(&NcmecReport{}).Where("id = ?", reportID).Updates(updateMap)
	if result.Error != nil {
		return nil, shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, &SafetyError{Err: ErrReportNotFound}
	}

	var report NcmecReport
	if err := r.db.Where("id = ?", reportID).First(&report).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &report, nil
}

func (r *PgNcmecReportRepository) FindPending(_ context.Context) ([]NcmecReport, error) {
	var reports []NcmecReport
	if err := r.db.Where("status = ?", "pending").Order("created_at ASC").Find(&reports).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return reports, nil
}

// ─── PgBotSignalRepository ───────────────────────────────────────────────────

// PgBotSignalRepository implements BotSignalRepository using PostgreSQL/GORM. [11-safety §6.7]
type PgBotSignalRepository struct {
	db *gorm.DB
}

// NewPgBotSignalRepository constructs a BotSignalRepository.
func NewPgBotSignalRepository(db *gorm.DB) BotSignalRepository {
	return &PgBotSignalRepository{db: db}
}

func (r *PgBotSignalRepository) Create(_ context.Context, input CreateBotSignalRow) (*BotSignal, error) {
	signal := &BotSignal{
		FamilyID:   input.FamilyID,
		ParentID:   input.ParentID,
		SignalType: input.SignalType,
		Details:    input.Details,
	}
	if err := r.db.Create(signal).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return signal, nil
}

func (r *PgBotSignalRepository) CountRecent(_ context.Context, parentID uuid.UUID, withinMinutes uint32) (int64, error) {
	var count int64
	cutoff := time.Now().Add(-time.Duration(withinMinutes) * time.Minute)
	if err := r.db.Model(&BotSignal{}).
		Where("parent_id = ? AND created_at > ?", parentID, cutoff).
		Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}
