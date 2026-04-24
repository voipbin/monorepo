package errors

// newVoipbinError is the single internal factory. The exported
// constructors below pin one canonical Status each so callers never
// pass a free-form status string.
func newVoipbinError(s Status, domain, reason, message string) *VoipbinError {
	return &VoipbinError{Status: s, Domain: domain, Reason: reason, Message: message}
}

// InvalidArgument returns a VoipbinError with StatusInvalidArgument.
func InvalidArgument(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusInvalidArgument, domain, reason, message)
}

// Unauthenticated returns a VoipbinError with StatusUnauthenticated.
func Unauthenticated(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusUnauthenticated, domain, reason, message)
}

// PaymentRequired returns a VoipbinError with StatusPaymentRequired.
func PaymentRequired(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusPaymentRequired, domain, reason, message)
}

// PermissionDenied returns a VoipbinError with StatusPermissionDenied.
func PermissionDenied(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusPermissionDenied, domain, reason, message)
}

// NotFound returns a VoipbinError with StatusNotFound.
func NotFound(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusNotFound, domain, reason, message)
}

// AlreadyExists returns a VoipbinError with StatusAlreadyExists.
func AlreadyExists(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusAlreadyExists, domain, reason, message)
}

// FailedPrecondition returns a VoipbinError with StatusFailedPrecondition.
func FailedPrecondition(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusFailedPrecondition, domain, reason, message)
}

// ResourceExhausted returns a VoipbinError with StatusResourceExhausted.
func ResourceExhausted(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusResourceExhausted, domain, reason, message)
}

// Unavailable returns a VoipbinError with StatusUnavailable.
func Unavailable(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusUnavailable, domain, reason, message)
}

// Internal returns a VoipbinError with StatusInternal.
func Internal(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusInternal, domain, reason, message)
}
