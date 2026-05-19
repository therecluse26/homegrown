package billing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Background Task Definitions (Phase 2) [10-billing §12] ─────────────────

// AggregatePayoutsPayload is the job payload for monthly payout aggregation.
// Runs on the 1st of each month at 6:00 AM UTC.
type AggregatePayoutsPayload struct{}

func (AggregatePayoutsPayload) TaskType() string { return "billing:aggregate_payouts" }

// ExecutePayoutsPayload is the job payload for executing pending payouts.
// Processes pending bill_payouts rows via adapter.CreatePayout().
type ExecutePayoutsPayload struct{}

func (ExecutePayoutsPayload) TaskType() string { return "billing:execute_payouts" }

// RegisterTaskHandlers registers asynq task handlers for billing background jobs.
// Called from main.go during worker setup. [10-billing §12]
func RegisterTaskHandlers(
	worker shared.JobWorker,
	payoutRepo PayoutRepository,
	taxSummaryRepo CreatorTaxSummaryRepository,
	adapter SubscriptionPaymentAdapter,
	mktAdapter MktServiceForBilling,
	events *shared.EventBus,
) {
	worker.Handle("billing:aggregate_payouts", func(ctx context.Context, _ []byte) error {
		// Aggregate previous month's creator sales into bill_payouts rows. [10-billing §12]
		now := time.Now().UTC()
		periodStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		periodEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

		earnings, err := mktAdapter.GetAllCreatorSales(ctx, periodStart, periodEnd)
		if err != nil {
			slog.Error("billing: aggregate_payouts failed to fetch creator sales", "error", err)
			return err
		}
		if len(earnings) == 0 {
			slog.Info("billing: aggregate_payouts — no creator sales for period", "period_start", periodStart, "period_end", periodEnd)
		}

		var created int
		for _, e := range earnings {
			netPayout := e.TotalPayoutCents - e.RefundDeductionCents
			if netPayout <= 0 {
				continue
			}
			if _, createErr := payoutRepo.Create(ctx, CreatePayoutRow{
				CreatorID:            e.CreatorID,
				AmountCents:          netPayout,
				Currency:             "usd",
				PeriodStart:          periodStart,
				PeriodEnd:            periodEnd,
				PurchaseCount:        e.PurchaseCount,
				RefundDeductionCents: e.RefundDeductionCents,
			}); createErr != nil {
				slog.Error("billing: aggregate_payouts create failed", "creator_id", e.CreatorID, "error", createErr)
				continue
			}
			created++
		}
		slog.Info("billing: aggregate_payouts completed", "payouts_created", created, "period_start", periodStart, "period_end", periodEnd)

		// ── 1099-K tax summary update ─────────────────────────────────────────
		// Compute year-to-date cumulative earnings for each creator and upsert into
		// bill_creator_tax_summaries. Fire CreatorThresholdReached the first time
		// a creator crosses the $600 IRS threshold. [HOM-62]
		taxYear := now.Year()
		ytdStart := time.Date(taxYear, 1, 1, 0, 0, 0, 0, time.UTC)
		ytdEnd := now

		ytdEarnings, ytdErr := mktAdapter.GetAllCreatorSales(ctx, ytdStart, ytdEnd)
		if ytdErr != nil {
			slog.Error("billing: aggregate_payouts failed to fetch YTD sales for tax summary", "error", ytdErr)
			return nil // don't fail the payout task over tax summary errors
		}

		for _, e := range ytdEarnings {
			ytdNet := e.TotalPayoutCents - e.RefundDeductionCents
			if ytdNet < 0 {
				ytdNet = 0
			}

			prev, findErr := taxSummaryRepo.FindByCreatorAndYear(ctx, e.CreatorID, taxYear)
			wasBelow := findErr != nil || prev.EarningsCents < TaxThreshold1099KCents

			summary, upsertErr := taxSummaryRepo.Upsert(ctx, e.CreatorID, taxYear, ytdNet)
			if upsertErr != nil {
				slog.Error("billing: tax summary upsert failed", "creator_id", e.CreatorID, "error", upsertErr)
				continue
			}

			// Fire event if threshold newly crossed (was below, now at or above).
			if wasBelow && summary.EarningsCents >= TaxThreshold1099KCents && summary.ThresholdReachedAt == nil {
				reachedAt := time.Now()
				if _, setErr := taxSummaryRepo.SetThresholdReached(ctx, e.CreatorID, taxYear, reachedAt); setErr != nil {
					slog.Error("billing: set threshold reached failed", "creator_id", e.CreatorID, "error", setErr)
				}
				_ = events.Publish(ctx, CreatorThresholdReached{
					CreatorID:     e.CreatorID,
					TaxYear:       taxYear,
					EarningsCents: summary.EarningsCents,
				})
				slog.Info("billing: creator crossed 1099-K threshold", "creator_id", e.CreatorID, "tax_year", taxYear, "earnings_cents", summary.EarningsCents)
			}
		}
		return nil
	})

	worker.Handle("billing:execute_payouts", func(ctx context.Context, _ []byte) error {
		// BypassRLSTransaction: payouts are system-level cross-family operations run by background worker.
		pending, err := payoutRepo.FindPending(ctx, 50)
		if err != nil {
			slog.Error("billing: execute_payouts find pending", "error", err)
			return err
		}
		if len(pending) == 0 {
			slog.Info("billing: execute_payouts — no pending payouts")
			return nil
		}
		var failCount int
		for _, p := range pending {
			result, createErr := adapter.CreatePayout(ctx, p.CreatorID.String(), p.AmountCents, p.Currency, nil)
			if createErr != nil {
				failCount++
				slog.Error("billing: execute payout failed", "payout_id", p.ID, "error", createErr)
				if _, statusErr := payoutRepo.UpdateStatus(ctx, p.ID, "failed", nil); statusErr != nil {
					slog.Error("billing: failed to mark payout as failed", "payout_id", p.ID, "error", statusErr)
				}
				continue
			}
			if _, statusErr := payoutRepo.UpdateStatus(ctx, p.ID, "processing", &result.ID); statusErr != nil {
				slog.Error("billing: failed to mark payout as processing — payout may be re-processed", "payout_id", p.ID, "external_id", result.ID, "error", statusErr)
			}
		}
		slog.Info("billing: execute_payouts completed", "total", len(pending), "failed", failCount)
		if failCount > 0 {
			return fmt.Errorf("billing: %d of %d payouts failed", failCount, len(pending))
		}
		return nil
	})
}
