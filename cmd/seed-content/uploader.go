package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/media"
)

const (
	pdfMagic        = "%PDF"
	r2PresignTTLSec = 3 * 365 * 24 * 3600 // 3-year presigned URL; long-lived for seed data
	maxPDFBytes     = 50 * 1024 * 1024     // 50 MB guard; CK Foundation PDFs are well under this
)

// R2Uploader implements sources.Uploader backed by an ObjectStorageAdapter.
// It downloads a PDF from a remote URL, validates magic bytes per [CODING §5.2],
// uploads it to R2, and returns a presigned GET URL with a 3-year TTL.
type R2Uploader struct {
	storage media.ObjectStorageAdapter
	client  *http.Client
}

func newR2Uploader(storage media.ObjectStorageAdapter) *R2Uploader {
	return &R2Uploader{
		storage: storage,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Upload downloads srcURL, validates the PDF magic bytes, uploads the file to R2
// under storageKey, and returns (presignedGetURL, sizeBytes, error).
// Satisfies the sources.Uploader interface.
func (u *R2Uploader) Upload(ctx context.Context, srcURL, storageKey string) (string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("build request for %s: %w", srcURL, err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("fetch %s: %w", srcURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("fetch %s: HTTP %d", srcURL, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPDFBytes))
	if err != nil {
		return "", 0, fmt.Errorf("read response from %s: %w", srcURL, err)
	}

	// Validate PDF magic bytes — never rely on Content-Type or file extension alone. [CODING §5.2]
	if len(data) < len(pdfMagic) || string(data[:len(pdfMagic)]) != pdfMagic {
		return "", 0, fmt.Errorf("%s: not a valid PDF (magic bytes mismatch)", srcURL)
	}

	if err := u.storage.PutObject(ctx, storageKey, data, "application/pdf"); err != nil {
		return "", 0, fmt.Errorf("R2 upload of %s: %w", storageKey, err)
	}

	signedURL, err := u.storage.PresignedGet(ctx, storageKey, r2PresignTTLSec)
	if err != nil {
		return "", 0, fmt.Errorf("presign %s: %w", storageKey, err)
	}

	return signedURL, int64(len(data)), nil
}
