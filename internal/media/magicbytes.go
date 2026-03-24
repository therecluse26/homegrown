package media

import "bytes"

// detectFileType detects file type from magic bytes (first 12+ bytes).
// Returns a general MIME type string or "" if unrecognized. [09-media §11.1]
func detectFileType(header []byte) string {
	if len(header) < 3 {
		return ""
	}

	// JPEG: FF D8 FF
	if header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return "image/jpeg"
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if len(header) >= 8 && bytes.Equal(header[0:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "image/png"
	}

	// GIF: 47 49 46 38
	if len(header) >= 4 && bytes.Equal(header[0:4], []byte{0x47, 0x49, 0x46, 0x38}) {
		return "image/gif"
	}

	// PDF: 25 50 44 46
	if len(header) >= 4 && bytes.Equal(header[0:4], []byte{0x25, 0x50, 0x44, 0x46}) {
		return "application/pdf"
	}

	// RIFF container: WebP, WAV, AVI (check sub-type at offset 8)
	if len(header) >= 12 && bytes.Equal(header[0:4], []byte{0x52, 0x49, 0x46, 0x46}) {
		subType := header[8:12]
		switch {
		case bytes.Equal(subType, []byte{0x57, 0x45, 0x42, 0x50}): // WEBP
			return "image/webp"
		case bytes.Equal(subType, []byte{0x57, 0x41, 0x56, 0x45}): // WAVE
			return "audio/wav"
		case bytes.Equal(subType, []byte{0x41, 0x56, 0x49, 0x20}): // AVI\x20
			return "video/x-msvideo"
		}
	}

	// AIFF: FORM....AIFF
	if len(header) >= 12 &&
		bytes.Equal(header[0:4], []byte{0x46, 0x4F, 0x52, 0x4D}) &&
		bytes.Equal(header[8:12], []byte{0x41, 0x49, 0x46, 0x46}) {
		return "audio/aiff"
	}

	// FLAC: fLaC
	if len(header) >= 4 && bytes.Equal(header[0:4], []byte{0x66, 0x4C, 0x61, 0x43}) {
		return "audio/flac"
	}

	// MP3: FF FB/F3/F2 (sync word) or ID3 tag header
	if (header[0] == 0xFF && (header[1]&0xE0) == 0xE0) ||
		(len(header) >= 3 && bytes.Equal(header[0:3], []byte{0x49, 0x44, 0x33})) {
		return "audio/mpeg"
	}

	// WebM/MKV: EBML header 1A 45 DF A3
	if len(header) >= 4 && bytes.Equal(header[0:4], []byte{0x1A, 0x45, 0xDF, 0xA3}) {
		return "video/webm"
	}

	// ftyp box at offset 4: MP4, MOV, M4A all share this container
	if len(header) >= 8 && bytes.Equal(header[4:8], []byte{0x66, 0x74, 0x79, 0x70}) {
		return "video/mp4" // generic ftyp — isCompatible handles sub-types
	}

	// ZIP: PK\x03\x04
	if len(header) >= 4 && bytes.Equal(header[0:4], []byte{0x50, 0x4B, 0x03, 0x04}) {
		return "application/zip"
	}

	return ""
}

// isCompatible checks whether a declared MIME type is compatible with the
// detected type from magic bytes. Handles the ftyp container ambiguity where
// MP4, MOV, and M4A/AAC all share the same ftyp box signature. [09-media §11.2]
func isCompatible(declared, detected string) bool {
	if detected == "" {
		return false
	}
	if declared == detected {
		return true
	}

	// ftyp container family: video/mp4, video/quicktime, audio/mp4 all detect as "video/mp4"
	ftypFamily := map[string]bool{
		"video/mp4":       true,
		"video/quicktime": true,
		"audio/mp4":       true,
	}
	if ftypFamily[declared] && detected == "video/mp4" {
		return true
	}

	return false
}
