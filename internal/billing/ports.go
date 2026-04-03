package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface [10-billing §5]
// ═══════════════════════════════════════════════════════════════════════════════

// BillingService defines all use cases for the billing domain.
type BillingService interface {
	// ─── Queries (read, no side effects) ────────────────────────────────

	// GetSubscription returns the current subscription for a family. Returns free-tier default if none.
	GetSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error)

	// ListTransactions returns transaction history for a family.
	ListTransactions(ctx context.Context, params TransactionListParams, scope shared.FamilyScope) (*TransactionListResponse, error)

	// ListInvoices returns Hyperswitch invoices for a family's subscription. (Phase 2)
	ListInvoices(ctx context.Context, params InvoiceListParams, scope shared.FamilyScope) (*InvoiceListResponse, error)

	// ListPaymentMethods returns attached payment methods for a family. (Phase 2)
	ListPaymentMethods(ctx context.Context, scope shared.FamilyScope) ([]PaymentMethodResponse, error)

	// EstimateSubscription previews pricing for a subscription or plan change. (Phase 2)
	EstimateSubscription(ctx context.Context, query EstimateSubscriptionQuery, scope shared.FamilyScope) (*EstimateResponse, error)

	// ListPayouts returns creator payout history. (Phase 2)
	ListPayouts(ctx context.Context, params PayoutListParams, creatorID uuid.UUID) (*PayoutListResponse, error)

	// ─── Commands (write, has side effects) ─────────────────────────────

	// CreateSubscription creates a new premium subscription via Hyperswitch. (Phase 2)
	CreateSubscription(ctx context.Context, cmd CreateSubscriptionCommand, scope shared.FamilyScope) (*SubscriptionResponse, error)

	// UpdateSubscription updates subscription (billing interval change) with proration. (Phase 2)
	UpdateSubscription(ctx context.Context, cmd UpdateSubscriptionCommand, scope shared.FamilyScope) (*SubscriptionResponse, error)

	// CancelSubscription cancels subscription at end of current billing period. (Phase 2)
	CancelSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error)

	// ReactivateSubscription reverses a pending cancellation. (Phase 2)
	ReactivateSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error)

	// PauseSubscription pauses an active subscription. (Phase 2)
	PauseSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error)

	// ResumeSubscription resumes a paused subscription. (Phase 2)
	ResumeSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error)

	// AttachPaymentMethod attaches a payment method via Hyperswitch SetupIntent. (Phase 2)
	AttachPaymentMethod(ctx context.Context, cmd AttachPaymentMethodCommand, scope shared.FamilyScope) (*PaymentMethodResponse, error)

	// DetachPaymentMethod detaches a payment method. (Phase 2)
	DetachPaymentMethod(ctx context.Context, paymentMethodID string, scope shared.FamilyScope) error

	// ProcessCoppaVerification processes a COPPA micro-charge verification ($0.50 charge + immediate refund).
	// Called by iam:: during COPPA consent flow. [S§1.4]
	ProcessCoppaVerification(ctx context.Context, cmd CoppaVerificationCommand, scope shared.FamilyScope) (*CoppaVerificationResult, error)

	// ─── Event handlers ─────────────────────────────────────────────────

	// HandleFamilyDeletionScheduled cancels subscription in Hyperswitch.
	HandleFamilyDeletionScheduled(ctx context.Context, event FamilyDeletionScheduledEvent) error

	// HandlePrimaryParentTransferred updates Hyperswitch customer email.
	HandlePrimaryParentTransferred(ctx context.Context, event PrimaryParentTransferredEvent) error

	// HandlePurchaseCompleted records creator earnings. (Phase 2)
	HandlePurchaseCompleted(ctx context.Context, event PurchaseCompletedEvent) error

	// HandlePurchaseRefunded deducts from creator earnings. (Phase 2)
	HandlePurchaseRefunded(ctx context.Context, event PurchaseRefundedEvent) error

	// ─── Webhook processing ─────────────────────────────────────────────

	// ProcessHyperswitchWebhook processes a verified Hyperswitch webhook payload.
	ProcessHyperswitchWebhook(ctx context.Context, payload []byte, signature string) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interfaces [10-billing §6]
// ═══════════════════════════════════════════════════════════════════════════════

