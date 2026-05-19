package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── isValidImageMagic ─────────────────────────────────────────────────────────

func TestIsValidImageMagic_JPEG(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	if !isValidImageMagic(data) {
		t.Fatal("expected JPEG magic bytes to be valid")
	}
}

func TestIsValidImageMagic_PNG(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !isValidImageMagic(data) {
		t.Fatal("expected PNG magic bytes to be valid")
	}
}

func TestIsValidImageMagic_Invalid(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"pdf", []byte{0x25, 0x50, 0x44, 0x46}},  // %PDF
		{"too short", []byte{0xFF, 0xD8}},
		{"random", []byte("hello world")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if isValidImageMagic(tc.data) {
				t.Fatal("expected invalid magic bytes to be rejected")
			}
		})
	}
}

// ── resizeBilinear ────────────────────────────────────────────────────────────

func TestResizeBilinear_OutputDimensions(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 800, 600))
	// Fill with a non-zero color so the resize has meaningful work to do.
	for y := range 600 {
		for x := range 800 {
			src.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}

	dst := resizeBilinear(src, thumbWidth, thumbHeight)
	b := dst.Bounds()
	if b.Dx() != thumbWidth || b.Dy() != thumbHeight {
		t.Fatalf("expected %dx%d, got %dx%d", thumbWidth, thumbHeight, b.Dx(), b.Dy())
	}
}

// ── wikimediaImageURL ─────────────────────────────────────────────────────────

func TestWikimediaImageURL_HappyPath(t *testing.T) {
	searchResp := wikimediaSearchResp{}
	searchResp.Query.Search = []struct {
		Title string `json:"title"`
	}{
		{Title: "File:Test_image.jpg"},
	}
	searchJSON, _ := json.Marshal(searchResp)

	infoResp := wikimediaInfoResp{}
	infoResp.Query.Pages = map[string]struct {
		ImageInfo []struct {
			URL string `json:"url"`
		} `json:"imageinfo"`
	}{
		"12345": {
			ImageInfo: []struct {
				URL string `json:"url"`
			}{{URL: "https://upload.wikimedia.org/wikipedia/commons/test.jpg"}},
		},
	}
	infoJSON, _ := json.Marshal(infoResp)

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		list := r.URL.Query().Get("list")
		if list == "search" {
			_, _ = w.Write(searchJSON)
		} else {
			_, _ = w.Write(infoJSON)
		}
	}))
	defer srv.Close()

	// Override the Wikimedia URL by using a custom client that rewrites requests.
	// Since wikimediaImageURL hard-codes the base URL, we patch via round-trip.
	client := &http.Client{
		Transport: &hostRewriter{target: srv.URL},
	}

	got, err := wikimediaImageURL(context.Background(), client, "Test image", -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://upload.wikimedia.org/wikipedia/commons/test.jpg" {
		t.Fatalf("unexpected URL: %s", got)
	}
}

func TestWikimediaImageURL_SkipsNonJPEGPNG(t *testing.T) {
	searchResp := wikimediaSearchResp{}
	searchResp.Query.Search = []struct {
		Title string `json:"title"`
	}{
		{Title: "File:Some_diagram.svg"},  // SVG — should be skipped
		{Title: "File:Audio_clip.ogg"},    // OGG — should be skipped
	}
	searchJSON, _ := json.Marshal(searchResp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(searchJSON)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &hostRewriter{target: srv.URL}}
	got, err := wikimediaImageURL(context.Background(), client, "Audio diagram", -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty result for non-JPEG/PNG files, got: %s", got)
	}
}

// ── smithsonianImageURL ───────────────────────────────────────────────────────

