// Package shared — retry.go
// Provides RetryableHTTPDo: a thin wrapper around http.Client.Do that retries
// transient failures with exponential backoff and jitter.
//
// # Where to apply
//
// The following adapter call-sites are good candidates for adoption:
//   - internal/iam/adapters/kratos.go      — KratosAdapterImpl.doRequest / httpClient.Do
//   - internal/billing/adapters/hyperswitch.go — HyperswitchSubscriptionAdapter.doRequest
//   - internal/mkt/adapters/payment.go     — HyperswitchPaymentAdapter.doRequest
//   - internal/search/adapters/typesense.go — HttpTypesenseAdapter (all client.Do calls)
//   - internal/notify/adapters/postmark.go  — PostmarkEmailAdapter (all client.Do calls)
//
// To integrate, replace `client.Do(req)` with `shared.RetryableHTTPDo(ctx, client, req, nil)`.
// Pass a non-nil *RetryConfig to override defaults.
package shared

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"time"
)

// RetryConfig controls the retry behaviour of RetryableHTTPDo.
type RetryConfig struct {
	// MaxAttempts is the total number of attempts (first try + retries).
	// Must be >= 1. Defaults to 3 (i.e. up to 2 retries).
	MaxAttempts int

	// InitialBackoff is the wait time before the first retry.
	// Defaults to 100 ms.
	InitialBackoff time.Duration

	// MaxBackoff caps the exponential growth. Defaults to 5 s.
	MaxBackoff time.Duration

	// JitterFactor is the fraction of the current backoff to add as random
	// jitter, preventing thundering-herd on concurrent retries. Range [0, 1].
	// Defaults to 0.3 (±30 % of the computed backoff).
	JitterFactor float64
}

func defaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		JitterFactor:   0.3,
	}
}

// isRetryable reports whether the attempt should be retried based on the
// response status or the transport-level error.
//
// Retry policy:
//   - Network / transport errors (nil response): always retry.
//   - HTTP 5xx (server-side transient failures): retry.
//   - HTTP 4xx, 3xx, 2xx: do NOT retry — the caller must handle these.
func isRetryable(resp *http.Response, err error) bool {
	if err != nil {
		// Transport error (e.g. connection refused, EOF, timeout).
		return true
	}
	return resp != nil && resp.StatusCode >= 500
}

// RetryableHTTPDo executes req via client, retrying on transient failures
// (network errors or HTTP 5xx responses) with exponential backoff and jitter.
//
// If cfg is nil, default values are used (3 attempts, 100 ms initial backoff,
// 5 s max backoff, 30 % jitter).
//
// The caller is responsible for closing resp.Body on a successful (non-nil)
// return, exactly as with a plain http.Client.Do call.
//
// If the context is cancelled between retries the function returns immediately
// with the context error.
//
// If every attempt fails, the error from the final attempt is returned wrapped
// with the attempt count for observability.
func RetryableHTTPDo(ctx context.Context, client *http.Client, req *http.Request, cfg *RetryConfig) (*http.Response, error) {
	c := defaultRetryConfig()
	if cfg != nil {
		if cfg.MaxAttempts > 0 {
			c.MaxAttempts = cfg.MaxAttempts
		}
		if cfg.InitialBackoff > 0 {
			c.InitialBackoff = cfg.InitialBackoff
		}
		if cfg.MaxBackoff > 0 {
			c.MaxBackoff = cfg.MaxBackoff
		}
		if cfg.JitterFactor >= 0 {
			c.JitterFactor = cfg.JitterFactor
		}
	}

	var (
		resp     *http.Response
		lastErr  error
	)

	for attempt := 1; attempt <= c.MaxAttempts; attempt++ {
		// Check context before each attempt so we bail early on cancellation.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Replay the request body for retries (the body is consumed by client.Do).
		if attempt > 1 && req.GetBody != nil {
			body, bodyErr := req.GetBody()
			if bodyErr != nil {
				return nil, fmt.Errorf("retry body replay: %w", bodyErr)
			}
			req.Body = body
		}

		resp, lastErr = client.Do(req)

		if !isRetryable(resp, lastErr) {
			// Success or a non-retryable error (4xx) — return immediately.
			return resp, lastErr
		}

		// If there is a retryable response body we must drain and close it
		// before the next attempt to avoid leaking the connection.
		if resp != nil {
			_, _ = resp.Body.Read(make([]byte, 512)) // drain up to 512 bytes
			_ = resp.Body.Close()
		}

		if attempt == c.MaxAttempts {
			// No more retries.
			break
		}

		backoff := computeBackoff(c, attempt)
		slog.WarnContext(ctx, "retryable HTTP error; backing off",
			"attempt", attempt,
			"max_attempts", c.MaxAttempts,
			"backoff_ms", backoff.Milliseconds(),
			"url", req.URL.String(),
			"error", lastErr,
			"status_code", statusCode(resp),
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}

	// All attempts exhausted.
	if lastErr != nil {
		return nil, fmt.Errorf("all %d attempts failed: %w", c.MaxAttempts, lastErr)
	}
	// Final attempt got a 5xx — return the response so the caller can inspect
	// the status code and body if needed.
	return resp, errors.New("all attempts failed with server error")
}

// computeBackoff returns the backoff duration for the given attempt number
// (1-based), applying exponential growth, a max cap, and random jitter.
func computeBackoff(c RetryConfig, attempt int) time.Duration {
	// Exponential: initialBackoff * 2^(attempt-1)
	exp := math.Pow(2, float64(attempt-1))
	base := min(time.Duration(float64(c.InitialBackoff)*exp), c.MaxBackoff)
	// Jitter: add up to JitterFactor * base of random duration.
	jitter := time.Duration(float64(base) * c.JitterFactor * rand.Float64()) //nolint:gosec // non-crypto jitter
	return base + jitter
}

// statusCode is a nil-safe helper for extracting the HTTP status code from a
// response pointer (returns 0 when resp is nil).
func statusCode(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}
