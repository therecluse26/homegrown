package billing

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Mocks
// ═══════════════════════════════════════════════════════════════════════════════

// --- SubscriptionRepository mock ---

type mockSubscriptionRepo struct{ mock.Mock }

func (m *mockSubscriptionRepo) Create(ctx context.Context, input CreateSubscriptionRow) (*BillSubscription, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillSubscription), args.Error(1)
}

func (m *mockSubscriptionRepo) FindByFamily(ctx context.Context, scope shared.FamilyScope) (*BillSubscription, error) {
	args := m.Called(ctx, scope)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillSubscription), args.Error(1)
}

func (m *mockSubscriptionRepo) FindByHyperswitchID(ctx context.Context, hsSubID string) (*BillSubscription, error) {
	args := m.Called(ctx, hsSubID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillSubscription), args.Error(1)
}

func (m *mockSubscriptionRepo) Update(ctx context.Context, subID uuid.UUID, updates SubscriptionUpdate) (*BillSubscription, error) {
	args := m.Called(ctx, subID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillSubscription), args.Error(1)
}

func (m *mockSubscriptionRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return m.Called(ctx, familyID).Error(0)
}

// --- TransactionRepository mock ---

type mockTransactionRepo struct{ mock.Mock }

func (m *mockTransactionRepo) Create(ctx context.Context, input CreateTransactionRow) (*BillTransaction, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillTransaction), args.Error(1)
}

func (m *mockTransactionRepo) ListByFamily(ctx context.Context, scope shared.FamilyScope, params *TransactionListParams) ([]BillTransaction, error) {
	args := m.Called(ctx, scope, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]BillTransaction), args.Error(1)
}

func (m *mockTransactionRepo) ExistsByPaymentID(ctx context.Context, paymentID string, txnType string) (bool, error) {
	args := m.Called(ctx, paymentID, txnType)
	return args.Bool(0), args.Error(1)
}

// --- CustomerRepository mock ---

type mockCustomerRepo struct{ mock.Mock }

func (m *mockCustomerRepo) Upsert(ctx context.Context, familyID uuid.UUID, input UpsertCustomerRow) (*BillHyperswitchCustomer, error) {
	args := m.Called(ctx, familyID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillHyperswitchCustomer), args.Error(1)
}

func (m *mockCustomerRepo) FindByFamily(ctx context.Context, familyID uuid.UUID) (*BillHyperswitchCustomer, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillHyperswitchCustomer), args.Error(1)
}

func (m *mockCustomerRepo) FindByHyperswitchID(ctx context.Context, hsCustomerID string) (*BillHyperswitchCustomer, error) {
	args := m.Called(ctx, hsCustomerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillHyperswitchCustomer), args.Error(1)
}

// --- PayoutRepository mock ---

type mockPayoutRepo struct{ mock.Mock }

func (m *mockPayoutRepo) Create(ctx context.Context, input CreatePayoutRow) (*BillPayout, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillPayout), args.Error(1)
}

func (m *mockPayoutRepo) ListByCreator(ctx context.Context, creatorID uuid.UUID, params *PayoutListParams) ([]BillPayout, error) {
	args := m.Called(ctx, creatorID, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]BillPayout), args.Error(1)
}

func (m *mockPayoutRepo) UpdateStatus(ctx context.Context, payoutID uuid.UUID, status string, hpID *string) (*BillPayout, error) {
	args := m.Called(ctx, payoutID, status, hpID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillPayout), args.Error(1)
}

func (m *mockPayoutRepo) FindPending(ctx context.Context, limit uint32) ([]BillPayout, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]BillPayout), args.Error(1)
}

// --- SubscriptionPaymentAdapter mock ---

type mockAdapter struct{ mock.Mock }

func (m *mockAdapter) CreateCustomer(ctx context.Context, email, name string, metadata map[string]string) (string, error) {
	args := m.Called(ctx, email, name, metadata)
	return args.String(0), args.Error(1)
}

func (m *mockAdapter) UpdateCustomer(ctx context.Context, customerID, email, name string) error {
	return m.Called(ctx, customerID, email, name).Error(0)
}

func (m *mockAdapter) CreateSubscription(ctx context.Context, customerID, priceID, paymentMethodID string, metadata map[string]string) (*HyperswitchSubscription, error) {
	args := m.Called(ctx, customerID, priceID, paymentMethodID, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HyperswitchSubscription), args.Error(1)
}

func (m *mockAdapter) UpdateSubscription(ctx context.Context, subID, newPriceID string) (*HyperswitchSubscription, error) {
	args := m.Called(ctx, subID, newPriceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HyperswitchSubscription), args.Error(1)
}

func (m *mockAdapter) CancelSubscription(ctx context.Context, subID string) (*HyperswitchSubscription, error) {
	args := m.Called(ctx, subID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HyperswitchSubscription), args.Error(1)
}

func (m *mockAdapter) PauseSubscription(ctx context.Context, subID string) (*HyperswitchSubscription, error) {
	args := m.Called(ctx, subID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HyperswitchSubscription), args.Error(1)
}

func (m *mockAdapter) ResumeSubscription(ctx context.Context, subID string) (*HyperswitchSubscription, error) {
	args := m.Called(ctx, subID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HyperswitchSubscription), args.Error(1)
}

func (m *mockAdapter) ReactivateSubscription(ctx context.Context, subID string) (*HyperswitchSubscription, error) {
	args := m.Called(ctx, subID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HyperswitchSubscription), args.Error(1)
}

func (m *mockAdapter) EstimateSubscription(ctx context.Context, customerID, priceID string, currentSubID *string) (*HyperswitchEstimate, error) {
	args := m.Called(ctx, customerID, priceID, currentSubID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HyperswitchEstimate), args.Error(1)
}

func (m *mockAdapter) CreateSetupIntent(ctx context.Context, customerID string) (*SetupIntentResponse, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SetupIntentResponse), args.Error(1)
}

func (m *mockAdapter) ListPaymentMethods(ctx context.Context, customerID string) ([]HyperswitchPaymentMethod, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]HyperswitchPaymentMethod), args.Error(1)
}

func (m *mockAdapter) DetachPaymentMethod(ctx context.Context, pmID string) error {
	return m.Called(ctx, pmID).Error(0)
}

func (m *mockAdapter) ProcessMicroCharge(ctx context.Context, customerID, paymentMethodID string, amountCents int64, description string, metadata map[string]string) (string, string, error) {
	args := m.Called(ctx, customerID, paymentMethodID, amountCents, description, metadata)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockAdapter) ListInvoices(ctx context.Context, customerID string, limit uint32) ([]HyperswitchInvoice, error) {
	args := m.Called(ctx, customerID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]HyperswitchInvoice), args.Error(1)
}

