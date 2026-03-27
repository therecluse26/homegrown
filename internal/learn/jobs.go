package learn

import (
	"context"
	"log/slog"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// SnapshotProgressPayload is the asynq job payload for the weekly progress snapshot job.
// The payload is empty — the handler iterates all active students itself. [06-learn §12.3]
type SnapshotProgressPayload struct{}

func (SnapshotProgressPayload) TaskType() string { return "learn:snapshot_progress" }

// RegisterLearnWorkers registers asynq task handlers for learn background jobs.
// Called from main.go during worker setup. [06-learn §12.3]
func RegisterLearnWorkers(worker shared.JobWorker, svc LearningService) {
	worker.Handle("learn:snapshot_progress", func(ctx context.Context, _ []byte) error {
		if err := svc.SnapshotProgress(ctx); err != nil {
			slog.Error("learn: snapshot_progress job failed", "error", err)
			return err
		}
		slog.Info("learn: snapshot_progress completed")
		return nil
	})
}
