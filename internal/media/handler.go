package media

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Handler handles HTTP requests for the media domain. [09-media §4]
type Handler struct {
	svc MediaService
}

// NewHandler constructs a media Handler.
func NewHandler(svc MediaService) *Handler {
	return &Handler{svc: svc}
}

// Register registers media routes on the authenticated route group. [09-media §4.1]
func (h *Handler) Register(authGroup *echo.Group) {
	g := authGroup.Group("/media")
	g.POST("/uploads", h.requestUpload)
	g.POST("/uploads/:upload_id/confirm", h.confirmUpload)
	g.POST("/uploads/:upload_id/reprocess", h.reprocessUpload)
	g.GET("/uploads/:upload_id", h.getUpload)
	g.DELETE("/uploads/:upload_id", h.deleteUpload)
	g.GET("/uploads", h.listUploads)
}

// ─── POST /v1/media/uploads ───────────────────────────────────────────────────

func (h *Handler) requestUpload(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}

	var cmd RequestUploadCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.RequestUpload(c.Request().Context(), &RequestUploadInput{
		FamilyID:    auth.FamilyID,
		UploadedBy:  auth.ParentID,
		Context:     cmd.Context,
		ContentType: cmd.ContentType,
		Filename:    cmd.Filename,
		SizeBytes:   cmd.SizeBytes,
	})
	if err != nil {
		return mapMediaError(err)
	}

	return c.JSON(http.StatusCreated, resp)
}

// ─── POST /v1/media/uploads/:upload_id/confirm ───────────────────────────────

func (h *Handler) confirmUpload(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}

	uploadID, err := uuid.Parse(c.Param("upload_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid upload ID")
	}

	info, err := h.svc.ConfirmUpload(c.Request().Context(), uploadID, auth.FamilyID)
	if err != nil {
		return mapMediaError(err)
	}

	return c.JSON(http.StatusOK, info)
}

// ─── GET /v1/media/uploads/:upload_id ─────────────────────────────────────────

func (h *Handler) getUpload(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}

	uploadID, err := uuid.Parse(c.Param("upload_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid upload ID")
	}

	info, err := h.svc.GetUpload(c.Request().Context(), uploadID, auth.FamilyID)
	if err != nil {
		return mapMediaError(err)
	}

	return c.JSON(http.StatusOK, info)
}

// ─── DELETE /v1/media/uploads/:upload_id ──────────────────────────────────────

func (h *Handler) deleteUpload(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	uploadID, err := uuid.Parse(c.Param("upload_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid upload ID")
	}
	if err := h.svc.DeleteUpload(c.Request().Context(), uploadID, auth.FamilyID); err != nil {
		return mapMediaError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── GET /v1/media/uploads ────────────────────────────────────────────────────

func (h *Handler) listUploads(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}

	var limit uint32 = 20
	if l := c.QueryParam("limit"); l != "" {
		var n uint32
		if _, err := fmt.Sscanf(l, "%d", &n); err == nil && n > 0 {
			limit = n
		}
	}

	var afterID *uuid.UUID
	if a := c.QueryParam("after"); a != "" {
		id, err := uuid.Parse(a)
		if err != nil {
			return shared.ErrBadRequest("invalid after cursor")
		}
		afterID = &id
	}

	resp, err := h.svc.ListUploads(c.Request().Context(), auth.FamilyID, limit, afterID)
	if err != nil {
		return mapMediaError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── POST /v1/media/uploads/:upload_id/reprocess ──────────────────────────────

func (h *Handler) reprocessUpload(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}

	uploadID, err := uuid.Parse(c.Param("upload_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid upload ID")
	}

	scope := shared.NewFamilyScopeFromAuth(auth)
	if err := h.svc.ReprocessUpload(c.Request().Context(), scope, uploadID); err != nil {
		return mapMediaError(err)
	}
	return c.NoContent(http.StatusAccepted)
}

// ─── Error Mapping ────────────────────────────────────────────────────────────

// mapMediaError converts domain errors to AppError for HTTP responses. [09-media §15.1]
func mapMediaError(err error) *shared.AppError {
	var mediaErr *MediaError
	if errors.As(err, &mediaErr) {
		return mediaErr.toAppError()
	}
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return shared.ErrInternal(err)
}
