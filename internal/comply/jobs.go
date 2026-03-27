package comply

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/comply/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// Task type constants for background job routing. [14-comply §9]
const (
	TaskTypeSyncStateConfigs        = "comply:sync_state_configs"
	TaskTypeGeneratePortfolio       = "comply:generate_portfolio"
	TaskTypeGenerateTranscript      = "comply:generate_transcript"
	TaskTypeAttendanceThresholdCheck = "comply:attendance_threshold_check"
)

// ─── Task Payload Types ───────────────────────────────────────────────────────

// SyncStateConfigsPayload is the payload for the daily state config sync task.
// [14-comply §9.1] Schedule: daily at 4:00 AM UTC.
type SyncStateConfigsPayload struct{}

func (SyncStateConfigsPayload) TaskType() string { return TaskTypeSyncStateConfigs }

// GeneratePortfolioPayload is the payload for on-demand portfolio PDF generation.
// [14-comply §9.2] Trigger: POST .../generate.
type GeneratePortfolioPayload struct {
	PortfolioID uuid.UUID `json:"portfolio_id"`
	FamilyID    uuid.UUID `json:"family_id"`
}

func (GeneratePortfolioPayload) TaskType() string { return TaskTypeGeneratePortfolio }

// GenerateTranscriptPayload is the payload for on-demand transcript PDF generation (Phase 3).
// [14-comply §9.3] Trigger: POST .../generate.
type GenerateTranscriptPayload struct {
	TranscriptID uuid.UUID `json:"transcript_id"`
	FamilyID     uuid.UUID `json:"family_id"`
}

func (GenerateTranscriptPayload) TaskType() string { return TaskTypeGenerateTranscript }

// AttendanceThresholdCheckPayload is the payload for the weekly attendance threshold check.
// [14-comply §9.4] Schedule: weekly on Sundays at 5:00 AM UTC.
type AttendanceThresholdCheckPayload struct{}

func (AttendanceThresholdCheckPayload) TaskType() string { return TaskTypeAttendanceThresholdCheck }

// Compile-time interface checks.
var (
	_ shared.JobPayload = SyncStateConfigsPayload{}
	_ shared.JobPayload = GeneratePortfolioPayload{}
	_ shared.JobPayload = GenerateTranscriptPayload{}
	_ shared.JobPayload = AttendanceThresholdCheckPayload{}
)

// ─── Task Registration ────────────────────────────────────────────────────────

// RegisterTaskHandlers registers comply:: background task handlers with the job worker.
// Accepts *gorm.DB for cross-family batch reads that bypass FamilyScope. [14-comply §9]
func RegisterTaskHandlers(
	worker shared.JobWorker,
	db *gorm.DB,
	stateConfigRepo StateConfigRepository,
	familyConfigRepo FamilyConfigRepository,
	attendanceRepo AttendanceRepository,
	portfolioRepo PortfolioRepository,
	portfolioItemRepo PortfolioItemRepository,
	transcriptRepo TranscriptRepository,
	courseRepo CourseRepository,
	iamSvc IamServiceForComply,
	discoverySvc DiscoveryServiceForComply,
	mediaSvc MediaServiceForComply,
	events *shared.EventBus,
) {
	worker.Handle(TaskTypeSyncStateConfigs, handleSyncStateConfigsTask(stateConfigRepo, discoverySvc))
	worker.Handle(TaskTypeGeneratePortfolio, handleGeneratePortfolioTask(db, portfolioRepo, portfolioItemRepo, attendanceRepo, iamSvc, mediaSvc, events))
	worker.Handle(TaskTypeGenerateTranscript, handleGenerateTranscriptTask(db, transcriptRepo, courseRepo, iamSvc, mediaSvc, events))
	worker.Handle(TaskTypeAttendanceThresholdCheck, handleAttendanceThresholdCheckTask(db, stateConfigRepo, attendanceRepo, events))
}

// ─── Task Handlers ────────────────────────────────────────────────────────────

