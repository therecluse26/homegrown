package lifecycle

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// A. Data Export — RequestExport (Reqs 1–8) [15-data-lifecycle §9]
// ═══════════════════════════════════════════════════════════════════════════════

// Req 1: RequestExport creates export record with status pending and returns UUID.
func TestRequestExport_CreatesRecordAndReturnsUUID(t *testing.T) {
	auth := testAuth()
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())

	var capturedInput *CreateExportRequest
	svc := newTestService(
		&stubExportRepo{createFn: func(_ context.Context, _ *shared.FamilyScope, input *CreateExportRequest) (*ExportRequest, error) {
			capturedInput = input
			return &ExportRequest{ID: exportID, Status: ExportStatusPending}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	id, err := svc.RequestExport(context.Background(), auth, scope, &RequestExportInput{})
	if err != nil {
		t.Fatalf("RequestExport error: %v", err)
	}
	if id != exportID {
		t.Errorf("returned ID = %v, want %v", id, exportID)
	}
	if capturedInput == nil {
		t.Fatal("repo.Create was not called")
	}
	if capturedInput.RequestedBy != auth.ParentID {
		t.Errorf("RequestedBy = %v, want %v", capturedInput.RequestedBy, auth.ParentID)
	}
}

// Req 2: RequestExport defaults format to JSON when Format is nil.
func TestRequestExport_DefaultsFormatToJSON(t *testing.T) {
	auth := testAuth()
	scope := testScope()

	var capturedInput *CreateExportRequest
	svc := newTestService(
		&stubExportRepo{createFn: func(_ context.Context, _ *shared.FamilyScope, input *CreateExportRequest) (*ExportRequest, error) {
			capturedInput = input
			return &ExportRequest{ID: uuid.Must(uuid.NewV7()), Status: ExportStatusPending}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestExport(context.Background(), auth, scope, &RequestExportInput{Format: nil})
	if err != nil {
		t.Fatalf("RequestExport error: %v", err)
	}
	if capturedInput.Format != ExportFormatJSON {
		t.Errorf("Format = %q, want %q", capturedInput.Format, ExportFormatJSON)
	}
}

// Req 3: RequestExport uses explicit CSV format when provided.
func TestRequestExport_UsesExplicitCSVFormat(t *testing.T) {
	auth := testAuth()
	scope := testScope()
	csv := ExportFormatCSV

	var capturedInput *CreateExportRequest
	svc := newTestService(
		&stubExportRepo{createFn: func(_ context.Context, _ *shared.FamilyScope, input *CreateExportRequest) (*ExportRequest, error) {
			capturedInput = input
			return &ExportRequest{ID: uuid.Must(uuid.NewV7()), Status: ExportStatusPending}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestExport(context.Background(), auth, scope, &RequestExportInput{Format: &csv})
	if err != nil {
		t.Fatalf("RequestExport error: %v", err)
	}
	if capturedInput.Format != ExportFormatCSV {
		t.Errorf("Format = %q, want %q", capturedInput.Format, ExportFormatCSV)
	}
}

// Req 4: RequestExport passes nil IncludeDomains as-is (all domains).
func TestRequestExport_NilIncludeDomainsPassedAsIs(t *testing.T) {
	auth := testAuth()
	scope := testScope()

	var capturedInput *CreateExportRequest
	svc := newTestService(
		&stubExportRepo{createFn: func(_ context.Context, _ *shared.FamilyScope, input *CreateExportRequest) (*ExportRequest, error) {
			capturedInput = input
			return &ExportRequest{ID: uuid.Must(uuid.NewV7()), Status: ExportStatusPending}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestExport(context.Background(), auth, scope, &RequestExportInput{IncludeDomains: nil})
	if err != nil {
		t.Fatalf("RequestExport error: %v", err)
	}
	if capturedInput.IncludeDomains != nil {
		t.Errorf("IncludeDomains = %v, want nil", capturedInput.IncludeDomains)
	}
}

// Req 5: RequestExport passes specific IncludeDomains list to repo.
func TestRequestExport_PassesIncludeDomains(t *testing.T) {
	auth := testAuth()
	scope := testScope()
	domains := []string{"learning", "social"}

	var capturedInput *CreateExportRequest
	svc := newTestService(
		&stubExportRepo{createFn: func(_ context.Context, _ *shared.FamilyScope, input *CreateExportRequest) (*ExportRequest, error) {
			capturedInput = input
			return &ExportRequest{ID: uuid.Must(uuid.NewV7()), Status: ExportStatusPending}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestExport(context.Background(), auth, scope, &RequestExportInput{IncludeDomains: domains})
	if err != nil {
		t.Fatalf("RequestExport error: %v", err)
	}
	if len(capturedInput.IncludeDomains) != 2 {
		t.Fatalf("IncludeDomains len = %d, want 2", len(capturedInput.IncludeDomains))
	}
	if capturedInput.IncludeDomains[0] != "learning" || capturedInput.IncludeDomains[1] != "social" {
		t.Errorf("IncludeDomains = %v, want [learning social]", capturedInput.IncludeDomains)
	}
}

// Req 6: RequestExport enqueues DataExportJob via JobEnqueuer.
func TestRequestExport_EnqueuesJob(t *testing.T) {
	auth := testAuth()
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())

	var capturedPayload shared.JobPayload
	svc := newTestService(
		&stubExportRepo{createFn: func(_ context.Context, _ *shared.FamilyScope, _ *CreateExportRequest) (*ExportRequest, error) {
			return &ExportRequest{ID: exportID, FamilyID: scope.FamilyID(), Status: ExportStatusPending}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{enqueueFn: func(_ context.Context, payload shared.JobPayload) error {
			capturedPayload = payload
			return nil
		}},
		nil, nil,
	)

	_, err := svc.RequestExport(context.Background(), auth, scope, &RequestExportInput{})
	if err != nil {
		t.Fatalf("RequestExport error: %v", err)
	}
	if capturedPayload == nil {
		t.Fatal("JobEnqueuer.Enqueue was not called")
	}
	job, ok := capturedPayload.(DataExportJob)
	if !ok {
		t.Fatalf("payload type = %T, want DataExportJob", capturedPayload)
	}
	if job.ExportID != exportID {
		t.Errorf("job.ExportID = %v, want %v", job.ExportID, exportID)
	}
}

// Req 7: RequestExport returns error when repository Create fails.
func TestRequestExport_ReturnsErrorOnRepoFailure(t *testing.T) {
	auth := testAuth()
	scope := testScope()
	repoErr := errors.New("db down")

	svc := newTestService(
		&stubExportRepo{createFn: func(_ context.Context, _ *shared.FamilyScope, _ *CreateExportRequest) (*ExportRequest, error) {
			return nil, repoErr
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestExport(context.Background(), auth, scope, &RequestExportInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Req 8: RequestExport returns error when JobEnqueuer fails.
func TestRequestExport_ReturnsErrorOnJobEnqueueFailure(t *testing.T) {
	auth := testAuth()
	scope := testScope()
	jobErr := errors.New("redis down")

	svc := newTestService(
		&stubExportRepo{createFn: func(_ context.Context, _ *shared.FamilyScope, _ *CreateExportRequest) (*ExportRequest, error) {
			return &ExportRequest{ID: uuid.Must(uuid.NewV7()), Status: ExportStatusPending}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{enqueueFn: func(_ context.Context, _ shared.JobPayload) error {
			return jobErr
		}},
		nil, nil,
	)

	_, err := svc.RequestExport(context.Background(), auth, scope, &RequestExportInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// B. Data Export — GetExportStatus (Reqs 9–12) [15-data-lifecycle §9]
// ═══════════════════════════════════════════════════════════════════════════════

// Req 9: GetExportStatus returns status for a pending export (no download URL).
func TestGetExportStatus_PendingExport(t *testing.T) {
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())
	now := time.Now().UTC()

	svc := newTestService(
		&stubExportRepo{findByIDFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) (*ExportRequest, error) {
			return &ExportRequest{
				ID:        id,
				Status:    ExportStatusPending,
				Format:    ExportFormatJSON,
				CreatedAt: now,
				ExpiresAt: now.Add(7 * 24 * time.Hour),
			}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	resp, err := svc.GetExportStatus(context.Background(), scope, exportID)
	if err != nil {
		t.Fatalf("GetExportStatus error: %v", err)
	}
	if resp.Status != ExportStatusPending {
		t.Errorf("Status = %q, want %q", resp.Status, ExportStatusPending)
	}
	if resp.DownloadURL != nil {
		t.Errorf("DownloadURL = %v, want nil for pending export", *resp.DownloadURL)
	}
}

// Req 10: GetExportStatus returns download URL for completed export.
func TestGetExportStatus_CompletedExportWithURL(t *testing.T) {
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())
	now := time.Now().UTC()
	url := "https://r2.example.com/export.zip"
	var size int64 = 1024

	svc := newTestService(
		&stubExportRepo{findByIDFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) (*ExportRequest, error) {
			return &ExportRequest{
				ID:          id,
				Status:      ExportStatusCompleted,
				Format:      ExportFormatJSON,
				DownloadURL: &url,
				SizeBytes:   &size,
				CreatedAt:   now,
				CompletedAt: &now,
				ExpiresAt:   now.Add(7 * 24 * time.Hour),
			}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	resp, err := svc.GetExportStatus(context.Background(), scope, exportID)
	if err != nil {
		t.Fatalf("GetExportStatus error: %v", err)
	}
	if resp.Status != ExportStatusCompleted {
		t.Errorf("Status = %q, want %q", resp.Status, ExportStatusCompleted)
	}
	if resp.DownloadURL == nil || *resp.DownloadURL != url {
		t.Errorf("DownloadURL = %v, want %q", resp.DownloadURL, url)
	}
	if resp.SizeBytes == nil || *resp.SizeBytes != size {
		t.Errorf("SizeBytes = %v, want %d", resp.SizeBytes, size)
	}
}

// Req 11: GetExportStatus returns ErrExportNotFound when not found.
func TestGetExportStatus_NotFound(t *testing.T) {
	scope := testScope()

	svc := newTestService(
		&stubExportRepo{findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ExportRequest, error) {
			return nil, ErrExportNotFound
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.GetExportStatus(context.Background(), scope, uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrExportNotFound) {
		t.Fatalf("want ErrExportNotFound, got %v", err)
	}
}

// Req 12: GetExportStatus returns ErrExportExpired for expired export.
func TestGetExportStatus_Expired(t *testing.T) {
	scope := testScope()
	past := time.Now().UTC().Add(-1 * time.Hour)

	svc := newTestService(
		&stubExportRepo{findByIDFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) (*ExportRequest, error) {
			return &ExportRequest{
				ID:        id,
				Status:    ExportStatusCompleted,
				Format:    ExportFormatJSON,
				CreatedAt: past.Add(-24 * time.Hour),
				ExpiresAt: past, // expired
			}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.GetExportStatus(context.Background(), scope, uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrExportExpired) {
		t.Fatalf("want ErrExportExpired, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// C. Data Export — ListExports (Reqs 13–15) [15-data-lifecycle §9]
// ═══════════════════════════════════════════════════════════════════════════════

// Req 13: ListExports returns paginated export summaries.
func TestListExports_ReturnsSummaries(t *testing.T) {
	scope := testScope()
	now := time.Now().UTC()

	svc := newTestService(
		&stubExportRepo{listByFamilyFn: func(_ context.Context, _ *shared.FamilyScope, _ *PaginationParams) ([]ExportRequest, error) {
			return []ExportRequest{
				{ID: uuid.Must(uuid.NewV7()), Status: ExportStatusCompleted, Format: ExportFormatJSON, CreatedAt: now},
				{ID: uuid.Must(uuid.NewV7()), Status: ExportStatusPending, Format: ExportFormatCSV, CreatedAt: now},
			}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	resp, err := svc.ListExports(context.Background(), scope, &PaginationParams{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListExports error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("Items len = %d, want 2", len(resp.Items))
	}
}

// Req 14: ListExports returns empty list when family has no exports.
func TestListExports_EmptyList(t *testing.T) {
	scope := testScope()

	svc := newTestService(
		&stubExportRepo{listByFamilyFn: func(_ context.Context, _ *shared.FamilyScope, _ *PaginationParams) ([]ExportRequest, error) {
			return []ExportRequest{}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	resp, err := svc.ListExports(context.Background(), scope, &PaginationParams{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListExports error: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("Items len = %d, want 0", len(resp.Items))
	}
}

// Req 15: ListExports forwards pagination params to repository.
func TestListExports_ForwardsPagination(t *testing.T) {
	scope := testScope()
	var capturedPagination *PaginationParams

	svc := newTestService(
		&stubExportRepo{listByFamilyFn: func(_ context.Context, _ *shared.FamilyScope, pagination *PaginationParams) ([]ExportRequest, error) {
			capturedPagination = pagination
			return []ExportRequest{}, nil
		}},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	params := &PaginationParams{Limit: 25, Offset: 50}
	_, err := svc.ListExports(context.Background(), scope, params)
	if err != nil {
		t.Fatalf("ListExports error: %v", err)
	}
	if capturedPagination == nil {
		t.Fatal("pagination not forwarded")
	}
	if capturedPagination.Limit != 25 || capturedPagination.Offset != 50 {
		t.Errorf("pagination = %+v, want Limit=25 Offset=50", capturedPagination)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// D. Data Export — ProcessExport (Reqs 16–22) [15-data-lifecycle §9]
// ═══════════════════════════════════════════════════════════════════════════════

// Req 16: ProcessExport calls each registered ExportHandler.
func TestProcessExport_CallsEachHandler(t *testing.T) {
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())
	familyID := scope.FamilyID()
	var calledHandlers []string

	svc := newTestService(
		&stubExportRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ExportRequest, error) {
				return &ExportRequest{ID: exportID, FamilyID: familyID, Status: ExportStatusPending, Format: ExportFormatJSON}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, _ ExportStatus, _ *string, _ *int64) error {
				return nil
			},
		},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{},
		[]ExportHandler{
			&stubExportHandler{domainName: "learning", exportFamilyData: func(_ context.Context, _ uuid.UUID, _ ExportFormat) ([]ExportFile, error) {
				calledHandlers = append(calledHandlers, "learning")
				return []ExportFile{{Filename: "activities.json", Content: []byte(`[]`)}}, nil
			}},
			&stubExportHandler{domainName: "social", exportFamilyData: func(_ context.Context, _ uuid.UUID, _ ExportFormat) ([]ExportFile, error) {
				calledHandlers = append(calledHandlers, "social")
				return []ExportFile{{Filename: "posts.json", Content: []byte(`[]`)}}, nil
			}},
		},
		nil,
	)

	err := svc.ProcessExport(context.Background(), exportID, familyID)
	if err != nil {
		t.Fatalf("ProcessExport error: %v", err)
	}
	if len(calledHandlers) != 2 {
		t.Fatalf("called handlers = %v, want [learning social]", calledHandlers)
	}
}

// Req 17: ProcessExport transitions status from pending to processing.
func TestProcessExport_TransitionsToProcessing(t *testing.T) {
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())
	familyID := scope.FamilyID()
	var statusUpdates []ExportStatus

	svc := newTestService(
		&stubExportRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ExportRequest, error) {
				return &ExportRequest{ID: exportID, FamilyID: familyID, Status: ExportStatusPending, Format: ExportFormatJSON}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, status ExportStatus, _ *string, _ *int64) error {
				statusUpdates = append(statusUpdates, status)
				return nil
			},
		},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.ProcessExport(context.Background(), exportID, familyID)
	if err != nil {
		t.Fatalf("ProcessExport error: %v", err)
	}
	if len(statusUpdates) < 1 || statusUpdates[0] != ExportStatusProcessing {
		t.Errorf("first status update = %v, want %q", statusUpdates, ExportStatusProcessing)
	}
}

// Req 18: ProcessExport transitions status to completed on success with archive_key and size_bytes.
func TestProcessExport_TransitionsToCompletedOnSuccess(t *testing.T) {
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())
	familyID := scope.FamilyID()
	var finalStatus ExportStatus
	var finalArchiveKey *string
	var finalSizeBytes *int64

	svc := newTestService(
		&stubExportRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ExportRequest, error) {
				return &ExportRequest{ID: exportID, FamilyID: familyID, Status: ExportStatusPending, Format: ExportFormatJSON}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, status ExportStatus, archiveKey *string, sizeBytes *int64) error {
				finalStatus = status
				finalArchiveKey = archiveKey
				finalSizeBytes = sizeBytes
				return nil
			},
		},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{},
		[]ExportHandler{
			&stubExportHandler{domainName: "learning", exportFamilyData: func(_ context.Context, _ uuid.UUID, _ ExportFormat) ([]ExportFile, error) {
				return []ExportFile{{Filename: "data.json", Content: []byte(`{"test":true}`)}}, nil
			}},
		},
		nil,
	)

	err := svc.ProcessExport(context.Background(), exportID, familyID)
	if err != nil {
		t.Fatalf("ProcessExport error: %v", err)
	}
	if finalStatus != ExportStatusCompleted {
		t.Errorf("final status = %q, want %q", finalStatus, ExportStatusCompleted)
	}
	if finalArchiveKey == nil {
		t.Error("archive_key is nil, want non-nil")
	}
	if finalSizeBytes == nil || *finalSizeBytes <= 0 {
		t.Errorf("size_bytes = %v, want > 0", finalSizeBytes)
	}
}

// Req 19: ProcessExport transitions status to failed when a handler errors.
func TestProcessExport_TransitionsToFailedOnHandlerError(t *testing.T) {
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())
	familyID := scope.FamilyID()
	var finalStatus ExportStatus

	svc := newTestService(
		&stubExportRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ExportRequest, error) {
				return &ExportRequest{ID: exportID, FamilyID: familyID, Status: ExportStatusPending, Format: ExportFormatJSON}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, status ExportStatus, _ *string, _ *int64) error {
				finalStatus = status
				return nil
			},
		},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{},
		[]ExportHandler{
			&stubExportHandler{domainName: "learning", exportFamilyData: func(_ context.Context, _ uuid.UUID, _ ExportFormat) ([]ExportFile, error) {
				return nil, errors.New("export failed")
			}},
		},
		nil,
	)

	err := svc.ProcessExport(context.Background(), exportID, familyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if finalStatus != ExportStatusFailed {
		t.Errorf("final status = %q, want %q", finalStatus, ExportStatusFailed)
	}
}

// Req 20: ProcessExport filters handlers by IncludeDomains when specified.
func TestProcessExport_FiltersHandlersByIncludeDomains(t *testing.T) {
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())
	familyID := scope.FamilyID()
	var calledHandlers []string

	svc := newTestService(
		&stubExportRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ExportRequest, error) {
				return &ExportRequest{
					ID:             exportID,
					FamilyID:       familyID,
					Status:         ExportStatusPending,
					Format:         ExportFormatJSON,
					IncludeDomains: []string{"social"},
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, _ ExportStatus, _ *string, _ *int64) error {
				return nil
			},
		},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{},
		[]ExportHandler{
			&stubExportHandler{domainName: "learning", exportFamilyData: func(_ context.Context, _ uuid.UUID, _ ExportFormat) ([]ExportFile, error) {
				calledHandlers = append(calledHandlers, "learning")
				return []ExportFile{{Filename: "data.json", Content: []byte(`[]`)}}, nil
			}},
			&stubExportHandler{domainName: "social", exportFamilyData: func(_ context.Context, _ uuid.UUID, _ ExportFormat) ([]ExportFile, error) {
				calledHandlers = append(calledHandlers, "social")
				return []ExportFile{{Filename: "posts.json", Content: []byte(`[]`)}}, nil
			}},
		},
		nil,
	)

	err := svc.ProcessExport(context.Background(), exportID, familyID)
	if err != nil {
		t.Fatalf("ProcessExport error: %v", err)
	}
	if len(calledHandlers) != 1 || calledHandlers[0] != "social" {
		t.Errorf("called handlers = %v, want [social]", calledHandlers)
	}
}

// Req 21: ProcessExport calls all handlers when IncludeDomains is nil.
func TestProcessExport_CallsAllHandlersWhenIncludeDomainsNil(t *testing.T) {
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())
	familyID := scope.FamilyID()
	var calledHandlers []string

	svc := newTestService(
		&stubExportRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ExportRequest, error) {
				return &ExportRequest{
					ID:             exportID,
					FamilyID:       familyID,
					Status:         ExportStatusPending,
					Format:         ExportFormatJSON,
					IncludeDomains: nil,
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, _ ExportStatus, _ *string, _ *int64) error {
				return nil
			},
		},
		nil, nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{},
		[]ExportHandler{
			&stubExportHandler{domainName: "learning", exportFamilyData: func(_ context.Context, _ uuid.UUID, _ ExportFormat) ([]ExportFile, error) {
				calledHandlers = append(calledHandlers, "learning")
				return []ExportFile{{Filename: "data.json", Content: []byte(`[]`)}}, nil
			}},
			&stubExportHandler{domainName: "social", exportFamilyData: func(_ context.Context, _ uuid.UUID, _ ExportFormat) ([]ExportFile, error) {
				calledHandlers = append(calledHandlers, "social")
				return []ExportFile{{Filename: "posts.json", Content: []byte(`[]`)}}, nil
			}},
		},
		nil,
	)

	err := svc.ProcessExport(context.Background(), exportID, familyID)
	if err != nil {
		t.Fatalf("ProcessExport error: %v", err)
	}
	if len(calledHandlers) != 2 {
		t.Fatalf("called handlers = %v, want [learning social]", calledHandlers)
	}
}

// Req 22: ProcessExport publishes DataExportCompleted event on success.
func TestProcessExport_PublishesCompletedEvent(t *testing.T) {
	scope := testScope()
	exportID := uuid.Must(uuid.NewV7())
	familyID := scope.FamilyID()

	bus := shared.NewEventBus()
	var publishedEvent *DataExportCompleted
	bus.Subscribe(
		eventType[DataExportCompleted](),
		handlerFunc(func(_ context.Context, event shared.DomainEvent) error {
			e := event.(DataExportCompleted)
			publishedEvent = &e
			return nil
		}),
	)

	svc := NewLifecycleService(
		&stubExportRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ExportRequest, error) {
				return &ExportRequest{ID: exportID, FamilyID: familyID, Status: ExportStatusPending, Format: ExportFormatJSON}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, _ ExportStatus, _ *string, _ *int64) error {
				return nil
			},
		},
		nil, nil, &stubIamService{}, &stubBillingService{},
		bus, &stubJobEnqueuer{},
		[]ExportHandler{
			&stubExportHandler{domainName: "learning", exportFamilyData: func(_ context.Context, _ uuid.UUID, _ ExportFormat) ([]ExportFile, error) {
				return []ExportFile{{Filename: "data.json", Content: []byte(`[]`)}}, nil
			}},
		},
		nil,
	)

	err := svc.ProcessExport(context.Background(), exportID, familyID)
	if err != nil {
		t.Fatalf("ProcessExport error: %v", err)
	}
	if publishedEvent == nil {
		t.Fatal("DataExportCompleted event was not published")
	}
	if publishedEvent.ExportID != exportID {
		t.Errorf("event.ExportID = %v, want %v", publishedEvent.ExportID, exportID)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// E. Account Deletion — RequestDeletion / Family (Reqs 23–28) [15-data-lifecycle §10.1]
// ═══════════════════════════════════════════════════════════════════════════════

// Req 23: RequestDeletion creates family deletion with 30-day grace period and status grace_period.
func TestRequestDeletion_FamilyCreatesWithGracePeriod(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()

	var capturedInput *CreateDeletionRequest
	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
			createFn: func(_ context.Context, _ *shared.FamilyScope, input *CreateDeletionRequest) (*DeletionRequest, error) {
				capturedInput = input
				return &DeletionRequest{ID: uuid.Must(uuid.NewV7())}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeFamily,
	})
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}
	if capturedInput == nil {
		t.Fatal("repo.Create was not called")
	}
	if capturedInput.Status != DeletionStatusGracePeriod {
		t.Errorf("Status = %q, want %q", capturedInput.Status, DeletionStatusGracePeriod)
	}
	expectedGrace := time.Now().UTC().Add(30 * 24 * time.Hour)
	if capturedInput.GracePeriodEndsAt.Before(expectedGrace.Add(-1*time.Minute)) ||
		capturedInput.GracePeriodEndsAt.After(expectedGrace.Add(1*time.Minute)) {
		t.Errorf("GracePeriodEndsAt = %v, want ~%v", capturedInput.GracePeriodEndsAt, expectedGrace)
	}
}

// Req 24: RequestDeletion returns ErrNotPrimaryParent when non-primary requests family deletion.
func TestRequestDeletion_ReturnsErrNotPrimaryParent(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = false
	scope := testScope()

	svc := newTestService(
		nil, &stubDeletionRepo{}, nil,
		&stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeFamily,
	})
	if !errors.Is(err, ErrNotPrimaryParent) {
		t.Fatalf("want ErrNotPrimaryParent, got %v", err)
	}
}

// Req 25: RequestDeletion returns ErrDeletionAlreadyPending when active request exists.
func TestRequestDeletion_ReturnsErrDeletionAlreadyPending(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return &DeletionRequest{ID: uuid.Must(uuid.NewV7())}, nil // active request exists
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeFamily,
	})
	if !errors.Is(err, ErrDeletionAlreadyPending) {
		t.Fatalf("want ErrDeletionAlreadyPending, got %v", err)
	}
}

// Req 26: RequestDeletion does NOT enqueue immediate job for family deletion (grace period).
func TestRequestDeletion_FamilyDoesNotEnqueueImmediateJob(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()
	jobEnqueued := false

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
			createFn: func(_ context.Context, _ *shared.FamilyScope, _ *CreateDeletionRequest) (*DeletionRequest, error) {
				return &DeletionRequest{ID: uuid.Must(uuid.NewV7())}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{enqueueFn: func(_ context.Context, _ shared.JobPayload) error {
			jobEnqueued = true
			return nil
		}},
		nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeFamily,
	})
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}
	if jobEnqueued {
		t.Error("job was enqueued for family deletion — should wait for grace period")
	}
}

// Req 27: RequestDeletion publishes AccountDeletionRequested event.
func TestRequestDeletion_PublishesEvent(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()

	bus := shared.NewEventBus()
	var publishedEvent *AccountDeletionRequested
	bus.Subscribe(
		eventType[AccountDeletionRequested](),
		handlerFunc(func(_ context.Context, event shared.DomainEvent) error {
			e := event.(AccountDeletionRequested)
			publishedEvent = &e
			return nil
		}),
	)

	svc := NewLifecycleService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
			createFn: func(_ context.Context, _ *shared.FamilyScope, _ *CreateDeletionRequest) (*DeletionRequest, error) {
				return &DeletionRequest{
					ID:                uuid.Must(uuid.NewV7()),
					FamilyID:          scope.FamilyID(),
					GracePeriodEndsAt: time.Now().Add(30 * 24 * time.Hour),
				}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		bus, &stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeFamily,
	})
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}
	if publishedEvent == nil {
		t.Fatal("AccountDeletionRequested event was not published")
	}
}

// Req 28: RequestDeletion returns error when repository Create fails.
func TestRequestDeletion_ReturnsErrorOnRepoFailure(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
			createFn: func(_ context.Context, _ *shared.FamilyScope, _ *CreateDeletionRequest) (*DeletionRequest, error) {
				return nil, errors.New("db down")
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeFamily,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// F. Account Deletion — RequestDeletion / Student (Reqs 29–30) [15-data-lifecycle §10.2]
// ═══════════════════════════════════════════════════════════════════════════════

// Req 29: RequestDeletion creates student deletion with 7-day grace period.
func TestRequestDeletion_StudentCreatesWithSevenDayGrace(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	var capturedInput *CreateDeletionRequest
	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
			createFn: func(_ context.Context, _ *shared.FamilyScope, input *CreateDeletionRequest) (*DeletionRequest, error) {
				capturedInput = input
				return &DeletionRequest{ID: uuid.Must(uuid.NewV7())}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeStudent,
		StudentID:    &studentID,
	})
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}
	expectedGrace := time.Now().UTC().Add(7 * 24 * time.Hour)
	if capturedInput.GracePeriodEndsAt.Before(expectedGrace.Add(-1*time.Minute)) ||
		capturedInput.GracePeriodEndsAt.After(expectedGrace.Add(1*time.Minute)) {
		t.Errorf("GracePeriodEndsAt = %v, want ~%v", capturedInput.GracePeriodEndsAt, expectedGrace)
	}
	if capturedInput.StudentID == nil || *capturedInput.StudentID != studentID {
		t.Errorf("StudentID = %v, want %v", capturedInput.StudentID, studentID)
	}
}

// Req 30: RequestDeletion requires StudentID for student deletion type (validation error if nil).
func TestRequestDeletion_StudentRequiresStudentID(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()

	svc := newTestService(
		nil, &stubDeletionRepo{}, nil,
		&stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeStudent,
		StudentID:    nil,
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// G. Account Deletion — RequestDeletion / COPPA (Reqs 31–35) [15-data-lifecycle §10.3]
// ═══════════════════════════════════════════════════════════════════════════════

// Req 31: COPPA deletion has 0-day grace period (grace_period_ends_at ≈ now).
func TestRequestDeletion_CoppaZeroGracePeriod(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	var capturedInput *CreateDeletionRequest
	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
			createFn: func(_ context.Context, _ *shared.FamilyScope, input *CreateDeletionRequest) (*DeletionRequest, error) {
				capturedInput = input
				return &DeletionRequest{ID: uuid.Must(uuid.NewV7()), FamilyID: scope.FamilyID()}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeCoppa,
		StudentID:    &studentID,
	})
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}
	now := time.Now().UTC()
	if capturedInput.GracePeriodEndsAt.After(now.Add(1 * time.Minute)) {
		t.Errorf("GracePeriodEndsAt = %v, want ≈ now (%v)", capturedInput.GracePeriodEndsAt, now)
	}
}

// Req 32: COPPA deletion transitions directly to processing status.
func TestRequestDeletion_CoppaStatusIsProcessing(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	var capturedInput *CreateDeletionRequest
	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
			createFn: func(_ context.Context, _ *shared.FamilyScope, input *CreateDeletionRequest) (*DeletionRequest, error) {
				capturedInput = input
				return &DeletionRequest{ID: uuid.Must(uuid.NewV7()), FamilyID: scope.FamilyID()}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeCoppa,
		StudentID:    &studentID,
	})
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}
	if capturedInput.Status != DeletionStatusProcessing {
		t.Errorf("Status = %q, want %q", capturedInput.Status, DeletionStatusProcessing)
	}
}

// Req 33: COPPA deletion enqueues immediate ProcessDeletionJob.
func TestRequestDeletion_CoppaEnqueuesImmediateJob(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())
	deletionID := uuid.Must(uuid.NewV7())

	var capturedPayload shared.JobPayload
	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
			createFn: func(_ context.Context, _ *shared.FamilyScope, _ *CreateDeletionRequest) (*DeletionRequest, error) {
				return &DeletionRequest{ID: deletionID, FamilyID: scope.FamilyID()}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{enqueueFn: func(_ context.Context, payload shared.JobPayload) error {
			capturedPayload = payload
			return nil
		}},
		nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeCoppa,
		StudentID:    &studentID,
	})
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}
	if capturedPayload == nil {
		t.Fatal("job was not enqueued for COPPA deletion")
	}
	job, ok := capturedPayload.(ProcessDeletionJob)
	if !ok {
		t.Fatalf("payload type = %T, want ProcessDeletionJob", capturedPayload)
	}
	if job.DeletionID != deletionID {
		t.Errorf("job.DeletionID = %v, want %v", job.DeletionID, deletionID)
	}
}

// Req 34: COPPA deletion publishes CoppaDeleteRequested event.
func TestRequestDeletion_CoppaPublishesEvent(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var publishedEvent *CoppaDeleteRequested
	bus.Subscribe(
		eventType[CoppaDeleteRequested](),
		handlerFunc(func(_ context.Context, event shared.DomainEvent) error {
			e := event.(CoppaDeleteRequested)
			publishedEvent = &e
			return nil
		}),
	)

	svc := NewLifecycleService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
			createFn: func(_ context.Context, _ *shared.FamilyScope, _ *CreateDeletionRequest) (*DeletionRequest, error) {
				return &DeletionRequest{ID: uuid.Must(uuid.NewV7()), FamilyID: scope.FamilyID()}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		bus, &stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeCoppa,
		StudentID:    &studentID,
	})
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}
	if publishedEvent == nil {
		t.Fatal("CoppaDeleteRequested event was not published")
	}
	if publishedEvent.StudentID != studentID {
		t.Errorf("event.StudentID = %v, want %v", publishedEvent.StudentID, studentID)
	}
}

// Req 35: COPPA deletion requires StudentID (validation error if nil).
func TestRequestDeletion_CoppaRequiresStudentID(t *testing.T) {
	auth := testAuth()
	auth.IsPrimaryParent = true
	scope := testScope()

	svc := newTestService(
		nil, &stubDeletionRepo{}, nil,
		&stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RequestDeletion(context.Background(), auth, scope, &RequestDeletionInput{
		DeletionType: DeletionTypeCoppa,
		StudentID:    nil,
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// H. Account Deletion — GetDeletionStatus (Reqs 36–37)
// ═══════════════════════════════════════════════════════════════════════════════

// Req 36: GetDeletionStatus returns active deletion status response.
func TestGetDeletionStatus_ReturnsActiveStatus(t *testing.T) {
	scope := testScope()
	deletionID := uuid.Must(uuid.NewV7())
	now := time.Now().UTC()

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return &DeletionRequest{
					ID:                deletionID,
					Status:            DeletionStatusGracePeriod,
					DeletionType:      DeletionTypeFamily,
					GracePeriodEndsAt: now.Add(30 * 24 * time.Hour),
					ExportOffered:     true,
					CreatedAt:         now,
				}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	resp, err := svc.GetDeletionStatus(context.Background(), scope)
	if err != nil {
		t.Fatalf("GetDeletionStatus error: %v", err)
	}
	if resp.ID != deletionID {
		t.Errorf("ID = %v, want %v", resp.ID, deletionID)
	}
	if resp.Status != DeletionStatusGracePeriod {
		t.Errorf("Status = %q, want %q", resp.Status, DeletionStatusGracePeriod)
	}
}

// Req 37: GetDeletionStatus returns ErrDeletionNotFound when no active deletion.
func TestGetDeletionStatus_NotFound(t *testing.T) {
	scope := testScope()

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.GetDeletionStatus(context.Background(), scope)
	if !errors.Is(err, ErrDeletionNotFound) {
		t.Fatalf("want ErrDeletionNotFound, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// I. Account Deletion — CancelDeletion (Reqs 38–41)
// ═══════════════════════════════════════════════════════════════════════════════

// Req 38: CancelDeletion succeeds during grace period.
func TestCancelDeletion_SucceedsDuringGracePeriod(t *testing.T) {
	scope := testScope()
	deletionID := uuid.Must(uuid.NewV7())
	cancelCalled := false

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return &DeletionRequest{
					ID:                deletionID,
					Status:            DeletionStatusGracePeriod,
					GracePeriodEndsAt: time.Now().UTC().Add(24 * time.Hour), // still in grace period
				}, nil
			},
			cancelFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
				cancelCalled = true
				return nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.CancelDeletion(context.Background(), scope)
	if err != nil {
		t.Fatalf("CancelDeletion error: %v", err)
	}
	if !cancelCalled {
		t.Error("repo.Cancel was not called")
	}
}

// Req 39: CancelDeletion returns ErrGracePeriodExpired after grace period ends.
func TestCancelDeletion_ReturnsErrGracePeriodExpired(t *testing.T) {
	scope := testScope()

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return &DeletionRequest{
					ID:                uuid.Must(uuid.NewV7()),
					Status:            DeletionStatusGracePeriod,
					GracePeriodEndsAt: time.Now().UTC().Add(-1 * time.Hour), // expired
				}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.CancelDeletion(context.Background(), scope)
	if !errors.Is(err, ErrGracePeriodExpired) {
		t.Fatalf("want ErrGracePeriodExpired, got %v", err)
	}
}

// Req 40: CancelDeletion returns not-found error when no active deletion exists.
func TestCancelDeletion_NotFound(t *testing.T) {
	scope := testScope()

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return nil, ErrDeletionNotFound
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.CancelDeletion(context.Background(), scope)
	if !errors.Is(err, ErrDeletionNotFound) {
		t.Fatalf("want ErrDeletionNotFound, got %v", err)
	}
}

// Req 41: CancelDeletion calls repository Cancel to set status cancelled.
func TestCancelDeletion_CallsRepoCancel(t *testing.T) {
	scope := testScope()
	deletionID := uuid.Must(uuid.NewV7())
	var cancelledID uuid.UUID

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) (*DeletionRequest, error) {
				return &DeletionRequest{
					ID:                deletionID,
					Status:            DeletionStatusGracePeriod,
					GracePeriodEndsAt: time.Now().UTC().Add(24 * time.Hour),
				}, nil
			},
			cancelFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) error {
				cancelledID = id
				return nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.CancelDeletion(context.Background(), scope)
	if err != nil {
		t.Fatalf("CancelDeletion error: %v", err)
	}
	if cancelledID != deletionID {
		t.Errorf("cancelled ID = %v, want %v", cancelledID, deletionID)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// J. Deletion Job — ProcessDeletion (Reqs 42–51)
// ═══════════════════════════════════════════════════════════════════════════════

// Req 42: ProcessDeletion finds requests past grace period via FindReadyForDeletion.
func TestProcessDeletion_FindsReadyRequests(t *testing.T) {
	findCalled := false

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findReadyForDeletionFn: func(_ context.Context) ([]DeletionRequest, error) {
				findCalled = true
				return []DeletionRequest{}, nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.ProcessDeletion(context.Background())
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if !findCalled {
		t.Error("FindReadyForDeletion was not called")
	}
}

// Req 43: ProcessDeletion transitions each to processing status.
func TestProcessDeletion_TransitionsToProcessing(t *testing.T) {
	deletionID := uuid.Must(uuid.NewV7())
	familyID := uuid.Must(uuid.NewV7())
	var statusUpdates []DeletionStatus

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findReadyForDeletionFn: func(_ context.Context) ([]DeletionRequest, error) {
				return []DeletionRequest{
					{ID: deletionID, FamilyID: familyID, DeletionType: DeletionTypeFamily},
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, status DeletionStatus) error {
				statusUpdates = append(statusUpdates, status)
				return nil
			},
			updateDomainStatusFn: func(_ context.Context, _ uuid.UUID, _ string, _ bool) error {
				return nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.ProcessDeletion(context.Background())
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if len(statusUpdates) < 1 || statusUpdates[0] != DeletionStatusProcessing {
		t.Errorf("first status update = %v, want %q", statusUpdates, DeletionStatusProcessing)
	}
}

// Req 44: ProcessDeletion revokes all Kratos sessions for the family.
func TestProcessDeletion_RevokesKratosSessions(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	var revokedFamilyID uuid.UUID

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findReadyForDeletionFn: func(_ context.Context) ([]DeletionRequest, error) {
				return []DeletionRequest{
					{ID: uuid.Must(uuid.NewV7()), FamilyID: familyID, DeletionType: DeletionTypeFamily},
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, _ DeletionStatus) error { return nil },
			updateDomainStatusFn: func(_ context.Context, _ uuid.UUID, _ string, _ bool) error {
				return nil
			},
		},
		nil,
		&stubIamService{
			revokeFamilySessionsFn: func(_ context.Context, fid uuid.UUID) error {
				revokedFamilyID = fid
				return nil
			},
		},
		&stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.ProcessDeletion(context.Background())
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if revokedFamilyID != familyID {
		t.Errorf("revoked family ID = %v, want %v", revokedFamilyID, familyID)
	}
}

// Req 45: ProcessDeletion cancels subscriptions via BillingService.
func TestProcessDeletion_CancelsSubscriptions(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	var cancelledFamilyID uuid.UUID

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findReadyForDeletionFn: func(_ context.Context) ([]DeletionRequest, error) {
				return []DeletionRequest{
					{ID: uuid.Must(uuid.NewV7()), FamilyID: familyID, DeletionType: DeletionTypeFamily},
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, _ DeletionStatus) error { return nil },
			updateDomainStatusFn: func(_ context.Context, _ uuid.UUID, _ string, _ bool) error {
				return nil
			},
		},
		nil, &stubIamService{},
		&stubBillingService{
			cancelFamilySubscriptionsFn: func(_ context.Context, fid uuid.UUID) error {
				cancelledFamilyID = fid
				return nil
			},
		},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.ProcessDeletion(context.Background())
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if cancelledFamilyID != familyID {
		t.Errorf("cancelled family ID = %v, want %v", cancelledFamilyID, familyID)
	}
}

// Req 46: ProcessDeletion calls DeleteFamilyData on each DeletionHandler for family type.
func TestProcessDeletion_CallsDeleteFamilyData(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	var calledHandlers []string

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findReadyForDeletionFn: func(_ context.Context) ([]DeletionRequest, error) {
				return []DeletionRequest{
					{ID: uuid.Must(uuid.NewV7()), FamilyID: familyID, DeletionType: DeletionTypeFamily},
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, _ DeletionStatus) error { return nil },
			updateDomainStatusFn: func(_ context.Context, _ uuid.UUID, _ string, _ bool) error {
				return nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil,
		[]DeletionHandler{
			&stubDeletionHandler{domainName: "learning", deleteFamilyData: func(_ context.Context, _ uuid.UUID) error {
				calledHandlers = append(calledHandlers, "learning")
				return nil
			}},
			&stubDeletionHandler{domainName: "social", deleteFamilyData: func(_ context.Context, _ uuid.UUID) error {
				calledHandlers = append(calledHandlers, "social")
				return nil
			}},
		},
	)

	err := svc.ProcessDeletion(context.Background())
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if len(calledHandlers) != 2 {
		t.Fatalf("called handlers = %v, want [learning social]", calledHandlers)
	}
}

// Req 47: ProcessDeletion calls DeleteStudentData on each DeletionHandler for student type.
func TestProcessDeletion_CallsDeleteStudentData(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	studentID := uuid.Must(uuid.NewV7())
	var calledHandlers []string

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findReadyForDeletionFn: func(_ context.Context) ([]DeletionRequest, error) {
				return []DeletionRequest{
					{ID: uuid.Must(uuid.NewV7()), FamilyID: familyID, DeletionType: DeletionTypeStudent, StudentID: &studentID},
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, _ DeletionStatus) error { return nil },
			updateDomainStatusFn: func(_ context.Context, _ uuid.UUID, _ string, _ bool) error {
				return nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil,
		[]DeletionHandler{
			&stubDeletionHandler{domainName: "learning", deleteStudentData: func(_ context.Context, _ uuid.UUID, sid uuid.UUID) error {
				if sid == studentID {
					calledHandlers = append(calledHandlers, "learning")
				}
				return nil
			}},
		},
	)

	err := svc.ProcessDeletion(context.Background())
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if len(calledHandlers) != 1 || calledHandlers[0] != "learning" {
		t.Fatalf("called handlers = %v, want [learning]", calledHandlers)
	}
}

// Req 48: ProcessDeletion updates domain_status as each handler completes.
func TestProcessDeletion_UpdatesDomainStatus(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	deletionID := uuid.Must(uuid.NewV7())
	var domainUpdates []string

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findReadyForDeletionFn: func(_ context.Context) ([]DeletionRequest, error) {
				return []DeletionRequest{
					{ID: deletionID, FamilyID: familyID, DeletionType: DeletionTypeFamily},
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, _ DeletionStatus) error { return nil },
			updateDomainStatusFn: func(_ context.Context, _ uuid.UUID, domain string, _ bool) error {
				domainUpdates = append(domainUpdates, domain)
				return nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil,
		[]DeletionHandler{
			&stubDeletionHandler{domainName: "learning"},
			&stubDeletionHandler{domainName: "social"},
		},
	)

	err := svc.ProcessDeletion(context.Background())
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if len(domainUpdates) != 2 {
		t.Fatalf("domain updates = %v, want 2 entries", domainUpdates)
	}
}

// Req 49: ProcessDeletion transitions to completed after all handlers succeed.
func TestProcessDeletion_TransitionsToCompleted(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	var statusUpdates []DeletionStatus

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findReadyForDeletionFn: func(_ context.Context) ([]DeletionRequest, error) {
				return []DeletionRequest{
					{ID: uuid.Must(uuid.NewV7()), FamilyID: familyID, DeletionType: DeletionTypeFamily},
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ uuid.UUID, status DeletionStatus) error {
				statusUpdates = append(statusUpdates, status)
				return nil
			},
			updateDomainStatusFn: func(_ context.Context, _ uuid.UUID, _ string, _ bool) error {
				return nil
			},
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil,
		[]DeletionHandler{&stubDeletionHandler{domainName: "learning"}},
	)

	err := svc.ProcessDeletion(context.Background())
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if len(statusUpdates) < 2 || statusUpdates[len(statusUpdates)-1] != DeletionStatusCompleted {
		t.Errorf("status updates = %v, want final status %q", statusUpdates, DeletionStatusCompleted)
	}
}

// Req 50: ProcessDeletion continues with remaining handlers if one fails.
func TestProcessDeletion_ContinuesOnHandlerFailure(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	var calledHandlers []string

	svc := newTestService(
		nil,
		&stubDeletionRepo{
			findReadyForDeletionFn: func(_ context.Context) ([]DeletionRequest, error) {
				return []DeletionRequest{
					{ID: uuid.Must(uuid.NewV7()), FamilyID: familyID, DeletionType: DeletionTypeFamily},
				}, nil
			},
			updateStatusFn:       func(_ context.Context, _ uuid.UUID, _ DeletionStatus) error { return nil },
			updateDomainStatusFn: func(_ context.Context, _ uuid.UUID, _ string, _ bool) error { return nil },
		},
		nil, &stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil,
		[]DeletionHandler{
			&stubDeletionHandler{domainName: "learning", deleteFamilyData: func(_ context.Context, _ uuid.UUID) error {
				calledHandlers = append(calledHandlers, "learning")
				return errors.New("learning failed")
			}},
			&stubDeletionHandler{domainName: "social", deleteFamilyData: func(_ context.Context, _ uuid.UUID) error {
				calledHandlers = append(calledHandlers, "social")
				return nil
			}},
		},
	)

	// ProcessDeletion should not return error — it logs failures and continues.
	err := svc.ProcessDeletion(context.Background())
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if len(calledHandlers) != 2 {
		t.Fatalf("called handlers = %v, want [learning social]", calledHandlers)
	}
}

// Req 51: ProcessDeletion publishes AccountDeletionCompleted event on success.
func TestProcessDeletion_PublishesCompletedEvent(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var publishedEvent *AccountDeletionCompleted
	bus.Subscribe(
		eventType[AccountDeletionCompleted](),
		handlerFunc(func(_ context.Context, event shared.DomainEvent) error {
			e := event.(AccountDeletionCompleted)
			publishedEvent = &e
			return nil
		}),
	)

	svc := NewLifecycleService(
		nil,
		&stubDeletionRepo{
			findReadyForDeletionFn: func(_ context.Context) ([]DeletionRequest, error) {
				return []DeletionRequest{
					{ID: uuid.Must(uuid.NewV7()), FamilyID: familyID, DeletionType: DeletionTypeFamily},
				}, nil
			},
			updateStatusFn:       func(_ context.Context, _ uuid.UUID, _ DeletionStatus) error { return nil },
			updateDomainStatusFn: func(_ context.Context, _ uuid.UUID, _ string, _ bool) error { return nil },
		},
		nil, &stubIamService{}, &stubBillingService{},
		bus, &stubJobEnqueuer{}, nil,
		[]DeletionHandler{&stubDeletionHandler{domainName: "learning"}},
	)

	err := svc.ProcessDeletion(context.Background())
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if publishedEvent == nil {
		t.Fatal("AccountDeletionCompleted event was not published")
	}
	if publishedEvent.FamilyID != familyID {
		t.Errorf("event.FamilyID = %v, want %v", publishedEvent.FamilyID, familyID)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// K. Account Recovery — InitiateRecovery (Reqs 52–55)
// ═══════════════════════════════════════════════════════════════════════════════

// Req 52: InitiateRecovery always returns UUID (enumeration prevention).
func TestInitiateRecovery_AlwaysReturnsUUID(t *testing.T) {
	recoveryID := uuid.Must(uuid.NewV7())

	svc := newTestService(
		nil, nil,
		&stubRecoveryRepo{
			createFn: func(_ context.Context, _ *CreateRecoveryRequest) (*RecoveryRequest, error) {
				return &RecoveryRequest{ID: recoveryID}, nil
			},
		},
		&stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	id, err := svc.InitiateRecovery(context.Background(), &InitiateRecoveryInput{Email: "test@example.com"})
	if err != nil {
		t.Fatalf("InitiateRecovery error: %v", err)
	}
	if id == uuid.Nil {
		t.Error("returned UUID is nil, want non-nil")
	}
}

// Req 53: InitiateRecovery creates recovery request with verification_method=email and status=pending.
func TestInitiateRecovery_CreatesWithEmailMethod(t *testing.T) {
	var capturedInput *CreateRecoveryRequest

	svc := newTestService(
		nil, nil,
		&stubRecoveryRepo{
			createFn: func(_ context.Context, input *CreateRecoveryRequest) (*RecoveryRequest, error) {
				capturedInput = input
				return &RecoveryRequest{ID: uuid.Must(uuid.NewV7())}, nil
			},
		},
		&stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.InitiateRecovery(context.Background(), &InitiateRecoveryInput{Email: "test@example.com"})
	if err != nil {
		t.Fatalf("InitiateRecovery error: %v", err)
	}
	if capturedInput.VerificationMethod != VerificationMethodEmail {
		t.Errorf("VerificationMethod = %q, want %q", capturedInput.VerificationMethod, VerificationMethodEmail)
	}
	if capturedInput.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", capturedInput.Email, "test@example.com")
	}
}

// Req 54: InitiateRecovery requires non-empty email (validation error).
func TestInitiateRecovery_RequiresEmail(t *testing.T) {
	svc := newTestService(
		nil, nil, &stubRecoveryRepo{},
		&stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.InitiateRecovery(context.Background(), &InitiateRecoveryInput{Email: ""})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

// Req 55: InitiateRecovery still succeeds even when IAM/Kratos call fails (enumeration prevention).
func TestInitiateRecovery_SucceedsEvenOnKratosFailure(t *testing.T) {
	svc := newTestService(
		nil, nil,
		&stubRecoveryRepo{
			createFn: func(_ context.Context, _ *CreateRecoveryRequest) (*RecoveryRequest, error) {
				return &RecoveryRequest{ID: uuid.Must(uuid.NewV7())}, nil
			},
		},
		&stubIamService{
			initiateRecoveryFlowFn: func(_ context.Context, _ string) error {
				return errors.New("kratos down")
			},
		},
		&stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	id, err := svc.InitiateRecovery(context.Background(), &InitiateRecoveryInput{Email: "test@example.com"})
	if err != nil {
		t.Fatalf("InitiateRecovery should succeed even on Kratos failure, got: %v", err)
	}
	if id == uuid.Nil {
		t.Error("returned UUID is nil, want non-nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// L. Account Recovery — GetRecoveryStatus (Reqs 56–58)
// ═══════════════════════════════════════════════════════════════════════════════

// Req 56: GetRecoveryStatus returns status for valid recovery ID.
func TestGetRecoveryStatus_ReturnsStatus(t *testing.T) {
	recoveryID := uuid.Must(uuid.NewV7())
	now := time.Now().UTC()

	svc := newTestService(
		nil, nil,
		&stubRecoveryRepo{
			findByIDFn: func(_ context.Context, id uuid.UUID) (*RecoveryRequest, error) {
				return &RecoveryRequest{
					ID:                 id,
					Status:             RecoveryStatusPending,
					VerificationMethod: VerificationMethodEmail,
					CreatedAt:          now,
					ExpiresAt:          now.Add(7 * 24 * time.Hour),
				}, nil
			},
		},
		&stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	resp, err := svc.GetRecoveryStatus(context.Background(), recoveryID)
	if err != nil {
		t.Fatalf("GetRecoveryStatus error: %v", err)
	}
	if resp.Status != RecoveryStatusPending {
		t.Errorf("Status = %q, want %q", resp.Status, RecoveryStatusPending)
	}
}

// Req 57: GetRecoveryStatus returns ErrRecoveryNotFound for unknown ID.
func TestGetRecoveryStatus_NotFound(t *testing.T) {
	svc := newTestService(
		nil, nil,
		&stubRecoveryRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*RecoveryRequest, error) {
				return nil, ErrRecoveryNotFound
			},
		},
		&stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.GetRecoveryStatus(context.Background(), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrRecoveryNotFound) {
		t.Fatalf("want ErrRecoveryNotFound, got %v", err)
	}
}

// Req 58: GetRecoveryStatus returns ErrRecoveryExpired for expired request.
func TestGetRecoveryStatus_Expired(t *testing.T) {
	past := time.Now().UTC().Add(-1 * time.Hour)

	svc := newTestService(
		nil, nil,
		&stubRecoveryRepo{
			findByIDFn: func(_ context.Context, id uuid.UUID) (*RecoveryRequest, error) {
				return &RecoveryRequest{
					ID:                 id,
					Status:             RecoveryStatusPending,
					VerificationMethod: VerificationMethodEmail,
					CreatedAt:          past.Add(-7 * 24 * time.Hour),
					ExpiresAt:          past, // expired
				}, nil
			},
		},
		&stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.GetRecoveryStatus(context.Background(), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrRecoveryExpired) {
		t.Fatalf("want ErrRecoveryExpired, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// M. Session Management — ListSessions (Reqs 59–61)
// ═══════════════════════════════════════════════════════════════════════════════

// Req 59: ListSessions returns sessions from IAM service.
func TestListSessions_ReturnsSessions(t *testing.T) {
	auth := testAuthWithSession("current-session-id")

	svc := newTestService(
		nil, nil, nil,
		&stubIamService{
			listSessionsFn: func(_ context.Context, _ uuid.UUID) ([]SessionInfo, error) {
				return []SessionInfo{
					{SessionID: "session-1", LastActive: time.Now()},
					{SessionID: "current-session-id", LastActive: time.Now()},
				}, nil
			},
		},
		&stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	sessions, err := svc.ListSessions(context.Background(), auth)
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("sessions len = %d, want 2", len(sessions))
	}
}

// Req 60: ListSessions marks current session with IsCurrent=true.
func TestListSessions_MarksCurrentSession(t *testing.T) {
	auth := testAuthWithSession("current-session-id")

	svc := newTestService(
		nil, nil, nil,
		&stubIamService{
			listSessionsFn: func(_ context.Context, _ uuid.UUID) ([]SessionInfo, error) {
				return []SessionInfo{
					{SessionID: "other-session", LastActive: time.Now()},
					{SessionID: "current-session-id", LastActive: time.Now()},
				}, nil
			},
		},
		&stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	sessions, err := svc.ListSessions(context.Background(), auth)
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}

	var currentFound bool
	for _, s := range sessions {
		if s.SessionID == "current-session-id" {
			if !s.IsCurrent {
				t.Error("current session not marked as IsCurrent=true")
			}
			currentFound = true
		} else if s.IsCurrent {
			t.Errorf("non-current session %q marked as IsCurrent=true", s.SessionID)
		}
	}
	if !currentFound {
		t.Error("current session not found in results")
	}
}

// Req 61: ListSessions returns empty slice when no sessions exist.
func TestListSessions_EmptySlice(t *testing.T) {
	auth := testAuthWithSession("current-session-id")

	svc := newTestService(
		nil, nil, nil,
		&stubIamService{
			listSessionsFn: func(_ context.Context, _ uuid.UUID) ([]SessionInfo, error) {
				return []SessionInfo{}, nil
			},
		},
		&stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	sessions, err := svc.ListSessions(context.Background(), auth)
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions len = %d, want 0", len(sessions))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// N. Session Management — RevokeSession (Reqs 62–65)
// ═══════════════════════════════════════════════════════════════════════════════

// Req 62: RevokeSession revokes a non-current session successfully.
func TestRevokeSession_Success(t *testing.T) {
	auth := testAuthWithSession("current-session-id")
	var revokedSessionID string

	svc := newTestService(
		nil, nil, nil,
		&stubIamService{
			revokeSessionFn: func(_ context.Context, sid string) error {
				revokedSessionID = sid
				return nil
			},
		},
		&stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.RevokeSession(context.Background(), auth, "other-session")
	if err != nil {
		t.Fatalf("RevokeSession error: %v", err)
	}
	if revokedSessionID != "other-session" {
		t.Errorf("revoked session = %q, want %q", revokedSessionID, "other-session")
	}
}

// Req 63: RevokeSession returns ErrCannotRevokeCurrent for current session.
func TestRevokeSession_CannotRevokeCurrent(t *testing.T) {
	auth := testAuthWithSession("current-session-id")

	svc := newTestService(
		nil, nil, nil,
		&stubIamService{}, &stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.RevokeSession(context.Background(), auth, "current-session-id")
	if !errors.Is(err, ErrCannotRevokeCurrent) {
		t.Fatalf("want ErrCannotRevokeCurrent, got %v", err)
	}
}

// Req 64: RevokeSession publishes SessionRevoked event.
func TestRevokeSession_PublishesEvent(t *testing.T) {
	auth := testAuthWithSession("current-session-id")

	bus := shared.NewEventBus()
	var publishedEvent *SessionRevoked
	bus.Subscribe(
		eventType[SessionRevoked](),
		handlerFunc(func(_ context.Context, event shared.DomainEvent) error {
			e := event.(SessionRevoked)
			publishedEvent = &e
			return nil
		}),
	)

	svc := NewLifecycleService(
		nil, nil, nil,
		&stubIamService{
			revokeSessionFn: func(_ context.Context, _ string) error { return nil },
		},
		&stubBillingService{},
		bus, &stubJobEnqueuer{}, nil, nil,
	)

	err := svc.RevokeSession(context.Background(), auth, "other-session")
	if err != nil {
		t.Fatalf("RevokeSession error: %v", err)
	}
	if publishedEvent == nil {
		t.Fatal("SessionRevoked event was not published")
	}
	if publishedEvent.SessionID != "other-session" {
		t.Errorf("event.SessionID = %q, want %q", publishedEvent.SessionID, "other-session")
	}
	if publishedEvent.RevokeType != "single" {
		t.Errorf("event.RevokeType = %q, want %q", publishedEvent.RevokeType, "single")
	}
}

// Req 65: RevokeSession returns error when IAM service fails.
func TestRevokeSession_ReturnsErrorOnIAMFailure(t *testing.T) {
	auth := testAuthWithSession("current-session-id")

	svc := newTestService(
		nil, nil, nil,
		&stubIamService{
			revokeSessionFn: func(_ context.Context, _ string) error {
				return errors.New("iam down")
			},
		},
		&stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	err := svc.RevokeSession(context.Background(), auth, "other-session")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// O. Session Management — RevokeAllSessions (Reqs 66–69)
// ═══════════════════════════════════════════════════════════════════════════════

// Req 66: RevokeAllSessions revokes all except current and returns count.
func TestRevokeAllSessions_RevokesAllExceptCurrent(t *testing.T) {
	auth := testAuthWithSession("current-session-id")

	svc := newTestService(
		nil, nil, nil,
		&stubIamService{
			revokeAllSessionsFn: func(_ context.Context, _ uuid.UUID, currentSID string) (uint32, error) {
				if currentSID != "current-session-id" {
					t.Errorf("currentSessionID = %q, want %q", currentSID, "current-session-id")
				}
				return 3, nil
			},
		},
		&stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	count, err := svc.RevokeAllSessions(context.Background(), auth)
	if err != nil {
		t.Fatalf("RevokeAllSessions error: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

// Req 67: RevokeAllSessions returns 0 when only current session exists.
func TestRevokeAllSessions_ReturnsZeroWhenOnlyCurrent(t *testing.T) {
	auth := testAuthWithSession("current-session-id")

	svc := newTestService(
		nil, nil, nil,
		&stubIamService{
			revokeAllSessionsFn: func(_ context.Context, _ uuid.UUID, _ string) (uint32, error) {
				return 0, nil
			},
		},
		&stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	count, err := svc.RevokeAllSessions(context.Background(), auth)
	if err != nil {
		t.Fatalf("RevokeAllSessions error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

// Req 68: RevokeAllSessions publishes SessionRevoked event with RevokeType "all".
func TestRevokeAllSessions_PublishesEvent(t *testing.T) {
	auth := testAuthWithSession("current-session-id")

	bus := shared.NewEventBus()
	var publishedEvent *SessionRevoked
	bus.Subscribe(
		eventType[SessionRevoked](),
		handlerFunc(func(_ context.Context, event shared.DomainEvent) error {
			e := event.(SessionRevoked)
			publishedEvent = &e
			return nil
		}),
	)

	svc := NewLifecycleService(
		nil, nil, nil,
		&stubIamService{
			revokeAllSessionsFn: func(_ context.Context, _ uuid.UUID, _ string) (uint32, error) {
				return 2, nil
			},
		},
		&stubBillingService{},
		bus, &stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RevokeAllSessions(context.Background(), auth)
	if err != nil {
		t.Fatalf("RevokeAllSessions error: %v", err)
	}
	if publishedEvent == nil {
		t.Fatal("SessionRevoked event was not published")
	}
	if publishedEvent.RevokeType != "all" {
		t.Errorf("event.RevokeType = %q, want %q", publishedEvent.RevokeType, "all")
	}
}

// Req 69: RevokeAllSessions returns error when IAM service fails.
func TestRevokeAllSessions_ReturnsErrorOnIAMFailure(t *testing.T) {
	auth := testAuthWithSession("current-session-id")

	svc := newTestService(
		nil, nil, nil,
		&stubIamService{
			revokeAllSessionsFn: func(_ context.Context, _ uuid.UUID, _ string) (uint32, error) {
				return 0, errors.New("iam down")
			},
		},
		&stubBillingService{},
		&stubJobEnqueuer{}, nil, nil,
	)

	_, err := svc.RevokeAllSessions(context.Background(), auth)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// P. Domain Events — Structure (Reqs 70–75)
// ═══════════════════════════════════════════════════════════════════════════════

// Req 70: DataExportRequested implements DomainEvent with correct EventName.
func TestDataExportRequested_EventName(t *testing.T) {
	e := DataExportRequested{}
	if got := e.EventName(); got != "lifecycle.data_export_requested" {
		t.Errorf("EventName = %q, want %q", got, "lifecycle.data_export_requested")
	}
}

// Req 71: DataExportCompleted implements DomainEvent with correct EventName.
func TestDataExportCompleted_EventName(t *testing.T) {
	e := DataExportCompleted{}
	if got := e.EventName(); got != "lifecycle.data_export_completed" {
		t.Errorf("EventName = %q, want %q", got, "lifecycle.data_export_completed")
	}
}

// Req 72: AccountDeletionRequested implements DomainEvent with correct EventName.
func TestAccountDeletionRequested_EventName(t *testing.T) {
	e := AccountDeletionRequested{}
	if got := e.EventName(); got != "lifecycle.account_deletion_requested" {
		t.Errorf("EventName = %q, want %q", got, "lifecycle.account_deletion_requested")
	}
}

// Req 73: AccountDeletionCompleted implements DomainEvent with correct EventName.
func TestAccountDeletionCompleted_EventName(t *testing.T) {
	e := AccountDeletionCompleted{}
	if got := e.EventName(); got != "lifecycle.account_deletion_completed" {
		t.Errorf("EventName = %q, want %q", got, "lifecycle.account_deletion_completed")
	}
}

// Req 74: CoppaDeleteRequested implements DomainEvent with correct EventName.
func TestCoppaDeleteRequested_EventName(t *testing.T) {
	e := CoppaDeleteRequested{}
	if got := e.EventName(); got != "lifecycle.coppa_delete_requested" {
		t.Errorf("EventName = %q, want %q", got, "lifecycle.coppa_delete_requested")
	}
}

// Req 75: SessionRevoked implements DomainEvent with correct EventName.
func TestSessionRevoked_EventName(t *testing.T) {
	e := SessionRevoked{}
	if got := e.EventName(); got != "lifecycle.session_revoked" {
		t.Errorf("EventName = %q, want %q", got, "lifecycle.session_revoked")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Q. Grace Period Computation (Reqs 76–78)
// ═══════════════════════════════════════════════════════════════════════════════

// Req 76: GracePeriodFor family = 30 days.
func TestGracePeriodFor_Family(t *testing.T) {
	expected := 30 * 24 * time.Hour
	if got := GracePeriodFor(DeletionTypeFamily); got != expected {
		t.Errorf("GracePeriodFor(family) = %v, want %v", got, expected)
	}
}

// Req 77: GracePeriodFor student = 7 days.
func TestGracePeriodFor_Student(t *testing.T) {
	expected := 7 * 24 * time.Hour
	if got := GracePeriodFor(DeletionTypeStudent); got != expected {
		t.Errorf("GracePeriodFor(student) = %v, want %v", got, expected)
	}
}

// Req 78: GracePeriodFor COPPA = 0 days.
func TestGracePeriodFor_Coppa(t *testing.T) {
	if got := GracePeriodFor(DeletionTypeCoppa); got != 0 {
		t.Errorf("GracePeriodFor(coppa) = %v, want 0", got)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test Utilities (event subscription helpers)
// ═══════════════════════════════════════════════════════════════════════════════

// eventType returns the reflect.Type for a generic event type.
func eventType[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// handlerFunc adapts a function to the DomainEventHandler interface.
type handlerFunc func(ctx context.Context, event shared.DomainEvent) error

func (f handlerFunc) Handle(ctx context.Context, event shared.DomainEvent) error {
	return f(ctx, event)
}
