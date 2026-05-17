//go:build integration

package adapters_test

// Adapter-level integration tests against a real Hyperswitch sandbox.
//
// Run with:
//
//	HYPERSWITCH_BASE_URL=https://sandbox.hyperswitch.io \
//	HYPERSWITCH_API_KEY=snd_... \
//	HYPERSWITCH_BILLING_PROFILE_ID=pro_... \
//	HYPERSWITCH_MONTHLY_PRICE_ID=price_... \
//	HYPERSWITCH_ANNUAL_PRICE_ID=price_... \
//	go test -tags=integration ./internal/billing/adapters/...
//
// All tests are skipped automatically when the required env vars are absent,
// so the suite stays green in local dev without Hyperswitch configured.
//
// These tests use Hyperswitch test-mode credentials and do not charge real money.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/billing"
	"github.com/homegrown-academy/homegrown-academy/internal/billing/adapters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Test Setup ───────────────────────────────────────────────────────────────

type sandboxCreds struct {
	baseURL    string
	apiKey     string
	profileID  string
	webhookKey string
	monthlyID  string
	annualID   string
}

func loadCreds(t *testing.T) sandboxCreds {
	t.Helper()
	baseURL := os.Getenv("HYPERSWITCH_BASE_URL")
	apiKey := os.Getenv("HYPERSWITCH_API_KEY")
	if baseURL == "" || apiKey == "" {
		t.Skip("HYPERSWITCH_BASE_URL and HYPERSWITCH_API_KEY not set — skipping Hyperswitch sandbox tests")
	}
	return sandboxCreds{
		baseURL:    baseURL,
		apiKey:     apiKey,
		profileID:  os.Getenv("HYPERSWITCH_BILLING_PROFILE_ID"),
		webhookKey: os.Getenv("HYPERSWITCH_WEBHOOK_KEY"),
		monthlyID:  os.Getenv("HYPERSWITCH_MONTHLY_PRICE_ID"),
		annualID:   os.Getenv("HYPERSWITCH_ANNUAL_PRICE_ID"),
	}
}

func newSandboxAdapter(creds sandboxCreds) billing.SubscriptionPaymentAdapter {
	return adapters.NewHyperswitchSubscriptionAdapter(
		creds.baseURL, creds.apiKey, creds.profileID, creds.webhookKey,
	)
}

// ─── Customer ─────────────────────────────────────────────────────────────────

// TestHyperswitchAdapter_CreateCustomer verifies the adapter can create a
// Hyperswitch customer and returns a non-empty customer ID.
func TestHyperswitchAdapter_CreateCustomer(t *testing.T) {
	creds := loadCreds(t)
	ctx := context.Background()
	adapter := newSandboxAdapter(creds)

	customerID, err := adapter.CreateCustomer(ctx, "billing-test@homegrown.example", "Billing Test Family", map[string]string{
		"family_id": "test-family-integration",
		"source":    "integration_test",
	})
	require.NoError(t, err, "CreateCustomer should succeed against sandbox")
	assert.NotEmpty(t, customerID, "returned customer ID must be non-empty")
	t.Logf("created customer: %s", customerID)
}

// TestHyperswitchAdapter_UpdateCustomer verifies the adapter can update a
// customer's email after creation.
func TestHyperswitchAdapter_UpdateCustomer(t *testing.T) {
	creds := loadCreds(t)
	ctx := context.Background()
	adapter := newSandboxAdapter(creds)

	customerID, err := adapter.CreateCustomer(ctx, "update-test@homegrown.example", "Update Test", nil)
	require.NoError(t, err)

	err = adapter.UpdateCustomer(ctx, customerID, "updated@homegrown.example", "Updated Name")
	assert.NoError(t, err, "UpdateCustomer should succeed")
}

// ─── Subscription State Machine ───────────────────────────────────────────────

