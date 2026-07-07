// Package serviceerrors defines sentinel errors used by servicehandler
// methods in bin-api-manager. The translator in server/error_translate.go
// matches these sentinels (via errors.Is) and converts them into
// VoipbinError responses for the client.
//
// This layer is intentionally thin: servicehandler code can choose to
// construct a richer *cerrors.VoipbinError directly (preferred when the
// site has context like a resource ID), and fall back to a sentinel
// only when there is no specific context to add.
package serviceerrors

import stderrors "errors"

var (
	ErrPermissionDenied             = stderrors.New("permission denied")
	ErrNotFound                     = stderrors.New("not found")
	ErrAuthenticationRequired       = stderrors.New("authentication required")
	ErrDirectAccessNotSupported     = stderrors.New("direct access not supported")
	ErrInvalidArgument              = stderrors.New("invalid argument")
	ErrInternal                     = stderrors.New("internal error")
	ErrIdentityVerificationRequired = stderrors.New("identity verification required")
	ErrStateInvalid                 = stderrors.New("state invalid")
	ErrServiceUnavailable           = stderrors.New("service unavailable")
	ErrInsufficientBalance          = stderrors.New("insufficient balance")

	// ErrCaseClosed is returned by Case-message-send validation (design
	// §4.5 step 1) when the target case is not status='open'. Points the
	// caller at POST /v1.0/cases/{id}/continue.
	ErrCaseClosed = stderrors.New("case is closed; call continue to reopen it before sending a message")

	// ErrCaseDestinationNotAssociated is the SINGLE GENERIC error for
	// design §4.5 step 2's destination-to-case binding check. It MUST be
	// returned identically (same sentinel, same message) regardless of
	// which of the two binding sub-checks failed (contact address list
	// miss vs. peer_target mismatch) -- this is the anti-oracle property
	// that prevents a caller from probing which branch failed. Do not
	// introduce a second, more specific error for either sub-case.
	ErrCaseDestinationNotAssociated = stderrors.New("destination is not associated with this case")

	// ErrCaseSourceNotOwned is returned by design §4.5 step 3 (round-17
	// correction) when the given source number is not an active, normal
	// number owned by the case's customer.
	ErrCaseSourceNotOwned = stderrors.New("source is not an active number owned by this customer")
)
