package billing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// BillingServiceImpl implements BillingService. [10-billing §5]
type BillingServiceImpl struct {
	subscriptionRepo SubscriptionRepository
	transactionRepo  TransactionRepository
	customerRepo     CustomerRepository
	payoutRepo       PayoutRepository
	adapter          SubscriptionPaymentAdapter
	iamService       IamServiceForBilling
	events           *shared.EventBus
	config           BillingConfig
}

// NewBillingService creates a new BillingServiceImpl.
func NewBillingService(
	subscriptionRepo SubscriptionRepository,
	transactionRepo TransactionRepository,
	customerRepo CustomerRepository,
	payoutRepo PayoutRepository,
	adapter SubscriptionPaymentAdapter,
	iamService IamServiceForBilling,
	events *shared.EventBus,
	config BillingConfig,
) *BillingServiceImpl {
	if config.CoppaChargeCents <= 0 {
		panic("billing: CoppaChargeCents must be > 0 (COPPA compliance requirement)")
	}
	return &BillingServiceImpl{
		subscriptionRepo: subscriptionRepo,
		transactionRepo:  transactionRepo,
		customerRepo:     customerRepo,
		payoutRepo:       payoutRepo,
		adapter:          adapter,
		iamService:       iamService,
		events:           events,
		config:           config,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Queries [10-billing §5]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *BillingServiceImpl) GetSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error) {
	sub, err := s.subscriptionRepo.FindByFamily(ctx, scope)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("get subscription: %w", err)}
	}

	// No subscription — return free-tier default. [10-billing §4.1]
	if sub == nil {
		return &SubscriptionResponse{
			Tier:              TierFree,
			CancelAtPeriodEnd: false,
		}, nil
	}

	return subscriptionToResponse(sub), nil
}

func (s *BillingServiceImpl) ListTransactions(ctx context.Context, params TransactionListParams, scope shared.FamilyScope) (*TransactionListResponse, error) {
	txns, err := s.transactionRepo.ListByFamily(ctx, scope, &params)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("list transactions: %w", err)}
	}

	limit := 20
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = *params.Limit
	}

	var nextCursor *string
	if len(txns) > limit {
		txns = txns[:limit]
		last := txns[limit-1]
		cursor := shared.EncodeCursor(last.ID, last.CreatedAt)
		nextCursor = &cursor
	}

	resp := &TransactionListResponse{
		Transactions: make([]TransactionResponse, len(txns)),
		NextCursor:   nextCursor,
	}
	for i, tx := range txns {
		resp.Transactions[i] = TransactionResponse{
			ID:              tx.ID,
			TransactionType: tx.TransactionType,
			Status:          tx.Status,
			AmountCents:     tx.AmountCents,
			Currency:        tx.Currency,
			Description:     tx.Description,
			CreatedAt:       tx.CreatedAt,
		}
	}
	return resp, nil
}

func (s *BillingServiceImpl) ListInvoices(ctx context.Context, params InvoiceListParams, scope shared.FamilyScope) (*InvoiceListResponse, error) {
	customer, err := s.customerRepo.FindByFamily(ctx, scope.FamilyID())
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("list invoices: %w", err)}
	}
	if customer == nil {
		return &InvoiceListResponse{Invoices: []InvoiceResponse{}}, nil
	}

	limit := uint32(20)
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = uint32(*params.Limit)
	}

	invoices, err := s.adapter.ListInvoices(ctx, customer.HyperswitchCustomerID, limit)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("list invoices from adapter: %w", err)}
	}

	resp := &InvoiceListResponse{
		Invoices: make([]InvoiceResponse, len(invoices)),
	}
	for i, inv := range invoices {
		resp.Invoices[i] = InvoiceResponse(inv)
	}
	return resp, nil
}

