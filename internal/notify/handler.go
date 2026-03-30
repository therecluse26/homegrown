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

// listNotifications handles GET /v1/notifications. [08-notify §4.1]
func (h *Handler) listNotifications(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var params NotificationListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}

	resp, err := h.svc.ListNotifications(c.Request().Context(), params, &scope)
	if err != nil {
		return mapNotifyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getUnreadCount handles GET /v1/notifications/unread-count. [08-notify §4.1]
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

// markRead handles PATCH /v1/notifications/:id/read. [08-notify §4.1]
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

// markAllRead handles PATCH /v1/notifications/read-all. [08-notify §4.1]
func (h *Handler) markAllRead(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var req MarkAllReadRequest
	if err := c.Bind(&req); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}

	count, err := h.svc.MarkAllRead(c.Request().Context(), &scope, req.Category)
	if err != nil {
		return mapNotifyError(err)
	}
	return c.JSON(http.StatusOK, MarkAllReadResponse{UpdatedCount: count})
}

// getPreferences handles GET /v1/notifications/preferences. [08-notify §4.1]
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

// updatePreferences handles PATCH /v1/notifications/preferences. [08-notify §4.1]
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

// unsubscribe handles GET /v1/notifications/unsubscribe?token=<signed_token>. [08-notify §4.1]
// Returns HTML — email clients may not support JavaScript. [CAN-SPAM]
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
