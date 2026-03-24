package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Job Payloads ─────────────────────────────────────────────────────────────

// ProcessUploadPayload is the payload for the ProcessUploadJob. [09-media §10]
type ProcessUploadPayload struct {
	UploadID uuid.UUID `json:"upload_id"`
}

func (ProcessUploadPayload) TaskType() string { return "media:process_upload" }

// TranscodeVideoPayload is the payload for the HLS transcode job. [09-media §10.8]
type TranscodeVideoPayload struct {
	UploadID uuid.UUID `json:"upload_id"`
}

func (TranscodeVideoPayload) TaskType() string { return "media:transcode_video" }

// CleanupOrphansPayload is the payload for the orphan cleanup job. [09-media §12]
type CleanupOrphansPayload struct{}

func (CleanupOrphansPayload) TaskType() string { return "media:cleanup_orphans" }

// ─── Process Upload Runner ────────────────────────────────────────────────────

// processUploadRunner holds dependencies for the ProcessUploadJob pipeline.
// Tested directly with mocks — no asynq serialization in unit tests. [09-media §10.1]
type processUploadRunner struct {
	uploads    UploadRepository
	procJobs   ProcessingJobRepository
	transcodes TranscodeJobRepository
	storage    ObjectStorageAdapter
	safety     SafetyScanAdapter
	events     eventPublisher
	jobs       shared.JobEnqueuer
}

