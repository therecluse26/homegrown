package comply

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// State Config Repository [14-comply §6]
// NOT family-scoped — platform-authored reference data.
// ═══════════════════════════════════════════════════════════════════════════════

// PgStateConfigRepository implements StateConfigRepository using GORM/PostgreSQL.
type PgStateConfigRepository struct {
	db *gorm.DB
}

// NewPgStateConfigRepository creates a new PgStateConfigRepository.
func NewPgStateConfigRepository(db *gorm.DB) *PgStateConfigRepository {
	return &PgStateConfigRepository{db: db}
}

func (r *PgStateConfigRepository) ListAll(ctx context.Context) ([]ComplyStateConfig, error) {
	type row struct {
		StateCode               string    `gorm:"column:state_code"`
		StateName               string    `gorm:"column:state_name"`
		NotificationRequired    bool      `gorm:"column:notification_required"`
		NotificationDetails     *string   `gorm:"column:notification_details"`
		RequiredSubjects        string    `gorm:"column:required_subjects"`
		AssessmentRequired      bool      `gorm:"column:assessment_required"`
		AssessmentDetails       *string   `gorm:"column:assessment_details"`
		RecordKeepingRequired   bool      `gorm:"column:record_keeping_required"`
		RecordKeepingDetails    *string   `gorm:"column:record_keeping_details"`
		AttendanceRequired      bool      `gorm:"column:attendance_required"`
		AttendanceDays          *int16    `gorm:"column:attendance_days"`
		AttendanceHours         *int16    `gorm:"column:attendance_hours"`
		AttendanceDetails       *string   `gorm:"column:attendance_details"`
		UmbrellaSchoolAvailable bool      `gorm:"column:umbrella_school_available"`
		UmbrellaSchoolDetails   *string   `gorm:"column:umbrella_school_details"`
		RegulationLevel         string    `gorm:"column:regulation_level"`
		SyncedAt                time.Time `gorm:"column:synced_at"`
		CreatedAt               time.Time `gorm:"column:created_at"`
		UpdatedAt               time.Time `gorm:"column:updated_at"`
	}

	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT state_code, state_name, notification_required, notification_details,
		       required_subjects::text, assessment_required, assessment_details,
		       record_keeping_required, record_keeping_details,
		       attendance_required, attendance_days, attendance_hours, attendance_details,
		       umbrella_school_available, umbrella_school_details,
		       regulation_level, synced_at, created_at, updated_at
		FROM comply_state_configs
		ORDER BY state_name`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("comply state config list all: %w", err)
	}

	out := make([]ComplyStateConfig, len(rows))
	for i, r := range rows {
		out[i] = ComplyStateConfig{
			StateCode:               r.StateCode,
			StateName:               r.StateName,
			NotificationRequired:    r.NotificationRequired,
			NotificationDetails:     r.NotificationDetails,
			RequiredSubjects:        parsePostgresTextArray(r.RequiredSubjects),
			AssessmentRequired:      r.AssessmentRequired,
			AssessmentDetails:       r.AssessmentDetails,
			RecordKeepingRequired:   r.RecordKeepingRequired,
			RecordKeepingDetails:    r.RecordKeepingDetails,
			AttendanceRequired:      r.AttendanceRequired,
			AttendanceDays:          r.AttendanceDays,
			AttendanceHours:         r.AttendanceHours,
			AttendanceDetails:       r.AttendanceDetails,
			UmbrellaSchoolAvailable: r.UmbrellaSchoolAvailable,
			UmbrellaSchoolDetails:   r.UmbrellaSchoolDetails,
			RegulationLevel:         r.RegulationLevel,
			SyncedAt:                r.SyncedAt,
			CreatedAt:               r.CreatedAt,
			UpdatedAt:               r.UpdatedAt,
		}
	}
	return out, nil
}

func (r *PgStateConfigRepository) FindByStateCode(ctx context.Context, stateCode string) (*ComplyStateConfig, error) {
	type row struct {
		StateCode               string    `gorm:"column:state_code"`
		StateName               string    `gorm:"column:state_name"`
		NotificationRequired    bool      `gorm:"column:notification_required"`
		NotificationDetails     *string   `gorm:"column:notification_details"`
		RequiredSubjects        string    `gorm:"column:required_subjects"`
		AssessmentRequired      bool      `gorm:"column:assessment_required"`
		AssessmentDetails       *string   `gorm:"column:assessment_details"`
		RecordKeepingRequired   bool      `gorm:"column:record_keeping_required"`
		RecordKeepingDetails    *string   `gorm:"column:record_keeping_details"`
		AttendanceRequired      bool      `gorm:"column:attendance_required"`
		AttendanceDays          *int16    `gorm:"column:attendance_days"`
		AttendanceHours         *int16    `gorm:"column:attendance_hours"`
		AttendanceDetails       *string   `gorm:"column:attendance_details"`
		UmbrellaSchoolAvailable bool      `gorm:"column:umbrella_school_available"`
		UmbrellaSchoolDetails   *string   `gorm:"column:umbrella_school_details"`
		RegulationLevel         string    `gorm:"column:regulation_level"`
		SyncedAt                time.Time `gorm:"column:synced_at"`
		CreatedAt               time.Time `gorm:"column:created_at"`
		UpdatedAt               time.Time `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT state_code, state_name, notification_required, notification_details,
		       required_subjects::text, assessment_required, assessment_details,
		       record_keeping_required, record_keeping_details,
		       attendance_required, attendance_days, attendance_hours, attendance_details,
		       umbrella_school_available, umbrella_school_details,
		       regulation_level, synced_at, created_at, updated_at
		FROM comply_state_configs
		WHERE state_code = ?`, stateCode).First(&r_).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("comply state config find: %w", err)
	}

	return &ComplyStateConfig{
		StateCode:               r_.StateCode,
		StateName:               r_.StateName,
		NotificationRequired:    r_.NotificationRequired,
		NotificationDetails:     r_.NotificationDetails,
		RequiredSubjects:        parsePostgresTextArray(r_.RequiredSubjects),
		AssessmentRequired:      r_.AssessmentRequired,
		AssessmentDetails:       r_.AssessmentDetails,
		RecordKeepingRequired:   r_.RecordKeepingRequired,
		RecordKeepingDetails:    r_.RecordKeepingDetails,
		AttendanceRequired:      r_.AttendanceRequired,
		AttendanceDays:          r_.AttendanceDays,
		AttendanceHours:         r_.AttendanceHours,
		AttendanceDetails:       r_.AttendanceDetails,
		UmbrellaSchoolAvailable: r_.UmbrellaSchoolAvailable,
		UmbrellaSchoolDetails:   r_.UmbrellaSchoolDetails,
		RegulationLevel:         r_.RegulationLevel,
		SyncedAt:                r_.SyncedAt,
		CreatedAt:               r_.CreatedAt,
		UpdatedAt:               r_.UpdatedAt,
	}, nil
}

func (r *PgStateConfigRepository) Upsert(ctx context.Context, config UpsertStateConfigRow) (*ComplyStateConfig, error) {
	subjectsJSON, _ := json.Marshal(config.RequiredSubjects)
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO comply_state_configs
			(state_code, state_name, notification_required, notification_details,
			 required_subjects, assessment_required, assessment_details,
			 record_keeping_required, record_keeping_details,
			 attendance_required, attendance_days, attendance_hours, attendance_details,
			 umbrella_school_available, umbrella_school_details,
			 regulation_level, synced_at)
		VALUES (?, ?, ?, ?, ?::text[], ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())
		ON CONFLICT (state_code) DO UPDATE SET
			state_name = EXCLUDED.state_name,
			notification_required = EXCLUDED.notification_required,
			notification_details = EXCLUDED.notification_details,
			required_subjects = EXCLUDED.required_subjects,
			assessment_required = EXCLUDED.assessment_required,
			assessment_details = EXCLUDED.assessment_details,
			record_keeping_required = EXCLUDED.record_keeping_required,
			record_keeping_details = EXCLUDED.record_keeping_details,
			attendance_required = EXCLUDED.attendance_required,
			attendance_days = EXCLUDED.attendance_days,
			attendance_hours = EXCLUDED.attendance_hours,
			attendance_details = EXCLUDED.attendance_details,
			umbrella_school_available = EXCLUDED.umbrella_school_available,
			umbrella_school_details = EXCLUDED.umbrella_school_details,
			regulation_level = EXCLUDED.regulation_level,
			synced_at = EXCLUDED.synced_at,
			updated_at = now()`,
		config.StateCode, config.StateName,
		config.NotificationRequired, config.NotificationDetails,
		string(subjectsJSON),
		config.AssessmentRequired, config.AssessmentDetails,
		config.RecordKeepingRequired, config.RecordKeepingDetails,
		config.AttendanceRequired, config.AttendanceDays, config.AttendanceHours, config.AttendanceDetails,
		config.UmbrellaSchoolAvailable, config.UmbrellaSchoolDetails,
		config.RegulationLevel,
	).Error
	if err != nil {
		return nil, fmt.Errorf("comply state config upsert: %w", err)
	}

	return r.FindByStateCode(ctx, config.StateCode)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Family Config Repository [14-comply §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgFamilyConfigRepository implements FamilyConfigRepository using GORM/PostgreSQL.
type PgFamilyConfigRepository struct {
	db *gorm.DB
}

// NewPgFamilyConfigRepository creates a new PgFamilyConfigRepository.
func NewPgFamilyConfigRepository(db *gorm.DB) *PgFamilyConfigRepository {
	return &PgFamilyConfigRepository{db: db}
}

func (r *PgFamilyConfigRepository) Upsert(ctx context.Context, scope shared.FamilyScope, input UpsertFamilyConfigRow) (*ComplyFamilyConfig, error) {
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO comply_family_configs
			(family_id, state_code, school_year_start, school_year_end, total_school_days,
			 custom_schedule_id, gpa_scale, gpa_custom_config)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (family_id) DO UPDATE SET
			state_code = EXCLUDED.state_code,
			school_year_start = EXCLUDED.school_year_start,
			school_year_end = EXCLUDED.school_year_end,
			total_school_days = EXCLUDED.total_school_days,
			custom_schedule_id = EXCLUDED.custom_schedule_id,
			gpa_scale = EXCLUDED.gpa_scale,
			gpa_custom_config = EXCLUDED.gpa_custom_config,
			updated_at = now()`,
		scope.FamilyID(), input.StateCode,
		input.SchoolYearStart.Format("2006-01-02"), input.SchoolYearEnd.Format("2006-01-02"),
		input.TotalSchoolDays, input.CustomScheduleID, input.GpaScale, input.GpaCustomConfig,
	).Error
	if err != nil {
		return nil, fmt.Errorf("comply family config upsert: %w", err)
	}

	return r.FindByFamily(ctx, scope)
}

func (r *PgFamilyConfigRepository) FindByFamily(ctx context.Context, scope shared.FamilyScope) (*ComplyFamilyConfig, error) {
	type row struct {
		FamilyID         uuid.UUID       `gorm:"column:family_id"`
		StateCode        string          `gorm:"column:state_code"`
		SchoolYearStart  time.Time       `gorm:"column:school_year_start"`
		SchoolYearEnd    time.Time       `gorm:"column:school_year_end"`
		TotalSchoolDays  int16           `gorm:"column:total_school_days"`
		CustomScheduleID *uuid.UUID      `gorm:"column:custom_schedule_id"`
		GpaScale         string          `gorm:"column:gpa_scale"`
		GpaCustomConfig  json.RawMessage `gorm:"column:gpa_custom_config"`
		CreatedAt        time.Time       `gorm:"column:created_at"`
		UpdatedAt        time.Time       `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT family_id, state_code, school_year_start, school_year_end, total_school_days,
		       custom_schedule_id, gpa_scale, gpa_custom_config, created_at, updated_at
		FROM comply_family_configs
		WHERE family_id = ?`, scope.FamilyID()).First(&r_).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("comply family config find: %w", err)
	}

	return &ComplyFamilyConfig{
		FamilyID:         r_.FamilyID,
		StateCode:        r_.StateCode,
		SchoolYearStart:  r_.SchoolYearStart,
		SchoolYearEnd:    r_.SchoolYearEnd,
		TotalSchoolDays:  r_.TotalSchoolDays,
		CustomScheduleID: r_.CustomScheduleID,
		GpaScale:         r_.GpaScale,
		GpaCustomConfig:  r_.GpaCustomConfig,
		CreatedAt:        r_.CreatedAt,
		UpdatedAt:        r_.UpdatedAt,
	}, nil
}

func (r *PgFamilyConfigRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_family_configs WHERE family_id = ?`, familyID,
	).Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Schedule Repository [14-comply §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgScheduleRepository implements ScheduleRepository using GORM/PostgreSQL.
type PgScheduleRepository struct {
	db *gorm.DB
}

// NewPgScheduleRepository creates a new PgScheduleRepository.
func NewPgScheduleRepository(db *gorm.DB) *PgScheduleRepository {
	return &PgScheduleRepository{db: db}
}

func (r *PgScheduleRepository) Create(ctx context.Context, scope shared.FamilyScope, input CreateScheduleRow) (*ComplyCustomSchedule, error) {
	id := uuid.New()
	schoolDaysJSON, _ := json.Marshal(input.SchoolDays)
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO comply_custom_schedules (id, family_id, name, school_days, exclusion_periods)
		VALUES (?, ?, ?, ?::boolean[], ?)`,
		id, scope.FamilyID(), input.Name, string(schoolDaysJSON), input.ExclusionPeriods,
	).Error
	if err != nil {
		return nil, fmt.Errorf("comply schedule create: %w", err)
	}

	return r.FindByID(ctx, id, scope)
}

func (r *PgScheduleRepository) FindByID(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) (*ComplyCustomSchedule, error) {
	type row struct {
		ID               uuid.UUID       `gorm:"column:id"`
		FamilyID         uuid.UUID       `gorm:"column:family_id"`
		Name             string          `gorm:"column:name"`
		SchoolDays       string          `gorm:"column:school_days"`
		ExclusionPeriods json.RawMessage `gorm:"column:exclusion_periods"`
		CreatedAt        time.Time       `gorm:"column:created_at"`
		UpdatedAt        time.Time       `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, name, school_days::text, exclusion_periods, created_at, updated_at
		FROM comply_custom_schedules
		WHERE id = ? AND family_id = ?`, scheduleID, scope.FamilyID()).First(&r_).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("comply schedule find: %w", err)
	}

	return &ComplyCustomSchedule{
		ID:               r_.ID,
		FamilyID:         r_.FamilyID,
		Name:             r_.Name,
		SchoolDays:       parsePostgresBoolArray(r_.SchoolDays),
		ExclusionPeriods: r_.ExclusionPeriods,
		CreatedAt:        r_.CreatedAt,
		UpdatedAt:        r_.UpdatedAt,
	}, nil
}

func (r *PgScheduleRepository) ListByFamily(ctx context.Context, scope shared.FamilyScope) ([]ComplyCustomSchedule, error) {
	type row struct {
		ID               uuid.UUID       `gorm:"column:id"`
		FamilyID         uuid.UUID       `gorm:"column:family_id"`
		Name             string          `gorm:"column:name"`
		SchoolDays       string          `gorm:"column:school_days"`
		ExclusionPeriods json.RawMessage `gorm:"column:exclusion_periods"`
		CreatedAt        time.Time       `gorm:"column:created_at"`
		UpdatedAt        time.Time       `gorm:"column:updated_at"`
	}

	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, name, school_days::text, exclusion_periods, created_at, updated_at
		FROM comply_custom_schedules
		WHERE family_id = ?
		ORDER BY name`, scope.FamilyID()).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("comply schedule list: %w", err)
	}

	out := make([]ComplyCustomSchedule, len(rows))
	for i, r := range rows {
		out[i] = ComplyCustomSchedule{
			ID:               r.ID,
			FamilyID:         r.FamilyID,
			Name:             r.Name,
			SchoolDays:       parsePostgresBoolArray(r.SchoolDays),
			ExclusionPeriods: r.ExclusionPeriods,
			CreatedAt:        r.CreatedAt,
			UpdatedAt:        r.UpdatedAt,
		}
	}
	return out, nil
}