// TestHyperswitchAdapter_SubscriptionLifecycle_CreateAndCancel exercises the
// create → cancel (end-of-period) lifecycle against the real Hyperswitch sandbox.
// Requires HYPERSWITCH_MONTHLY_PRICE_ID to be set with a valid test-mode price.
func TestHyperswitchAdapter_SubscriptionLifecycle_CreateAndCancel(t *testing.T) {
	creds := loadCreds(t)
	if creds.monthlyID == "" {
		t.Skip("HYPERSWITCH_MONTHLY_PRICE_ID not set — skipping subscription lifecycle test")
	}
	ctx := context.Background()
	adapter := newSandboxAdapter(creds)

	// Step 1: Create a test customer.
	customerID, err := adapter.CreateCustomer(ctx, "lifecycle-test@homegrown.example", "Lifecycle Test", nil)
	require.NoError(t, err)
	t.Logf("customer: %s", customerID)

	// Step 2: Create a setup intent to obtain a test payment method.
	// In test mode, Hyperswitch returns a client_secret that can be used with
	// test card numbers to produce a payment_method_id.
	intent, err := adapter.CreateSetupIntent(ctx, customerID)
	require.NoError(t, err, "CreateSetupIntent should succeed")
	require.NotEmpty(t, intent.ClientSecret, "client_secret must be non-empty")
	t.Logf("setup intent client_secret: %s", intent.ClientSecret[:min(20, len(intent.ClientSecret))]+"...")

	// Step 3: Create subscription.
	// Note: In a real test flow, payment_method_id would come from completing
	// the setup intent with a test card. Here we use a Hyperswitch test token
	// if available via HYPERSWITCH_TEST_PAYMENT_METHOD_ID, otherwise skip.
	testPMID := os.Getenv("HYPERSWITCH_TEST_PAYMENT_METHOD_ID")
	if testPMID == "" {
		t.Skip("HYPERSWITCH_TEST_PAYMENT_METHOD_ID not set — cannot create subscription without a payment method")
	}

	sub, err := adapter.CreateSubscription(ctx, customerID, creds.monthlyID, testPMID, map[string]string{
		"source": "integration_test",
	})
	require.NoError(t, err, "CreateSubscription should succeed")
	require.NotEmpty(t, sub.ID, "subscription ID must be non-empty")
	assert.Equal(t, customerID, sub.CustomerID)
	t.Logf("subscription: %s status=%s", sub.ID, sub.Status)

	// Step 4: Cancel at end of period.
	canceled, err := adapter.CancelSubscription(ctx, sub.ID)
	require.NoError(t, err, "CancelSubscription should succeed")
	assert.True(t, canceled.CancelAtPeriodEnd, "CancelAtPeriodEnd must be true after end-of-period cancel")
	t.Logf("canceled subscription: %s cancel_at_period_end=%v", canceled.ID, canceled.CancelAtPeriodEnd)
}

// TestHyperswitchAdapter_SubscriptionLifecycle_CreateUpgradeReactivate exercises
// create → upgrade (proration: monthly→annual) → reactivate after cancel.
func TestHyperswitchAdapter_SubscriptionLifecycle_CreateUpgradeReactivate(t *testing.T) {
	creds := loadCreds(t)
	if creds.monthlyID == "" || creds.annualID == "" {
		t.Skip("HYPERSWITCH_MONTHLY_PRICE_ID or HYPERSWITCH_ANNUAL_PRICE_ID not set")
	}
	testPMID := os.Getenv("HYPERSWITCH_TEST_PAYMENT_METHOD_ID")
	if testPMID == "" {
		t.Skip("HYPERSWITCH_TEST_PAYMENT_METHOD_ID not set")
	}

	ctx := context.Background()
	adapter := newSandboxAdapter(creds)

	customerID, err := adapter.CreateCustomer(ctx, "upgrade-test@homegrown.example", "Upgrade Test", nil)
	require.NoError(t, err)

	// Create with monthly plan.
	sub, err := adapter.CreateSubscription(ctx, customerID, creds.monthlyID, testPMID, nil)
	require.NoError(t, err)
	t.Logf("created monthly subscription: %s", sub.ID)

	// Upgrade to annual (proration applies in test mode).
	upgraded, err := adapter.UpdateSubscription(ctx, sub.ID, creds.annualID)
	require.NoError(t, err, "UpdateSubscription (upgrade) should succeed")
	assert.Equal(t, creds.annualID, upgraded.PriceID, "PriceID should reflect annual plan after upgrade")
	t.Logf("upgraded to annual: price_id=%s amount=%d", upgraded.PriceID, upgraded.AmountCents)

	// Cancel so we can test reactivation.
	_, err = adapter.CancelSubscription(ctx, sub.ID)
	require.NoError(t, err)

	// Reactivate.
	reactivated, err := adapter.ReactivateSubscription(ctx, sub.ID)
	require.NoError(t, err, "ReactivateSubscription should succeed")
	assert.False(t, reactivated.CancelAtPeriodEnd, "CancelAtPeriodEnd must be false after reactivation")
	t.Logf("reactivated: cancel_at_period_end=%v", reactivated.CancelAtPeriodEnd)
}

