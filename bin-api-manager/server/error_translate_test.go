package server

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/requesthandler"

	pkgerrors "github.com/pkg/errors"
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

// TestTranslatePkgErrorsWrappedTypedError verifies that the pkg/errors v0.9+
// wrapping path (used widely by api-manager's servicehandler) preserves the
// upstream typed *cerrors.VoipbinError through `errors.As`.
//
// This is the production wrapping pattern — e.g., `serviceHandler.callGet`
// does `errors.Wrapf(err, "could not get the call info")` over the typed
// VoipbinError that bin-call-manager emits via `cerrors.NotFound("call-manager",
// "CALL_NOT_FOUND", ...)`. Without this passthrough, every upstream typed
// reason would be flattened to the api-manager generic ones.
//
// pkg/errors v0.9.0+ implements Unwrap() on its wrapper types, so the stdlib
// errors.As walks the chain transparently. This test guards against an
// accidental dependency downgrade or chain-breaking refactor.
func TestTranslatePkgErrorsWrappedTypedError(t *testing.T) {
	in := cerrors.NotFound("call-manager", "CALL_NOT_FOUND", "The call was not found.")
	wrapped := pkgerrors.Wrapf(in, "could not get the call info")
	out := translateToVoipbinError(wrapped)
	if out != in {
		t.Errorf("pkg/errors-wrapped typed error should unwrap to original, got %+v", out)
	}
	if out.Reason != "CALL_NOT_FOUND" {
		t.Errorf("upstream domain reason lost; got %q want CALL_NOT_FOUND", out.Reason)
	}
	if out.Domain != "call-manager" {
		t.Errorf("upstream domain lost; got %q want call-manager", out.Domain)
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
		{"identity_verification_required", serviceerrors.ErrIdentityVerificationRequired, cerrors.StatusPermissionDenied, "IDENTITY_VERIFICATION_REQUIRED"},
		{"state_invalid", serviceerrors.ErrStateInvalid, cerrors.StatusFailedPrecondition, "STATE_INVALID"},
		{"service_unavailable", serviceerrors.ErrServiceUnavailable, cerrors.StatusUnavailable, "SERVICE_UNAVAILABLE"},
		{"insufficient_balance", serviceerrors.ErrInsufficientBalance, cerrors.StatusPaymentRequired, "INSUFFICIENT_BALANCE"},
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

// TestTranslateLegacyStringFallsThroughToInternal verifies that bare
// fmt.Errorf strings (which the previous translator caught via section 4
// substring fallback) now correctly degrade to INTERNAL. Servicehandler
// must emit typed sentinels for any meaningful status code; unmatched
// legacy strings represent a bug to be fixed at the emission site.
func TestTranslateLegacyStringFallsThroughToInternal(t *testing.T) {
	legacy := []string{
		"user has no permission",
		"call not found",
		"agent already hangup",
		"insufficient balance",
		"upstream service unavailable",
	}
	for _, msg := range legacy {
		t.Run(msg, func(t *testing.T) {
			got := translateToVoipbinError(stderrors.New(msg))
			if got.Status != cerrors.StatusInternal {
				t.Errorf("legacy bare string should fall through to INTERNAL, got %q for %q", got.Status, msg)
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

// TestTranslateBareStatusSentinels verifies that a bare requesthandler
// HTTP-status sentinel (produced when a backend returns a bare
// simpleResponse(<code>) with no typed VoipbinError body) is translated to
// the matching cerrors status/reason, mirroring cerrors.HTTPStatusFor.
// This is the regression guard for issue #953.
func TestTranslateBareStatusSentinels(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus cerrors.Status
		wantReason string
		wantHTTP   int
	}{
		{"bad_request", requesthandler.ErrBadRequest, cerrors.StatusInvalidArgument, "INVALID_ARGUMENT", 400},
		{"unauthorized", requesthandler.ErrUnauthorized, cerrors.StatusUnauthenticated, "AUTHENTICATION_REQUIRED", 401},
		{"payment_required", requesthandler.ErrPaymentRequired, cerrors.StatusPaymentRequired, "INSUFFICIENT_BALANCE", 402},
		{"forbidden", requesthandler.ErrForbidden, cerrors.StatusPermissionDenied, "PERMISSION_DENIED", 403},
		{"not_found", requesthandler.ErrNotFound, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND", 404},
		{"conflict", requesthandler.ErrConflict, cerrors.StatusFailedPrecondition, "STATE_INVALID", 409},
		{"too_many_requests", requesthandler.ErrTooManyRequests, cerrors.StatusResourceExhausted, "RATE_LIMIT_EXCEEDED", 429},
		{"service_unavailable", requesthandler.ErrServiceUnavailable, cerrors.StatusUnavailable, "SERVICE_UNAVAILABLE", 503},
		{"internal", requesthandler.ErrInternal, cerrors.StatusInternal, "INTERNAL", 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateToVoipbinError(tt.err)
			if got.Status != tt.wantStatus || got.Reason != tt.wantReason {
				t.Errorf("got status=%q reason=%q want %q/%q", got.Status, got.Reason, tt.wantStatus, tt.wantReason)
			}
			// HTTPStatusFor must round-trip the resulting status back to the
			// exact HTTP code the backend originally emitted.
			if gotHTTP := cerrors.HTTPStatusFor(got.Status); gotHTTP != tt.wantHTTP {
				t.Errorf("HTTPStatusFor(%q) = %d want %d", got.Status, gotHTTP, tt.wantHTTP)
			}
		})
	}
}

// TestTranslateBareNotFoundWrapped verifies the production wrapping path:
// servicehandler wraps the bare-404 sentinel with pkg/errors.Wrapf (e.g.
// serviceHandler.aipromptproposalGet does errors.Wrapf(err, "could not get
// ai prompt proposal info")). The sentinel must still be recovered through
// the wrap.
func TestTranslateBareNotFoundWrapped(t *testing.T) {
	wrapped := pkgerrors.Wrapf(requesthandler.ErrNotFound, "could not get ai prompt proposal info")
	got := translateToVoipbinError(wrapped)
	if got.Status != cerrors.StatusNotFound {
		t.Errorf("wrapped bare 404 should map to NOT_FOUND, got %q", got.Status)
	}
	if got.Reason != "RESOURCE_NOT_FOUND" {
		t.Errorf("reason = %q want RESOURCE_NOT_FOUND", got.Reason)
	}
}