func (r *PgScheduleRepository) Update(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope, updates UpdateScheduleRow) (*ComplyCustomSchedule, error) {
	sets := "updated_at = now()"
	args := []any{}
	if updates.Name != nil {
		sets += ", name = ?"
		args = append(args, *updates.Name)
	}
	if updates.SchoolDays != nil {
		daysJSON, _ := json.Marshal(*updates.SchoolDays)
		sets += ", school_days = ?::boolean[]"
		args = append(args, string(daysJSON))
	}
	if updates.ExclusionPeriods != nil {
		sets += ", exclusion_periods = ?"
		args = append(args, *updates.ExclusionPeriods)
	}
	args = append(args, scheduleID, scope.FamilyID())

	tx := r.db.WithContext(ctx).Exec(
		fmt.Sprintf("UPDATE comply_custom_schedules SET %s WHERE id = ? AND family_id = ?", sets),
		args...,
	)
	if tx.Error != nil {
		return nil, fmt.Errorf("comply schedule update: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}

	return r.FindByID(ctx, scheduleID, scope)
}

func (r *PgScheduleRepository) Delete(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) error {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_custom_schedules WHERE id = ? AND family_id = ?`,
		scheduleID, scope.FamilyID(),
	)
	if tx.Error != nil {
		return fmt.Errorf("comply schedule delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrScheduleNotFound
	}
	return nil
}

func (r *PgScheduleRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_custom_schedules WHERE family_id = ?`, familyID,
	).Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Attendance Repository [14-comply §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgAttendanceRepository implements AttendanceRepository using GORM/PostgreSQL.
type PgAttendanceRepository struct {
	db *gorm.DB
}

// NewPgAttendanceRepository creates a new PgAttendanceRepository.
func NewPgAttendanceRepository(db *gorm.DB) *PgAttendanceRepository {
	return &PgAttendanceRepository{db: db}
}

func (r *PgAttendanceRepository) Upsert(ctx context.Context, scope shared.FamilyScope, input UpsertAttendanceRow) (*ComplyAttendance, error) {
	type row struct {
		ID uuid.UUID `gorm:"column:id"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO comply_attendance
			(family_id, student_id, attendance_date, status, duration_minutes, notes, is_auto, manual_override)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (family_id, student_id, attendance_date)
		DO UPDATE SET
			status = EXCLUDED.status,
			duration_minutes = EXCLUDED.duration_minutes,
			notes = EXCLUDED.notes,
			is_auto = EXCLUDED.is_auto,
			manual_override = EXCLUDED.manual_override,
			updated_at = now()
		RETURNING id`,
		scope.FamilyID(), input.StudentID,
		input.AttendanceDate.Format("2006-01-02"),
		input.Status, input.DurationMinutes, input.Notes,
		input.IsAuto, input.ManualOverride,
	).First(&r_).Error
	if err != nil {
		return nil, fmt.Errorf("comply attendance upsert: %w", err)
	}

	return r.FindByID(ctx, r_.ID, scope)
}

func (r *PgAttendanceRepository) FindByID(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope) (*ComplyAttendance, error) {
	type row struct {
		ID              uuid.UUID `gorm:"column:id"`
		FamilyID        uuid.UUID `gorm:"column:family_id"`
		StudentID       uuid.UUID `gorm:"column:student_id"`
		AttendanceDate  time.Time `gorm:"column:attendance_date"`
		Status          string    `gorm:"column:status"`
		DurationMinutes *int16    `gorm:"column:duration_minutes"`
		Notes           *string   `gorm:"column:notes"`
		IsAuto          bool      `gorm:"column:is_auto"`
		ManualOverride  bool      `gorm:"column:manual_override"`
		CreatedAt       time.Time `gorm:"column:created_at"`
		UpdatedAt       time.Time `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, attendance_date, status, duration_minutes,
		       notes, is_auto, manual_override, created_at, updated_at
		FROM comply_attendance
		WHERE id = ? AND family_id = ?`, attendanceID, scope.FamilyID()).First(&r_).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("comply attendance find: %w", err)
	}

	return &ComplyAttendance{
		ID: r_.ID, FamilyID: r_.FamilyID, StudentID: r_.StudentID,
		AttendanceDate: r_.AttendanceDate, Status: r_.Status,
		DurationMinutes: r_.DurationMinutes, Notes: r_.Notes,
		IsAuto: r_.IsAuto, ManualOverride: r_.ManualOverride,
		CreatedAt: r_.CreatedAt, UpdatedAt: r_.UpdatedAt,
	}, nil
}

func (r *PgAttendanceRepository) FindByStudentAndDate(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, date time.Time) (*ComplyAttendance, error) {
	type row struct {
		ID              uuid.UUID `gorm:"column:id"`
		FamilyID        uuid.UUID `gorm:"column:family_id"`
		StudentID       uuid.UUID `gorm:"column:student_id"`
		AttendanceDate  time.Time `gorm:"column:attendance_date"`
		Status          string    `gorm:"column:status"`
		DurationMinutes *int16    `gorm:"column:duration_minutes"`
		Notes           *string   `gorm:"column:notes"`
		IsAuto          bool      `gorm:"column:is_auto"`
		ManualOverride  bool      `gorm:"column:manual_override"`
		CreatedAt       time.Time `gorm:"column:created_at"`
		UpdatedAt       time.Time `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, attendance_date, status, duration_minutes,
		       notes, is_auto, manual_override, created_at, updated_at
		FROM comply_attendance
		WHERE family_id = ? AND student_id = ? AND attendance_date = ?`,
		scope.FamilyID(), studentID, date.Format("2006-01-02")).First(&r_).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("comply attendance find by student and date: %w", err)
	}

	return &ComplyAttendance{
		ID: r_.ID, FamilyID: r_.FamilyID, StudentID: r_.StudentID,
		AttendanceDate: r_.AttendanceDate, Status: r_.Status,
		DurationMinutes: r_.DurationMinutes, Notes: r_.Notes,
		IsAuto: r_.IsAuto, ManualOverride: r_.ManualOverride,
		CreatedAt: r_.CreatedAt, UpdatedAt: r_.UpdatedAt,
	}, nil
}

func (r *PgAttendanceRepository) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *AttendanceListParams) ([]ComplyAttendance, error) {
	type row struct {
		ID              uuid.UUID `gorm:"column:id"`
		FamilyID        uuid.UUID `gorm:"column:family_id"`
		StudentID       uuid.UUID `gorm:"column:student_id"`
		AttendanceDate  time.Time `gorm:"column:attendance_date"`
		Status          string    `gorm:"column:status"`
		DurationMinutes *int16    `gorm:"column:duration_minutes"`
		Notes           *string   `gorm:"column:notes"`
		IsAuto          bool      `gorm:"column:is_auto"`
		ManualOverride  bool      `gorm:"column:manual_override"`
		CreatedAt       time.Time `gorm:"column:created_at"`
		UpdatedAt       time.Time `gorm:"column:updated_at"`
	}

	limit := int64(50)
	if params != nil && params.Limit != nil {
		limit = min(int64(*params.Limit), 100)
	}

	args := []any{scope.FamilyID(), studentID}
	where := "family_id = ? AND student_id = ?"

	if params != nil && !params.StartDate.IsZero() {
		where += " AND attendance_date >= ?"
		args = append(args, params.StartDate.Format("2006-01-02"))
	}
	if params != nil && !params.EndDate.IsZero() {
		where += " AND attendance_date <= ?"
		args = append(args, params.EndDate.Format("2006-01-02"))
	}
	if params != nil && params.Status != nil {
		where += " AND status = ?"
		args = append(args, *params.Status)
	}

	offset := 0
	if params != nil && params.Cursor != nil {
		var err error
		offset, err = decodeComplyCursor(*params.Cursor)
		if err != nil {
			return nil, err
		}
	}
	args = append(args, limit+1, offset)

	var rows []row
	err := r.db.WithContext(ctx).Raw(
		fmt.Sprintf(`SELECT id, family_id, student_id, attendance_date, status, duration_minutes,
		       notes, is_auto, manual_override, created_at, updated_at
		FROM comply_attendance
		WHERE %s
		ORDER BY attendance_date DESC
		LIMIT ? OFFSET ?`, where), args...,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("comply attendance list: %w", err)
	}

	// Trim to limit (caller uses len(result) > limit to set next cursor)
	out := make([]ComplyAttendance, 0, len(rows))
	for _, r := range rows {
		out = append(out, ComplyAttendance(r))
	}
	return out, nil
}

