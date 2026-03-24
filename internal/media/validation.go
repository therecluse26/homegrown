package media

import (
	"path/filepath"
	"strings"
	"unicode"
)

// ─── Context Rules ────────────────────────────────────────────────────────────

// contextRules defines the maximum file size (bytes) and allowed content types
// for a given upload context. Populated from the §9.1 table.
type contextRules struct {
	MaxSizeBytes uint64
	AllowedTypes []string
}

// AllowsContentType checks whether a content type is permitted for this context.
func (r *contextRules) AllowsContentType(contentType string) bool {
	for _, t := range r.AllowedTypes {
		if t == contentType {
			return true
		}
	}
	return false
}

const (
	mb = 1024 * 1024
	gb = 1024 * 1024 * 1024
)

// contextRulesTable maps each upload context to its validation rules. [09-media §9.1]
var contextRulesTable = map[UploadContext]*contextRules{
	UploadContextProfilePhoto: {
		MaxSizeBytes: 5 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
	},
	UploadContextPostAttachment: {
		MaxSizeBytes: 25 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif", "video/mp4", "video/webm"},
	},
	UploadContextMessageAttachment: {
		MaxSizeBytes: 10 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif"},
	},
	UploadContextActivityAttachment: {
		MaxSizeBytes: 25 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp", "application/pdf"},
	},
	UploadContextJournalImage: {
		MaxSizeBytes: 10 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif"},
	},
	UploadContextNatureJournalImage: {
		MaxSizeBytes: 10 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
	},
	UploadContextProjectAttachment: {
		MaxSizeBytes: 50 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp", "application/pdf", "video/mp4"},
	},
	UploadContextReadingCover: {
		MaxSizeBytes: 5 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
	},
	UploadContextMarketplaceFile: {
		MaxSizeBytes: 500 * mb,
		AllowedTypes: []string{"application/pdf", "application/zip", "video/mp4", "audio/mpeg"},
	},
	UploadContextListingPreview: {
		MaxSizeBytes: 25 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp", "video/mp4"},
	},
	UploadContextListingThumbnail: {
		MaxSizeBytes: 5 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
	},
	UploadContextCreatorLogo: {
		MaxSizeBytes: 2 * mb,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
	},
	UploadContextDataExport: {
		MaxSizeBytes: 5 * gb,
		AllowedTypes: []string{"application/zip"},
	},
	UploadContextAudioAttachment: {
		MaxSizeBytes: 50 * mb,
		AllowedTypes: []string{"audio/mpeg", "audio/mp4", "audio/wav", "audio/flac", "audio/aiff"},
	},
	UploadContextVideoLesson: {
		MaxSizeBytes: 5 * gb,
		AllowedTypes: []string{"video/mp4", "video/quicktime", "video/x-msvideo", "video/webm"},
	},
}

// getContextRules returns the validation rules for the given upload context.
// Returns nil if the context is unknown.
func getContextRules(ctx UploadContext) *contextRules {
	return contextRulesTable[ctx]
}

// ─── Attachment Validation ────────────────────────────────────────────────────

// validateAttachmentRules validates content type and size against context rules.
// Returns a MediaError wrapping the appropriate sentinel if validation fails.
func validateAttachmentRules(ctx UploadContext, contentType string, sizeBytes uint64) error {
	rules := getContextRules(ctx)
	if rules == nil {
		return &MediaError{Err: ErrInvalidFileType}
	}
	if !rules.AllowsContentType(contentType) {
		return &MediaError{Err: ErrInvalidFileType}
	}
	if sizeBytes > rules.MaxSizeBytes {
		return &MediaError{Err: ErrFileTooLarge}
	}
	return nil
}

// ─── Filename Sanitization ────────────────────────────────────────────────────

// sanitizeFilename strips path separators, control characters, and truncates to 255 bytes.
// Preserves the file extension. [09-media §9.2]
func sanitizeFilename(filename string) string {
	// Handle Windows-style backslash paths (filepath.Base only handles OS separator)
	if idx := strings.LastIndex(filename, "\\"); idx >= 0 {
		filename = filename[idx+1:]
	}

	// Strip directory path components
	filename = filepath.Base(filename)

	// Replace any remaining path separators
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")

	// Remove control characters and non-printable characters
	var b strings.Builder
	for _, r := range filename {
		if unicode.IsControl(r) || !unicode.IsPrint(r) {
			continue
		}
		b.WriteRune(r)
	}
	filename = b.String()

	// Replace spaces with hyphens for URL-friendliness
	filename = strings.ReplaceAll(filename, " ", "-")

	// Ensure we have a non-empty filename
	if filename == "" || filename == "." || filename == ".." {
		filename = "upload"
	}

	// Truncate to 255 bytes (preserving the extension if possible)
	if len(filename) > 255 {
		ext := filepath.Ext(filename)
		base := filename[:len(filename)-len(ext)]
		maxBase := 255 - len(ext)
		if maxBase < 1 {
			maxBase = 255
			ext = ""
		}
		if len(base) > maxBase {
			base = base[:maxBase]
		}
		filename = base + ext
	}

	return filename
}
