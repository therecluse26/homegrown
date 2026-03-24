package media

import (
	"errors"
	"strings"
	"testing"
)

// ─── getContextRules Tests ────────────────────────────────────────────────────

func TestGetContextRules_all_contexts_have_rules(t *testing.T) {
	contexts := []UploadContext{
		UploadContextProfilePhoto, UploadContextPostAttachment,
		UploadContextMessageAttachment, UploadContextActivityAttachment,
		UploadContextJournalImage, UploadContextNatureJournalImage,
		UploadContextProjectAttachment, UploadContextReadingCover,
		UploadContextMarketplaceFile, UploadContextListingPreview,
		UploadContextListingThumbnail, UploadContextCreatorLogo,
		UploadContextDataExport, UploadContextAudioAttachment,
		UploadContextVideoLesson,
	}

	for _, ctx := range contexts {
		rules := getContextRules(ctx)
		if rules == nil {
			t.Errorf("context %q has no rules", ctx)
			continue
		}
		if rules.MaxSizeBytes == 0 {
			t.Errorf("context %q has zero MaxSizeBytes", ctx)
		}
		if len(rules.AllowedTypes) == 0 {
			t.Errorf("context %q has no AllowedTypes", ctx)
		}
	}
}

func TestGetContextRules_unknown_context_returns_nil(t *testing.T) {
	rules := getContextRules("nonexistent_context")
	if rules != nil {
		t.Error("expected nil for unknown context")
	}
}

func TestGetContextRules_specific_limits(t *testing.T) {
	tests := []struct {
		ctx         UploadContext
		wantMaxSize uint64
	}{
		{UploadContextProfilePhoto, 5 * mb},
		{UploadContextPostAttachment, 25 * mb},
		{UploadContextMessageAttachment, 10 * mb},
		{UploadContextCreatorLogo, 2 * mb},
		{UploadContextMarketplaceFile, 500 * mb},
		{UploadContextDataExport, 5 * gb},
		{UploadContextVideoLesson, 5 * gb},
	}

	for _, tc := range tests {
		t.Run(string(tc.ctx), func(t *testing.T) {
			rules := getContextRules(tc.ctx)
			if rules.MaxSizeBytes != tc.wantMaxSize {
				t.Errorf("MaxSizeBytes = %d, want %d", rules.MaxSizeBytes, tc.wantMaxSize)
			}
		})
	}
}

func TestGetContextRules_allowed_types(t *testing.T) {
	tests := []struct {
		ctx       UploadContext
		wantType  string
		wantAllow bool
	}{
		{UploadContextProfilePhoto, "image/jpeg", true},
		{UploadContextProfilePhoto, "image/png", true},
		{UploadContextProfilePhoto, "image/webp", true},
		{UploadContextProfilePhoto, "video/mp4", false},
		{UploadContextProfilePhoto, "application/pdf", false},
		{UploadContextPostAttachment, "video/mp4", true},
		{UploadContextPostAttachment, "video/webm", true},
		{UploadContextActivityAttachment, "application/pdf", true},
		{UploadContextActivityAttachment, "video/mp4", false},
		{UploadContextMarketplaceFile, "application/zip", true},
		{UploadContextMarketplaceFile, "image/jpeg", false},
		{UploadContextDataExport, "application/zip", true},
		{UploadContextDataExport, "application/pdf", false},
		{UploadContextAudioAttachment, "audio/mpeg", true},
		{UploadContextAudioAttachment, "audio/flac", true},
		{UploadContextAudioAttachment, "image/jpeg", false},
		{UploadContextVideoLesson, "video/quicktime", true},
		{UploadContextVideoLesson, "video/x-msvideo", true},
		{UploadContextVideoLesson, "audio/mpeg", false},
	}

	for _, tc := range tests {
		t.Run(string(tc.ctx)+"/"+tc.wantType, func(t *testing.T) {
			rules := getContextRules(tc.ctx)
			got := rules.AllowsContentType(tc.wantType)
			if got != tc.wantAllow {
				t.Errorf("AllowsContentType(%q) = %v, want %v", tc.wantType, got, tc.wantAllow)
			}
		})
	}
}

// ─── validateAttachmentRules Tests ────────────────────────────────────────────