func (r *PgAttendanceRepository) Summarize(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, startDate time.Time, endDate time.Time) (*AttendanceSummaryRow, error) {
	type row struct {
		PresentFull    int32 `gorm:"column:present_full"`
		PresentPartial int32 `gorm:"column:present_partial"`
		Absent         int32 `gorm:"column:absent"`
		NotApplicable  int32 `gorm:"column:not_applicable"`
		TotalMinutes   int64 `gorm:"column:total_minutes"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(CASE WHEN status = 'present_full' THEN 1 ELSE 0 END), 0) AS present_full,
			COALESCE(SUM(CASE WHEN status = 'present_partial' THEN 1 ELSE 0 END), 0) AS present_partial,
			COALESCE(SUM(CASE WHEN status = 'absent' THEN 1 ELSE 0 END), 0) AS absent,
			COALESCE(SUM(CASE WHEN status = 'not_applicable' THEN 1 ELSE 0 END), 0) AS not_applicable,
			COALESCE(SUM(COALESCE(duration_minutes, 0)), 0) AS total_minutes
		FROM comply_attendance
		WHERE family_id = ? AND student_id = ?
		  AND attendance_date >= ? AND attendance_date <= ?`,
		scope.FamilyID(), studentID,
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"),
	).Scan(&r_).Error
	if err != nil {
		return nil, fmt.Errorf("comply attendance summarize: %w", err)
	}

	return &AttendanceSummaryRow{
		PresentFull:    r_.PresentFull,
		PresentPartial: r_.PresentPartial,
		Absent:         r_.Absent,
		NotApplicable:  r_.NotApplicable,
		TotalMinutes:   r_.TotalMinutes,
	}, nil
}

