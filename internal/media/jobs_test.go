package media

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func makeTestUpload() *Upload {
	size := int64(2048576)
	return &Upload{
		ID:               uuid.Must(uuid.NewV7()),
		FamilyID:         uuid.Must(uuid.NewV7()),
		UploadedBy:       uuid.Must(uuid.NewV7()),
		Context:          UploadContextJournalImage,
		Status:           UploadStatusProcessing,
		ContentType:      "image/jpeg",
		OriginalFilename: "photo.jpg",
		StorageKey:       "uploads/fam/upload/photo.jpg",
		SizeBytes:        &size,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// writeTestJPEG creates a minimal JPEG file at the given path for testing.
func writeTestJPEG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := range 100 {
		for x := range 100 {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatal(err)
	}
}

func newRunner() (*processUploadRunner, *mockUploadRepository, *mockProcessingJobRepository, *mockObjectStorageAdapter, *mockSafetyScanAdapter, *mockEventBus) {
	ur := newMockUploadRepository()
	pj := newMockProcessingJobRepository()
	sa := newMockObjectStorageAdapter()
	ss := newMockSafetyScanAdapter()
	eb := newMockEventBus()
	return &processUploadRunner{
		uploads:    ur,
		procJobs:   pj,
		transcodes: newMockTranscodeJobRepository(),
		storage:    sa,
		safety:     ss,
		events:     eb,
	}, ur, pj, sa, ss, eb
}

// setupFFProbeMock sets up mock behavior to handle ffprobe step:
// - DownloadToFile writes a test JPEG
// - Uses a mock ffprobe via newCommand override
func setupFFProbeMock(t *testing.T, sa *mockObjectStorageAdapter) {
	t.Helper()

	// Mock DownloadToFile to write a valid JPEG file
	sa.downloadToFileFn = func(_ context.Context, _ string, filepath string) error {
		writeTestJPEG(t, filepath)
		return nil
	}

	// Override newCommand to return mock ffprobe output
	originalNewCommand := newCommand
	t.Cleanup(func() { newCommand = originalNewCommand })

	newCommand = func(_ context.Context, name string, _ ...string) command {
		if name == "ffprobe" {
			return &mockCommand{output: []byte(`{
				"format": {"format_name": "jpeg_pipe", "bit_rate": "0"},
				"streams": [{"codec_type": "video", "codec_name": "mjpeg", "width": 100, "height": 100}]
			}`)}
		}
		return &mockCommand{output: []byte{}, err: errors.New("unexpected command: " + name)}
	}
}

type mockCommand struct {
	output []byte
	err    error
}

func (m *mockCommand) CombinedOutput() ([]byte, error) { return m.output, m.err }

// ─── Pipeline: Full Success ───────────────────────────────────────────────────

func TestProcessUpload_full_success_image(t *testing.T) {
	runner, ur, _, sa, ss, eb := newRunner()
	upload := makeTestUpload()

	ur.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Upload, error) {
		return upload, nil
	}
	// Magic bytes: valid JPEG
	sa.getObjectBytesFn = func(_ context.Context, _ string, _ uint64, _ uint64) ([]byte, error) {
		return []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01}, nil
	}

	// Set up ffprobe mock
	setupFFProbeMock(t, sa)

	// CSAM scan: clean
	ss.scanCSAMFn = func(_ context.Context, _ string) (*CSAMScanResult, error) {
		return &CSAMScanResult{IsCSAM: false}, nil
	}
	// Moderation: clean
	ss.scanModerationFn = func(_ context.Context, _ string) (*ModerationResult, error) {
		return &ModerationResult{HasViolations: false}, nil
	}

	putCount := 0
	sa.putObjectFn = func(_ context.Context, _ string, _ []byte, _ string) error {
		putCount++
		return nil
	}
	ur.setVariantFlagsFn = func(_ context.Context, _ uuid.UUID, _ bool, _ bool) error {
		return nil
	}
	ur.updateStatusFn = func(_ context.Context, id uuid.UUID, status UploadStatus, _ *UploadStatusUpdate) (*Upload, error) {
		upload.Status = status
		return upload, nil
	}

	err := runner.Run(context.Background(), upload.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if putCount != 2 {
		t.Errorf("expected 2 PutObject calls (thumb + medium), got %d", putCount)
	}

	// Should publish UploadPublished event
	if len(eb.published) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eb.published))
	}
	if eb.published[0].EventName() != "media.UploadPublished" {
		t.Errorf("expected UploadPublished event, got %q", eb.published[0].EventName())
	}
}

