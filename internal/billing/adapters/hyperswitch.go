package adapters

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/billing"
)

// HyperswitchSubscriptionAdapter wraps the Hyperswitch REST API for subscription + billing flows.
// Uses the billing-specific Hyperswitch business profile, separate from mkt::'s marketplace profile.
// [10-billing §7, 07-mkt §18.5]
type HyperswitchSubscriptionAdapter struct {
	baseURL    string
	apiKey     string
	profileID  string
	webhookKey string
	client     *http.Client
}

// NewHyperswitchSubscriptionAdapter creates a new billing Hyperswitch adapter.
// Returns a noop adapter if baseURL or apiKey are empty (Hyperswitch not configured).
func NewHyperswitchSubscriptionAdapter(baseURL, apiKey, profileID, webhookKey string) billing.SubscriptionPaymentAdapter {
	if baseURL == "" || apiKey == "" {
		return &noopSubscriptionAdapter{}
	}
	return &HyperswitchSubscriptionAdapter{
		baseURL:    baseURL,
		apiKey:     apiKey,
		profileID:  profileID,
		webhookKey: webhookKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ─── Customer Management ────────────────────────────────────────────────────

func (a *HyperswitchSubscriptionAdapter) CreateCustomer(_ context.Context, email string, name string, metadata map[string]string) (string, error) {
	body := map[string]any{
		"email":    email,
		"name":     name,
		"metadata": metadata,
	}

	resp, err := a.doRequest(http.MethodPost, "/customers", body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", billing.ErrPaymentAdapterUnavailable
	}

	var result struct {
		CustomerID string `json:"customer_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return result.CustomerID, nil
}

func (a *HyperswitchSubscriptionAdapter) UpdateCustomer(_ context.Context, customerID string, email string, name string) error {
	body := map[string]any{
		"email": email,
		"name":  name,
	}

	resp, err := a.doRequest(http.MethodPost, "/customers/"+customerID, body)
	if err != nil {
		return fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return billing.ErrPaymentAdapterUnavailable
	}
	return nil
}

// ─── Subscriptions ──────────────────────────────────────────────────────────

func (a *HyperswitchSubscriptionAdapter) CreateSubscription(_ context.Context, customerID string, priceID string, paymentMethodID string, metadata map[string]string) (*billing.HyperswitchSubscription, error) {
	body := map[string]any{
		"customer_id":       customerID,
		"price_id":          priceID,
		"payment_method_id": paymentMethodID,
		"profile_id":        a.profileID,
		"metadata":          metadata,
	}

	resp, err := a.doRequest(http.MethodPost, "/subscriptions", body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		// success
	case http.StatusUnprocessableEntity:
		return nil, billing.ErrPaymentDeclined
	default:
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var sub billing.HyperswitchSubscription
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return &sub, nil
}

func (a *HyperswitchSubscriptionAdapter) UpdateSubscription(_ context.Context, subscriptionID string, newPriceID string) (*billing.HyperswitchSubscription, error) {
	body := map[string]any{
		"price_id": newPriceID,
	}

	resp, err := a.doRequest(http.MethodPost, "/subscriptions/"+subscriptionID, body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var sub billing.HyperswitchSubscription
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return &sub, nil
}

func (a *HyperswitchSubscriptionAdapter) CancelSubscription(_ context.Context, subscriptionID string) (*billing.HyperswitchSubscription, error) {
	body := map[string]any{
		"cancel_option": "end_of_term",
	}

	resp, err := a.doRequest(http.MethodPost, "/subscriptions/"+subscriptionID+"/cancel", body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var sub billing.HyperswitchSubscription
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return &sub, nil
}

func (a *HyperswitchSubscriptionAdapter) PauseSubscription(_ context.Context, subscriptionID string) (*billing.HyperswitchSubscription, error) {
	resp, err := a.doRequest(http.MethodPost, "/subscriptions/"+subscriptionID+"/pause", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var sub billing.HyperswitchSubscription
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return &sub, nil
}

func (a *HyperswitchSubscriptionAdapter) ResumeSubscription(_ context.Context, subscriptionID string) (*billing.HyperswitchSubscription, error) {
	resp, err := a.doRequest(http.MethodPost, "/subscriptions/"+subscriptionID+"/resume", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var sub billing.HyperswitchSubscription
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return &sub, nil
}

func (a *HyperswitchSubscriptionAdapter) ReactivateSubscription(_ context.Context, subscriptionID string) (*billing.HyperswitchSubscription, error) {
	resp, err := a.doRequest(http.MethodPost, "/subscriptions/"+subscriptionID+"/reactivate", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var sub billing.HyperswitchSubscription
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return &sub, nil
}

func (a *HyperswitchSubscriptionAdapter) EstimateSubscription(_ context.Context, customerID string, priceID string, currentSubscriptionID *string) (*billing.HyperswitchEstimate, error) {
	body := map[string]any{
		"customer_id": customerID,
		"price_id":    priceID,
	}
	if currentSubscriptionID != nil {
		body["subscription_id"] = *currentSubscriptionID
	}

	resp, err := a.doRequest(http.MethodPost, "/subscriptions/estimate", body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var estimate billing.HyperswitchEstimate
	if err := json.NewDecoder(resp.Body).Decode(&estimate); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return &estimate, nil
}

// ─── Payment Methods ────────────────────────────────────────────────────────

func (a *HyperswitchSubscriptionAdapter) CreateSetupIntent(_ context.Context, customerID string) (*billing.SetupIntentResponse, error) {
	body := map[string]any{
		"customer_id": customerID,
		"profile_id":  a.profileID,
	}

	resp, err := a.doRequest(http.MethodPost, "/setup_intents", body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var result billing.SetupIntentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return &result, nil
}

func (a *HyperswitchSubscriptionAdapter) ListPaymentMethods(_ context.Context, customerID string) ([]billing.HyperswitchPaymentMethod, error) {
	resp, err := a.doRequest(http.MethodGet, "/customers/"+customerID+"/payment_methods", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var methods []billing.HyperswitchPaymentMethod
	if err := json.NewDecoder(resp.Body).Decode(&methods); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return methods, nil
}

func (a *HyperswitchSubscriptionAdapter) DetachPaymentMethod(_ context.Context, paymentMethodID string) error {
	resp, err := a.doRequest(http.MethodDelete, "/payment_methods/"+paymentMethodID, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return billing.ErrPaymentMethodNotFound
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return billing.ErrPaymentAdapterUnavailable
	}
	return nil
}

// ─── One-Time Payments (COPPA) ──────────────────────────────────────────────

func (a *HyperswitchSubscriptionAdapter) ProcessMicroCharge(_ context.Context, customerID string, paymentMethodID string, amountCents int64, description string, metadata map[string]string) (string, string, error) {
	// Step 1: Create one-time payment
	payBody := map[string]any{
		"customer_id":       customerID,
		"payment_method_id": paymentMethodID,
		"amount":            amountCents,
		"currency":          "usd",
		"description":       description,
		"profile_id":        a.profileID,
		"metadata":          metadata,
		"confirm":           true,
	}

	payResp, err := a.doRequest(http.MethodPost, "/payments", payBody)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = payResp.Body.Close() }()

	if payResp.StatusCode == http.StatusUnprocessableEntity {
		return "", "", billing.ErrPaymentDeclined
	}
	if payResp.StatusCode != http.StatusOK && payResp.StatusCode != http.StatusCreated {
		return "", "", billing.ErrPaymentAdapterUnavailable
	}

	var payResult struct {
		PaymentID string `json:"payment_id"`
	}
	if err := json.NewDecoder(payResp.Body).Decode(&payResult); err != nil {
		return "", "", fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}

	// Step 2: Immediately refund
	refundBody := map[string]any{
		"payment_id": payResult.PaymentID,
		"amount":     amountCents,
		"reason":     "COPPA micro-charge verification refund",
	}

	refundResp, err := a.doRequest(http.MethodPost, "/refunds", refundBody)
	if err != nil {
		// Charge succeeded but refund failed — return charge ID and empty refund.
		// Service layer logs this and still returns success. [10-billing §13]
		slog.Error("COPPA refund failed after charge", "payment_id", payResult.PaymentID, "error", err)
		return payResult.PaymentID, "", nil
	}
	defer func() { _ = refundResp.Body.Close() }()

	if refundResp.StatusCode != http.StatusOK && refundResp.StatusCode != http.StatusCreated {
		slog.Error("COPPA refund HTTP error", "payment_id", payResult.PaymentID, "status", refundResp.StatusCode)
		return payResult.PaymentID, "", nil
	}

	var refundResult struct {
		RefundID string `json:"refund_id"`
	}
	if err := json.NewDecoder(refundResp.Body).Decode(&refundResult); err != nil {
		slog.Error("COPPA refund decode error", "payment_id", payResult.PaymentID, "error", err)
		return payResult.PaymentID, "", nil
	}

	return payResult.PaymentID, refundResult.RefundID, nil
}

// ─── Invoices ───────────────────────────────────────────────────────────────

func (a *HyperswitchSubscriptionAdapter) ListInvoices(_ context.Context, customerID string, limit uint32) ([]billing.HyperswitchInvoice, error) {
	url := fmt.Sprintf("/customers/%s/invoices?limit=%d", customerID, limit)
	resp, err := a.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var invoices []billing.HyperswitchInvoice
	if err := json.NewDecoder(resp.Body).Decode(&invoices); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return invoices, nil
}

// ─── Payouts ────────────────────────────────────────────────────────────────

func (a *HyperswitchSubscriptionAdapter) CreatePayout(_ context.Context, paymentAccountID string, amountCents int64, currency string, metadata map[string]string) (*billing.HyperswitchPayout, error) {
	body := map[string]any{
		"merchant_id": paymentAccountID,
		"amount":      amountCents,
		"currency":    currency,
		"metadata":    metadata,
	}

	resp, err := a.doRequest(http.MethodPost, "/payouts/create", body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, billing.ErrPaymentAdapterUnavailable
	}

	var payout billing.HyperswitchPayout
	if err := json.NewDecoder(resp.Body).Decode(&payout); err != nil {
		return nil, fmt.Errorf("%w: %v", billing.ErrPaymentAdapterUnavailable, err)
	}
	return &payout, nil
}

// ─── Webhooks ───────────────────────────────────────────────────────────────

func (a *HyperswitchSubscriptionAdapter) VerifyWebhook(_ context.Context, payload []byte, signature string) (bool, error) {
	if a.webhookKey == "" {
		return false, billing.ErrInvalidWebhookSignature
	}

	mac := hmac.New(sha256.New, []byte(a.webhookKey))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return false, billing.ErrInvalidWebhookSignature
	}
	return true, nil
}

func (a *HyperswitchSubscriptionAdapter) ParseWebhookEvent(_ context.Context, payload []byte) (*billing.BillingWebhookEvent, error) {
	var raw struct {
		Type    string `json:"type"`
		Content json.RawMessage `json:"content"`
	}

	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("malformed webhook payload: %w", err)
	}

	event := &billing.BillingWebhookEvent{Type: raw.Type}

	switch raw.Type {
	case "subscription.created":
		var sub billing.HyperswitchSubscription
		if err := json.Unmarshal(raw.Content, &sub); err != nil {
			return nil, fmt.Errorf("malformed subscription.created payload: %w", err)
		}
		event.SubscriptionCreated = &billing.BillingWebhookSubscriptionCreated{Subscription: sub}

	case "subscription.updated":
		var sub billing.HyperswitchSubscription
		if err := json.Unmarshal(raw.Content, &sub); err != nil {
			return nil, fmt.Errorf("malformed subscription.updated payload: %w", err)
		}
		event.SubscriptionUpdated = &billing.BillingWebhookSubscriptionUpdated{Subscription: sub}

	case "subscription.deleted":
		var content struct {
			SubscriptionID string `json:"subscription_id"`
		}
		if err := json.Unmarshal(raw.Content, &content); err != nil {
			return nil, fmt.Errorf("malformed subscription.deleted payload: %w", err)
		}
		event.SubscriptionDeleted = &billing.BillingWebhookSubscriptionDeleted{SubscriptionID: content.SubscriptionID}

	case "invoice.paid":
		var content billing.BillingWebhookInvoicePaid
		if err := json.Unmarshal(raw.Content, &content); err != nil {
			return nil, fmt.Errorf("malformed invoice.paid payload: %w", err)
		}
		event.InvoicePaid = &content

	case "payment.failed":
		var content billing.BillingWebhookPaymentFailed
		if err := json.Unmarshal(raw.Content, &content); err != nil {
			return nil, fmt.Errorf("malformed payment.failed payload: %w", err)
		}
		event.PaymentFailed = &content
	}

	return event, nil
}

// ─── HTTP Helpers ────────────────────────────────────────────────────────────

func (a *HyperswitchSubscriptionAdapter) doRequest(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, a.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", a.apiKey)

	return a.client.Do(req)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Noop Adapter (dev/test — no Hyperswitch configured) [10-billing §7]
// ═══════════════════════════════════════════════════════════════════════════════

type noopSubscriptionAdapter struct{}

func (n *noopSubscriptionAdapter) CreateCustomer(_ context.Context, _ string, _ string, _ map[string]string) (string, error) {
	slog.Warn("billing adapter not configured — CreateCustomer is a no-op")
	return "", billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) UpdateCustomer(_ context.Context, _ string, _ string, _ string) error {
	return billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) CreateSubscription(_ context.Context, _ string, _ string, _ string, _ map[string]string) (*billing.HyperswitchSubscription, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) UpdateSubscription(_ context.Context, _ string, _ string) (*billing.HyperswitchSubscription, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) CancelSubscription(_ context.Context, _ string) (*billing.HyperswitchSubscription, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) PauseSubscription(_ context.Context, _ string) (*billing.HyperswitchSubscription, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) ResumeSubscription(_ context.Context, _ string) (*billing.HyperswitchSubscription, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) ReactivateSubscription(_ context.Context, _ string) (*billing.HyperswitchSubscription, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) EstimateSubscription(_ context.Context, _ string, _ string, _ *string) (*billing.HyperswitchEstimate, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) CreateSetupIntent(_ context.Context, _ string) (*billing.SetupIntentResponse, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) ListPaymentMethods(_ context.Context, _ string) ([]billing.HyperswitchPaymentMethod, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) DetachPaymentMethod(_ context.Context, _ string) error {
	return billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) ProcessMicroCharge(_ context.Context, _ string, _ string, _ int64, _ string, _ map[string]string) (string, string, error) {
	return "", "", billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) ListInvoices(_ context.Context, _ string, _ uint32) ([]billing.HyperswitchInvoice, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) CreatePayout(_ context.Context, _ string, _ int64, _ string, _ map[string]string) (*billing.HyperswitchPayout, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) VerifyWebhook(_ context.Context, _ []byte, _ string) (bool, error) {
	return false, billing.ErrPaymentAdapterUnavailable
}

func (n *noopSubscriptionAdapter) ParseWebhookEvent(_ context.Context, _ []byte) (*billing.BillingWebhookEvent, error) {
	return nil, billing.ErrPaymentAdapterUnavailable
}
