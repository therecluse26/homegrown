package safety

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ─── Safety Domain Prometheus Metrics ────────────────────────────────────────
// Instrument content scanning, moderation, and report operations. [11-safety §14]

// SafetyScanTotal counts safety scan invocations by scan type and result.
var SafetyScanTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "safety_scan_total",
		Help: "Total safety scans by type (csam, moderation, grooming) and result (clean, flagged, error).",
	},
	[]string{"scan_type", "result"},
)

// SafetyFlagRateGauge tracks the current flag rate as a ratio of flagged/total scans.
var SafetyFlagRateGauge = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "safety_flag_rate",
		Help: "Current flag rate by scan type (rolling ratio of flagged to total).",
	},
	[]string{"scan_type"},
)

// SafetyModerationQueueDepth tracks the current depth of the moderation review queue.
var SafetyModerationQueueDepth = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "safety_moderation_queue_depth",
		Help: "Current number of items pending manual review in the moderation queue.",
	},
)

// SafetyReportsTotal counts NCMEC/abuse reports filed by type.
var SafetyReportsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "safety_reports_total",
		Help: "Total safety reports filed by type (ncmec, abuse, appeal).",
	},
	[]string{"type"},
)