func (s *BillingServiceImpl) ListPaymentMethods(ctx context.Context, scope shared.FamilyScope) ([]PaymentMethodResponse, error) {
	customer, err := s.customerRepo.FindByFamily(ctx, scope.FamilyID())
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("list payment methods: %w", err)}
	}
	if customer == nil {
		return []PaymentMethodResponse{}, nil
	}

	methods, err := s.adapter.ListPaymentMethods(ctx, customer.HyperswitchCustomerID)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("list payment methods from adapter: %w", err)}
	}

	result := make([]PaymentMethodResponse, len(methods))
	for i, m := range methods {
		result[i] = PaymentMethodResponse(m)
	}
	return result, nil
}

func (s *BillingServiceImpl) EstimateSubscription(ctx context.Context, query EstimateSubscriptionQuery, scope shared.FamilyScope) (*EstimateResponse, error) {
	priceID, err := s.resolvePriceID(query.BillingInterval)
	if err != nil {
		return nil, &BillingError{Err: err}
	}

	customer, err := s.customerRepo.FindByFamily(ctx, scope.FamilyID())
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("estimate: %w", err)}
	}
	if customer == nil {
		return nil, &BillingError{Err: fmt.Errorf("estimate: %w", ErrPaymentAdapterUnavailable)}
	}

	// Check if there's an existing subscription for proration calculation
	sub, err := s.subscriptionRepo.FindByFamily(ctx, scope)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("estimate: %w", err)}
	}

	var currentSubID *string
	if sub != nil {
		currentSubID = &sub.HyperswitchSubscriptionID
	}

	estimate, err := s.adapter.EstimateSubscription(ctx, customer.HyperswitchCustomerID, priceID, currentSubID)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("estimate from adapter: %w", err)}
	}

	return &EstimateResponse{
		AmountCents:           estimate.AmountCents,
		Currency:              estimate.Currency,
		BillingInterval:       query.BillingInterval,
		ProrationCreditsCents: estimate.ProrationCreditsCents,
		TotalDueTodayCents:    estimate.TotalDueTodayCents,
		NextBillingDate:       estimate.NextBillingDate,
	}, nil
}

func (s *BillingServiceImpl) ListPayouts(ctx context.Context, params PayoutListParams, creatorID uuid.UUID) (*PayoutListResponse, error) {
	payouts, err := s.payoutRepo.ListByCreator(ctx, creatorID, &params)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("list payouts: %w", err)}
	}

	limit := 20
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = *params.Limit
	}

	var nextCursor *string
	if len(payouts) > limit {
		payouts = payouts[:limit]
		last := payouts[limit-1]
		cursor := shared.EncodeCursor(last.ID, last.CreatedAt)
		nextCursor = &cursor
	}

	resp := &PayoutListResponse{
		Payouts:    make([]PayoutResponse, len(payouts)),
		NextCursor: nextCursor,
	}
	for i, p := range payouts {
		resp.Payouts[i] = PayoutResponse{
			ID:                   p.ID,
			Status:               p.Status,
			AmountCents:          p.AmountCents,
			Currency:             p.Currency,
			PeriodStart:          p.PeriodStart,
			PeriodEnd:            p.PeriodEnd,
			PurchaseCount:        p.PurchaseCount,
			RefundDeductionCents: p.RefundDeductionCents,
			ProcessedAt:          p.ProcessedAt,
			CreatedAt:            p.CreatedAt,
		}
	}
	return resp, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Subscription CRUD (Phase 2) [10-billing §5]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *BillingServiceImpl) CreateSubscription(ctx context.Context, cmd CreateSubscriptionCommand, scope shared.FamilyScope) (*SubscriptionResponse, error) {
	// Check if subscription already exists
	existing, err := s.subscriptionRepo.FindByFamily(ctx, scope)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("create subscription: %w", err)}
	}
	if existing != nil {
		return nil, &BillingError{Err: ErrSubscriptionAlreadyExists}
	}

	priceID, err := s.resolvePriceID(cmd.BillingInterval)
	if err != nil {
		return nil, &BillingError{Err: err}
	}

	// Get or create Hyperswitch customer
	customerID, err := s.getOrCreateCustomer(ctx, scope.FamilyID())
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("create subscription: %w", err)}
	}

	// Create subscription via adapter
	hsSub, err := s.adapter.CreateSubscription(ctx, customerID, priceID, cmd.PaymentMethodID, map[string]string{
		"family_id": scope.FamilyID().String(),
	})
	if err != nil {
		return nil, &BillingError{Err: err}
	}

	// Create local subscription record with status=incomplete (updated to active via webhook)
	sub, err := s.subscriptionRepo.Create(ctx, CreateSubscriptionRow{
		FamilyID:                  scope.FamilyID(),
		HyperswitchSubscriptionID: hsSub.ID,
		HyperswitchCustomerID:     hsSub.CustomerID,
		Tier:                      TierPremium,
		Status:                    SubscriptionStatusIncomplete,
		BillingInterval:           cmd.BillingInterval,
		CurrentPeriodStart:        hsSub.CurrentPeriodStart,
		CurrentPeriodEnd:          hsSub.CurrentPeriodEnd,
		AmountCents:               hsSub.AmountCents,
		Currency:                  hsSub.Currency,
		HyperswitchPriceID:        priceID,
	})
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("create subscription row: %w", err)}
	}

	return subscriptionToResponse(sub), nil
}

