package weather

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/internal/weatherapi"
)

// retryDelays are the wait periods between the initial attempt and each retry
// for weather API calls. The initial call is made immediately, so this slice
// represents 4 retries after transient failures.
var retryDelays = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
	10 * time.Second,
}

// IsRetryable returns true if err represents a transient failure that is worth
// retrying. Context cancellation is never retried; deadline exceeded, network
// timeouts, and 5xx HTTP responses are retried.
func IsRetryable(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}

	// Never retry if the context is already done.
	if ctx.Err() != nil {
		return false
	}

	// Never retry explicit cancellation.
	if errors.Is(err, context.Canceled) {
		return false
	}

	// Retry on deadline exceeded. The caller's context is checked between
	// attempts, so a caller-level timeout will still stop the loop.
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Retry network timeouts.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Retry 5xx HTTP responses.
	var httpErr *weatherapi.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode >= http.StatusInternalServerError {
		return true
	}

	return false
}

// WithRetries executes operation with the configured retry delays. It returns
// the result of the first successful attempt or the last error if all attempts
// fail.
func WithRetries[T any](ctx context.Context, operation func(context.Context) (T, error)) (T, error) {
	var zero T

	if ctx.Err() != nil {
		return zero, ctx.Err()
	}

	result, err := operation(ctx)
	if err == nil {
		return result, nil
	}
	if !IsRetryable(ctx, err) {
		return zero, err
	}

	for _, delay := range retryDelays {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}

		result, err = operation(ctx)
		if err == nil {
			return result, nil
		}
		if !IsRetryable(ctx, err) {
			return zero, err
		}
	}

	return zero, err
}

// SetRetryDelaysForTest overrides the retry delays used by WithRetries. It
// returns a function that restores the original delays. This is intended for
// tests that need to exercise retry logic without waiting for the production
// backoff schedule.
func SetRetryDelaysForTest(delays []time.Duration) func() {
	original := retryDelays
	retryDelays = delays
	return func() {
		retryDelays = original
	}
}
