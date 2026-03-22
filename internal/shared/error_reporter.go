package shared

import "time"

// ErrorReporter is the generic error reporting port.
// The concrete implementation (sentryReporter) lives in cmd/server/main.go and wraps
// github.com/getsentry/sentry-go — no Sentry types leak into application-layer code. [ARCH §4.1]
type ErrorReporter interface {
	// CaptureException records an unexpected error for alerting and triage.
	CaptureException(err error)

	// Flush waits up to timeout for buffered events to be delivered.
	// Returns true if flushing completed within the timeout.
	Flush(timeout time.Duration) bool
}

// NoopErrorReporter satisfies ErrorReporter when Sentry is not configured.
// Used in development and test environments where SENTRY_DSN is absent.
type NoopErrorReporter struct{}

func (NoopErrorReporter) CaptureException(_ error)        {}
func (NoopErrorReporter) Flush(_ time.Duration) bool      { return true }
