package media

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

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

// ─── parseFFProbeOutput Tests ────────────────────────────────────────────────

func TestParseFFProbeOutput_video(t *testing.T) {
	raw := `{
		"format": {"format_name": "mov,mp4,m4a,3gp,3g2,mj2", "bit_rate": "2500000", "duration": "120.5"},
		"streams": [
			{"codec_type": "video", "codec_name": "h264", "width": 1920, "height": 1080, "bit_rate": "2200000"},
			{"codec_type": "audio", "codec_name": "aac", "bit_rate": "128000", "sample_rate": "44100", "channels": 2, "duration": "120.5"}
		]
	}`

	result, err := parseFFProbeOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Format != "mov,mp4,m4a,3gp,3g2,mj2" {
		t.Errorf("Format = %q, want mov,mp4...", result.Format)
	}
	if result.Width != 1920 || result.Height != 1080 {
		t.Errorf("Dimensions = %dx%d, want 1920x1080", result.Width, result.Height)
	}
	if result.Codec != "h264" {
		t.Errorf("Codec = %q, want h264", result.Codec)
	}
	if result.BitrateBps != 2200000 {
		t.Errorf("BitrateBps = %d, want 2200000", result.BitrateBps)
	}
	if result.DurationSeconds != 120.5 {
		t.Errorf("DurationSeconds = %f, want 120.5", result.DurationSeconds)
	}
	if result.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", result.SampleRate)
	}
	if result.Channels != 2 {
		t.Errorf("Channels = %d, want 2", result.Channels)
	}
}

func TestParseFFProbeOutput_audio_only(t *testing.T) {
	raw := `{
		"format": {"format_name": "mp3", "bit_rate": "320000", "duration": "240.0"},
		"streams": [
			{"codec_type": "audio", "codec_name": "mp3", "bit_rate": "320000", "sample_rate": "48000", "channels": 2, "duration": "240.0"}
		]
	}`

	result, err := parseFFProbeOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Codec != "mp3" {
		t.Errorf("Codec = %q, want mp3", result.Codec)
	}
	if result.Width != 0 || result.Height != 0 {
		t.Errorf("Dimensions = %dx%d, want 0x0", result.Width, result.Height)
	}
	if result.SampleRate != 48000 {
		t.Errorf("SampleRate = %d, want 48000", result.SampleRate)
	}
	if result.DurationSeconds != 240.0 {
		t.Errorf("DurationSeconds = %f, want 240.0", result.DurationSeconds)
	}
}

func TestParseFFProbeOutput_image(t *testing.T) {
	raw := `{
		"format": {"format_name": "image2", "bit_rate": "0"},
		"streams": [
			{"codec_type": "video", "codec_name": "mjpeg", "width": 4000, "height": 3000}
		]
	}`

	result, err := parseFFProbeOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Width != 4000 || result.Height != 3000 {
		t.Errorf("Dimensions = %dx%d, want 4000x3000", result.Width, result.Height)
	}
	if result.Codec != "mjpeg" {
		t.Errorf("Codec = %q, want mjpeg", result.Codec)
	}
}