// handleSyncStateConfigsTask returns a JobHandler that syncs state compliance requirements
// from discover:: into comply_state_configs. [14-comply §9.1]
func handleSyncStateConfigsTask(
	stateConfigRepo StateConfigRepository,
	discoverySvc DiscoveryServiceForComply,
) shared.JobHandler {
	return func(ctx context.Context, payload []byte) error {
		var task SyncStateConfigsPayload
		if err := json.Unmarshal(payload, &task); err != nil {
			return fmt.Errorf("comply: unmarshal sync_state_configs task: %w", err)
		}

		guides, err := discoverySvc.ListStateGuides(ctx)
		if err != nil {
			return fmt.Errorf("comply: sync_state_configs list guides: %w", err)
		}

		var upserted int
		for _, guide := range guides {
			reqs, err := discoverySvc.GetStateRequirements(ctx, guide.StateCode)
			if err != nil {
				slog.Error("comply: sync_state_configs get requirements",
					"state_code", guide.StateCode, "error", err)
				continue
			}
			if reqs == nil {
				continue
			}

			_, err = stateConfigRepo.Upsert(ctx, UpsertStateConfigRow{
				StateCode:               reqs.StateCode,
				StateName:               reqs.StateName,
				NotificationRequired:    reqs.NotificationRequired,
				NotificationDetails:     reqs.NotificationDetails,
				RequiredSubjects:        reqs.RequiredSubjects,
				AssessmentRequired:      reqs.AssessmentRequired,
				AssessmentDetails:       reqs.AssessmentDetails,
				RecordKeepingRequired:   reqs.RecordKeepingRequired,
				RecordKeepingDetails:    reqs.RecordKeepingDetails,
				AttendanceRequired:      reqs.AttendanceRequired,
				AttendanceDays:          reqs.AttendanceDays,
				AttendanceHours:         reqs.AttendanceHours,
				AttendanceDetails:       reqs.AttendanceDetails,
				UmbrellaSchoolAvailable: reqs.UmbrellaSchoolAvailable,
				UmbrellaSchoolDetails:   reqs.UmbrellaSchoolDetails,
				RegulationLevel:         reqs.RegulationLevel,
			})
			if err != nil {
				slog.Error("comply: sync_state_configs upsert",
					"state_code", guide.StateCode, "error", err)
				continue
			}
			upserted++
		}

		slog.Info("comply: sync_state_configs complete",
			"states_listed", len(guides),
			"upserted", upserted,
		)
		return nil
	}
}

