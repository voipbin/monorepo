package requestexternal

import "fmt"

// HTTPStatusError is returned when the external service answers with an unexpected HTTP status.
// It lets the callers tell a refused request (4xx) from a failure whose outcome is unknown.
type HTTPStatusError struct {
	StatusCode int

	// ErrorCode is the error code of the provider in the response body (e.g. the twilio "code").
	// It is zero when the provider sent none. It is for logs only; callers classify by StatusCode.
	ErrorCode int
}

// Error implements the error interface.
func (e *HTTPStatusError) Error() string {
	if e.ErrorCode != 0 {
		return fmt.Sprintf("could not get correct response. status: %d, code: %d", e.StatusCode, e.ErrorCode)
	}
	return fmt.Sprintf("could not get correct response. status: %d", e.StatusCode)
}
