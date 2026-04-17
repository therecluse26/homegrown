package billing

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── GORM Models ────────────────────────────────────────────────────────────

// BillHyperswitchCustomer maps a family to its Hyperswitch customer record. [10-billing §3.2]
type BillHyperswitchCustomer struct {
	FamilyID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	HyperswitchCustomerID  string    `gorm:"not null;uniqueIndex"`
	DefaultPaymentMethodID *string
	CreatedAt              time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt              time.Time `gorm:"not null;autoUpdateTime"`
}

func (BillHyperswitchCustomer) TableName() string { return "bill_hyperswitch_customers" }

// BillSubscription mirrors Hyperswitch subscription state. [10-billing §3.2]
type BillSubscription struct {
	ID                        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID                  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	HyperswitchSubscriptionID string    `gorm:"not null;uniqueIndex"`
	HyperswitchCustomerID     string    `gorm:"not null"`
	Tier                      string    `gorm:"not null;default:premium"`
	Status                    string    `gorm:"not null;default:incomplete"`
	BillingInterval           string    `gorm:"not null"`
	CurrentPeriodStart        time.Time `gorm:"not null"`
	CurrentPeriodEnd          time.Time `gorm:"not null"`
	CancelAtPeriodEnd         bool      `gorm:"not null;default:false"`
	CanceledAt                *time.Time
	AmountCents               int64     `gorm:"not null"`
	Currency                  string    `gorm:"not null;default:usd"`
	HyperswitchPriceID        string    `gorm:"not null"`
	CreatedAt                 time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt                 time.Time `gorm:"not null;autoUpdateTime"`
}

func (BillSubscription) TableName() string { return "bill_subscriptions" }

func (m *BillSubscription) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// BillTransaction records a financial transaction. [10-billing §3.2]
type BillTransaction struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID             uuid.UUID `gorm:"type:uuid;not null"`
	TransactionType      string    `gorm:"not null"`
	Status               string    `gorm:"not null;default:pending"`
	AmountCents          int64     `gorm:"not null"`
	Currency             string    `gorm:"not null;default:usd"`
	HyperswitchPaymentID *string
	HyperswitchInvoiceID *string
	Description          *string
	Metadata             map[string]any `gorm:"type:jsonb;serializer:json;not null;default:'{}'"`
	CreatedAt            time.Time      `gorm:"not null;autoCreateTime"`
}

func (BillTransaction) TableName() string { return "bill_transactions" }

func (m *BillTransaction) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// BillPayout is a creator payout aggregation record. (Phase 2) [10-billing §3.2]
type BillPayout struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatorID            uuid.UUID `gorm:"type:uuid;not null"`
	Status               string    `gorm:"not null;default:pending"`
	AmountCents          int64     `gorm:"not null"`
	Currency             string    `gorm:"not null;default:usd"`
	PeriodStart          time.Time `gorm:"not null"`
	PeriodEnd            time.Time `gorm:"not null"`
	PurchaseCount        int32     `gorm:"not null;default:0"`
	RefundDeductionCents int64     `gorm:"not null;default:0"`
	HyperswitchPayoutID  *string
	ProcessedAt          *time.Time
	CreatedAt            time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt            time.Time `gorm:"not null;autoUpdateTime"`
}

func (BillPayout) TableName() string { return "bill_payouts" }

func (m *BillPayout) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ─── Subscription Status Constants ──────────────────────────────────────────

const (
	SubscriptionStatusActive     = "active"
	SubscriptionStatusTrialing   = "trialing"
	SubscriptionStatusPastDue    = "past_due"
	SubscriptionStatusCanceled   = "canceled"
	SubscriptionStatusPaused     = "paused"
	SubscriptionStatusIncomplete = "incomplete"
)

// ─── Transaction Type Constants ─────────────────────────────────────────────

const (
	TransactionTypeSubscriptionPayment = "subscription_payment"
	TransactionTypeSubscriptionRefund  = "subscription_refund"
	TransactionTypeCoppaCharge         = "coppa_charge"
	TransactionTypeCoppaRefund         = "coppa_refund"
)

// ─── Transaction Status Constants ───────────────────────────────────────────