// ─── Pipeline: Magic Byte Failure ─────────────────────────────────────────────

func TestProcessUpload_magic_byte_mismatch(t *testing.T) {
	runner, ur, _, sa, _, _ := newRunner()
	upload := makeTestUpload()

	ur.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Upload, error) {
		return upload, nil
	}
	sa.getObjectBytesFn = func(_ context.Context, _ string, _ uint64, _ uint64) ([]byte, error) {
		// Return PNG bytes for a JPEG declared file
		return []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}, nil
	}
	ur.updateStatusFn = func(_ context.Context, _ uuid.UUID, _ UploadStatus, _ *UploadStatusUpdate) (*Upload, error) {
		return upload, nil
	}

	err := runner.Run(context.Background(), upload.ID)
	if !errors.Is(err, ErrMagicByteMismatch) {
		t.Errorf("expected ErrMagicByteMismatch, got %v", err)
	}
}

// ─── Pipeline: CSAM Detected ──────────────────────────────────────────────────

func TestProcessUpload_csam_quarantine(t *testing.T) {
	runner, ur, _, sa, ss, eb := newRunner()
	upload := makeTestUpload()

	ur.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Upload, error) {
		return upload, nil
	}
	sa.getObjectBytesFn = func(_ context.Context, _ string, _ uint64, _ uint64) ([]byte, error) {
		return []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01}, nil
	}

	setupFFProbeMock(t, sa)

	ss.scanCSAMFn = func(_ context.Context, _ string) (*CSAMScanResult, error) {
		return &CSAMScanResult{IsCSAM: true}, nil
	}
	ur.updateStatusFn = func(_ context.Context, _ uuid.UUID, status UploadStatus, _ *UploadStatusUpdate) (*Upload, error) {
		if status != UploadStatusQuarantined {
			t.Errorf("expected quarantined status, got %q", status)
		}
		return upload, nil
	}
	ss.reportCSAMFn = func(_ context.Context, _ uuid.UUID, _ *CSAMScanResult) error {
		return nil
	}

	err := runner.Run(context.Background(), upload.ID)
	if err != nil {
		t.Fatalf("expected nil (short-circuit), got %v", err)
	}

	if len(eb.published) != 1 || eb.published[0].EventName() != "media.UploadQuarantined" {
		t.Error("expected UploadQuarantined event")
	}
}

// ─── Pipeline: Moderation Auto-Reject ─────────────────────────────────────────

func TestProcessUpload_moderation_auto_reject(t *testing.T) {
	runner, ur, _, sa, ss, eb := newRunner()
	upload := makeTestUpload()

	ur.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Upload, error) {
		return upload, nil
	}
	sa.getObjectBytesFn = func(_ context.Context, _ string, _ uint64, _ uint64) ([]byte, error) {
		return []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01}, nil
	}

	setupFFProbeMock(t, sa)

	ss.scanCSAMFn = func(_ context.Context, _ string) (*CSAMScanResult, error) {
		return &CSAMScanResult{IsCSAM: false}, nil
	}
	ss.scanModerationFn = func(_ context.Context, _ string) (*ModerationResult, error) {
		return &ModerationResult{
			HasViolations: true,
			AutoReject:    true,
			Labels:        []ModerationLabel{{Name: "Explicit Nudity", Confidence: 99.5}},
		}, nil
	}
	ur.setModerationLabelsFn = func(_ context.Context, _ uuid.UUID, _ json.RawMessage) error {
		return nil
	}
	ur.updateStatusFn = func(_ context.Context, _ uuid.UUID, status UploadStatus, _ *UploadStatusUpdate) (*Upload, error) {
		if status != UploadStatusRejected {
			t.Errorf("expected rejected status, got %q", status)
		}
		return upload, nil
	}

	err := runner.Run(context.Background(), upload.ID)
	if err != nil {
		t.Fatalf("expected nil (short-circuit), got %v", err)
	}

	if len(eb.published) != 1 || eb.published[0].EventName() != "media.UploadRejected" {
		t.Error("expected UploadRejected event")
	}
}

// ─── Pipeline: Moderation Flag ────────────────────────────────────────────────