func (m *mockAdapter) CreatePayout(ctx context.Context, paymentAccountID string, amountCents int64, currency string, metadata map[string]string) (*HyperswitchPayout, error) {
	args := m.Called(ctx, paymentAccountID, amountCents, currency, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HyperswitchPayout), args.Error(1)
}

func (m *mockAdapter) VerifyWebhook(ctx context.Context, payload []byte, signature string) (bool, error) {
	args := m.Called(ctx, payload, signature)
	return args.Bool(0), args.Error(1)
}

func (m *mockAdapter) ParseWebhookEvent(ctx context.Context, payload []byte) (*BillingWebhookEvent, error) {
	args := m.Called(ctx, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BillingWebhookEvent), args.Error(1)
}

// --- IamServiceForBilling mock ---

type mockIamService struct{ mock.Mock }

func (m *mockIamService) GetFamilyPrimaryEmail(ctx context.Context, familyID uuid.UUID) (string, string, error) {
	args := m.Called(ctx, familyID)
	return args.String(0), args.String(1), args.Error(2)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func newTestService(subRepo *mockSubscriptionRepo, txnRepo *mockTransactionRepo, custRepo *mockCustomerRepo, payoutRepo *mockPayoutRepo, adapter *mockAdapter, iam *mockIamService) *BillingServiceImpl {
	return NewBillingService(
		subRepo, txnRepo, custRepo, payoutRepo,
		adapter, iam, shared.NewEventBus(),
		BillingConfig{
			MonthlyPriceID:       "price_monthly_v1",
			AnnualPriceID:        "price_annual_v1",
			CoppaChargeCents:     50,
			WebhookSigningSecret: "test-secret",
		},
	)
}

func testScope() shared.FamilyScope {
	return shared.NewFamilyScopeFromAuth(&shared.AuthContext{
		FamilyID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	})
}

func testFamilyID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func activeSub() *BillSubscription {
	now := time.Now()
	return &BillSubscription{
		ID:                        uuid.Must(uuid.NewV7()),
		FamilyID:                  testFamilyID(),
		HyperswitchSubscriptionID: "sub_test_123",
		HyperswitchCustomerID:     "cus_test_123",
		Tier:                      TierPremium,
		Status:                    SubscriptionStatusActive,
		BillingInterval:           IntervalMonthly,
		CurrentPeriodStart:        now.Add(-30 * 24 * time.Hour),
		CurrentPeriodEnd:          now.Add(30 * 24 * time.Hour),
		AmountCents:               1499,
		Currency:                  "usd",
		HyperswitchPriceID:        "price_monthly_v1",
	}
}

func testCustomer() *BillHyperswitchCustomer {
	return &BillHyperswitchCustomer{
		FamilyID:              testFamilyID(),
		HyperswitchCustomerID: "cus_test_123",
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// 1-2: GetSubscription
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetSubscription_ReturnsFreeDefault_WhenNoSubscription(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(nil, nil)

	resp, err := svc.GetSubscription(context.Background(), testScope())
	require.NoError(t, err)
	assert.Equal(t, TierFree, resp.Tier)
	assert.Nil(t, resp.Status)
	assert.False(t, resp.CancelAtPeriodEnd)
}

func TestGetSubscription_ReturnsFullDetails_WhenSubscriptionExists(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	sub := activeSub()
	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(sub, nil)

	resp, err := svc.GetSubscription(context.Background(), testScope())
	require.NoError(t, err)
	assert.Equal(t, TierPremium, resp.Tier)
	require.NotNil(t, resp.Status)
	assert.Equal(t, SubscriptionStatusActive, *resp.Status)
	require.NotNil(t, resp.BillingInterval)
	assert.Equal(t, IntervalMonthly, *resp.BillingInterval)
	require.NotNil(t, resp.AmountCents)
	assert.Equal(t, int64(1499), *resp.AmountCents)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 3-5: ListTransactions
// ═══════════════════════════════════════════════════════════════════════════════

func TestListTransactions_ReturnsPaginatedTransactions(t *testing.T) {
	txnRepo := new(mockTransactionRepo)
	svc := newTestService(new(mockSubscriptionRepo), txnRepo, new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	txns := []BillTransaction{
		{ID: uuid.Must(uuid.NewV7()), FamilyID: testFamilyID(), TransactionType: TransactionTypeSubscriptionPayment, Status: TransactionStatusSucceeded, AmountCents: 1499, Currency: "usd", CreatedAt: time.Now()},
	}
	params := TransactionListParams{}
	txnRepo.On("ListByFamily", mock.Anything, testScope(), &params).Return(txns, nil)

	resp, err := svc.ListTransactions(context.Background(), params, testScope())
	require.NoError(t, err)
	assert.Len(t, resp.Transactions, 1)
	assert.Equal(t, TransactionTypeSubscriptionPayment, resp.Transactions[0].TransactionType)
	assert.Nil(t, resp.NextCursor)
}

func TestListTransactions_RespectsLimit_WithCursor(t *testing.T) {
	txnRepo := new(mockTransactionRepo)
	svc := newTestService(new(mockSubscriptionRepo), txnRepo, new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	limit := 1
	params := TransactionListParams{Limit: &limit}

	// Return 2 items when limit is 1 (repo fetches limit+1 for hasMore)
	txns := []BillTransaction{
		{ID: uuid.Must(uuid.NewV7()), TransactionType: TransactionTypeCoppaCharge, Status: TransactionStatusSucceeded, AmountCents: 50, Currency: "usd", CreatedAt: time.Now()},
		{ID: uuid.Must(uuid.NewV7()), TransactionType: TransactionTypeCoppaRefund, Status: TransactionStatusSucceeded, AmountCents: 50, Currency: "usd", CreatedAt: time.Now().Add(-time.Second)},
	}
	txnRepo.On("ListByFamily", mock.Anything, testScope(), &params).Return(txns, nil)

	resp, err := svc.ListTransactions(context.Background(), params, testScope())
	require.NoError(t, err)
	assert.Len(t, resp.Transactions, 1)
	assert.NotNil(t, resp.NextCursor)
}

func TestListTransactions_ReturnsEmptyList_WhenNoTransactions(t *testing.T) {
	txnRepo := new(mockTransactionRepo)
	svc := newTestService(new(mockSubscriptionRepo), txnRepo, new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	params := TransactionListParams{}
	txnRepo.On("ListByFamily", mock.Anything, testScope(), &params).Return([]BillTransaction{}, nil)

	resp, err := svc.ListTransactions(context.Background(), params, testScope())
	require.NoError(t, err)
	assert.Empty(t, resp.Transactions)
	assert.Nil(t, resp.NextCursor)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 6-7: ListInvoices
// ═══════════════════════════════════════════════════════════════════════════════

func TestListInvoices_ReturnsInvoicesFromAdapter(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("ListInvoices", mock.Anything, "cus_test_123", uint32(20)).Return([]HyperswitchInvoice{
		{ID: "inv_1", AmountCents: 1499, Currency: "usd", Status: "paid"},
	}, nil)

	resp, err := svc.ListInvoices(context.Background(), InvoiceListParams{}, testScope())
	require.NoError(t, err)
	assert.Len(t, resp.Invoices, 1)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 7-8: ListPaymentMethods
// ═══════════════════════════════════════════════════════════════════════════════

func TestListPaymentMethods_ReturnsMethodsFromAdapter(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("ListPaymentMethods", mock.Anything, "cus_test_123").Return([]HyperswitchPaymentMethod{
		{ID: "pm_1", MethodType: "card", IsDefault: true},
	}, nil)

	methods, err := svc.ListPaymentMethods(context.Background(), testScope())
	require.NoError(t, err)
	assert.Len(t, methods, 1)
	assert.Equal(t, "pm_1", methods[0].ID)
}

func TestListPaymentMethods_ReturnsEmptyList_WhenNoCustomer(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), custRepo, new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(nil, nil)

	methods, err := svc.ListPaymentMethods(context.Background(), testScope())
	require.NoError(t, err)
	assert.Empty(t, methods)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 9-10: EstimateSubscription
// ═══════════════════════════════════════════════════════════════════════════════

func TestEstimateSubscription_ReturnsPricingFromAdapter(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	subRepo := new(mockSubscriptionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(nil, nil)
	adapter.On("EstimateSubscription", mock.Anything, "cus_test_123", "price_annual_v1", (*string)(nil)).Return(&HyperswitchEstimate{
		AmountCents:           11999,
		Currency:              "usd",
		ProrationCreditsCents: 0,
		TotalDueTodayCents:    11999,
		NextBillingDate:       time.Now().Add(365 * 24 * time.Hour),
	}, nil)

	resp, err := svc.EstimateSubscription(context.Background(), EstimateSubscriptionQuery{BillingInterval: IntervalAnnual}, testScope())
	require.NoError(t, err)
	assert.Equal(t, int64(11999), resp.AmountCents)
	assert.Equal(t, IntervalAnnual, resp.BillingInterval)
}

func TestEstimateSubscription_ResolvesCorrectPriceID(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	subRepo := new(mockSubscriptionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(nil, nil)
	adapter.On("EstimateSubscription", mock.Anything, "cus_test_123", "price_monthly_v1", (*string)(nil)).Return(&HyperswitchEstimate{
		AmountCents: 1499, Currency: "usd",
	}, nil)

	resp, err := svc.EstimateSubscription(context.Background(), EstimateSubscriptionQuery{BillingInterval: IntervalMonthly}, testScope())
	require.NoError(t, err)
	assert.Equal(t, int64(1499), resp.AmountCents)
	assert.Equal(t, IntervalMonthly, resp.BillingInterval)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 11: ListPayouts
// ═══════════════════════════════════════════════════════════════════════════════

func TestListPayouts_ReturnsPaginatedPayouts(t *testing.T) {
	payoutRepo := new(mockPayoutRepo)
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), new(mockCustomerRepo), payoutRepo, new(mockAdapter), new(mockIamService))

	creatorID := uuid.Must(uuid.NewV7())
	params := PayoutListParams{}
	payoutRepo.On("ListByCreator", mock.Anything, creatorID, &params).Return([]BillPayout{
		{ID: uuid.Must(uuid.NewV7()), CreatorID: creatorID, Status: PayoutStatusCompleted, AmountCents: 5000, Currency: "usd", CreatedAt: time.Now()},
	}, nil)

	resp, err := svc.ListPayouts(context.Background(), params, creatorID)
	require.NoError(t, err)
	assert.Len(t, resp.Payouts, 1)
	assert.Nil(t, resp.NextCursor)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 12-17: CreateSubscription
// ═══════════════════════════════════════════════════════════════════════════════

func TestCreateSubscription_CreatesCustomerIfNotExists(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	iamSvc := new(mockIamService)
	svc := newTestService(subRepo, new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, iamSvc)

	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(nil, nil)
	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(nil, nil)
	iamSvc.On("GetFamilyPrimaryEmail", mock.Anything, testFamilyID()).Return("test@example.com", "Test Family", nil)
	adapter.On("CreateCustomer", mock.Anything, "test@example.com", "Test Family", mock.Anything).Return("cus_new_123", nil)
	custRepo.On("Upsert", mock.Anything, testFamilyID(), mock.Anything).Return(testCustomer(), nil)

	hsSub := &HyperswitchSubscription{
		ID:                 "sub_new_123",
		CustomerID:         "cus_new_123",
		Status:             SubscriptionStatusIncomplete,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		AmountCents:        1499,
		Currency:           "usd",
	}
	adapter.On("CreateSubscription", mock.Anything, "cus_new_123", "price_monthly_v1", "pm_test", mock.Anything).Return(hsSub, nil)

	sub := &BillSubscription{
		ID: uuid.Must(uuid.NewV7()), FamilyID: testFamilyID(), Status: SubscriptionStatusIncomplete,
		Tier: TierPremium, BillingInterval: IntervalMonthly, AmountCents: 1499, Currency: "usd",
		CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
	}
	subRepo.On("Create", mock.Anything, mock.Anything).Return(sub, nil)

	resp, err := svc.CreateSubscription(context.Background(), CreateSubscriptionCommand{
		BillingInterval: IntervalMonthly,
		PaymentMethodID: "pm_test",
	}, testScope())
	require.NoError(t, err)
	assert.Equal(t, TierPremium, resp.Tier)
	adapter.AssertCalled(t, "CreateCustomer", mock.Anything, "test@example.com", "Test Family", mock.Anything)
}

func TestCreateSubscription_CreatesLocalRow_WithIncompleteStatus(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	iamSvc := new(mockIamService)
	svc := newTestService(subRepo, new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, iamSvc)

	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(nil, nil)
	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("CreateSubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&HyperswitchSubscription{
		ID: "sub_123", CustomerID: "cus_test_123", Status: SubscriptionStatusIncomplete,
		CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
		AmountCents: 1499, Currency: "usd",
	}, nil)

	resultSub := &BillSubscription{
		ID: uuid.Must(uuid.NewV7()), Status: SubscriptionStatusIncomplete, Tier: TierPremium,
		BillingInterval: IntervalMonthly, AmountCents: 1499, Currency: "usd",
		CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
	}
	subRepo.On("Create", mock.Anything, mock.MatchedBy(func(input CreateSubscriptionRow) bool {
		return input.Status == SubscriptionStatusIncomplete && input.Tier == TierPremium
	})).Return(resultSub, nil)

	resp, err := svc.CreateSubscription(context.Background(), CreateSubscriptionCommand{
		BillingInterval: IntervalMonthly,
		PaymentMethodID: "pm_test",
	}, testScope())
	require.NoError(t, err)
	require.NotNil(t, resp.Status)
	assert.Equal(t, SubscriptionStatusIncomplete, *resp.Status)
}

func TestCreateSubscription_ReturnsAlreadyExists_WhenFamilyHasSubscription(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(activeSub(), nil)

	_, err := svc.CreateSubscription(context.Background(), CreateSubscriptionCommand{
		BillingInterval: IntervalMonthly, PaymentMethodID: "pm_test",
	}, testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrSubscriptionAlreadyExists))
}

func TestCreateSubscription_ReturnsPaymentDeclined_WhenPaymentFails(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(nil, nil)
	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("CreateSubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, ErrPaymentDeclined)

	_, err := svc.CreateSubscription(context.Background(), CreateSubscriptionCommand{
		BillingInterval: IntervalMonthly, PaymentMethodID: "pm_test",
	}, testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrPaymentDeclined))
}

func TestCreateSubscription_ReturnsAdapterUnavailable_WhenAdapterDown(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(nil, nil)
	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("CreateSubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, ErrPaymentAdapterUnavailable)

	_, err := svc.CreateSubscription(context.Background(), CreateSubscriptionCommand{
		BillingInterval: IntervalMonthly, PaymentMethodID: "pm_test",
	}, testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrPaymentAdapterUnavailable))
}

// ═══════════════════════════════════════════════════════════════════════════════
// 18-20: UpdateSubscription
// ═══════════════════════════════════════════════════════════════════════════════

func TestUpdateSubscription_CallsAdapterWithNewPriceID(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), adapter, new(mockIamService))

	sub := activeSub()
	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(sub, nil)
	adapter.On("UpdateSubscription", mock.Anything, sub.HyperswitchSubscriptionID, "price_annual_v1").Return(&HyperswitchSubscription{}, nil)
	subRepo.On("Update", mock.Anything, sub.ID, mock.Anything).Return(sub, nil)

	_, err := svc.UpdateSubscription(context.Background(), UpdateSubscriptionCommand{BillingInterval: IntervalAnnual}, testScope())
	require.NoError(t, err)
	adapter.AssertCalled(t, "UpdateSubscription", mock.Anything, sub.HyperswitchSubscriptionID, "price_annual_v1")
}

func TestUpdateSubscription_ReturnsNotFound_WhenNoSubscription(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(nil, nil)

	_, err := svc.UpdateSubscription(context.Background(), UpdateSubscriptionCommand{BillingInterval: IntervalAnnual}, testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrSubscriptionNotFound))
}

func TestUpdateSubscription_ReturnsNotActive_WhenSubscriptionInactive(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	sub := activeSub()
	sub.Status = SubscriptionStatusCanceled
	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(sub, nil)

	_, err := svc.UpdateSubscription(context.Background(), UpdateSubscriptionCommand{BillingInterval: IntervalAnnual}, testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrSubscriptionNotActive))
}

// ═══════════════════════════════════════════════════════════════════════════════
// 21-24: CancelSubscription
// ═══════════════════════════════════════════════════════════════════════════════

func TestCancelSubscription_CallsAdapterToCancel(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), adapter, new(mockIamService))

	sub := activeSub()
	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(sub, nil)
	adapter.On("CancelSubscription", mock.Anything, sub.HyperswitchSubscriptionID).Return(&HyperswitchSubscription{}, nil)

	updatedSub := *sub
	updatedSub.CancelAtPeriodEnd = true
	subRepo.On("Update", mock.Anything, sub.ID, mock.Anything).Return(&updatedSub, nil)

	resp, err := svc.CancelSubscription(context.Background(), testScope())
	require.NoError(t, err)
	assert.True(t, resp.CancelAtPeriodEnd)
	adapter.AssertCalled(t, "CancelSubscription", mock.Anything, sub.HyperswitchSubscriptionID)
}

func TestCancelSubscription_SetsCancelAtPeriodEnd(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), adapter, new(mockIamService))

	sub := activeSub()
	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(sub, nil)
	adapter.On("CancelSubscription", mock.Anything, mock.Anything).Return(&HyperswitchSubscription{}, nil)

	updatedSub := *sub
	updatedSub.CancelAtPeriodEnd = true
	subRepo.On("Update", mock.Anything, sub.ID, mock.MatchedBy(func(u SubscriptionUpdate) bool {
		return u.CancelAtPeriodEnd != nil && *u.CancelAtPeriodEnd && u.CanceledAt != nil
	})).Return(&updatedSub, nil)

	resp, err := svc.CancelSubscription(context.Background(), testScope())
	require.NoError(t, err)
	assert.True(t, resp.CancelAtPeriodEnd)
}

func TestCancelSubscription_ReturnsNotFound(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(nil, nil)

	_, err := svc.CancelSubscription(context.Background(), testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrSubscriptionNotFound))
}