func (s *BillingServiceImpl) UpdateSubscription(ctx context.Context, cmd UpdateSubscriptionCommand, scope shared.FamilyScope) (*SubscriptionResponse, error) {
	sub, err := s.subscriptionRepo.FindByFamily(ctx, scope)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("update subscription: %w", err)}
	}
	if sub == nil {
		return nil, &BillingError{Err: ErrSubscriptionNotFound}
	}
	if sub.Status != SubscriptionStatusActive {
		return nil, &BillingError{Err: ErrSubscriptionNotActive}
	}

	priceID, err := s.resolvePriceID(cmd.BillingInterval)
	if err != nil {
		return nil, &BillingError{Err: err}
	}

	_, err = s.adapter.UpdateSubscription(ctx, sub.HyperswitchSubscriptionID, priceID)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("update subscription in adapter: %w", err)}
	}

	// Update local mirror
	updated, err := s.subscriptionRepo.Update(ctx, sub.ID, SubscriptionUpdate{
		BillingInterval:    &cmd.BillingInterval,
		HyperswitchPriceID: &priceID,
	})
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("update subscription row: %w", err)}
	}

	return subscriptionToResponse(updated), nil
}

func (s *BillingServiceImpl) CancelSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error) {
	sub, err := s.subscriptionRepo.FindByFamily(ctx, scope)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("cancel subscription: %w", err)}
	}
	if sub == nil {
		return nil, &BillingError{Err: ErrSubscriptionNotFound}
	}
	if sub.Status != SubscriptionStatusActive || sub.CancelAtPeriodEnd {
		return nil, &BillingError{Err: ErrSubscriptionNotActive}
	}

	_, err = s.adapter.CancelSubscription(ctx, sub.HyperswitchSubscriptionID)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("cancel subscription in adapter: %w", err)}
	}

	// Set cancel_at_period_end and canceled_at locally
	now := time.Now()
	trueVal := true
	updated, err := s.subscriptionRepo.Update(ctx, sub.ID, SubscriptionUpdate{
		CancelAtPeriodEnd: &trueVal,
		CanceledAt:        &now,
	})
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("update subscription for cancel: %w", err)}
	}

	return subscriptionToResponse(updated), nil
}

func (s *BillingServiceImpl) ReactivateSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error) {
	sub, err := s.subscriptionRepo.FindByFamily(ctx, scope)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("reactivate subscription: %w", err)}
	}
	if sub == nil {
		return nil, &BillingError{Err: ErrSubscriptionNotFound}
	}
	if !sub.CancelAtPeriodEnd || sub.Status != SubscriptionStatusActive {
		return nil, &BillingError{Err: ErrCannotReactivate}
	}

	_, err = s.adapter.ReactivateSubscription(ctx, sub.HyperswitchSubscriptionID)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("reactivate subscription in adapter: %w", err)}
	}

	// Clear cancel_at_period_end and canceled_at
	falseVal := false
	updated, err := s.subscriptionRepo.Update(ctx, sub.ID, SubscriptionUpdate{
		CancelAtPeriodEnd: &falseVal,
	})
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("update subscription for reactivate: %w", err)}
	}

	return subscriptionToResponse(updated), nil
}

