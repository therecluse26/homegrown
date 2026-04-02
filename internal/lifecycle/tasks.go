package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// RegisterTaskHandlers registers asynq task handlers for lifecycle background jobs.
// Called from main.go during worker setup. [15-data-lifecycle §14]
func RegisterTaskHandlers(worker shared.JobWorker, svc LifecycleService) {
	worker.Handle("lifecycle:data_export", func(ctx context.Context, payload []byte) error {
		var job DataExportJob
		if err := json.Unmarshal(payload, &job); err != nil {
			return fmt.Errorf("lifecycle: unmarshal data_export payload: %w", err)
		}
		slog.Info("lifecycle: processing data export", "export_id", job.ExportID, "family_id", job.FamilyID)
		if err := svc.ProcessExport(ctx, job.ExportID, job.FamilyID); err != nil {
			slog.Error("lifecycle: data_export job failed", "export_id", job.ExportID, "error", err)
			return err
		}
		slog.Info("lifecycle: data_export completed", "export_id", job.ExportID)
		return nil
	})

	worker.Handle("lifecycle:process_deletion", func(ctx context.Context, payload []byte) error {
		var job ProcessDeletionJob
		if err := json.Unmarshal(payload, &job); err != nil {
			return fmt.Errorf("lifecycle: unmarshal process_deletion payload: %w", err)
		}
		slog.Info("lifecycle: processing deletion", "deletion_id", job.DeletionID, "family_id", job.FamilyID)
		if err := svc.ProcessSingleDeletion(ctx, job.DeletionID, job.FamilyID); err != nil {
			slog.Error("lifecycle: process_deletion job failed", "deletion_id", job.DeletionID, "error", err)
			return err
		}
		slog.Info("lifecycle: process_deletion completed", "deletion_id", job.DeletionID)
		return nil
	})
}
