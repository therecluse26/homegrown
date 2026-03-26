package lifecycle

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Implementation [15-data-lifecycle §5]
// ═══════════════════════════════════════════════════════════════════════════════

// LifecycleServiceImpl implements LifecycleService.
type LifecycleServiceImpl struct {
	exportRepo       ExportRequestRepository
	deletionRepo     DeletionRequestRepository
	recoveryRepo     RecoveryRequestRepository
	iamSvc           IamServiceForLifecycle
	billingSvc       BillingServiceForLifecycle
	events           *shared.EventBus
	jobs             shared.JobEnqueuer
	exportHandlers   []ExportHandler
	deletionHandlers []DeletionHandler
}

// NewLifecycleService creates a LifecycleService backed by the provided dependencies.
func NewLifecycleService(
	exportRepo ExportRequestRepository,
	deletionRepo DeletionRequestRepository,
	recoveryRepo RecoveryRequestRepository,
	iamSvc IamServiceForLifecycle,
	billingSvc BillingServiceForLifecycle,
	events *shared.EventBus,
	jobs shared.JobEnqueuer,
	exportHandlers []ExportHandler,
	deletionHandlers []DeletionHandler,
) LifecycleService {
	return &LifecycleServiceImpl{
		exportRepo:       exportRepo,
		deletionRepo:     deletionRepo,
		recoveryRepo:     recoveryRepo,
		iamSvc:           iamSvc,
		billingSvc:       billingSvc,
		events:           events,
		jobs:             jobs,
		exportHandlers:   exportHandlers,
		deletionHandlers: deletionHandlers,
	}
}

// ─── Data Export ──────────────────────────────────────────────────────────────