func (r *PgAttendanceRepository) Update(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope, updates UpdateAttendanceRow) (*ComplyAttendance, error) {
	sets := "updated_at = now()"
	args := []any{}
	if updates.Status != nil {
		sets += ", status = ?"
		args = append(args, *updates.Status)
	}
	if updates.DurationMinutes != nil {
		sets += ", duration_minutes = ?"
		args = append(args, *updates.DurationMinutes)
	}
	if updates.Notes != nil {
		sets += ", notes = ?"
		args = append(args, *updates.Notes)
	}
	args = append(args, attendanceID, scope.FamilyID())

	tx := r.db.WithContext(ctx).Exec(
		fmt.Sprintf("UPDATE comply_attendance SET %s WHERE id = ? AND family_id = ?", sets),
		args...,
	)
	if tx.Error != nil {
		return nil, fmt.Errorf("comply attendance update: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}

	return r.FindByID(ctx, attendanceID, scope)
}

func (r *PgAttendanceRepository) Delete(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope) error {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_attendance WHERE id = ? AND family_id = ?`,
		attendanceID, scope.FamilyID(),
	)
	if tx.Error != nil {
		return fmt.Errorf("comply attendance delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrAttendanceNotFound
	}
	return nil
}

func (r *PgAttendanceRepository) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_attendance WHERE student_id = ? AND family_id = ?`,
		studentID, familyID,
	).Error
}

func (r *PgAttendanceRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_attendance WHERE family_id = ?`, familyID,
	).Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Assessment Repository [14-comply §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgAssessmentRepository implements AssessmentRepository using GORM/PostgreSQL.
type PgAssessmentRepository struct {
	db *gorm.DB
}

// NewPgAssessmentRepository creates a new PgAssessmentRepository.
func NewPgAssessmentRepository(db *gorm.DB) *PgAssessmentRepository {
	return &PgAssessmentRepository{db: db}
}

func (r *PgAssessmentRepository) Create(ctx context.Context, scope shared.FamilyScope, input CreateAssessmentRow) (*ComplyAssessmentRecord, error) {
	type idRow struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	var r_ idRow
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO comply_assessment_records
			(family_id, student_id, title, subject, assessment_type, score, max_score,
			 grade_letter, grade_points, is_passing, source_activity_id, assessment_date, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		scope.FamilyID(), input.StudentID, input.Title, input.Subject, input.AssessmentType,
		input.Score, input.MaxScore, input.GradeLetter, input.GradePoints, input.IsPassing,
		input.SourceActivityID, input.AssessmentDate.Format("2006-01-02"), input.Notes,
	).First(&r_).Error
	if err != nil {
		return nil, fmt.Errorf("comply assessment create: %w", err)
	}

	return r.FindByID(ctx, r_.ID, scope)
}

func (r *PgAssessmentRepository) FindByID(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope) (*ComplyAssessmentRecord, error) {
	type row struct {
		ID               uuid.UUID  `gorm:"column:id"`
		FamilyID         uuid.UUID  `gorm:"column:family_id"`
		StudentID        uuid.UUID  `gorm:"column:student_id"`
		Title            string     `gorm:"column:title"`
		Subject          string     `gorm:"column:subject"`
		AssessmentType   string     `gorm:"column:assessment_type"`
		Score            *float64   `gorm:"column:score"`
		MaxScore         *float64   `gorm:"column:max_score"`
		GradeLetter      *string    `gorm:"column:grade_letter"`
		GradePoints      *float64   `gorm:"column:grade_points"`
		IsPassing        *bool      `gorm:"column:is_passing"`
		SourceActivityID *uuid.UUID `gorm:"column:source_activity_id"`
		AssessmentDate   time.Time  `gorm:"column:assessment_date"`
		Notes            *string    `gorm:"column:notes"`
		CreatedAt        time.Time  `gorm:"column:created_at"`
		UpdatedAt        time.Time  `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, title, subject, assessment_type, score, max_score,
		       grade_letter, grade_points, is_passing, source_activity_id, assessment_date,
		       notes, created_at, updated_at
		FROM comply_assessment_records
		WHERE id = ? AND family_id = ?`, assessmentID, scope.FamilyID()).First(&r_).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("comply assessment find: %w", err)
	}

	return &ComplyAssessmentRecord{
		ID: r_.ID, FamilyID: r_.FamilyID, StudentID: r_.StudentID,
		Title: r_.Title, Subject: r_.Subject, AssessmentType: r_.AssessmentType,
		Score: r_.Score, MaxScore: r_.MaxScore, GradeLetter: r_.GradeLetter,
		GradePoints: r_.GradePoints, IsPassing: r_.IsPassing,
		SourceActivityID: r_.SourceActivityID, AssessmentDate: r_.AssessmentDate,
		Notes: r_.Notes, CreatedAt: r_.CreatedAt, UpdatedAt: r_.UpdatedAt,
	}, nil
}

func (r *PgAssessmentRepository) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *AssessmentListParams) ([]ComplyAssessmentRecord, error) {
	type row struct {
		ID               uuid.UUID  `gorm:"column:id"`
		FamilyID         uuid.UUID  `gorm:"column:family_id"`
		StudentID        uuid.UUID  `gorm:"column:student_id"`
		Title            string     `gorm:"column:title"`
		Subject          string     `gorm:"column:subject"`
		AssessmentType   string     `gorm:"column:assessment_type"`
		Score            *float64   `gorm:"column:score"`
		MaxScore         *float64   `gorm:"column:max_score"`
		GradeLetter      *string    `gorm:"column:grade_letter"`
		GradePoints      *float64   `gorm:"column:grade_points"`
		IsPassing        *bool      `gorm:"column:is_passing"`
		SourceActivityID *uuid.UUID `gorm:"column:source_activity_id"`
		AssessmentDate   time.Time  `gorm:"column:assessment_date"`
		Notes            *string    `gorm:"column:notes"`
		CreatedAt        time.Time  `gorm:"column:created_at"`
		UpdatedAt        time.Time  `gorm:"column:updated_at"`
	}

	limit := int64(50)
	if params != nil && params.Limit != nil {
		limit = min(int64(*params.Limit), 100)
	}

	args := []any{scope.FamilyID(), studentID}
	where := "family_id = ? AND student_id = ?"

	if params != nil && params.Subject != nil {
		where += " AND subject = ?"
		args = append(args, *params.Subject)
	}
	if params != nil && params.StartDate != nil {
		where += " AND assessment_date >= ?"
		args = append(args, params.StartDate.Format("2006-01-02"))
	}
	if params != nil && params.EndDate != nil {
		where += " AND assessment_date <= ?"
		args = append(args, params.EndDate.Format("2006-01-02"))
	}

	offset := 0
	if params != nil && params.Cursor != nil {
		var err error
		offset, err = decodeComplyCursor(*params.Cursor)
		if err != nil {
			return nil, err
		}
	}
	args = append(args, limit+1, offset)

	var rows []row
	err := r.db.WithContext(ctx).Raw(
		fmt.Sprintf(`SELECT id, family_id, student_id, title, subject, assessment_type, score, max_score,
		       grade_letter, grade_points, is_passing, source_activity_id, assessment_date,
		       notes, created_at, updated_at
		FROM comply_assessment_records
		WHERE %s
		ORDER BY assessment_date DESC
		LIMIT ? OFFSET ?`, where), args...,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("comply assessment list: %w", err)
	}

	out := make([]ComplyAssessmentRecord, 0, len(rows))
	for _, r := range rows {
		out = append(out, ComplyAssessmentRecord(r))
	}
	return out, nil
}

func (r *PgAssessmentRepository) Update(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope, updates UpdateAssessmentRow) (*ComplyAssessmentRecord, error) {
	sets := "updated_at = now()"
	args := []any{}
	if updates.Title != nil {
		sets += ", title = ?"
		args = append(args, *updates.Title)
	}
	if updates.Subject != nil {
		sets += ", subject = ?"
		args = append(args, *updates.Subject)
	}
	if updates.Score != nil {
		sets += ", score = ?"
		args = append(args, *updates.Score)
	}
	if updates.MaxScore != nil {
		sets += ", max_score = ?"
		args = append(args, *updates.MaxScore)
	}
	if updates.GradeLetter != nil {
		sets += ", grade_letter = ?"
		args = append(args, *updates.GradeLetter)
	}
	if updates.GradePoints != nil {
		sets += ", grade_points = ?"
		args = append(args, *updates.GradePoints)
	}
	if updates.IsPassing != nil {
		sets += ", is_passing = ?"
		args = append(args, *updates.IsPassing)
	}
	if updates.AssessmentDate != nil {
		sets += ", assessment_date = ?"
		args = append(args, updates.AssessmentDate.Format("2006-01-02"))
	}
	if updates.Notes != nil {
		sets += ", notes = ?"
		args = append(args, *updates.Notes)
	}
	args = append(args, assessmentID, scope.FamilyID())

	tx := r.db.WithContext(ctx).Exec(
		fmt.Sprintf("UPDATE comply_assessment_records SET %s WHERE id = ? AND family_id = ?", sets),
		args...,
	)
	if tx.Error != nil {
		return nil, fmt.Errorf("comply assessment update: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}

	return r.FindByID(ctx, assessmentID, scope)
}

func (r *PgAssessmentRepository) Delete(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope) error {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_assessment_records WHERE id = ? AND family_id = ?`,
		assessmentID, scope.FamilyID(),
	)
	if tx.Error != nil {
		return fmt.Errorf("comply assessment delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrAssessmentNotFound
	}
	return nil
}

func (r *PgAssessmentRepository) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_assessment_records WHERE student_id = ? AND family_id = ?`,
		studentID, familyID,
	).Error
}

func (r *PgAssessmentRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_assessment_records WHERE family_id = ?`, familyID,
	).Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test Score Repository [14-comply §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgTestScoreRepository implements TestScoreRepository using GORM/PostgreSQL.
type PgTestScoreRepository struct {
	db *gorm.DB
}

// NewPgTestScoreRepository creates a new PgTestScoreRepository.
func NewPgTestScoreRepository(db *gorm.DB) *PgTestScoreRepository {
	return &PgTestScoreRepository{db: db}
}

func (r *PgTestScoreRepository) Create(ctx context.Context, scope shared.FamilyScope, input CreateTestScoreRow) (*ComplyStandardizedTest, error) {
	type idRow struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	var r_ idRow
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO comply_standardized_tests
			(family_id, student_id, test_name, test_date, grade_level, scores, composite_score, percentile, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		scope.FamilyID(), input.StudentID, input.TestName, input.TestDate.Format("2006-01-02"),
		input.GradeLevel, input.Scores, input.CompositeScore, input.Percentile, input.Notes,
	).First(&r_).Error
	if err != nil {
		return nil, fmt.Errorf("comply test score create: %w", err)
	}

	return r.findByID(ctx, r_.ID, scope)
}

func (r *PgTestScoreRepository) findByID(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope) (*ComplyStandardizedTest, error) {
	type row struct {
		ID             uuid.UUID       `gorm:"column:id"`
		FamilyID       uuid.UUID       `gorm:"column:family_id"`
		StudentID      uuid.UUID       `gorm:"column:student_id"`
		TestName       string          `gorm:"column:test_name"`
		TestDate       time.Time       `gorm:"column:test_date"`
		GradeLevel     *int16          `gorm:"column:grade_level"`
		Scores         json.RawMessage `gorm:"column:scores"`
		CompositeScore *float64        `gorm:"column:composite_score"`
		Percentile     *int16          `gorm:"column:percentile"`
		Notes          *string         `gorm:"column:notes"`
		CreatedAt      time.Time       `gorm:"column:created_at"`
		UpdatedAt      time.Time       `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, test_name, test_date, grade_level, scores,
		       composite_score, percentile, notes, created_at, updated_at
		FROM comply_standardized_tests
		WHERE id = ? AND family_id = ?`, testID, scope.FamilyID()).First(&r_).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("comply test score find: %w", err)
	}

	return &ComplyStandardizedTest{
		ID: r_.ID, FamilyID: r_.FamilyID, StudentID: r_.StudentID,
		TestName: r_.TestName, TestDate: r_.TestDate, GradeLevel: r_.GradeLevel,
		Scores: r_.Scores, CompositeScore: r_.CompositeScore,
		Percentile: r_.Percentile, Notes: r_.Notes,
		CreatedAt: r_.CreatedAt, UpdatedAt: r_.UpdatedAt,
	}, nil
}