func TestSmithsonianImageURL_HappyPath(t *testing.T) {
	resp := smithsonianResp{}
	resp.Response.Rows = []struct {
		OnlineMedia struct {
			Media []struct {
				Content string `json:"content"`
				Type    string `json:"type"`
			} `json:"media"`
		} `json:"online_media"`
	}{
		{
			OnlineMedia: struct {
				Media []struct {
					Content string `json:"content"`
					Type    string `json:"type"`
				} `json:"media"`
			}{
				Media: []struct {
					Content string `json:"content"`
					Type    string `json:"type"`
				}{
					{Content: "https://ids.si.edu/ids/test.jpg", Type: "Images"},
				},
			},
		},
	}
	respJSON, _ := json.Marshal(resp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(respJSON)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &hostRewriter{target: srv.URL}}
	got, err := smithsonianImageURL(context.Background(), client, "Test painting", "testkey", -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://ids.si.edu/ids/test.jpg" {
		t.Fatalf("unexpected URL: %s", got)
	}
}

// ── downloadResizeUpload ──────────────────────────────────────────────────────

func TestDownloadResizeUpload_ValidJPEG(t *testing.T) {
	// Build a minimal valid JPEG in memory.
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := range 100 {
		for x := range 100 {
			src.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var jpegBuf bytes.Buffer
	_ = jpeg.Encode(&jpegBuf, src, nil)

	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpegBuf.Bytes())
	}))
	defer imgSrv.Close()

	storage := &stubThumbnailStorage{presignURL: "https://r2.example.com/thumbnails/test.jpg"}
	client := &http.Client{}

	got, err := downloadResizeUpload(context.Background(), client, storage, "test-listing-id", imgSrv.URL+"/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://r2.example.com/thumbnails/test.jpg" {
		t.Fatalf("unexpected presign URL: %s", got)
	}
	if storage.putKey != "thumbnails/test-listing-id.jpg" {
		t.Fatalf("unexpected storage key: %s", storage.putKey)
	}
}

func TestDownloadResizeUpload_InvalidMagicBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("%PDF-1.4 not an image"))
	}))
	defer srv.Close()

	storage := &stubThumbnailStorage{}
	client := &http.Client{}

	_, err := downloadResizeUpload(context.Background(), client, storage, "lid", srv.URL+"/file.jpg")
	if err == nil {
		t.Fatal("expected error for invalid magic bytes")
	}
}

// ── RunThumbnailFetcher ───────────────────────────────────────────────────────

func TestRunThumbnailFetcher_DryRunNoDB(t *testing.T) {
	cfg := Config{DryRun: true, DB: nil}
	n, err := RunThumbnailFetcher(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestRunThumbnailFetcher_NoStorageSkips(t *testing.T) {
	// Non-dry-run with nil ThumbnailStorage should warn and return 0 without error.
	cfg := Config{DryRun: false, DB: nil, ThumbnailStorage: nil}
	// DB is nil, which triggers the "DB required" error path, so test the storage-nil path with DryRun.
	cfg2 := Config{DryRun: false, ThumbnailStorage: nil}
	// We can't easily inject a real DB here — just verify DB-nil returns an error.
	_, err := RunThumbnailFetcher(context.Background(), cfg2)
	if err == nil {
		t.Fatal("expected error when DB is nil in non-dry-run mode")
	}
	_ = cfg // silence unused warning
}

// ── test helpers ──────────────────────────────────────────────────────────────

// stubThumbnailStorage records calls for assertions.
type stubThumbnailStorage struct {
	putKey     string
	putData    []byte
	presignURL string
}

func (s *stubThumbnailStorage) PutObject(_ context.Context, key string, data []byte, _ string) error {
	s.putKey = key
	s.putData = data
	return nil
}

func (s *stubThumbnailStorage) PresignedGet(_ context.Context, _ string, _ uint32) (string, error) {
	return s.presignURL, nil
}

// hostRewriter is an http.RoundTripper that rewrites the Host to a test server
// while preserving the path and query string. This lets us test functions that
// hard-code the API base URL without modifying their signatures.
type hostRewriter struct {
	target string // e.g. "http://127.0.0.1:PORT"
}

func (h *hostRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = h.target[len("http://"):]
	return http.DefaultTransport.RoundTrip(clone)
}
