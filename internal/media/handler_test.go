package media

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

func setupEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e
}

type testValidator struct {
	v *validator.Validate
}

func (tv *testValidator) Validate(i interface{}) error {
	return tv.v.Struct(i)
}

func setAuth(c echo.Context) {
	shared.SetAuthContext(c, &shared.AuthContext{
		ParentID: uuid.Must(uuid.NewV7()),
		FamilyID: uuid.Must(uuid.NewV7()),
	})
}

// ─── POST /v1/media/uploads ───────────────────────────────────────────────────

func TestHandler_RequestUpload_201(t *testing.T) {
	e := setupEcho()
	mockSvc := newMockMediaService()
	h := NewHandler(mockSvc)

	mockSvc.requestUploadFn = func(_ context.Context, input *RequestUploadInput) (*UploadResponse, error) {
		return &UploadResponse{
			UploadID:         uuid.Must(uuid.NewV7()),
			PresignedURL:     "https://s3.example.com/presigned",
			StorageKey:       "uploads/fam/id/photo.jpg",
			ExpiresInSeconds: 3600,
		}, nil
	}

	body := `{"context":"journal_image","content_type":"image/jpeg","filename":"photo.jpg","size_bytes":2048576}`
	req := httptest.NewRequest(http.MethodPost, "/v1/media/uploads", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuth(c)

	h.Register(e.Group("/v1"))
	err := h.requestUpload(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp UploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.PresignedURL == "" {
		t.Error("expected non-empty presigned URL")
	}
}

func TestHandler_RequestUpload_422_invalid_type(t *testing.T) {
	e := setupEcho()
	mockSvc := newMockMediaService()
	h := NewHandler(mockSvc)

	mockSvc.requestUploadFn = func(_ context.Context, _ *RequestUploadInput) (*UploadResponse, error) {
		return nil, &MediaError{Err: ErrInvalidFileType}
	}

	body := `{"context":"profile_photo","content_type":"video/mp4","filename":"vid.mp4","size_bytes":1048576}`
	req := httptest.NewRequest(http.MethodPost, "/v1/media/uploads", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuth(c)

	err := h.requestUpload(c)
	if err == nil {
		t.Fatal("expected error")
	}

	var appErr *shared.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusUnprocessableEntity)
	}
}

func TestHandler_RequestUpload_400_bad_body(t *testing.T) {
	e := setupEcho()
	h := NewHandler(newMockMediaService())

	req := httptest.NewRequest(http.MethodPost, "/v1/media/uploads", strings.NewReader("not json"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuth(c)

	err := h.requestUpload(c)
	if err == nil {
		t.Fatal("expected error for bad body")
	}

	var appErr *shared.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusBadRequest)
	}
}

// ─── POST /v1/media/uploads/:id/confirm ───────────────────────────────────────

func TestHandler_ConfirmUpload_200(t *testing.T) {
	e := setupEcho()
	mockSvc := newMockMediaService()
	h := NewHandler(mockSvc)

	mockSvc.confirmUploadFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*UploadInfo, error) {
		return &UploadInfo{
			UploadID: uuid.Must(uuid.NewV7()),
			Status:   "processing",
			Context:  "journal_image",
		}, nil
	}

	uploadID := uuid.Must(uuid.NewV7())
	req := httptest.NewRequest(http.MethodPost, "/v1/media/uploads/"+uploadID.String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("upload_id")
	c.SetParamValues(uploadID.String())
	setAuth(c)

	err := h.confirmUpload(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_ConfirmUpload_404(t *testing.T) {
	e := setupEcho()
	mockSvc := newMockMediaService()
	h := NewHandler(mockSvc)

	mockSvc.confirmUploadFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*UploadInfo, error) {
		return nil, &MediaError{Err: ErrUploadNotFound}
	}

	uploadID := uuid.Must(uuid.NewV7())
	req := httptest.NewRequest(http.MethodPost, "/v1/media/uploads/"+uploadID.String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("upload_id")
	c.SetParamValues(uploadID.String())
	setAuth(c)

	err := h.confirmUpload(c)
	var appErr *shared.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %v", err)
	}
}

func TestHandler_ConfirmUpload_409(t *testing.T) {
	e := setupEcho()
	mockSvc := newMockMediaService()
	h := NewHandler(mockSvc)

	mockSvc.confirmUploadFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*UploadInfo, error) {
		return nil, &MediaError{Err: ErrUploadNotConfirmed}
	}

	uploadID := uuid.Must(uuid.NewV7())
	req := httptest.NewRequest(http.MethodPost, "/v1/media/uploads/"+uploadID.String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("upload_id")
	c.SetParamValues(uploadID.String())
	setAuth(c)

	err := h.confirmUpload(c)
	var appErr *shared.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %v", err)
	}
}

func TestHandler_ConfirmUpload_410_expired(t *testing.T) {
	e := setupEcho()
	mockSvc := newMockMediaService()
	h := NewHandler(mockSvc)

	mockSvc.confirmUploadFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*UploadInfo, error) {
		return nil, &MediaError{Err: ErrUploadExpired}
	}

	uploadID := uuid.Must(uuid.NewV7())
	req := httptest.NewRequest(http.MethodPost, "/v1/media/uploads/"+uploadID.String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("upload_id")
	c.SetParamValues(uploadID.String())
	setAuth(c)

	err := h.confirmUpload(c)
	var appErr *shared.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != http.StatusGone {
		t.Errorf("expected 410, got %v", err)
	}
}

func TestHandler_ConfirmUpload_502_storage_error(t *testing.T) {
	e := setupEcho()
	mockSvc := newMockMediaService()
	h := NewHandler(mockSvc)

	mockSvc.confirmUploadFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*UploadInfo, error) {
		return nil, &MediaError{Err: ErrObjectStorageError}
	}

	uploadID := uuid.Must(uuid.NewV7())
	req := httptest.NewRequest(http.MethodPost, "/v1/media/uploads/"+uploadID.String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("upload_id")
	c.SetParamValues(uploadID.String())
	setAuth(c)

	err := h.confirmUpload(c)
	var appErr *shared.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %v", err)
	}
}

