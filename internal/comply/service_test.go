package comply

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/comply/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// TestNewComplianceService_Scaffolding verifies the package compiles
// and the constructor wires all dependencies correctly.
func TestNewComplianceService_Scaffolding(t *testing.T) {
	svc := newTestService(
		&stubStateConfigRepo{},
		&stubFamilyConfigRepo{},
		&stubScheduleRepo{},
		&stubAttendanceRepo{},
		&stubAssessmentRepo{},
		&stubTestScoreRepo{},
		&stubPortfolioRepo{},
		&stubPortfolioItemRepo{},
		&stubTranscriptRepo{},
		&stubCourseRepo{},
		&stubIamService{},
		&stubLearningService{},
		&stubDiscoveryService{},
		&stubMediaService{},
	)
	if svc == nil {
		t.Fatal("expected non-nil ComplianceService")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group B: Family Config & State Config (B1–B10)
// ═══════════════════════════════════════════════════════════════════════════════

func TestUpsertFamilyConfig_CreatesNewConfig(t *testing.T) {
	scope := testScope()
	now := time.Now().UTC()
	stateRepo := &stubStateConfigRepo{
		findByStateCodeFn: func(_ context.Context, code string) (*ComplyStateConfig, error) {
			if code == "TX" {
				return &ComplyStateConfig{StateCode: "TX", StateName: "Texas"}, nil
			}
			return nil, nil
		},
	}
	familyRepo := &stubFamilyConfigRepo{
		upsertFn: func(_ context.Context, _ shared.FamilyScope, input UpsertFamilyConfigRow) (*ComplyFamilyConfig, error) {
			return &ComplyFamilyConfig{
				FamilyID:        scope.FamilyID(),
				StateCode:       input.StateCode,
				SchoolYearStart: input.SchoolYearStart,
				SchoolYearEnd:   input.SchoolYearEnd,
				TotalSchoolDays: input.TotalSchoolDays,
				GpaScale:        input.GpaScale,
				CreatedAt:       now,
				UpdatedAt:       now,
			}, nil
		},
	}
	svc := newTestService(stateRepo, familyRepo, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.UpsertFamilyConfig(context.Background(), UpsertFamilyConfigCommand{
		StateCode:       "TX",
		SchoolYearStart: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		SchoolYearEnd:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		TotalSchoolDays: 180,
		GpaScale:        "standard_4",
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StateCode != "TX" {
		t.Fatalf("got state_code=%q, want TX", resp.StateCode)
	}
}

func TestUpsertFamilyConfig_RejectsInvalidStateCode(t *testing.T) {
	scope := testScope()
	stateRepo := &stubStateConfigRepo{
		findByStateCodeFn: func(_ context.Context, _ string) (*ComplyStateConfig, error) {
			return nil, nil // not found
		},
	}
	svc := newTestService(stateRepo, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.UpsertFamilyConfig(context.Background(), UpsertFamilyConfigCommand{
		StateCode:       "ZZ",
		SchoolYearStart: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		SchoolYearEnd:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		TotalSchoolDays: 180,
		GpaScale:        "standard_4",
	}, *scope)
	if !errors.Is(err, ErrInvalidStateCode) {
		t.Fatalf("got %v, want ErrInvalidStateCode", err)
	}
}

func TestUpsertFamilyConfig_RejectsInvalidSchoolYearRange(t *testing.T) {
	scope := testScope()
	stateRepo := &stubStateConfigRepo{
		findByStateCodeFn: func(_ context.Context, _ string) (*ComplyStateConfig, error) {
			return &ComplyStateConfig{StateCode: "TX"}, nil
		},
	}
	svc := newTestService(stateRepo, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.UpsertFamilyConfig(context.Background(), UpsertFamilyConfigCommand{
		StateCode:       "TX",
		SchoolYearStart: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		SchoolYearEnd:   time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), // end before start
		TotalSchoolDays: 180,
		GpaScale:        "standard_4",
	}, *scope)
	if !errors.Is(err, ErrInvalidSchoolYearRange) {
		t.Fatalf("got %v, want ErrInvalidSchoolYearRange", err)
	}
}

func TestUpsertFamilyConfig_UpdatesExistingConfig(t *testing.T) {
	scope := testScope()
	now := time.Now().UTC()
	stateRepo := &stubStateConfigRepo{
		findByStateCodeFn: func(_ context.Context, _ string) (*ComplyStateConfig, error) {
			return &ComplyStateConfig{StateCode: "CA", StateName: "California"}, nil
		},
	}
	familyRepo := &stubFamilyConfigRepo{
		upsertFn: func(_ context.Context, _ shared.FamilyScope, input UpsertFamilyConfigRow) (*ComplyFamilyConfig, error) {
			return &ComplyFamilyConfig{
				FamilyID:        scope.FamilyID(),
				StateCode:       input.StateCode,
				SchoolYearStart: input.SchoolYearStart,
				SchoolYearEnd:   input.SchoolYearEnd,
				TotalSchoolDays: input.TotalSchoolDays,
				GpaScale:        input.GpaScale,
				CreatedAt:       now.Add(-24 * time.Hour),
				UpdatedAt:       now,
			}, nil
		},
	}
	svc := newTestService(stateRepo, familyRepo, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.UpsertFamilyConfig(context.Background(), UpsertFamilyConfigCommand{
		StateCode:       "CA",
		SchoolYearStart: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		SchoolYearEnd:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		TotalSchoolDays: 175,
		GpaScale:        "standard_4",
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StateCode != "CA" {
		t.Fatalf("got state_code=%q, want CA", resp.StateCode)
	}
}

func TestUpsertFamilyConfig_ValidatesCustomScheduleID(t *testing.T) {
	scope := testScope()
	schedID := uuid.Must(uuid.NewV7())
	stateRepo := &stubStateConfigRepo{
		findByStateCodeFn: func(_ context.Context, _ string) (*ComplyStateConfig, error) {
			return &ComplyStateConfig{StateCode: "TX"}, nil
		},
	}
	schedRepo := &stubScheduleRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyCustomSchedule, error) {
			return nil, nil // not found
		},
	}
	svc := newTestService(stateRepo, &stubFamilyConfigRepo{}, schedRepo, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.UpsertFamilyConfig(context.Background(), UpsertFamilyConfigCommand{
		StateCode:        "TX",
		SchoolYearStart:  time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		SchoolYearEnd:    time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		TotalSchoolDays:  180,
		CustomScheduleID: &schedID,
		GpaScale:         "standard_4",
	}, *scope)
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("got %v, want ErrScheduleNotFound", err)
	}
}

func TestGetFamilyConfig_ReturnsNilForMissingConfig(t *testing.T) {
	scope := testScope()
	familyRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return nil, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, familyRepo, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GetFamilyConfig(context.Background(), *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Fatal("expected nil response for missing config")
	}
}

func TestGetFamilyConfig_ReturnsExistingConfig(t *testing.T) {
	scope := testScope()
	now := time.Now().UTC()
	familyRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return &ComplyFamilyConfig{
				FamilyID:        scope.FamilyID(),
				StateCode:       "TX",
				SchoolYearStart: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
				SchoolYearEnd:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
				TotalSchoolDays: 180,
				GpaScale:        "standard_4",
				CreatedAt:       now,
				UpdatedAt:       now,
			}, nil
		},
	}
	stateRepo := &stubStateConfigRepo{
		findByStateCodeFn: func(_ context.Context, _ string) (*ComplyStateConfig, error) {
			return &ComplyStateConfig{StateCode: "TX", StateName: "Texas"}, nil
		},
	}
	svc := newTestService(stateRepo, familyRepo, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GetFamilyConfig(context.Background(), *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.StateCode != "TX" {
		t.Fatalf("got %+v, want config with StateCode=TX", resp)
	}
}

func TestListStateConfigs_ReturnsAllCachedConfigs(t *testing.T) {
	stateRepo := &stubStateConfigRepo{
		listAllFn: func(_ context.Context) ([]ComplyStateConfig, error) {
			return []ComplyStateConfig{
				{StateCode: "TX", StateName: "Texas", RegulationLevel: "low", AttendanceRequired: true, AttendanceDays: int16Ptr(180)},
				{StateCode: "CA", StateName: "California", RegulationLevel: "moderate", AttendanceRequired: false},
			}, nil
		},
	}
	svc := newTestService(stateRepo, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.ListStateConfigs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("got %d configs, want 2", len(resp))
	}
}

func TestGetStateConfig_ReturnsConfigForValidCode(t *testing.T) {
	stateRepo := &stubStateConfigRepo{
		findByStateCodeFn: func(_ context.Context, code string) (*ComplyStateConfig, error) {
			if code == "TX" {
				return &ComplyStateConfig{StateCode: "TX", StateName: "Texas", RegulationLevel: "low"}, nil
			}
			return nil, nil
		},
	}
	svc := newTestService(stateRepo, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GetStateConfig(context.Background(), "TX")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StateCode != "TX" {
		t.Fatalf("got state_code=%q, want TX", resp.StateCode)
	}
}

func TestGetStateConfig_ReturnsNotFoundForUnknown(t *testing.T) {
	stateRepo := &stubStateConfigRepo{
		findByStateCodeFn: func(_ context.Context, _ string) (*ComplyStateConfig, error) {
			return nil, nil
		},
	}
	svc := newTestService(stateRepo, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.GetStateConfig(context.Background(), "ZZ")
	if !errors.Is(err, ErrStateConfigNotFound) {
		t.Fatalf("got %v, want ErrStateConfigNotFound", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group C: Custom Schedules (C1–C7)
// ═══════════════════════════════════════════════════════════════════════════════

func TestCreateSchedule_ValidSevenElements(t *testing.T) {
	scope := testScope()
	schedRepo := &stubScheduleRepo{
		createFn: func(_ context.Context, _ shared.FamilyScope, input CreateScheduleRow) (*ComplyCustomSchedule, error) {
			return &ComplyCustomSchedule{
				ID:         uuid.Must(uuid.NewV7()),
				FamilyID:   scope.FamilyID(),
				Name:       input.Name,
				SchoolDays: input.SchoolDays,
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, schedRepo, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.CreateSchedule(context.Background(), CreateScheduleCommand{
		Name:       "4-Day Week",
		SchoolDays: []bool{false, true, true, true, true, false, false},
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "4-Day Week" {
		t.Fatalf("got name=%q, want 4-Day Week", resp.Name)
	}
}

func TestCreateSchedule_RejectsNon7Elements(t *testing.T) {
	scope := testScope()
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.CreateSchedule(context.Background(), CreateScheduleCommand{
		Name:       "Bad",
		SchoolDays: []bool{true, true, true},
	}, *scope)
	if !errors.Is(err, ErrInvalidSchoolDaysArray) {
		t.Fatalf("got %v, want ErrInvalidSchoolDaysArray", err)
	}
}

func TestListSchedules_ReturnsFamilySchedules(t *testing.T) {
	scope := testScope()
	schedRepo := &stubScheduleRepo{
		listByFamilyFn: func(_ context.Context, _ shared.FamilyScope) ([]ComplyCustomSchedule, error) {
			return []ComplyCustomSchedule{
				{ID: uuid.Must(uuid.NewV7()), Name: "Standard", SchoolDays: []bool{false, true, true, true, true, true, false}},
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, schedRepo, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.ListSchedules(context.Background(), *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("got %d schedules, want 1", len(resp))
	}
}

func TestUpdateSchedule_UpdatesExisting(t *testing.T) {
	scope := testScope()
	schedID := uuid.Must(uuid.NewV7())
	newName := "Updated Name"
	schedRepo := &stubScheduleRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID, _ shared.FamilyScope) (*ComplyCustomSchedule, error) {
			if id == schedID {
				return &ComplyCustomSchedule{ID: schedID, Name: "Old Name"}, nil
			}
			return nil, nil
		},
		updateFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, updates UpdateScheduleRow) (*ComplyCustomSchedule, error) {
			return &ComplyCustomSchedule{ID: schedID, Name: *updates.Name}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, schedRepo, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.UpdateSchedule(context.Background(), schedID, UpdateScheduleCommand{Name: &newName}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "Updated Name" {
		t.Fatalf("got name=%q, want Updated Name", resp.Name)
	}
}

func TestUpdateSchedule_ReturnsNotFoundForNonExistent(t *testing.T) {
	scope := testScope()
	schedRepo := &stubScheduleRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyCustomSchedule, error) {
			return nil, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, schedRepo, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.UpdateSchedule(context.Background(), uuid.Must(uuid.NewV7()), UpdateScheduleCommand{}, *scope)
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("got %v, want ErrScheduleNotFound", err)
	}
}

func TestDeleteSchedule_DeletesScheduleNotInUse(t *testing.T) {
	scope := testScope()
	schedID := uuid.Must(uuid.NewV7())
	deleted := false
	schedRepo := &stubScheduleRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyCustomSchedule, error) {
			return &ComplyCustomSchedule{ID: schedID}, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) error {
			deleted = true
			return nil
		},
	}
	familyRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return nil, nil // no config → schedule not in use
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, familyRepo, schedRepo, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.DeleteSchedule(context.Background(), schedID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected schedule to be deleted")
	}
}

func TestDeleteSchedule_RejectsScheduleInUse(t *testing.T) {
	scope := testScope()
	schedID := uuid.Must(uuid.NewV7())
	schedRepo := &stubScheduleRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyCustomSchedule, error) {
			return &ComplyCustomSchedule{ID: schedID}, nil
		},
	}
	familyRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return &ComplyFamilyConfig{CustomScheduleID: &schedID}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, familyRepo, schedRepo, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.DeleteSchedule(context.Background(), schedID, *scope)
	if !errors.Is(err, ErrScheduleInUse) {
		t.Fatalf("got %v, want ErrScheduleInUse", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group D: Attendance Service (D1–D14)
// ═══════════════════════════════════════════════════════════════════════════════

func TestRecordAttendance_CreatesManualRecord(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	today := time.Now().UTC().Truncate(24 * time.Hour)

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	attRepo := &stubAttendanceRepo{
		upsertFn: func(_ context.Context, _ shared.FamilyScope, input UpsertAttendanceRow) (*ComplyAttendance, error) {
			if input.IsAuto || !input.ManualOverride {
				t.Fatal("manual record should have is_auto=false, manual_override=true")
			}
			return &ComplyAttendance{
				ID:             uuid.Must(uuid.NewV7()),
				FamilyID:       scope.FamilyID(),
				StudentID:      input.StudentID,
				AttendanceDate: input.AttendanceDate,
				Status:         input.Status,
				IsAuto:         false,
				ManualOverride: true,
				CreatedAt:      time.Now().UTC(),
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.RecordAttendance(context.Background(), studentID, RecordAttendanceCommand{
		AttendanceDate: today,
		Status:         "present_full",
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.IsAuto {
		t.Fatal("expected is_auto=false")
	}
	if !resp.ManualOverride {
		t.Fatal("expected manual_override=true")
	}
}

func TestRecordAttendance_UpsertsOverAutoRecord(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	today := time.Now().UTC().Truncate(24 * time.Hour)
	upserted := false

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	attRepo := &stubAttendanceRepo{
		upsertFn: func(_ context.Context, _ shared.FamilyScope, input UpsertAttendanceRow) (*ComplyAttendance, error) {
			upserted = true
			return &ComplyAttendance{
				ID:             uuid.Must(uuid.NewV7()),
				StudentID:      input.StudentID,
				AttendanceDate: input.AttendanceDate,
				Status:         input.Status,
				ManualOverride: true,
				IsAuto:         false,
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.RecordAttendance(context.Background(), studentID, RecordAttendanceCommand{
		AttendanceDate: today,
		Status:         "present_full",
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !upserted {
		t.Fatal("expected upsert to be called")
	}
}

func TestRecordAttendance_RejectsFutureDate(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	future := time.Now().UTC().AddDate(0, 0, 1)

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.RecordAttendance(context.Background(), studentID, RecordAttendanceCommand{
		AttendanceDate: future,
		Status:         "present_full",
	}, *scope)
	if !errors.Is(err, domain.ErrFutureAttendanceDate) {
		t.Fatalf("got %v, want ErrFutureAttendanceDate", err)
	}
}

func TestRecordAttendance_RejectsPartialWithoutDuration(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	today := time.Now().UTC().Truncate(24 * time.Hour)

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.RecordAttendance(context.Background(), studentID, RecordAttendanceCommand{
		AttendanceDate: today,
		Status:         "present_partial",
	}, *scope)
	if !errors.Is(err, domain.ErrDurationRequiredForPartial) {
		t.Fatalf("got %v, want ErrDurationRequiredForPartial", err)
	}
}

func TestRecordAttendance_ValidatesStudentBelongsToFamily(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return false, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.RecordAttendance(context.Background(), studentID, RecordAttendanceCommand{
		AttendanceDate: time.Now().UTC().Truncate(24 * time.Hour),
		Status:         "present_full",
	}, *scope)
	if !errors.Is(err, ErrStudentNotInFamily) {
		t.Fatalf("got %v, want ErrStudentNotInFamily", err)
	}
}

func TestBulkRecordAttendance_CreatesUpTo31Records(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	baseDate := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	attRepo := &stubAttendanceRepo{
		upsertFn: func(_ context.Context, _ shared.FamilyScope, input UpsertAttendanceRow) (*ComplyAttendance, error) {
			return &ComplyAttendance{
				ID:             uuid.Must(uuid.NewV7()),
				StudentID:      input.StudentID,
				AttendanceDate: input.AttendanceDate,
				Status:         input.Status,
				ManualOverride: true,
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	records := make([]RecordAttendanceCommand, 5)
	for i := range records {
		records[i] = RecordAttendanceCommand{
			AttendanceDate: baseDate.AddDate(0, 0, i),
			Status:         "present_full",
		}
	}
	resp, err := svc.BulkRecordAttendance(context.Background(), studentID, BulkRecordAttendanceCommand{Records: records}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 5 {
		t.Fatalf("got %d records, want 5", len(resp))
	}
}

func TestBulkRecordAttendance_RejectsOver31Records(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	records := make([]RecordAttendanceCommand, 32)
	for i := range records {
		records[i] = RecordAttendanceCommand{
			AttendanceDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i),
			Status:         "present_full",
		}
	}
	_, err := svc.BulkRecordAttendance(context.Background(), studentID, BulkRecordAttendanceCommand{Records: records}, *scope)
	if !errors.Is(err, ErrBulkAttendanceLimitExceeded) {
		t.Fatalf("got %v, want ErrBulkAttendanceLimitExceeded", err)
	}
}

func TestBulkRecordAttendance_ValidatesEachRecord(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	future := time.Now().UTC().AddDate(0, 0, 5)

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	records := []RecordAttendanceCommand{
		{AttendanceDate: future, Status: "present_full"}, // future date should fail
	}
	_, err := svc.BulkRecordAttendance(context.Background(), studentID, BulkRecordAttendanceCommand{Records: records}, *scope)
	if !errors.Is(err, domain.ErrFutureAttendanceDate) {
		t.Fatalf("got %v, want ErrFutureAttendanceDate", err)
	}
}

func TestUpdateAttendance_UpdatesExistingRecord(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	attID := uuid.Must(uuid.NewV7())
	newStatus := "absent"

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	attRepo := &stubAttendanceRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID, _ shared.FamilyScope) (*ComplyAttendance, error) {
			if id == attID {
				return &ComplyAttendance{ID: attID, StudentID: studentID, Status: "present_full"}, nil
			}
			return nil, nil
		},
		updateFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, updates UpdateAttendanceRow) (*ComplyAttendance, error) {
			return &ComplyAttendance{ID: attID, StudentID: studentID, Status: *updates.Status}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.UpdateAttendance(context.Background(), studentID, attID, UpdateAttendanceCommand{Status: &newStatus}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "absent" {
		t.Fatalf("got status=%q, want absent", resp.Status)
	}
}

func TestUpdateAttendance_ReturnsNotFound(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	attRepo := &stubAttendanceRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyAttendance, error) {
			return nil, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.UpdateAttendance(context.Background(), studentID, uuid.Must(uuid.NewV7()), UpdateAttendanceCommand{}, *scope)
	if !errors.Is(err, ErrAttendanceNotFound) {
		t.Fatalf("got %v, want ErrAttendanceNotFound", err)
	}
}

func TestDeleteAttendance_DeletesRecord(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	attID := uuid.Must(uuid.NewV7())
	deleted := false

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	attRepo := &stubAttendanceRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyAttendance, error) {
			return &ComplyAttendance{ID: attID, StudentID: studentID}, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) error {
			deleted = true
			return nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.DeleteAttendance(context.Background(), studentID, attID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected attendance to be deleted")
	}
}

func TestListAttendance_ReturnsRecordsInDateRange(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	attRepo := &stubAttendanceRepo{
		listByStudentFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, _ *AttendanceListParams) ([]ComplyAttendance, error) {
			return []ComplyAttendance{
				{ID: uuid.Must(uuid.NewV7()), StudentID: studentID, Status: "present_full"},
				{ID: uuid.Must(uuid.NewV7()), StudentID: studentID, Status: "absent"},
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.ListAttendance(context.Background(), studentID, AttendanceListParams{
		StartDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC),
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Records) != 2 {
		t.Fatalf("got %d records, want 2", len(resp.Records))
	}
}

func TestGetAttendanceSummary_ReturnsCorrectCounts(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	attRepo := &stubAttendanceRepo{
		summarizeFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, _ time.Time, _ time.Time) (*AttendanceSummaryRow, error) {
			return &AttendanceSummaryRow{
				PresentFull:    80,
				PresentPartial: 5,
				Absent:         10,
				NotApplicable:  5,
				TotalMinutes:   25800,
			}, nil
		},
	}
	familyRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return nil, nil // no config
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, familyRepo, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GetAttendanceSummary(context.Background(), studentID, AttendanceSummaryParams{
		StartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PresentFull != 80 || resp.Absent != 10 {
		t.Fatalf("got present_full=%d, absent=%d, want 80, 10", resp.PresentFull, resp.Absent)
	}
}

func TestGetAttendanceSummary_IncludesPaceCalculation(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	reqDays := int16(180)

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	attRepo := &stubAttendanceRepo{
		summarizeFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, _ time.Time, _ time.Time) (*AttendanceSummaryRow, error) {
			return &AttendanceSummaryRow{
				PresentFull:    80,
				PresentPartial: 5,
				Absent:         10,
				NotApplicable:  5,
				TotalMinutes:   25800,
			}, nil
		},
	}
	familyRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return &ComplyFamilyConfig{
				StateCode:       "TX",
				TotalSchoolDays: 180,
				SchoolYearStart: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
				SchoolYearEnd:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
			}, nil
		},
	}
	stateRepo := &stubStateConfigRepo{
		findByStateCodeFn: func(_ context.Context, _ string) (*ComplyStateConfig, error) {
			return &ComplyStateConfig{AttendanceRequired: true, AttendanceDays: &reqDays}, nil
		},
	}
	svc := newTestService(stateRepo, familyRepo, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GetAttendanceSummary(context.Background(), studentID, AttendanceSummaryParams{
		StartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PaceStatus == nil {
		t.Fatal("expected non-nil pace_status")
	}
	if resp.StateRequiredDays == nil || *resp.StateRequiredDays != 180 {
		t.Fatal("expected state_required_days=180")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group E: Assessments (E1–E6)
// ═══════════════════════════════════════════════════════════════════════════════

func TestCreateAssessment_CreatesRecord(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	assessRepo := &stubAssessmentRepo{
		createFn: func(_ context.Context, _ shared.FamilyScope, input CreateAssessmentRow) (*ComplyAssessmentRecord, error) {
			return &ComplyAssessmentRecord{
				ID:             uuid.Must(uuid.NewV7()),
				FamilyID:       scope.FamilyID(),
				StudentID:      input.StudentID,
				Title:          input.Title,
				Subject:        input.Subject,
				AssessmentType: input.AssessmentType,
				AssessmentDate: input.AssessmentDate,
				CreatedAt:      time.Now().UTC(),
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, assessRepo, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.CreateAssessment(context.Background(), studentID, CreateAssessmentCommand{
		Title:          "Math Quiz",
		Subject:        "Mathematics",
		AssessmentType: "quiz",
		AssessmentDate: time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Title != "Math Quiz" {
		t.Fatalf("got title=%q, want Math Quiz", resp.Title)
	}
}

func TestCreateAssessment_ValidatesStudentBelongsToFamily(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return false, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.CreateAssessment(context.Background(), studentID, CreateAssessmentCommand{
		Title:          "Math Quiz",
		Subject:        "Mathematics",
		AssessmentType: "quiz",
		AssessmentDate: time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
	}, *scope)
	if !errors.Is(err, ErrStudentNotInFamily) {
		t.Fatalf("got %v, want ErrStudentNotInFamily", err)
	}
}

func TestListAssessments_FiltersCorrectly(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	assessRepo := &stubAssessmentRepo{
		listByStudentFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, _ *AssessmentListParams) ([]ComplyAssessmentRecord, error) {
			return []ComplyAssessmentRecord{
				{ID: uuid.Must(uuid.NewV7()), Title: "Math Quiz", Subject: "Mathematics"},
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, assessRepo, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	subject := "Mathematics"
	resp, err := svc.ListAssessments(context.Background(), studentID, AssessmentListParams{Subject: &subject}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Records) != 1 {
		t.Fatalf("got %d records, want 1", len(resp.Records))
	}
}

func TestUpdateAssessment_UpdatesExisting(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	assessID := uuid.Must(uuid.NewV7())
	newTitle := "Updated Quiz"

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	assessRepo := &stubAssessmentRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID, _ shared.FamilyScope) (*ComplyAssessmentRecord, error) {
			if id == assessID {
				return &ComplyAssessmentRecord{ID: assessID, StudentID: studentID}, nil
			}
			return nil, nil
		},
		updateFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, updates UpdateAssessmentRow) (*ComplyAssessmentRecord, error) {
			return &ComplyAssessmentRecord{ID: assessID, StudentID: studentID, Title: *updates.Title}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, assessRepo, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.UpdateAssessment(context.Background(), studentID, assessID, UpdateAssessmentCommand{Title: &newTitle}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Title != "Updated Quiz" {
		t.Fatalf("got title=%q, want Updated Quiz", resp.Title)
	}
}

func TestUpdateAssessment_ReturnsNotFound(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	assessRepo := &stubAssessmentRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyAssessmentRecord, error) {
			return nil, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, assessRepo, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.UpdateAssessment(context.Background(), studentID, uuid.Must(uuid.NewV7()), UpdateAssessmentCommand{}, *scope)
	if !errors.Is(err, ErrAssessmentNotFound) {
		t.Fatalf("got %v, want ErrAssessmentNotFound", err)
	}
}

func TestDeleteAssessment_DeletesRecord(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	assessID := uuid.Must(uuid.NewV7())
	deleted := false

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	assessRepo := &stubAssessmentRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyAssessmentRecord, error) {
			return &ComplyAssessmentRecord{ID: assessID, StudentID: studentID}, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) error {
			deleted = true
			return nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, assessRepo, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.DeleteAssessment(context.Background(), studentID, assessID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected assessment to be deleted")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group F: Standardized Tests (F1–F5)
// ═══════════════════════════════════════════════════════════════════════════════

func TestCreateTestScore_StoresJSONBScores(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	scores := json.RawMessage(`{"math": 95, "reading": 90}`)

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	testRepo := &stubTestScoreRepo{
		createFn: func(_ context.Context, _ shared.FamilyScope, input CreateTestScoreRow) (*ComplyStandardizedTest, error) {
			return &ComplyStandardizedTest{
				ID:        uuid.Must(uuid.NewV7()),
				StudentID: input.StudentID,
				TestName:  input.TestName,
				TestDate:  input.TestDate,
				Scores:    input.Scores,
				CreatedAt: time.Now().UTC(),
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, testRepo, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.CreateTestScore(context.Background(), studentID, CreateTestScoreCommand{
		TestName: "SAT",
		TestDate: time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
		Scores:   scores,
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TestName != "SAT" {
		t.Fatalf("got test_name=%q, want SAT", resp.TestName)
	}
}

func TestCreateTestScore_ValidatesStudentBelongsToFamily(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return false, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.CreateTestScore(context.Background(), studentID, CreateTestScoreCommand{
		TestName: "SAT",
		TestDate: time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
		Scores:   json.RawMessage(`{}`),
	}, *scope)
	if !errors.Is(err, ErrStudentNotInFamily) {
		t.Fatalf("got %v, want ErrStudentNotInFamily", err)
	}
}

func TestListTestScores_ReturnsSorted(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	testRepo := &stubTestScoreRepo{
		listByStudentFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, _ *TestListParams) ([]ComplyStandardizedTest, error) {
			return []ComplyStandardizedTest{
				{ID: uuid.Must(uuid.NewV7()), TestName: "SAT"},
				{ID: uuid.Must(uuid.NewV7()), TestName: "ACT"},
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, testRepo, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.ListTestScores(context.Background(), studentID, TestListParams{}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tests) != 2 {
		t.Fatalf("got %d tests, want 2", len(resp.Tests))
	}
}

func TestUpdateTestScore_UpdatesExisting(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	testID := uuid.Must(uuid.NewV7())
	newName := "Updated SAT"

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	testRepo := &stubTestScoreRepo{
		updateFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, updates UpdateTestScoreRow) (*ComplyStandardizedTest, error) {
			return &ComplyStandardizedTest{ID: testID, StudentID: studentID, TestName: *updates.TestName}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, testRepo, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.UpdateTestScore(context.Background(), studentID, testID, UpdateTestScoreCommand{TestName: &newName}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TestName != "Updated SAT" {
		t.Fatalf("got test_name=%q, want Updated SAT", resp.TestName)
	}
}

func TestDeleteTestScore_DeletesRecord(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	testID := uuid.Must(uuid.NewV7())
	deleted := false

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	testRepo := &stubTestScoreRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) error {
			deleted = true
			return nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, testRepo, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.DeleteTestScore(context.Background(), studentID, testID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected test score to be deleted")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group G: Portfolios (G1–G14)
// ═══════════════════════════════════════════════════════════════════════════════

func TestCreatePortfolio_CreatesInConfiguringStatus(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		createFn: func(_ context.Context, _ shared.FamilyScope, input CreatePortfolioRow) (*ComplyPortfolio, error) {
			return &ComplyPortfolio{
				ID:             uuid.Must(uuid.NewV7()),
				FamilyID:       scope.FamilyID(),
				StudentID:      input.StudentID,
				Title:          input.Title,
				Organization:   input.Organization,
				DateRangeStart: input.DateRangeStart,
				DateRangeEnd:   input.DateRangeEnd,
				Status:         string(PortfolioStatusConfiguring),
				CreatedAt:      time.Now().UTC(),
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.CreatePortfolio(context.Background(), studentID, CreatePortfolioCommand{
		Title:          "2025 Portfolio",
		Organization:   "by_subject",
		DateRangeStart: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		DateRangeEnd:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != string(PortfolioStatusConfiguring) {
		t.Fatalf("got status=%q, want configuring", resp.Status)
	}
}

func TestCreatePortfolio_RejectsInvalidDateRange(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.CreatePortfolio(context.Background(), studentID, CreatePortfolioCommand{
		Title:          "Bad Portfolio",
		Organization:   "by_subject",
		DateRangeStart: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		DateRangeEnd:   time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
	}, *scope)
	if !errors.Is(err, ErrInvalidSchoolYearRange) {
		t.Fatalf("got %v, want ErrInvalidSchoolYearRange", err)
	}
}

func TestAddPortfolioItems_CachesDisplayData(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	portfolioID := uuid.Must(uuid.NewV7())
	sourceID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			return &ComplyPortfolio{ID: portfolioID, StudentID: studentID, Status: "configuring"}, nil
		},
	}
	learnSvc := &stubLearningService{
		getPortfolioItemDataFn: func(_ context.Context, _ string, _ uuid.UUID) (*PortfolioItemData, error) {
			return &PortfolioItemData{
				Title:   "Math Activity",
				Date:    time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
				Subject: strPtr("Mathematics"),
			}, nil
		},
	}
	itemRepo := &stubPortfolioItemRepo{
		createBatchFn: func(_ context.Context, items []CreatePortfolioItemRow) ([]ComplyPortfolioItem, error) {
			result := make([]ComplyPortfolioItem, len(items))
			for i, item := range items {
				result[i] = ComplyPortfolioItem{
					ID:          uuid.Must(uuid.NewV7()),
					PortfolioID: item.PortfolioID,
					SourceType:  item.SourceType,
					SourceID:    item.SourceID,
					CachedTitle: item.CachedTitle,
					CachedDate:  item.CachedDate,
				}
			}
			return result, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, itemRepo, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, learnSvc, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.AddPortfolioItems(context.Background(), studentID, portfolioID, AddPortfolioItemsCommand{
		Items: []PortfolioItemInput{{SourceType: "activity", SourceID: sourceID}},
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0].CachedTitle != "Math Activity" {
		t.Fatalf("got %+v, want 1 item with cached title 'Math Activity'", resp)
	}
}

func TestAddPortfolioItems_RejectsNonConfiguringStatus(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	portfolioID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			return &ComplyPortfolio{ID: portfolioID, StudentID: studentID, Status: "ready"}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.AddPortfolioItems(context.Background(), studentID, portfolioID, AddPortfolioItemsCommand{
		Items: []PortfolioItemInput{{SourceType: "activity", SourceID: uuid.Must(uuid.NewV7())}},
	}, *scope)
	if !errors.Is(err, domain.ErrPortfolioNotConfiguring) {
		t.Fatalf("got %v, want ErrPortfolioNotConfiguring", err)
	}
}

func TestAddPortfolioItems_RejectsSourceNotFound(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	portfolioID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			return &ComplyPortfolio{ID: portfolioID, StudentID: studentID, Status: "configuring"}, nil
		},
	}
	learnSvc := &stubLearningService{
		getPortfolioItemDataFn: func(_ context.Context, _ string, _ uuid.UUID) (*PortfolioItemData, error) {
			return nil, nil // not found
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, learnSvc, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.AddPortfolioItems(context.Background(), studentID, portfolioID, AddPortfolioItemsCommand{
		Items: []PortfolioItemInput{{SourceType: "activity", SourceID: uuid.Must(uuid.NewV7())}},
	}, *scope)
	if !errors.Is(err, ErrPortfolioItemSourceNotFound) {
		t.Fatalf("got %v, want ErrPortfolioItemSourceNotFound", err)
	}
}

func TestGeneratePortfolio_TransitionsToGenerating(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	portfolioID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			return &ComplyPortfolio{
				ID:        portfolioID,
				StudentID: studentID,
				Status:    "configuring",
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status string, _ *uuid.UUID, _ *string) (*ComplyPortfolio, error) {
			return &ComplyPortfolio{
				ID:        portfolioID,
				StudentID: studentID,
				Status:    status,
			}, nil
		},
	}
	itemRepo := &stubPortfolioItemRepo{
		listByPortfolioFn: func(_ context.Context, _ uuid.UUID) ([]ComplyPortfolioItem, error) {
			return make([]ComplyPortfolioItem, 5), nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, itemRepo, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GeneratePortfolio(context.Background(), studentID, portfolioID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "generating" {
		t.Fatalf("got status=%q, want generating", resp.Status)
	}
}

func TestGeneratePortfolio_RejectsEmptyPortfolio(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	portfolioID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			return &ComplyPortfolio{ID: portfolioID, StudentID: studentID, Status: "configuring"}, nil
		},
	}
	itemRepo := &stubPortfolioItemRepo{
		listByPortfolioFn: func(_ context.Context, _ uuid.UUID) ([]ComplyPortfolioItem, error) {
			return nil, nil // empty
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, itemRepo, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.GeneratePortfolio(context.Background(), studentID, portfolioID, *scope)
	if !errors.Is(err, domain.ErrEmptyPortfolio) {
		t.Fatalf("got %v, want ErrEmptyPortfolio", err)
	}
}

func TestGeneratePortfolio_RejectsNonConfiguring(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	portfolioID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			return &ComplyPortfolio{ID: portfolioID, StudentID: studentID, Status: "ready"}, nil
		},
	}
	itemRepo := &stubPortfolioItemRepo{
		listByPortfolioFn: func(_ context.Context, _ uuid.UUID) ([]ComplyPortfolioItem, error) {
			return make([]ComplyPortfolioItem, 5), nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, itemRepo, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.GeneratePortfolio(context.Background(), studentID, portfolioID, *scope)
	if !errors.Is(err, domain.ErrPortfolioNotConfiguring) {
		t.Fatalf("got %v, want ErrPortfolioNotConfiguring", err)
	}
}

func TestGetPortfolio_ReturnsPortfolioWithItems(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	portfolioID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			return &ComplyPortfolio{
				ID:        portfolioID,
				StudentID: studentID,
				Title:     "Portfolio",
				Status:    "configuring",
			}, nil
		},
	}
	itemRepo := &stubPortfolioItemRepo{
		listByPortfolioFn: func(_ context.Context, _ uuid.UUID) ([]ComplyPortfolioItem, error) {
			return []ComplyPortfolioItem{
				{ID: uuid.Must(uuid.NewV7()), CachedTitle: "Item 1"},
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, itemRepo, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GetPortfolio(context.Background(), studentID, portfolioID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Title != "Portfolio" || len(resp.Items) != 1 {
		t.Fatalf("got title=%q items=%d, want Portfolio with 1 item", resp.Title, len(resp.Items))
	}
}

func TestGetPortfolio_ReturnsNotFound(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			return nil, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.GetPortfolio(context.Background(), studentID, uuid.Must(uuid.NewV7()), *scope)
	if !errors.Is(err, ErrPortfolioNotFound) {
		t.Fatalf("got %v, want ErrPortfolioNotFound", err)
	}
}

func TestListPortfolios_ReturnsStudentPortfolios(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		listByStudentFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) ([]ComplyPortfolio, error) {
			return []ComplyPortfolio{
				{ID: uuid.Must(uuid.NewV7()), Title: "Portfolio 1", Status: "configuring"},
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.ListPortfolios(context.Background(), studentID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("got %d portfolios, want 1", len(resp))
	}
}

func TestGetPortfolioDownloadURL_ReturnsPresignedURL(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	portfolioID := uuid.Must(uuid.NewV7())
	uploadID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			exp := time.Now().UTC().Add(24 * time.Hour)
			return &ComplyPortfolio{
				ID:        portfolioID,
				StudentID: studentID,
				Status:    "ready",
				UploadID:  &uploadID,
				ExpiresAt: &exp,
			}, nil
		},
	}
	mediaSvc := &stubMediaService{
		presignedGetFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return "https://storage.example.com/portfolio.pdf", nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, mediaSvc)

	url, err := svc.GetPortfolioDownloadURL(context.Background(), studentID, portfolioID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://storage.example.com/portfolio.pdf" {
		t.Fatalf("got url=%q, want presigned URL", url)
	}
}

func TestGetPortfolioDownloadURL_RejectsNonReady(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	portfolioID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			return &ComplyPortfolio{ID: portfolioID, StudentID: studentID, Status: "configuring"}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.GetPortfolioDownloadURL(context.Background(), studentID, portfolioID, *scope)
	if !errors.Is(err, domain.ErrPortfolioNotConfiguring) {
		t.Fatalf("got %v, want ErrPortfolioNotConfiguring", err)
	}
}

func TestGetPortfolioDownloadURL_RejectsExpired(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	portfolioID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	portfolioRepo := &stubPortfolioRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyPortfolio, error) {
			exp := time.Now().UTC().Add(-1 * time.Hour)
			return &ComplyPortfolio{
				ID:        portfolioID,
				StudentID: studentID,
				Status:    "ready",
				ExpiresAt: &exp,
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, portfolioRepo, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.GetPortfolioDownloadURL(context.Background(), studentID, portfolioID, *scope)
	if !errors.Is(err, ErrPortfolioExpired) {
		t.Fatalf("got %v, want ErrPortfolioExpired", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group H: Event Handlers (H1–H5)
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleActivityLogged_CreatesAutoAttendance(t *testing.T) {
	created := false
	attRepo := &stubAttendanceRepo{
		upsertFn: func(_ context.Context, _ shared.FamilyScope, input UpsertAttendanceRow) (*ComplyAttendance, error) {
			if !input.IsAuto {
				t.Fatal("expected is_auto=true for auto-attendance")
			}
			created = true
			return &ComplyAttendance{ID: uuid.Must(uuid.NewV7())}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.HandleActivityLogged(context.Background(), &ActivityLoggedEvent{
		FamilyID:     uuid.Must(uuid.NewV7()),
		StudentID:    uuid.Must(uuid.NewV7()),
		ActivityID:   uuid.Must(uuid.NewV7()),
		ActivityDate: time.Now().UTC().Truncate(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("expected auto-attendance to be created")
	}
}

func TestHandleActivityLogged_DoesNotOverrideManual(t *testing.T) {
	attRepo := &stubAttendanceRepo{
		upsertFn: func(_ context.Context, _ shared.FamilyScope, input UpsertAttendanceRow) (*ComplyAttendance, error) {
			// Repo upsert should be called with is_auto=true, manual_override=false
			// The repo handles the "don't override manual" logic via ON CONFLICT
			if input.ManualOverride {
				t.Fatal("auto-attendance should not set manual_override=true")
			}
			return &ComplyAttendance{ID: uuid.Must(uuid.NewV7())}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.HandleActivityLogged(context.Background(), &ActivityLoggedEvent{
		FamilyID:     uuid.Must(uuid.NewV7()),
		StudentID:    uuid.Must(uuid.NewV7()),
		ActivityID:   uuid.Must(uuid.NewV7()),
		ActivityDate: time.Now().UTC().Truncate(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleStudentDeleted_CascadesDelete(t *testing.T) {
	attDeleted := false
	assessDeleted := false
	familyID := uuid.Must(uuid.NewV7())
	studentID := uuid.Must(uuid.NewV7())

	attRepo := &stubAttendanceRepo{
		deleteByStudentFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
			attDeleted = true
			return nil
		},
	}
	assessRepo := &stubAssessmentRepo{
		deleteByStudentFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
			assessDeleted = true
			return nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, attRepo, assessRepo, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.HandleStudentDeleted(context.Background(), &StudentDeletedEvent{
		FamilyID:  familyID,
		StudentID: studentID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !attDeleted || !assessDeleted {
		t.Fatal("expected all student data to be cascade deleted")
	}
}

func TestHandleFamilyDeletionScheduled_CascadesDelete(t *testing.T) {
	attDeleted := false
	configDeleted := false
	familyID := uuid.Must(uuid.NewV7())

	attRepo := &stubAttendanceRepo{
		deleteByFamilyFn: func(_ context.Context, _ uuid.UUID) error {
			attDeleted = true
			return nil
		},
	}
	familyRepo := &stubFamilyConfigRepo{
		deleteByFamilyFn: func(_ context.Context, _ uuid.UUID) error {
			configDeleted = true
			return nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, familyRepo, &stubScheduleRepo{}, attRepo, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.HandleFamilyDeletionScheduled(context.Background(), &FamilyDeletionScheduledEvent{
		FamilyID:    familyID,
		DeleteAfter: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !attDeleted || !configDeleted {
		t.Fatal("expected all family data to be cascade deleted")
	}
}

func TestHandleSubscriptionCancelled_PreservesData(t *testing.T) {
	// No repos should be called — data is preserved
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.HandleSubscriptionCancelled(context.Background(), &SubscriptionCancelledEvent{
		FamilyID:    uuid.Must(uuid.NewV7()),
		EffectiveAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group I: Dashboard (I1–I2)
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetDashboard_ReturnsNullConfigWhenUnconfigured(t *testing.T) {
	scope := testScope()
	familyRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return nil, nil
		},
	}
	iamSvc := &stubIamService{}
	svc := newTestService(&stubStateConfigRepo{}, familyRepo, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GetDashboard(context.Background(), *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FamilyConfig != nil {
		t.Fatal("expected nil family_config for unconfigured family")
	}
}

func TestGetDashboard_ReturnsStudentSummaries(t *testing.T) {
	scope := testScope()
	familyRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return &ComplyFamilyConfig{
				FamilyID:        scope.FamilyID(),
				StateCode:       "TX",
				TotalSchoolDays: 180,
				SchoolYearStart: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
				SchoolYearEnd:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
				GpaScale:        "standard_4",
			}, nil
		},
	}
	stateRepo := &stubStateConfigRepo{
		findByStateCodeFn: func(_ context.Context, _ string) (*ComplyStateConfig, error) {
			return &ComplyStateConfig{StateCode: "TX", StateName: "Texas"}, nil
		},
	}
	svc := newTestService(stateRepo, familyRepo, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, &stubIamService{}, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GetDashboard(context.Background(), *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FamilyConfig == nil {
		t.Fatal("expected non-nil family_config")
	}
	if resp.FamilyConfig.StateCode != "TX" {
		t.Fatalf("got state_code=%q, want TX", resp.FamilyConfig.StateCode)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group J: Transcripts (J1–J7)
// ═══════════════════════════════════════════════════════════════════════════════

func TestCreateTranscript_CreatesInConfiguringStatus(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
		getStudentNameFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return "Alice Johnson", nil
		},
	}
	transcriptRepo := &stubTranscriptRepo{
		createFn: func(_ context.Context, _ shared.FamilyScope, input CreateTranscriptRow) (*ComplyTranscript, error) {
			return &ComplyTranscript{
				ID:          uuid.Must(uuid.NewV7()),
				FamilyID:    scope.FamilyID(),
				StudentID:   input.StudentID,
				Title:       input.Title,
				StudentName: input.StudentName,
				GradeLevels: input.GradeLevels,
				Status:      string(PortfolioStatusConfiguring),
				CreatedAt:   time.Now().UTC(),
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, transcriptRepo, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.CreateTranscript(context.Background(), studentID, CreateTranscriptCommand{
		Title:       "High School Transcript",
		GradeLevels: []string{"9", "10"},
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "configuring" {
		t.Fatalf("got status=%q, want configuring", resp.Status)
	}
	if resp.StudentName != "Alice Johnson" {
		t.Fatalf("got student_name=%q, want Alice Johnson", resp.StudentName)
	}
}

func TestGenerateTranscript_TransitionsToGenerating(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	transcriptID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	transcriptRepo := &stubTranscriptRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID, _ shared.FamilyScope) (*ComplyTranscript, error) {
			if id == transcriptID {
				return &ComplyTranscript{
					ID:     transcriptID,
					Status: string(PortfolioStatusConfiguring),
				}, nil
			}
			return nil, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status string, _ *uuid.UUID, _ *float64, _ *float64, _ *string) (*ComplyTranscript, error) {
			return &ComplyTranscript{
				ID:     transcriptID,
				Status: status,
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, transcriptRepo, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GenerateTranscript(context.Background(), studentID, transcriptID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "generating" {
		t.Fatalf("got status=%q, want generating", resp.Status)
	}
}

func TestGenerateTranscript_RejectsNonConfiguring(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	transcriptID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	transcriptRepo := &stubTranscriptRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyTranscript, error) {
			return &ComplyTranscript{
				ID:     transcriptID,
				Status: string(PortfolioStatusReady),
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, transcriptRepo, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.GenerateTranscript(context.Background(), studentID, transcriptID, *scope)
	var transErr *domain.InvalidPortfolioTransitionError
	if !errors.As(err, &transErr) {
		t.Fatalf("got %v, want InvalidPortfolioTransitionError", err)
	}
}

func TestGetTranscript_ReturnsCourseAndGPA(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	transcriptID := uuid.Must(uuid.NewV7())
	gp := 3.5

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	transcriptRepo := &stubTranscriptRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyTranscript, error) {
			return &ComplyTranscript{
				ID:          transcriptID,
				StudentID:   studentID,
				Title:       "HS Transcript",
				StudentName: "Alice",
				Status:      "configuring",
				CreatedAt:   time.Now().UTC(),
			}, nil
		},
	}
	courseRepo := &stubCourseRepo{
		listByStudentFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, _ *CourseListParams) ([]ComplyCourse, error) {
			return []ComplyCourse{
				{ID: uuid.Must(uuid.NewV7()), StudentID: studentID, Title: "Algebra I", Credits: 1.0, GradePoints: &gp, Level: "regular", GradeLevel: 9},
			}, nil
		},
	}
	familyConfigRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return &ComplyFamilyConfig{GpaScale: "standard_4"}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, familyConfigRepo, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, transcriptRepo, courseRepo, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.GetTranscript(context.Background(), studentID, transcriptID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Courses) != 1 {
		t.Fatalf("got %d courses, want 1", len(resp.Courses))
	}
	if resp.GPAUnweighted == nil || *resp.GPAUnweighted != 3.5 {
		t.Fatalf("got gpa_unweighted=%v, want 3.5", resp.GPAUnweighted)
	}
}

func TestListTranscripts_ReturnsForStudent(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	transcriptRepo := &stubTranscriptRepo{
		listByStudentFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) ([]ComplyTranscript, error) {
			return []ComplyTranscript{
				{ID: uuid.Must(uuid.NewV7()), Title: "9th Grade", Status: "configuring", GradeLevels: []string{"9"}, CreatedAt: time.Now().UTC()},
				{ID: uuid.Must(uuid.NewV7()), Title: "10th Grade", Status: "ready", GradeLevels: []string{"10"}, CreatedAt: time.Now().UTC()},
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, transcriptRepo, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	results, err := svc.ListTranscripts(context.Background(), studentID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d transcripts, want 2", len(results))
	}
	if results[0].Title != "9th Grade" {
		t.Fatalf("got title=%q, want 9th Grade", results[0].Title)
	}
}

func TestDeleteTranscript_DeletesExisting(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	transcriptID := uuid.Must(uuid.NewV7())
	deleted := false

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	transcriptRepo := &stubTranscriptRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyTranscript, error) {
			return &ComplyTranscript{ID: transcriptID, Status: "configuring"}, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) error {
			deleted = true
			return nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, transcriptRepo, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.DeleteTranscript(context.Background(), studentID, transcriptID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected transcript to be deleted")
	}
}

func TestGetTranscriptDownloadURL_ReturnsPresignedURL(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	transcriptID := uuid.Must(uuid.NewV7())
	uploadID := uuid.Must(uuid.NewV7())
	future := time.Now().UTC().Add(24 * time.Hour)

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	transcriptRepo := &stubTranscriptRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (*ComplyTranscript, error) {
			return &ComplyTranscript{
				ID:        transcriptID,
				Status:    string(PortfolioStatusReady),
				UploadID:  &uploadID,
				ExpiresAt: &future,
			}, nil
		},
	}
	mediaSvc := &stubMediaService{
		presignedGetFn: func(_ context.Context, id uuid.UUID) (string, error) {
			if id == uploadID {
				return "https://cdn.example.com/transcript.pdf?sig=abc", nil
			}
			return "", nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, transcriptRepo, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, mediaSvc)

	url, err := svc.GetTranscriptDownloadURL(context.Background(), studentID, transcriptID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://cdn.example.com/transcript.pdf?sig=abc" {
		t.Fatalf("got url=%q, want presigned URL", url)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group K: Courses (K1–K5)
// ═══════════════════════════════════════════════════════════════════════════════

func TestCreateCourse_CreatesWithValidLevel(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	gp := 3.7

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	courseRepo := &stubCourseRepo{
		createFn: func(_ context.Context, _ shared.FamilyScope, input CreateCourseRow) (*ComplyCourse, error) {
			return &ComplyCourse{
				ID:          uuid.Must(uuid.NewV7()),
				FamilyID:    scope.FamilyID(),
				StudentID:   input.StudentID,
				Title:       input.Title,
				Subject:     input.Subject,
				GradeLevel:  input.GradeLevel,
				Credits:     input.Credits,
				GradeLetter: input.GradeLetter,
				GradePoints: input.GradePoints,
				Level:       input.Level,
				SchoolYear:  input.SchoolYear,
				Semester:    input.Semester,
				CreatedAt:   time.Now().UTC(),
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, courseRepo, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.CreateCourse(context.Background(), studentID, CreateCourseCommand{
		Title:       "Algebra I",
		Subject:     "Mathematics",
		GradeLevel:  9,
		Credits:     1.0,
		GradePoints: &gp,
		Level:       "regular",
		SchoolYear:  "2025-2026",
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Title != "Algebra I" {
		t.Fatalf("got title=%q, want Algebra I", resp.Title)
	}
	if resp.Level != "regular" {
		t.Fatalf("got level=%q, want regular", resp.Level)
	}
}

func TestCreateCourse_ValidatesStudentInFamily(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return false, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, &stubCourseRepo{}, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	_, err := svc.CreateCourse(context.Background(), studentID, CreateCourseCommand{
		Title:      "Algebra I",
		Subject:    "Mathematics",
		GradeLevel: 9,
		Credits:    1.0,
		Level:      "regular",
		SchoolYear: "2025-2026",
	}, *scope)
	if !errors.Is(err, ErrStudentNotInFamily) {
		t.Fatalf("got %v, want ErrStudentNotInFamily", err)
	}
}

func TestListCourses_FiltersByGradeLevelAndSchoolYear(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	var capturedParams *CourseListParams
	courseRepo := &stubCourseRepo{
		listByStudentFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, params *CourseListParams) ([]ComplyCourse, error) {
			capturedParams = params
			return []ComplyCourse{
				{ID: uuid.Must(uuid.NewV7()), Title: "Algebra I", GradeLevel: 9, SchoolYear: "2025-2026", Level: "regular", Credits: 1.0},
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, courseRepo, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	gl := int16(9)
	sy := "2025-2026"
	resp, err := svc.ListCourses(context.Background(), studentID, CourseListParams{GradeLevel: &gl, SchoolYear: &sy}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Courses) != 1 {
		t.Fatalf("got %d courses, want 1", len(resp.Courses))
	}
	if capturedParams == nil || capturedParams.GradeLevel == nil || *capturedParams.GradeLevel != 9 {
		t.Fatal("expected params to pass through grade_level=9")
	}
}

func TestUpdateCourse_UpdatesExisting(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	courseID := uuid.Must(uuid.NewV7())
	newTitle := "Algebra II"

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	courseRepo := &stubCourseRepo{
		updateFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, updates UpdateCourseRow) (*ComplyCourse, error) {
			return &ComplyCourse{
				ID:        courseID,
				StudentID: studentID,
				Title:     *updates.Title,
				Level:     "regular",
				Credits:   1.0,
			}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, courseRepo, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.UpdateCourse(context.Background(), studentID, courseID, UpdateCourseCommand{Title: &newTitle}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Title != "Algebra II" {
		t.Fatalf("got title=%q, want Algebra II", resp.Title)
	}
}

func TestDeleteCourse_DeletesExisting(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	courseID := uuid.Must(uuid.NewV7())
	deleted := false

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	courseRepo := &stubCourseRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) error {
			deleted = true
			return nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, &stubFamilyConfigRepo{}, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, courseRepo, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	err := svc.DeleteCourse(context.Background(), studentID, courseID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected course to be deleted")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group L: GPA Service (L1–L3)
// ═══════════════════════════════════════════════════════════════════════════════

func TestCalculateGPA_ReturnsWithGradeLevelBreakdown(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	gp4 := 4.0
	gp3 := 3.0

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	courseRepo := &stubCourseRepo{
		listByStudentFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, _ *CourseListParams) ([]ComplyCourse, error) {
			return []ComplyCourse{
				{ID: uuid.Must(uuid.NewV7()), Title: "Algebra I", GradeLevel: 9, Credits: 1.0, GradePoints: &gp4, Level: "regular"},
				{ID: uuid.Must(uuid.NewV7()), Title: "English 9", GradeLevel: 9, Credits: 1.0, GradePoints: &gp3, Level: "regular"},
				{ID: uuid.Must(uuid.NewV7()), Title: "Biology", GradeLevel: 10, Credits: 1.0, GradePoints: &gp4, Level: "honors"},
			}, nil
		},
	}
	familyConfigRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return &ComplyFamilyConfig{GpaScale: "standard_4"}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, familyConfigRepo, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, courseRepo, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.CalculateGPA(context.Background(), studentID, GpaParams{}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCourses != 3 {
		t.Fatalf("got total_courses=%d, want 3", resp.TotalCourses)
	}
	if resp.TotalCredits != 3.0 {
		t.Fatalf("got total_credits=%v, want 3.0", resp.TotalCredits)
	}
	if len(resp.ByGradeLevel) != 2 {
		t.Fatalf("got %d grade levels, want 2", len(resp.ByGradeLevel))
	}
	// Grade 9: (4.0*1 + 3.0*1)/2 = 3.5
	if resp.ByGradeLevel[0].GradeLevel != 9 {
		t.Fatalf("got first grade_level=%d, want 9", resp.ByGradeLevel[0].GradeLevel)
	}
	if resp.ByGradeLevel[0].Unweighted != 3.5 {
		t.Fatalf("got grade 9 unweighted=%v, want 3.5", resp.ByGradeLevel[0].Unweighted)
	}
	// Grade 10: 4.0/1 = 4.0 unweighted, (4.0+0.5)/1 = 4.5 weighted (honors)
	if resp.ByGradeLevel[1].Weighted != 4.5 {
		t.Fatalf("got grade 10 weighted=%v, want 4.5", resp.ByGradeLevel[1].Weighted)
	}
}

func TestCalculateGPAWhatIf_ProjectsWithHypotheticalCourses(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	gp3 := 3.0

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	courseRepo := &stubCourseRepo{
		listByStudentFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, _ *CourseListParams) ([]ComplyCourse, error) {
			return []ComplyCourse{
				{ID: uuid.Must(uuid.NewV7()), Title: "Algebra I", Credits: 1.0, GradePoints: &gp3, Level: "regular"},
			}, nil
		},
	}
	familyConfigRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return &ComplyFamilyConfig{GpaScale: "standard_4"}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, familyConfigRepo, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, courseRepo, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	resp, err := svc.CalculateGPAWhatIf(context.Background(), studentID, GpaWhatIfParams{
		AdditionalCourses: []WhatIfCourse{
			{Credits: 1.0, GradePoints: 4.0, Level: "regular"},
		},
	}, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCourses != 2 {
		t.Fatalf("got total_courses=%d, want 2", resp.TotalCourses)
	}
	// (3.0*1 + 4.0*1) / 2 = 3.5
	if resp.UnweightedGPA != 3.5 {
		t.Fatalf("got unweighted_gpa=%v, want 3.5", resp.UnweightedGPA)
	}
}

func TestGetGPAHistory_ReturnsBySchoolYear(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	gp4 := 4.0
	gp3 := 3.0
	fall := "fall"
	spring := "spring"

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return true, nil
		},
	}
	courseRepo := &stubCourseRepo{
		listByStudentFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, _ *CourseListParams) ([]ComplyCourse, error) {
			return []ComplyCourse{
				{ID: uuid.Must(uuid.NewV7()), Title: "Algebra I", Credits: 1.0, GradePoints: &gp4, Level: "regular", SchoolYear: "2024-2025", Semester: &fall},
				{ID: uuid.Must(uuid.NewV7()), Title: "English 9", Credits: 1.0, GradePoints: &gp3, Level: "regular", SchoolYear: "2024-2025", Semester: &spring},
				{ID: uuid.Must(uuid.NewV7()), Title: "Geometry", Credits: 1.0, GradePoints: &gp4, Level: "regular", SchoolYear: "2025-2026", Semester: &fall},
			}, nil
		},
	}
	familyConfigRepo := &stubFamilyConfigRepo{
		findByFamilyFn: func(_ context.Context, _ shared.FamilyScope) (*ComplyFamilyConfig, error) {
			return &ComplyFamilyConfig{GpaScale: "standard_4"}, nil
		},
	}
	svc := newTestService(&stubStateConfigRepo{}, familyConfigRepo, &stubScheduleRepo{}, &stubAttendanceRepo{}, &stubAssessmentRepo{}, &stubTestScoreRepo{}, &stubPortfolioRepo{}, &stubPortfolioItemRepo{}, &stubTranscriptRepo{}, courseRepo, iamSvc, &stubLearningService{}, &stubDiscoveryService{}, &stubMediaService{})

	results, err := svc.GetGPAHistory(context.Background(), studentID, *scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d terms, want 3", len(results))
	}
	// Sorted: 2024-2025 fall, 2024-2025 spring, 2025-2026 fall
	if results[0].SchoolYear != "2024-2025" || *results[0].Semester != "fall" {
		t.Fatalf("got first term=%s/%v, want 2024-2025/fall", results[0].SchoolYear, results[0].Semester)
	}
	if results[0].UnweightedGPA != 4.0 {
		t.Fatalf("got first term gpa=%v, want 4.0", results[0].UnweightedGPA)
	}
	if results[2].SchoolYear != "2025-2026" {
		t.Fatalf("got last school_year=%q, want 2025-2026", results[2].SchoolYear)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func int16Ptr(v int16) *int16 { return &v }
func strPtr(s string) *string { return &s }