// ─── COPPA Micro-Charge ───────────────────────────────────────────────────────

// TestHyperswitchAdapter_ProcessMicroCharge_COPPA verifies the COPPA flow:
// a $0.50 charge followed by an immediate refund. Both IDs must be returned.
func TestHyperswitchAdapter_ProcessMicroCharge_COPPA(t *testing.T) {
	creds := loadCreds(t)
	testPMID := os.Getenv("HYPERSWITCH_TEST_PAYMENT_METHOD_ID")
	if testPMID == "" {
		t.Skip("HYPERSWITCH_TEST_PAYMENT_METHOD_ID not set — cannot test COPPA micro-charge")
	}

	ctx := context.Background()
	adapter := newSandboxAdapter(creds)

	customerID, err := adapter.CreateCustomer(ctx, "coppa-test@homegrown.example", "COPPA Test", nil)
	require.NoError(t, err)

	chargeID, refundID, err := adapter.ProcessMicroCharge(
		ctx, customerID, testPMID, 50, // $0.50
		"COPPA parental consent verification",
		map[string]string{"source": "integration_test", "purpose": "coppa_verification"},
	)
	require.NoError(t, err, "ProcessMicroCharge should succeed")
	assert.NotEmpty(t, chargeID, "charge ID must be returned")
	assert.NotEmpty(t, refundID, "refund ID must be returned (immediate refund)")
	t.Logf("COPPA charge: %s refund: %s", chargeID, refundID)
}

// ─── Webhook Verification ─────────────────────────────────────────────────────

// TestHyperswitchAdapter_VerifyWebhook_ValidSignature verifies that a payload
// signed with the configured webhook key passes verification. This is a pure
// crypto test — no HTTP call is made.
func TestHyperswitchAdapter_VerifyWebhook_ValidSignature(t *testing.T) {
	creds := loadCreds(t)
	if creds.webhookKey == "" {
		t.Skip("HYPERSWITCH_WEBHOOK_KEY not set")
	}

	ctx := context.Background()
	adapter := newSandboxAdapter(creds)

	payload := []byte(`{"type":"invoice.paid","content":{"invoice_id":"inv_test","amount":1499}}`)
	mac := hmac.New(sha256.New, []byte(creds.webhookKey))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	valid, err := adapter.VerifyWebhook(ctx, payload, sig)
	require.NoError(t, err)
	assert.True(t, valid, "valid HMAC signature must be accepted")
}