const (
	TransactionStatusSucceeded = "succeeded"
	TransactionStatusPending   = "pending"
	TransactionStatusFailed    = "failed"
	TransactionStatusRefunded  = "refunded"
)

// ─── Payout Status Constants ────────────────────────────────────────────────

const (
	PayoutStatusPending    = "pending"
	PayoutStatusProcessing = "processing"
	PayoutStatusCompleted  = "completed"
	PayoutStatusFailed     = "failed"
)

// ─── Tier Constants ─────────────────────────────────────────────────────────

const (
	TierFree    = "free"
	TierPremium = "premium"
)

// ─── Billing Interval Constants ─────────────────────────────────────────────

const (
	IntervalMonthly = "monthly"
	IntervalAnnual  = "annual"
)

// ─── Request Types ──────────────────────────────────────────────────────────

// CreateSubscriptionCommand is the body for POST /v1/billing/subscription. (Phase 2) [10-billing §8.1]
type CreateSubscriptionCommand struct {
	BillingInterval string `json:"billing_interval" validate:"required,oneof=monthly annual"`
	PaymentMethodID string `json:"payment_method_id" validate:"required"`
}

// UpdateSubscriptionCommand is the body for PATCH /v1/billing/subscription. (Phase 2) [10-billing §8.1]
type UpdateSubscriptionCommand struct {
	BillingInterval string `json:"billing_interval" validate:"required,oneof=monthly annual"`
}

// CoppaVerificationCommand is the body for POST /v1/billing/coppa-verify. [10-billing §8.1]
type CoppaVerificationCommand struct {
	PaymentMethodID string `json:"payment_method_id" validate:"required"`
}

// AttachPaymentMethodCommand is the body for POST /v1/billing/payment-methods. (Phase 2) [10-billing §8.1]
type AttachPaymentMethodCommand struct {
	SetupIntentClientSecret string `json:"setup_intent_client_secret" validate:"required"`
}

// EstimateSubscriptionQuery is the body for POST /v1/billing/subscription/estimate. (Phase 2) [10-billing §8.1]
type EstimateSubscriptionQuery struct {
	BillingInterval string `json:"billing_interval" validate:"required,oneof=monthly annual"`
}

// TransactionListParams holds query parameters for GET /v1/billing/transactions. [10-billing §8.1]
type TransactionListParams struct {
	Cursor *string `query:"cursor"`
	Limit  *int    `query:"limit"`
}

// InvoiceListParams holds query parameters for GET /v1/billing/invoices. (Phase 2) [10-billing §8.1]
type InvoiceListParams struct {
	Cursor *string `query:"cursor"`
	Limit  *int    `query:"limit"`
}

// PayoutListParams holds query parameters for GET /v1/billing/payouts. (Phase 2) [10-billing §8.1]
type PayoutListParams struct {
	Cursor *string `query:"cursor"`
	Limit  *int    `query:"limit"`
}

// ─── Response Types ─────────────────────────────────────────────────────────

// SubscriptionResponse is the subscription status response. [10-billing §8.2]
type SubscriptionResponse struct {
	Tier              string     `json:"tier"`
	Status            *string    `json:"status"`
	BillingInterval   *string    `json:"billing_interval,omitempty"`
	CurrentPeriodEnd  *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd bool       `json:"cancel_at_period_end"`
	AmountCents       *int64     `json:"amount_cents,omitempty"`
	Currency          *string    `json:"currency,omitempty"`
}

// TransactionResponse is a single financial transaction. [10-billing §8.2]
type TransactionResponse struct {
	ID              uuid.UUID `json:"id"`
	TransactionType string    `json:"transaction_type"`
	Status          string    `json:"status"`
	AmountCents     int64     `json:"amount_cents"`
	Currency        string    `json:"currency"`
	Description     *string   `json:"description,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// TransactionListResponse is a paginated transaction list. [10-billing §8.2]
type TransactionListResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	NextCursor   *string               `json:"next_cursor,omitempty"`
}

// InvoiceResponse is a subscription invoice. (Phase 2) [10-billing §8.2]
type InvoiceResponse struct {
	ID          string     `json:"id"`
	AmountCents int64      `json:"amount_cents"`
	Currency    string     `json:"currency"`
	Status      string     `json:"status"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	PDFURL      *string    `json:"pdf_url,omitempty"`
}

