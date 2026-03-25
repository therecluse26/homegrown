package safety

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/homegrown-academy/homegrown-academy/internal/media"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social"
)

// ─── media.UploadQuarantined → HandleCsamDetection ───────────────────────────

// UploadQuarantinedHandler handles media.UploadQuarantined events. [11-safety §14]
type UploadQuarantinedHandler struct{ svc SafetyService }

func NewUploadQuarantinedHandler(svc SafetyService) *UploadQuarantinedHandler {
	return &UploadQuarantinedHandler{svc: svc}
}

func (h *UploadQuarantinedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(media.UploadQuarantined)
	if !ok {
		return fmt.Errorf("safety.UploadQuarantinedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleCsamDetection(ctx, e.UploadID, e.FamilyID, &CsamScanResult{
		IsCSAM: true,
	})
}

// ─── media.UploadRejected → auto-rejected flag ──────────────────────────────

// UploadRejectedHandler handles media.UploadRejected events. [11-safety §14]
type UploadRejectedHandler struct {
	flagRepo ContentFlagRepository
	events   eventPublisher
}

func NewUploadRejectedHandler(flagRepo ContentFlagRepository, events eventPublisher) *UploadRejectedHandler {
	return &UploadRejectedHandler{flagRepo: flagRepo, events: events}
}

func (h *UploadRejectedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(media.UploadRejected)
	if !ok {
		return fmt.Errorf("safety.UploadRejectedHandler: unexpected event type %T", event)
	}

	// Calculate max confidence from labels.
	var maxConf float64
	for _, l := range e.Labels {
		if l.Confidence > maxConf {
			maxConf = l.Confidence
		}
	}

	// Convert labels to safety domain type.
	safetyLabels := convertMediaLabels(e.Labels)

	if _, err := h.flagRepo.Create(ctx, CreateContentFlagRow{
		Source:       "automated",
		TargetType:   "upload",
		TargetID:     e.UploadID,
		FlagType:     "explicit_content",
		Confidence:   &maxConf,
		Labels:       marshalLabels(safetyLabels),
		AutoRejected: true,
	}); err != nil {
		return fmt.Errorf("create auto-rejected flag: %w", err)
	}

	// Publish upload auto-rejected notification for notify:: [11-safety §16.3]
	_ = h.events.Publish(ctx, UploadAutoRejectedNotification{
		FamilyID: e.FamilyID,
		UploadID: e.UploadID,
	})

	return nil
}

// ─── media.UploadFlagged → content flag for admin review ────────────────────

// UploadFlaggedHandler handles media.UploadFlagged events. [11-safety §14]
type UploadFlaggedHandler struct {
	flagRepo ContentFlagRepository
}

func NewUploadFlaggedHandler(flagRepo ContentFlagRepository) *UploadFlaggedHandler {
	return &UploadFlaggedHandler{flagRepo: flagRepo}
}

func (h *UploadFlaggedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(media.UploadFlagged)
	if !ok {
		return fmt.Errorf("safety.UploadFlaggedHandler: unexpected event type %T", event)
	}

	// Determine flag type from priority.
	flagType := "explicit_content"
	if e.Priority != nil && *e.Priority == "critical" {
		flagType = "suspected_underage_exploitation"
	}

	// Calculate max confidence.
	var maxConf float64
	for _, l := range e.Labels {
		if l.Confidence > maxConf {
			maxConf = l.Confidence
		}
	}

	safetyLabels := convertMediaLabels(e.Labels)

	if _, err := h.flagRepo.Create(ctx, CreateContentFlagRow{
		Source:     "automated",
		TargetType: "upload",
		TargetID:   e.UploadID,
		FlagType:   flagType,
		Confidence: &maxConf,
		Labels:     marshalLabels(safetyLabels),
	}); err != nil {
		return fmt.Errorf("create flagged content: %w", err)
	}

	return nil
}

// ─── social.PostCreated → text scan ─────────────────────────────────────────

// PostCreatedHandler handles social.PostCreated events. [11-safety §14]
type PostCreatedHandler struct {
	svc      SafetyService
	flagRepo ContentFlagRepository
}

func NewPostCreatedHandler(svc SafetyService, flagRepo ContentFlagRepository) *PostCreatedHandler {
	return &PostCreatedHandler{svc: svc, flagRepo: flagRepo}
}