func (r *PgTestScoreRepository) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *TestListParams) ([]ComplyStandardizedTest, error) {
	type row struct {
		ID             uuid.UUID       `gorm:"column:id"`
		FamilyID       uuid.UUID       `gorm:"column:family_id"`
		StudentID      uuid.UUID       `gorm:"column:student_id"`
		TestName       string          `gorm:"column:test_name"`
		TestDate       time.Time       `gorm:"column:test_date"`
		GradeLevel     *int16          `gorm:"column:grade_level"`
		Scores         json.RawMessage `gorm:"column:scores"`
		CompositeScore *float64        `gorm:"column:composite_score"`
		Percentile     *int16          `gorm:"column:percentile"`
		Notes          *string         `gorm:"column:notes"`
		CreatedAt      time.Time       `gorm:"column:created_at"`
		UpdatedAt      time.Time       `gorm:"column:updated_at"`
	}

	limit := int64(50)
	if params != nil && params.Limit != nil {
		limit = min(int64(*params.Limit), 100)
	}

	offset := 0
	if params != nil && params.Cursor != nil {
		var err error
		offset, err = decodeComplyCursor(*params.Cursor)
		if err != nil {
			return nil, err
		}
	}

	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, test_name, test_date, grade_level, scores,
		       composite_score, percentile, notes, created_at, updated_at
		FROM comply_standardized_tests
		WHERE family_id = ? AND student_id = ?
		ORDER BY test_date DESC
		LIMIT ? OFFSET ?`,
		scope.FamilyID(), studentID, limit+1, offset,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("comply test score list: %w", err)
	}

	out := make([]ComplyStandardizedTest, 0, len(rows))
	for _, r := range rows {
		out = append(out, ComplyStandardizedTest(r))
	}
	return out, nil
}

func (r *PgTestScoreRepository) Update(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope, updates UpdateTestScoreRow) (*ComplyStandardizedTest, error) {
	sets := "updated_at = now()"
	args := []any{}
	if updates.TestName != nil {
		sets += ", test_name = ?"
		args = append(args, *updates.TestName)
	}
	if updates.TestDate != nil {
		sets += ", test_date = ?"
		args = append(args, updates.TestDate.Format("2006-01-02"))
	}
	if updates.Scores != nil {
		sets += ", scores = ?"
		args = append(args, *updates.Scores)
	}
	if updates.CompositeScore != nil {
		sets += ", composite_score = ?"
		args = append(args, *updates.CompositeScore)
	}
	if updates.Percentile != nil {
		sets += ", percentile = ?"
		args = append(args, *updates.Percentile)
	}
	if updates.Notes != nil {
		sets += ", notes = ?"
		args = append(args, *updates.Notes)
	}
	args = append(args, testID, scope.FamilyID())

	tx := r.db.WithContext(ctx).Exec(
		fmt.Sprintf("UPDATE comply_standardized_tests SET %s WHERE id = ? AND family_id = ?", sets),
		args...,
	)
	if tx.Error != nil {
		return nil, fmt.Errorf("comply test score update: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}

	return r.findByID(ctx, testID, scope)
}

func (r *PgTestScoreRepository) Delete(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope) error {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_standardized_tests WHERE id = ? AND family_id = ?`,
		testID, scope.FamilyID(),
	)
	if tx.Error != nil {
		return fmt.Errorf("comply test score delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrTestScoreNotFound
	}
	return nil
}

func (r *PgTestScoreRepository) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_standardized_tests WHERE student_id = ? AND family_id = ?`,
		studentID, familyID,
	).Error
}

func (r *PgTestScoreRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_standardized_tests WHERE family_id = ?`, familyID,
	).Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Portfolio Repository [14-comply §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgPortfolioRepository implements PortfolioRepository using GORM/PostgreSQL.
type PgPortfolioRepository struct {
	db *gorm.DB
}

// NewPgPortfolioRepository creates a new PgPortfolioRepository.
func NewPgPortfolioRepository(db *gorm.DB) *PgPortfolioRepository {
	return &PgPortfolioRepository{db: db}
}

func (r *PgPortfolioRepository) Create(ctx context.Context, scope shared.FamilyScope, input CreatePortfolioRow) (*ComplyPortfolio, error) {
	type idRow struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	var r_ idRow
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO comply_portfolios
			(family_id, student_id, title, description, organization,
			 date_range_start, date_range_end, include_attendance, include_assessments)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		scope.FamilyID(), input.StudentID, input.Title, input.Description, input.Organization,
		input.DateRangeStart.Format("2006-01-02"), input.DateRangeEnd.Format("2006-01-02"),
		input.IncludeAttendance, input.IncludeAssessments,
	).First(&r_).Error
	if err != nil {
		return nil, fmt.Errorf("comply portfolio create: %w", err)
	}

	return r.FindByID(ctx, r_.ID, scope)
}

func (r *PgPortfolioRepository) FindByID(ctx context.Context, portfolioID uuid.UUID, scope shared.FamilyScope) (*ComplyPortfolio, error) {
	type row struct {
		ID                 uuid.UUID  `gorm:"column:id"`
		FamilyID           uuid.UUID  `gorm:"column:family_id"`
		StudentID          uuid.UUID  `gorm:"column:student_id"`
		Title              string     `gorm:"column:title"`
		Description        *string    `gorm:"column:description"`
		Organization       string     `gorm:"column:organization"`
		DateRangeStart     time.Time  `gorm:"column:date_range_start"`
		DateRangeEnd       time.Time  `gorm:"column:date_range_end"`
		IncludeAttendance  bool       `gorm:"column:include_attendance"`
		IncludeAssessments bool       `gorm:"column:include_assessments"`
		Status             string     `gorm:"column:status"`
		UploadID           *uuid.UUID `gorm:"column:upload_id"`
		GeneratedAt        *time.Time `gorm:"column:generated_at"`
		ExpiresAt          *time.Time `gorm:"column:expires_at"`
		ErrorMessage       *string    `gorm:"column:error_message"`
		RetryCount         int16      `gorm:"column:retry_count"`
		CreatedAt          time.Time  `gorm:"column:created_at"`
		UpdatedAt          time.Time  `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, title, description, organization,
		       date_range_start, date_range_end, include_attendance, include_assessments,
		       status, upload_id, generated_at, expires_at, error_message, retry_count,
		       created_at, updated_at
		FROM comply_portfolios
		WHERE id = ? AND family_id = ?`, portfolioID, scope.FamilyID()).First(&r_).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("comply portfolio find: %w", err)
	}

	return &ComplyPortfolio{
		ID: r_.ID, FamilyID: r_.FamilyID, StudentID: r_.StudentID,
		Title: r_.Title, Description: r_.Description, Organization: r_.Organization,
		DateRangeStart: r_.DateRangeStart, DateRangeEnd: r_.DateRangeEnd,
		IncludeAttendance: r_.IncludeAttendance, IncludeAssessments: r_.IncludeAssessments,
		Status: r_.Status, UploadID: r_.UploadID,
		GeneratedAt: r_.GeneratedAt, ExpiresAt: r_.ExpiresAt,
		ErrorMessage: r_.ErrorMessage, RetryCount: r_.RetryCount,
		CreatedAt: r_.CreatedAt, UpdatedAt: r_.UpdatedAt,
	}, nil
}