// SubscriptionRepository is family-scoped. One subscription per family. [S§15.3]
type SubscriptionRepository interface {
	Create(ctx context.Context, input CreateSubscriptionRow) (*BillSubscription, error)
	FindByFamily(ctx context.Context, scope shared.FamilyScope) (*BillSubscription, error)
	FindByHyperswitchID(ctx context.Context, hyperswitchSubscriptionID string) (*BillSubscription, error)
	Update(ctx context.Context, subscriptionID uuid.UUID, updates SubscriptionUpdate) (*BillSubscription, error)
	DeleteByFamily(ctx context.Context, familyID uuid.UUID) error
}

// TransactionRepository is family-scoped. Immutable records — insert only, no updates.
type TransactionRepository interface {
	Create(ctx context.Context, input CreateTransactionRow) (*BillTransaction, error)
	ListByFamily(ctx context.Context, scope shared.FamilyScope, params *TransactionListParams) ([]BillTransaction, error)
	ExistsByPaymentID(ctx context.Context, hyperswitchPaymentID string, transactionType string) (bool, error)
}

// CustomerRepository is family-scoped (family_id is PK).
type CustomerRepository interface {
	Upsert(ctx context.Context, familyID uuid.UUID, input UpsertCustomerRow) (*BillHyperswitchCustomer, error)
	FindByFamily(ctx context.Context, familyID uuid.UUID) (*BillHyperswitchCustomer, error)
	FindByHyperswitchID(ctx context.Context, hyperswitchCustomerID string) (*BillHyperswitchCustomer, error)
}

