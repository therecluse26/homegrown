package search

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ─── Search Domain Prometheus Metrics ────────────────────────────────────────
// Instrument search query latency and index operations. [12-search §14]

// SearchQueryDuration records search query latency by collection.
var SearchQueryDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "search_query_duration_seconds",
		Help:    "Duration of search queries by collection.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"collection"},
)

// SearchIndexOperationsTotal counts index operations by collection, action, and result.
var SearchIndexOperationsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "search_index_operations_total",
		Help: "Total search index operations by collection, action (index, remove, bulk_index), and result.",
	},
	[]string{"collection", "action", "result"},
)