func TestCancelSubscription_ReturnsNotActive_WhenAlreadyCanceled(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	sub := activeSub()
	sub.CancelAtPeriodEnd = true
	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(sub, nil)

	_, err := svc.CancelSubscription(context.Background(), testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrSubscriptionNotActive))
}

// ═══════════════════════════════════════════════════════════════════════════════
// 25-27: ReactivateSubscription
// ═══════════════════════════════════════════════════════════════════════════════

func TestReactivateSubscription_CallsAdapterAndClearsCancellation(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), adapter, new(mockIamService))

	sub := activeSub()
	sub.CancelAtPeriodEnd = true
	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(sub, nil)
	adapter.On("ReactivateSubscription", mock.Anything, sub.HyperswitchSubscriptionID).Return(&HyperswitchSubscription{}, nil)

	updatedSub := *sub
	updatedSub.CancelAtPeriodEnd = false
	subRepo.On("Update", mock.Anything, sub.ID, mock.MatchedBy(func(u SubscriptionUpdate) bool {
		return u.CancelAtPeriodEnd != nil && !*u.CancelAtPeriodEnd
	})).Return(&updatedSub, nil)

	resp, err := svc.ReactivateSubscription(context.Background(), testScope())
	require.NoError(t, err)
	assert.False(t, resp.CancelAtPeriodEnd)
	adapter.AssertCalled(t, "ReactivateSubscription", mock.Anything, sub.HyperswitchSubscriptionID)
}

