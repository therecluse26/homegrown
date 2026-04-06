package billing

import (
	"errors"
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

// getSubscription godoc
//
// @Summary     Get current subscription
// @Tags        billing
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} SubscriptionResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/subscription [get]
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

// listTransactions godoc
//
// @Summary     List billing transactions
// @Tags        billing
// @Produce     json
// @Security    BearerAuth
// @Param       type query string false "Filter by type" Enums(subscription,purchase,payout,refund)
// @Param       from query string false "Start date (ISO 8601)"
// @Param       to   query string false "End date (ISO 8601)"
// @Param       page query int    false "Page number"
// @Success     200 {object} TransactionListResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/transactions [get]
func (h *Handler) listTransactions(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var params TransactionListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.ListTransactions(c.Request().Context(), params, scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// coppaVerify godoc
//
// @Summary     Verify COPPA compliance
// @Tags        billing
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body CoppaVerificationCommand true "COPPA verification"
// @Success     200 {object} CoppaVerificationResult
// @Failure     401 {object} shared.AppError
// @Router      /billing/coppa-verify [post]
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

// hyperswitchWebhook handles POST /hooks/hyperswitch/billing. [10-billing §4.1, P1-1]
func (h *Handler) hyperswitchWebhook(c echo.Context) error {
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	signature := c.Request().Header.Get("X-Webhook-Signature")

	// Return appropriate status codes so the payment provider retries on transient failure. [P1-1]
	// Signature failures → 400 (permanent, don't retry). Processing errors → 500 (transient, retry).
	if err := h.svc.ProcessHyperswitchWebhook(c.Request().Context(), payload, signature); err != nil {
		if errors.Is(err, ErrInvalidWebhookSignature) {
			return c.NoContent(http.StatusBadRequest)
		}
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

// ─── Phase 2 Handlers ───────────────────────────────────────────────────────

// createSubscription godoc
//
// @Summary     Create a subscription
// @Tags        billing
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body CreateSubscriptionCommand true "Create subscription"
// @Success     201 {object} SubscriptionResponse
// @Failure     401 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /billing/subscription [post]
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

// updateSubscription godoc
//
// @Summary     Update subscription plan
// @Tags        billing
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body UpdateSubscriptionCommand true "Update subscription"
// @Success     200 {object} SubscriptionResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/subscription [patch]
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

// cancelSubscription godoc
//
// @Summary     Cancel subscription
// @Tags        billing
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} SubscriptionResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/subscription [delete]
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

// reactivateSubscription godoc
//
// @Summary     Reactivate cancelled subscription
// @Tags        billing
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} SubscriptionResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/subscription/reactivate [post]
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

// pauseSubscription godoc
//
// @Summary     Pause subscription
// @Tags        billing
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} SubscriptionResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/subscription/pause [post]
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

// resumeSubscription godoc
//
// @Summary     Resume paused subscription
// @Tags        billing
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} SubscriptionResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/subscription/resume [post]
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

// estimateSubscription godoc
//
// @Summary     Estimate subscription cost
// @Tags        billing
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body EstimateSubscriptionQuery true "Estimate query"
// @Success     200 {object} EstimateResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/subscription/estimate [post]
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

// attachPaymentMethod godoc
//
// @Summary     Attach a payment method
// @Tags        billing
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body AttachPaymentMethodCommand true "Attach payment method"
// @Success     201 {object} PaymentMethodResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/payment-methods [post]
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

// listPaymentMethods godoc
//
// @Summary     List payment methods
// @Tags        billing
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} PaymentMethodResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/payment-methods [get]
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

// detachPaymentMethod godoc
//
// @Summary     Remove a payment method
// @Tags        billing
// @Security    BearerAuth
// @Param       id path string true "Payment method ID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Router      /billing/payment-methods/{id} [delete]
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

// listInvoices godoc
//
// @Summary     List invoices
// @Tags        billing
// @Produce     json
// @Security    BearerAuth
// @Param       status query string false "Filter by status"
// @Param       cursor query string false "Pagination cursor"
// @Param       limit  query int    false "Results per page"
// @Success     200 {object} InvoiceListResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/invoices [get]
func (h *Handler) listInvoices(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var params InvoiceListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.ListInvoices(c.Request().Context(), params, scope)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// listPayouts godoc
//
// @Summary     List creator payouts
// @Tags        billing
// @Produce     json
// @Security    BearerAuth
// @Param       status query string false "Filter by status"
// @Param       cursor query string false "Pagination cursor"
// @Param       limit  query int    false "Results per page"
// @Success     200 {object} PayoutListResponse
// @Failure     401 {object} shared.AppError
// @Router      /billing/payouts [get]
func (h *Handler) listPayouts(c echo.Context) error {
	creator, err := middleware.RequireCreator(c, h.db)
	if err != nil {
		return err
	}

	var params PayoutListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.ListPayouts(c.Request().Context(), params, creator.CreatorID)
	if err != nil {
		return mapBillingError(err)
	}
	return c.JSON(http.StatusOK, resp)
}
