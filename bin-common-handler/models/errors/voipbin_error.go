package errors

import "fmt"

// VoipbinError is the canonical error shape returned from the external
// VoIPbin API and (eventually) over RPC between internal managers.
// The Cause field is for server-side logging only and is never
// serialized to clients.
type VoipbinError struct {
	Status  Status `json:"status"`
	Reason  string `json:"reason"`
	Domain  string `json:"domain"`
	Message string `json:"message"`
	// Details is reserved for future per-field or structured error
	// detail (e.g., BadRequest field violations). Optional; may be
	// omitted. New detail shapes must be additive-only so existing
	// consumers tolerate unknown keys.
	Details []map[string]any `json:"details,omitempty"`
	// Cause is for server-side logging only — never serialized to
	// clients. WARNING: Error() DOES include Cause.Error() in its
	// return value. Never use err.Error() as an HTTP response body.
	Cause error `json:"-"`
}

// Error satisfies the error interface. Format:
//
//	<domain>: <reason>: <message>[: <cause>]
//
// The cause is included when non-nil so server-side logs see the
// underlying error without any extra call at the log site.
func (e *VoipbinError) Error() string {
	if e == nil {
		return "<nil VoipbinError>"
	}
	base := fmt.Sprintf("%s: %s: %s", e.Domain, e.Reason, e.Message)
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s", base, e.Cause.Error())
	}
	return base
}

// Unwrap returns the underlying cause so errors.Is and errors.As
// can walk the chain across fmt.Errorf("%w", VoipbinError) wraps.
func (e *VoipbinError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Wrap returns a shallow copy of the VoipbinError with Cause set to
// the given error. Callers can chain: Internal(...).Wrap(err). Wrap
// does NOT mutate the receiver — this prevents aliasing surprises if
// the receiver has been stored in a package-level variable or passed
// to another goroutine.
//
// The cause is only used in server-side logs (Error() includes it);
// JSON serialization still excludes it via the json:"-" tag.
func (e *VoipbinError) Wrap(cause error) *VoipbinError {
	if e == nil {
		return nil
	}
	out := *e
	out.Cause = cause
	return &out
}
