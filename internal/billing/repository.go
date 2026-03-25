package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// PgSubscriptionRepository [10-billing §6]
// ═══════════════════════════════════════════════════════════════════════════════

type PgSubscriptionRepository struct{ db *gorm.DB }

func NewPgSubscriptionRepository(db *gorm.DB) *PgSubscriptionRepository {
	return &PgSubscriptionRepository{db: db}
}

func (r *PgSubscriptionRepository) Create(ctx context.Context, input CreateSubscriptionRow) (*BillSubscription, error) {
	sub := BillSubscription{
		FamilyID:                  input.FamilyID,
		HyperswitchSubscriptionID: input.HyperswitchSubscriptionID,
		HyperswitchCustomerID:     input.HyperswitchCustomerID,
		Tier:                      input.Tier,
		Status:                    input.Status,
		BillingInterval:           input.BillingInterval,
		CurrentPeriodStart:        input.CurrentPeriodStart,
		CurrentPeriodEnd:          input.CurrentPeriodEnd,
		AmountCents:               input.AmountCents,
		Currency:                  input.Currency,
		HyperswitchPriceID:        input.HyperswitchPriceID,
	}
	if err := r.db.WithContext(ctx).Create(&sub).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return &sub, nil
}

func (r *PgSubscriptionRepository) FindByFamily(ctx context.Context, scope shared.FamilyScope) (*BillSubscription, error) {
	var sub BillSubscription
	err := r.db.WithContext(ctx).Where("family_id = ?", scope.FamilyID()).First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return &sub, nil
}

func (r *PgSubscriptionRepository) FindByHyperswitchID(ctx context.Context, hyperswitchSubscriptionID string) (*BillSubscription, error) {
	var sub BillSubscription
	err := r.db.WithContext(ctx).Where("hyperswitch_subscription_id = ?", hyperswitchSubscriptionID).First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return &sub, nil
}

func (r *PgSubscriptionRepository) Update(ctx context.Context, subscriptionID uuid.UUID, updates SubscriptionUpdate) (*BillSubscription, error) {
	updateMap := map[string]any{"updated_at": time.Now()}
	if updates.Status != nil {
		updateMap["status"] = *updates.Status
	}
	if updates.BillingInterval != nil {
		updateMap["billing_interval"] = *updates.BillingInterval
	}
	if updates.CurrentPeriodStart != nil {
		updateMap["current_period_start"] = *updates.CurrentPeriodStart
	}
	if updates.CurrentPeriodEnd != nil {
		updateMap["current_period_end"] = *updates.CurrentPeriodEnd
	}
	if updates.CancelAtPeriodEnd != nil {
		updateMap["cancel_at_period_end"] = *updates.CancelAtPeriodEnd
	}
	if updates.CanceledAt != nil {
		updateMap["canceled_at"] = *updates.CanceledAt
	}
	if updates.AmountCents != nil {
		updateMap["amount_cents"] = *updates.AmountCents
	}
	if updates.HyperswitchPriceID != nil {
		updateMap["hyperswitch_price_id"] = *updates.HyperswitchPriceID
	}

	var sub BillSubscription
	if err := r.db.WithContext(ctx).Model(&sub).Where("id = ?", subscriptionID).Updates(updateMap).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	if err := r.db.WithContext(ctx).First(&sub, "id = ?", subscriptionID).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return &sub, nil
}

func (r *PgSubscriptionRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("family_id = ?", familyID).Delete(&BillSubscription{}).Error; err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgTransactionRepository [10-billing §6]
// ═══════════════════════════════════════════════════════════════════════════════

type PgTransactionRepository struct{ db *gorm.DB }

func NewPgTransactionRepository(db *gorm.DB) *PgTransactionRepository {
	return &PgTransactionRepository{db: db}
}

func (r *PgTransactionRepository) Create(ctx context.Context, input CreateTransactionRow) (*BillTransaction, error) {
	tx := BillTransaction{
		FamilyID:             input.FamilyID,
		TransactionType:      input.TransactionType,
		Status:               input.Status,
		AmountCents:          input.AmountCents,
		Currency:             input.Currency,
		HyperswitchPaymentID: input.HyperswitchPaymentID,
		HyperswitchInvoiceID: input.HyperswitchInvoiceID,
		Description:          input.Description,
		Metadata:             input.Metadata,
	}
	if tx.Metadata == nil {
		tx.Metadata = map[string]any{}
	}
	if err := r.db.WithContext(ctx).Create(&tx).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return &tx, nil
}

func (r *PgTransactionRepository) ListByFamily(ctx context.Context, scope shared.FamilyScope, params *TransactionListParams) ([]BillTransaction, error) {
	limit := 20
	if params != nil && params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = *params.Limit
	}

	q := r.db.WithContext(ctx).
		Where("family_id = ?", scope.FamilyID()).
		Order("created_at DESC, id DESC").
		Limit(limit + 1) // fetch one extra for hasMore

	if params != nil && params.Cursor != nil {
		cursorID, cursorAt, err := shared.DecodeCursor(*params.Cursor)
		if err != nil {
			return nil, err
		}
		q = q.Where("(created_at, id) < (?, ?)", cursorAt, cursorID)
	}

	var txns []BillTransaction
	if err := q.Find(&txns).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return txns, nil
}

