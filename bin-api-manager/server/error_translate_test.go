package server

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
)

func TestTranslateTypedPassthrough(t *testing.T) {
	in := cerrors.NotFound("call-manager", "CALL_NOT_FOUND", "x")
	out := translateToVoipbinError(in)
	if out != in {
		t.Errorf("typed error should pass through, got %+v", out)
	}
}

func TestTranslateWrappedTypedError(t *testing.T) {
	in := cerrors.NotFound("call-manager", "CALL_NOT_FOUND", "x")
	wrapped := fmt.Errorf("context: %w", in)
	out := translateToVoipbinError(wrapped)
	if out != in {
		t.Errorf("wrapped typed error should unwrap to original, got %+v", out)
	}
}

func TestTranslateSentinels(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus cerrors.Status
		wantReason string
	}{
		{"permission_denied", serviceerrors.ErrPermissionDenied, cerrors.StatusPermissionDenied, "PERMISSION_DENIED"},
		{"not_found", serviceerrors.ErrNotFound, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND"},
		{"auth_required", serviceerrors.ErrAuthenticationRequired, cerrors.StatusUnauthenticated, "AUTHENTICATION_REQUIRED"},
		{"direct_access", serviceerrors.ErrDirectAccessNotSupported, cerrors.StatusPermissionDenied, "DIRECT_ACCESS_NOT_SUPPORTED"},
		{"invalid_argument", serviceerrors.ErrInvalidArgument, cerrors.StatusInvalidArgument, "INVALID_ARGUMENT"},
		{"internal", serviceerrors.ErrInternal, cerrors.StatusInternal, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateToVoipbinError(tt.err)
			if got.Status != tt.wantStatus || got.Reason != tt.wantReason {
				t.Errorf("got status=%q reason=%q want %q/%q", got.Status, got.Reason, tt.wantStatus, tt.wantReason)
			}
		})
	}
}

func TestTranslateSentinelErrInternalWrapsCause(t *testing.T) {
	got := translateToVoipbinError(serviceerrors.ErrInternal)
	if got.Status != cerrors.StatusInternal {
		t.Errorf("wrong status: %q", got.Status)
	}
	if got.Cause == nil {
		t.Error("Cause should be set (wraps the sentinel)")
	}
	if !stderrors.Is(got, serviceerrors.ErrInternal) {
		t.Error("errors.Is chain should still find the sentinel")
	}
}

func TestTranslateTransportErrors(t *testing.T) {
	if out := translateToVoipbinError(context.DeadlineExceeded); out.Status != cerrors.StatusUnavailable {
		t.Errorf("DeadlineExceeded should map to UNAVAILABLE, got %+v", out)
	}
	if out := translateToVoipbinError(context.Canceled); out.Status != cerrors.StatusUnavailable {
		t.Errorf("Canceled should map to UNAVAILABLE, got %+v", out)
	}
}

func TestTranslateSubstringFallback(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus cerrors.Status
	}{
		{stderrors.New("user has no permission"), cerrors.StatusPermissionDenied},
		{stderrors.New("agent has no permission"), cerrors.StatusPermissionDenied},
		{stderrors.New("permission denied"), cerrors.StatusPermissionDenied},
		{stderrors.New("Forbidden"), cerrors.StatusPermissionDenied},
		{stderrors.New("direct access not supported"), cerrors.StatusPermissionDenied},
		{stderrors.New("the number does not belong to this customer"), cerrors.StatusPermissionDenied},
		{stderrors.New("agent authentication required"), cerrors.StatusUnauthenticated},
		{stderrors.New("Unauthorized"), cerrors.StatusUnauthenticated},
		{stderrors.New("call not found"), cerrors.StatusNotFound},
		{stderrors.New("Not Found"), cerrors.StatusNotFound},
		{stderrors.New("upstream service unavailable"), cerrors.StatusUnavailable},
		{stderrors.New("Service Unavailable"), cerrors.StatusUnavailable},
		{stderrors.New("call already hangup"), cerrors.StatusFailedPrecondition},
		{stderrors.New("recording already active"), cerrors.StatusFailedPrecondition},
		{stderrors.New("recording not active"), cerrors.StatusFailedPrecondition},
		{stderrors.New("deleted call"), cerrors.StatusFailedPrecondition},
		{stderrors.New("Insufficient balance"), cerrors.StatusPaymentRequired},
		{stderrors.New("insufficient funds for this operation"), cerrors.StatusPaymentRequired},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			got := translateToVoipbinError(tt.err)
			if got.Status != tt.wantStatus {
				t.Errorf("got %q want %q", got.Status, tt.wantStatus)
			}
		})
	}
}

// TestTranslateIdentityVerificationRequired verifies that error messages
// containing "identity verification required" map to PERMISSION_DENIED with
// the distinct reason IDENTITY_VERIFICATION_REQUIRED — not to the generic
// PERMISSION_DENIED reason. The pattern lands in PR 4 to surface number-
// purchase identity gates as user-actionable 403s instead of opaque 500s.
func TestTranslateIdentityVerificationRequired(t *testing.T) {
	tests := []error{
		stderrors.New("customer identity verification required for number purchase"),
		stderrors.New("identity verification required"),
	}
	for _, in := range tests {
		t.Run(in.Error(), func(t *testing.T) {
			got := translateToVoipbinError(in)
			if got.Status != cerrors.StatusPermissionDenied {
				t.Errorf("got status %q want %q", got.Status, cerrors.StatusPermissionDenied)
			}
			if got.Reason != "IDENTITY_VERIFICATION_REQUIRED" {
				t.Errorf("got reason %q want IDENTITY_VERIFICATION_REQUIRED", got.Reason)
			}
		})
	}
}

func TestTranslateDefault(t *testing.T) {
	orig := stderrors.New("something nobody anticipated")
	got := translateToVoipbinError(orig)
	if got.Status != cerrors.StatusInternal {
		t.Errorf("unknown error should map to INTERNAL, got %q", got.Status)
	}
	if got.Cause != orig {
		t.Errorf("Cause should wrap original error, got %v", got.Cause)
	}
}

func TestTranslateNil(t *testing.T) {
	got := translateToVoipbinError(nil)
	if got == nil {
		t.Fatal("translator must never return nil")
	}
	if got.Status != cerrors.StatusInternal {
		t.Errorf("nil error should map to INTERNAL, got %q", got.Status)
	}
}

func TestTranslatePanicRecovery(t *testing.T) {
	got := translateToVoipbinError(panickingError{})
	if got == nil {
		t.Fatal("translator must never return nil even on panic")
	}
	if got.Status != cerrors.StatusInternal {
		t.Errorf("panic path must degrade to INTERNAL, got %q", got.Status)
	}
}

type panickingError struct{}

func (panickingError) Error() string { panic("boom") }
