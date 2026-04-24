// Package errors defines the shared VoipbinError type used across the
// VoIPbin monorepo for external API error responses.
package errors

// Status is the canonical error status. It maps 1:1 to an HTTP status
// code (see bin-api-manager for the mapping). The set is intentionally
// closed — extending it requires a coordinated schema update.
type Status string

const (
	StatusInvalidArgument    Status = "INVALID_ARGUMENT"
	StatusUnauthenticated    Status = "UNAUTHENTICATED"
	StatusPaymentRequired    Status = "PAYMENT_REQUIRED"
	StatusPermissionDenied   Status = "PERMISSION_DENIED"
	StatusNotFound           Status = "NOT_FOUND"
	StatusAlreadyExists      Status = "ALREADY_EXISTS"
	StatusFailedPrecondition Status = "FAILED_PRECONDITION"
	StatusResourceExhausted  Status = "RESOURCE_EXHAUSTED"
	StatusUnavailable        Status = "UNAVAILABLE"
	StatusInternal           Status = "INTERNAL"
)