// PayoutRepository is creator-scoped. (Phase 2)
type PayoutRepository interface {
	Create(ctx context.Context, input CreatePayoutRow) (*BillPayout, error)
	ListByCreator(ctx context.Context, creatorID uuid.UUID, params *PayoutListParams) ([]BillPayout, error)
	UpdateStatus(ctx context.Context, payoutID uuid.UUID, status string, hyperswitchPayoutID *string) (*BillPayout, error)
	FindPending(ctx context.Context, limit uint32) ([]BillPayout, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Adapter Interface [10-billing §7]
// ═══════════════════════════════════════════════════════════════════════════════

// SubscriptionPaymentAdapter is a processor-agnostic subscription + payment adapter backed by Hyperswitch.
// Uses the billing-specific Hyperswitch business profile, separate from the marketplace profile. [07-mkt §18.5]
type SubscriptionPaymentAdapter interface {
	// ─── Customer Management ────────────────────────────────────────────
	CreateCustomer(ctx context.Context, email string, name string, metadata map[string]string) (string, error)
	UpdateCustomer(ctx context.Context, customerID string, email string, name string) error

	// ─── Subscriptions ──────────────────────────────────────────────────
	CreateSubscription(ctx context.Context, customerID string, priceID string, paymentMethodID string, metadata map[string]string) (*HyperswitchSubscription, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, newPriceID string) (*HyperswitchSubscription, error)
	CancelSubscription(ctx context.Context, subscriptionID string) (*HyperswitchSubscription, error)
	PauseSubscription(ctx context.Context, subscriptionID string) (*HyperswitchSubscription, error)
	ResumeSubscription(ctx context.Context, subscriptionID string) (*HyperswitchSubscription, error)
	ReactivateSubscription(ctx context.Context, subscriptionID string) (*HyperswitchSubscription, error)
	EstimateSubscription(ctx context.Context, customerID string, priceID string, currentSubscriptionID *string) (*HyperswitchEstimate, error)

	// ─── Payment Methods ────────────────────────────────────────────────
	CreateSetupIntent(ctx context.Context, customerID string) (*SetupIntentResponse, error)
	ListPaymentMethods(ctx context.Context, customerID string) ([]HyperswitchPaymentMethod, error)
	DetachPaymentMethod(ctx context.Context, paymentMethodID string) error

	// ─── One-Time Payments (COPPA) ──────────────────────────────────────
	ProcessMicroCharge(ctx context.Context, customerID string, paymentMethodID string, amountCents int64, description string, metadata map[string]string) (string, string, error)

	// ─── Invoices ───────────────────────────────────────────────────────
	ListInvoices(ctx context.Context, customerID string, limit uint32) ([]HyperswitchInvoice, error)

	// ─── Payouts (Phase 2) ──────────────────────────────────────────────
	CreatePayout(ctx context.Context, paymentAccountID string, amountCents int64, currency string, metadata map[string]string) (*HyperswitchPayout, error)

	// ─── Webhooks ───────────────────────────────────────────────────────
	VerifyWebhook(ctx context.Context, payload []byte, signature string) (bool, error)
	ParseWebhookEvent(ctx context.Context, payload []byte) (*BillingWebhookEvent, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Consumer-Defined Cross-Domain Interfaces [10-billing §16.2]
// ═══════════════════════════════════════════════════════════════════════════════

// IamServiceForBilling provides IAM data needed by billing.
// Consumer-defined interface — avoids circular import with iam package.
type IamServiceForBilling interface {
	// GetFamilyPrimaryEmail returns the primary parent's email and family display name.
	GetFamilyPrimaryEmail(ctx context.Context, familyID uuid.UUID) (email string, displayName string, err error)
}

// MktServiceForBilling provides marketplace sales data for payout aggregation.
// Consumer-defined interface — avoids circular import with mkt package. [ARCH §4.4]
type MktServiceForBilling interface {
	// GetAllCreatorSales returns aggregated sales per creator for a billing period.
	GetAllCreatorSales(ctx context.Context, from, to time.Time) ([]CreatorEarningSummary, error)
}

// CreatorEarningSummary is the billing-side view of aggregated creator earnings.
type CreatorEarningSummary struct {
	CreatorID            uuid.UUID
	TotalPayoutCents     int64
	PurchaseCount        int32
	RefundDeductionCents int64
}

// MktAdapter implements MktServiceForBilling via function closures wired in main.go.
type MktAdapter struct {
	getAllCreatorSalesFn func(ctx context.Context, from, to time.Time) ([]CreatorEarningSummary, error)
}

// NewMktAdapter creates a new MktAdapter from closure functions.
func NewMktAdapter(
	getAllCreatorSales func(ctx context.Context, from, to time.Time) ([]CreatorEarningSummary, error),
) *MktAdapter {
	return &MktAdapter{getAllCreatorSalesFn: getAllCreatorSales}
}

func (a *MktAdapter) GetAllCreatorSales(ctx context.Context, from, to time.Time) ([]CreatorEarningSummary, error) {
	return a.getAllCreatorSalesFn(ctx, from, to)
}

// ─── Mirror Event Types ─────────────────────────────────────────────────────
// These mirror types are used by event handlers when the source domain events
// haven't been defined yet. Following the same pattern as notify:: handlers.

// FamilyDeletionScheduledEvent mirrors iam::FamilyDeletionScheduled.
type FamilyDeletionScheduledEvent struct {
	FamilyID uuid.UUID
}

// PrimaryParentTransferredEvent mirrors iam::PrimaryParentTransferred.
type PrimaryParentTransferredEvent struct {
	FamilyID     uuid.UUID
	NewPrimaryID uuid.UUID
	NewEmail     string
}

// PurchaseCompletedEvent mirrors mkt::PurchaseCompleted.
type PurchaseCompletedEvent struct {
	FamilyID   uuid.UUID
	PurchaseID uuid.UUID
	ListingID  uuid.UUID
}

// PurchaseRefundedEvent mirrors mkt::PurchaseRefunded.
type PurchaseRefundedEvent struct {
	PurchaseID        uuid.UUID
	ListingID         uuid.UUID
	FamilyID          uuid.UUID
	RefundAmountCents int64
}

// ─── Consumer-Defined Adapters ──────────────────────────────────────────────

// IamAdapter implements IamServiceForBilling via function closures wired in main.go.
type IamAdapter struct {
	getFamilyPrimaryEmailFn func(ctx context.Context, familyID uuid.UUID) (string, string, error)
}

// NewIamAdapter creates a new IamAdapter from closure functions.
func NewIamAdapter(
	getFamilyPrimaryEmail func(ctx context.Context, familyID uuid.UUID) (string, string, error),
) *IamAdapter {
	return &IamAdapter{
		getFamilyPrimaryEmailFn: getFamilyPrimaryEmail,
	}
}

func (a *IamAdapter) GetFamilyPrimaryEmail(ctx context.Context, familyID uuid.UUID) (string, string, error) {
	return a.getFamilyPrimaryEmailFn(ctx, familyID)
}
