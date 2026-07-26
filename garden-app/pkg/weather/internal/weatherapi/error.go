// Package weatherapi provides shared types for weather API clients.
package weatherapi

import "fmt"

// HTTPError represents a non-2xx HTTP response from a weather API. It is used by
// client implementations and the retry logic to decide whether a failure is
// transient and should be retried.
type HTTPError struct {
	StatusCode int
	Body       string
}

// Error returns a string representation of the HTTPError.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("weather API returned status %d: %s", e.StatusCode, e.Body)
}
