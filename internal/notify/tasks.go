package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Task type constants for background job routing. [08-notify §14]
const (
	TaskTypeSendEmail    = "notify:send_email"
	TaskTypeCompileDigest = "notify:compile_digest"
)

// SendEmailTaskPayload is the background task payload for transactional email delivery.
// Implements shared.JobPayload. [CODING §8.1b]
type SendEmailTaskPayload struct {
	To             string         `json:"to"`
	TemplateAlias  string         `json:"template_alias"`
	TemplateModel  map[string]any `json:"template_model"`
	IdempotencyKey string         `json:"idempotency_key"`
}

func (SendEmailTaskPayload) TaskType() string { return TaskTypeSendEmail }

// CompileDigestPayload is the background task payload for digest compilation (Phase 2 stub).
type CompileDigestPayload struct {
	DigestType string `json:"digest_type"`
}

func (CompileDigestPayload) TaskType() string { return TaskTypeCompileDigest }

// Ensure task payloads implement JobPayload at compile time.
var (
	_ shared.JobPayload = SendEmailTaskPayload{}
	_ shared.JobPayload = CompileDigestPayload{}
)

// RegisterTaskHandlers registers notify:: background task handlers with the job worker.
func RegisterTaskHandlers(worker shared.JobWorker, adapter EmailAdapter) {
	worker.Handle(TaskTypeSendEmail, handleSendEmailTask(adapter))
}

// handleSendEmailTask returns a JobHandler for email delivery tasks.
func handleSendEmailTask(adapter EmailAdapter) shared.JobHandler {
	return func(ctx context.Context, payload []byte) error {
		var task SendEmailTaskPayload
		if err := json.Unmarshal(payload, &task); err != nil {
			return fmt.Errorf("unmarshalling send email task: %w", err)
		}
		if err := adapter.SendTransactional(ctx, task.To, task.TemplateAlias, task.TemplateModel); err != nil {
			slog.Error("email delivery failed",
				"idempotency_key", task.IdempotencyKey,
				"template", task.TemplateAlias,
				"error", err,
			)
			return err
		}
		return nil
	}
}
