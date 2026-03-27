package billing

import (
	"io"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/middleware"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Handler holds the billing HTTP handler dependencies.
type Handler struct {
	svc           BillingService
	webhookSecret string
	db            *gorm.DB
}

// NewHandler creates a new billing Handler.
func NewHandler(svc BillingService, webhookSecret string, db *gorm.DB) *Handler {
	return &Handler{svc: svc, webhookSecret: webhookSecret, db: db}
}

// Register registers all billing routes.
// Authenticated endpoints under authGroup, webhook under hooksGroup. [10-billing §4]
func (h *Handler) Register(authGroup *echo.Group, hooksGroup *echo.Group) {
	b := authGroup.Group("/billing")

	// Phase 1 — Queries
	b.GET("/subscription", h.getSubscription)
	b.GET("/transactions", h.listTransactions)

	// Phase 1 — COPPA
	b.POST("/coppa-verify", h.coppaVerify)

	// Phase 2 — Subscription CRUD
	b.POST("/subscription", h.createSubscription)
	b.PATCH("/subscription", h.updateSubscription)
	b.DELETE("/subscription", h.cancelSubscription)
	b.POST("/subscription/reactivate", h.reactivateSubscription)
	b.POST("/subscription/pause", h.pauseSubscription)
	b.POST("/subscription/resume", h.resumeSubscription)
	b.POST("/subscription/estimate", h.estimateSubscription)

	// Phase 2 — Payment Methods
	b.POST("/payment-methods", h.attachPaymentMethod)
	b.GET("/payment-methods", h.listPaymentMethods)
	b.DELETE("/payment-methods/:id", h.detachPaymentMethod)

	// Phase 2 — Invoices & Payouts
	b.GET("/invoices", h.listInvoices)
	b.GET("/payouts", h.listPayouts)

	// Webhook endpoint (unauthenticated — signature verified by adapter)
	hooksGroup.POST("/hyperswitch/billing", h.hyperswitchWebhook)
}

// ─── Phase 1 Handlers ───────────────────────────────────────────────────────

// getSubscription handles GET /v1/billing/subscription. [10-billing §4.1]
func (h *Handler) getSubscription(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	resp, err := h.svc.GetSubscription(c.Request().Context(), scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// listTransactions handles GET /v1/billing/transactions. [10-billing §4.1]
func (h *Handler) listTransactions(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var params TransactionListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}

	resp, err := h.svc.ListTransactions(c.Request().Context(), params, scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// coppaVerify handles POST /v1/billing/coppa-verify. [10-billing §4.1]
func (h *Handler) coppaVerify(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var cmd CoppaVerificationCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.ProcessCoppaVerification(c.Request().Context(), cmd, scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// hyperswitchWebhook handles POST /hooks/hyperswitch/billing. [10-billing §4.1]
func (h *Handler) hyperswitchWebhook(c echo.Context) error {
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		// Always return 200 to Hyperswitch to prevent retries. [10-billing §14]
		return c.NoContent(http.StatusOK)
	}

	signature := c.Request().Header.Get("X-Webhook-Signature")

	// Errors are logged internally; always return 200 to Hyperswitch.
	_ = h.svc.ProcessHyperswitchWebhook(c.Request().Context(), payload, signature)

	return c.NoContent(http.StatusOK)
}

// ─── Phase 2 Handlers ───────────────────────────────────────────────────────

// createSubscription handles POST /v1/billing/subscription. [10-billing §4.2]
func (h *Handler) createSubscription(c echo.Context) error {
	auth, err := middleware.RequirePrimaryParent(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var cmd CreateSubscriptionCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.CreateSubscription(c.Request().Context(), cmd, scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// updateSubscription handles PATCH /v1/billing/subscription. [10-billing §4.2]
func (h *Handler) updateSubscription(c echo.Context) error {
	auth, err := middleware.RequirePrimaryParent(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var cmd UpdateSubscriptionCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.UpdateSubscription(c.Request().Context(), cmd, scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// cancelSubscription handles DELETE /v1/billing/subscription. [10-billing §4.2]
func (h *Handler) cancelSubscription(c echo.Context) error {
	auth, err := middleware.RequirePrimaryParent(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.CancelSubscription(c.Request().Context(), scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// reactivateSubscription handles POST /v1/billing/subscription/reactivate. [10-billing §4.2]
func (h *Handler) reactivateSubscription(c echo.Context) error {
	auth, err := middleware.RequirePrimaryParent(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.ReactivateSubscription(c.Request().Context(), scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// pauseSubscription handles POST /v1/billing/subscription/pause. [10-billing §4.2]
func (h *Handler) pauseSubscription(c echo.Context) error {
	auth, err := middleware.RequirePrimaryParent(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.PauseSubscription(c.Request().Context(), scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// resumeSubscription handles POST /v1/billing/subscription/resume. [10-billing §4.2]
func (h *Handler) resumeSubscription(c echo.Context) error {
	auth, err := middleware.RequirePrimaryParent(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.ResumeSubscription(c.Request().Context(), scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// estimateSubscription handles POST /v1/billing/subscription/estimate. [10-billing §4.2]
func (h *Handler) estimateSubscription(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var query EstimateSubscriptionQuery
	if err := c.Bind(&query); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(query); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.EstimateSubscription(c.Request().Context(), query, scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// attachPaymentMethod handles POST /v1/billing/payment-methods. [10-billing §4.2]
func (h *Handler) attachPaymentMethod(c echo.Context) error {
	auth, err := middleware.RequirePrimaryParent(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var cmd AttachPaymentMethodCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AttachPaymentMethod(c.Request().Context(), cmd, scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listPaymentMethods handles GET /v1/billing/payment-methods. [10-billing §4.2]
func (h *Handler) listPaymentMethods(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	resp, err := h.svc.ListPaymentMethods(c.Request().Context(), scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// detachPaymentMethod handles DELETE /v1/billing/payment-methods/:id. [10-billing §4.2]
func (h *Handler) detachPaymentMethod(c echo.Context) error {
	auth, err := middleware.RequirePrimaryParent(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	paymentMethodID := c.Param("id")
	if paymentMethodID == "" {
		return shared.ErrBadRequest("payment method ID is required")
	}

	if err := h.svc.DetachPaymentMethod(c.Request().Context(), paymentMethodID, scope); err != nil {
		return mapBillingError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// listInvoices handles GET /v1/billing/invoices. [10-billing §4.2]
func (h *Handler) listInvoices(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var params InvoiceListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}

	resp, err := h.svc.ListInvoices(c.Request().Context(), params, scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// listPayouts handles GET /v1/billing/payouts. [10-billing §4.2]
func (h *Handler) listPayouts(c echo.Context) error {
	creator, err := middleware.RequireCreator(c, h.db)
	if err != nil {
		return err
	}

	var params PayoutListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}

	resp, err := h.svc.ListPayouts(c.Request().Context(), params, creator.CreatorID)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}
