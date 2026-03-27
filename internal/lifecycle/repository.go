package lifecycle

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Export Request Repository [15-data-lifecycle §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgExportRequestRepository implements ExportRequestRepository using GORM.
type PgExportRequestRepository struct {
	db *gorm.DB
}

// NewPgExportRequestRepository creates a new PgExportRequestRepository.
func NewPgExportRequestRepository(db *gorm.DB) ExportRequestRepository {
	return &PgExportRequestRepository{db: db}
}

func (r *PgExportRequestRepository) Create(ctx context.Context, scope *shared.FamilyScope, input *CreateExportRequest) (*ExportRequest, error) {
	row := exportRequestRow{
		FamilyID:       scope.FamilyID(),
		RequestedBy:    input.RequestedBy,
		Format:         string(input.Format),
		IncludeDomains: textArray(input.IncludeDomains),
		Status:         string(ExportStatusPending),
	}
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	result := row.toDomain()
	return &result, nil
}

func (r *PgExportRequestRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ExportRequest, error) {
	var row exportRequestRow
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).First(&row).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrExportNotFound
		}
		return nil, err
	}
	result := row.toDomain()
	return &result, nil
}

func (r *PgExportRequestRepository) ListByFamily(ctx context.Context, scope *shared.FamilyScope, pagination *PaginationParams) ([]ExportRequest, int64, error) {
	var rows []exportRequestRow
	var total int64

	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		if err := tx.Model(&exportRequestRow{}).Count(&total).Error; err != nil {
			return err
		}
		return tx.Order("created_at DESC").
			Offset(int(pagination.Offset)).
			Limit(int(pagination.Limit)).
			Find(&rows).Error
	})
	if err != nil {
		return nil, 0, err
	}

	exports := make([]ExportRequest, len(rows))
	for i, row := range rows {
		exports[i] = row.toDomain()
	}
	return exports, total, nil
}

func (r *PgExportRequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status ExportStatus, archiveKey *string, sizeBytes *int64, errorMessage *string) error {
	updates := map[string]any{"status": string(status)}
	if archiveKey != nil {
		updates["archive_key"] = *archiveKey
	}
	if sizeBytes != nil {
		updates["size_bytes"] = *sizeBytes
	}
	if errorMessage != nil {
		updates["error_message"] = *errorMessage
	}
	if status == ExportStatusCompleted {
		updates["completed_at"] = gorm.Expr("now()")
	}
	return shared.BypassRLSTransaction(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Model(&exportRequestRow{}).Where("id = ?", id).Updates(updates).Error
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Deletion Request Repository [15-data-lifecycle §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgDeletionRequestRepository implements DeletionRequestRepository using GORM.
type PgDeletionRequestRepository struct {
	db *gorm.DB
}

// NewPgDeletionRequestRepository creates a new PgDeletionRequestRepository.
func NewPgDeletionRequestRepository(db *gorm.DB) DeletionRequestRepository {
	return &PgDeletionRequestRepository{db: db}
}

func (r *PgDeletionRequestRepository) Create(ctx context.Context, scope *shared.FamilyScope, input *CreateDeletionRequest) (*DeletionRequest, error) {
	row := deletionRequestRow{
		FamilyID:           scope.FamilyID(),
		RequestedBy:        input.RequestedBy,
		DeletionType:       string(input.DeletionType),
		StudentID:          input.StudentID,
		Reason:             input.Reason,
		GracePeriodEndsAt:  input.GracePeriodEndsAt,
		Status:             string(input.Status),
		DomainStatus:       domainStatusMap{},
	}
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	result := row.toDomain()
	return &result, nil
}

func (r *PgDeletionRequestRepository) FindActiveByFamily(ctx context.Context, scope *shared.FamilyScope) (*DeletionRequest, error) {
	var row deletionRequestRow
	err := shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Where("status NOT IN ('completed', 'cancelled')").
			Order("created_at DESC").
			First(&row).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrDeletionNotFound
		}
		return nil, err
	}
	result := row.toDomain()
	return &result, nil
}

func (r *PgDeletionRequestRepository) FindByID(ctx context.Context, id uuid.UUID) (*DeletionRequest, error) {
	var row deletionRequestRow
	err := shared.BypassRLSTransaction(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).First(&row).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrDeletionNotFound
		}
		return nil, err
	}
	result := row.toDomain()
	return &result, nil
}