func TestValidateAttachmentRules_valid(t *testing.T) {
	err := validateAttachmentRules(UploadContextProfilePhoto, "image/jpeg", 1*mb)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateAttachmentRules_invalid_type(t *testing.T) {
	err := validateAttachmentRules(UploadContextProfilePhoto, "video/mp4", 1*mb)
	if !errors.Is(err, ErrInvalidFileType) {
		t.Errorf("expected ErrInvalidFileType, got %v", err)
	}
}

func TestValidateAttachmentRules_too_large(t *testing.T) {
	err := validateAttachmentRules(UploadContextProfilePhoto, "image/jpeg", 10*mb)
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestValidateAttachmentRules_exact_limit_passes(t *testing.T) {
	err := validateAttachmentRules(UploadContextProfilePhoto, "image/jpeg", 5*mb)
	if err != nil {
		t.Errorf("expected nil at exact limit, got %v", err)
	}
}

func TestValidateAttachmentRules_one_over_limit_fails(t *testing.T) {
	err := validateAttachmentRules(UploadContextProfilePhoto, "image/jpeg", 5*mb+1)
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge at limit+1, got %v", err)
	}
}

func TestValidateAttachmentRules_unknown_context(t *testing.T) {
	err := validateAttachmentRules("unknown", "image/jpeg", 1*mb)
	if !errors.Is(err, ErrInvalidFileType) {
		t.Errorf("expected ErrInvalidFileType for unknown context, got %v", err)
	}
}

// ─── sanitizeFilename Tests ───────────────────────────────────────────────────

func TestSanitizeFilename_basic(t *testing.T) {
	got := sanitizeFilename("photo.jpg")
	if got != "photo.jpg" {
		t.Errorf("got %q, want %q", got, "photo.jpg")
	}
}

func TestSanitizeFilename_strips_path(t *testing.T) {
	got := sanitizeFilename("/home/user/photos/nature.png")
	if got != "nature.png" {
		t.Errorf("got %q, want %q", got, "nature.png")
	}
}

func TestSanitizeFilename_strips_windows_path(t *testing.T) {
	got := sanitizeFilename("C:\\Users\\test\\photo.jpg")
	if got != "photo.jpg" {
		t.Errorf("got %q, want %q", got, "photo.jpg")
	}
}

func TestSanitizeFilename_strips_control_chars(t *testing.T) {
	got := sanitizeFilename("photo\x00\x01\x02.jpg")
	if got != "photo.jpg" {
		t.Errorf("got %q, want %q", got, "photo.jpg")
	}
}

func TestSanitizeFilename_replaces_spaces(t *testing.T) {
	got := sanitizeFilename("my photo file.jpg")
	if got != "my-photo-file.jpg" {
		t.Errorf("got %q, want %q", got, "my-photo-file.jpg")
	}
}

func TestSanitizeFilename_empty_becomes_upload(t *testing.T) {
	got := sanitizeFilename("")
	if got != "upload" {
		t.Errorf("got %q, want %q", got, "upload")
	}
}

func TestSanitizeFilename_dot_becomes_upload(t *testing.T) {
	got := sanitizeFilename(".")
	if got != "upload" {
		t.Errorf("got %q, want %q", got, "upload")
	}
}

func TestSanitizeFilename_dotdot_becomes_upload(t *testing.T) {
	got := sanitizeFilename("..")
	if got != "upload" {
		t.Errorf("got %q, want %q", got, "upload")
	}
}

func TestSanitizeFilename_truncates_to_255_bytes(t *testing.T) {
	long := strings.Repeat("a", 300) + ".jpg"
	got := sanitizeFilename(long)
	if len(got) > 255 {
		t.Errorf("length = %d, want <= 255", len(got))
	}
	if !strings.HasSuffix(got, ".jpg") {
		t.Errorf("should preserve extension, got %q", got)
	}
}

func TestSanitizeFilename_truncation_preserves_extension(t *testing.T) {
	long := strings.Repeat("b", 260) + ".png"
	got := sanitizeFilename(long)
	if len(got) > 255 {
		t.Errorf("length = %d, want <= 255", len(got))
	}
	if !strings.HasSuffix(got, ".png") {
		t.Errorf("extension not preserved: %q", got)
	}
}