func (s *BillingServiceImpl) PauseSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error) {
	sub, err := s.subscriptionRepo.FindByFamily(ctx, scope)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("pause subscription: %w", err)}
	}
	if sub == nil {
		return nil, &BillingError{Err: ErrSubscriptionNotFound}
	}
	if sub.Status != SubscriptionStatusActive {
		return nil, &BillingError{Err: ErrSubscriptionNotActive}
	}

	_, err = s.adapter.PauseSubscription(ctx, sub.HyperswitchSubscriptionID)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("pause subscription in adapter: %w", err)}
	}

	paused := SubscriptionStatusPaused
	updated, err := s.subscriptionRepo.Update(ctx, sub.ID, SubscriptionUpdate{Status: &paused})
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("update subscription for pause: %w", err)}
	}

	return subscriptionToResponse(updated), nil
}

func (s *BillingServiceImpl) ResumeSubscription(ctx context.Context, scope shared.FamilyScope) (*SubscriptionResponse, error) {
	sub, err := s.subscriptionRepo.FindByFamily(ctx, scope)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("resume subscription: %w", err)}
	}
	if sub == nil {
		return nil, &BillingError{Err: ErrSubscriptionNotFound}
	}
	if sub.Status != SubscriptionStatusPaused {
		return nil, &BillingError{Err: ErrSubscriptionNotPaused}
	}

	_, err = s.adapter.ResumeSubscription(ctx, sub.HyperswitchSubscriptionID)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("resume subscription in adapter: %w", err)}
	}

	active := SubscriptionStatusActive
	updated, err := s.subscriptionRepo.Update(ctx, sub.ID, SubscriptionUpdate{Status: &active})
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("update subscription for resume: %w", err)}
	}

	return subscriptionToResponse(updated), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Payment Methods (Phase 2) [10-billing §5]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *BillingServiceImpl) AttachPaymentMethod(ctx context.Context, _ AttachPaymentMethodCommand, scope shared.FamilyScope) (*PaymentMethodResponse, error) {
	customerID, err := s.getOrCreateCustomer(ctx, scope.FamilyID())
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("attach payment method: %w", err)}
	}

	intent, err := s.adapter.CreateSetupIntent(ctx, customerID)
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("create setup intent: %w", err)}
	}

	return &PaymentMethodResponse{
		ID:         intent.ClientSecret,
		MethodType: "setup_intent",
	}, nil
}