// ─── GET /v1/media/uploads/:id ────────────────────────────────────────────────

func TestHandler_GetUpload_200(t *testing.T) {
	e := setupEcho()
	mockSvc := newMockMediaService()
	h := NewHandler(mockSvc)

	publishedAt := time.Now()
	thumbURL := "https://media.example.com/uploads/fam/id/photo__thumb.jpg"
	mockSvc.getUploadFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*UploadInfo, error) {
		return &UploadInfo{
			UploadID:         uuid.Must(uuid.NewV7()),
			Status:           "published",
			Context:          "journal_image",
			ContentType:      "image/jpeg",
			OriginalFilename: "photo.jpg",
			HasThumb:         true,
			URLs:             &UploadURLs{Original: "https://media.example.com/uploads/fam/id/photo.jpg", Thumb: &thumbURL},
			PublishedAt:      &publishedAt,
		}, nil
	}

	uploadID := uuid.Must(uuid.NewV7())
	req := httptest.NewRequest(http.MethodGet, "/v1/media/uploads/"+uploadID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("upload_id")
	c.SetParamValues(uploadID.String())
	setAuth(c)

	err := h.getUpload(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_GetUpload_404(t *testing.T) {
	e := setupEcho()
	mockSvc := newMockMediaService()
	h := NewHandler(mockSvc)

	mockSvc.getUploadFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*UploadInfo, error) {
		return nil, &MediaError{Err: ErrUploadNotFound}
	}

	uploadID := uuid.Must(uuid.NewV7())
	req := httptest.NewRequest(http.MethodGet, "/v1/media/uploads/"+uploadID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("upload_id")
	c.SetParamValues(uploadID.String())
	setAuth(c)

	err := h.getUpload(c)
	var appErr *shared.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %v", err)
	}
}

// ─── Auth Context Missing ─────────────────────────────────────────────────────

func TestHandler_RequestUpload_no_auth(t *testing.T) {
	e := setupEcho()
	h := NewHandler(newMockMediaService())

	body := `{"context":"journal_image","content_type":"image/jpeg","filename":"photo.jpg","size_bytes":1048576}`
	req := httptest.NewRequest(http.MethodPost, "/v1/media/uploads", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Not calling setAuth()

	err := h.requestUpload(c)
	var appErr *shared.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %v", err)
	}
}

func TestHandler_ConfirmUpload_bad_uuid(t *testing.T) {
	e := setupEcho()
	h := NewHandler(newMockMediaService())

	req := httptest.NewRequest(http.MethodPost, "/v1/media/uploads/not-a-uuid/confirm", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("upload_id")
	c.SetParamValues("not-a-uuid")
	setAuth(c)

	err := h.confirmUpload(c)
	var appErr *shared.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %v", err)
	}
}
