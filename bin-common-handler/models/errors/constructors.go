package errors

import (
	"monorepo/bin-common-handler/models/outline"
)

// newVoipbinError is the single internal factory. The exported
// constructors below pin one canonical Status each so callers never
// pass a free-form status string.
func newVoipbinError(s Status, domain outline.ServiceName, reason, message string) *VoipbinError {
	return &VoipbinError{Status: s, Domain: string(domain), Reason: reason, Message: message}
}

// InvalidArgument returns a VoipbinError with StatusInvalidArgument.
func InvalidArgument(domain outline.ServiceName, reason, message string) *VoipbinError {
	return newVoipbinError(StatusInvalidArgument, domain, reason, message)
}

// Unauthenticated returns a VoipbinError with StatusUnauthenticated.
func Unauthenticated(domain outline.ServiceName, reason, message string) *VoipbinError {
	return newVoipbinError(StatusUnauthenticated, domain, reason, message)
}

// PaymentRequired returns a VoipbinError with StatusPaymentRequired.
func PaymentRequired(domain outline.ServiceName, reason, message string) *VoipbinError {
	return newVoipbinError(StatusPaymentRequired, domain, reason, message)
}

// PermissionDenied returns a VoipbinError with StatusPermissionDenied.
func PermissionDenied(domain outline.ServiceName, reason, message string) *VoipbinError {
	return newVoipbinError(StatusPermissionDenied, domain, reason, message)
}

// NotFound returns a VoipbinError with StatusNotFound.
func NotFound(domain outline.ServiceName, reason, message string) *VoipbinError {
	return newVoipbinError(StatusNotFound, domain, reason, message)
}

// AlreadyExists returns a VoipbinError with StatusAlreadyExists.
//
// Note: the api-manager translator never emits ALREADY_EXISTS as a
// fallback mapping — 409 responses default to FAILED_PRECONDITION.
// Use this constructor explicitly at sites that genuinely represent
// a duplicate-create conflict (e.g., CreateXxx handlers hitting a
// unique-constraint violation).
func AlreadyExists(domain outline.ServiceName, reason, message string) *VoipbinError {
	return newVoipbinError(StatusAlreadyExists, domain, reason, message)
}

// FailedPrecondition returns a VoipbinError with StatusFailedPrecondition.
func FailedPrecondition(domain outline.ServiceName, reason, message string) *VoipbinError {
	return newVoipbinError(StatusFailedPrecondition, domain, reason, message)
}

// ResourceExhausted returns a VoipbinError with StatusResourceExhausted.
func ResourceExhausted(domain outline.ServiceName, reason, message string) *VoipbinError {
	return newVoipbinError(StatusResourceExhausted, domain, reason, message)
}

// Unavailable returns a VoipbinError with StatusUnavailable.
func Unavailable(domain outline.ServiceName, reason, message string) *VoipbinError {
	return newVoipbinError(StatusUnavailable, domain, reason, message)
}

// Internal returns a VoipbinError with StatusInternal.
func Internal(domain outline.ServiceName, reason, message string) *VoipbinError {
	return newVoipbinError(StatusInternal, domain, reason, message)
}
