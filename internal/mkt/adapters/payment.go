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

	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
)

// HyperswitchPaymentAdapter wraps the Hyperswitch REST API for marketplace payments.
// Processor-agnostic: Stripe is configured in Hyperswitch as the initial connector,
// swappable without code changes. [07-mkt §7, supersedes ADR-007]
type HyperswitchPaymentAdapter struct {
	baseURL    string
	apiKey     string
	webhookKey string
	client     *http.Client
}

// NewHyperswitchPaymentAdapter creates a new Hyperswitch payment adapter.
// Returns nil if baseURL or apiKey are empty (Hyperswitch not configured).
func NewHyperswitchPaymentAdapter(baseURL, apiKey, webhookKey string) mkt.PaymentAdapter {
	if baseURL == "" || apiKey == "" {
		return &noopPaymentAdapter{}
	}
	return &HyperswitchPaymentAdapter{
		baseURL:    baseURL,
		apiKey:     apiKey,
		webhookKey: webhookKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ─── Account Management ─────────────────────────────────────────────────────

func (a *HyperswitchPaymentAdapter) CreateSubMerchant(_ context.Context, config mkt.SubMerchantConfig) (string, error) {
	body := map[string]any{
		"merchant_name": config.StoreName,
		"metadata": map[string]string{
			"creator_id": config.CreatorID.String(),
			"email":      config.Email,
			"country":    config.Country,
		},
	}

	resp, err := a.doRequest(http.MethodPost, "/accounts", body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", mkt.ErrPaymentCreationFailed
	}

	var result struct {
		MerchantID string `json:"merchant_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}
	return result.MerchantID, nil
}

func (a *HyperswitchPaymentAdapter) CreateOnboardingLink(_ context.Context, paymentAccountID, returnURL string) (string, error) {
	body := map[string]any{
		"merchant_id": paymentAccountID,
		"return_url":  returnURL,
	}

	resp, err := a.doRequest(http.MethodPost, "/account_link", body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", mkt.ErrPaymentProviderUnavailable
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}
	return result.URL, nil
}

func (a *HyperswitchPaymentAdapter) GetAccountStatus(_ context.Context, paymentAccountID string) (mkt.PaymentAccountStatus, error) {
	resp, err := a.doRequest(http.MethodGet, fmt.Sprintf("/accounts/%s", paymentAccountID), nil)
	if err != nil {
		return mkt.PaymentAccountStatusPending, fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return mkt.PaymentAccountStatusPending, mkt.ErrPaymentProviderUnavailable
	}

	var result struct {
		IsVerified bool   `json:"is_verified"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return mkt.PaymentAccountStatusPending, fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}

	if result.IsVerified {
		return mkt.PaymentAccountStatusActive, nil
	}
	switch result.Status {
	case "active":
		return mkt.PaymentAccountStatusActive, nil
	case "suspended":
		return mkt.PaymentAccountStatusSuspended, nil
	default:
		return mkt.PaymentAccountStatusOnboarding, nil
	}
}

// ─── Payments ────────────────────────────────────────────────────────────────

func (a *HyperswitchPaymentAdapter) CreatePayment(_ context.Context, lineItems []mkt.PaymentLineItem, splitRules []mkt.SplitRule, returnURL string, metadata map[string]string) (*mkt.PaymentSession, error) {
	var totalCents int64
	for _, item := range lineItems {
		totalCents += item.AmountCents
	}

	body := map[string]any{
		"amount":     totalCents,
		"currency":   "USD",
		"return_url": returnURL,
		"metadata":   metadata,
		"splits":     splitRules,
		"order_details": func() []map[string]any {
			details := make([]map[string]any, len(lineItems))
			for i, item := range lineItems {
				details[i] = map[string]any{
					"product_id":   item.ListingID.String(),
					"amount":       item.AmountCents,
					"product_name": item.Description,
				}
			}
			return details
		}(),
	}

	resp, err := a.doRequest(http.MethodPost, "/payments", body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, mkt.ErrPaymentCreationFailed
	}

	var result struct {
		PaymentID string `json:"payment_id"`
		ClientSecret string `json:"client_secret"`
		NextAction struct {
			RedirectToURL string `json:"redirect_to_url"`
		} `json:"next_action"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}

	checkoutURL := result.NextAction.RedirectToURL
	if checkoutURL == "" {
		checkoutURL = fmt.Sprintf("%s/payments/%s", a.baseURL, result.PaymentID)
	}

	return &mkt.PaymentSession{
		CheckoutURL:      checkoutURL,
		PaymentSessionID: result.PaymentID,
	}, nil
}

func (a *HyperswitchPaymentAdapter) GetPaymentStatus(_ context.Context, paymentID string) (mkt.PaymentStatus, error) {
	resp, err := a.doRequest(http.MethodGet, fmt.Sprintf("/payments/%s", paymentID), nil)
	if err != nil {
		return mkt.PaymentStatusProcessing, fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return mkt.PaymentStatusProcessing, mkt.ErrPaymentProviderUnavailable
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return mkt.PaymentStatusProcessing, fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}

	switch result.Status {
	case "succeeded":
		return mkt.PaymentStatusSucceeded, nil
	case "failed":
		return mkt.PaymentStatusFailed, nil
	case "cancelled":
		return mkt.PaymentStatusCancelled, nil
	default:
		return mkt.PaymentStatusProcessing, nil
	}
}

// ─── Payouts ─────────────────────────────────────────────────────────────────

func (a *HyperswitchPaymentAdapter) CreatePayout(_ context.Context, paymentAccountID string, amountCents int64, currency string) (*mkt.PayoutResult, error) {
	body := map[string]any{
		"merchant_id": paymentAccountID,
		"amount":      amountCents,
		"currency":    currency,
		"payout_type": "bank",
	}

	resp, err := a.doRequest(http.MethodPost, "/payouts/create", body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, mkt.ErrPaymentProviderUnavailable
	}

	var result struct {
		PayoutID string `json:"payout_id"`
		Amount   int64  `json:"amount"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}

	return &mkt.PayoutResult{
		PayoutID:    result.PayoutID,
		AmountCents: result.Amount,
		Status:      result.Status,
	}, nil
}

// ─── Refunds ─────────────────────────────────────────────────────────────────

func (a *HyperswitchPaymentAdapter) CreateRefund(_ context.Context, paymentID string, amountCents int64, reason string) (*mkt.RefundResult, error) {
	body := map[string]any{
		"payment_id": paymentID,
		"amount":     amountCents,
		"reason":     reason,
	}

	resp, err := a.doRequest(http.MethodPost, "/refunds", body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, mkt.ErrPaymentProviderUnavailable
	}

	var result struct {
		RefundID string `json:"refund_id"`
		Amount   int64  `json:"amount"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: %v", mkt.ErrPaymentProviderUnavailable, err)
	}

	return &mkt.RefundResult{
		RefundID:    result.RefundID,
		AmountCents: result.Amount,
		Status:      result.Status,
	}, nil
}

// ─── Webhooks ────────────────────────────────────────────────────────────────

func (a *HyperswitchPaymentAdapter) VerifyWebhook(_ context.Context, payload []byte, signature string) (bool, error) {
	if a.webhookKey == "" {
		return false, mkt.ErrInvalidWebhookSignature
	}

	mac := hmac.New(sha256.New, []byte(a.webhookKey))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return false, mkt.ErrInvalidWebhookSignature
	}
	return true, nil
}

func (a *HyperswitchPaymentAdapter) ParseEvent(_ context.Context, payload []byte) (*mkt.PaymentEvent, error) {
	var raw struct {
		Type    string `json:"type"`
		Content struct {
			PaymentID   string            `json:"payment_id"`
			RefundID    string            `json:"refund_id"`
			PayoutID    string            `json:"payout_id"`
			MerchantID  string            `json:"merchant_id"`
			Amount      int64             `json:"amount"`
			Reason      string            `json:"reason"`
			Metadata    map[string]string `json:"metadata"`
		} `json:"content"`
	}

	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", mkt.ErrMalformedWebhookPayload, err)
	}

	return &mkt.PaymentEvent{
		Type:        raw.Type,
		PaymentID:   raw.Content.PaymentID,
		Metadata:    raw.Content.Metadata,
		Reason:      raw.Content.Reason,
		RefundID:    raw.Content.RefundID,
		AmountCents: raw.Content.Amount,
		MerchantID:  raw.Content.MerchantID,
		PayoutID:    raw.Content.PayoutID,
	}, nil
}

// ─── HTTP Helpers ────────────────────────────────────────────────────────────

func (a *HyperswitchPaymentAdapter) doRequest(method, path string, body any) (*http.Response, error) {
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

// ─── Noop Adapter (Hyperswitch not configured) ──────────────────────────────

// noopPaymentAdapter returns ErrPaymentProviderUnavailable for all operations.
// Used when Hyperswitch is not configured (development, testing). [07-mkt §7]
type noopPaymentAdapter struct{}

func (n *noopPaymentAdapter) CreateSubMerchant(_ context.Context, _ mkt.SubMerchantConfig) (string, error) {
	slog.Warn("payment adapter not configured — CreateSubMerchant is a no-op")
	return "", mkt.ErrPaymentProviderUnavailable
}

func (n *noopPaymentAdapter) CreateOnboardingLink(_ context.Context, _, _ string) (string, error) {
	slog.Warn("payment adapter not configured — CreateOnboardingLink is a no-op")
	return "", mkt.ErrPaymentProviderUnavailable
}

func (n *noopPaymentAdapter) GetAccountStatus(_ context.Context, _ string) (mkt.PaymentAccountStatus, error) {
	return mkt.PaymentAccountStatusPending, mkt.ErrPaymentProviderUnavailable
}

func (n *noopPaymentAdapter) CreatePayment(_ context.Context, _ []mkt.PaymentLineItem, _ []mkt.SplitRule, _ string, _ map[string]string) (*mkt.PaymentSession, error) {
	return nil, mkt.ErrPaymentProviderUnavailable
}

func (n *noopPaymentAdapter) GetPaymentStatus(_ context.Context, _ string) (mkt.PaymentStatus, error) {
	return mkt.PaymentStatusProcessing, mkt.ErrPaymentProviderUnavailable
}

func (n *noopPaymentAdapter) CreatePayout(_ context.Context, _ string, _ int64, _ string) (*mkt.PayoutResult, error) {
	return nil, mkt.ErrPaymentProviderUnavailable
}

func (n *noopPaymentAdapter) CreateRefund(_ context.Context, _ string, _ int64, _ string) (*mkt.RefundResult, error) {
	return nil, mkt.ErrPaymentProviderUnavailable
}

func (n *noopPaymentAdapter) VerifyWebhook(_ context.Context, _ []byte, _ string) (bool, error) {
	return false, mkt.ErrPaymentProviderUnavailable
}

func (n *noopPaymentAdapter) ParseEvent(_ context.Context, _ []byte) (*mkt.PaymentEvent, error) {
	return nil, mkt.ErrPaymentProviderUnavailable
}
