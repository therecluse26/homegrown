package billing

// Handler tests for the billing domain — exercise HTTP handler behaviour
// without a database by injecting a pre-built AuthContext. [10-billing §4]

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// ─── Mock BillingService ──────────────────────────────────────────────────────

type mockBillingService struct {
	getSubscriptionFn      func(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error)
	listTransactionsFn     func(ctx context.Context, params TransactionListParams, scope shared.FamilyScope) (*TransactionListResponse, error)
	estimateSubscriptionFn func(ctx context.Context, query EstimateSubscriptionQuery, scope shared.FamilyScope) (*EstimateResponse, error)
	processWebhookFn       func(ctx context.Context, payload []byte, sig string) error
}

func (m *mockBillingService) GetSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error) {
	if m.getSubscriptionFn != nil {
		return m.getSubscriptionFn(ctx, scope)
	}
	return &SubscriptionResponse{Tier: "free"}, nil
}
func (m *mockBillingService) ListTransactions(ctx context.Context, params TransactionListParams, scope shared.FamilyScope) (*TransactionListResponse, error) {
	if m.listTransactionsFn != nil {
		return m.listTransactionsFn(ctx, params, scope)
	}
	return &TransactionListResponse{Transactions: []TransactionResponse{}}, nil
}
func (m *mockBillingService) EstimateSubscription(ctx context.Context, query EstimateSubscriptionQuery, scope shared.FamilyScope) (*EstimateResponse, error) {
	if m.estimateSubscriptionFn != nil {
		return m.estimateSubscriptionFn(ctx, query, scope)
	}
	return &EstimateResponse{}, nil
}
func (m *mockBillingService) ProcessHyperswitchWebhook(ctx context.Context, payload []byte, sig string) error {
	if m.processWebhookFn != nil {
		return m.processWebhookFn(ctx, payload, sig)
	}
	return nil
}
func (m *mockBillingService) CreateSubscription(_ context.Context, _ CreateSubscriptionCommand, _ shared.FamilyScope) (*SubscriptionResponse, error) {
	return &SubscriptionResponse{}, nil
}
func (m *mockBillingService) UpdateSubscription(_ context.Context, _ UpdateSubscriptionCommand, _ shared.FamilyScope) (*SubscriptionResponse, error) {
	return &SubscriptionResponse{}, nil
}
func (m *mockBillingService) CancelSubscription(_ context.Context, _ shared.FamilyScope) (*SubscriptionResponse, error) {
	return &SubscriptionResponse{}, nil
}
func (m *mockBillingService) ReactivateSubscription(_ context.Context, _ shared.FamilyScope) (*SubscriptionResponse, error) {
	return &SubscriptionResponse{}, nil
}
func (m *mockBillingService) PauseSubscription(_ context.Context, _ shared.FamilyScope) (*SubscriptionResponse, error) {
	return &SubscriptionResponse{}, nil
}
func (m *mockBillingService) ResumeSubscription(_ context.Context, _ shared.FamilyScope) (*SubscriptionResponse, error) {
	return &SubscriptionResponse{}, nil
}
func (m *mockBillingService) AttachPaymentMethod(_ context.Context, _ AttachPaymentMethodCommand, _ shared.FamilyScope) (*PaymentMethodResponse, error) {
	return &PaymentMethodResponse{}, nil
}
func (m *mockBillingService) ListPaymentMethods(_ context.Context, _ shared.FamilyScope) ([]PaymentMethodResponse, error) {
	return []PaymentMethodResponse{}, nil
}
func (m *mockBillingService) DetachPaymentMethod(_ context.Context, _ string, _ shared.FamilyScope) error {
	return nil
}
func (m *mockBillingService) ListInvoices(_ context.Context, _ InvoiceListParams, _ shared.FamilyScope) (*InvoiceListResponse, error) {
	return &InvoiceListResponse{}, nil
}
func (m *mockBillingService) ListPayouts(_ context.Context, _ PayoutListParams, _ uuid.UUID) (*PayoutListResponse, error) {
	return &PayoutListResponse{}, nil
}
func (m *mockBillingService) ProcessCoppaVerification(_ context.Context, _ CoppaVerificationCommand, _ shared.FamilyScope) (*CoppaVerificationResult, error) {
	return &CoppaVerificationResult{Verified: true}, nil
}
func (m *mockBillingService) HandleFamilyDeletionScheduled(_ context.Context, _ FamilyDeletionScheduledEvent) error {
	return nil
}
func (m *mockBillingService) HandlePrimaryParentTransferred(_ context.Context, _ PrimaryParentTransferredEvent) error {
	return nil
}
func (m *mockBillingService) HandlePurchaseCompleted(_ context.Context, _ PurchaseCompletedEvent) error {
	return nil
}
func (m *mockBillingService) HandlePurchaseRefunded(_ context.Context, _ PurchaseRefundedEvent) error {
	return nil
}

// Compile-time check.
var _ BillingService = (*mockBillingService)(nil)

// ─── Test Helpers ─────────────────────────────────────────────────────────────

func setupBillingHandlerTest(svc BillingService) (*echo.Echo, *Handler) {
	e := echo.New()
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	// nil db: only used by listPayouts via RequireCreator, not tested here.
	h := NewHandler(svc, "test-secret", nil)
	return e, h
}

func setBillingTestAuth(c echo.Context) {
	shared.SetAuthContext(c, &shared.AuthContext{
		ParentID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		FamilyID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		IsPrimaryParent:  true,
		SubscriptionTier: shared.SubscriptionTierFree,
	})
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestHandler_GetSubscription_200(t *testing.T) {
	svc := &mockBillingService{
		getSubscriptionFn: func(_ context.Context, _ shared.FamilyScope) (*SubscriptionResponse, error) {
			return &SubscriptionResponse{Tier: "free"}, nil
		},
	}
	e, h := setupBillingHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/billing/subscription", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setBillingTestAuth(c)

	if err := h.getSubscription(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_GetSubscription_MissingAuth_Errors(t *testing.T) {
	e, h := setupBillingHandlerTest(&mockBillingService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/billing/subscription", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// no auth context set

	if err := h.getSubscription(c); err == nil {
		t.Fatal("expected error for missing auth")
	}
}

func TestHandler_ListTransactions_200(t *testing.T) {
	svc := &mockBillingService{
		listTransactionsFn: func(_ context.Context, _ TransactionListParams, _ shared.FamilyScope) (*TransactionListResponse, error) {
			return &TransactionListResponse{Transactions: []TransactionResponse{}}, nil
		},
	}
	e, h := setupBillingHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/billing/transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setBillingTestAuth(c)

	if err := h.listTransactions(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_HyperswitchWebhook_AlwaysReturns200(t *testing.T) {
	// Webhook endpoint must always return 200 to prevent Hyperswitch retries. [10-billing §14]
	svc := &mockBillingService{
		processWebhookFn: func(_ context.Context, _ []byte, _ string) error {
			return shared.ErrBadRequest("simulated internal error") // ignored by design
		},
	}
	e, h := setupBillingHandlerTest(svc)
	req := httptest.NewRequest(http.MethodPost, "/hooks/hyperswitch/billing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.hyperswitchWebhook(c); err != nil {
		t.Fatalf("webhook handler must not return an error, got: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d (webhook must always 200)", rec.Code)
	}
}

func TestHandler_ListPaymentMethods_200(t *testing.T) {
	e, h := setupBillingHandlerTest(&mockBillingService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/billing/payment-methods", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setBillingTestAuth(c)

	if err := h.listPaymentMethods(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}