// handleGeneratePortfolioTask returns a JobHandler that generates a portfolio PDF.
// [14-comply §9.2] Retry policy: max 3 attempts. On final failure, status → failed.
func handleGeneratePortfolioTask(
	db *gorm.DB,
	portfolioRepo PortfolioRepository,
	portfolioItemRepo PortfolioItemRepository,
	attendanceRepo AttendanceRepository,
	iamSvc IamServiceForComply,
	mediaSvc MediaServiceForComply,
	events *shared.EventBus,
) shared.JobHandler {
	return func(ctx context.Context, payload []byte) error {
		var task GeneratePortfolioPayload
		if err := json.Unmarshal(payload, &task); err != nil {
			return fmt.Errorf("comply: unmarshal generate_portfolio task: %w", err)
		}

		// Load portfolio (no FamilyScope — background job).
		var portfolio ComplyPortfolio
		if err := db.WithContext(ctx).Raw(
			`SELECT * FROM comply_portfolios WHERE id = ?`, task.PortfolioID,
		).Scan(&portfolio).Error; err != nil {
			return fmt.Errorf("comply: generate_portfolio load: %w", err)
		}
		if portfolio.ID == uuid.Nil {
			return fmt.Errorf("comply: generate_portfolio: portfolio %s not found", task.PortfolioID)
		}

		if portfolio.Status != string(PortfolioStatusGenerating) {
			slog.Info("comply: generate_portfolio skipped (not in generating status)",
				"portfolio_id", task.PortfolioID, "status", portfolio.Status)
			return nil
		}

		// Load items.
		items, err := portfolioItemRepo.ListByPortfolio(ctx, task.PortfolioID)
		if err != nil {
			return markPortfolioFailed(portfolioRepo, ctx, task.PortfolioID, portfolio.RetryCount, err)
		}

		// Load attendance summary if requested.
		var attendanceSummary *AttendanceSummaryRow
		if portfolio.IncludeAttendance {
			scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: task.FamilyID})
			attendanceSummary, err = attendanceRepo.Summarize(ctx, portfolio.StudentID, scope, portfolio.DateRangeStart, portfolio.DateRangeEnd)
			if err != nil {
				slog.Error("comply: generate_portfolio attendance summary", "error", err)
				// Non-fatal — continue without attendance data.
			}
		}

		// Load student name for the cover page.
		studentName, err := iamSvc.GetStudentName(ctx, portfolio.StudentID)
		if err != nil {
			slog.Error("comply: generate_portfolio get student name", "error", err)
			studentName = "Student"
		}

		// Render PDF.
		pdfBytes, err := renderPortfolioPDF(&portfolio, items, studentName, attendanceSummary)
		if err != nil {
			return markPortfolioFailed(portfolioRepo, ctx, task.PortfolioID, portfolio.RetryCount, err)
		}

		// Upload PDF to media::.
		filename := fmt.Sprintf("portfolio_%s.pdf", task.PortfolioID)
		uploadID, err := mediaSvc.RequestUpload(ctx, task.FamilyID, "comply_portfolio", filename, "application/pdf", pdfBytes)
		if err != nil {
			return markPortfolioFailed(portfolioRepo, ctx, task.PortfolioID, portfolio.RetryCount, err)
		}

		// Transition status: generating → ready.
		expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
		_, err = portfolioRepo.UpdateStatus(ctx, task.PortfolioID, string(PortfolioStatusReady), uploadID, nil)
		if err != nil {
			return fmt.Errorf("comply: generate_portfolio update status: %w", err)
		}

		// Set expires_at and generated_at.
		now := time.Now().UTC()
		if err := db.WithContext(ctx).Exec(
			`UPDATE comply_portfolios SET generated_at = ?, expires_at = ?, updated_at = ? WHERE id = ?`,
			now, expiresAt, now, task.PortfolioID,
		).Error; err != nil {
			slog.Error("comply: generate_portfolio set timestamps", "error", err)
		}

		// Publish event.
		_ = events.Publish(ctx, PortfolioGenerated{
			FamilyID:       task.FamilyID,
			StudentID:      portfolio.StudentID,
			PortfolioID:    task.PortfolioID,
			PortfolioTitle: portfolio.Title,
		})

		slog.Info("comply: generate_portfolio complete",
			"portfolio_id", task.PortfolioID,
			"items", len(items),
		)
		return nil
	}
}

// markPortfolioFailed transitions a portfolio to failed status with error message.
func markPortfolioFailed(repo PortfolioRepository, ctx context.Context, portfolioID uuid.UUID, retryCount int16, cause error) error {
	if int32(retryCount) >= maxPortfolioRetries {
		errMsg := cause.Error()
		if _, err := repo.UpdateStatus(ctx, portfolioID, string(PortfolioStatusFailed), nil, &errMsg); err != nil {
			slog.Error("comply: mark portfolio failed", "portfolio_id", portfolioID, "error", err)
		}
		return nil // Don't retry — max attempts reached.
	}
	return fmt.Errorf("comply: generate_portfolio: %w", cause) // Returning error triggers asynq retry.
}

