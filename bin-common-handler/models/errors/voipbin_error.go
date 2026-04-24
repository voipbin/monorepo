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
	Cause   error  `json:"-"`
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
