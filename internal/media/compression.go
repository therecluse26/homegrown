package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/webp" // register WebP decoder for image.Decode
	"golang.org/x/image/draw"
)

// ─── ffprobe ──────────────────────────────────────────────────────────────────

// ProbeResult holds normalized ffprobe output for all asset types.
// Stored as probe_metadata JSONB on media_uploads. [09-media §10.5]
type ProbeResult struct {
	Format          string  `json:"format"`
	Width           int     `json:"width,omitempty"`
	Height          int     `json:"height,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	BitrateBps      int64   `json:"bitrate_bps,omitempty"`
	Codec           string  `json:"codec,omitempty"`
	SampleRate      int     `json:"sample_rate,omitempty"`
	Channels        int     `json:"channels,omitempty"`
}

// ffprobeOutput is the raw JSON structure returned by ffprobe -print_format json.
type ffprobeOutput struct {
	Format  *ffprobeFormat   `json:"format"`
	Streams []ffprobeStream  `json:"streams"`
}

type ffprobeFormat struct {
	FormatName string `json:"format_name"`
	BitRate    string `json:"bit_rate"`
	Duration   string `json:"duration"`
}

type ffprobeStream struct {
	CodecType  string `json:"codec_type"`
	CodecName  string `json:"codec_name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	BitRate    string `json:"bit_rate"`
	SampleRate string `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Duration   string `json:"duration"`
}

// runFFProbe executes the ffprobe binary on a local file and parses JSON output.
// Returns a normalized ProbeResult suitable for JSONB storage. [09-media §10.1 step 2]
func runFFProbe(ctx context.Context, filePath string) (*ProbeResult, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe execution failed: %w", err)
	}

	return parseFFProbeOutput(out)
}

// parseFFProbeOutput parses raw ffprobe JSON output into a ProbeResult.
// Exported for testing.
func parseFFProbeOutput(data []byte) (*ProbeResult, error) {
	var raw ffprobeOutput
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing ffprobe JSON: %w", err)
	}

	result := &ProbeResult{}

	// Extract format name
	if raw.Format != nil {
		result.Format = raw.Format.FormatName
		if raw.Format.BitRate != "" {
			result.BitrateBps = parseNumber(raw.Format.BitRate)
		}
		if raw.Format.Duration != "" {
			result.DurationSeconds = parseFloat(raw.Format.Duration)
		}
	}

	// Find the primary video and audio streams
	for _, s := range raw.Streams {
		switch s.CodecType {
		case "video":
			// First video stream wins
			if result.Width == 0 {
				result.Width = s.Width
				result.Height = s.Height
				result.Codec = s.CodecName
				if s.BitRate != "" {
					result.BitrateBps = parseNumber(s.BitRate)
				}
			}
		case "audio":
			// First audio stream wins (set codec if no video)
			if result.Codec == "" {
				result.Codec = s.CodecName
			}
			if s.BitRate != "" && result.BitrateBps == 0 {
				result.BitrateBps = parseNumber(s.BitRate)
			}
			if s.SampleRate != "" {
				result.SampleRate = int(parseNumber(s.SampleRate))
			}
			result.Channels = s.Channels
			if s.Duration != "" && result.DurationSeconds == 0 {
				result.DurationSeconds = parseFloat(s.Duration)
			}
		}
	}

	return result, nil
}

// parseNumber parses a string number (int64) — ffprobe returns numbers as strings.
func parseNumber(s string) int64 {
	var n int64
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

// parseFloat parses a string float — ffprobe returns durations as string floats.
func parseFloat(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return f
}

// buildProbeMetadataJSON creates the probe_metadata JSONB to store on media_uploads. [09-media §10.5]
func buildProbeMetadataJSON(probe *ProbeResult, wasCompressed bool, originalSize int64, compressedSize *int64) json.RawMessage {
	m := map[string]any{
		"format":              probe.Format,
		"was_compressed":      wasCompressed,
		"file_size_original":  originalSize,
	}
	if probe.Width > 0 {
		m["width"] = probe.Width
		m["height"] = probe.Height
	}
	if probe.DurationSeconds > 0 {
		m["duration_seconds"] = probe.DurationSeconds
	}
	if probe.BitrateBps > 0 {
		m["bitrate_bps"] = probe.BitrateBps
	}
	if probe.Codec != "" {
		m["codec"] = probe.Codec
	}
	if probe.SampleRate > 0 {
		m["sample_rate"] = probe.SampleRate
		m["channels"] = probe.Channels
	}
	if wasCompressed && compressedSize != nil {
		m["file_size_compressed"] = *compressedSize
		if originalSize > 0 {
			m["compression_ratio"] = float64(originalSize) / float64(*compressedSize)
		}
	} else {
		m["file_size_compressed"] = nil
	}
	if probe.Width > 0 && probe.Height > 0 && originalSize > 0 {
		m["bytes_per_pixel"] = float64(originalSize) / float64(int64(probe.Width)*int64(probe.Height))
	}

	data, _ := json.Marshal(m)
	return data
}

// ─── Compression ──────────────────────────────────────────────────────────────

// shouldCompress determines whether an asset should be compressed based on its
// probe metadata. Returns true if the asset exceeds the threshold for its type.
// [09-media §10.2]
func shouldCompress(contentType string, sizeBytes int64, width, height int, bitrateBps int64, codec string) bool {
	// PDF / ZIP: never compress
	if contentType == "application/pdf" || contentType == "application/zip" {
		return false
	}

	// Images: bytes-per-pixel threshold
	if isImageType(contentType) {
		if width <= 0 || height <= 0 {
			return false
		}
		pixels := int64(width) * int64(height)
		bpp := float64(sizeBytes) / float64(pixels)

		switch contentType {
		case "image/jpeg":
			return bpp > 1.5
		case "image/png":
			return bpp > 4.0
		case "image/webp":
			return bpp > 1.0
		case "image/gif":
			return sizeBytes > 5*1024*1024 // 5 MB
		}
		return false
	}

	// Video: codec and bitrate threshold
	if strings.HasPrefix(contentType, "video/") {
		if codec != "av1" {
			return true
		}
		if height >= 1080 {
			return bitrateBps > 4_000_000
		}
		return bitrateBps > 2_000_000
	}

	// Audio: format and bitrate threshold
	if strings.HasPrefix(contentType, "audio/") {
		switch contentType {
		case "audio/wav", "audio/flac", "audio/aiff":
			return true
		}
		return bitrateBps > 256_000
	}

	return false
}

// compress runs the appropriate compression for the given content type.
// Returns the path to the compressed output file. [09-media §10.2]
func compress(ctx context.Context, inputPath string, contentType string) (string, error) {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	outPath := filepath.Join(dir, "compressed"+ext)

	switch {
	case contentType == "image/jpeg":
		return compressJPEG(inputPath, outPath)
	case contentType == "image/png":
		return compressPNG(inputPath, outPath)
	case contentType == "image/webp":
		return compressWithFFmpeg(ctx, inputPath, outPath, []string{"-quality", "82"})
	case contentType == "image/gif":
		return compressGIF(ctx, inputPath, outPath)
	case strings.HasPrefix(contentType, "video/"):
		return compressVideo(ctx, inputPath, outPath)
	case strings.HasPrefix(contentType, "audio/"):
		outPath = filepath.Join(dir, "compressed.m4a")
		return compressAudio(ctx, inputPath, outPath)
	}

	return inputPath, nil // no compression — pass through
}

// compressJPEG decodes a JPEG, strips EXIF (by re-encoding), and re-encodes at quality 85.
func compressJPEG(inputPath, outputPath string) (string, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("opening JPEG: %w", err)
	}
	defer func() { _ = f.Close() }()

	img, err := jpeg.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decoding JPEG: %w", err)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("creating output JPEG: %w", err)
	}
	defer func() { _ = out.Close() }()

	if err := jpeg.Encode(out, img, &jpeg.Options{Quality: 85}); err != nil {
		return "", fmt.Errorf("encoding JPEG: %w", err)
	}

	return outputPath, nil
}

// compressPNG decodes and re-encodes as optimized PNG.
func compressPNG(inputPath, outputPath string) (string, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("opening PNG: %w", err)
	}
	defer func() { _ = f.Close() }()

	img, err := png.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decoding PNG: %w", err)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("creating output PNG: %w", err)
	}
	defer func() { _ = out.Close() }()

	enc := &png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(out, img); err != nil {
		return "", fmt.Errorf("encoding PNG: %w", err)
	}

	return outputPath, nil
}

// compressWithFFmpeg runs ffmpeg with custom args for a single output.
func compressWithFFmpeg(ctx context.Context, inputPath, outputPath string, extraArgs []string) (string, error) {
	args := []string{"-i", inputPath, "-y"}
	args = append(args, extraArgs...)
	args = append(args, outputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg compression failed: %w (output: %s)", err, string(output))
	}

	return outputPath, nil
}

// compressGIF re-encodes a GIF with optimized palette via ffmpeg.
func compressGIF(ctx context.Context, inputPath, outputPath string) (string, error) {
	return compressWithFFmpeg(ctx, inputPath, outputPath, []string{
		"-vf", "fps=15,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse",
	})
}

// compressVideo transcodes video to AV1 using libsvtav1 at CRF 30, preset 6.
func compressVideo(ctx context.Context, inputPath, outputPath string) (string, error) {
	return compressWithFFmpeg(ctx, inputPath, outputPath, []string{
		"-c:v", "libsvtav1", "-crf", "30", "-preset", "6",
		"-c:a", "aac", "-b:a", "192k",
	})
}

// compressAudio transcodes audio to AAC at 192 kbps.
func compressAudio(ctx context.Context, inputPath, outputPath string) (string, error) {
	return compressWithFFmpeg(ctx, inputPath, outputPath, []string{
		"-c:a", "aac", "-b:a", "192k", "-vn",
	})
}

// ─── CompressAssetPayload ─────────────────────────────────────────────────────

// CompressAssetPayload is the job payload for future separate compression jobs.
// Currently compression is run inline in ProcessUploadJob. [09-media §14]
type CompressAssetPayload struct {
	UploadID   string `json:"upload_id"`
	StorageKey string `json:"storage_key"`
}

// TaskType returns the asynq task type.
func (CompressAssetPayload) TaskType() string { return "media:compress_asset" }

// ─── Image Variant Generation ─────────────────────────────────────────────────

// resizeFitWithin resizes an image to fit within maxW x maxH, preserving aspect ratio.
// Uses CatmullRom interpolation for high-quality output. [09-media §10.7]
func resizeFitWithin(src image.Image, maxW, maxH int) image.Image {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	if srcW <= maxW && srcH <= maxH {
		return src // already fits
	}

	// Calculate scale factor preserving aspect ratio
	scaleW := float64(maxW) / float64(srcW)
	scaleH := float64(maxH) / float64(srcH)
	scale := scaleW
	if scaleH < scaleW {
		scale = scaleH
	}

	dstW := int(float64(srcW) * scale)
	dstH := int(float64(srcH) * scale)
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, srcBounds, draw.Over, nil)

	return dst
}

// generateVariant decodes an image from bytes, resizes, and re-encodes.
// Returns the encoded bytes for the variant. [09-media §10.7]
func generateVariant(imgBytes []byte, contentType string, maxW, maxH int) ([]byte, error) {
	reader := bytes.NewReader(imgBytes)
	src, _, err := image.Decode(reader)
	if err != nil {
		slog.Warn("could not decode image for variant generation", "error", err, "content_type", contentType)
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	resized := resizeFitWithin(src, maxW, maxH)

	var buf bytes.Buffer
	switch contentType {
	case "image/png":
		if err := png.Encode(&buf, resized); err != nil {
			return nil, fmt.Errorf("encoding PNG variant: %w", err)
		}
	case "image/gif":
		if err := gif.Encode(&buf, resized, nil); err != nil {
			return nil, fmt.Errorf("encoding GIF variant: %w", err)
		}
	default:
		// JPEG and WebP variants are encoded as JPEG (WebP encode not in stdlib)
		if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("encoding JPEG variant: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// variantContentType returns the content type for a variant of the given source type.
// WebP variants are encoded as JPEG since Go stdlib doesn't support WebP encoding.
func variantContentType(sourceContentType string) string {
	switch sourceContentType {
	case "image/png":
		return "image/png"
	case "image/gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

// variantExtension returns the file extension for a variant.
func variantExtension(sourceContentType string) string {
	switch sourceContentType {
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	default:
		return "jpg"
	}
}

// ─── Utilities ────────────────────────────────────────────────────────────────

// isImageType returns true if the content type is an image type.
func isImageType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}