// Run executes the full 8-stage processing pipeline for a single upload.
// [09-media §10.1]:
//  1. Magic byte validation
//  2. ffprobe analysis
//  3. CSAM scan (+ last_csam_scanned_at update)
//  4. Content moderation
//  5. Compression decision
//  6. [Conditional] Inline compression
//  7. Variant generation (images only)
//  8. Publish + transcode dispatch for video_lesson
func (r *processUploadRunner) Run(ctx context.Context, uploadID uuid.UUID) error {
	// Fetch upload (unscoped — background job context)
	upload, err := r.uploads.FindByIDUnscoped(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("finding upload: %w", err)
	}

	// ── Step 1: Magic byte validation ─────────────────────────────────────
	header, err := r.storage.GetObjectBytes(ctx, upload.StorageKey, 0, 16)
	if err != nil {
		return fmt.Errorf("reading magic bytes: %w", err)
	}
	detectedType := detectFileType(header)
	if !isCompatible(upload.ContentType, detectedType) {
		slog.Warn("magic byte mismatch",
			"upload_id", uploadID,
			"declared_type", upload.ContentType,
			"detected_type", detectedType,
		)
		if _, statusErr := r.uploads.UpdateStatus(ctx, uploadID, UploadStatusRejected, &UploadStatusUpdate{}); statusErr != nil {
			slog.Error("failed to reject upload after magic byte mismatch", "upload_id", uploadID, "error", statusErr)
		}
		return &MediaError{Err: ErrMagicByteMismatch}
	}

	// ── Step 2: ffprobe analysis ──────────────────────────────────────────
	// Download file to temp dir for ffprobe (and reuse for compression/resize)
	tmpDir, err := os.MkdirTemp("", "media-process-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ext := extensionForContentType(upload.ContentType)
	tmpFile := filepath.Join(tmpDir, "input"+ext)
	if err := r.storage.DownloadToFile(ctx, upload.StorageKey, tmpFile); err != nil {
		return fmt.Errorf("downloading file for probe: %w", err)
	}

	probe, err := runFFProbe(ctx, tmpFile)
	if err != nil {
		// ffprobe failure is non-fatal — log and continue with nil probe
		slog.Warn("ffprobe analysis failed — continuing without probe data", "upload_id", uploadID, "error", err)
		probe = nil
	}

	// Store probe metadata
	if probe != nil {
		originalSize := int64(0)
		if upload.SizeBytes != nil {
			originalSize = *upload.SizeBytes
		}
		probeJSON := buildProbeMetadataJSON(probe, false, originalSize, nil)
		if probeErr := r.uploads.UpdateProbeMetadata(ctx, uploadID, probeJSON, false, nil); probeErr != nil {
			slog.Error("failed to store probe metadata", "upload_id", uploadID, "error", probeErr)
		}
	}

	// ── Step 3: CSAM scan ─────────────────────────────────────────────────
	csamResult, err := r.safety.ScanCSAM(ctx, upload.StorageKey)
	if err != nil {
		var scanErr *ScanError
		if errors.As(err, &scanErr) && scanErr.Code == "unavailable" {
			slog.Warn("CSAM scan unavailable — continuing", "upload_id", uploadID)
		} else {
			return fmt.Errorf("CSAM scan failed: %w", err)
		}
	} else {
		// Update last_csam_scanned_at on successful scan (clean or not)
		if scannedErr := r.uploads.SetCSAMScannedAt(ctx, uploadID); scannedErr != nil {
			slog.Error("failed to update csam scanned timestamp", "upload_id", uploadID, "error", scannedErr)
		}

		if csamResult.IsCSAM {
			// Quarantine immediately — short-circuit
			if _, statusErr := r.uploads.UpdateStatus(ctx, uploadID, UploadStatusQuarantined, &UploadStatusUpdate{}); statusErr != nil {
				return fmt.Errorf("quarantining upload: %w", statusErr)
			}
			if reportErr := r.safety.ReportCSAM(ctx, uploadID, csamResult); reportErr != nil {
				slog.Error("failed to report CSAM", "upload_id", uploadID, "error", reportErr)
			}
			_ = r.events.Publish(ctx, UploadQuarantined{
				UploadID: uploadID,
				FamilyID: upload.FamilyID,
				Context:  upload.Context,
			})
			return nil
		}
	}

	// ── Step 4: Content moderation ────────────────────────────────────────
	modResult, err := r.safety.ScanModeration(ctx, upload.StorageKey)
	if err != nil {
		var scanErr *ScanError
		if errors.As(err, &scanErr) && scanErr.Code == "unavailable" {
			slog.Warn("moderation scan unavailable — continuing", "upload_id", uploadID)
		} else {
			return fmt.Errorf("moderation scan failed: %w", err)
		}
	} else if modResult.AutoReject {
		labelsJSON, _ := json.Marshal(modResult.Labels)
		_ = r.uploads.SetModerationLabels(ctx, uploadID, labelsJSON)
		if _, statusErr := r.uploads.UpdateStatus(ctx, uploadID, UploadStatusRejected, &UploadStatusUpdate{}); statusErr != nil {
			return fmt.Errorf("rejecting upload: %w", statusErr)
		}
		_ = r.events.Publish(ctx, UploadRejected{
			UploadID: uploadID,
			FamilyID: upload.FamilyID,
			Context:  upload.Context,
			Labels:   modResult.Labels,
		})
		return nil
	} else if modResult.HasViolations {
		labelsJSON, _ := json.Marshal(modResult.Labels)
		_ = r.uploads.SetModerationLabels(ctx, uploadID, labelsJSON)
		if _, statusErr := r.uploads.UpdateStatus(ctx, uploadID, UploadStatusFlagged, &UploadStatusUpdate{}); statusErr != nil {
			return fmt.Errorf("flagging upload: %w", statusErr)
		}
		_ = r.events.Publish(ctx, UploadFlagged{
			UploadID: uploadID,
			FamilyID: upload.FamilyID,
			Context:  upload.Context,
			Labels:   modResult.Labels,
			Priority: modResult.Priority,
		})
		return nil
	}

	// ── Steps 5-6: Compression decision + inline compression ──────────────
	wasCompressed := false
	var originalSize *int64
	if probe != nil && upload.SizeBytes != nil {
		if shouldCompress(upload.ContentType, *upload.SizeBytes, probe.Width, probe.Height, probe.BitrateBps, probe.Codec) {
			compressedPath, compErr := compress(ctx, tmpFile, upload.ContentType)
			if compErr != nil {
				slog.Warn("compression failed — continuing with original", "upload_id", uploadID, "error", compErr)
			} else if compressedPath != tmpFile {
				// Upload compressed file back to S3 (replaces original)
				if upErr := r.storage.UploadFromFile(ctx, upload.StorageKey, compressedPath, upload.ContentType); upErr != nil {
					slog.Warn("failed to upload compressed file — continuing with original", "upload_id", uploadID, "error", upErr)
				} else {
					wasCompressed = true
					origSize := *upload.SizeBytes
					originalSize = &origSize
					// Read compressed file size
					if fi, statErr := os.Stat(compressedPath); statErr == nil {
						newSize := fi.Size()
						upload.SizeBytes = &newSize
					}
					// Update temp file path for variant generation
					tmpFile = compressedPath
				}
			}
		}
	}

	// Update probe metadata with compression info
	if wasCompressed && probe != nil {
		var compressedSize *int64
		if upload.SizeBytes != nil {
			cs := *upload.SizeBytes
			compressedSize = &cs
		}
		probeJSON := buildProbeMetadataJSON(probe, true, *originalSize, compressedSize)
		if probeErr := r.uploads.UpdateProbeMetadata(ctx, uploadID, probeJSON, true, originalSize); probeErr != nil {
			slog.Error("failed to update probe metadata after compression", "upload_id", uploadID, "error", probeErr)
		}
	}

	// ── Step 7: Variant generation (images only) ──────────────────────────
	hasThumb := false
	hasMedium := false
	if isImageType(upload.ContentType) {
		// Read the final file (post-compression or original) for variant generation
		imgBytes, readErr := os.ReadFile(tmpFile)
		if readErr != nil {
			return fmt.Errorf("reading image for variants: %w", readErr)
		}

		variants := []struct {
			suffix string
			maxW   int
			maxH   int
		}{
			{"thumb", 200, 200},
			{"medium", 800, 800},
		}

		for _, v := range variants {
			variantBytes, genErr := generateVariant(imgBytes, upload.ContentType, v.maxW, v.maxH)
			if genErr != nil {
				slog.Warn("variant generation failed", "upload_id", uploadID, "variant", v.suffix, "error", genErr)
				continue
			}
			vExt := variantExtension(upload.ContentType)
			vContentType := variantContentType(upload.ContentType)
			// Strip extension from storage key for variant naming
			baseKey := strings.TrimSuffix(upload.StorageKey, filepath.Ext(upload.StorageKey))
			variantKey := fmt.Sprintf("%s__%s.%s", baseKey, v.suffix, vExt)
			if err := r.storage.PutObject(ctx, variantKey, variantBytes, vContentType); err != nil {
				return fmt.Errorf("uploading %s variant: %w", v.suffix, err)
			}
		}

		hasThumb = true
		hasMedium = true
		if err := r.uploads.SetVariantFlags(ctx, uploadID, hasThumb, hasMedium); err != nil {
			return fmt.Errorf("setting variant flags: %w", err)
		}
	}

	// ── Step 8: Publish ───────────────────────────────────────────────────
	now := time.Now()
	sizeBytes := int64(0)
	if upload.SizeBytes != nil {
		sizeBytes = *upload.SizeBytes
	}
	upload, err = r.uploads.UpdateStatus(ctx, uploadID, UploadStatusPublished, &UploadStatusUpdate{
		PublishedAt: &now,
	})
	if err != nil {
		return fmt.Errorf("publishing upload: %w", err)
	}

	_ = r.events.Publish(ctx, UploadPublished{
		UploadID:    uploadID,
		FamilyID:    upload.FamilyID,
		Context:     upload.Context,
		StorageKey:  upload.StorageKey,
		ContentType: upload.ContentType,
		SizeBytes:   sizeBytes,
		HasThumb:    hasThumb,
		HasMedium:   hasMedium,
	})

	// ── Transcode dispatch for video_lesson ───────────────────────────────
	if upload.Context == UploadContextVideoLesson && r.jobs != nil {
		if enqErr := r.jobs.Enqueue(ctx, &TranscodeVideoPayload{UploadID: uploadID}); enqErr != nil {
			slog.Error("failed to enqueue transcode job", "upload_id", uploadID, "error", enqErr)
		}
	}

	return nil
}

// extensionForContentType returns the file extension (with dot) for a content type.
func extensionForContentType(ct string) string {
	switch ct {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "video/mp4":
		return ".mp4"
	case "video/quicktime":
		return ".mov"
	case "video/x-msvideo":
		return ".avi"
	case "video/webm":
		return ".webm"
	case "audio/mpeg":
		return ".mp3"
	case "audio/mp4":
		return ".m4a"
	case "audio/wav":
		return ".wav"
	case "audio/flac":
		return ".flac"
	case "audio/aiff":
		return ".aiff"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	default:
		return ""
	}
}

// ─── Orphan Cleanup ───────────────────────────────────────────────────────────

// RunCleanup finds and marks expired pending uploads. [09-media §12.1]
// Uses continue-on-error pattern — individual failures don't abort the batch.
func RunCleanup(ctx context.Context, uploads UploadRepository, storage ObjectStorageAdapter) (uint32, error) {
	orphans, err := uploads.FindExpiredPending(ctx, time.Now(), 1000)
	if err != nil {
		return 0, fmt.Errorf("finding expired uploads: %w", err)
	}

	var cleaned uint32
	for _, orphan := range orphans {
		// Best-effort delete from S3 (may not exist)
		_ = storage.DeleteObject(ctx, orphan.StorageKey)

		if _, err := uploads.UpdateStatus(ctx, orphan.ID, UploadStatusExpired, &UploadStatusUpdate{}); err != nil {
			slog.Error("failed to expire orphan upload", "upload_id", orphan.ID, "error", err)
			continue
		}
		cleaned++
	}

	if cleaned > 0 {
		slog.Info("cleaned up orphan uploads", "count", cleaned)
	}

	return cleaned, nil
}

// ─── Transcode Video Runner ──────────────────────────────────────────────────

// transcodeVideoRunner holds dependencies for the TranscodeVideoJob. [09-media §10.8]
type transcodeVideoRunner struct {
	uploads    UploadRepository
	transcodes TranscodeJobRepository
	storage    ObjectStorageAdapter
}

// Run executes the HLS transcode pipeline for a video_lesson upload.
func (r *transcodeVideoRunner) Run(ctx context.Context, uploadID uuid.UUID) error {
	upload, err := r.uploads.FindByIDUnscoped(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("finding upload for transcode: %w", err)
	}

	// Create transcode job record
	job, err := r.transcodes.Create(ctx, uploadID, upload.StorageKey)
	if err != nil {
		return fmt.Errorf("creating transcode job: %w", err)
	}

	if err := r.transcodes.MarkRunning(ctx, job.ID); err != nil {
		return fmt.Errorf("marking transcode running: %w", err)
	}

	// Download video to temp dir
	tmpDir, err := os.MkdirTemp("", "media-transcode-*")
	if err != nil {
		_ = r.transcodes.MarkFailed(ctx, job.ID, err.Error())
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	inputFile := filepath.Join(tmpDir, "input"+extensionForContentType(upload.ContentType))
	if err := r.storage.DownloadToFile(ctx, upload.StorageKey, inputFile); err != nil {
		_ = r.transcodes.MarkFailed(ctx, job.ID, err.Error())
		return fmt.Errorf("downloading video: %w", err)
	}

	// Probe for duration and source resolution
	probe, err := runFFProbe(ctx, inputFile)
	if err != nil {
		_ = r.transcodes.MarkFailed(ctx, job.ID, err.Error())
		return fmt.Errorf("probing video: %w", err)
	}

	// Determine quality levels based on source resolution
	levels := []hlsQualityLevel{
		{"480p", 854, 480, "1000k"},
		{"720p", 1280, 720, "2500k"},
	}
	if probe.Height >= 1080 {
		levels = append(levels, hlsQualityLevel{"1080p", 1920, 1080, "5000k"})
	}

	// Generate HLS output for each level
	outputBase := fmt.Sprintf("media/video/%s", uploadID.String())
	outputKeys := map[string]any{
		"master_playlist": outputBase + "/master.m3u8",
		"variants":        []map[string]string{},
	}

	for _, level := range levels {
		levelDir := filepath.Join(tmpDir, level.name)
		if mkErr := os.MkdirAll(levelDir, 0o755); mkErr != nil {
			_ = r.transcodes.MarkFailed(ctx, job.ID, mkErr.Error())
			return fmt.Errorf("creating level dir: %w", mkErr)
		}

		playlistPath := filepath.Join(levelDir, "playlist.m3u8")
		segmentPattern := filepath.Join(levelDir, "segment_%03d.ts")

		args := []string{
			"-i", inputFile,
			"-c:v", "libx264",
			"-b:v", level.bitrate,
			"-s", fmt.Sprintf("%dx%d", level.width, level.height),
			"-c:a", "aac", "-b:a", "128k",
			"-f", "hls",
			"-hls_time", "10",
			"-hls_list_size", "0",
			"-hls_segment_filename", segmentPattern,
			"-y", playlistPath,
		}

		cmd := newCommand(ctx, "ffmpeg", args...)
		if output, cmdErr := cmd.CombinedOutput(); cmdErr != nil {
			errMsg := fmt.Sprintf("ffmpeg transcode failed for %s: %s", level.name, string(output))
			_ = r.transcodes.MarkFailed(ctx, job.ID, errMsg)
			return fmt.Errorf("transcode %s: %w", level.name, cmdErr)
		}

		// Upload segments and playlist
		playlistKey := fmt.Sprintf("%s/%s/playlist.m3u8", outputBase, level.name)
		if upErr := r.storage.UploadFromFile(ctx, playlistKey, playlistPath, "application/x-mpegURL"); upErr != nil {
			_ = r.transcodes.MarkFailed(ctx, job.ID, upErr.Error())
			return fmt.Errorf("uploading playlist: %w", upErr)
		}

		// Upload segments
		segFiles, _ := filepath.Glob(filepath.Join(levelDir, "segment_*.ts"))
		for _, segFile := range segFiles {
			segKey := fmt.Sprintf("%s/%s/%s", outputBase, level.name, filepath.Base(segFile))
			if upErr := r.storage.UploadFromFile(ctx, segKey, segFile, "video/MP2T"); upErr != nil {
				slog.Error("failed to upload segment", "key", segKey, "error", upErr)
			}
		}

		variants := outputKeys["variants"].([]map[string]string)
		outputKeys["variants"] = append(variants, map[string]string{
			"resolution":    level.name,
			"playlist_key":  playlistKey,
			"segment_prefix": fmt.Sprintf("%s/%s/segment_", outputBase, level.name),
		})
	}

	// Generate master playlist
	masterPath := filepath.Join(tmpDir, "master.m3u8")
	if genErr := generateMasterPlaylist(masterPath, levels); genErr != nil {
		_ = r.transcodes.MarkFailed(ctx, job.ID, genErr.Error())
		return fmt.Errorf("generating master playlist: %w", genErr)
	}
	masterKey := outputBase + "/master.m3u8"
	if upErr := r.storage.UploadFromFile(ctx, masterKey, masterPath, "application/x-mpegURL"); upErr != nil {
		_ = r.transcodes.MarkFailed(ctx, job.ID, upErr.Error())
		return fmt.Errorf("uploading master playlist: %w", upErr)
	}

	// Mark completed
	outJSON, _ := json.Marshal(outputKeys)
	duration := int(probe.DurationSeconds)
	if err := r.transcodes.MarkCompleted(ctx, job.ID, outJSON, duration); err != nil {
		return fmt.Errorf("marking transcode completed: %w", err)
	}

	return nil
}

// hlsQualityLevel defines an HLS adaptive quality level for video transcoding.
type hlsQualityLevel struct {
	name    string
	width   int
	height  int
	bitrate string
}

// generateMasterPlaylist creates an HLS master playlist file.
func generateMasterPlaylist(path string, levels []hlsQualityLevel) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, _ = fmt.Fprintln(f, "#EXTM3U")
	for _, level := range levels {
		bw := parseBitrateString(level.bitrate)
		_, _ = fmt.Fprintf(f, "#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n", bw, level.width, level.height)
		_, _ = fmt.Fprintf(f, "%s/playlist.m3u8\n", level.name)
	}

	return nil
}

// parseBitrateString converts a bitrate string like "1000k" to bps.
func parseBitrateString(s string) int {
	s = strings.TrimSuffix(s, "k")
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n * 1000
}

// newCommand creates an exec.Cmd — extracted for testability.
var newCommand = newCommandImpl

func newCommandImpl(ctx context.Context, name string, args ...string) command {
	return execCommand{cmd: exec.CommandContext(ctx, name, args...)}
}

// command interface for testing.
type command interface {
	CombinedOutput() ([]byte, error)
}

type execCommand struct {
	cmd *exec.Cmd
}

func (c execCommand) CombinedOutput() ([]byte, error) { return c.cmd.CombinedOutput() }

// ─── Worker Registration ──────────────────────────────────────────────────────

// RegisterMediaWorkers registers media background job handlers with the worker. [09-media §10]
func RegisterMediaWorkers(
	worker shared.JobWorker,
	uploads UploadRepository,
	procJobs ProcessingJobRepository,
	transcodes TranscodeJobRepository,
	storage ObjectStorageAdapter,
	safety SafetyScanAdapter,
	events eventPublisher,
	jobs shared.JobEnqueuer,
) {
	runner := &processUploadRunner{
		uploads:    uploads,
		procJobs:   procJobs,
		transcodes: transcodes,
		storage:    storage,
		safety:     safety,
		events:     events,
		jobs:       jobs,
	}

	worker.Handle("media:process_upload", func(ctx context.Context, payload []byte) error {
		var p ProcessUploadPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshaling process upload payload: %w", err)
		}

		slog.Info("processing upload", "upload_id", p.UploadID)

		if err := runner.Run(ctx, p.UploadID); err != nil {
			slog.Error("upload processing failed", "upload_id", p.UploadID, "error", err)
			return err
		}

		return nil
	})

	transcodeRunner := &transcodeVideoRunner{
		uploads:    uploads,
		transcodes: transcodes,
		storage:    storage,
	}

	worker.Handle("media:transcode_video", func(ctx context.Context, payload []byte) error {
		var p TranscodeVideoPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshaling transcode payload: %w", err)
		}

		slog.Info("transcoding video", "upload_id", p.UploadID)

		if err := transcodeRunner.Run(ctx, p.UploadID); err != nil {
			slog.Error("video transcode failed", "upload_id", p.UploadID, "error", err)
			return err
		}

		return nil
	})

	worker.Handle("media:cleanup_orphans", func(ctx context.Context, _ []byte) error {
		count, err := RunCleanup(ctx, uploads, storage)
		if err != nil {
			return err
		}
		slog.Info("orphan cleanup completed", "count", count)
		return nil
	})
}
