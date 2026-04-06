package notify

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Handler holds the notify HTTP handler dependencies.
type Handler struct {
	svc               NotificationService
	unsubscribeSecret string
}

// NewHandler creates a new notify Handler.
func NewHandler(svc NotificationService, unsubscribeSecret string) *Handler {
	return &Handler{svc: svc, unsubscribeSecret: unsubscribeSecret}
}

// Register registers all notification routes.
// Authenticated endpoints under authGroup, unsubscribe is public. [08-notify §4]
func (h *Handler) Register(authGroup *echo.Group, publicGroup *echo.Group) {
	n := authGroup.Group("/notifications")

	// Notification Feed
	n.GET("", h.listNotifications)
	n.GET("/unread-count", h.getUnreadCount)
	n.PATCH("/:id/read", h.markRead)
	n.PATCH("/read-all", h.markAllRead)

	// Preferences
	n.GET("/preferences", h.getPreferences)
	n.PATCH("/preferences", h.updatePreferences)

	// Unsubscribe (unauthenticated — signed token)
	publicGroup.GET("/notifications/unsubscribe", h.unsubscribe)
}

// listNotifications godoc
//
// @Summary     List notifications
// @Tags        notifications
// @Produce     json
// @Security    BearerAuth
// @Param       cursor      query string false "Pagination cursor"
// @Param       limit       query int    false "Results per page"
// @Param       category    query string false "Filter by category"
// @Param       unread_only query bool   false "Show only unread notifications"
// @Success     200 {object} NotificationListResponse
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Failure     500 {object} shared.AppError
// @Router      /notifications [get]
func (h *Handler) listNotifications(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var params NotificationListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.ListNotifications(c.Request().Context(), params, &scope)
	if err != nil {
		return mapNotifyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getUnreadCount godoc
//
// @Summary     Get unread notification count
// @Tags        notifications
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string]int64
// @Failure     401 {object} shared.AppError
// @Failure     500 {object} shared.AppError
// @Router      /notifications/unread-count [get]
func (h *Handler) getUnreadCount(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	count, err := h.svc.GetUnreadCount(c.Request().Context(), &scope)
	if err != nil {
		return mapNotifyError(err)
	}
	return c.JSON(http.StatusOK, map[string]int64{"count": count})
}

// markRead godoc
//
// @Summary     Mark a notification as read
// @Tags        notifications
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Notification ID (UUID)"
// @Success     200 {object} NotificationResponse
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Failure     500 {object} shared.AppError
// @Router      /notifications/{id}/read [patch]
func (h *Handler) markRead(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid notification ID")
	}

	resp, err := h.svc.MarkRead(c.Request().Context(), notificationID, &scope)
	if err != nil {
		return mapNotifyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// markAllRead godoc
//
// @Summary     Mark all notifications as read
// @Tags        notifications
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body MarkAllReadRequest false "Optional category filter"
// @Success     200 {object} MarkAllReadResponse
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Failure     500 {object} shared.AppError
// @Router      /notifications/read-all [patch]
func (h *Handler) markAllRead(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var req MarkAllReadRequest
	if err := c.Bind(&req); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return shared.ValidationError(err)
	}

	count, err := h.svc.MarkAllRead(c.Request().Context(), &scope, req.Category)
	if err != nil {
		return mapNotifyError(err)
	}
	return c.JSON(http.StatusOK, MarkAllReadResponse{UpdatedCount: count})
}

// getPreferences godoc
//
// @Summary     Get notification preferences
// @Tags        notifications
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  PreferenceResponse
// @Failure     401 {object} shared.AppError
// @Failure     500 {object} shared.AppError
// @Router      /notifications/preferences [get]
func (h *Handler) getPreferences(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	resp, err := h.svc.GetPreferences(c.Request().Context(), &scope)
	if err != nil {
		return mapNotifyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updatePreferences godoc
//
// @Summary     Update notification preferences
// @Tags        notifications
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body UpdatePreferencesCommand true "Preference updates"
// @Success     200 {array}  PreferenceResponse
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Failure     500 {object} shared.AppError
// @Router      /notifications/preferences [patch]
func (h *Handler) updatePreferences(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var cmd UpdatePreferencesCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.UpdatePreferences(c.Request().Context(), cmd, &scope)
	if err != nil {
		return mapNotifyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// unsubscribe godoc
//
// @Summary     Unsubscribe from email notifications
// @Tags        notifications
// @Produce     html
// @Param       token query string true "Signed unsubscribe token"
// @Success     200 {string} string "HTML confirmation page"
// @Failure     400 {string} string "HTML error page"
// @Failure     500 {string} string "HTML error page"
// @Router      /notifications/unsubscribe [get]
func (h *Handler) unsubscribe(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return c.HTML(http.StatusBadRequest, unsubscribeErrorHTML)
	}

	if err := h.svc.ProcessUnsubscribe(c.Request().Context(), token); err != nil {
		return c.HTML(http.StatusOK, unsubscribeErrorHTML)
	}
	return c.HTML(http.StatusOK, unsubscribeSuccessHTML)
}

const unsubscribeSuccessHTML = `<!DOCTYPE html>
<html><head><title>Unsubscribed</title></head>
<body style="font-family:sans-serif;text-align:center;padding:40px">
<h2>You've been unsubscribed</h2>
<p>You will no longer receive these email notifications.</p>
<p>You can re-enable them anytime in your notification preferences.</p>
</body></html>`

const unsubscribeErrorHTML = `<!DOCTYPE html>
<html><head><title>Unsubscribe Error</title></head>
<body style="font-family:sans-serif;text-align:center;padding:40px">
<h2>Unable to unsubscribe</h2>
<p>This unsubscribe link is invalid or has expired.</p>
<p>Please manage your notification preferences from your account settings.</p>
</body></html>`
