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
		{"identity_verification_required", ErrIdentityVerificationRequired, "identity verification required"},
		{"state_invalid", ErrStateInvalid, "state invalid"},
		{"service_unavailable", ErrServiceUnavailable, "service unavailable"},
		{"insufficient_balance", ErrInsufficientBalance, "insufficient balance"},
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
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrPermissionDenied", ErrPermissionDenied},
		{"ErrNotFound", ErrNotFound},
		{"ErrAuthenticationRequired", ErrAuthenticationRequired},
		{"ErrDirectAccessNotSupported", ErrDirectAccessNotSupported},
		{"ErrInvalidArgument", ErrInvalidArgument},
		{"ErrInternal", ErrInternal},
		{"ErrIdentityVerificationRequired", ErrIdentityVerificationRequired},
		{"ErrStateInvalid", ErrStateInvalid},
		{"ErrServiceUnavailable", ErrServiceUnavailable},
		{"ErrInsufficientBalance", ErrInsufficientBalance},
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i == j {
				continue
			}
			if stderrors.Is(a.err, b.err) {
				t.Errorf("%s must not match %s", a.name, b.name)
			}
		}
	}
}
