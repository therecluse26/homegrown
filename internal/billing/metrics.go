package billing

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ─── Billing Domain Prometheus Metrics ───────────────────────────────────────
// Instrument subscription operations, payment processing, and webhook handling. [10-billing §14]

// BillingSubscriptionOperationsTotal counts subscription lifecycle operations.
var BillingSubscriptionOperationsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "billing_subscription_operations_total",
		Help: "Total subscription operations by action (create, upgrade, downgrade, cancel, pause, resume) and result.",
	},
	[]string{"action", "result"},
)

// BillingPaymentProcessingDuration records payment processing latency.
var BillingPaymentProcessingDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "billing_payment_processing_duration_seconds",
		Help:    "Duration of payment processing operations.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"operation"},
)

// BillingWebhookTotal counts incoming billing webhooks by type and result.
var BillingWebhookTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "billing_webhook_total",
		Help: "Total billing webhooks received by event type and result.",
	},
	[]string{"event_type", "result"},
)