// TestHyperswitchAdapter_VerifyWebhook_InvalidSignature verifies that a tampered
// payload fails verification with ErrInvalidWebhookSignature.
func TestHyperswitchAdapter_VerifyWebhook_InvalidSignature(t *testing.T) {
	creds := loadCreds(t)
	if creds.webhookKey == "" {
		t.Skip("HYPERSWITCH_WEBHOOK_KEY not set")
	}

	ctx := context.Background()
	adapter := newSandboxAdapter(creds)

	payload := []byte(`{"type":"invoice.paid","content":{}}`)
	_, err := adapter.VerifyWebhook(ctx, payload, "tampered-signature")
	assert.ErrorIs(t, err, billing.ErrInvalidWebhookSignature)
}

// TestHyperswitchAdapter_ParseWebhookEvent_SubscriptionUpdated verifies that
// a subscription.updated payload is parsed into the correct event type.
// This is a pure JSON test — no HTTP call is made.
func TestHyperswitchAdapter_ParseWebhookEvent_AllEventTypes(t *testing.T) {
	creds := loadCreds(t)
	ctx := context.Background()
	adapter := newSandboxAdapter(creds)

	now := time.Now()

	tests := []struct {
		eventType   string
		content     any
		assertField func(t *testing.T, event *billing.BillingWebhookEvent)
	}{
		{
			eventType: "subscription.created",
			content: billing.HyperswitchSubscription{
				ID: "sub_test", CustomerID: "cus_test",
				Status:             billing.SubscriptionStatusIncomplete,
				CurrentPeriodStart: now, CurrentPeriodEnd: now.Add(30 * 24 * time.Hour),
				AmountCents: 1499, Currency: "usd",
			},
			assertField: func(t *testing.T, e *billing.BillingWebhookEvent) {
				require.NotNil(t, e.SubscriptionCreated)
				assert.Equal(t, "sub_test", e.SubscriptionCreated.Subscription.ID)
			},
		},
		{
			eventType: "subscription.updated",
			content: billing.HyperswitchSubscription{
				ID: "sub_upd", Status: billing.SubscriptionStatusActive,
				CurrentPeriodStart: now, CurrentPeriodEnd: now.Add(30 * 24 * time.Hour),
			},
			assertField: func(t *testing.T, e *billing.BillingWebhookEvent) {
				require.NotNil(t, e.SubscriptionUpdated)
				assert.Equal(t, billing.SubscriptionStatusActive, e.SubscriptionUpdated.Subscription.Status)
			},
		},
		{
			eventType: "subscription.deleted",
			content:   map[string]string{"subscription_id": "sub_del"},
			assertField: func(t *testing.T, e *billing.BillingWebhookEvent) {
				require.NotNil(t, e.SubscriptionDeleted)
				assert.Equal(t, "sub_del", e.SubscriptionDeleted.SubscriptionID)
			},
		},
		{
			eventType: "invoice.paid",
			content: billing.BillingWebhookInvoicePaid{
				InvoiceID: "inv_1", SubscriptionID: "sub_1", AmountCents: 1499, PaymentID: "pay_1",
			},
			assertField: func(t *testing.T, e *billing.BillingWebhookEvent) {
				require.NotNil(t, e.InvoicePaid)
				assert.Equal(t, "pay_1", e.InvoicePaid.PaymentID)
				assert.Equal(t, int64(1499), e.InvoicePaid.AmountCents)
			},
		},
		{
			eventType: "payment.failed",
			content: billing.BillingWebhookPaymentFailed{
				PaymentID: "pay_fail", Reason: "insufficient_funds",
			},
			assertField: func(t *testing.T, e *billing.BillingWebhookEvent) {
				require.NotNil(t, e.PaymentFailed)
				assert.Equal(t, "insufficient_funds", e.PaymentFailed.Reason)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.eventType, func(t *testing.T) {
			raw, err := json.Marshal(tc.content)
			require.NoError(t, err)
			payload, err := json.Marshal(map[string]any{
				"type":    tc.eventType,
				"content": json.RawMessage(raw),
			})
			require.NoError(t, err)

			event, err := adapter.ParseWebhookEvent(ctx, payload)
			require.NoError(t, err)
			assert.Equal(t, tc.eventType, event.Type)
			tc.assertField(t, event)
		})
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
