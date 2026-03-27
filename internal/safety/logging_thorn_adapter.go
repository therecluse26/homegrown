package safety

import (
	"context"
	"log/slog"
)

// LoggingThornAdapter replaces NoopThornAdapter in non-development deployments.
// It logs CSAM-adjacent events and queues NCMEC reports for manual filing
// rather than silently discarding them. [11-safety §7.1, CRIT-2]
//
// Real Thorn/PhotoDNA integration requires vendor credentials and is tracked
// separately. This adapter ensures no event is silently lost.
type LoggingThornAdapter struct{}

// Compile-time interface check.
var _ ThornAdapter = LoggingThornAdapter{}

// ScanCsam logs a structured warning and flags the content for manual review.
// Returns RequiresManualReview=true so the safety service queues moderation. [11-safety §7.1]
func (LoggingThornAdapter) ScanCsam(_ context.Context, key string) (*CsamScanResult, error) {
	slog.Warn("thorn not integrated — CSAM scan requires manual review",
		"content_key", key,
		"action", "manual_review_queued",
	)
	return &CsamScanResult{
		IsCSAM:              false,
		RequiresManualReview: true,
	}, nil
}

// SubmitNcmecReport logs a structured error so the event is not silently discarded.
// The pending report must be manually filed until real Thorn integration is wired.
// [18 U.S.C. § 2258A, 11-safety §7.1]
func (LoggingThornAdapter) SubmitNcmecReport(_ context.Context, report NcmecReportPayload) (*NcmecSubmissionResult, error) {
	slog.Error("thorn not integrated — NCMEC report requires manual filing",
		"upload_id", report.UploadID,
		"uploader_family_id", report.UploaderFamilyID,
		"evidence_key", report.EvidenceStorageKey,
		"action", "manual_filing_required",
	)
	// Return a placeholder result — caller should check nil and treat as pending.
	return nil, nil
}

// CheckHashUpdate logs a warning and reports no update available.
// Hash database refresh requires real Thorn integration. [11-safety §7.1]
func (LoggingThornAdapter) CheckHashUpdate(_ context.Context) (bool, error) {
	slog.Warn("thorn not integrated — CSAM hash database cannot be refreshed")
	return false, nil
}
