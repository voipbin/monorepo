package errors

import "testing"

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"invalid_argument", StatusInvalidArgument, "INVALID_ARGUMENT"},
		{"unauthenticated", StatusUnauthenticated, "UNAUTHENTICATED"},
		{"payment_required", StatusPaymentRequired, "PAYMENT_REQUIRED"},
		{"permission_denied", StatusPermissionDenied, "PERMISSION_DENIED"},
		{"not_found", StatusNotFound, "NOT_FOUND"},
		{"already_exists", StatusAlreadyExists, "ALREADY_EXISTS"},
		{"failed_precondition", StatusFailedPrecondition, "FAILED_PRECONDITION"},
		{"resource_exhausted", StatusResourceExhausted, "RESOURCE_EXHAUSTED"},
		{"unavailable", StatusUnavailable, "UNAVAILABLE"},
		{"internal", StatusInternal, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("got %q want %q", string(tt.constant), tt.expected)
			}
		})
	}
}