func (s *BillingServiceImpl) DetachPaymentMethod(ctx context.Context, paymentMethodID string, scope shared.FamilyScope) error {
	// Check if there's an active subscription
	sub, err := s.subscriptionRepo.FindByFamily(ctx, scope)
	if err != nil {
		return &BillingError{Err: fmt.Errorf("detach payment method: %w", err)}
	}

	if sub != nil && sub.Status == SubscriptionStatusActive {
		// Verify this isn't the last payment method
		customer, err := s.customerRepo.FindByFamily(ctx, scope.FamilyID())
		if err != nil {
			return &BillingError{Err: fmt.Errorf("detach payment method: %w", err)}
		}
		if customer != nil {
			methods, err := s.adapter.ListPaymentMethods(ctx, customer.HyperswitchCustomerID)
			if err != nil {
				return &BillingError{Err: fmt.Errorf("list methods for detach check: %w", err)}
			}
			if len(methods) <= 1 {
				return &BillingError{Err: ErrCannotRemoveLastPaymentMethod}
			}
		}
	}

	if err := s.adapter.DetachPaymentMethod(ctx, paymentMethodID); err != nil {
		return &BillingError{Err: err}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// COPPA Verification [10-billing §13]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *BillingServiceImpl) ProcessCoppaVerification(ctx context.Context, cmd CoppaVerificationCommand, scope shared.FamilyScope) (*CoppaVerificationResult, error) {
	// Get or create Hyperswitch customer
	customerID, err := s.getOrCreateCustomer(ctx, scope.FamilyID())
	if err != nil {
		return nil, &BillingError{Err: fmt.Errorf("coppa verification: %w", err)}
	}

	// Process micro-charge via adapter (charge + immediate refund)
	chargeID, refundID, err := s.adapter.ProcessMicroCharge(
		ctx, customerID, cmd.PaymentMethodID, s.config.CoppaChargeCents,
		"COPPA parental consent verification",
		map[string]string{
			"family_id": scope.FamilyID().String(),
			"purpose":   "coppa_verification",
		},
	)
	if err != nil {
		return nil, &BillingError{Err: err}
	}

	// Create charge transaction row
	chargeDesc := "COPPA verification charge"
	_, err = s.transactionRepo.Create(ctx, CreateTransactionRow{
		FamilyID:             scope.FamilyID(),
		TransactionType:      TransactionTypeCoppaCharge,
		Status:               TransactionStatusSucceeded,
		AmountCents:          s.config.CoppaChargeCents,
		Currency:             "usd",
		HyperswitchPaymentID: &chargeID,
		Description:          &chargeDesc,
	})
	if err != nil {
		slog.Error("failed to create COPPA charge transaction", "family_id", scope.FamilyID(), "error", err)
	}

	// Create refund transaction row (even if refund failed — charge was verified)
	if refundID != "" {
		refundDesc := "COPPA verification refund"
		_, err = s.transactionRepo.Create(ctx, CreateTransactionRow{
			FamilyID:             scope.FamilyID(),
			TransactionType:      TransactionTypeCoppaRefund,
			Status:               TransactionStatusSucceeded,
			AmountCents:          s.config.CoppaChargeCents,
			Currency:             "usd",
			HyperswitchPaymentID: &refundID,
			Description:          &refundDesc,
		})
		if err != nil {
			slog.Error("failed to create COPPA refund transaction", "family_id", scope.FamilyID(), "error", err)
		}
	} else {
		// Refund failed — charge was verified, log warning. [10-billing §13]
		slog.Warn("COPPA refund failed after charge — charge was verified",
			"family_id", scope.FamilyID(),
			"charge_id", chargeID,
		)
	}

	return &CoppaVerificationResult{
		Verified: true,
		ChargeID: chargeID,
		RefundID: refundID,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Webhook Processing [10-billing §14]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *BillingServiceImpl) ProcessHyperswitchWebhook(ctx context.Context, payload []byte, signature string) error {
	// Step 1: Verify signature
	valid, err := s.adapter.VerifyWebhook(ctx, payload, signature)
	if err != nil || !valid {
		slog.Warn("invalid webhook signature", "error", err)
		return nil // always return nil — webhook endpoint returns 200 regardless
	}

	// Step 2: Parse event
	event, err := s.adapter.ParseWebhookEvent(ctx, payload)
	if err != nil {
		slog.Error("failed to parse webhook event", "error", err)
		return nil
	}

	// Step 3: Dispatch by event type
	switch event.Type {
	case "subscription.created":
		return s.handleWebhookSubscriptionCreated(ctx, event.SubscriptionCreated)
	case "subscription.updated":
		return s.handleWebhookSubscriptionUpdated(ctx, event.SubscriptionUpdated)
	case "subscription.deleted":
		return s.handleWebhookSubscriptionDeleted(ctx, event.SubscriptionDeleted)
	case "invoice.paid":
		return s.handleWebhookInvoicePaid(ctx, event.InvoicePaid)
	case "payment.failed":
		return s.handleWebhookPaymentFailed(ctx, event.PaymentFailed)
	default:
		slog.Info("unhandled webhook event type", "type", event.Type)
		return nil
	}
}

func (s *BillingServiceImpl) handleWebhookSubscriptionCreated(ctx context.Context, data *BillingWebhookSubscriptionCreated) error {
	if data == nil {
		return nil
	}

	// Upsert local subscription record
	existing, err := s.subscriptionRepo.FindByHyperswitchID(ctx, data.Subscription.ID)
	if err != nil {
		slog.Error("webhook subscription.created: find error", "error", err)
		return nil
	}

	if existing != nil {
		// Already exists — update fields
		_, err = s.subscriptionRepo.Update(ctx, existing.ID, SubscriptionUpdate{
			Status:             &data.Subscription.Status,
			CurrentPeriodStart: &data.Subscription.CurrentPeriodStart,
			CurrentPeriodEnd:   &data.Subscription.CurrentPeriodEnd,
			AmountCents:        &data.Subscription.AmountCents,
		})
		if err != nil {
			slog.Error("webhook subscription.created: update error", "error", err)
		}
		return nil
	}

	// Find customer by Hyperswitch customer ID to get family_id
	customer, err := s.customerRepo.FindByHyperswitchID(ctx, data.Subscription.CustomerID)
	if err != nil || customer == nil {
		slog.Error("webhook subscription.created: customer not found", "customer_id", data.Subscription.CustomerID)
		return nil
	}

	_, err = s.subscriptionRepo.Create(ctx, CreateSubscriptionRow{
		FamilyID:                  customer.FamilyID,
		HyperswitchSubscriptionID: data.Subscription.ID,
		HyperswitchCustomerID:     data.Subscription.CustomerID,
		Tier:                      TierPremium,
		Status:                    data.Subscription.Status,
		BillingInterval:           IntervalMonthly, // default, updated on subsequent events
		CurrentPeriodStart:        data.Subscription.CurrentPeriodStart,
		CurrentPeriodEnd:          data.Subscription.CurrentPeriodEnd,
		AmountCents:               data.Subscription.AmountCents,
		Currency:                  data.Subscription.Currency,
		HyperswitchPriceID:        data.Subscription.PriceID,
	})
	if err != nil {
		slog.Error("webhook subscription.created: create error", "error", err)
	}
	return nil
}

func (s *BillingServiceImpl) handleWebhookSubscriptionUpdated(ctx context.Context, data *BillingWebhookSubscriptionUpdated) error {
	if data == nil {
		return nil
	}

	sub, err := s.subscriptionRepo.FindByHyperswitchID(ctx, data.Subscription.ID)
	if err != nil || sub == nil {
		slog.Error("webhook subscription.updated: subscription not found", "subscription_id", data.Subscription.ID)
		return nil
	}

	wasActive := sub.Status == SubscriptionStatusActive
	cancelAtPeriodEnd := data.Subscription.CancelAtPeriodEnd

	// Update local mirror
	_, err = s.subscriptionRepo.Update(ctx, sub.ID, SubscriptionUpdate{
		Status:             &data.Subscription.Status,
		CurrentPeriodStart: &data.Subscription.CurrentPeriodStart,
		CurrentPeriodEnd:   &data.Subscription.CurrentPeriodEnd,
		CancelAtPeriodEnd:  &cancelAtPeriodEnd,
		AmountCents:        &data.Subscription.AmountCents,
		HyperswitchPriceID: &data.Subscription.PriceID,
	})
	if err != nil {
		slog.Error("webhook subscription.updated: update error", "error", err)
		return nil
	}

	// Publish domain events based on state transition
	if data.Subscription.Status == SubscriptionStatusActive && !wasActive {
		// First activation → SubscriptionCreated
		_ = s.events.Publish(ctx, SubscriptionCreated{
			FamilyID:         sub.FamilyID,
			Tier:             TierPremium,
			BillingInterval:  sub.BillingInterval,
			CurrentPeriodEnd: data.Subscription.CurrentPeriodEnd,
		})
	} else if data.Subscription.Status == SubscriptionStatusActive && wasActive {
		// Subsequent update → SubscriptionChanged
		_ = s.events.Publish(ctx, SubscriptionChanged{
			FamilyID:         sub.FamilyID,
			Tier:             TierPremium,
			BillingInterval:  sub.BillingInterval,
			CurrentPeriodEnd: data.Subscription.CurrentPeriodEnd,
			ChangeType:       "interval_change",
		})
	}

	return nil
}

func (s *BillingServiceImpl) handleWebhookSubscriptionDeleted(ctx context.Context, data *BillingWebhookSubscriptionDeleted) error {
	if data == nil {
		return nil
	}

	sub, err := s.subscriptionRepo.FindByHyperswitchID(ctx, data.SubscriptionID)
	if err != nil || sub == nil {
		slog.Error("webhook subscription.deleted: subscription not found", "subscription_id", data.SubscriptionID)
		return nil
	}

	canceledStatus := SubscriptionStatusCanceled
	_, err = s.subscriptionRepo.Update(ctx, sub.ID, SubscriptionUpdate{
		Status: &canceledStatus,
	})
	if err != nil {
		slog.Error("webhook subscription.deleted: update error", "error", err)
		return nil
	}

	// Publish SubscriptionCancelled event
	_ = s.events.Publish(ctx, SubscriptionCancelled{
		FamilyID:    sub.FamilyID,
		EffectiveAt: time.Now(),
	})

	return nil
}

func (s *BillingServiceImpl) handleWebhookInvoicePaid(ctx context.Context, data *BillingWebhookInvoicePaid) error {
	if data == nil {
		return nil
	}

	// Idempotency check: skip if we already recorded this payment
	exists, err := s.transactionRepo.ExistsByPaymentID(ctx, data.PaymentID, TransactionTypeSubscriptionPayment)
	if err != nil {
		slog.Error("webhook invoice.paid: idempotency check error", "error", err)
		return nil
	}
	if exists {
		return nil // duplicate — already processed
	}

	// Find subscription to get family_id
	sub, err := s.subscriptionRepo.FindByHyperswitchID(ctx, data.SubscriptionID)
	if err != nil || sub == nil {
		slog.Error("webhook invoice.paid: subscription not found", "subscription_id", data.SubscriptionID)
		return nil
	}

	desc := "Premium subscription payment"
	_, err = s.transactionRepo.Create(ctx, CreateTransactionRow{
		FamilyID:             sub.FamilyID,
		TransactionType:      TransactionTypeSubscriptionPayment,
		Status:               TransactionStatusSucceeded,
		AmountCents:          data.AmountCents,
		Currency:             "usd",
		HyperswitchPaymentID: &data.PaymentID,
		HyperswitchInvoiceID: &data.InvoiceID,
		Description:          &desc,
	})
	if err != nil {
		slog.Error("webhook invoice.paid: create transaction error", "error", err)
	}
	return nil
}

func (s *BillingServiceImpl) handleWebhookPaymentFailed(ctx context.Context, data *BillingWebhookPaymentFailed) error {
	if data == nil {
		return nil
	}

	if data.SubscriptionID == nil {
		return nil
	}

	sub, err := s.subscriptionRepo.FindByHyperswitchID(ctx, *data.SubscriptionID)
	if err != nil || sub == nil {
		slog.Error("webhook payment.failed: subscription not found", "subscription_id", *data.SubscriptionID)
		return nil
	}

	pastDueStatus := SubscriptionStatusPastDue
	_, err = s.subscriptionRepo.Update(ctx, sub.ID, SubscriptionUpdate{
		Status: &pastDueStatus,
	})
	if err != nil {
		slog.Error("webhook payment.failed: update error", "error", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Event Handlers [10-billing §16.4]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *BillingServiceImpl) HandleFamilyDeletionScheduled(ctx context.Context, event FamilyDeletionScheduledEvent) error {
	// Cancel subscription in Hyperswitch immediately (no end-of-term wait)
	sub, err := s.subscriptionRepo.FindByFamily(ctx, shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: event.FamilyID}))
	if err != nil {
		return fmt.Errorf("handle family deletion: %w", err)
	}
	if sub != nil && sub.Status == SubscriptionStatusActive {
		if _, err := s.adapter.CancelSubscription(ctx, sub.HyperswitchSubscriptionID); err != nil {
			slog.Error("failed to cancel subscription on family deletion", "family_id", event.FamilyID, "error", err)
		}
	}

	// Delete local records
	if err := s.subscriptionRepo.DeleteByFamily(ctx, event.FamilyID); err != nil {
		slog.Error("failed to delete subscription on family deletion", "family_id", event.FamilyID, "error", err)
	}
	return nil
}

func (s *BillingServiceImpl) HandlePrimaryParentTransferred(ctx context.Context, event PrimaryParentTransferredEvent) error {
	customer, err := s.customerRepo.FindByFamily(ctx, event.FamilyID)
	if err != nil || customer == nil {
		return nil // no Hyperswitch customer — nothing to update
	}

	// Look up the new primary's email directly from IAM (authoritative source).
	// The event fires after the transfer is committed, so GetFamilyPrimaryEmail returns the new primary's email.
	email, _, err := s.iamService.GetFamilyPrimaryEmail(ctx, event.FamilyID)
	if err != nil {
		slog.Error("billing: failed to look up new primary email after transfer", "family_id", event.FamilyID, "error", err)
		return nil
	}

	if err := s.adapter.UpdateCustomer(ctx, customer.HyperswitchCustomerID, email, ""); err != nil {
		slog.Error("billing: failed to update Hyperswitch customer email after transfer", "family_id", event.FamilyID, "error", err)
	}
	return nil
}

func (s *BillingServiceImpl) HandlePurchaseCompleted(_ context.Context, event PurchaseCompletedEvent) error {
	// Phase 2: Record creator earnings for payout aggregation.
	// For now, no-op. AggregatePayoutsTask calculates earnings from mkt_purchases directly.
	slog.Info("billing: PurchaseCompleted received (deferred to AggregatePayoutsTask)",
		"purchase_id", event.PurchaseID,
	)
	return nil
}

func (s *BillingServiceImpl) HandlePurchaseRefunded(_ context.Context, event PurchaseRefundedEvent) error {
	// Phase 2: Deduct refund from creator earnings.
	// For now, no-op. AggregatePayoutsTask accounts for refunds when calculating earnings.
	slog.Info("billing: PurchaseRefunded received (deferred to AggregatePayoutsTask)",
		"purchase_id", event.PurchaseID,
		"refund_cents", event.RefundAmountCents,
	)
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// resolvePriceID maps billing_interval to the correct Hyperswitch price ID. [10-billing §10]
func (s *BillingServiceImpl) resolvePriceID(interval string) (string, error) {
	switch interval {
	case IntervalMonthly:
		return s.config.MonthlyPriceID, nil
	case IntervalAnnual:
		return s.config.AnnualPriceID, nil
	default:
		return "", ErrInvalidBillingInterval
	}
}

// getOrCreateCustomer ensures a Hyperswitch customer exists for the family.
func (s *BillingServiceImpl) getOrCreateCustomer(ctx context.Context, familyID uuid.UUID) (string, error) {
	existing, err := s.customerRepo.FindByFamily(ctx, familyID)
	if err != nil {
		return "", fmt.Errorf("find customer: %w", err)
	}
	if existing != nil {
		return existing.HyperswitchCustomerID, nil
	}

	// Create new Hyperswitch customer
	email, displayName, err := s.iamService.GetFamilyPrimaryEmail(ctx, familyID)
	if err != nil {
		return "", fmt.Errorf("get family email: %w", err)
	}

	hsCustomerID, err := s.adapter.CreateCustomer(ctx, email, displayName, map[string]string{
		"family_id": familyID.String(),
	})
	if err != nil {
		return "", fmt.Errorf("create customer in adapter: %w", err)
	}

	// Save mapping
	_, err = s.customerRepo.Upsert(ctx, familyID, UpsertCustomerRow{
		HyperswitchCustomerID: hsCustomerID,
	})
	if err != nil {
		return "", fmt.Errorf("upsert customer: %w", err)
	}

	return hsCustomerID, nil
}

// subscriptionToResponse converts a BillSubscription to a SubscriptionResponse.
func subscriptionToResponse(sub *BillSubscription) *SubscriptionResponse {
	return &SubscriptionResponse{
		Tier:              sub.Tier,
		Status:            &sub.Status,
		BillingInterval:   &sub.BillingInterval,
		CurrentPeriodEnd:  &sub.CurrentPeriodEnd,
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
		AmountCents:       &sub.AmountCents,
		Currency:          &sub.Currency,
	}
}