// InvoiceListResponse is a paginated invoice list. (Phase 2) [10-billing §8.2]
type InvoiceListResponse struct {
	Invoices   []InvoiceResponse `json:"invoices"`
	NextCursor *string           `json:"next_cursor,omitempty"`
}

// PaymentMethodResponse is an attached payment method. (Phase 2) [10-billing §8.2]
type PaymentMethodResponse struct {
	ID         string  `json:"id"`
	MethodType string  `json:"method_type"`
	LastFour   *string `json:"last_four,omitempty"`
	Brand      *string `json:"brand,omitempty"`
	ExpMonth   *uint8  `json:"exp_month,omitempty"`
	ExpYear    *uint16 `json:"exp_year,omitempty"`
	IsDefault  bool    `json:"is_default"`
}

// EstimateResponse is pricing estimate for subscription changes. (Phase 2) [10-billing §8.2]
type EstimateResponse struct {
	AmountCents           int64     `json:"amount_cents"`
	Currency              string    `json:"currency"`
	BillingInterval       string    `json:"billing_interval"`
	ProrationCreditsCents int64     `json:"proration_credits_cents"`
	TotalDueTodayCents    int64     `json:"total_due_today_cents"`
	NextBillingDate       time.Time `json:"next_billing_date"`
}

// CoppaVerificationResult is the COPPA micro-charge verification result. [10-billing §8.2]
type CoppaVerificationResult struct {
	Verified bool   `json:"verified"`
	ChargeID string `json:"charge_id"`
	RefundID string `json:"refund_id"`
}

// PayoutResponse is a creator payout record. (Phase 2) [10-billing §8.2]
type PayoutResponse struct {
	ID                   uuid.UUID  `json:"id"`
	Status               string     `json:"status"`
	AmountCents          int64      `json:"amount_cents"`
	Currency             string     `json:"currency"`
	PeriodStart          time.Time  `json:"period_start"`
	PeriodEnd            time.Time  `json:"period_end"`
	PurchaseCount        int32      `json:"purchase_count"`
	RefundDeductionCents int64      `json:"refund_deduction_cents"`
	ProcessedAt          *time.Time `json:"processed_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

// PayoutListResponse is a paginated payout list. (Phase 2) [10-billing §8.2]
type PayoutListResponse struct {
	Payouts    []PayoutResponse `json:"payouts"`
	NextCursor *string          `json:"next_cursor,omitempty"`
}

// ─── Config ─────────────────────────────────────────────────────────────────

// BillingConfig holds runtime configuration for the billing domain. [10-billing §8.3]
type BillingConfig struct {
	HyperswitchAPIKey    string
	HyperswitchProfileID string
	HyperswitchBaseURL   string
	MonthlyPriceID       string
	AnnualPriceID        string
	CoppaChargeCents     int64
	WebhookSigningSecret string
}

// ─── Repository Input Types ─────────────────────────────────────────────────

// CreateSubscriptionRow is the input for creating a subscription record.
type CreateSubscriptionRow struct {
	FamilyID                  uuid.UUID
	HyperswitchSubscriptionID string
	HyperswitchCustomerID     string
	Tier                      string
	Status                    string
	BillingInterval           string
	CurrentPeriodStart        time.Time
	CurrentPeriodEnd          time.Time
	AmountCents               int64
	Currency                  string
	HyperswitchPriceID        string
}

// SubscriptionUpdate holds updatable fields for a subscription.
type SubscriptionUpdate struct {
	Status             *string
	BillingInterval    *string
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	CancelAtPeriodEnd  *bool
	CanceledAt         *time.Time
	AmountCents        *int64
	HyperswitchPriceID *string
}

// CreateTransactionRow is the input for creating a transaction record.
type CreateTransactionRow struct {
	FamilyID             uuid.UUID
	TransactionType      string
	Status               string
	AmountCents          int64
	Currency             string
	HyperswitchPaymentID *string
	HyperswitchInvoiceID *string
	Description          *string
	Metadata             map[string]any
}

// UpsertCustomerRow is the input for upserting a Hyperswitch customer mapping.
type UpsertCustomerRow struct {
	HyperswitchCustomerID  string
	DefaultPaymentMethodID *string
}

// CreatePayoutRow is the input for creating a payout record.
type CreatePayoutRow struct {
	CreatorID            uuid.UUID
	AmountCents          int64
	Currency             string
	PeriodStart          time.Time
	PeriodEnd            time.Time
	PurchaseCount        int32
	RefundDeductionCents int64
}

// ─── Adapter Supporting Types ───────────────────────────────────────────────

// HyperswitchSubscription is the adapter-level subscription representation. [10-billing §7]
type HyperswitchSubscription struct {
	ID                 string    `json:"id"`
	CustomerID         string    `json:"customer_id"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CancelAtPeriodEnd  bool      `json:"cancel_at_period_end"`
	PriceID            string    `json:"price_id"`
	AmountCents        int64     `json:"amount_cents"`
	Currency           string    `json:"currency"`
}

