package safety

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ─── CsamReportRunner ──────────────────────────────────────────────────────────

func TestCsamReportRunner_SuccessfulSubmission(t *testing.T) {
	// [11-safety §10.3] — successful NCMEC submission → status=submitted.
	ncmecRepo := newMockNcmecRepo()
	thorn := newMockThornAdapter()

	reportID := uuid.Must(uuid.NewV7())
	uploadID := uuid.Must(uuid.NewV7())

	ncmecRepo.findByIDFn = func(_ context.Context, id uuid.UUID) (*NcmecReport, error) {
		return &NcmecReport{
			ID:                 id,
			UploadID:           uploadID,
			FamilyID:           uuid.Must(uuid.NewV7()),
			ParentID:           uuid.Must(uuid.NewV7()),
			EvidenceStorageKey: "evidence/csam/test",
			Status:             "pending",
			CreatedAt:          time.Now(),
		}, nil
	}

	ncmecReportIDStr := "NCMEC-12345"
	thorn.submitNcmecReportFn = func(_ context.Context, report NcmecReportPayload) (*NcmecSubmissionResult, error) {
		if report.UploadID != uploadID {
			t.Errorf("uploadID = %v, want %v", report.UploadID, uploadID)
		}
		return &NcmecSubmissionResult{
			NcmecReportID: ncmecReportIDStr,
			SubmittedAt:   time.Now(),
		}, nil
	}

	var updatedStatus string
	var updatedNCMECID *string
	ncmecRepo.updateStatusFn = func(_ context.Context, _ uuid.UUID, status string, ncmecID *string, _ *string) (*NcmecReport, error) {
		updatedStatus = status
		updatedNCMECID = ncmecID
		return &NcmecReport{}, nil
	}

	runner := &csamReportRunner{ncmecRepo: ncmecRepo, thorn: thorn}
	payload, _ := json.Marshal(CsamReportPayload{NcmecReportID: reportID})
	err := runner.Run(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedStatus != "submitted" {
		t.Errorf("status = %q, want %q", updatedStatus, "submitted")
	}
	if updatedNCMECID == nil || *updatedNCMECID != ncmecReportIDStr {
		t.Errorf("ncmec_report_id = %v, want %q", updatedNCMECID, ncmecReportIDStr)
	}
}

func TestCsamReportRunner_ThornAPIFailure(t *testing.T) {
	// [11-safety §10.3] — Thorn API failure → status=failed, retryable error returned.
	ncmecRepo := newMockNcmecRepo()
	thorn := newMockThornAdapter()

	ncmecRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*NcmecReport, error) {
		return &NcmecReport{
			ID:                 uuid.Must(uuid.NewV7()),
			UploadID:           uuid.Must(uuid.NewV7()),
			FamilyID:           uuid.Must(uuid.NewV7()),
			ParentID:           uuid.Must(uuid.NewV7()),
			EvidenceStorageKey: "evidence/csam/test",
			Status:             "pending",
			CreatedAt:          time.Now(),
		}, nil
	}

	thorn.submitNcmecReportFn = func(_ context.Context, _ NcmecReportPayload) (*NcmecSubmissionResult, error) {
		return nil, errors.New("thorn API timeout")
	}

	var updatedStatus string
	var savedErrorMsg *string
	ncmecRepo.updateStatusFn = func(_ context.Context, _ uuid.UUID, status string, _ *string, errMsg *string) (*NcmecReport, error) {
		updatedStatus = status
		savedErrorMsg = errMsg
		return &NcmecReport{}, nil
	}

	runner := &csamReportRunner{ncmecRepo: ncmecRepo, thorn: thorn}
	payload, _ := json.Marshal(CsamReportPayload{NcmecReportID: uuid.Must(uuid.NewV7())})
	err := runner.Run(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error")
	}
	if updatedStatus != "failed" {
		t.Errorf("status = %q, want %q", updatedStatus, "failed")
	}
	if savedErrorMsg == nil || *savedErrorMsg == "" {
		t.Error("expected non-empty error message")
	}
}

// ─── CsamHashCheckRunner ───────────────────────────────────────────────────────

func TestCsamHashCheckRunner_NoUpdate(t *testing.T) {
	// [11-safety §10.7] — no update available → no action.
	thorn := newMockThornAdapter()
	thorn.checkHashUpdateFn = func(_ context.Context) (bool, error) {
		return false, nil
	}

	runner := &csamHashCheckRunner{thorn: thorn}
	err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCsamHashCheckRunner_UpdateAvailable(t *testing.T) {
	// [11-safety §10.7] — update available → returns without error.
	thorn := newMockThornAdapter()
	thorn.checkHashUpdateFn = func(_ context.Context) (bool, error) {
		return true, nil
	}

	runner := &csamHashCheckRunner{thorn: thorn}
	err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