func TestProcessUpload_moderation_flag(t *testing.T) {
	runner, ur, _, sa, ss, eb := newRunner()
	upload := makeTestUpload()

	ur.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Upload, error) {
		return upload, nil
	}
	sa.getObjectBytesFn = func(_ context.Context, _ string, _ uint64, _ uint64) ([]byte, error) {
		return []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01}, nil
	}

	setupFFProbeMock(t, sa)

	ss.scanCSAMFn = func(_ context.Context, _ string) (*CSAMScanResult, error) {
		return &CSAMScanResult{IsCSAM: false}, nil
	}
	priority := "high"
	ss.scanModerationFn = func(_ context.Context, _ string) (*ModerationResult, error) {
		return &ModerationResult{
			HasViolations: true,
			AutoReject:    false,
			Labels:        []ModerationLabel{{Name: "Suggestive", Confidence: 85.0}},
			Priority:      &priority,
		}, nil
	}
	ur.setModerationLabelsFn = func(_ context.Context, _ uuid.UUID, _ json.RawMessage) error {
		return nil
	}
	ur.updateStatusFn = func(_ context.Context, _ uuid.UUID, status UploadStatus, _ *UploadStatusUpdate) (*Upload, error) {
		if status != UploadStatusFlagged {
			t.Errorf("expected flagged status, got %q", status)
		}
		return upload, nil
	}

	err := runner.Run(context.Background(), upload.ID)
	if err != nil {
		t.Fatalf("expected nil (short-circuit), got %v", err)
	}

	if len(eb.published) != 1 || eb.published[0].EventName() != "media.UploadFlagged" {
		t.Error("expected UploadFlagged event")
	}
}

// ─── Pipeline: Scan Unavailable Graceful Degradation ──────────────────────────

func TestProcessUpload_csam_scan_unavailable_continues(t *testing.T) {
	runner, ur, _, sa, ss, eb := newRunner()
	upload := makeTestUpload()

	ur.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Upload, error) {
		return upload, nil
	}
	sa.getObjectBytesFn = func(_ context.Context, _ string, _ uint64, _ uint64) ([]byte, error) {
		return []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01}, nil
	}

	setupFFProbeMock(t, sa)

	ss.scanCSAMFn = func(_ context.Context, _ string) (*CSAMScanResult, error) {
		return nil, ErrScanUnavailable
	}
	ss.scanModerationFn = func(_ context.Context, _ string) (*ModerationResult, error) {
		return &ModerationResult{HasViolations: false}, nil
	}
	sa.putObjectFn = func(_ context.Context, _ string, _ []byte, _ string) error { return nil }
	ur.setVariantFlagsFn = func(_ context.Context, _ uuid.UUID, _ bool, _ bool) error { return nil }
	ur.updateStatusFn = func(_ context.Context, _ uuid.UUID, _ UploadStatus, _ *UploadStatusUpdate) (*Upload, error) {
		return upload, nil
	}

	err := runner.Run(context.Background(), upload.ID)
	if err != nil {
		t.Fatalf("expected pipeline to continue despite scan unavailability, got %v", err)
	}

	// Should still publish
	if len(eb.published) != 1 || eb.published[0].EventName() != "media.UploadPublished" {
		t.Error("expected UploadPublished event even with scan unavailable")
	}
}

func TestProcessUpload_moderation_unavailable_continues(t *testing.T) {
	runner, ur, _, sa, ss, eb := newRunner()
	upload := makeTestUpload()

	ur.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Upload, error) {
		return upload, nil
	}
	sa.getObjectBytesFn = func(_ context.Context, _ string, _ uint64, _ uint64) ([]byte, error) {
		return []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01}, nil
	}

	setupFFProbeMock(t, sa)

	ss.scanCSAMFn = func(_ context.Context, _ string) (*CSAMScanResult, error) {
		return &CSAMScanResult{IsCSAM: false}, nil
	}
	ss.scanModerationFn = func(_ context.Context, _ string) (*ModerationResult, error) {
		return nil, ErrScanUnavailable
	}
	sa.putObjectFn = func(_ context.Context, _ string, _ []byte, _ string) error { return nil }
	ur.setVariantFlagsFn = func(_ context.Context, _ uuid.UUID, _ bool, _ bool) error { return nil }
	ur.updateStatusFn = func(_ context.Context, _ uuid.UUID, _ UploadStatus, _ *UploadStatusUpdate) (*Upload, error) {
		return upload, nil
	}

	err := runner.Run(context.Background(), upload.ID)
	if err != nil {
		t.Fatalf("expected pipeline to continue, got %v", err)
	}
	if len(eb.published) != 1 || eb.published[0].EventName() != "media.UploadPublished" {
		t.Error("expected UploadPublished event")
	}
}

// ─── Pipeline: Non-Image Skips Variants ───────────────────────────────────────