// handleGenerateTranscriptTask returns a JobHandler that generates a transcript PDF (Phase 3).
// [14-comply §9.3]
func handleGenerateTranscriptTask(
	db *gorm.DB,
	transcriptRepo TranscriptRepository,
	courseRepo CourseRepository,
	iamSvc IamServiceForComply,
	mediaSvc MediaServiceForComply,
	events *shared.EventBus,
) shared.JobHandler {
	return func(ctx context.Context, payload []byte) error {
		var task GenerateTranscriptPayload
		if err := json.Unmarshal(payload, &task); err != nil {
			return fmt.Errorf("comply: unmarshal generate_transcript task: %w", err)
		}

		// Load transcript (no FamilyScope — background job).
		var transcript ComplyTranscript
		if err := db.WithContext(ctx).Raw(
			`SELECT * FROM comply_transcripts WHERE id = ?`, task.TranscriptID,
		).Scan(&transcript).Error; err != nil {
			return fmt.Errorf("comply: generate_transcript load: %w", err)
		}
		if transcript.ID == uuid.Nil {
			return fmt.Errorf("comply: generate_transcript: transcript %s not found", task.TranscriptID)
		}

		if transcript.Status != string(PortfolioStatusGenerating) {
			slog.Info("comply: generate_transcript skipped (not in generating status)",
				"transcript_id", task.TranscriptID, "status", transcript.Status)
			return nil
		}

		// Load courses for this student.
		scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: task.FamilyID})
		courses, err := courseRepo.ListByStudent(ctx, transcript.StudentID, scope, nil)
		if err != nil {
			return fmt.Errorf("comply: generate_transcript load courses: %w", err)
		}

		// Calculate GPA snapshot at generation time.
		var gpaCourses []domain.CourseForGpa
		for _, c := range courses {
			gpaCourses = append(gpaCourses, domain.CourseForGpa{
				Credits:     c.Credits,
				GradePoints: c.GradePoints,
				Level:       c.Level,
			})
		}
		gpaResult := domain.CalculateGPA(gpaCourses, domain.GpaScaleWeighted, nil)
		unweightedGPA := gpaResult.Unweighted
		weightedGPA := gpaResult.Weighted

		// Load student name.
		studentName, err := iamSvc.GetStudentName(ctx, transcript.StudentID)
		if err != nil {
			slog.Error("comply: generate_transcript get student name", "error", err)
			studentName = transcript.StudentName
		}

		// Render PDF.
		pdfBytes, err := renderTranscriptPDF(&transcript, courses, studentName, unweightedGPA, weightedGPA)
		if err != nil {
			return fmt.Errorf("comply: generate_transcript render PDF: %w", err)
		}

		// Upload PDF to media::.
		filename := fmt.Sprintf("transcript_%s.pdf", task.TranscriptID)
		uploadID, err := mediaSvc.RequestUpload(ctx, task.FamilyID, "comply_transcript", filename, "application/pdf", pdfBytes)
		if err != nil {
			return fmt.Errorf("comply: generate_transcript upload: %w", err)
		}

		// Transition status: generating → ready, snapshot GPA values.
		_, err = transcriptRepo.UpdateStatus(ctx, task.TranscriptID, string(PortfolioStatusReady), uploadID, &unweightedGPA, &weightedGPA, nil)
		if err != nil {
			return fmt.Errorf("comply: generate_transcript update status: %w", err)
		}

		// Set expires_at and generated_at.
		now := time.Now().UTC()
		expiresAt := now.Add(90 * 24 * time.Hour)
		if err := db.WithContext(ctx).Exec(
			`UPDATE comply_transcripts SET generated_at = ?, expires_at = ?, updated_at = ? WHERE id = ?`,
			now, expiresAt, now, task.TranscriptID,
		).Error; err != nil {
			slog.Error("comply: generate_transcript set timestamps", "error", err)
		}

		// Publish event.
		_ = events.Publish(ctx, TranscriptGenerated{
			FamilyID:     task.FamilyID,
			StudentID:    transcript.StudentID,
			TranscriptID: task.TranscriptID,
		})

		slog.Info("comply: generate_transcript complete",
			"transcript_id", task.TranscriptID,
			"courses", len(courses),
			"gpa_unweighted", unweightedGPA,
			"gpa_weighted", weightedGPA,
		)
		return nil
	}
}

