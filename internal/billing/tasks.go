package billing

// ─── Background Task Definitions (Phase 2) [10-billing §12] ─────────────────

// AggregatePayoutsPayload is the job payload for monthly payout aggregation.
// Runs on the 1st of each month at 6:00 AM UTC.
type AggregatePayoutsPayload struct{}

func (AggregatePayoutsPayload) TaskType() string { return "billing:aggregate_payouts" }

// ExecutePayoutsPayload is the job payload for executing pending payouts.
// Processes pending bill_payouts rows via adapter.CreatePayout().
type ExecutePayoutsPayload struct{}

func (ExecutePayoutsPayload) TaskType() string { return "billing:execute_payouts" }
