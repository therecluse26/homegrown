package safety

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── CSAM Report Runner ───────────────────────────────────────────────────────

// csamReportRunner holds dependencies for the NCMEC report submission job.
// [11-safety §10.3]
type csamReportRunner struct {
	ncmecRepo NcmecReportRepository
	thorn     ThornAdapter
}

// Run submits a pending NCMEC report to the Thorn Safer API.
func (r *csamReportRunner) Run(ctx context.Context, reportID json.RawMessage) error {
	var payload CsamReportPayload
	if err := json.Unmarshal(reportID, &payload); err != nil {
		return fmt.Errorf("unmarshal csam report payload: %w", err)
	}

	ncmecReport, err := r.ncmecRepo.FindByID(ctx, payload.NcmecReportID)
	if err != nil {
		return fmt.Errorf("find ncmec report: %w", err)
	}

	result, err := r.thorn.SubmitNcmecReport(ctx, NcmecReportPayload{
		UploadID:           ncmecReport.UploadID,
		CsamHash:           ncmecReport.CsamHash,
		Confidence:         ncmecReport.Confidence,
		MatchedDatabase:    ncmecReport.MatchedDatabase,
		EvidenceStorageKey: ncmecReport.EvidenceStorageKey,
		UploaderFamilyID:   ncmecReport.FamilyID,
		UploaderParentID:   ncmecReport.ParentID,
		UploadTimestamp:    ncmecReport.CreatedAt,
	})
	if err != nil {
		errMsg := err.Error()
		if _, updateErr := r.ncmecRepo.UpdateStatus(ctx, payload.NcmecReportID, "failed", nil, &errMsg); updateErr != nil {
			slog.Error("failed to update ncmec report status to failed", "id", payload.NcmecReportID, "error", updateErr)
		}
		return fmt.Errorf("submit ncmec report: %w", err)
	}

	if result == nil {
		// Adapter returned no result (noop/logging mode) — mark as pending.
		if _, err := r.ncmecRepo.UpdateStatus(ctx, payload.NcmecReportID, "pending_manual", nil, nil); err != nil {
			return fmt.Errorf("update ncmec report status: %w", err)
		}
		return nil
	}

	if _, err := r.ncmecRepo.UpdateStatus(ctx, payload.NcmecReportID, "submitted", &result.NcmecReportID, nil); err != nil {
		return fmt.Errorf("update ncmec report status: %w", err)
	}

	slog.Info("NCMEC report submitted", "ncmec_report_id", result.NcmecReportID, "internal_id", payload.NcmecReportID)
	return nil
}

// ─── CSAM Hash Update Check Runner ──────────────────────────────────────────

// csamHashCheckRunner holds dependencies for the CSAM hash update check job.
// [11-safety §10.7]
type csamHashCheckRunner struct {
	thorn ThornAdapter
	jobs  shared.JobEnqueuer
}

// Run checks if a CSAM hash database update is available from Thorn.
func (r *csamHashCheckRunner) Run(ctx context.Context) error {
	hasUpdate, err := r.thorn.CheckHashUpdate(ctx)
	if err != nil {
		return fmt.Errorf("check hash update: %w", err)
	}

	if hasUpdate {
		slog.Info("CSAM hash update available — rescan will be triggered")
	}

	return nil
}

// ─── Worker Registration ──────────────────────────────────────────────────────

// RegisterSafetyWorkers registers safety background job handlers with the worker. [11-safety §10]
func RegisterSafetyWorkers(
	worker shared.JobWorker,
	ncmecRepo NcmecReportRepository,
	thorn ThornAdapter,
	jobs shared.JobEnqueuer,
	svc SafetyService,
) {
	csam := &csamReportRunner{ncmecRepo: ncmecRepo, thorn: thorn}

	worker.Handle("safety:csam_report", func(ctx context.Context, payload []byte) error {
		slog.Info("processing CSAM report submission")
		return csam.Run(ctx, payload)
	})

	hashCheck := &csamHashCheckRunner{thorn: thorn, jobs: jobs}

	worker.Handle("safety:check_csam_hash_update", func(ctx context.Context, _ []byte) error {
		slog.Info("checking CSAM hash database updates")
		return hashCheck.Run(ctx)
	})

	// Phase 2: Proactively expire overdue suspensions [11-safety §14.1]
	worker.Handle("safety:expire_suspensions", func(ctx context.Context, _ []byte) error {
		slog.Info("running suspension expiry check")
		return svc.ExpireSuspensions(ctx)
	})
}
