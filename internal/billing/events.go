package billing

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionCreated is published when a family's subscription becomes active for the first time.
// Consumed by iam:: (set tier=premium) and notify:: (welcome email). [10-billing §16.3]
type SubscriptionCreated struct {
	FamilyID         uuid.UUID `json:"family_id"`
	Tier             string    `json:"tier"`
	BillingInterval  string    `json:"billing_interval"`
	CurrentPeriodEnd time.Time `json:"current_period_end"`
}

func (SubscriptionCreated) EventName() string { return "billing.SubscriptionCreated" }

// SubscriptionChanged is published when a subscription is modified (interval change, renewal, reactivation).
// Consumed by iam:: (update tier if changed) and notify:: (plan change notification). [10-billing §16.3]
type SubscriptionChanged struct {
	FamilyID         uuid.UUID `json:"family_id"`
	Tier             string    `json:"tier"`
	BillingInterval  string    `json:"billing_interval"`
	CurrentPeriodEnd time.Time `json:"current_period_end"`
	ChangeType       string    `json:"change_type"` // "interval_change" | "renewal" | "reactivation"
}

func (SubscriptionChanged) EventName() string { return "billing.SubscriptionChanged" }

// SubscriptionCancelled is published when a subscription is fully canceled (end of term reached).
// Consumed by iam:: (set tier=free) and notify:: (cancellation confirmation).
//
// IMPORTANT: This event fires at the END of the billing period, not when
// the family requests cancellation. [S§15.3, 10-billing §16.3]
type SubscriptionCancelled struct {
	FamilyID    uuid.UUID `json:"family_id"`
	EffectiveAt time.Time `json:"effective_at"`
}

func (SubscriptionCancelled) EventName() string { return "billing.SubscriptionCancelled" }

// SubscriptionRenewalUpcoming is published when Hyperswitch fires invoice.upcoming (~7 days before renewal).
// Consumed by notify:: (advance renewal notice email + in-app notification). [10-billing §14, S§15.3]
type SubscriptionRenewalUpcoming struct {
	FamilyID    uuid.UUID `json:"family_id"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	RenewsAt    time.Time `json:"renews_at"`
}

func (SubscriptionRenewalUpcoming) EventName() string { return "billing.SubscriptionRenewalUpcoming" }

// PayoutCompleted is published when a creator payout is completed. (Phase 2)
// Consumed by notify:: (payout confirmation notification). [10-billing §16.3]
type PayoutCompleted struct {
	CreatorID   uuid.UUID `json:"creator_id"`
	PayoutID    uuid.UUID `json:"payout_id"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
}

func (PayoutCompleted) EventName() string { return "billing.PayoutCompleted" }

// CreatorThresholdReached is published the first time a creator's cumulative yearly earnings
// exceed the IRS 1099-K reporting threshold ($600/year). Consumed by notify::. [10-billing §16.3, HOM-62]
type CreatorThresholdReached struct {
	CreatorID     uuid.UUID `json:"creator_id"`
	TaxYear       int       `json:"tax_year"`
	EarningsCents int64     `json:"earnings_cents"`
}

func (CreatorThresholdReached) EventName() string { return "billing.CreatorThresholdReached" }
