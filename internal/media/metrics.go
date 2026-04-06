package media

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ─── Media Domain Prometheus Metrics ─────────────────────────────────────────
// Instrument upload processing, variant generation, and compression. [09-media §14]

// MediaUploadProcessingDuration records upload processing latency by status (success/failure).
var MediaUploadProcessingDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "media_upload_processing_duration_seconds",
		Help:    "Duration of media upload processing pipeline.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"status"},
)

// MediaUploadTotal counts uploads by context (avatar, post, resource) and status.
var MediaUploadTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "media_upload_total",
		Help: "Total media uploads by context and status.",
	},
	[]string{"context", "status"},
)

// MediaVariantGenerationDuration records variant generation latency by variant type.
var MediaVariantGenerationDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "media_variant_generation_duration_seconds",
		Help:    "Duration of media variant (thumbnail, medium) generation.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"variant"},
)

// MediaCompressionRatio records the compression ratio achieved during processing.
var MediaCompressionRatio = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "media_compression_ratio",
		Help:    "Compression ratio of processed media (original_size / compressed_size).",
		Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
	},
)

// MediaSafetyScanTotal counts safety scan invocations by scan type and result.
var MediaSafetyScanTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "media_safety_scan_total",
		Help: "Total safety scans by type (csam, moderation) and result (clean, flagged, error).",
	},
	[]string{"scan_type", "result"},
)