func (r *PgPortfolioRepository) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]ComplyPortfolio, error) {
	type row struct {
		ID                 uuid.UUID  `gorm:"column:id"`
		FamilyID           uuid.UUID  `gorm:"column:family_id"`
		StudentID          uuid.UUID  `gorm:"column:student_id"`
		Title              string     `gorm:"column:title"`
		Description        *string    `gorm:"column:description"`
		Organization       string     `gorm:"column:organization"`
		DateRangeStart     time.Time  `gorm:"column:date_range_start"`
		DateRangeEnd       time.Time  `gorm:"column:date_range_end"`
		IncludeAttendance  bool       `gorm:"column:include_attendance"`
		IncludeAssessments bool       `gorm:"column:include_assessments"`
		Status             string     `gorm:"column:status"`
		UploadID           *uuid.UUID `gorm:"column:upload_id"`
		GeneratedAt        *time.Time `gorm:"column:generated_at"`
		ExpiresAt          *time.Time `gorm:"column:expires_at"`
		ErrorMessage       *string    `gorm:"column:error_message"`
		RetryCount         int16      `gorm:"column:retry_count"`
		CreatedAt          time.Time  `gorm:"column:created_at"`
		UpdatedAt          time.Time  `gorm:"column:updated_at"`
	}

	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, title, description, organization,
		       date_range_start, date_range_end, include_attendance, include_assessments,
		       status, upload_id, generated_at, expires_at, error_message, retry_count,
		       created_at, updated_at
		FROM comply_portfolios
		WHERE family_id = ? AND student_id = ?
		ORDER BY created_at DESC`,
		scope.FamilyID(), studentID,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("comply portfolio list: %w", err)
	}

	out := make([]ComplyPortfolio, len(rows))
	for i, r := range rows {
		out[i] = ComplyPortfolio(r)
	}
	return out, nil
}

func (r *PgPortfolioRepository) UpdateStatus(ctx context.Context, portfolioID uuid.UUID, status string, uploadID *uuid.UUID, errorMessage *string) (*ComplyPortfolio, error) {
	sets := "status = ?, updated_at = now()"
	args := []any{status}

	if status == string(PortfolioStatusReady) {
		sets += ", generated_at = now(), expires_at = now() + interval '90 days'"
	}
	if uploadID != nil {
		sets += ", upload_id = ?"
		args = append(args, *uploadID)
	}
	if errorMessage != nil {
		sets += ", error_message = ?, retry_count = retry_count + 1"
		args = append(args, *errorMessage)
	}
	args = append(args, portfolioID)

	tx := r.db.WithContext(ctx).Exec(
		fmt.Sprintf("UPDATE comply_portfolios SET %s WHERE id = ?", sets),
		args...,
	)
	if tx.Error != nil {
		return nil, fmt.Errorf("comply portfolio update status: %w", tx.Error)
	}

	// Return updated record without family scope (called from background jobs)
	type row struct {
		ID                 uuid.UUID  `gorm:"column:id"`
		FamilyID           uuid.UUID  `gorm:"column:family_id"`
		StudentID          uuid.UUID  `gorm:"column:student_id"`
		Title              string     `gorm:"column:title"`
		Description        *string    `gorm:"column:description"`
		Organization       string     `gorm:"column:organization"`
		DateRangeStart     time.Time  `gorm:"column:date_range_start"`
		DateRangeEnd       time.Time  `gorm:"column:date_range_end"`
		IncludeAttendance  bool       `gorm:"column:include_attendance"`
		IncludeAssessments bool       `gorm:"column:include_assessments"`
		Status             string     `gorm:"column:status"`
		UploadID           *uuid.UUID `gorm:"column:upload_id"`
		GeneratedAt        *time.Time `gorm:"column:generated_at"`
		ExpiresAt          *time.Time `gorm:"column:expires_at"`
		ErrorMessage       *string    `gorm:"column:error_message"`
		RetryCount         int16      `gorm:"column:retry_count"`
		CreatedAt          time.Time  `gorm:"column:created_at"`
		UpdatedAt          time.Time  `gorm:"column:updated_at"`
	}
	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, title, description, organization,
		       date_range_start, date_range_end, include_attendance, include_assessments,
		       status, upload_id, generated_at, expires_at, error_message, retry_count,
		       created_at, updated_at
		FROM comply_portfolios WHERE id = ?`, portfolioID).First(&r_).Error
	if err != nil {
		return nil, fmt.Errorf("comply portfolio find after update: %w", err)
	}
	return &ComplyPortfolio{
		ID: r_.ID, FamilyID: r_.FamilyID, StudentID: r_.StudentID,
		Title: r_.Title, Description: r_.Description, Organization: r_.Organization,
		DateRangeStart: r_.DateRangeStart, DateRangeEnd: r_.DateRangeEnd,
		IncludeAttendance: r_.IncludeAttendance, IncludeAssessments: r_.IncludeAssessments,
		Status: r_.Status, UploadID: r_.UploadID,
		GeneratedAt: r_.GeneratedAt, ExpiresAt: r_.ExpiresAt,
		ErrorMessage: r_.ErrorMessage, RetryCount: r_.RetryCount,
		CreatedAt: r_.CreatedAt, UpdatedAt: r_.UpdatedAt,
	}, nil
}

func (r *PgPortfolioRepository) FindExpired(ctx context.Context, before time.Time) ([]ComplyPortfolio, error) {
	type row struct {
		ID                 uuid.UUID  `gorm:"column:id"`
		FamilyID           uuid.UUID  `gorm:"column:family_id"`
		StudentID          uuid.UUID  `gorm:"column:student_id"`
		Title              string     `gorm:"column:title"`
		Description        *string    `gorm:"column:description"`
		Organization       string     `gorm:"column:organization"`
		DateRangeStart     time.Time  `gorm:"column:date_range_start"`
		DateRangeEnd       time.Time  `gorm:"column:date_range_end"`
		IncludeAttendance  bool       `gorm:"column:include_attendance"`
		IncludeAssessments bool       `gorm:"column:include_assessments"`
		Status             string     `gorm:"column:status"`
		UploadID           *uuid.UUID `gorm:"column:upload_id"`
		GeneratedAt        *time.Time `gorm:"column:generated_at"`
		ExpiresAt          *time.Time `gorm:"column:expires_at"`
		ErrorMessage       *string    `gorm:"column:error_message"`
		RetryCount         int16      `gorm:"column:retry_count"`
		CreatedAt          time.Time  `gorm:"column:created_at"`
		UpdatedAt          time.Time  `gorm:"column:updated_at"`
	}

	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, title, description, organization,
		       date_range_start, date_range_end, include_attendance, include_assessments,
		       status, upload_id, generated_at, expires_at, error_message, retry_count,
		       created_at, updated_at
		FROM comply_portfolios
		WHERE status = 'ready' AND expires_at < ?`, before).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("comply portfolio find expired: %w", err)
	}

	out := make([]ComplyPortfolio, len(rows))
	for i, r := range rows {
		out[i] = ComplyPortfolio(r)
	}
	return out, nil
}

func (r *PgPortfolioRepository) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_portfolios WHERE student_id = ? AND family_id = ?`,
		studentID, familyID,
	).Error
}

func (r *PgPortfolioRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_portfolios WHERE family_id = ?`, familyID,
	).Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Portfolio Item Repository [14-comply §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgPortfolioItemRepository implements PortfolioItemRepository using GORM/PostgreSQL.
type PgPortfolioItemRepository struct {
	db *gorm.DB
}

// NewPgPortfolioItemRepository creates a new PgPortfolioItemRepository.
func NewPgPortfolioItemRepository(db *gorm.DB) *PgPortfolioItemRepository {
	return &PgPortfolioItemRepository{db: db}
}

func (r *PgPortfolioItemRepository) CreateBatch(ctx context.Context, items []CreatePortfolioItemRow) ([]ComplyPortfolioItem, error) {
	if len(items) == 0 {
		return nil, nil
	}

	for _, item := range items {
		id := uuid.New()
		err := r.db.WithContext(ctx).Exec(`
			INSERT INTO comply_portfolio_items
				(id, portfolio_id, source_type, source_id, display_order,
				 cached_title, cached_subject, cached_date, cached_description, cached_attachments)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, item.PortfolioID, item.SourceType, item.SourceID, item.DisplayOrder,
			item.CachedTitle, item.CachedSubject, item.CachedDate.Format("2006-01-02"),
			item.CachedDescription, item.CachedAttachments,
		).Error
		if err != nil {
			return nil, fmt.Errorf("comply portfolio item create batch: %w", err)
		}
	}

	return r.ListByPortfolio(ctx, items[0].PortfolioID)
}

func (r *PgPortfolioItemRepository) ListByPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]ComplyPortfolioItem, error) {
	type row struct {
		ID                uuid.UUID       `gorm:"column:id"`
		PortfolioID       uuid.UUID       `gorm:"column:portfolio_id"`
		SourceType        string          `gorm:"column:source_type"`
		SourceID          uuid.UUID       `gorm:"column:source_id"`
		DisplayOrder      int16           `gorm:"column:display_order"`
		CachedTitle       string          `gorm:"column:cached_title"`
		CachedSubject     *string         `gorm:"column:cached_subject"`
		CachedDate        time.Time       `gorm:"column:cached_date"`
		CachedDescription *string         `gorm:"column:cached_description"`
		CachedAttachments json.RawMessage `gorm:"column:cached_attachments"`
		CreatedAt         time.Time       `gorm:"column:created_at"`
	}

	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, portfolio_id, source_type, source_id, display_order,
		       cached_title, cached_subject, cached_date, cached_description,
		       cached_attachments, created_at
		FROM comply_portfolio_items
		WHERE portfolio_id = ?
		ORDER BY display_order`, portfolioID).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("comply portfolio item list: %w", err)
	}

	out := make([]ComplyPortfolioItem, len(rows))
	for i, r := range rows {
		out[i] = ComplyPortfolioItem(r)
	}
	return out, nil
}

func (r *PgPortfolioItemRepository) CountByPortfolio(ctx context.Context, portfolioID uuid.UUID) (int32, error) {
	var count int32
	err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM comply_portfolio_items WHERE portfolio_id = ?`, portfolioID,
	).Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("comply portfolio item count: %w", err)
	}
	return count, nil
}

func (r *PgPortfolioItemRepository) DeleteByPortfolio(ctx context.Context, portfolioID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_portfolio_items WHERE portfolio_id = ?`, portfolioID,
	).Error
}

func (r *PgPortfolioItemRepository) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_portfolio_items WHERE portfolio_id IN (SELECT id FROM comply_portfolios WHERE student_id = ? AND family_id = ?)`,
		studentID, familyID,
	).Error
}

