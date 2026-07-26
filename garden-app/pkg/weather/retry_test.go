package weather

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/internal/weatherapi"
	"github.com/stretchr/testify/assert"
)

func TestWithRetries_SuccessFirstAttempt(t *testing.T) {
	restore := SetRetryDelaysForTest([]time.Duration{0, 0, 0, 0})
	defer restore()

	calls := 0
	result, err := WithRetries(context.Background(), func(ctx context.Context) (int, error) {
		calls++
		return 42, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 1, calls)
}

func TestWithRetries_SuccessAfterFailures(t *testing.T) {
	restore := SetRetryDelaysForTest([]time.Duration{0, 0, 0, 0})
	defer restore()

	calls := 0
	result, err := WithRetries(context.Background(), func(ctx context.Context) (int, error) {
		calls++
		if calls < 3 {
			return 0, &weatherapi.HTTPError{StatusCode: http.StatusServiceUnavailable, Body: "unavailable"}
		}
		return 42, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 3, calls)
}

func TestWithRetries_NonRetryableHTTPError(t *testing.T) {
	restore := SetRetryDelaysForTest([]time.Duration{0, 0, 0, 0})
	defer restore()

	calls := 0
	_, err := WithRetries(context.Background(), func(ctx context.Context) (int, error) {
		calls++
		return 0, &weatherapi.HTTPError{StatusCode: http.StatusBadRequest, Body: "bad request"}
	})

	assert.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestWithRetries_MaxRetriesExceeded(t *testing.T) {
	restore := SetRetryDelaysForTest([]time.Duration{0, 0, 0, 0})
	defer restore()

	calls := 0
	_, err := WithRetries(context.Background(), func(ctx context.Context) (int, error) {
		calls++
		return 0, &weatherapi.HTTPError{StatusCode: http.StatusServiceUnavailable, Body: "unavailable"}
	})

	assert.Error(t, err)
	assert.Equal(t, 5, calls) // initial + 4 retries
}

func TestWithRetries_ContextCancelled(t *testing.T) {
	restore := SetRetryDelaysForTest([]time.Duration{time.Hour, time.Hour, time.Hour, time.Hour})
	defer restore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	_, err := WithRetries(ctx, func(ctx context.Context) (int, error) {
		calls++
		return 0, errors.New("transient")
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, calls)
}

func TestWithRetries_ContextDeadline(t *testing.T) {
	restore := SetRetryDelaysForTest([]time.Duration{time.Hour, time.Hour, time.Hour, time.Hour})
	defer restore()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	// Wait for the deadline to expire.
	time.Sleep(5 * time.Millisecond)

	calls := 0
	_, err := WithRetries(ctx, func(ctx context.Context) (int, error) {
		calls++
		return 0, context.DeadlineExceeded
	})

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 0, calls)
}

func TestWithRetries_NetworkError(t *testing.T) {
	restore := SetRetryDelaysForTest([]time.Duration{0, 0, 0, 0})
	defer restore()

	calls := 0
	_, err := WithRetries(context.Background(), func(ctx context.Context) (int, error) {
		calls++
		return 0, &net.DNSError{Err: "timeout", IsTimeout: true}
	})

	assert.Error(t, err)
	assert.Equal(t, 5, calls)
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			ctx:       context.Background(),
			err:       nil,
			retryable: false,
		},
		{
			name:      "context canceled",
			ctx:       context.Background(),
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "context deadline exceeded",
			ctx:       context.Background(),
			err:       context.DeadlineExceeded,
			retryable: true,
		},
		{
			name:      "5xx HTTP error",
			ctx:       context.Background(),
			err:       &weatherapi.HTTPError{StatusCode: http.StatusServiceUnavailable},
			retryable: true,
		},
		{
			name:      "4xx HTTP error",
			ctx:       context.Background(),
			err:       &weatherapi.HTTPError{StatusCode: http.StatusBadRequest},
			retryable: false,
		},
		{
			name:      "wrapped 5xx HTTP error",
			ctx:       context.Background(),
			err:       errors.New("wrapped: " + (&weatherapi.HTTPError{StatusCode: http.StatusGatewayTimeout}).Error()),
			retryable: false, // string wrapping breaks errors.As
		},
		{
			name:      "wrapped 5xx with fmt.Errorf %w",
			ctx:       context.Background(),
			err:       fmt.Errorf("wrapped: %w", &weatherapi.HTTPError{StatusCode: http.StatusGatewayTimeout}),
			retryable: true,
		},
		{
			name:      "cancelled context error",
			ctx:       func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			err:       errors.New("transient"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.retryable, IsRetryable(tt.ctx, tt.err))
		})
	}
}
