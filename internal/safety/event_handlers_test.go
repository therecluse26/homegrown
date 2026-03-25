package safety

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/media"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social"
)

// ─── UploadQuarantinedHandler ──────────────────────────────────────────────────

func TestUploadQuarantinedHandler_DelegatesToHandleCsamDetection(t *testing.T) {
	// [11-safety §14] — quarantined uploads trigger CSAM pipeline.
	var calledUploadID, calledFamilyID uuid.UUID
	svc := &mockSafetyService{
		handleCsamDetectionFn: func(_ context.Context, uploadID, familyID uuid.UUID, result *CsamScanResult) error {
			calledUploadID = uploadID
			calledFamilyID = familyID
			if !result.IsCSAM {
				t.Error("expected IsCSAM=true")
			}
			return nil
		},
	}

	h := NewUploadQuarantinedHandler(svc)
	uploadID := uuid.Must(uuid.NewV7())
	familyID := uuid.Must(uuid.NewV7())

	err := h.Handle(context.Background(), media.UploadQuarantined{
		UploadID: uploadID,
		FamilyID: familyID,
		Context:  "journal_image",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledUploadID != uploadID {
		t.Errorf("uploadID = %v, want %v", calledUploadID, uploadID)
	}
	if calledFamilyID != familyID {
		t.Errorf("familyID = %v, want %v", calledFamilyID, familyID)
	}
}

func TestUploadQuarantinedHandler_WrongEventType(t *testing.T) {
	h := NewUploadQuarantinedHandler(&mockSafetyService{})
	err := h.Handle(context.Background(), media.UploadRejected{})
	if err == nil {
		t.Fatal("expected error for wrong event type")
	}
}

// ─── UploadRejectedHandler ─────────────────────────────────────────────────────

func TestUploadRejectedHandler_CreatesAutoRejectedFlag(t *testing.T) {
	// [11-safety §14] — rejected uploads create auto-rejected content flags.
	flagRepo := newMockFlagRepo()
	events := newMockEventBus()

	var createdFlag CreateContentFlagRow
	flagRepo.createFn = func(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
		createdFlag = input
		return &ContentFlag{ID: uuid.Must(uuid.NewV7())}, nil
	}

	h := NewUploadRejectedHandler(flagRepo, events)
	uploadID := uuid.Must(uuid.NewV7())

	err := h.Handle(context.Background(), media.UploadRejected{
		UploadID: uploadID,
		FamilyID: uuid.Must(uuid.NewV7()),
		Labels: []media.ModerationLabel{
			{Name: "Explicit Nudity", Confidence: 95.0},
			{Name: "Nudity", Confidence: 80.0},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdFlag.Source != "automated" {
		t.Errorf("source = %q, want %q", createdFlag.Source, "automated")
	}
	if createdFlag.TargetType != "upload" {
		t.Errorf("target_type = %q, want %q", createdFlag.TargetType, "upload")
	}
	if createdFlag.TargetID != uploadID {
		t.Errorf("target_id = %v, want %v", createdFlag.TargetID, uploadID)
	}
	if createdFlag.FlagType != "explicit_content" {
		t.Errorf("flag_type = %q, want %q", createdFlag.FlagType, "explicit_content")
	}
	if !createdFlag.AutoRejected {
		t.Error("expected auto_rejected=true")
	}
}

func TestUploadRejectedHandler_CalculatesMaxConfidence(t *testing.T) {
	// [11-safety §14] — max confidence from all labels.
	flagRepo := newMockFlagRepo()
	events := newMockEventBus()

	var savedConf float64
	flagRepo.createFn = func(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
		if input.Confidence != nil {
			savedConf = *input.Confidence
		}
		return &ContentFlag{ID: uuid.Must(uuid.NewV7())}, nil
	}

	h := NewUploadRejectedHandler(flagRepo, events)
	err := h.Handle(context.Background(), media.UploadRejected{
		UploadID: uuid.Must(uuid.NewV7()),
		FamilyID: uuid.Must(uuid.NewV7()),
		Labels: []media.ModerationLabel{
			{Name: "Nudity", Confidence: 60.0},
			{Name: "Explicit Nudity", Confidence: 98.5},
			{Name: "Suggestive", Confidence: 85.0},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedConf != 98.5 {
		t.Errorf("max confidence = %f, want 98.5", savedConf)
	}
}

func TestUploadRejectedHandler_PublishesUploadAutoRejectedNotification(t *testing.T) {
	flagRepo := newMockFlagRepo()
	events := newMockEventBus()

	flagRepo.createFn = func(_ context.Context, _ CreateContentFlagRow) (*ContentFlag, error) {
		return &ContentFlag{ID: uuid.Must(uuid.NewV7())}, nil
	}

	h := NewUploadRejectedHandler(flagRepo, events)
	familyID := uuid.Must(uuid.NewV7())
	uploadID := uuid.Must(uuid.NewV7())

	err := h.Handle(context.Background(), media.UploadRejected{
		UploadID: uploadID,
		FamilyID: familyID,
		Labels:   []media.ModerationLabel{{Name: "Nudity", Confidence: 90.0}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events.published) != 1 {
		t.Fatalf("published %d events, want 1", len(events.published))
	}
	evt, ok := events.published[0].(UploadAutoRejectedNotification)
	if !ok {
		t.Fatalf("event type = %T, want UploadAutoRejectedNotification", events.published[0])
	}
	if evt.FamilyID != familyID {
		t.Errorf("family_id = %v, want %v", evt.FamilyID, familyID)
	}
	if evt.UploadID != uploadID {
		t.Errorf("upload_id = %v, want %v", evt.UploadID, uploadID)
	}
}

// ─── UploadFlaggedHandler ──────────────────────────────────────────────────────

func TestUploadFlaggedHandler_CriticalPriority(t *testing.T) {
	// [11-safety §14] — critical priority → suspected_underage_exploitation.
	flagRepo := newMockFlagRepo()
	var createdFlag CreateContentFlagRow
	flagRepo.createFn = func(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
		createdFlag = input
		return &ContentFlag{ID: uuid.Must(uuid.NewV7())}, nil
	}

	h := NewUploadFlaggedHandler(flagRepo)
	critical := "critical"
	err := h.Handle(context.Background(), media.UploadFlagged{
		UploadID: uuid.Must(uuid.NewV7()),
		FamilyID: uuid.Must(uuid.NewV7()),
		Priority: &critical,
		Labels:   []media.ModerationLabel{{Name: "Suggestive", Confidence: 85.0}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdFlag.FlagType != "suspected_underage_exploitation" {
		t.Errorf("flag_type = %q, want %q", createdFlag.FlagType, "suspected_underage_exploitation")
	}
}

func TestUploadFlaggedHandler_NonCriticalPriority(t *testing.T) {
	// [11-safety §14] — non-critical priority → explicit_content.
	flagRepo := newMockFlagRepo()
	var createdFlag CreateContentFlagRow
	flagRepo.createFn = func(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
		createdFlag = input
		return &ContentFlag{ID: uuid.Must(uuid.NewV7())}, nil
	}

	h := NewUploadFlaggedHandler(flagRepo)
	normal := "normal"
	err := h.Handle(context.Background(), media.UploadFlagged{
		UploadID: uuid.Must(uuid.NewV7()),
		FamilyID: uuid.Must(uuid.NewV7()),
		Priority: &normal,
		Labels:   []media.ModerationLabel{{Name: "Violence", Confidence: 75.0}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdFlag.FlagType != "explicit_content" {
		t.Errorf("flag_type = %q, want %q", createdFlag.FlagType, "explicit_content")
	}
}

// ─── PostCreatedHandler ────────────────────────────────────────────────────────

func TestPostCreatedHandler_ScansTextAndCreatesFlag(t *testing.T) {
	// [11-safety §14] — scans post text, creates flag on violation.
	flagRepo := newMockFlagRepo()
	svc := &mockSafetyService{
		scanTextFn: func(_ context.Context, _ string) (*TextScanResult, error) {
			return &TextScanResult{HasViolations: true, Severity: "high"}, nil
		},
	}
	var createdFlag CreateContentFlagRow
	flagRepo.createFn = func(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
		createdFlag = input
		return &ContentFlag{ID: uuid.Must(uuid.NewV7())}, nil
	}

	h := NewPostCreatedHandler(svc, flagRepo)
	content := "some harmful content"
	postID := uuid.Must(uuid.NewV7())

	err := h.Handle(context.Background(), social.PostCreated{
		PostID:   postID,
		FamilyID: uuid.Must(uuid.NewV7()),
		Content:  &content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdFlag.TargetType != "post" {
		t.Errorf("target_type = %q, want %q", createdFlag.TargetType, "post")
	}
	if createdFlag.TargetID != postID {
		t.Errorf("target_id = %v, want %v", createdFlag.TargetID, postID)
	}
	if createdFlag.FlagType != "harassment" {
		t.Errorf("flag_type = %q, want %q (from high severity)", createdFlag.FlagType, "harassment")
	}
}

func TestPostCreatedHandler_NilContentSkipsScan(t *testing.T) {
	// [11-safety §14] — nil content skips scan entirely.
	svc := &mockSafetyService{
		scanTextFn: func(_ context.Context, _ string) (*TextScanResult, error) {
			t.Fatal("ScanText should not be called for nil content")
			return nil, nil
		},
	}
	h := NewPostCreatedHandler(svc, newMockFlagRepo())
	err := h.Handle(context.Background(), social.PostCreated{
		PostID:  uuid.Must(uuid.NewV7()),
		Content: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── ReviewCreatedHandler ──────────────────────────────────────────────────────

func TestReviewCreatedHandler_ScansTextAndCreatesFlag(t *testing.T) {
	// [11-safety §14] — scans review text, creates flag on violation.
	flagRepo := newMockFlagRepo()
	svc := &mockSafetyService{
		scanTextFn: func(_ context.Context, _ string) (*TextScanResult, error) {
			return &TextScanResult{HasViolations: true, Severity: "critical"}, nil
		},
	}
	var createdFlag CreateContentFlagRow
	flagRepo.createFn = func(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
		createdFlag = input
		return &ContentFlag{ID: uuid.Must(uuid.NewV7())}, nil
	}

	h := NewReviewCreatedHandler(svc, flagRepo)
	text := "review with violations"
	reviewID := uuid.Must(uuid.NewV7())

	err := h.Handle(context.Background(), mkt.ReviewCreated{
		ReviewID:   reviewID,
		ReviewText: &text,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdFlag.TargetType != "review" {
		t.Errorf("target_type = %q, want %q", createdFlag.TargetType, "review")
	}
	if createdFlag.TargetID != reviewID {
		t.Errorf("target_id = %v, want %v", createdFlag.TargetID, reviewID)
	}
	if createdFlag.FlagType != "csam" {
		t.Errorf("flag_type = %q, want %q (from critical severity)", createdFlag.FlagType, "csam")
	}
}

// ─── MessageReportedHandler ──────────────────────────────────────────────────

func TestMessageReportedHandler_CreatesFlag(t *testing.T) {
	// [11-safety §16.4] — message reports create community_report content flags.
	flagRepo := newMockFlagRepo()
	var createdFlag CreateContentFlagRow
	flagRepo.createFn = func(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
		createdFlag = input
		return &ContentFlag{ID: uuid.Must(uuid.NewV7())}, nil
	}

	h := NewMessageReportedHandler(flagRepo)
	messageID := uuid.Must(uuid.NewV7())

	err := h.Handle(context.Background(), social.MessageReported{
		MessageID:        messageID,
		ConversationID:   uuid.Must(uuid.NewV7()),
		ReporterFamilyID: uuid.Must(uuid.NewV7()),
		SenderParentID:   uuid.Must(uuid.NewV7()),
		Reason:           "harassment",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdFlag.Source != "community_report" {
		t.Errorf("source = %q, want %q", createdFlag.Source, "community_report")
	}
	if createdFlag.TargetType != "message" {
		t.Errorf("target_type = %q, want %q", createdFlag.TargetType, "message")
	}
	if createdFlag.TargetID != messageID {
		t.Errorf("target_id = %v, want %v", createdFlag.TargetID, messageID)
	}
	if createdFlag.FlagType != "harassment" {
		t.Errorf("flag_type = %q, want %q", createdFlag.FlagType, "harassment")
	}
}

func TestMessageReportedHandler_SpamReason(t *testing.T) {
	flagRepo := newMockFlagRepo()
	var createdFlag CreateContentFlagRow
	flagRepo.createFn = func(_ context.Context, input CreateContentFlagRow) (*ContentFlag, error) {
		createdFlag = input
		return &ContentFlag{ID: uuid.Must(uuid.NewV7())}, nil
	}

	h := NewMessageReportedHandler(flagRepo)
	err := h.Handle(context.Background(), social.MessageReported{
		MessageID:        uuid.Must(uuid.NewV7()),
		ConversationID:   uuid.Must(uuid.NewV7()),
		ReporterFamilyID: uuid.Must(uuid.NewV7()),
		SenderParentID:   uuid.Must(uuid.NewV7()),
		Reason:           "spam",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdFlag.FlagType != "spam" {
		t.Errorf("flag_type = %q, want %q", createdFlag.FlagType, "spam")
	}
}

func TestMessageReportedHandler_WrongEventType(t *testing.T) {
	h := NewMessageReportedHandler(newMockFlagRepo())
	err := h.Handle(context.Background(), media.UploadRejected{})
	if err == nil {
		t.Fatal("expected error for wrong event type")
	}
}

// ─── flagTypeFromSeverity ──────────────────────────────────────────────────────

func TestFlagTypeFromSeverity(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"critical", "csam"},
		{"high", "harassment"},
		{"low", "prohibited_content"},
		{"none", "prohibited_content"},
	}
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			got := flagTypeFromSeverity(tt.severity)
			if got != tt.want {
				t.Errorf("flagTypeFromSeverity(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

// ─── Compile-time interface checks ─────────────────────────────────────────────

var (
	_ shared.DomainEventHandler = (*UploadQuarantinedHandler)(nil)
	_ shared.DomainEventHandler = (*UploadRejectedHandler)(nil)
	_ shared.DomainEventHandler = (*UploadFlaggedHandler)(nil)
	_ shared.DomainEventHandler = (*PostCreatedHandler)(nil)
	_ shared.DomainEventHandler = (*ReviewCreatedHandler)(nil)
	_ shared.DomainEventHandler = (*MessageReportedHandler)(nil)
)
