package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// ─── Queue Inspector Types ──────────────────────────────────────────────────

// QueueInfo represents the status of a single job queue.
type QueueInfo struct {
	Name         string
	Pending      int64
	Processing   int64
	Completed24h int64
	Failed24h    int64
}

// DeadLetterJobInfo represents a job in the dead-letter (archived) queue.
type DeadLetterJobInfo struct {
	ID           string
	Queue        string
	JobType      string
	Payload      json.RawMessage
	ErrorMessage string
	FailedAt     time.Time
	RetryCount   int32
}

// QueueInspector provides read-only inspection of background job queues. [16-admin §11.2]
type QueueInspector interface {
	GetQueues(ctx context.Context) ([]QueueInfo, int64, error) // queues + total archived count
	GetDeadLetterJobs(ctx context.Context, offset, limit int) ([]DeadLetterJobInfo, error)
	RetryDeadLetterJob(ctx context.Context, queue, jobID string) error
}

// ─── Asynq Implementation ───────────────────────────────────────────────────

// AsynqQueueInspector wraps asynq.Inspector. [16-admin §11.2]
type AsynqQueueInspector struct {
	inspector *asynq.Inspector
}

// NewAsynqQueueInspector creates a QueueInspector backed by asynq.
func NewAsynqQueueInspector(redisURL string) (*AsynqQueueInspector, error) {
	redisOpt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL for queue inspector: %w", err)
	}
	return &AsynqQueueInspector{
		inspector: asynq.NewInspector(redisOpt),
	}, nil
}

func (a *AsynqQueueInspector) GetQueues(_ context.Context) ([]QueueInfo, int64, error) {
	qnames, err := a.inspector.Queues()
	if err != nil {
		return nil, 0, fmt.Errorf("list queues: %w", err)
	}

	var totalArchived int64
	queues := make([]QueueInfo, 0, len(qnames))
	for _, qname := range qnames {
		info, qErr := a.inspector.GetQueueInfo(qname)
		if qErr != nil {
			slog.Warn("skipping inaccessible queue", "queue", qname, "error", qErr)
			continue
		}
		totalArchived += int64(info.Archived)
		queues = append(queues, QueueInfo{
			Name:         qname,
			Pending:      int64(info.Pending),
			Processing:   int64(info.Active),
			Completed24h: int64(info.Processed),
			Failed24h:    int64(info.Failed),
		})
	}
	return queues, totalArchived, nil
}

func (a *AsynqQueueInspector) GetDeadLetterJobs(_ context.Context, offset, limit int) ([]DeadLetterJobInfo, error) {
	qnames, err := a.inspector.Queues()
	if err != nil {
		return nil, fmt.Errorf("list queues: %w", err)
	}

	var allJobs []DeadLetterJobInfo
	for _, qname := range qnames {
		tasks, tErr := a.inspector.ListArchivedTasks(qname, asynq.PageSize(limit))
		if tErr != nil {
			slog.Warn("skipping dead-letter listing for queue", "queue", qname, "error", tErr)
			continue
		}
		for _, t := range tasks {
			allJobs = append(allJobs, DeadLetterJobInfo{
				ID:           t.ID,
				Queue:        t.Queue,
				JobType:      t.Type,
				Payload:      json.RawMessage(t.Payload),
				ErrorMessage: t.LastErr,
				FailedAt:     t.LastFailedAt,
				RetryCount:   int32(t.Retried),
			})
		}
	}

	// Apply offset/limit across all queues.
	if offset >= len(allJobs) {
		return []DeadLetterJobInfo{}, nil
	}
	end := min(offset+limit, len(allJobs))
	return allJobs[offset:end], nil
}

func (a *AsynqQueueInspector) RetryDeadLetterJob(_ context.Context, queue, jobID string) error {
	if err := a.inspector.RunTask(queue, jobID); err != nil {
		return fmt.Errorf("retry dead letter job %s/%s: %w", queue, jobID, err)
	}
	return nil
}
