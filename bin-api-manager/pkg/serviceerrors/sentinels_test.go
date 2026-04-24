package serviceerrors

import (
	stderrors "errors"
	"testing"
)

func TestSentinelsExist(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"permission_denied", ErrPermissionDenied, "permission denied"},
		{"not_found", ErrNotFound, "not found"},
		{"authentication_required", ErrAuthenticationRequired, "authentication required"},
		{"direct_access_not_supported", ErrDirectAccessNotSupported, "direct access not supported"},
		{"invalid_argument", ErrInvalidArgument, "invalid argument"},
		{"internal_error", ErrInternal, "internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("sentinel is nil")
			}
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("got %q want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestSentinelsAreDistinct(t *testing.T) {
	if stderrors.Is(ErrPermissionDenied, ErrNotFound) {
		t.Error("ErrPermissionDenied must not match ErrNotFound")
	}
	if stderrors.Is(ErrNotFound, ErrPermissionDenied) {
		t.Error("ErrNotFound must not match ErrPermissionDenied")
	}
}
