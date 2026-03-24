package media

import "strings"

// shouldCompress determines whether an asset should be compressed based on its
// probe metadata. Returns true if the asset exceeds the threshold for its type.
// [09-media §10.2]
//
// Thresholds:
//   - JPEG: > 1.5 bytes/pixel
//   - PNG:  > 4.0 bytes/pixel
//   - WebP: > 1.0 bytes/pixel
//   - GIF:  > 5 MB total size
//   - Video (AV1): > 4 Mbps for 1080p, > 2 Mbps for 720p
//   - Video (non-AV1): always compress (transcode to AV1)
//   - Audio (uncompressed): WAV, FLAC, AIFF → always compress
//   - Audio (compressed): > 256 kbps
//   - PDF: never compress
func shouldCompress(contentType string, sizeBytes int64, width, height int, bitrateBps int64, codec string) bool {
	// PDF: never compress
	if contentType == "application/pdf" || contentType == "application/zip" {
		return false
	}

	// Images: bytes-per-pixel threshold
	if isImageType(contentType) {
		if width <= 0 || height <= 0 {
			return false // no dimensions available, can't determine BPP
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
		// Non-AV1 codecs always need transcoding
		if codec != "av1" {
			return true
		}
		// AV1: check bitrate thresholds
		if height >= 1080 {
			return bitrateBps > 4_000_000 // 4 Mbps for 1080p
		}
		return bitrateBps > 2_000_000 // 2 Mbps for 720p and below
	}

	// Audio: format and bitrate threshold
	if strings.HasPrefix(contentType, "audio/") {
		// Uncompressed formats always need compression
		switch contentType {
		case "audio/wav", "audio/flac", "audio/aiff":
			return true
		}
		// Compressed formats: > 256 kbps
		return bitrateBps > 256_000
	}

	return false
}

// isImageType returns true if the content type is an image type.
func isImageType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}
