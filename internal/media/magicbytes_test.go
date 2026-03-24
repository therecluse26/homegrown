package media

import "testing"

// ─── detectFileType Tests ─────────────────────────────────────────────────────

func TestDetectFileType_jpeg(t *testing.T) {
	header := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46}
	if got := detectFileType(header); got != "image/jpeg" {
		t.Errorf("got %q, want image/jpeg", got)
	}
}

func TestDetectFileType_png(t *testing.T) {
	header := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	if got := detectFileType(header); got != "image/png" {
		t.Errorf("got %q, want image/png", got)
	}
}

func TestDetectFileType_gif(t *testing.T) {
	header := []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x00, 0x00}
	if got := detectFileType(header); got != "image/gif" {
		t.Errorf("got %q, want image/gif", got)
	}
}

func TestDetectFileType_webp(t *testing.T) {
	header := []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}
	if got := detectFileType(header); got != "image/webp" {
		t.Errorf("got %q, want image/webp", got)
	}
}

func TestDetectFileType_pdf(t *testing.T) {
	header := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}
	if got := detectFileType(header); got != "application/pdf" {
		t.Errorf("got %q, want application/pdf", got)
	}
}

func TestDetectFileType_mp4(t *testing.T) {
	// ftyp box: size at 0-3, "ftyp" at 4-7
	header := []byte{0x00, 0x00, 0x00, 0x1C, 0x66, 0x74, 0x79, 0x70, 0x69, 0x73, 0x6F, 0x6D}
	if got := detectFileType(header); got != "video/mp4" {
		t.Errorf("got %q, want video/mp4", got)
	}
}

func TestDetectFileType_mov(t *testing.T) {
	// MOV also uses ftyp box with brand "qt  "
	header := []byte{0x00, 0x00, 0x00, 0x14, 0x66, 0x74, 0x79, 0x70, 0x71, 0x74, 0x20, 0x20}
	if got := detectFileType(header); got != "video/mp4" {
		t.Errorf("got %q, want video/mp4 (ftyp family)", got)
	}
}

func TestDetectFileType_m4a(t *testing.T) {
	// M4A uses ftyp box with brand "M4A "
	header := []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, 0x4D, 0x34, 0x41, 0x20}
	if got := detectFileType(header); got != "video/mp4" {
		t.Errorf("got %q, want video/mp4 (ftyp family)", got)
	}
}

func TestDetectFileType_webm(t *testing.T) {
	header := []byte{0x1A, 0x45, 0xDF, 0xA3, 0x93, 0x42, 0x82, 0x88}
	if got := detectFileType(header); got != "video/webm" {
		t.Errorf("got %q, want video/webm", got)
	}
}

func TestDetectFileType_avi(t *testing.T) {
	header := []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x41, 0x56, 0x49, 0x20}
	if got := detectFileType(header); got != "video/x-msvideo" {
		t.Errorf("got %q, want video/x-msvideo", got)
	}
}

func TestDetectFileType_mp3_sync(t *testing.T) {
	header := []byte{0xFF, 0xFB, 0x90, 0x04, 0x00, 0x00, 0x00, 0x00}
	if got := detectFileType(header); got != "audio/mpeg" {
		t.Errorf("got %q, want audio/mpeg", got)
	}
}

func TestDetectFileType_mp3_id3(t *testing.T) {
	header := []byte{0x49, 0x44, 0x33, 0x04, 0x00, 0x00, 0x00, 0x00}
	if got := detectFileType(header); got != "audio/mpeg" {
		t.Errorf("got %q, want audio/mpeg", got)
	}
}

func TestDetectFileType_wav(t *testing.T) {
	header := []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x41, 0x56, 0x45}
	if got := detectFileType(header); got != "audio/wav" {
		t.Errorf("got %q, want audio/wav", got)
	}
}

func TestDetectFileType_flac(t *testing.T) {
	header := []byte{0x66, 0x4C, 0x61, 0x43, 0x00, 0x00, 0x00, 0x22}
	if got := detectFileType(header); got != "audio/flac" {
		t.Errorf("got %q, want audio/flac", got)
	}
}

func TestDetectFileType_aiff(t *testing.T) {
	header := []byte{0x46, 0x4F, 0x52, 0x4D, 0x00, 0x00, 0x00, 0x00, 0x41, 0x49, 0x46, 0x46}
	if got := detectFileType(header); got != "audio/aiff" {
		t.Errorf("got %q, want audio/aiff", got)
	}
}

func TestDetectFileType_zip(t *testing.T) {
	header := []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00, 0x06, 0x00}
	if got := detectFileType(header); got != "application/zip" {
		t.Errorf("got %q, want application/zip", got)
	}
}

func TestDetectFileType_unknown(t *testing.T) {
	header := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if got := detectFileType(header); got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestDetectFileType_too_short(t *testing.T) {
	header := []byte{0xFF, 0xD8}
	if got := detectFileType(header); got != "" {
		t.Errorf("got %q for 2-byte input, want empty string", got)
	}
}

func TestDetectFileType_empty(t *testing.T) {
	if got := detectFileType(nil); got != "" {
		t.Errorf("got %q for nil input, want empty string", got)
	}
}

// ─── isCompatible Tests ───────────────────────────────────────────────────────

func TestIsCompatible_exact_match(t *testing.T) {
	if !isCompatible("image/jpeg", "image/jpeg") {
		t.Error("expected true for exact match")
	}
}

func TestIsCompatible_mismatch(t *testing.T) {
	if isCompatible("image/jpeg", "image/png") {
		t.Error("expected false for mismatch")
	}
}

func TestIsCompatible_ftyp_mp4(t *testing.T) {
	if !isCompatible("video/mp4", "video/mp4") {
		t.Error("expected true for video/mp4")
	}
}

func TestIsCompatible_ftyp_quicktime(t *testing.T) {
	// MOV declares video/quicktime but detects as video/mp4 (ftyp container)
	if !isCompatible("video/quicktime", "video/mp4") {
		t.Error("expected true for quicktime declared, mp4 detected")
	}
}

func TestIsCompatible_ftyp_audio_mp4(t *testing.T) {
	// M4A declares audio/mp4 but detects as video/mp4 (ftyp container)
	if !isCompatible("audio/mp4", "video/mp4") {
		t.Error("expected true for audio/mp4 declared, mp4 detected")
	}
}

func TestIsCompatible_unknown_detected(t *testing.T) {
	if isCompatible("image/jpeg", "") {
		t.Error("expected false for empty detected type")
	}
}

func TestIsCompatible_non_ftyp_mismatch(t *testing.T) {
	if isCompatible("image/jpeg", "video/mp4") {
		t.Error("expected false for jpeg declared but mp4 detected")
	}
}
