//go:build integration

package billing

// Integration tests that require a real PostgreSQL + PostGIS database.
//
// Run with:
//
//	go test -tags=integration ./internal/billing/...
//
// Tests spin up a postgis/postgis Docker container via testcontainers-go,
// run all goose migrations, and verify the subscription state machine,
// COPPA micro-charge flow, and webhook processing pipeline against real DB.
//
// Skipped automatically if Docker is unavailable.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	ctx := context.Background()
	db, teardown, err := startTestDB(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: skipping db setup: %v\n", err)
		os.Exit(m.Run())
	}
	testDB = db
	code := m.Run()
	teardown()
	os.Exit(code)
}

func startTestDB(ctx context.Context) (*gorm.DB, func(), error) {
	req := testcontainers.ContainerRequest{
		Image: "postgis/postgis:18-3.6",
		Env: map[string]string{
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
			wait.ForListeningPort("5432/tcp"),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("testcontainers: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("get host: %w", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("get port: %w", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=postgres password=testpass dbname=testdb sslmode=disable",
		host, port.Port(),
	)

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("gorm open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("get sql.DB: %w", err)
	}

	wd, _ := os.Getwd()
	migrationsDir := filepath.Join(wd, "..", "..", "migrations")

	goose.SetDialect("postgres") //nolint:errcheck
	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		_ = sqlDB.Close()
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("goose up: %w", err)
	}

	teardown := func() {
		_ = sqlDB.Close()
		_ = container.Terminate(ctx)
	}
	return db, teardown, nil
}

func skipIfNoTestDB(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("testDB not available (Docker required for integration tests)")
	}
}

