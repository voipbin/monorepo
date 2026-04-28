package errors

import "fmt"

// VoipbinError is the canonical error shape used over RPC between
// internal managers and (via the api-manager boundary) for external
// HTTP API error responses.
//
// Wire contract:
//   - Status, Reason, Message are emitted to external HTTP clients
//     AND to internal RPC consumers.
//   - Domain is emitted to internal RPC consumers (so a downstream
//     manager knows which upstream emitted the typed error) but is
//     INTENTIONALLY STRIPPED from external HTTP responses by
//     bin-api-manager/lib/apierror.EnvelopeFor — external clients
//     must not see the originating internal service name. The field
//     stays on the struct (and in this RPC wire format) so internal
//     callers and server-side logs can still observe it.
//   - Details is reserved for structured detail; omitempty. Preserved
//     across both internal RPC and external HTTP paths.
//   - Cause is server-side only (json:"-"); it is included by Error()
//     for server logs but MUST NOT be written into HTTP response
//     bodies. Call json.Marshal(e) or construct a response envelope
//     from fields directly — do not use e.Error() as a user-visible
//     message.
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
// the given error. Callers can chain: Internal(...).Wrap(err).
//
// Wrap does NOT mutate the receiver's Cause field — string, Status,
// Reason, Domain, and Message are independently copied. However, the
// Details slice is a shallow copy: mutating an existing Details entry
// on the returned value will also mutate the receiver's same entry
// (they share the backing array). In practice this is never an issue
// because Wrap is called immediately after construction, before
// Details is populated — but do not mutate Details entries after
// calling Wrap if you intend to reuse the receiver.
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