func TestParseFFProbeOutput_invalid_json(t *testing.T) {
	_, err := parseFFProbeOutput([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseFFProbeOutput_empty_streams(t *testing.T) {
	raw := `{"format": {"format_name": "unknown"}, "streams": []}`

	result, err := parseFFProbeOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != "unknown" {
		t.Errorf("Format = %q, want unknown", result.Format)
	}
	if result.Width != 0 || result.Height != 0 {
		t.Error("expected zero dimensions with no streams")
	}
}

func TestParseFFProbeOutput_video_takes_priority(t *testing.T) {
	// When both video and audio are present, codec should come from video stream
	raw := `{
		"format": {"format_name": "mp4"},
		"streams": [
			{"codec_type": "audio", "codec_name": "aac", "sample_rate": "44100", "channels": 2},
			{"codec_type": "video", "codec_name": "hevc", "width": 1280, "height": 720, "bit_rate": "1500000"}
		]
	}`

	result, err := parseFFProbeOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Codec != "hevc" {
		t.Errorf("Codec = %q, want hevc (video should take priority)", result.Codec)
	}
}

// ─── buildProbeMetadataJSON Tests ────────────────────────────────────────────

func TestBuildProbeMetadataJSON_image(t *testing.T) {
	probe := &ProbeResult{
		Format: "image2",
		Width:  2000,
		Height: 1500,
	}

	data := buildProbeMetadataJSON(probe, false, 5_000_000, nil)

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if m["format"] != "image2" {
		t.Errorf("format = %v, want image2", m["format"])
	}
	if m["was_compressed"] != false {
		t.Errorf("was_compressed = %v, want false", m["was_compressed"])
	}
	if m["file_size_original"] != float64(5_000_000) {
		t.Errorf("file_size_original = %v, want 5000000", m["file_size_original"])
	}
	if m["width"] != float64(2000) || m["height"] != float64(1500) {
		t.Errorf("dimensions = %vx%v, want 2000x1500", m["width"], m["height"])
	}
	if m["file_size_compressed"] != nil {
		t.Errorf("file_size_compressed = %v, want nil", m["file_size_compressed"])
	}
	if _, ok := m["bytes_per_pixel"]; !ok {
		t.Error("expected bytes_per_pixel for image with dimensions")
	}
}

func TestBuildProbeMetadataJSON_compressed(t *testing.T) {
	probe := &ProbeResult{
		Format: "image2",
		Width:  1000,
		Height: 1000,
	}
	compressedSize := int64(2_000_000)

	data := buildProbeMetadataJSON(probe, true, 5_000_000, &compressedSize)

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if m["was_compressed"] != true {
		t.Errorf("was_compressed = %v, want true", m["was_compressed"])
	}
	if m["file_size_compressed"] != float64(2_000_000) {
		t.Errorf("file_size_compressed = %v, want 2000000", m["file_size_compressed"])
	}
	ratio, ok := m["compression_ratio"].(float64)
	if !ok || ratio < 2.4 || ratio > 2.6 {
		t.Errorf("compression_ratio = %v, want ~2.5", m["compression_ratio"])
	}
}

func TestBuildProbeMetadataJSON_video(t *testing.T) {
	probe := &ProbeResult{
		Format:          "mp4",
		Width:           1920,
		Height:          1080,
		DurationSeconds: 300.5,
		BitrateBps:      2_500_000,
		Codec:           "h264",
	}

	data := buildProbeMetadataJSON(probe, false, 90_000_000, nil)

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if m["duration_seconds"] != 300.5 {
		t.Errorf("duration_seconds = %v, want 300.5", m["duration_seconds"])
	}
	if m["bitrate_bps"] != float64(2_500_000) {
		t.Errorf("bitrate_bps = %v, want 2500000", m["bitrate_bps"])
	}
	if m["codec"] != "h264" {
		t.Errorf("codec = %v, want h264", m["codec"])
	}
}

func TestBuildProbeMetadataJSON_audio(t *testing.T) {
	probe := &ProbeResult{
		Format:          "mp3",
		DurationSeconds: 240.0,
		BitrateBps:      320_000,
		Codec:           "mp3",
		SampleRate:      44100,
		Channels:        2,
	}

	data := buildProbeMetadataJSON(probe, false, 10_000_000, nil)

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if m["sample_rate"] != float64(44100) {
		t.Errorf("sample_rate = %v, want 44100", m["sample_rate"])
	}
	if m["channels"] != float64(2) {
		t.Errorf("channels = %v, want 2", m["channels"])
	}
	if _, ok := m["bytes_per_pixel"]; ok {
		t.Error("expected no bytes_per_pixel for audio")
	}
}

// ─── resizeFitWithin Tests ───────────────────────────────────────────────────

func TestResizeFitWithin_already_fits(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 80))
	result := resizeFitWithin(src, 200, 200)

	if result.Bounds().Dx() != 100 || result.Bounds().Dy() != 80 {
		t.Errorf("result = %dx%d, want 100x80 (unchanged)", result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestResizeFitWithin_landscape(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1000, 500))
	result := resizeFitWithin(src, 200, 200)

	w := result.Bounds().Dx()
	h := result.Bounds().Dy()
	if w != 200 {
		t.Errorf("width = %d, want 200", w)
	}
	if h != 100 {
		t.Errorf("height = %d, want 100", h)
	}
}

func TestResizeFitWithin_portrait(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 500, 1000))
	result := resizeFitWithin(src, 200, 200)

	w := result.Bounds().Dx()
	h := result.Bounds().Dy()
	if w != 100 {
		t.Errorf("width = %d, want 100", w)
	}
	if h != 200 {
		t.Errorf("height = %d, want 200", h)
	}
}

func TestResizeFitWithin_square(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 800, 800))
	result := resizeFitWithin(src, 200, 200)

	w := result.Bounds().Dx()
	h := result.Bounds().Dy()
	if w != 200 || h != 200 {
		t.Errorf("result = %dx%d, want 200x200", w, h)
	}
}

