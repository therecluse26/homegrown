package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/homegrown-academy/homegrown-academy/internal/config"
)

// JobPayload is the data passed to a background job. Domains define their own typed
// payload structs and register them with the asynq server. The TaskType method drives
// routing — asynq multiplexes by task type string, not Go type.
//
// Convention: "{domain}:{action}" (e.g., "media:scan_upload", "notify:send_email")
type JobPayload interface {
	// TaskType returns the unique string identifier for this job type.
	TaskType() string
}

// JobEnqueuer is the background-job port. No asynq types are referenced in this
// interface — the concrete implementation (asynqJobEnqueuer) is unexported.
// Domains receive JobEnqueuer via dependency injection; they never import asynq directly.
//
// Jobs are serialized as JSON. Payloads MUST be JSON-marshalable.
type JobEnqueuer interface {
	// Enqueue submits a job for immediate background execution.
	// Idempotency is the responsibility of the job handler, not the enqueuer. [ARCH §10.8]
	Enqueue(ctx context.Context, payload JobPayload) error

	// EnqueueIn submits a job for execution after the given delay.
	EnqueueIn(ctx context.Context, payload JobPayload, delay time.Duration) error

	// Close shuts down the enqueuer client gracefully.
	Close() error
}

// JobHandler processes a background job payload. Domains define typed handlers
// that unmarshal the payload and execute business logic. [ARCH §10.8]
type JobHandler func(ctx context.Context, payload []byte) error

// JobWorker is the background-job consumer. It receives jobs from Redis and
// dispatches them to registered handlers by task type. [ARCH §10.8]
type JobWorker interface {
	// Handle registers a handler for a given task type.
	Handle(taskType string, handler JobHandler)

	// Start begins processing jobs. Blocks until Stop is called or ctx is cancelled.
	Start() error

	// Stop gracefully shuts down the worker.
	Stop()
}

// CreateJobWorker creates a JobWorker backed by asynq (Redis-based).
func CreateJobWorker(cfg *config.AppConfig) (JobWorker, error) {
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL for job worker: %w", err)
	}
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
	})
	return &asynqJobWorker{
		server: srv,
		mux:    asynq.NewServeMux(),
	}, nil
}

type asynqJobWorker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

func (w *asynqJobWorker) Handle(taskType string, handler JobHandler) {
	w.mux.HandleFunc(taskType, func(_ context.Context, t *asynq.Task) error {
		return handler(context.Background(), t.Payload())
	})
}

func (w *asynqJobWorker) Start() error {
	return w.server.Start(w.mux)
}

func (w *asynqJobWorker) Stop() {
	w.server.Stop()
}

// ─── NoopJobEnqueuer ─────────────────────────────────────────────────────────

// NoopJobEnqueuer satisfies JobEnqueuer for tests and environments without Redis.
// All enqueue calls succeed silently and no jobs are actually queued.
type NoopJobEnqueuer struct{}

func (NoopJobEnqueuer) Enqueue(_ context.Context, _ JobPayload) error                   { return nil }
func (NoopJobEnqueuer) EnqueueIn(_ context.Context, _ JobPayload, _ time.Duration) error { return nil }
func (NoopJobEnqueuer) Close() error                                                     { return nil }

// ─── asynqJobEnqueuer ────────────────────────────────────────────────────────
// asynqJobEnqueuer wraps github.com/hibiken/asynq. This is the only file in the
// codebase where the asynq package is permitted to be imported. [ARCH §4.3]

type asynqJobEnqueuer struct {
	client *asynq.Client
}

// CreateJobEnqueuer creates a JobEnqueuer backed by asynq (Redis-based).
// Uses the same RedisURL as the Cache — asynq requires no additional infrastructure.
func CreateJobEnqueuer(cfg *config.AppConfig) (JobEnqueuer, error) {
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL for job enqueuer: %w", err)
	}
	client := asynq.NewClient(redisOpt)
	return &asynqJobEnqueuer{client: client}, nil
}

func (e *asynqJobEnqueuer) Enqueue(ctx context.Context, payload JobPayload) error {
	task, err := marshalTask(payload)
	if err != nil {
		return err
	}
	if _, err := e.client.EnqueueContext(ctx, task); err != nil {
		return ErrInternal(fmt.Errorf("job enqueue (%s): %w", payload.TaskType(), err))
	}
	return nil
}

func (e *asynqJobEnqueuer) EnqueueIn(ctx context.Context, payload JobPayload, delay time.Duration) error {
	task, err := marshalTask(payload)
	if err != nil {
		return err
	}
	if _, err := e.client.EnqueueContext(ctx, task, asynq.ProcessIn(delay)); err != nil {
		return ErrInternal(fmt.Errorf("job enqueue-in (%s): %w", payload.TaskType(), err))
	}
	return nil
}

func (e *asynqJobEnqueuer) Close() error {
	return e.client.Close()
}

// marshalTask serializes a JobPayload to an asynq.Task.
// Isolated here so both Enqueue paths share the same serialization logic.
func marshalTask(payload JobPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, ErrInternal(fmt.Errorf("job marshal (%s): %w", payload.TaskType(), err))
	}
	return asynq.NewTask(payload.TaskType(), data), nil
}
