package sources_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/homegrown-academy/homegrown-academy/cmd/seed-content/sources"
)

func TestRunCKFoundation_DryRun_CountMatchesTarget(t *testing.T) {
	cfg := sources.Config{DryRun: true, Target: 5}
	n, err := sources.RunCKFoundation(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunCKFoundation dry-run: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 dry-run listings, got %d", n)
	}
}

func TestRunCKFoundation_DryRun_NoTargetReturnsFull(t *testing.T) {
	cfg := sources.Config{DryRun: true, Target: 0}
	n, err := sources.RunCKFoundation(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunCKFoundation dry-run: %v", err)
	}
	// Manifest has ≥ 80 entries.
	if n < 80 {
		t.Errorf("expected ≥ 80 manifest entries, dry-run returned %d", n)
	}
}

func TestRunCKFoundation_NilStorage_ReturnsZero(t *testing.T) {
	cfg := sources.Config{DryRun: false, Storage: nil}
	n, err := sources.RunCKFoundation(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error with nil storage: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 with nil storage, got %d", n)
	}
}

func TestLoadCKManifestHasExpectedCount(t *testing.T) {
	// Verify the embedded manifest parses and meets the 80-listing target.
	cfg := sources.Config{DryRun: true, Target: 0}
	n, err := sources.RunCKFoundation(context.Background(), cfg)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if n < 80 {
		t.Errorf("manifest must have ≥ 80 entries for HOM-45, got %d", n)
	}
}

func TestRunCKFoundation_UploadFailureIsSkipped(t *testing.T) {
	// Server returns 404 — upload should be skipped, no fatal error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// Use a real uploader pointing at the test server — but we need to inject it.
	// Since RunCKFoundation uses cfg.Storage.Upload and then skips on error,
	// we test that path by using a mock that always errors.
	errUploader := &errAlwaysUploader{}
	cfg := sources.Config{
		DryRun:  false,
		Target:  3,
		Storage: errUploader,
		// DB is nil; we won't reach DB insert because all uploads fail.
	}

	n, err := sources.RunCKFoundation(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	// All uploads failed → nothing inserted.
	if n != 0 {
		t.Errorf("expected 0 inserted when all uploads fail, got %d", n)
	}
}

// errAlwaysUploader is a sources.Uploader that always returns an error.
type errAlwaysUploader struct{}

func (e *errAlwaysUploader) Upload(_ context.Context, _, _ string) (string, int64, error) {
	return "", 0, &uploadTestError{}
}

type uploadTestError struct{}

func (e *uploadTestError) Error() string { return "simulated upload failure" }
