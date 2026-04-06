package lifecycle

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ─── Lifecycle Domain Prometheus Metrics ─────────────────────────────────────
// Instrument deletion processing, data export, and sweep operations. [15-data-lifecycle §14]

// LifecycleDeletionProcessingDuration records family deletion processing latency.
var LifecycleDeletionProcessingDuration = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "lifecycle_deletion_processing_duration_seconds",
		Help:    "Duration of family deletion processing across all domain handlers.",
		Buckets: prometheus.DefBuckets,
	},
)

// LifecycleDeletionTotal counts deletion operations by result.
var LifecycleDeletionTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "lifecycle_deletion_total",
		Help: "Total family deletion operations by result (success, error).",
	},
	[]string{"result"},
)

// LifecycleExportProcessingDuration records data export processing latency.
var LifecycleExportProcessingDuration = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "lifecycle_export_processing_duration_seconds",
		Help:    "Duration of family data export processing.",
		Buckets: prometheus.DefBuckets,
	},
)

// LifecycleSweepResultsTotal counts families processed per deletion sweep run.
var LifecycleSweepResultsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "lifecycle_sweep_results_total",
		Help: "Number of families processed per deletion sweep, by result.",
	},
	[]string{"result"},
)