func (h *PostCreatedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(social.PostCreated)
	if !ok {
		return fmt.Errorf("safety.PostCreatedHandler: unexpected event type %T", event)
	}

	if e.Content == nil {
		return nil
	}

	result, err := h.svc.ScanText(ctx, *e.Content)
	if err != nil {
		slog.Error("text scan failed for post", "post_id", e.PostID, "error", err)
		return nil
	}

	if result.HasViolations {
		if _, err := h.flagRepo.Create(ctx, CreateContentFlagRow{
			Source:     "automated",
			TargetType: "post",
			TargetID:   e.PostID,
			FlagType:   flagTypeFromSeverity(result.Severity),
		}); err != nil {
			return fmt.Errorf("create text flag: %w", err)
		}
	}

	return nil
}

// ─── mkt.ReviewCreated → text scan ──────────────────────────────────────────

// ReviewCreatedHandler handles mkt.ReviewCreated events. [11-safety §14]
type ReviewCreatedHandler struct {
	svc      SafetyService
	flagRepo ContentFlagRepository
}

func NewReviewCreatedHandler(svc SafetyService, flagRepo ContentFlagRepository) *ReviewCreatedHandler {
	return &ReviewCreatedHandler{svc: svc, flagRepo: flagRepo}
}

func (h *ReviewCreatedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.ReviewCreated)
	if !ok {
		return fmt.Errorf("safety.ReviewCreatedHandler: unexpected event type %T", event)
	}

	if e.ReviewText == nil {
		return nil
	}

	result, err := h.svc.ScanText(ctx, *e.ReviewText)
	if err != nil {
		slog.Error("text scan failed for review", "review_id", e.ReviewID, "error", err)
		return nil
	}

	if result.HasViolations {
		if _, err := h.flagRepo.Create(ctx, CreateContentFlagRow{
			Source:     "automated",
			TargetType: "review",
			TargetID:   e.ReviewID,
			FlagType:   flagTypeFromSeverity(result.Severity),
		}); err != nil {
			return fmt.Errorf("create text flag: %w", err)
		}
	}

	return nil
}

// ─── social.MessageReported → content flag ───────────────────────────────────

// MessageReportedHandler handles social.MessageReported events. [11-safety §16.4]
type MessageReportedHandler struct {
	flagRepo ContentFlagRepository
}

func NewMessageReportedHandler(flagRepo ContentFlagRepository) *MessageReportedHandler {
	return &MessageReportedHandler{flagRepo: flagRepo}
}

func (h *MessageReportedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(social.MessageReported)
	if !ok {
		return fmt.Errorf("safety.MessageReportedHandler: unexpected event type %T", event)
	}

	flagType := "harassment"
	if e.Reason == "spam" {
		flagType = "spam"
	}

	if _, err := h.flagRepo.Create(ctx, CreateContentFlagRow{
		Source:     "community_report",
		TargetType: "message",
		TargetID:   e.MessageID,
		FlagType:   flagType,
	}); err != nil {
		return fmt.Errorf("create message flag: %w", err)
	}

	return nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func flagTypeFromSeverity(severity string) string {
	switch severity {
	case "critical":
		return "csam"
	case "high":
		return "harassment"
	default:
		return "prohibited_content"
	}
}

func convertMediaLabels(labels []media.ModerationLabel) []ModerationLabel {
	result := make([]ModerationLabel, len(labels))
	for i, l := range labels {
		result[i] = ModerationLabel{
			Name:       l.Name,
			Confidence: l.Confidence,
			ParentName: l.ParentName,
		}
	}
	return result
}

func marshalLabels(labels []ModerationLabel) *json.RawMessage {
	data, err := json.Marshal(labels)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(data)
	return &raw
}

// Ensure compile-time interface satisfaction.
var (
	_ shared.DomainEventHandler = (*UploadQuarantinedHandler)(nil)
	_ shared.DomainEventHandler = (*UploadRejectedHandler)(nil)
	_ shared.DomainEventHandler = (*UploadFlaggedHandler)(nil)
	_ shared.DomainEventHandler = (*PostCreatedHandler)(nil)
	_ shared.DomainEventHandler = (*ReviewCreatedHandler)(nil)
	_ shared.DomainEventHandler = (*MessageReportedHandler)(nil)
)