func (r *PgPortfolioItemRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_portfolio_items WHERE portfolio_id IN (SELECT id FROM comply_portfolios WHERE family_id = ?)`,
		familyID,
	).Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Transcript Repository [14-comply §6] — Phase 3
// ═══════════════════════════════════════════════════════════════════════════════

// PgTranscriptRepository implements TranscriptRepository using GORM/PostgreSQL.
type PgTranscriptRepository struct {
	db *gorm.DB
}

// NewPgTranscriptRepository creates a new PgTranscriptRepository.
func NewPgTranscriptRepository(db *gorm.DB) *PgTranscriptRepository {
	return &PgTranscriptRepository{db: db}
}

func (r *PgTranscriptRepository) Create(ctx context.Context, scope shared.FamilyScope, input CreateTranscriptRow) (*ComplyTranscript, error) {
	type idRow struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	levelsJSON, _ := json.Marshal(input.GradeLevels)
	var r_ idRow
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO comply_transcripts
			(family_id, student_id, title, student_name, grade_levels)
		VALUES (?, ?, ?, ?, ?::text[])
		RETURNING id`,
		scope.FamilyID(), input.StudentID, input.Title, input.StudentName, string(levelsJSON),
	).First(&r_).Error
	if err != nil {
		return nil, fmt.Errorf("comply transcript create: %w", err)
	}

	return r.FindByID(ctx, r_.ID, scope)
}

func (r *PgTranscriptRepository) FindByID(ctx context.Context, transcriptID uuid.UUID, scope shared.FamilyScope) (*ComplyTranscript, error) {
	type row struct {
		ID                    uuid.UUID  `gorm:"column:id"`
		FamilyID              uuid.UUID  `gorm:"column:family_id"`
		StudentID             uuid.UUID  `gorm:"column:student_id"`
		Title                 string     `gorm:"column:title"`
		StudentName           string     `gorm:"column:student_name"`
		GradeLevels           string     `gorm:"column:grade_levels"`
		Status                string     `gorm:"column:status"`
		SnapshotGpaUnweighted *float64   `gorm:"column:snapshot_gpa_unweighted"`
		SnapshotGpaWeighted   *float64   `gorm:"column:snapshot_gpa_weighted"`
		UploadID              *uuid.UUID `gorm:"column:upload_id"`
		GeneratedAt           *time.Time `gorm:"column:generated_at"`
		ExpiresAt             *time.Time `gorm:"column:expires_at"`
		ErrorMessage          *string    `gorm:"column:error_message"`
		RetryCount            int16      `gorm:"column:retry_count"`
		CreatedAt             time.Time  `gorm:"column:created_at"`
		UpdatedAt             time.Time  `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, title, student_name, grade_levels::text,
		       status, snapshot_gpa_unweighted, snapshot_gpa_weighted,
		       upload_id, generated_at, expires_at, error_message, retry_count,
		       created_at, updated_at
		FROM comply_transcripts
		WHERE id = ? AND family_id = ?`, transcriptID, scope.FamilyID()).First(&r_).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("comply transcript find: %w", err)
	}

	return &ComplyTranscript{
		ID: r_.ID, FamilyID: r_.FamilyID, StudentID: r_.StudentID,
		Title: r_.Title, StudentName: r_.StudentName,
		GradeLevels:           parsePostgresTextArray(r_.GradeLevels),
		Status:                r_.Status,
		SnapshotGpaUnweighted: r_.SnapshotGpaUnweighted,
		SnapshotGpaWeighted:   r_.SnapshotGpaWeighted,
		UploadID:              r_.UploadID,
		GeneratedAt: r_.GeneratedAt, ExpiresAt: r_.ExpiresAt,
		ErrorMessage: r_.ErrorMessage, RetryCount: r_.RetryCount,
		CreatedAt: r_.CreatedAt, UpdatedAt: r_.UpdatedAt,
	}, nil
}

func (r *PgTranscriptRepository) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]ComplyTranscript, error) {
	type row struct {
		ID                    uuid.UUID  `gorm:"column:id"`
		FamilyID              uuid.UUID  `gorm:"column:family_id"`
		StudentID             uuid.UUID  `gorm:"column:student_id"`
		Title                 string     `gorm:"column:title"`
		StudentName           string     `gorm:"column:student_name"`
		GradeLevels           string     `gorm:"column:grade_levels"`
		Status                string     `gorm:"column:status"`
		SnapshotGpaUnweighted *float64   `gorm:"column:snapshot_gpa_unweighted"`
		SnapshotGpaWeighted   *float64   `gorm:"column:snapshot_gpa_weighted"`
		UploadID              *uuid.UUID `gorm:"column:upload_id"`
		GeneratedAt           *time.Time `gorm:"column:generated_at"`
		ExpiresAt             *time.Time `gorm:"column:expires_at"`
		ErrorMessage          *string    `gorm:"column:error_message"`
		RetryCount            int16      `gorm:"column:retry_count"`
		CreatedAt             time.Time  `gorm:"column:created_at"`
		UpdatedAt             time.Time  `gorm:"column:updated_at"`
	}

	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, title, student_name, grade_levels::text,
		       status, snapshot_gpa_unweighted, snapshot_gpa_weighted,
		       upload_id, generated_at, expires_at, error_message, retry_count,
		       created_at, updated_at
		FROM comply_transcripts
		WHERE family_id = ? AND student_id = ?
		ORDER BY created_at DESC`,
		scope.FamilyID(), studentID,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("comply transcript list: %w", err)
	}

	out := make([]ComplyTranscript, len(rows))
	for i, r := range rows {
		out[i] = ComplyTranscript{
			ID: r.ID, FamilyID: r.FamilyID, StudentID: r.StudentID,
			Title: r.Title, StudentName: r.StudentName,
			GradeLevels:           parsePostgresTextArray(r.GradeLevels),
			Status:                r.Status,
			SnapshotGpaUnweighted: r.SnapshotGpaUnweighted,
			SnapshotGpaWeighted:   r.SnapshotGpaWeighted,
			UploadID:              r.UploadID,
			GeneratedAt: r.GeneratedAt, ExpiresAt: r.ExpiresAt,
			ErrorMessage: r.ErrorMessage, RetryCount: r.RetryCount,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
	}
	return out, nil
}

func (r *PgTranscriptRepository) UpdateStatus(ctx context.Context, transcriptID uuid.UUID, status string, uploadID *uuid.UUID, gpaUnweighted *float64, gpaWeighted *float64, errorMessage *string) (*ComplyTranscript, error) {
	sets := "status = ?, updated_at = now()"
	args := []any{status}

	if status == string(PortfolioStatusReady) {
		sets += ", generated_at = now(), expires_at = now() + interval '90 days'"
	}
	if uploadID != nil {
		sets += ", upload_id = ?"
		args = append(args, *uploadID)
	}
	if gpaUnweighted != nil {
		sets += ", snapshot_gpa_unweighted = ?"
		args = append(args, *gpaUnweighted)
	}
	if gpaWeighted != nil {
		sets += ", snapshot_gpa_weighted = ?"
		args = append(args, *gpaWeighted)
	}
	if errorMessage != nil {
		sets += ", error_message = ?, retry_count = retry_count + 1"
		args = append(args, *errorMessage)
	}
	args = append(args, transcriptID)

	tx := r.db.WithContext(ctx).Exec(
		fmt.Sprintf("UPDATE comply_transcripts SET %s WHERE id = ?", sets),
		args...,
	)
	if tx.Error != nil {
		return nil, fmt.Errorf("comply transcript update status: %w", tx.Error)
	}

	// Return without scope (background job context)
	type row struct {
		ID                    uuid.UUID  `gorm:"column:id"`
		FamilyID              uuid.UUID  `gorm:"column:family_id"`
		StudentID             uuid.UUID  `gorm:"column:student_id"`
		Title                 string     `gorm:"column:title"`
		StudentName           string     `gorm:"column:student_name"`
		GradeLevels           string     `gorm:"column:grade_levels"`
		Status                string     `gorm:"column:status"`
		SnapshotGpaUnweighted *float64   `gorm:"column:snapshot_gpa_unweighted"`
		SnapshotGpaWeighted   *float64   `gorm:"column:snapshot_gpa_weighted"`
		UploadID              *uuid.UUID `gorm:"column:upload_id"`
		GeneratedAt           *time.Time `gorm:"column:generated_at"`
		ExpiresAt             *time.Time `gorm:"column:expires_at"`
		ErrorMessage          *string    `gorm:"column:error_message"`
		RetryCount            int16      `gorm:"column:retry_count"`
		CreatedAt             time.Time  `gorm:"column:created_at"`
		UpdatedAt             time.Time  `gorm:"column:updated_at"`
	}
	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, title, student_name, grade_levels::text,
		       status, snapshot_gpa_unweighted, snapshot_gpa_weighted,
		       upload_id, generated_at, expires_at, error_message, retry_count,
		       created_at, updated_at
		FROM comply_transcripts WHERE id = ?`, transcriptID).First(&r_).Error
	if err != nil {
		return nil, fmt.Errorf("comply transcript find after update: %w", err)
	}
	return &ComplyTranscript{
		ID: r_.ID, FamilyID: r_.FamilyID, StudentID: r_.StudentID,
		Title: r_.Title, StudentName: r_.StudentName,
		GradeLevels:           parsePostgresTextArray(r_.GradeLevels),
		Status:                r_.Status,
		SnapshotGpaUnweighted: r_.SnapshotGpaUnweighted,
		SnapshotGpaWeighted:   r_.SnapshotGpaWeighted,
		UploadID:              r_.UploadID,
		GeneratedAt: r_.GeneratedAt, ExpiresAt: r_.ExpiresAt,
		ErrorMessage: r_.ErrorMessage, RetryCount: r_.RetryCount,
		CreatedAt: r_.CreatedAt, UpdatedAt: r_.UpdatedAt,
	}, nil
}

