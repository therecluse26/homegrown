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

// requestUpload godoc
//
// @Summary     Request a media upload
// @Tags        media
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body RequestUploadCommand true "Upload details"
// @Success     201 {object} UploadResponse
// @Failure     401 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /media/uploads [post]
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

// confirmUpload godoc
//
// @Summary     Confirm a media upload
// @Tags        media
// @Produce     json
// @Security    BearerAuth
// @Param       upload_id path string true "Upload ID"
// @Success     200 {object} UploadInfo
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /media/uploads/{upload_id}/confirm [post]
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

// getUpload godoc
//
// @Summary     Get upload details
// @Tags        media
// @Produce     json
// @Security    BearerAuth
// @Param       upload_id path string true "Upload ID"
// @Success     200 {object} UploadInfo
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /media/uploads/{upload_id} [get]
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

// deleteUpload godoc
//
// @Summary     Delete an upload
// @Tags        media
// @Security    BearerAuth
// @Param       upload_id path string true "Upload ID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /media/uploads/{upload_id} [delete]
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

// listUploads godoc
//
// @Summary     List uploads for the family
// @Tags        media
// @Produce     json
// @Security    BearerAuth
// @Param       limit  query  int     false  "Max items to return (default 20, max 100)"
// @Param       after  query  string  false  "Cursor: upload ID to start after"
// @Success     200 {object} UploadListResponse
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Router      /media/uploads [get]
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

// reprocessUpload godoc
//
// @Summary     Reprocess an upload
// @Tags        media
// @Security    BearerAuth
// @Param       upload_id path string true "Upload ID"
// @Success     202
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /media/uploads/{upload_id}/reprocess [post]
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