func (r *PgDeletionRequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status DeletionStatus) error {
	updates := map[string]any{"status": string(status)}
	if status == DeletionStatusCompleted {
		updates["completed_at"] = gorm.Expr("now()")
	}
	return shared.BypassRLSTransaction(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Model(&deletionRequestRow{}).Where("id = ?", id).Updates(updates).Error
	})
}

func (r *PgDeletionRequestRepository) UpdateDomainStatus(ctx context.Context, id uuid.UUID, domain string, completed bool) error {
	raw, _ := json.Marshal(completed)
	return shared.BypassRLSTransaction(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Exec(
			`UPDATE lifecycle_deletion_requests SET domain_status = domain_status || jsonb_build_object($1::text, $2::boolean) WHERE id = $3`,
			domain, string(raw), id,
		).Error
	})
}

func (r *PgDeletionRequestRepository) Cancel(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error {
	return shared.ScopedTransaction(ctx, r.db, *scope, func(tx *gorm.DB) error {
		return tx.Model(&deletionRequestRow{}).Where("id = ?", id).Updates(map[string]any{
			"status":       string(DeletionStatusCancelled),
			"cancelled_at": gorm.Expr("now()"),
		}).Error
	})
}

func (r *PgDeletionRequestRepository) FindReadyForDeletion(ctx context.Context) ([]DeletionRequest, error) {
	var rows []deletionRequestRow
	err := shared.BypassRLSTransaction(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Where(
			"(status = 'grace_period' AND grace_period_ends_at <= now()) OR status = 'processing'",
		).Find(&rows).Error
	})
	if err != nil {
		return nil, err
	}
	results := make([]DeletionRequest, len(rows))
	for i, row := range rows {
		results[i] = row.toDomain()
	}
	return results, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Recovery Request Repository [15-data-lifecycle §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgRecoveryRequestRepository implements RecoveryRequestRepository using GORM.
// All operations use BypassRLSTransaction — recovery is pre-auth with no family scope.
type PgRecoveryRequestRepository struct {
	db *gorm.DB
}

// NewPgRecoveryRequestRepository creates a new PgRecoveryRequestRepository.
func NewPgRecoveryRequestRepository(db *gorm.DB) RecoveryRequestRepository {
	return &PgRecoveryRequestRepository{db: db}
}

func (r *PgRecoveryRequestRepository) Create(ctx context.Context, input *CreateRecoveryRequest) (*RecoveryRequest, error) {
	row := recoveryRequestRow{
		Email:              input.Email,
		VerificationMethod: string(input.VerificationMethod),
		Status:             string(RecoveryStatusPending),
	}
	err := shared.BypassRLSTransaction(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	result := row.toDomain()
	return &result, nil
}

func (r *PgRecoveryRequestRepository) FindByID(ctx context.Context, id uuid.UUID) (*RecoveryRequest, error) {
	var row recoveryRequestRow
	err := shared.BypassRLSTransaction(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).First(&row).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecoveryNotFound
		}
		return nil, err
	}
	result := row.toDomain()
	return &result, nil
}

func (r *PgRecoveryRequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status RecoveryStatus, resolvedParentID *uuid.UUID) error {
	updates := map[string]any{"status": string(status)}
	if resolvedParentID != nil {
		updates["resolved_parent_id"] = *resolvedParentID
		updates["resolved_at"] = gorm.Expr("now()")
	}
	return shared.BypassRLSTransaction(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Model(&recoveryRequestRow{}).Where("id = ?", id).Updates(updates).Error
	})
}