// seedIntTestFamily inserts a minimal iam_families row (bypassing RLS) and returns its ID.
// Billing tables have a FK to iam_families, so a family must exist before any billing data.
func seedIntTestFamily(t *testing.T) uuid.UUID {
	t.Helper()
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := uuid.Must(uuid.NewV7())
	err := shared.BypassRLSTransaction(ctx, testDB, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO iam_families (id, display_name, primary_methodology_slug)
			 VALUES (?, 'Billing Test Family', 'charlotte-mason')`,
			familyID,
		).Error
	})
	require.NoError(t, err, "seedIntTestFamily")
	t.Cleanup(func() {
		testDB.Exec(`DELETE FROM bill_subscriptions WHERE family_id = ?`, familyID)
		testDB.Exec(`DELETE FROM bill_transactions WHERE family_id = ?`, familyID)
		testDB.Exec(`DELETE FROM bill_hyperswitch_customers WHERE family_id = ?`, familyID)
		testDB.Exec(`DELETE FROM iam_families WHERE id = ?`, familyID)
	})
	return familyID
}

// TestBillingIntegration_CustomerUpsertAndFind verifies that a Hyperswitch customer
// can be upserted and retrieved by family_id. [10-billing §3.2]
func TestBillingIntegration_CustomerUpsertAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgCustomerRepository(testDB)
	familyID := seedIntTestFamily(t)

	customer, err := repo.Upsert(ctx, familyID, UpsertCustomerRow{
		HyperswitchCustomerID: fmt.Sprintf("hs_cust_%s", familyID),
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if customer.FamilyID != familyID {
		t.Errorf("want FamilyID=%v, got %v", familyID, customer.FamilyID)
	}

	found, err := repo.FindByFamily(ctx, familyID)
	if err != nil {
		t.Fatalf("FindByFamily: %v", err)
	}
	if found.HyperswitchCustomerID != customer.HyperswitchCustomerID {
		t.Errorf("want customer_id=%q, got %q", customer.HyperswitchCustomerID, found.HyperswitchCustomerID)
	}
}

// TestBillingIntegration_SubscriptionCreateAndFind verifies subscription CRUD
// and that FindByFamily returns the correct record. [10-billing §3.2]
func TestBillingIntegration_SubscriptionCreateAndFind(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	// Billing subscription FKs to iam_families — seed one first.
	familyID := seedIntTestFamily(t)

	now := time.Now().UTC().Truncate(time.Second)
	repo := NewPgSubscriptionRepository(testDB)

	sub, err := repo.Create(ctx, CreateSubscriptionRow{
		FamilyID:                  familyID,
		HyperswitchSubscriptionID: fmt.Sprintf("hs_sub_%s", familyID),
		HyperswitchCustomerID:     fmt.Sprintf("hs_cust_%s", familyID),
		Tier:                      "premium",
		Status:                    "active",
		BillingInterval:           "monthly",
		CurrentPeriodStart:        now,
		CurrentPeriodEnd:          now.AddDate(0, 1, 0),
		AmountCents:               999,
		Currency:                  "usd",
		HyperswitchPriceID:        "price_monthly_test",
	})
	if err != nil {
		t.Fatalf("Create subscription: %v", err)
	}

	scope := shared.NewFamilyScopeFromID(familyID)
	found, err := repo.FindByFamily(ctx, scope)
	if err != nil {
		t.Fatalf("FindByFamily: %v", err)
	}
	if found.ID != sub.ID {
		t.Errorf("want subscription ID=%v, got %v", sub.ID, found.ID)
	}
	if found.Status != "active" {
		t.Errorf("want status=active, got %q", found.Status)
	}
}

// TestBillingIntegration_TransactionListEmpty verifies that ListByFamily returns an
// empty slice (not an error) when no transactions exist for a family. [10-billing §3.2]
func TestBillingIntegration_TransactionListEmpty(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()

	repo := NewPgTransactionRepository(testDB)
	familyID := uuid.Must(uuid.NewV7())
	scope := shared.NewFamilyScopeFromID(familyID)

	txns, err := repo.ListByFamily(ctx, scope, &TransactionListParams{})
	if err != nil {
		t.Fatalf("ListByFamily: %v", err)
	}
	if len(txns) != 0 {
		t.Errorf("expected 0 transactions for new family, got %d", len(txns))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers for state-machine and webhook tests
// ═══════════════════════════════════════════════════════════════════════════════

const intTestWebhookKey = "integration-test-webhook-key-32ch"

// intTestConfig returns billing config for integration tests.
func intTestConfig() BillingConfig {
	return BillingConfig{
		MonthlyPriceID:       "price_monthly_test",
		AnnualPriceID:        "price_annual_test",
		CoppaChargeCents:     50,
		WebhookSigningSecret: intTestWebhookKey,
	}
}

// seedActiveSub inserts an active subscription directly into the DB and returns it.
func seedActiveSub(t *testing.T, familyID uuid.UUID, hsSubID, hsCustID string) *BillSubscription {
	t.Helper()
	now := time.Now()
	sub, err := NewPgSubscriptionRepository(testDB).Create(context.Background(), CreateSubscriptionRow{
		FamilyID:                  familyID,
		HyperswitchSubscriptionID: hsSubID,
		HyperswitchCustomerID:     hsCustID,
		Tier:                      TierPremium,
		Status:                    SubscriptionStatusActive,
		BillingInterval:           IntervalMonthly,
		CurrentPeriodStart:        now,
		CurrentPeriodEnd:          now.Add(30 * 24 * time.Hour),
		AmountCents:               1499,
		Currency:                  "usd",
		HyperswitchPriceID:        "price_monthly_test",
	})
	require.NoError(t, err, "seedActiveSub")
	return sub
}

// signIntWebhook computes HMAC-SHA256 matching the adapter's VerifyWebhook logic.
func signIntWebhook(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(intTestWebhookKey))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// makeIntWebhookPayload serialises a typed event into the format ParseWebhookEvent expects.
func makeIntWebhookPayload(t *testing.T, eventType string, content any) []byte {
	t.Helper()
	raw, err := json.Marshal(content)
	require.NoError(t, err)
	body, err := json.Marshal(map[string]any{
		"type":    eventType,
		"content": json.RawMessage(raw),
	})
	require.NoError(t, err)
	return body
}

// webhookTestAdapter implements SubscriptionPaymentAdapter with real HMAC verification
// and real JSON parsing for webhook methods. All other methods return ErrPaymentAdapterUnavailable
// (they are not called by ProcessHyperswitchWebhook). This avoids importing
// billing/adapters (which would create an import cycle from package billing).
type webhookTestAdapter struct{}

func (a *webhookTestAdapter) VerifyWebhook(_ context.Context, payload []byte, sig string) (bool, error) {
	mac := hmac.New(sha256.New, []byte(intTestWebhookKey))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return false, ErrInvalidWebhookSignature
	}
	return true, nil
}

func (a *webhookTestAdapter) ParseWebhookEvent(_ context.Context, payload []byte) (*BillingWebhookEvent, error) {
	var raw struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	event := &BillingWebhookEvent{Type: raw.Type}
	switch raw.Type {
	case "subscription.created":
		var sub HyperswitchSubscription
		if err := json.Unmarshal(raw.Content, &sub); err != nil {
			return nil, err
		}
		event.SubscriptionCreated = &BillingWebhookSubscriptionCreated{Subscription: sub}
	case "subscription.updated":
		var sub HyperswitchSubscription
		if err := json.Unmarshal(raw.Content, &sub); err != nil {
			return nil, err
		}
		event.SubscriptionUpdated = &BillingWebhookSubscriptionUpdated{Subscription: sub}
	case "subscription.deleted":
		var c struct {
			SubscriptionID string `json:"subscription_id"`
		}
		if err := json.Unmarshal(raw.Content, &c); err != nil {
			return nil, err
		}
		event.SubscriptionDeleted = &BillingWebhookSubscriptionDeleted{SubscriptionID: c.SubscriptionID}
	case "invoice.paid":
		var c BillingWebhookInvoicePaid
		if err := json.Unmarshal(raw.Content, &c); err != nil {
			return nil, err
		}
		event.InvoicePaid = &c
	case "payment.failed":
		var c BillingWebhookPaymentFailed
		if err := json.Unmarshal(raw.Content, &c); err != nil {
			return nil, err
		}
		event.PaymentFailed = &c
	}
	return event, nil
}

func (a *webhookTestAdapter) CreateCustomer(_ context.Context, _, _ string, _ map[string]string) (string, error) {
	return "", ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) UpdateCustomer(_ context.Context, _, _, _ string) error {
	return ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) CreateSubscription(_ context.Context, _, _, _ string, _ map[string]string) (*HyperswitchSubscription, error) {
	return nil, ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) UpdateSubscription(_ context.Context, _, _ string) (*HyperswitchSubscription, error) {
	return nil, ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) CancelSubscription(_ context.Context, _ string) (*HyperswitchSubscription, error) {
	return nil, ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) PauseSubscription(_ context.Context, _ string) (*HyperswitchSubscription, error) {
	return nil, ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) ResumeSubscription(_ context.Context, _ string) (*HyperswitchSubscription, error) {
	return nil, ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) ReactivateSubscription(_ context.Context, _ string) (*HyperswitchSubscription, error) {
	return nil, ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) EstimateSubscription(_ context.Context, _, _ string, _ *string) (*HyperswitchEstimate, error) {
	return nil, ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) CreateSetupIntent(_ context.Context, _ string) (*SetupIntentResponse, error) {
	return nil, ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) ListPaymentMethods(_ context.Context, _ string) ([]HyperswitchPaymentMethod, error) {
	return nil, ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) DetachPaymentMethod(_ context.Context, _ string) error {
	return ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) ProcessMicroCharge(_ context.Context, _, _ string, _ int64, _ string, _ map[string]string) (string, string, error) {
	return "", "", ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) ListInvoices(_ context.Context, _ string, _ uint32) ([]HyperswitchInvoice, error) {
	return nil, ErrPaymentAdapterUnavailable
}
func (a *webhookTestAdapter) CreatePayout(_ context.Context, _ string, _ int64, _ string, _ map[string]string) (*HyperswitchPayout, error) {
	return nil, ErrPaymentAdapterUnavailable
}

// buildWebhookService returns a BillingServiceImpl wired to the real DB repos and webhookTestAdapter.
func buildWebhookService(t *testing.T, bus *shared.EventBus) *BillingServiceImpl {
	t.Helper()
	skipIfNoTestDB(t)
	return NewBillingService(
		NewPgSubscriptionRepository(testDB),
		NewPgTransactionRepository(testDB),
		NewPgCustomerRepository(testDB),
		NewPgPayoutRepository(testDB),
		NewPgCreatorTaxSummaryRepository(testDB),
		&webhookTestAdapter{},
		nil,
		bus,
		intTestConfig(),
	)
}

// intIAMStub satisfies IamServiceForBilling for state-machine tests.
type intIAMStub struct{ email, name string }

func (s *intIAMStub) GetFamilyPrimaryEmail(_ context.Context, _ uuid.UUID) (string, string, error) {
	return s.email, s.name, nil
}

// intCaptureHandler records the last published domain event.
type intCaptureHandler struct{ captured *shared.DomainEvent }

func (h *intCaptureHandler) Handle(_ context.Context, e shared.DomainEvent) error {
	*h.captured = e
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// §AC-1: Subscription State Machine (real DB persistence)
// ═══════════════════════════════════════════════════════════════════════════════

// TestBillingIntegration_Create_PersistsIncompleteRow verifies CreateSubscription
// writes status=incomplete to the DB (Hyperswitch confirms via webhook later).
func TestBillingIntegration_Create_PersistsIncompleteRow(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)

	adapter := new(mockAdapter)
	// FindByFamily/Upsert are repo methods handled by real DB; adapter only needs CreateCustomer + CreateSubscription.
	adapter.On("CreateCustomer", mock.Anything, "p@test.com", "Test", mock.Anything).Return("cus_int_create", nil)
	adapter.On("CreateSubscription", mock.Anything, "cus_int_create", "price_monthly_test", "pm_test", mock.Anything).Return(
		&HyperswitchSubscription{
			ID: "sub_int_create", CustomerID: "cus_int_create",
			Status:             SubscriptionStatusIncomplete,
			CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
			AmountCents: 1499, Currency: "usd",
		}, nil)

	svc := NewBillingService(
		NewPgSubscriptionRepository(testDB),
		NewPgTransactionRepository(testDB),
		NewPgCustomerRepository(testDB),
		NewPgPayoutRepository(testDB),
		NewPgCreatorTaxSummaryRepository(testDB),
		adapter,
		&intIAMStub{"p@test.com", "Test"},
		shared.NewEventBus(),
		intTestConfig(),
	)
	scope := shared.NewFamilyScopeFromID(familyID)

	resp, err := svc.CreateSubscription(ctx, CreateSubscriptionCommand{
		BillingInterval: IntervalMonthly, PaymentMethodID: "pm_test",
	}, scope)
	require.NoError(t, err)
	require.NotNil(t, resp.Status)
	assert.Equal(t, SubscriptionStatusIncomplete, *resp.Status)

	// Verify persisted in DB.
	sub, err := NewPgSubscriptionRepository(testDB).FindByFamily(ctx, scope)
	require.NoError(t, err)
	assert.Equal(t, SubscriptionStatusIncomplete, sub.Status)
}

// TestBillingIntegration_Create_RejectsDouble verifies a second CreateSubscription
// returns ErrSubscriptionAlreadyExists without writing a duplicate row.
func TestBillingIntegration_Create_RejectsDouble(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)
	_ = seedActiveSub(t, familyID, "sub_int_double", "cus_int_double")

	svc := NewBillingService(
		NewPgSubscriptionRepository(testDB),
		NewPgTransactionRepository(testDB),
		NewPgCustomerRepository(testDB),
		NewPgPayoutRepository(testDB),
		NewPgCreatorTaxSummaryRepository(testDB),
		new(mockAdapter),
		&intIAMStub{"p@test.com", "Test"},
		shared.NewEventBus(),
		intTestConfig(),
	)

	_, err := svc.CreateSubscription(ctx, CreateSubscriptionCommand{
		BillingInterval: IntervalMonthly, PaymentMethodID: "pm_2",
	}, shared.NewFamilyScopeFromID(familyID))
	require.Error(t, err)
	var be *BillingError
	require.ErrorAs(t, err, &be)
	assert.ErrorIs(t, be.Err, ErrSubscriptionAlreadyExists)
}

// TestBillingIntegration_Upgrade_PersistsNewInterval verifies UpdateSubscription
// (monthly→annual) updates billing_interval and price_id in the DB.
func TestBillingIntegration_Upgrade_PersistsNewInterval(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)
	sub := seedActiveSub(t, familyID, "sub_int_upgrade", "cus_int_upgrade")

	adapter := new(mockAdapter)
	adapter.On("UpdateSubscription", mock.Anything, sub.HyperswitchSubscriptionID, "price_annual_test").Return(
		&HyperswitchSubscription{}, nil)

	svc := NewBillingService(
		NewPgSubscriptionRepository(testDB),
		NewPgTransactionRepository(testDB),
		NewPgCustomerRepository(testDB),
		NewPgPayoutRepository(testDB),
		NewPgCreatorTaxSummaryRepository(testDB),
		adapter,
		nil, shared.NewEventBus(), intTestConfig(),
	)
	scope := shared.NewFamilyScopeFromID(familyID)

	resp, err := svc.UpdateSubscription(ctx, UpdateSubscriptionCommand{BillingInterval: IntervalAnnual}, scope)
	require.NoError(t, err)
	require.NotNil(t, resp.BillingInterval)
	assert.Equal(t, IntervalAnnual, *resp.BillingInterval)

	updated, err := NewPgSubscriptionRepository(testDB).FindByFamily(ctx, scope)
	require.NoError(t, err)
	assert.Equal(t, IntervalAnnual, updated.BillingInterval)
	assert.Equal(t, "price_annual_test", updated.HyperswitchPriceID)
}

// TestBillingIntegration_Cancel_SetsCancelAtPeriodEnd verifies CancelSubscription
// writes cancel_at_period_end=true while keeping status=active (end-of-period cancel).
func TestBillingIntegration_Cancel_SetsCancelAtPeriodEnd(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)
	sub := seedActiveSub(t, familyID, "sub_int_cancel", "cus_int_cancel")

	adapter := new(mockAdapter)
	adapter.On("CancelSubscription", mock.Anything, sub.HyperswitchSubscriptionID).Return(&HyperswitchSubscription{}, nil)

	svc := NewBillingService(
		NewPgSubscriptionRepository(testDB),
		NewPgTransactionRepository(testDB),
		NewPgCustomerRepository(testDB),
		NewPgPayoutRepository(testDB),
		NewPgCreatorTaxSummaryRepository(testDB),
		adapter, nil, shared.NewEventBus(), intTestConfig(),
	)
	scope := shared.NewFamilyScopeFromID(familyID)

	resp, err := svc.CancelSubscription(ctx, scope)
	require.NoError(t, err)
	assert.True(t, resp.CancelAtPeriodEnd)

	dbSub, err := NewPgSubscriptionRepository(testDB).FindByFamily(ctx, scope)
	require.NoError(t, err)
	assert.True(t, dbSub.CancelAtPeriodEnd)
	assert.NotNil(t, dbSub.CanceledAt)
	assert.Equal(t, SubscriptionStatusActive, dbSub.Status) // still active until period ends
}

// TestBillingIntegration_Reactivate_ClearsCancelFlag verifies ReactivateSubscription
// resets cancel_at_period_end=false in the DB.
func TestBillingIntegration_Reactivate_ClearsCancelFlag(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)
	sub := seedActiveSub(t, familyID, "sub_int_react", "cus_int_react")

	// Mark as pending-cancel.
	subRepo := NewPgSubscriptionRepository(testDB)
	trueVal, canceledAt := true, time.Now()
	_, err := subRepo.Update(ctx, sub.ID, SubscriptionUpdate{CancelAtPeriodEnd: &trueVal, CanceledAt: &canceledAt})
	require.NoError(t, err)

	adapter := new(mockAdapter)
	adapter.On("ReactivateSubscription", mock.Anything, sub.HyperswitchSubscriptionID).Return(&HyperswitchSubscription{}, nil)

	svc := NewBillingService(
		subRepo, NewPgTransactionRepository(testDB),
		NewPgCustomerRepository(testDB), NewPgPayoutRepository(testDB),
		NewPgCreatorTaxSummaryRepository(testDB),
		adapter, nil, shared.NewEventBus(), intTestConfig(),
	)
	scope := shared.NewFamilyScopeFromID(familyID)

	resp, err := svc.ReactivateSubscription(ctx, scope)
	require.NoError(t, err)
	assert.False(t, resp.CancelAtPeriodEnd)

	dbSub, err := subRepo.FindByFamily(ctx, scope)
	require.NoError(t, err)
	assert.False(t, dbSub.CancelAtPeriodEnd)
}

// ═══════════════════════════════════════════════════════════════════════════════
// §AC-2: COPPA Micro-Charge Flow (real DB persistence)
// ═══════════════════════════════════════════════════════════════════════════════

// TestBillingIntegration_Coppa_PersistsBothTransactions verifies ProcessCoppaVerification
// creates a coppa_charge + coppa_refund row in bill_transactions.
func TestBillingIntegration_Coppa_PersistsBothTransactions(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)

	// Real CustomerRepository hits the DB; adapter only needs CreateCustomer + ProcessMicroCharge.
	adapter := new(mockAdapter)
	adapter.On("CreateCustomer", mock.Anything, "coppa@test.com", "COPPA Fam", mock.Anything).Return("cus_coppa_int", nil)
	adapter.On("ProcessMicroCharge", mock.Anything, "cus_coppa_int", "pm_coppa_int", int64(50), mock.Anything, mock.Anything).
		Return("pay_coppa_int", "ref_coppa_int", nil)

	txnRepo := NewPgTransactionRepository(testDB)
	svc := NewBillingService(
		NewPgSubscriptionRepository(testDB), txnRepo,
		NewPgCustomerRepository(testDB), NewPgPayoutRepository(testDB),
		NewPgCreatorTaxSummaryRepository(testDB),
		adapter, &intIAMStub{"coppa@test.com", "COPPA Fam"},
		shared.NewEventBus(), intTestConfig(),
	)
	scope := shared.NewFamilyScopeFromID(familyID)

	resp, err := svc.ProcessCoppaVerification(ctx, CoppaVerificationCommand{PaymentMethodID: "pm_coppa_int"}, scope)
	require.NoError(t, err)
	assert.True(t, resp.Verified)
	assert.Equal(t, "pay_coppa_int", resp.ChargeID)
	assert.Equal(t, "ref_coppa_int", resp.RefundID)

	txns, err := txnRepo.ListByFamily(ctx, scope, &TransactionListParams{})
	require.NoError(t, err)
	require.Len(t, txns, 2, "expected charge + refund transaction rows")

	types := map[string]bool{}
	for _, tx := range txns {
		types[tx.TransactionType] = true
		assert.Equal(t, TransactionStatusSucceeded, tx.Status)
		assert.Equal(t, int64(50), tx.AmountCents)
	}
	assert.True(t, types[TransactionTypeCoppaCharge], "missing coppa_charge row")
	assert.True(t, types[TransactionTypeCoppaRefund], "missing coppa_refund row")
}

// TestBillingIntegration_Coppa_SingleTransaction_WhenRefundFails verifies that
// only the charge row is persisted when the adapter returns an empty refundID.
func TestBillingIntegration_Coppa_SingleTransaction_WhenRefundFails(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)

	adapter := new(mockAdapter)
	adapter.On("CreateCustomer", mock.Anything, "coppa2@test.com", "COPPA2", mock.Anything).Return("cus_coppa2_int", nil)
	adapter.On("ProcessMicroCharge", mock.Anything, "cus_coppa2_int", "pm_coppa2", int64(50), mock.Anything, mock.Anything).
		Return("pay_coppa2_int", "", nil) // empty refundID → refund failed

	txnRepo := NewPgTransactionRepository(testDB)
	svc := NewBillingService(
		NewPgSubscriptionRepository(testDB), txnRepo,
		NewPgCustomerRepository(testDB), NewPgPayoutRepository(testDB),
		NewPgCreatorTaxSummaryRepository(testDB),
		adapter, &intIAMStub{"coppa2@test.com", "COPPA2"},
		shared.NewEventBus(), intTestConfig(),
	)
	scope := shared.NewFamilyScopeFromID(familyID)

	resp, err := svc.ProcessCoppaVerification(ctx, CoppaVerificationCommand{PaymentMethodID: "pm_coppa2"}, scope)
	require.NoError(t, err)
	assert.True(t, resp.Verified)
	assert.Empty(t, resp.RefundID)

	txns, err := txnRepo.ListByFamily(ctx, scope, &TransactionListParams{})
	require.NoError(t, err)
	require.Len(t, txns, 1, "only charge row expected when refund fails")
	assert.Equal(t, TransactionTypeCoppaCharge, txns[0].TransactionType)
}

// ═══════════════════════════════════════════════════════════════════════════════
// §AC-3: Webhook Processing Pipeline (real HMAC verify + real DB)
// ═══════════════════════════════════════════════════════════════════════════════

// TestBillingIntegration_Webhook_RejectsInvalidSignature verifies the real
// HMAC verifier rejects tampered payloads with ErrInvalidWebhookSignature.
func TestBillingIntegration_Webhook_RejectsInvalidSignature(t *testing.T) {
	skipIfNoTestDB(t)
	svc := buildWebhookService(t, shared.NewEventBus())
	err := svc.ProcessHyperswitchWebhook(context.Background(), []byte(`{"type":"invoice.paid","content":{}}`), "bad-sig")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidWebhookSignature)
}

// TestBillingIntegration_Webhook_SubscriptionCreated_InsertsRow verifies that
// a subscription.created webhook event creates a DB row when none exists.
func TestBillingIntegration_Webhook_SubscriptionCreated_InsertsRow(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)
	hsSubID := "sub_wh_created_" + familyID.String()[:8]
	hsCustID := "cus_wh_created_" + familyID.String()[:8]

	// The subscription.created handler looks up the customer by HS ID to get family_id.
	require.NoError(t, testDB.Exec(
		`INSERT INTO bill_hyperswitch_customers (family_id, hyperswitch_customer_id) VALUES (?, ?)`,
		familyID, hsCustID,
	).Error)

	svc := buildWebhookService(t, shared.NewEventBus())
	now := time.Now()
	content := HyperswitchSubscription{
		ID: hsSubID, CustomerID: hsCustID,
		Status:             SubscriptionStatusIncomplete,
		CurrentPeriodStart: now, CurrentPeriodEnd: now.Add(30 * 24 * time.Hour),
		AmountCents: 1499, Currency: "usd", PriceID: "price_monthly_test",
	}
	payload := makeIntWebhookPayload(t, "subscription.created", content)
	require.NoError(t, svc.ProcessHyperswitchWebhook(ctx, payload, signIntWebhook(payload)))

	sub, err := NewPgSubscriptionRepository(testDB).FindByHyperswitchID(ctx, hsSubID)
	require.NoError(t, err)
	assert.Equal(t, hsSubID, sub.HyperswitchSubscriptionID)
	assert.Equal(t, SubscriptionStatusIncomplete, sub.Status)
}

// TestBillingIntegration_Webhook_SubscriptionActivation_TransitionsToActive verifies
// that subscription.updated with status=active promotes an incomplete row to active
// and publishes a SubscriptionCreated domain event. [10-billing §5, §14]
func TestBillingIntegration_Webhook_SubscriptionActivation_TransitionsToActive(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)
	hsSubID := "sub_wh_activate_" + familyID.String()[:8]

	subRepo := NewPgSubscriptionRepository(testDB)
	now := time.Now()
	_, err := subRepo.Create(ctx, CreateSubscriptionRow{
		FamilyID: familyID, HyperswitchSubscriptionID: hsSubID,
		HyperswitchCustomerID: "cus_wh_activate", Tier: TierPremium,
		Status: SubscriptionStatusIncomplete, BillingInterval: IntervalMonthly,
		CurrentPeriodStart: now, CurrentPeriodEnd: now.Add(30 * 24 * time.Hour),
		AmountCents: 1499, Currency: "usd", HyperswitchPriceID: "price_monthly_test",
	})
	require.NoError(t, err)

	var capturedEvent shared.DomainEvent
	bus := shared.NewEventBus()
	bus.Subscribe(reflect.TypeOf(SubscriptionCreated{}), &intCaptureHandler{captured: &capturedEvent})

	svc := buildWebhookService(t, bus)
	content := HyperswitchSubscription{
		ID: hsSubID, Status: SubscriptionStatusActive,
		CurrentPeriodStart: now, CurrentPeriodEnd: now.Add(30 * 24 * time.Hour),
		AmountCents: 1499, PriceID: "price_monthly_test",
	}
	payload := makeIntWebhookPayload(t, "subscription.updated", content)
	require.NoError(t, svc.ProcessHyperswitchWebhook(ctx, payload, signIntWebhook(payload)))

	dbSub, err := subRepo.FindByHyperswitchID(ctx, hsSubID)
	require.NoError(t, err)
	assert.Equal(t, SubscriptionStatusActive, dbSub.Status)

	require.NotNil(t, capturedEvent, "SubscriptionCreated event must be published on first activation")
	assert.Equal(t, "billing.SubscriptionCreated", capturedEvent.EventName())
}

// TestBillingIntegration_Webhook_SubscriptionDeleted_TransitionsToCanceled verifies
// subscription.deleted marks the row as canceled and publishes SubscriptionCancelled.
func TestBillingIntegration_Webhook_SubscriptionDeleted_TransitionsToCanceled(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)
	hsSubID := "sub_wh_del_" + familyID.String()[:8]
	_ = seedActiveSub(t, familyID, hsSubID, "cus_wh_del")

	var capturedEvent shared.DomainEvent
	bus := shared.NewEventBus()
	bus.Subscribe(reflect.TypeOf(SubscriptionCancelled{}), &intCaptureHandler{captured: &capturedEvent})

	svc := buildWebhookService(t, bus)
	payload := makeIntWebhookPayload(t, "subscription.deleted", map[string]string{"subscription_id": hsSubID})
	require.NoError(t, svc.ProcessHyperswitchWebhook(ctx, payload, signIntWebhook(payload)))

	sub, err := NewPgSubscriptionRepository(testDB).FindByHyperswitchID(ctx, hsSubID)
	require.NoError(t, err)
	assert.Equal(t, SubscriptionStatusCanceled, sub.Status)

	require.NotNil(t, capturedEvent, "SubscriptionCancelled event must be published")
	assert.Equal(t, "billing.SubscriptionCancelled", capturedEvent.EventName())
}

// TestBillingIntegration_Webhook_InvoicePaid_PersistsTransaction verifies
// invoice.paid creates a subscription_payment row in bill_transactions.
func TestBillingIntegration_Webhook_InvoicePaid_PersistsTransaction(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)
	hsSubID := "sub_wh_inv_" + familyID.String()[:8]
	_ = seedActiveSub(t, familyID, hsSubID, "cus_wh_inv")

	txnRepo := NewPgTransactionRepository(testDB)
	svc := buildWebhookService(t, shared.NewEventBus())
	payID := "pay_wh_inv_" + familyID.String()[:8]

	content := BillingWebhookInvoicePaid{
		InvoiceID: "inv_wh_test", SubscriptionID: hsSubID,
		AmountCents: 1499, PaymentID: payID,
	}
	payload := makeIntWebhookPayload(t, "invoice.paid", content)
	require.NoError(t, svc.ProcessHyperswitchWebhook(ctx, payload, signIntWebhook(payload)))

	scope := shared.NewFamilyScopeFromID(familyID)
	txns, err := txnRepo.ListByFamily(ctx, scope, &TransactionListParams{})
	require.NoError(t, err)
	require.Len(t, txns, 1)
	assert.Equal(t, TransactionTypeSubscriptionPayment, txns[0].TransactionType)
	assert.Equal(t, TransactionStatusSucceeded, txns[0].Status)
	assert.Equal(t, int64(1499), txns[0].AmountCents)
	require.NotNil(t, txns[0].HyperswitchPaymentID)
	assert.Equal(t, payID, *txns[0].HyperswitchPaymentID)
}

// TestBillingIntegration_Webhook_InvoicePaid_IdempotentOnDuplicate verifies that
// a second invoice.paid with the same payment_id is a no-op. [10-billing §14]
func TestBillingIntegration_Webhook_InvoicePaid_IdempotentOnDuplicate(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)
	hsSubID := "sub_wh_idem_" + familyID.String()[:8]
	_ = seedActiveSub(t, familyID, hsSubID, "cus_wh_idem")

	txnRepo := NewPgTransactionRepository(testDB)
	svc := buildWebhookService(t, shared.NewEventBus())
	payID := "pay_dup_" + familyID.String()[:8]

	content := BillingWebhookInvoicePaid{
		InvoiceID: "inv_dup", SubscriptionID: hsSubID, AmountCents: 1499, PaymentID: payID,
	}
	payload := makeIntWebhookPayload(t, "invoice.paid", content)
	sig := signIntWebhook(payload)

	require.NoError(t, svc.ProcessHyperswitchWebhook(ctx, payload, sig)) // first delivery
	require.NoError(t, svc.ProcessHyperswitchWebhook(ctx, payload, sig)) // duplicate delivery

	scope := shared.NewFamilyScopeFromID(familyID)
	txns, err := txnRepo.ListByFamily(ctx, scope, &TransactionListParams{})
	require.NoError(t, err)
	assert.Len(t, txns, 1, "duplicate invoice.paid must not insert a second row")
}

// TestBillingIntegration_Webhook_PaymentFailed_SetsPastDue verifies that
// payment.failed transitions the subscription status to past_due in the DB.
func TestBillingIntegration_Webhook_PaymentFailed_SetsPastDue(t *testing.T) {
	skipIfNoTestDB(t)
	ctx := context.Background()
	familyID := seedIntTestFamily(t)
	hsSubID := "sub_wh_fail_" + familyID.String()[:8]
	_ = seedActiveSub(t, familyID, hsSubID, "cus_wh_fail")

	svc := buildWebhookService(t, shared.NewEventBus())
	content := BillingWebhookPaymentFailed{
		PaymentID: "pay_fail_int", SubscriptionID: &hsSubID, Reason: "insufficient_funds",
	}
	payload := makeIntWebhookPayload(t, "payment.failed", content)
	require.NoError(t, svc.ProcessHyperswitchWebhook(ctx, payload, signIntWebhook(payload)))

	sub, err := NewPgSubscriptionRepository(testDB).FindByHyperswitchID(ctx, hsSubID)
	require.NoError(t, err)
	assert.Equal(t, SubscriptionStatusPastDue, sub.Status)
}