// HyperswitchEstimate is the adapter-level pricing estimate. [10-billing §7]
type HyperswitchEstimate struct {
	AmountCents           int64     `json:"amount_cents"`
	Currency              string    `json:"currency"`
	ProrationCreditsCents int64     `json:"proration_credits_cents"`
	TotalDueTodayCents    int64     `json:"total_due_today_cents"`
	NextBillingDate       time.Time `json:"next_billing_date"`
}

// SetupIntentResponse wraps Hyperswitch SetupIntent data for frontend confirmation.
type SetupIntentResponse struct {
	ClientSecret string `json:"client_secret"`
}

// HyperswitchPaymentMethod is the adapter-level payment method representation.
type HyperswitchPaymentMethod struct {
	ID         string  `json:"id"`
	MethodType string  `json:"method_type"`
	LastFour   *string `json:"last_four,omitempty"`
	Brand      *string `json:"brand,omitempty"`
	ExpMonth   *uint8  `json:"exp_month,omitempty"`
	ExpYear    *uint16 `json:"exp_year,omitempty"`
	IsDefault  bool    `json:"is_default"`
}

// HyperswitchInvoice is the adapter-level invoice representation.
type HyperswitchInvoice struct {
	ID          string     `json:"id"`
	AmountCents int64      `json:"amount_cents"`
	Currency    string     `json:"currency"`
	Status      string     `json:"status"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	PDFURL      *string    `json:"pdf_url,omitempty"`
}

// HyperswitchPayout is the adapter-level payout representation.
type HyperswitchPayout struct {
	ID          string `json:"id"`
	AmountCents int64  `json:"amount_cents"`
	Status      string `json:"status"`
}

// BillingWebhookEvent is the parsed webhook event from Hyperswitch. [10-billing §7]
type BillingWebhookEvent struct {
	Type                string
	SubscriptionCreated *BillingWebhookSubscriptionCreated
	SubscriptionUpdated *BillingWebhookSubscriptionUpdated
	SubscriptionDeleted *BillingWebhookSubscriptionDeleted
	InvoicePaid         *BillingWebhookInvoicePaid
	PaymentFailed       *BillingWebhookPaymentFailed
}

// BillingWebhookSubscriptionCreated carries data for subscription.created events.
type BillingWebhookSubscriptionCreated struct {
	Subscription HyperswitchSubscription
}

// BillingWebhookSubscriptionUpdated carries data for subscription.updated events.
type BillingWebhookSubscriptionUpdated struct {
	Subscription HyperswitchSubscription
}

// BillingWebhookSubscriptionDeleted carries data for subscription.deleted events.
type BillingWebhookSubscriptionDeleted struct {
	SubscriptionID string
}

// BillingWebhookInvoicePaid carries data for invoice.paid events.
type BillingWebhookInvoicePaid struct {
	InvoiceID      string
	SubscriptionID string
	AmountCents    int64
	PaymentID      string
}

// BillingWebhookPaymentFailed carries data for payment.failed events.
type BillingWebhookPaymentFailed struct {
	PaymentID      string
	SubscriptionID *string
	Reason         string
}