func TestReactivateSubscription_ReturnsCannotReactivate_WhenNotPendingCancellation(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	sub := activeSub() // CancelAtPeriodEnd is false
	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(sub, nil)

	_, err := svc.ReactivateSubscription(context.Background(), testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrCannotReactivate))
}

// ═══════════════════════════════════════════════════════════════════════════════
// 28-30: Payment Methods
// ═══════════════════════════════════════════════════════════════════════════════

func TestAttachPaymentMethod_CreatesSetupIntentViaAdapter(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("CreateSetupIntent", mock.Anything, "cus_test_123").Return(&SetupIntentResponse{ClientSecret: "seti_secret"}, nil)

	resp, err := svc.AttachPaymentMethod(context.Background(), AttachPaymentMethodCommand{}, testScope())
	require.NoError(t, err)
	assert.Equal(t, "seti_secret", resp.ID)
}

func TestDetachPaymentMethod_CallsAdapterToDetach(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), adapter, new(mockIamService))

	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(nil, nil) // no active sub

	adapter.On("DetachPaymentMethod", mock.Anything, "pm_123").Return(nil)

	err := svc.DetachPaymentMethod(context.Background(), "pm_123", testScope())
	require.NoError(t, err)
	adapter.AssertCalled(t, "DetachPaymentMethod", mock.Anything, "pm_123")
}

