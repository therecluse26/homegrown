package shared

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ─── Job Queue Prometheus Metrics ────────────────────────────────────────────
// Instrument async job enqueue/process operations across all domains. [ARCH §2.5]

// JobsEnqueuedTotal counts jobs enqueued by task type and result.
var JobsEnqueuedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "jobs_enqueued_total",
		Help: "Total jobs enqueued by task type and result (success, error).",
	},
	[]string{"task_type", "result"},
)

// JobsProcessedTotal counts jobs processed by task type and result.
var JobsProcessedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "jobs_processed_total",
		Help: "Total jobs processed by task type and result (success, error).",
	},
	[]string{"task_type", "result"},
)

// JobsProcessingDuration records job processing duration by task type.
var JobsProcessingDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "jobs_processing_duration_seconds",
		Help:    "Duration of job processing by task type.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"task_type"},
)