// handleAttendanceThresholdCheckTask returns a JobHandler that checks attendance pace
// for all configured families and publishes warnings for at-risk/behind students.
// [14-comply §9.4]
func handleAttendanceThresholdCheckTask(
	db *gorm.DB,
	stateConfigRepo StateConfigRepository,
	attendanceRepo AttendanceRepository,
	events *shared.EventBus,
) shared.JobHandler {
	return func(ctx context.Context, payload []byte) error {
		var task AttendanceThresholdCheckPayload
		if err := json.Unmarshal(payload, &task); err != nil {
			return fmt.Errorf("comply: unmarshal attendance_threshold_check task: %w", err)
		}

		// Load all family configs (cross-family batch read — no FamilyScope). [14-comply §9.4]
		type configRow struct {
			FamilyID        uuid.UUID `gorm:"column:family_id"`
			StateCode       string    `gorm:"column:state_code"`
			SchoolYearStart time.Time `gorm:"column:school_year_start"`
			SchoolYearEnd   time.Time `gorm:"column:school_year_end"`
			TotalSchoolDays int16     `gorm:"column:total_school_days"`
		}
		var configs []configRow
		if err := db.WithContext(ctx).Raw(
			`SELECT family_id, state_code, school_year_start, school_year_end, total_school_days
			 FROM comply_family_configs`,
		).Scan(&configs).Error; err != nil {
			return fmt.Errorf("comply: attendance_threshold_check load configs: %w", err)
		}

		if len(configs) == 0 {
			slog.Info("comply: attendance_threshold_check complete", "families", 0)
			return nil
		}

		// Pre-load all state configs for lookup.
		allStates, err := stateConfigRepo.ListAll(ctx)
		if err != nil {
			return fmt.Errorf("comply: attendance_threshold_check load states: %w", err)
		}
		stateMap := make(map[string]*ComplyStateConfig, len(allStates))
		for i := range allStates {
			stateMap[allStates[i].StateCode] = &allStates[i]
		}

		var warnings int
		for _, cfg := range configs {
			state, ok := stateMap[cfg.StateCode]
			if !ok || !state.AttendanceRequired || state.AttendanceDays == nil {
				continue
			}

			// Query students for this family.
			type studentRow struct {
				ID   uuid.UUID `gorm:"column:id"`
				Name string    `gorm:"column:first_name"`
			}
			var students []studentRow
			if err := db.WithContext(ctx).Raw(
				`SELECT id, first_name FROM iam_students WHERE family_id = ?`, cfg.FamilyID,
			).Scan(&students).Error; err != nil {
				slog.Error("comply: attendance_threshold_check list students",
					"family_id", cfg.FamilyID, "error", err)
				continue
			}

			for _, student := range students {
				scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: cfg.FamilyID})

				summary, err := attendanceRepo.Summarize(ctx, student.ID, scope, cfg.SchoolYearStart, cfg.SchoolYearEnd)
				if err != nil {
					slog.Error("comply: attendance_threshold_check summarize",
						"student_id", student.ID, "error", err)
					continue
				}

				presentDays := summary.PresentFull + summary.PresentPartial
				totalDays := summary.PresentFull + summary.PresentPartial + summary.Absent + summary.NotApplicable

				pace := domain.CalculatePace(presentDays, totalDays, int32(cfg.TotalSchoolDays), state.AttendanceDays)

				if pace == domain.PaceStatusAtRisk || pace == domain.PaceStatusBehind {
					// Calculate expected days at this point.
					expectedDays := int32(0)
					if cfg.TotalSchoolDays > 0 && totalDays > 0 {
						expectedDays = int32(float64(*state.AttendanceDays) * float64(totalDays) / float64(cfg.TotalSchoolDays))
					}

					_ = events.Publish(ctx, AttendanceThresholdWarning{
						FamilyID:     cfg.FamilyID,
						StudentID:    student.ID,
						StudentName:  student.Name,
						PaceStatus:   string(pace),
						ActualDays:   presentDays,
						ExpectedDays: expectedDays,
						RequiredDays: *state.AttendanceDays,
					})
					warnings++
				}
			}
		}

		slog.Info("comply: attendance_threshold_check complete",
			"families", len(configs),
			"warnings", warnings,
		)
		return nil
	}
}