func TestDetachPaymentMethod_ReturnsCannotRemoveLast_WhenActiveSubAndOnlyOneMethod(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	subRepo.On("FindByFamily", mock.Anything, testScope()).Return(activeSub(), nil)
	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("ListPaymentMethods", mock.Anything, "cus_test_123").Return([]HyperswitchPaymentMethod{
		{ID: "pm_only", MethodType: "card"},
	}, nil)

	err := svc.DetachPaymentMethod(context.Background(), "pm_only", testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrCannotRemoveLastPaymentMethod))
}

// ═══════════════════════════════════════════════════════════════════════════════
// 31-37: ProcessCoppaVerification
// ═══════════════════════════════════════════════════════════════════════════════

func TestProcessCoppaVerification_CreatesCustomerIfNotExists(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	txnRepo := new(mockTransactionRepo)
	adapter := new(mockAdapter)
	iamSvc := new(mockIamService)
	svc := newTestService(new(mockSubscriptionRepo), txnRepo, custRepo, new(mockPayoutRepo), adapter, iamSvc)

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(nil, nil)
	iamSvc.On("GetFamilyPrimaryEmail", mock.Anything, testFamilyID()).Return("parent@example.com", "Family", nil)
	adapter.On("CreateCustomer", mock.Anything, "parent@example.com", "Family", mock.Anything).Return("cus_new", nil)
	custRepo.On("Upsert", mock.Anything, testFamilyID(), mock.Anything).Return(testCustomer(), nil)

	adapter.On("ProcessMicroCharge", mock.Anything, "cus_new", "pm_coppa", int64(50), mock.Anything, mock.Anything).
		Return("pay_coppa", "ref_coppa", nil)

	txnRepo.On("Create", mock.Anything, mock.MatchedBy(func(input CreateTransactionRow) bool {
		return input.TransactionType == TransactionTypeCoppaCharge
	})).Return(&BillTransaction{}, nil)
	txnRepo.On("Create", mock.Anything, mock.MatchedBy(func(input CreateTransactionRow) bool {
		return input.TransactionType == TransactionTypeCoppaRefund
	})).Return(&BillTransaction{}, nil)

	resp, err := svc.ProcessCoppaVerification(context.Background(), CoppaVerificationCommand{PaymentMethodID: "pm_coppa"}, testScope())
	require.NoError(t, err)
	assert.True(t, resp.Verified)
	assert.Equal(t, "pay_coppa", resp.ChargeID)
	assert.Equal(t, "ref_coppa", resp.RefundID)
	adapter.AssertCalled(t, "CreateCustomer", mock.Anything, "parent@example.com", "Family", mock.Anything)
}