func (r *PgTransactionRepository) ExistsByPaymentID(ctx context.Context, hyperswitchPaymentID string, transactionType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&BillTransaction{}).
		Where("hyperswitch_payment_id = ? AND transaction_type = ?", hyperswitchPaymentID, transactionType).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return count > 0, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgCustomerRepository [10-billing §6]
// ═══════════════════════════════════════════════════════════════════════════════

type PgCustomerRepository struct{ db *gorm.DB }

func NewPgCustomerRepository(db *gorm.DB) *PgCustomerRepository {
	return &PgCustomerRepository{db: db}
}

func (r *PgCustomerRepository) Upsert(ctx context.Context, familyID uuid.UUID, input UpsertCustomerRow) (*BillHyperswitchCustomer, error) {
	customer := BillHyperswitchCustomer{
		FamilyID:               familyID,
		HyperswitchCustomerID:  input.HyperswitchCustomerID,
		DefaultPaymentMethodID: input.DefaultPaymentMethodID,
	}

	result := r.db.WithContext(ctx).
		Where("family_id = ?", familyID).
		Assign(BillHyperswitchCustomer{
			HyperswitchCustomerID:  input.HyperswitchCustomerID,
			DefaultPaymentMethodID: input.DefaultPaymentMethodID,
		}).
		FirstOrCreate(&customer)

	if result.Error != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, result.Error)
	}
	return &customer, nil
}

func (r *PgCustomerRepository) FindByFamily(ctx context.Context, familyID uuid.UUID) (*BillHyperswitchCustomer, error) {
	var customer BillHyperswitchCustomer
	err := r.db.WithContext(ctx).Where("family_id = ?", familyID).First(&customer).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return &customer, nil
}

func (r *PgCustomerRepository) FindByHyperswitchID(ctx context.Context, hyperswitchCustomerID string) (*BillHyperswitchCustomer, error) {
	var customer BillHyperswitchCustomer
	err := r.db.WithContext(ctx).Where("hyperswitch_customer_id = ?", hyperswitchCustomerID).First(&customer).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return &customer, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgPayoutRepository (Phase 2) [10-billing §6]
// ═══════════════════════════════════════════════════════════════════════════════

type PgPayoutRepository struct{ db *gorm.DB }

func NewPgPayoutRepository(db *gorm.DB) *PgPayoutRepository {
	return &PgPayoutRepository{db: db}
}

func (r *PgPayoutRepository) Create(ctx context.Context, input CreatePayoutRow) (*BillPayout, error) {
	payout := BillPayout{
		CreatorID:            input.CreatorID,
		Status:               PayoutStatusPending,
		AmountCents:          input.AmountCents,
		Currency:             input.Currency,
		PeriodStart:          input.PeriodStart,
		PeriodEnd:            input.PeriodEnd,
		PurchaseCount:        input.PurchaseCount,
		RefundDeductionCents: input.RefundDeductionCents,
	}
	if err := r.db.WithContext(ctx).Create(&payout).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return &payout, nil
}

func (r *PgPayoutRepository) ListByCreator(ctx context.Context, creatorID uuid.UUID, params *PayoutListParams) ([]BillPayout, error) {
	limit := 20
	if params != nil && params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = *params.Limit
	}

	q := r.db.WithContext(ctx).
		Where("creator_id = ?", creatorID).
		Order("created_at DESC, id DESC").
		Limit(limit + 1)

	if params != nil && params.Cursor != nil {
		cursorID, cursorAt, err := shared.DecodeCursor(*params.Cursor)
		if err != nil {
			return nil, err
		}
		q = q.Where("(created_at, id) < (?, ?)", cursorAt, cursorID)
	}

	var payouts []BillPayout
	if err := q.Find(&payouts).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return payouts, nil
}

func (r *PgPayoutRepository) UpdateStatus(ctx context.Context, payoutID uuid.UUID, status string, hyperswitchPayoutID *string) (*BillPayout, error) {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	if hyperswitchPayoutID != nil {
		updates["hyperswitch_payout_id"] = *hyperswitchPayoutID
	}
	if status == PayoutStatusCompleted || status == PayoutStatusFailed {
		now := time.Now()
		updates["processed_at"] = now
	}

	var payout BillPayout
	if err := r.db.WithContext(ctx).Model(&payout).Where("id = ?", payoutID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	if err := r.db.WithContext(ctx).First(&payout, "id = ?", payoutID).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return &payout, nil
}

func (r *PgPayoutRepository) FindPending(ctx context.Context, limit uint32) ([]BillPayout, error) {
	var payouts []BillPayout
	err := r.db.WithContext(ctx).
		Where("status = ?", PayoutStatusPending).
		Order("created_at ASC").
		Limit(int(limit)).
		Find(&payouts).Error
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	return payouts, nil
}
