package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// httpRequestsTotal counts all HTTP requests, labelled by method, path, and status code.
var httpRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	},
	[]string{"method", "path", "status"},
)

// httpRequestDurationSeconds records per-request latency as a histogram,
// labelled by method, path, and status code.
var httpRequestDurationSeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "path", "status"},
)

// Metrics returns an Echo middleware that records HTTP request count and
// duration for every request. Labels: method, path (raw route template),
// status (HTTP status code as a string).
//
// The /metrics endpoint itself is excluded to avoid self-instrumentation noise.
func Metrics() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			// Skip self-instrumentation of the metrics endpoint.
			if c.Request().URL.Path == "/metrics" {
				return err
			}

			// Use the matched route template (e.g. "/v1/families/:id") rather than
			// the concrete URL path to avoid high-cardinality label explosions.
			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}

			status := strconv.Itoa(c.Response().Status)
			if c.Response().Status == 0 {
				// Echo writes 200 by default; treat zero as 200.
				status = strconv.Itoa(http.StatusOK)
			}

			method := c.Request().Method
			elapsed := time.Since(start).Seconds()

			httpRequestsTotal.WithLabelValues(method, path, status).Inc()
			httpRequestDurationSeconds.WithLabelValues(method, path, status).Observe(elapsed)

			return err
		}
	}
}