func (r *PgTranscriptRepository) Delete(ctx context.Context, transcriptID uuid.UUID, scope shared.FamilyScope) error {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_transcripts WHERE id = ? AND family_id = ?`,
		transcriptID, scope.FamilyID(),
	)
	if tx.Error != nil {
		return fmt.Errorf("comply transcript delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrTranscriptNotFound
	}
	return nil
}

func (r *PgTranscriptRepository) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_transcripts WHERE student_id = ? AND family_id = ?`,
		studentID, familyID,
	).Error
}

func (r *PgTranscriptRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_transcripts WHERE family_id = ?`, familyID,
	).Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Course Repository [14-comply §6] — Phase 3
// ═══════════════════════════════════════════════════════════════════════════════

// PgCourseRepository implements CourseRepository using GORM/PostgreSQL.
type PgCourseRepository struct {
	db *gorm.DB
}

// NewPgCourseRepository creates a new PgCourseRepository.
func NewPgCourseRepository(db *gorm.DB) *PgCourseRepository {
	return &PgCourseRepository{db: db}
}

func (r *PgCourseRepository) Create(ctx context.Context, scope shared.FamilyScope, input CreateCourseRow) (*ComplyCourse, error) {
	type idRow struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	var r_ idRow
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO comply_courses
			(family_id, student_id, transcript_id, title, subject, grade_level,
			 credits, grade_letter, grade_points, level, school_year, semester)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		scope.FamilyID(), input.StudentID, input.TranscriptID,
		input.Title, input.Subject, input.GradeLevel,
		input.Credits, input.GradeLetter, input.GradePoints,
		input.Level, input.SchoolYear, input.Semester,
	).First(&r_).Error
	if err != nil {
		return nil, fmt.Errorf("comply course create: %w", err)
	}

	return r.findByID(ctx, r_.ID, scope)
}

func (r *PgCourseRepository) findByID(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope) (*ComplyCourse, error) {
	type row struct {
		ID           uuid.UUID  `gorm:"column:id"`
		FamilyID     uuid.UUID  `gorm:"column:family_id"`
		StudentID    uuid.UUID  `gorm:"column:student_id"`
		TranscriptID *uuid.UUID `gorm:"column:transcript_id"`
		Title        string     `gorm:"column:title"`
		Subject      string     `gorm:"column:subject"`
		GradeLevel   int16      `gorm:"column:grade_level"`
		Credits      float64    `gorm:"column:credits"`
		GradeLetter  *string    `gorm:"column:grade_letter"`
		GradePoints  *float64   `gorm:"column:grade_points"`
		Level        string     `gorm:"column:level"`
		SchoolYear   string     `gorm:"column:school_year"`
		Semester     *string    `gorm:"column:semester"`
		CreatedAt    time.Time  `gorm:"column:created_at"`
		UpdatedAt    time.Time  `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, transcript_id, title, subject, grade_level,
		       credits, grade_letter, grade_points, level, school_year, semester,
		       created_at, updated_at
		FROM comply_courses
		WHERE id = ? AND family_id = ?`, courseID, scope.FamilyID()).First(&r_).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("comply course find: %w", err)
	}

	return &ComplyCourse{
		ID: r_.ID, FamilyID: r_.FamilyID, StudentID: r_.StudentID,
		TranscriptID: r_.TranscriptID,
		Title: r_.Title, Subject: r_.Subject, GradeLevel: r_.GradeLevel,
		Credits: r_.Credits, GradeLetter: r_.GradeLetter,
		GradePoints: r_.GradePoints, Level: r_.Level,
		SchoolYear: r_.SchoolYear, Semester: r_.Semester,
		CreatedAt: r_.CreatedAt, UpdatedAt: r_.UpdatedAt,
	}, nil
}

func (r *PgCourseRepository) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *CourseListParams) ([]ComplyCourse, error) {
	type row struct {
		ID           uuid.UUID  `gorm:"column:id"`
		FamilyID     uuid.UUID  `gorm:"column:family_id"`
		StudentID    uuid.UUID  `gorm:"column:student_id"`
		TranscriptID *uuid.UUID `gorm:"column:transcript_id"`
		Title        string     `gorm:"column:title"`
		Subject      string     `gorm:"column:subject"`
		GradeLevel   int16      `gorm:"column:grade_level"`
		Credits      float64    `gorm:"column:credits"`
		GradeLetter  *string    `gorm:"column:grade_letter"`
		GradePoints  *float64   `gorm:"column:grade_points"`
		Level        string     `gorm:"column:level"`
		SchoolYear   string     `gorm:"column:school_year"`
		Semester     *string    `gorm:"column:semester"`
		CreatedAt    time.Time  `gorm:"column:created_at"`
		UpdatedAt    time.Time  `gorm:"column:updated_at"`
	}

	limit := int64(50)
	if params != nil && params.Limit != nil {
		limit = min(int64(*params.Limit), 100)
	}

	args := []any{scope.FamilyID(), studentID}
	where := "family_id = ? AND student_id = ?"

	if params != nil && params.GradeLevel != nil {
		where += " AND grade_level = ?"
		args = append(args, *params.GradeLevel)
	}
	if params != nil && params.SchoolYear != nil {
		where += " AND school_year = ?"
		args = append(args, *params.SchoolYear)
	}

	offset := 0
	if params != nil && params.Cursor != nil {
		var err error
		offset, err = decodeComplyCursor(*params.Cursor)
		if err != nil {
			return nil, err
		}
	}
	args = append(args, limit+1, offset)

	var rows []row
	err := r.db.WithContext(ctx).Raw(
		fmt.Sprintf(`SELECT id, family_id, student_id, transcript_id, title, subject, grade_level,
		       credits, grade_letter, grade_points, level, school_year, semester,
		       created_at, updated_at
		FROM comply_courses
		WHERE %s
		ORDER BY school_year DESC, grade_level DESC
		LIMIT ? OFFSET ?`, where), args...,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("comply course list: %w", err)
	}

	out := make([]ComplyCourse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ComplyCourse(r))
	}
	return out, nil
}

func (r *PgCourseRepository) Update(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope, updates UpdateCourseRow) (*ComplyCourse, error) {
	sets := "updated_at = now()"
	args := []any{}
	if updates.Title != nil {
		sets += ", title = ?"
		args = append(args, *updates.Title)
	}
	if updates.Subject != nil {
		sets += ", subject = ?"
		args = append(args, *updates.Subject)
	}
	if updates.Credits != nil {
		sets += ", credits = ?"
		args = append(args, *updates.Credits)
	}
	if updates.GradeLetter != nil {
		sets += ", grade_letter = ?"
		args = append(args, *updates.GradeLetter)
	}
	if updates.GradePoints != nil {
		sets += ", grade_points = ?"
		args = append(args, *updates.GradePoints)
	}
	if updates.Level != nil {
		sets += ", level = ?"
		args = append(args, *updates.Level)
	}
	if updates.Semester != nil {
		sets += ", semester = ?"
		args = append(args, *updates.Semester)
	}
	args = append(args, courseID, scope.FamilyID())

	tx := r.db.WithContext(ctx).Exec(
		fmt.Sprintf("UPDATE comply_courses SET %s WHERE id = ? AND family_id = ?", sets),
		args...,
	)
	if tx.Error != nil {
		return nil, fmt.Errorf("comply course update: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}

	return r.findByID(ctx, courseID, scope)
}

func (r *PgCourseRepository) Delete(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope) error {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_courses WHERE id = ? AND family_id = ?`,
		courseID, scope.FamilyID(),
	)
	if tx.Error != nil {
		return fmt.Errorf("comply course delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrCourseNotFound
	}
	return nil
}

func (r *PgCourseRepository) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_courses WHERE student_id = ? AND family_id = ?`,
		studentID, familyID,
	).Error
}

func (r *PgCourseRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM comply_courses WHERE family_id = ?`, familyID,
	).Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════


func decodeComplyCursor(cursor string) (int, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("comply: invalid cursor: %w", err)
	}
	n, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("comply: invalid cursor offset: %w", err)
	}
	return n, nil
}

// parsePostgresTextArray parses a PostgreSQL text[] literal like {"a","b"} into a []string.
func parsePostgresTextArray(s string) []string {
	if s == "" || s == "{}" {
		return nil
	}
	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range splitPostgresArray(s) {
		if len(part) >= 2 && part[0] == '"' && part[len(part)-1] == '"' {
			part = part[1 : len(part)-1]
		}
		out = append(out, part)
	}
	return out
}

// parsePostgresBoolArray parses a PostgreSQL boolean[] literal like {t,f,t} into a []bool.
func parsePostgresBoolArray(s string) []bool {
	if s == "" || s == "{}" {
		return nil
	}
	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		return nil
	}
	var out []bool
	for _, part := range splitPostgresArray(s) {
		out = append(out, part == "t" || part == "true")
	}
	return out
}

func splitPostgresArray(s string) []string {
	var parts []string
	inQuotes := false
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}