// RequestExport creates an export record and enqueues a background job. [15-data-lifecycle §9]
func (s *LifecycleServiceImpl) RequestExport(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, req *RequestExportInput) (uuid.UUID, error) {
	format := ExportFormatJSON
	if req.Format != nil {
		format = *req.Format
	}

	input := &CreateExportRequest{
		RequestedBy:    auth.ParentID,
		Format:         format,
		IncludeDomains: req.IncludeDomains,
	}

	export, err := s.exportRepo.Create(ctx, scope, input)
	if err != nil {
		return uuid.Nil, fmt.Errorf("lifecycle: create export request: %w", err)
	}

	if err := s.jobs.Enqueue(ctx, DataExportJob{
		ExportID: export.ID,
		FamilyID: export.FamilyID,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("lifecycle: enqueue export job: %w", err)
	}

	return export.ID, nil
}

// GetExportStatus returns the current status of an export request. [15-data-lifecycle §9]
func (s *LifecycleServiceImpl) GetExportStatus(ctx context.Context, scope *shared.FamilyScope, exportID uuid.UUID) (*ExportStatusResponse, error) {
	export, err := s.exportRepo.FindByID(ctx, scope, exportID)
	if err != nil {
		return nil, err
	}

	// Check expiry.
	if time.Now().UTC().After(export.ExpiresAt) {
		return nil, ErrExportExpired
	}

	return &ExportStatusResponse{
		ID:          export.ID,
		Status:      export.Status,
		Format:      export.Format,
		SizeBytes:   export.SizeBytes,
		DownloadURL: export.DownloadURL,
		CreatedAt:   export.CreatedAt,
		CompletedAt: export.CompletedAt,
		ExpiresAt:   export.ExpiresAt,
	}, nil
}

// ListExports returns paginated export summaries for the family. [15-data-lifecycle §9]
func (s *LifecycleServiceImpl) ListExports(ctx context.Context, scope *shared.FamilyScope, pagination *PaginationParams) (*PaginatedExports, error) {
	exports, err := s.exportRepo.ListByFamily(ctx, scope, pagination)
	if err != nil {
		return nil, fmt.Errorf("lifecycle: list exports: %w", err)
	}

	items := make([]ExportSummary, len(exports))
	for i, e := range exports {
		items[i] = ExportSummary{
			ID:        e.ID,
			Status:    e.Status,
			Format:    e.Format,
			SizeBytes: e.SizeBytes,
			CreatedAt: e.CreatedAt,
		}
	}

	return &PaginatedExports{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

// ProcessExport executes the cross-domain data export pipeline. [15-data-lifecycle §9.1]
// Called by the background job worker — not by HTTP handlers directly.
func (s *LifecycleServiceImpl) ProcessExport(ctx context.Context, exportID uuid.UUID, familyID uuid.UUID) error {
	// Build a FamilyScope from the family ID for repo calls.
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})

	export, err := s.exportRepo.FindByID(ctx, &scope, exportID)
	if err != nil {
		return fmt.Errorf("lifecycle: process export find: %w", err)
	}

	// Transition to processing.
	if err := s.exportRepo.UpdateStatus(ctx, exportID, ExportStatusProcessing, nil, nil); err != nil {
		return fmt.Errorf("lifecycle: process export set processing: %w", err)
	}

	// Determine which handlers to call.
	handlers := s.filterExportHandlers(export.IncludeDomains)

	// Call each handler, collecting files.
	var totalSize int64
	for _, h := range handlers {
		files, handlerErr := h.ExportFamilyData(ctx, familyID, export.Format)
		if handlerErr != nil {
			// Mark as failed.
			_ = s.exportRepo.UpdateStatus(ctx, exportID, ExportStatusFailed, nil, nil)
			return fmt.Errorf("lifecycle: export handler %q failed: %w", h.DomainName(), handlerErr)
		}
		for _, f := range files {
			totalSize += int64(len(f.Content))
		}
	}

	// Build archive key.
	archiveKey := fmt.Sprintf("exports/%s/%s.zip", familyID, exportID)

	// Transition to completed.
	if err := s.exportRepo.UpdateStatus(ctx, exportID, ExportStatusCompleted, &archiveKey, &totalSize); err != nil {
		return fmt.Errorf("lifecycle: process export set completed: %w", err)
	}

	// Publish completion event (errors logged, not propagated). [ARCH §11.3]
	_ = s.events.Publish(ctx, DataExportCompleted{
		FamilyID: familyID,
		ExportID: exportID,
	})

	return nil
}

// filterExportHandlers returns handlers matching the include list, or all if nil.
func (s *LifecycleServiceImpl) filterExportHandlers(includeDomains []string) []ExportHandler {
	if includeDomains == nil {
		return s.exportHandlers
	}
	var filtered []ExportHandler
	for _, h := range s.exportHandlers {
		if slices.Contains(includeDomains, h.DomainName()) {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

// ─── Account Deletion ────────────────────────────────────────────────────────

// RequestDeletion creates a deletion request with appropriate grace period. [15-data-lifecycle §10]
func (s *LifecycleServiceImpl) RequestDeletion(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, req *RequestDeletionInput) (uuid.UUID, error) {
	// Validate: family deletion requires primary parent.
	if req.DeletionType == DeletionTypeFamily && !auth.IsPrimaryParent {
		return uuid.Nil, ErrNotPrimaryParent
	}

	// Validate: student and COPPA types require StudentID.
	if (req.DeletionType == DeletionTypeStudent || req.DeletionType == DeletionTypeCoppa) && req.StudentID == nil {
		return uuid.Nil, shared.ErrValidation("student_id is required for student and COPPA deletion")
	}

	// Check for existing active deletion request.
	existing, err := s.deletionRepo.FindActiveByFamily(ctx, scope)
	if err != nil && err != ErrDeletionNotFound {
		return uuid.Nil, fmt.Errorf("lifecycle: check existing deletion: %w", err)
	}
	if existing != nil {
		return uuid.Nil, ErrDeletionAlreadyPending
	}

	// Compute grace period and initial status.
	gracePeriod := GracePeriodFor(req.DeletionType)
	gracePeriodEndsAt := time.Now().UTC().Add(gracePeriod)

	status := DeletionStatusGracePeriod
	if req.DeletionType == DeletionTypeCoppa {
		status = DeletionStatusProcessing
	}

	input := &CreateDeletionRequest{
		RequestedBy:       auth.ParentID,
		DeletionType:      req.DeletionType,
		StudentID:         req.StudentID,
		Reason:            req.Reason,
		GracePeriodEndsAt: gracePeriodEndsAt,
		Status:            status,
	}

	deletion, err := s.deletionRepo.Create(ctx, scope, input)
	if err != nil {
		return uuid.Nil, fmt.Errorf("lifecycle: create deletion request: %w", err)
	}

	// COPPA: enqueue immediate processing job and publish specific event.
	if req.DeletionType == DeletionTypeCoppa {
		if err := s.jobs.Enqueue(ctx, ProcessDeletionJob{
			DeletionID: deletion.ID,
			FamilyID:   deletion.FamilyID,
		}); err != nil {
			slog.Error("lifecycle: enqueue COPPA deletion job", "error", err)
		}

		_ = s.events.Publish(ctx, CoppaDeleteRequested{
			FamilyID:  deletion.FamilyID,
			StudentID: *req.StudentID,
		})

		return deletion.ID, nil
	}

	// Non-COPPA: publish AccountDeletionRequested event (no immediate job).
	_ = s.events.Publish(ctx, AccountDeletionRequested{
		FamilyID:          deletion.FamilyID,
		DeletionType:      req.DeletionType,
		GracePeriodEndsAt: deletion.GracePeriodEndsAt,
	})

	return deletion.ID, nil
}

// GetDeletionStatus returns the status of the family's active deletion request.
func (s *LifecycleServiceImpl) GetDeletionStatus(ctx context.Context, scope *shared.FamilyScope) (*DeletionStatusResponse, error) {
	deletion, err := s.deletionRepo.FindActiveByFamily(ctx, scope)
	if err != nil {
		return nil, err
	}

	return &DeletionStatusResponse{
		ID:                deletion.ID,
		Status:            deletion.Status,
		DeletionType:      deletion.DeletionType,
		GracePeriodEndsAt: deletion.GracePeriodEndsAt,
		ExportOffered:     deletion.ExportOffered,
		ExportRequestID:   deletion.ExportRequestID,
		CreatedAt:         deletion.CreatedAt,
	}, nil
}

// CancelDeletion cancels a pending deletion during the grace period. [15-data-lifecycle §10.1]
func (s *LifecycleServiceImpl) CancelDeletion(ctx context.Context, scope *shared.FamilyScope) error {
	deletion, err := s.deletionRepo.FindActiveByFamily(ctx, scope)
	if err != nil {
		return err
	}

	// Check if grace period has expired.
	if time.Now().UTC().After(deletion.GracePeriodEndsAt) {
		return ErrGracePeriodExpired
	}

	return s.deletionRepo.Cancel(ctx, scope, deletion.ID)
}

// ProcessDeletion processes deletion requests whose grace period has expired. [15-data-lifecycle §10.1]
// Called by the recurring background job — not by HTTP handlers.
func (s *LifecycleServiceImpl) ProcessDeletion(ctx context.Context) error {
	requests, err := s.deletionRepo.FindReadyForDeletion(ctx)
	if err != nil {
		return fmt.Errorf("lifecycle: find ready for deletion: %w", err)
	}

	for _, req := range requests {
		s.processSingleDeletion(ctx, req)
	}

	return nil
}

// processSingleDeletion handles one deletion request end-to-end.
func (s *LifecycleServiceImpl) processSingleDeletion(ctx context.Context, req DeletionRequest) {
	// Transition to processing.
	if err := s.deletionRepo.UpdateStatus(ctx, req.ID, DeletionStatusProcessing); err != nil {
		slog.Error("lifecycle: set deletion processing", "deletion_id", req.ID, "error", err)
		return
	}

	// Revoke all Kratos sessions for the family.
	if err := s.iamSvc.RevokeFamilySessions(ctx, req.FamilyID); err != nil {
		slog.Error("lifecycle: revoke family sessions", "family_id", req.FamilyID, "error", err)
	}

	// Cancel subscriptions.
	if err := s.billingSvc.CancelFamilySubscriptions(ctx, req.FamilyID); err != nil {
		slog.Error("lifecycle: cancel subscriptions", "family_id", req.FamilyID, "error", err)
	}

	// Call each DeletionHandler.
	allSucceeded := true
	for _, h := range s.deletionHandlers {
		var handlerErr error
		switch req.DeletionType {
		case DeletionTypeStudent, DeletionTypeCoppa:
			if req.StudentID != nil {
				handlerErr = h.DeleteStudentData(ctx, req.FamilyID, *req.StudentID)
			}
		default:
			handlerErr = h.DeleteFamilyData(ctx, req.FamilyID)
		}

		if handlerErr != nil {
			slog.Error("lifecycle: deletion handler failed",
				"domain", h.DomainName(),
				"family_id", req.FamilyID,
				"error", handlerErr,
			)
			allSucceeded = false
			_ = s.deletionRepo.UpdateDomainStatus(ctx, req.ID, h.DomainName(), false)
		} else {
			_ = s.deletionRepo.UpdateDomainStatus(ctx, req.ID, h.DomainName(), true)
		}
	}

	// Transition to completed (even partial — retries handle remaining domains).
	if allSucceeded {
		if err := s.deletionRepo.UpdateStatus(ctx, req.ID, DeletionStatusCompleted); err != nil {
			slog.Error("lifecycle: set deletion completed", "deletion_id", req.ID, "error", err)
		}
		_ = s.events.Publish(ctx, AccountDeletionCompleted{FamilyID: req.FamilyID})
	}
}

// ─── Account Recovery ────────────────────────────────────────────────────────

// InitiateRecovery creates a recovery request and triggers the Kratos flow. [15-data-lifecycle §13]
// Always returns a UUID regardless of whether the email exists (enumeration prevention).
func (s *LifecycleServiceImpl) InitiateRecovery(ctx context.Context, req *InitiateRecoveryInput) (uuid.UUID, error) {
	if req.Email == "" {
		return uuid.Nil, shared.ErrValidation("email is required")
	}

	recovery, err := s.recoveryRepo.Create(ctx, &CreateRecoveryRequest{
		Email:              req.Email,
		VerificationMethod: VerificationMethodEmail,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("lifecycle: create recovery request: %w", err)
	}

	// Trigger Kratos recovery flow — errors are logged, not propagated (enumeration prevention).
	if err := s.iamSvc.InitiateRecoveryFlow(ctx, req.Email); err != nil {
		slog.Error("lifecycle: kratos recovery flow failed", "error", err)
	}

	return recovery.ID, nil
}

// GetRecoveryStatus returns the status of a recovery request. [15-data-lifecycle §13]
func (s *LifecycleServiceImpl) GetRecoveryStatus(ctx context.Context, recoveryID uuid.UUID) (*RecoveryStatusResponse, error) {
	recovery, err := s.recoveryRepo.FindByID(ctx, recoveryID)
	if err != nil {
		return nil, err
	}

	// Check expiry.
	if time.Now().UTC().After(recovery.ExpiresAt) {
		return nil, ErrRecoveryExpired
	}

	return &RecoveryStatusResponse{
		ID:                 recovery.ID,
		Status:             recovery.Status,
		VerificationMethod: recovery.VerificationMethod,
		CreatedAt:          recovery.CreatedAt,
	}, nil
}

// ─── Session Management ──────────────────────────────────────────────────────

// ListSessions returns active sessions, marking the current one. [15-data-lifecycle §12]
func (s *LifecycleServiceImpl) ListSessions(ctx context.Context, auth *shared.AuthContext) ([]SessionInfo, error) {
	sessions, err := s.iamSvc.ListSessions(ctx, auth.ParentID)
	if err != nil {
		return nil, fmt.Errorf("lifecycle: list sessions: %w", err)
	}

	// Mark the current session.
	for i := range sessions {
		sessions[i].IsCurrent = sessions[i].SessionID == auth.SessionID
	}

	return sessions, nil
}

// RevokeSession revokes a specific non-current session. [15-data-lifecycle §12.2]
func (s *LifecycleServiceImpl) RevokeSession(ctx context.Context, auth *shared.AuthContext, sessionID string) error {
	if sessionID == auth.SessionID {
		return ErrCannotRevokeCurrent
	}

	if err := s.iamSvc.RevokeSession(ctx, sessionID); err != nil {
		return fmt.Errorf("lifecycle: revoke session: %w", err)
	}

	_ = s.events.Publish(ctx, SessionRevoked{
		ParentID:   auth.ParentID,
		SessionID:  sessionID,
		RevokeType: "single",
	})

	return nil
}

// RevokeAllSessions revokes all sessions except the current one. [15-data-lifecycle §12.2]
func (s *LifecycleServiceImpl) RevokeAllSessions(ctx context.Context, auth *shared.AuthContext) (uint32, error) {
	count, err := s.iamSvc.RevokeAllSessions(ctx, auth.ParentID, auth.SessionID)
	if err != nil {
		return 0, fmt.Errorf("lifecycle: revoke all sessions: %w", err)
	}

	if count > 0 {
		_ = s.events.Publish(ctx, SessionRevoked{
			ParentID:   auth.ParentID,
			SessionID:  auth.SessionID,
			RevokeType: "all",
		})
	}

	return count, nil
}
