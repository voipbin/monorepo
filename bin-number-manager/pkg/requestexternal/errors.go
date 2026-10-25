package requestexternal

import "fmt"

// HTTPStatusError is returned when the external service answers with an unexpected HTTP status.
// It lets the callers tell a refused request (4xx) from a failure whose outcome is unknown.
type HTTPStatusError struct {
	StatusCode int
}

// Error implements the error interface.
func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("could not get correct response. status: %d", e.StatusCode)
}