func TestResizeFitWithin_non_square_constraint(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 800, 600))
	result := resizeFitWithin(src, 400, 200)

	w := result.Bounds().Dx()
	h := result.Bounds().Dy()
	// Height is constraining: scale = 200/600 = 0.333, w = 266, h = 200
	if h != 200 {
		t.Errorf("height = %d, want 200", h)
	}
	if w < 260 || w > 270 {
		t.Errorf("width = %d, want ~266", w)
	}
}

// ─── generateVariant Tests ───────────────────────────────────────────────────

// makeTestJPEG creates a minimal JPEG image in memory with the given dimensions.
func makeTestJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Fill with a recognizable color
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("creating test JPEG: %v", err)
	}
	return buf.Bytes()
}

// makeTestPNG creates a minimal PNG image in memory.
func makeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("creating test PNG: %v", err)
	}
	return buf.Bytes()
}

func TestGenerateVariant_jpeg_thumb(t *testing.T) {
	jpegBytes := makeTestJPEG(t, 800, 600)

	variant, err := generateVariant(jpegBytes, "image/jpeg", 200, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(variant) == 0 {
		t.Fatal("expected non-empty variant bytes")
	}

	// Decode result and verify dimensions
	img, err := jpeg.Decode(bytes.NewReader(variant))
	if err != nil {
		t.Fatalf("decoding variant JPEG: %v", err)
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w != 200 {
		t.Errorf("width = %d, want 200", w)
	}
	if h != 150 {
		t.Errorf("height = %d, want 150", h)
	}
}

func TestGenerateVariant_png_medium(t *testing.T) {
	pngBytes := makeTestPNG(t, 2000, 1000)

	variant, err := generateVariant(pngBytes, "image/png", 800, 800)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(variant) == 0 {
		t.Fatal("expected non-empty variant bytes")
	}

	// PNG variants stay as PNG
	img, err := png.Decode(bytes.NewReader(variant))
	if err != nil {
		t.Fatalf("decoding variant PNG: %v", err)
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w != 800 {
		t.Errorf("width = %d, want 800", w)
	}
	if h != 400 {
		t.Errorf("height = %d, want 400", h)
	}
}

func TestGenerateVariant_already_fits(t *testing.T) {
	jpegBytes := makeTestJPEG(t, 100, 80)

	variant, err := generateVariant(jpegBytes, "image/jpeg", 200, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still produce output (re-encoded), but dimensions preserved
	img, err := jpeg.Decode(bytes.NewReader(variant))
	if err != nil {
		t.Fatalf("decoding variant: %v", err)
	}
	if img.Bounds().Dx() != 100 || img.Bounds().Dy() != 80 {
		t.Errorf("dimensions = %dx%d, want 100x80", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestGenerateVariant_invalid_data(t *testing.T) {
	_, err := generateVariant([]byte("not an image"), "image/jpeg", 200, 200)
	if err == nil {
		t.Error("expected error for invalid image data")
	}
}

// ─── variantContentType / variantExtension Tests ─────────────────────────────

func TestVariantContentType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"image/jpeg", "image/jpeg"},
		{"image/png", "image/png"},
		{"image/gif", "image/gif"},
		{"image/webp", "image/jpeg"}, // WebP falls back to JPEG
	}
	for _, tc := range tests {
		got := variantContentType(tc.input)
		if got != tc.want {
			t.Errorf("variantContentType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestVariantExtension(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"image/jpeg", "jpg"},
		{"image/png", "png"},
		{"image/gif", "gif"},
		{"image/webp", "jpg"}, // WebP falls back to JPG
	}
	for _, tc := range tests {
		got := variantExtension(tc.input)
		if got != tc.want {
			t.Errorf("variantExtension(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── parseNumber / parseFloat Tests ──────────────────────────────────────────

func TestParseNumber(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"320000", 320000},
		{"0", 0},
		{"", 0},
		{"notanumber", 0},
	}
	for _, tc := range tests {
		got := parseNumber(tc.input)
		if got != tc.want {
			t.Errorf("parseNumber(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"120.5", 120.5},
		{"0.0", 0.0},
		{"", 0.0},
		{"notanumber", 0.0},
	}
	for _, tc := range tests {
		got := parseFloat(tc.input)
		if got != tc.want {
			t.Errorf("parseFloat(%q) = %f, want %f", tc.input, got, tc.want)
		}
	}
}