func TestProcessCoppaVerification_ChargesAndRefunds(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	txnRepo := new(mockTransactionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(new(mockSubscriptionRepo), txnRepo, custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("ProcessMicroCharge", mock.Anything, "cus_test_123", "pm_test", int64(50), mock.Anything, mock.Anything).
		Return("pay_123", "ref_456", nil)
	txnRepo.On("Create", mock.Anything, mock.Anything).Return(&BillTransaction{}, nil)

	resp, err := svc.ProcessCoppaVerification(context.Background(), CoppaVerificationCommand{PaymentMethodID: "pm_test"}, testScope())
	require.NoError(t, err)
	assert.True(t, resp.Verified)
	assert.Equal(t, "pay_123", resp.ChargeID)
	assert.Equal(t, "ref_456", resp.RefundID)
}

func TestProcessCoppaVerification_CreatesTwoTransactionRows(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	txnRepo := new(mockTransactionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(new(mockSubscriptionRepo), txnRepo, custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("ProcessMicroCharge", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("pay_1", "ref_1", nil)
	txnRepo.On("Create", mock.Anything, mock.Anything).Return(&BillTransaction{}, nil)

	_, err := svc.ProcessCoppaVerification(context.Background(), CoppaVerificationCommand{PaymentMethodID: "pm_x"}, testScope())
	require.NoError(t, err)

	// Verify two Create calls (charge + refund)
	assert.Len(t, txnRepo.Calls, 2)
	chargeInput := txnRepo.Calls[0].Arguments.Get(1).(CreateTransactionRow)
	refundInput := txnRepo.Calls[1].Arguments.Get(1).(CreateTransactionRow)
	assert.Equal(t, TransactionTypeCoppaCharge, chargeInput.TransactionType)
	assert.Equal(t, TransactionTypeCoppaRefund, refundInput.TransactionType)
}

func TestProcessCoppaVerification_ReturnsPaymentDeclined_WhenCardDeclined(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("ProcessMicroCharge", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", "", ErrPaymentDeclined)

	_, err := svc.ProcessCoppaVerification(context.Background(), CoppaVerificationCommand{PaymentMethodID: "pm_bad"}, testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrPaymentDeclined))
}

func TestProcessCoppaVerification_ReturnsAdapterUnavailable(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	adapter.On("ProcessMicroCharge", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", "", ErrPaymentAdapterUnavailable)

	_, err := svc.ProcessCoppaVerification(context.Background(), CoppaVerificationCommand{PaymentMethodID: "pm_x"}, testScope())
	require.Error(t, err)
	var be *BillingError
	require.True(t, errors.As(err, &be))
	assert.True(t, errors.Is(be.Err, ErrPaymentAdapterUnavailable))
}

func TestProcessCoppaVerification_SucceedsEvenIfRefundFails(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	txnRepo := new(mockTransactionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(new(mockSubscriptionRepo), txnRepo, custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	// Charge succeeded but refund failed (empty refund ID)
	adapter.On("ProcessMicroCharge", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("pay_ok", "", nil)
	txnRepo.On("Create", mock.Anything, mock.Anything).Return(&BillTransaction{}, nil)

	resp, err := svc.ProcessCoppaVerification(context.Background(), CoppaVerificationCommand{PaymentMethodID: "pm_x"}, testScope())
	require.NoError(t, err)
	assert.True(t, resp.Verified)
	assert.Equal(t, "pay_ok", resp.ChargeID)
	assert.Empty(t, resp.RefundID)

	// Only charge transaction created (no refund row since refundID is empty)
	assert.Len(t, txnRepo.Calls, 1)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 38-46: ProcessHyperswitchWebhook
// ═══════════════════════════════════════════════════════════════════════════════

func TestProcessWebhook_RejectsInvalidSignature(t *testing.T) {
	adapter := new(mockAdapter)
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), adapter, new(mockIamService))

	adapter.On("VerifyWebhook", mock.Anything, mock.Anything, mock.Anything).Return(false, ErrInvalidWebhookSignature)

	err := svc.ProcessHyperswitchWebhook(context.Background(), []byte("{}"), "bad-sig")
	assert.NoError(t, err) // Returns nil — webhook always 200
}

func TestProcessWebhook_HandlesSubscriptionCreated(t *testing.T) {
	adapter := new(mockAdapter)
	subRepo := new(mockSubscriptionRepo)
	custRepo := new(mockCustomerRepo)
	svc := newTestService(subRepo, new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, new(mockIamService))

	payload := []byte(`{}`)
	adapter.On("VerifyWebhook", mock.Anything, payload, "valid-sig").Return(true, nil)
	adapter.On("ParseWebhookEvent", mock.Anything, payload).Return(&BillingWebhookEvent{
		Type: "subscription.created",
		SubscriptionCreated: &BillingWebhookSubscriptionCreated{
			Subscription: HyperswitchSubscription{
				ID: "sub_new", CustomerID: "cus_123", Status: SubscriptionStatusIncomplete,
				CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
				AmountCents: 1499, Currency: "usd", PriceID: "price_monthly_v1",
			},
		},
	}, nil)

	subRepo.On("FindByHyperswitchID", mock.Anything, "sub_new").Return(nil, nil)
	custRepo.On("FindByHyperswitchID", mock.Anything, "cus_123").Return(&BillHyperswitchCustomer{
		FamilyID: testFamilyID(), HyperswitchCustomerID: "cus_123",
	}, nil)
	subRepo.On("Create", mock.Anything, mock.Anything).Return(&BillSubscription{}, nil)

	err := svc.ProcessHyperswitchWebhook(context.Background(), payload, "valid-sig")
	assert.NoError(t, err)
	subRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestProcessWebhook_SubscriptionUpdated_PublishesCreatedOnFirstActivation(t *testing.T) {
	adapter := new(mockAdapter)
	subRepo := new(mockSubscriptionRepo)
	eventBus := shared.NewEventBus()
	svc := &BillingServiceImpl{
		subscriptionRepo: subRepo, transactionRepo: new(mockTransactionRepo),
		customerRepo: new(mockCustomerRepo), payoutRepo: new(mockPayoutRepo),
		adapter: adapter, iamService: new(mockIamService), events: eventBus,
		config: BillingConfig{MonthlyPriceID: "price_monthly_v1", AnnualPriceID: "price_annual_v1"},
	}

	payload := []byte(`{}`)
	adapter.On("VerifyWebhook", mock.Anything, payload, "sig").Return(true, nil)
	adapter.On("ParseWebhookEvent", mock.Anything, payload).Return(&BillingWebhookEvent{
		Type: "subscription.updated",
		SubscriptionUpdated: &BillingWebhookSubscriptionUpdated{
			Subscription: HyperswitchSubscription{
				ID: "sub_123", Status: SubscriptionStatusActive,
				CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
				AmountCents: 1499, PriceID: "price_monthly_v1",
			},
		},
	}, nil)

	// Existing sub with incomplete status (first activation)
	sub := &BillSubscription{
		ID: uuid.Must(uuid.NewV7()), FamilyID: testFamilyID(), Status: SubscriptionStatusIncomplete,
		BillingInterval: IntervalMonthly, HyperswitchSubscriptionID: "sub_123",
	}
	subRepo.On("FindByHyperswitchID", mock.Anything, "sub_123").Return(sub, nil)
	subRepo.On("Update", mock.Anything, sub.ID, mock.Anything).Return(sub, nil)

	var publishedEvent shared.DomainEvent
	eventBus.Subscribe(subscriptionCreatedType(), &captureHandler{captured: &publishedEvent})

	err := svc.ProcessHyperswitchWebhook(context.Background(), payload, "sig")
	assert.NoError(t, err)
	require.NotNil(t, publishedEvent)
	assert.Equal(t, "billing.SubscriptionCreated", publishedEvent.EventName())
}

func TestProcessWebhook_SubscriptionUpdated_PublishesChangedOnSubsequentUpdates(t *testing.T) {
	adapter := new(mockAdapter)
	subRepo := new(mockSubscriptionRepo)
	eventBus := shared.NewEventBus()
	svc := &BillingServiceImpl{
		subscriptionRepo: subRepo, transactionRepo: new(mockTransactionRepo),
		customerRepo: new(mockCustomerRepo), payoutRepo: new(mockPayoutRepo),
		adapter: adapter, iamService: new(mockIamService), events: eventBus,
		config: BillingConfig{},
	}

	payload := []byte(`{}`)
	adapter.On("VerifyWebhook", mock.Anything, payload, "sig").Return(true, nil)
	adapter.On("ParseWebhookEvent", mock.Anything, payload).Return(&BillingWebhookEvent{
		Type: "subscription.updated",
		SubscriptionUpdated: &BillingWebhookSubscriptionUpdated{
			Subscription: HyperswitchSubscription{
				ID: "sub_123", Status: SubscriptionStatusActive,
				CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
				AmountCents: 1499, PriceID: "price_monthly_v1",
			},
		},
	}, nil)

	// Already active sub (subsequent update)
	sub := &BillSubscription{
		ID: uuid.Must(uuid.NewV7()), FamilyID: testFamilyID(), Status: SubscriptionStatusActive,
		BillingInterval: IntervalMonthly, HyperswitchSubscriptionID: "sub_123",
	}
	subRepo.On("FindByHyperswitchID", mock.Anything, "sub_123").Return(sub, nil)
	subRepo.On("Update", mock.Anything, sub.ID, mock.Anything).Return(sub, nil)

	var publishedEvent shared.DomainEvent
	eventBus.Subscribe(subscriptionChangedType(), &captureHandler{captured: &publishedEvent})

	err := svc.ProcessHyperswitchWebhook(context.Background(), payload, "sig")
	assert.NoError(t, err)
	require.NotNil(t, publishedEvent)
	assert.Equal(t, "billing.SubscriptionChanged", publishedEvent.EventName())
}

func TestProcessWebhook_SubscriptionDeleted_PublishesCancelled(t *testing.T) {
	adapter := new(mockAdapter)
	subRepo := new(mockSubscriptionRepo)
	eventBus := shared.NewEventBus()
	svc := &BillingServiceImpl{
		subscriptionRepo: subRepo, transactionRepo: new(mockTransactionRepo),
		customerRepo: new(mockCustomerRepo), payoutRepo: new(mockPayoutRepo),
		adapter: adapter, iamService: new(mockIamService), events: eventBus,
		config: BillingConfig{},
	}

	payload := []byte(`{}`)
	adapter.On("VerifyWebhook", mock.Anything, payload, "sig").Return(true, nil)
	adapter.On("ParseWebhookEvent", mock.Anything, payload).Return(&BillingWebhookEvent{
		Type:                "subscription.deleted",
		SubscriptionDeleted: &BillingWebhookSubscriptionDeleted{SubscriptionID: "sub_123"},
	}, nil)

	sub := &BillSubscription{
		ID: uuid.Must(uuid.NewV7()), FamilyID: testFamilyID(), Status: SubscriptionStatusActive,
		HyperswitchSubscriptionID: "sub_123",
	}
	subRepo.On("FindByHyperswitchID", mock.Anything, "sub_123").Return(sub, nil)
	subRepo.On("Update", mock.Anything, sub.ID, mock.Anything).Return(sub, nil)

	var publishedEvent shared.DomainEvent
	eventBus.Subscribe(subscriptionCancelledType(), &captureHandler{captured: &publishedEvent})

	err := svc.ProcessHyperswitchWebhook(context.Background(), payload, "sig")
	assert.NoError(t, err)
	require.NotNil(t, publishedEvent)
	assert.Equal(t, "billing.SubscriptionCancelled", publishedEvent.EventName())
}

func TestProcessWebhook_InvoicePaid_CreatesTransactionRow(t *testing.T) {
	adapter := new(mockAdapter)
	subRepo := new(mockSubscriptionRepo)
	txnRepo := new(mockTransactionRepo)
	svc := newTestService(subRepo, txnRepo, new(mockCustomerRepo), new(mockPayoutRepo), adapter, new(mockIamService))

	payload := []byte(`{}`)
	adapter.On("VerifyWebhook", mock.Anything, payload, "sig").Return(true, nil)
	adapter.On("ParseWebhookEvent", mock.Anything, payload).Return(&BillingWebhookEvent{
		Type: "invoice.paid",
		InvoicePaid: &BillingWebhookInvoicePaid{
			InvoiceID: "inv_123", SubscriptionID: "sub_123",
			AmountCents: 1499, PaymentID: "pay_123",
		},
	}, nil)

	txnRepo.On("ExistsByPaymentID", mock.Anything, "pay_123", TransactionTypeSubscriptionPayment).Return(false, nil)
	subRepo.On("FindByHyperswitchID", mock.Anything, "sub_123").Return(&BillSubscription{
		FamilyID: testFamilyID(),
	}, nil)
	txnRepo.On("Create", mock.Anything, mock.MatchedBy(func(input CreateTransactionRow) bool {
		return input.TransactionType == TransactionTypeSubscriptionPayment && input.AmountCents == 1499
	})).Return(&BillTransaction{}, nil)

	err := svc.ProcessHyperswitchWebhook(context.Background(), payload, "sig")
	assert.NoError(t, err)
	txnRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestProcessWebhook_PaymentFailed_SetsStatusPastDue(t *testing.T) {
	adapter := new(mockAdapter)
	subRepo := new(mockSubscriptionRepo)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), adapter, new(mockIamService))

	payload := []byte(`{}`)
	subID := "sub_123"
	adapter.On("VerifyWebhook", mock.Anything, payload, "sig").Return(true, nil)
	adapter.On("ParseWebhookEvent", mock.Anything, payload).Return(&BillingWebhookEvent{
		Type: "payment.failed",
		PaymentFailed: &BillingWebhookPaymentFailed{
			PaymentID: "pay_fail", SubscriptionID: &subID, Reason: "insufficient_funds",
		},
	}, nil)

	sub := &BillSubscription{ID: uuid.Must(uuid.NewV7()), FamilyID: testFamilyID(), Status: SubscriptionStatusActive}
	subRepo.On("FindByHyperswitchID", mock.Anything, "sub_123").Return(sub, nil)
	subRepo.On("Update", mock.Anything, sub.ID, mock.MatchedBy(func(u SubscriptionUpdate) bool {
		return u.Status != nil && *u.Status == SubscriptionStatusPastDue
	})).Return(sub, nil)

	err := svc.ProcessHyperswitchWebhook(context.Background(), payload, "sig")
	assert.NoError(t, err)
	subRepo.AssertCalled(t, "Update", mock.Anything, sub.ID, mock.Anything)
}

func TestProcessWebhook_Idempotent_DuplicateEventsAreNoOps(t *testing.T) {
	adapter := new(mockAdapter)
	txnRepo := new(mockTransactionRepo)
	svc := newTestService(new(mockSubscriptionRepo), txnRepo, new(mockCustomerRepo), new(mockPayoutRepo), adapter, new(mockIamService))

	payload := []byte(`{}`)
	adapter.On("VerifyWebhook", mock.Anything, payload, "sig").Return(true, nil)
	adapter.On("ParseWebhookEvent", mock.Anything, payload).Return(&BillingWebhookEvent{
		Type: "invoice.paid",
		InvoicePaid: &BillingWebhookInvoicePaid{
			PaymentID: "pay_dup", SubscriptionID: "sub_123", AmountCents: 1499,
		},
	}, nil)

	// Already exists — duplicate event
	txnRepo.On("ExistsByPaymentID", mock.Anything, "pay_dup", TransactionTypeSubscriptionPayment).Return(true, nil)

	err := svc.ProcessHyperswitchWebhook(context.Background(), payload, "sig")
	assert.NoError(t, err)
	txnRepo.AssertNotCalled(t, "Create")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 47-50: Event Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleFamilyDeletionScheduled_CancelsAndDeletes(t *testing.T) {
	subRepo := new(mockSubscriptionRepo)
	adapter := new(mockAdapter)
	svc := newTestService(subRepo, new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), adapter, new(mockIamService))

	sub := activeSub()
	subRepo.On("FindByFamily", mock.Anything, mock.Anything).Return(sub, nil)
	adapter.On("CancelSubscription", mock.Anything, sub.HyperswitchSubscriptionID).Return(&HyperswitchSubscription{}, nil)
	subRepo.On("DeleteByFamily", mock.Anything, testFamilyID()).Return(nil)

	err := svc.HandleFamilyDeletionScheduled(context.Background(), FamilyDeletionScheduledEvent{FamilyID: testFamilyID()})
	require.NoError(t, err)
	adapter.AssertCalled(t, "CancelSubscription", mock.Anything, sub.HyperswitchSubscriptionID)
	subRepo.AssertCalled(t, "DeleteByFamily", mock.Anything, testFamilyID())
}

func TestHandlePrimaryParentTransferred_UpdatesCustomerEmail(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	adapter := new(mockAdapter)
	iamSvc := new(mockIamService)
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), custRepo, new(mockPayoutRepo), adapter, iamSvc)

	custRepo.On("FindByFamily", mock.Anything, testFamilyID()).Return(testCustomer(), nil)
	iamSvc.On("GetFamilyPrimaryEmail", mock.Anything, testFamilyID()).Return("new@example.com", "Test Family", nil)
	adapter.On("UpdateCustomer", mock.Anything, "cus_test_123", "new@example.com", "").Return(nil)

	err := svc.HandlePrimaryParentTransferred(context.Background(), PrimaryParentTransferredEvent{
		FamilyID: testFamilyID(), NewPrimaryID: uuid.Must(uuid.NewV7()),
	})
	require.NoError(t, err)
	adapter.AssertCalled(t, "UpdateCustomer", mock.Anything, "cus_test_123", "new@example.com", "")
}

func TestHandlePurchaseCompleted_NoOpForNow(t *testing.T) {
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	err := svc.HandlePurchaseCompleted(context.Background(), PurchaseCompletedEvent{PurchaseID: uuid.Must(uuid.NewV7())})
	assert.NoError(t, err)
}

func TestHandlePurchaseRefunded_NoOpForNow(t *testing.T) {
	svc := newTestService(new(mockSubscriptionRepo), new(mockTransactionRepo), new(mockCustomerRepo), new(mockPayoutRepo), new(mockAdapter), new(mockIamService))

	err := svc.HandlePurchaseRefunded(context.Background(), PurchaseRefundedEvent{PurchaseID: uuid.Must(uuid.NewV7()), RefundAmountCents: 1000})
	assert.NoError(t, err)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 51: Error Mapping
// ═══════════════════════════════════════════════════════════════════════════════

func TestBillingError_ToAppError_MapsCorrectStatusCodes(t *testing.T) {
	tests := []struct {
		sentinel   error
		expectCode int
	}{
		{ErrSubscriptionNotFound, 404},
		{ErrSubscriptionAlreadyExists, 409},
		{ErrCannotReactivate, 409},
		{ErrSubscriptionNotActive, 409},
		{ErrInvalidBillingInterval, 422},
		{ErrPaymentMethodNotFound, 404},
		{ErrCannotRemoveLastPaymentMethod, 409},
		{ErrPaymentDeclined, 422},
		{ErrCoppaVerificationFailed, 422},
		{ErrPaymentAdapterUnavailable, 502},
	}

	for _, tt := range tests {
		be := &BillingError{Err: tt.sentinel}
		appErr := be.toAppError()
		assert.Equal(t, tt.expectCode, appErr.StatusCode, "sentinel: %v", tt.sentinel)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// 52-55: Domain Events
// ═══════════════════════════════════════════════════════════════════════════════

func TestDomainEvents_ImplementDomainEvent(t *testing.T) {
	assert.Equal(t, "billing.SubscriptionCreated", SubscriptionCreated{}.EventName())
	assert.Equal(t, "billing.SubscriptionChanged", SubscriptionChanged{}.EventName())
	assert.Equal(t, "billing.SubscriptionCancelled", SubscriptionCancelled{}.EventName())
	assert.Equal(t, "billing.PayoutCompleted", PayoutCompleted{}.EventName())
}

// ═══════════════════════════════════════════════════════════════════════════════
// 56-57: Config (resolvePriceID)
// ═══════════════════════════════════════════════════════════════════════════════

func TestResolvePriceID_Monthly(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil, nil)
	priceID, err := svc.resolvePriceID(IntervalMonthly)
	require.NoError(t, err)
	assert.Equal(t, "price_monthly_v1", priceID)
}

func TestResolvePriceID_Annual(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil, nil)
	priceID, err := svc.resolvePriceID(IntervalAnnual)
	require.NoError(t, err)
	assert.Equal(t, "price_annual_v1", priceID)
}

func TestResolvePriceID_InvalidInterval(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil, nil)
	_, err := svc.resolvePriceID("weekly")
	assert.True(t, errors.Is(err, ErrInvalidBillingInterval))
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test helpers for event capture
// ═══════════════════════════════════════════════════════════════════════════════

type captureHandler struct {
	captured *shared.DomainEvent
}

func (h *captureHandler) Handle(_ context.Context, event shared.DomainEvent) error {
	*h.captured = event
	return nil
}

func subscriptionCreatedType() reflect.Type  { return reflect.TypeOf(SubscriptionCreated{}) }
func subscriptionChangedType() reflect.Type  { return reflect.TypeOf(SubscriptionChanged{}) }
func subscriptionCancelledType() reflect.Type { return reflect.TypeOf(SubscriptionCancelled{}) }
