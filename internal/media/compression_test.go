package media

import "testing"

// ─── shouldCompress Tests ─────────────────────────────────────────────────────

func TestShouldCompress_jpeg_under_threshold(t *testing.T) {
	// 1000x1000 image at 1.0 BPP = 1MB — under 1.5 threshold
	if shouldCompress("image/jpeg", 1_000_000, 1000, 1000, 0, "") {
		t.Error("expected false for JPEG at 1.0 BPP")
	}
}

func TestShouldCompress_jpeg_over_threshold(t *testing.T) {
	// 1000x1000 image at 2.0 BPP = 2MB — over 1.5 threshold
	if !shouldCompress("image/jpeg", 2_000_000, 1000, 1000, 0, "") {
		t.Error("expected true for JPEG at 2.0 BPP")
	}
}

func TestShouldCompress_png_under_threshold(t *testing.T) {
	// 1000x1000 at 3.0 BPP — under 4.0 threshold
	if shouldCompress("image/png", 3_000_000, 1000, 1000, 0, "") {
		t.Error("expected false for PNG at 3.0 BPP")
	}
}

func TestShouldCompress_png_over_threshold(t *testing.T) {
	// 1000x1000 at 5.0 BPP — over 4.0 threshold
	if !shouldCompress("image/png", 5_000_000, 1000, 1000, 0, "") {
		t.Error("expected true for PNG at 5.0 BPP")
	}
}

func TestShouldCompress_webp_over_threshold(t *testing.T) {
	// 1000x1000 at 1.5 BPP — over 1.0 threshold
	if !shouldCompress("image/webp", 1_500_000, 1000, 1000, 0, "") {
		t.Error("expected true for WebP at 1.5 BPP")
	}
}

func TestShouldCompress_gif_under_5mb(t *testing.T) {
	if shouldCompress("image/gif", 4_000_000, 500, 500, 0, "") {
		t.Error("expected false for GIF under 5MB")
	}
}

func TestShouldCompress_gif_over_5mb(t *testing.T) {
	if !shouldCompress("image/gif", 6_000_000, 500, 500, 0, "") {
		t.Error("expected true for GIF over 5MB")
	}
}

func TestShouldCompress_pdf_never(t *testing.T) {
	if shouldCompress("application/pdf", 100_000_000, 0, 0, 0, "") {
		t.Error("expected false for PDF (never compress)")
	}
}

func TestShouldCompress_zip_never(t *testing.T) {
	if shouldCompress("application/zip", 500_000_000, 0, 0, 0, "") {
		t.Error("expected false for ZIP (never compress)")
	}
}

func TestShouldCompress_video_non_av1_always(t *testing.T) {
	if !shouldCompress("video/mp4", 10_000_000, 1920, 1080, 1_000_000, "h264") {
		t.Error("expected true for non-AV1 video (always transcode)")
	}
}

func TestShouldCompress_video_av1_1080p_under_threshold(t *testing.T) {
	if shouldCompress("video/mp4", 50_000_000, 1920, 1080, 3_000_000, "av1") {
		t.Error("expected false for AV1 1080p at 3 Mbps (under 4 Mbps)")
	}
}

func TestShouldCompress_video_av1_1080p_over_threshold(t *testing.T) {
	if !shouldCompress("video/mp4", 100_000_000, 1920, 1080, 5_000_000, "av1") {
		t.Error("expected true for AV1 1080p at 5 Mbps (over 4 Mbps)")
	}
}

func TestShouldCompress_video_av1_720p_under_threshold(t *testing.T) {
	if shouldCompress("video/mp4", 30_000_000, 1280, 720, 1_500_000, "av1") {
		t.Error("expected false for AV1 720p at 1.5 Mbps (under 2 Mbps)")
	}
}

func TestShouldCompress_video_av1_720p_over_threshold(t *testing.T) {
	if !shouldCompress("video/mp4", 50_000_000, 1280, 720, 3_000_000, "av1") {
		t.Error("expected true for AV1 720p at 3 Mbps (over 2 Mbps)")
	}
}

func TestShouldCompress_audio_wav_always(t *testing.T) {
	if !shouldCompress("audio/wav", 50_000_000, 0, 0, 1_411_200, "") {
		t.Error("expected true for WAV (uncompressed)")
	}
}

func TestShouldCompress_audio_flac_always(t *testing.T) {
	if !shouldCompress("audio/flac", 30_000_000, 0, 0, 800_000, "") {
		t.Error("expected true for FLAC (uncompressed)")
	}
}

func TestShouldCompress_audio_aiff_always(t *testing.T) {
	if !shouldCompress("audio/aiff", 40_000_000, 0, 0, 1_411_200, "") {
		t.Error("expected true for AIFF (uncompressed)")
	}
}

func TestShouldCompress_audio_mp3_under_threshold(t *testing.T) {
	if shouldCompress("audio/mpeg", 5_000_000, 0, 0, 192_000, "") {
		t.Error("expected false for MP3 at 192 kbps (under 256 kbps)")
	}
}

func TestShouldCompress_audio_mp3_over_threshold(t *testing.T) {
	if !shouldCompress("audio/mpeg", 10_000_000, 0, 0, 320_000, "") {
		t.Error("expected true for MP3 at 320 kbps (over 256 kbps)")
	}
}

func TestShouldCompress_image_no_dimensions(t *testing.T) {
	if shouldCompress("image/jpeg", 2_000_000, 0, 0, 0, "") {
		t.Error("expected false when dimensions unavailable")
	}
}

// ─── isImageType Tests ────────────────────────────────────────────────────────

func TestIsImageType_true(t *testing.T) {
	types := []string{"image/jpeg", "image/png", "image/webp", "image/gif"}
	for _, ct := range types {
		if !isImageType(ct) {
			t.Errorf("expected true for %q", ct)
		}
	}
}

func TestIsImageType_false(t *testing.T) {
	types := []string{"video/mp4", "audio/mpeg", "application/pdf"}
	for _, ct := range types {
		if isImageType(ct) {
			t.Errorf("expected false for %q", ct)
		}
	}
}