func TestProcessUpload_non_image_skips_variants(t *testing.T) {
	runner, ur, _, sa, ss, eb := newRunner()
	upload := makeTestUpload()
	upload.ContentType = "application/pdf"

	ur.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Upload, error) {
		return upload, nil
	}
	sa.getObjectBytesFn = func(_ context.Context, _ string, _ uint64, _ uint64) ([]byte, error) {
		return []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34, 0x0A, 0x25, 0xC4, 0xE5, 0xF2, 0xE5, 0xEB, 0xA7}, nil
	}

	setupFFProbeMock(t, sa)

	ss.scanCSAMFn = func(_ context.Context, _ string) (*CSAMScanResult, error) {
		return &CSAMScanResult{IsCSAM: false}, nil
	}
	ss.scanModerationFn = func(_ context.Context, _ string) (*ModerationResult, error) {
		return &ModerationResult{HasViolations: false}, nil
	}
	putCalled := false
	sa.putObjectFn = func(_ context.Context, _ string, _ []byte, _ string) error {
		putCalled = true
		return nil
	}
	ur.updateStatusFn = func(_ context.Context, _ uuid.UUID, _ UploadStatus, _ *UploadStatusUpdate) (*Upload, error) {
		return upload, nil
	}

	err := runner.Run(context.Background(), upload.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if putCalled {
		t.Error("expected no PutObject calls for non-image files")
	}
	if len(eb.published) != 1 || eb.published[0].EventName() != "media.UploadPublished" {
		t.Error("expected UploadPublished event")
	}
}

// ─── Pipeline: Upload Not Found ───────────────────────────────────────────────

func TestProcessUpload_upload_not_found(t *testing.T) {
	runner, ur, _, _, _, _ := newRunner()

	ur.findByIDUnscopedFn = func(_ context.Context, _ uuid.UUID) (*Upload, error) {
		return nil, &MediaError{Err: ErrUploadNotFound}
	}

	err := runner.Run(context.Background(), uuid.Must(uuid.NewV7()))
	if err == nil {
		t.Fatal("expected error")
	}
}

// ─── Orphan Cleanup ───────────────────────────────────────────────────────────

func TestRunCleanup_cleans_expired(t *testing.T) {
	ur := newMockUploadRepository()
	sa := newMockObjectStorageAdapter()

	expired := time.Now().Add(-2 * time.Hour)
	orphan := Upload{
		ID:         uuid.Must(uuid.NewV7()),
		StorageKey: "uploads/test/orphan",
		ExpiresAt:  &expired,
		Status:     UploadStatusPending,
	}

	ur.findExpiredPendingFn = func(_ context.Context, _ time.Time, _ uint32) ([]Upload, error) {
		return []Upload{orphan}, nil
	}
	sa.deleteObjectFn = func(_ context.Context, _ string) error { return nil }
	ur.updateStatusFn = func(_ context.Context, _ uuid.UUID, status UploadStatus, _ *UploadStatusUpdate) (*Upload, error) {
		if status != UploadStatusExpired {
			t.Errorf("expected expired status, got %q", status)
		}
		return &orphan, nil
	}

	count, err := RunCleanup(context.Background(), ur, sa)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 cleaned, got %d", count)
	}
}

func TestRunCleanup_no_orphans(t *testing.T) {
	ur := newMockUploadRepository()
	sa := newMockObjectStorageAdapter()

	ur.findExpiredPendingFn = func(_ context.Context, _ time.Time, _ uint32) ([]Upload, error) {
		return nil, nil
	}

	count, err := RunCleanup(context.Background(), ur, sa)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 cleaned, got %d", count)
	}
}

// ─── Job Payload TaskType ─────────────────────────────────────────────────────

func TestProcessUploadPayload_TaskType(t *testing.T) {
	p := ProcessUploadPayload{}
	if p.TaskType() != "media:process_upload" {
		t.Errorf("TaskType() = %q, want %q", p.TaskType(), "media:process_upload")
	}
}

func TestTranscodeVideoPayload_TaskType(t *testing.T) {
	p := TranscodeVideoPayload{}
	if p.TaskType() != "media:transcode_video" {
		t.Errorf("TaskType() = %q, want %q", p.TaskType(), "media:transcode_video")
	}
}

func TestCleanupOrphansPayload_TaskType(t *testing.T) {
	p := CleanupOrphansPayload{}
	if p.TaskType() != "media:cleanup_orphans" {
		t.Errorf("TaskType() = %q, want %q", p.TaskType(), "media:cleanup_orphans")
	}
}

func TestCompressAssetPayload_TaskType(t *testing.T) {
	p := CompressAssetPayload{}
	if p.TaskType() != "media:compress_asset" {
		t.Errorf("TaskType() = %q, want %q", p.TaskType(), "media:compress_asset")
	}
}
