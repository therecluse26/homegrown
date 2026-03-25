package safety

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// ─── NoopThornAdapter ────────────────────────────────────────────────────────

// NoopThornAdapter returns placeholder responses for Thorn Safer API operations.
// Used in development and test environments until Thorn integration is configured. [11-safety §7.1]
type NoopThornAdapter struct{}

// Compile-time interface check.
var _ ThornAdapter = NoopThornAdapter{}

func (NoopThornAdapter) ScanCsam(_ context.Context, key string) (*CsamScanResult, error) {
	slog.Warn("thorn not configured — ScanCsam is a no-op", "key", key)
	return &CsamScanResult{IsCSAM: false}, nil
}

func (NoopThornAdapter) SubmitNcmecReport(_ context.Context, report NcmecReportPayload) (*NcmecSubmissionResult, error) {
	slog.Warn("thorn not configured — SubmitNcmecReport is a no-op", "upload_id", report.UploadID)
	return nil, nil
}

func (NoopThornAdapter) CheckHashUpdate(_ context.Context) (bool, error) {
	slog.Warn("thorn not configured — CheckHashUpdate is a no-op")
	return false, nil
}

// ─── NoopRekognitionAdapter ──────────────────────────────────────────────────

// NoopRekognitionAdapter returns clean moderation results for all operations.
// Used in development and test environments until AWS Rekognition is configured. [11-safety §7.2]
type NoopRekognitionAdapter struct{}

// Compile-time interface check.
var _ RekognitionAdapter = NoopRekognitionAdapter{}

func (NoopRekognitionAdapter) DetectModerationLabels(_ context.Context, key string) (*ModerationResult, error) {
	slog.Warn("rekognition not configured — DetectModerationLabels is a no-op", "key", key)
	return &ModerationResult{HasViolations: false, AutoReject: false}, nil
}

// ─── NoopIamServiceForSafety ─────────────────────────────────────────────────

// NoopIamServiceForSafety returns success for all IAM operations.
// Used in development and test environments. [11-safety §7.3]
type NoopIamServiceForSafety struct{}

// Compile-time interface check.
var _ IamServiceForSafety = NoopIamServiceForSafety{}

func (NoopIamServiceForSafety) RevokeSessions(_ context.Context, familyID uuid.UUID) error {
	slog.Warn("iam not configured for safety — RevokeSessions is a no-op", "family_id", familyID)
	return nil
}
