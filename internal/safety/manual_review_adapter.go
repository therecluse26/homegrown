package safety

import (
	"context"
	"log/slog"
)

// ManualReviewThornAdapter replaces LoggingThornAdapter with persistent storage.
// It queues flagged content for manual review in the safety_manual_review_queue table
// and stores pending NCMEC reports in safety_ncmec_pending_reports for manual filing.
//
// This adapter is the interim compliance solution until real Thorn/PhotoDNA
// vendor credentials are obtained. It ensures:
//   - No CSAM scan result is silently discarded (persisted to DB)
//   - NCMEC reports are queued for manual filing (18 U.S.C. § 2258A)
//   - Admin dashboard can query pending review items
//
// [11-safety §7.1, CRIT-1]
type ManualReviewThornAdapter struct {
	reviewRepo ManualReviewRepository
	ncmecRepo  NcmecPendingReportRepository
}

// Compile-time interface check.
var _ ThornAdapter = (*ManualReviewThornAdapter)(nil)

// NewManualReviewThornAdapter creates a new ManualReviewThornAdapter.
func NewManualReviewThornAdapter(
	reviewRepo ManualReviewRepository,
	ncmecRepo NcmecPendingReportRepository,
) *ManualReviewThornAdapter {
	return &ManualReviewThornAdapter{
		reviewRepo: reviewRepo,
		ncmecRepo:  ncmecRepo,
	}
}

// ScanCsam queues the content for manual CSAM review and returns RequiresManualReview=true.
// The safety service will create a moderation task based on this flag. [11-safety §7.1]
func (a *ManualReviewThornAdapter) ScanCsam(ctx context.Context, key string) (*CsamScanResult, error) {
	if err := a.reviewRepo.Create(ctx, &ManualReviewItem{
		StorageKey: key,
		ReviewType: "csam_scan",
		Status:     "pending",
	}); err != nil {
		slog.Error("manual review adapter: failed to queue CSAM scan for review",
			"content_key", key, "error", err)
		// Return the result even if persistence fails — don't block the upload pipeline.
	}

	slog.Warn("thorn not integrated — CSAM scan queued for manual review",
		"content_key", key,
		"action", "manual_review_queued",
	)
	return &CsamScanResult{
		IsCSAM:              false,
		RequiresManualReview: true,
	}, nil
}

// SubmitNcmecReport persists the report in safety_ncmec_pending_reports for manual filing.
// 18 U.S.C. § 2258A requires electronic service providers to report CSAM to NCMEC.
// This adapter ensures the report is never silently discarded. [11-safety §7.1]
func (a *ManualReviewThornAdapter) SubmitNcmecReport(ctx context.Context, report NcmecReportPayload) (*NcmecSubmissionResult, error) {
	if err := a.ncmecRepo.Create(ctx, &NcmecPendingReport{
		UploadID:         report.UploadID,
		UploaderFamilyID: report.UploaderFamilyID,
		UploaderParentID: report.UploaderParentID,
		EvidenceKey:      report.EvidenceStorageKey,
		CsamHash:         report.CsamHash,
		Confidence:       report.Confidence,
		MatchedDatabase:  report.MatchedDatabase,
		UploadTimestamp:  report.UploadTimestamp,
		Status:           "queued",
	}); err != nil {
		slog.Error("manual review adapter: failed to persist NCMEC pending report",
			"upload_id", report.UploadID, "error", err)
		return nil, err
	}

	slog.Error("thorn not integrated — NCMEC report queued for manual filing",
		"upload_id", report.UploadID,
		"uploader_family_id", report.UploaderFamilyID,
		"action", "ncmec_report_queued",
	)
	return &NcmecSubmissionResult{
		NcmecReportID: "pending-manual-filing",
	}, nil
}

// CheckHashUpdate reports no update available since the real Thorn hash database is not connected.
func (a *ManualReviewThornAdapter) CheckHashUpdate(_ context.Context) (bool, error) {
	slog.Warn("thorn not integrated — CSAM hash database cannot be refreshed")
	return false, nil
}
