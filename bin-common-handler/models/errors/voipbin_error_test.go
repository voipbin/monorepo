package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"
	"testing"
)

func TestVoipbinErrorFields(t *testing.T) {
	e := &VoipbinError{
		Status:  StatusNotFound,
		Reason:  "CALL_NOT_FOUND",
		Domain:  "call-manager",
		Message: "The call was not found.",
		Cause:   stderrors.New("underlying"),
	}
	if e.Status != StatusNotFound {
		t.Errorf("wrong Status: %v", e.Status)
	}
	if e.Reason != "CALL_NOT_FOUND" {
		t.Errorf("wrong Reason: %v", e.Reason)
	}
	if e.Domain != "call-manager" {
		t.Errorf("wrong Domain: %v", e.Domain)
	}
	if e.Message != "The call was not found." {
		t.Errorf("wrong Message: %v", e.Message)
	}
	if e.Cause == nil || e.Cause.Error() != "underlying" {
		t.Errorf("wrong Cause: %v", e.Cause)
	}
}

func TestVoipbinErrorJSONExcludesCause(t *testing.T) {
	e := &VoipbinError{
		Status:  StatusNotFound,
		Reason:  "CALL_NOT_FOUND",
		Domain:  "call-manager",
		Message: "The call was not found.",
		Cause:   stderrors.New("DB driver error: connection refused"),
	}

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	s := string(b)
	for _, want := range []string{
		`"status":"NOT_FOUND"`,
		`"reason":"CALL_NOT_FOUND"`,
		`"domain":"call-manager"`,
		`"message":"The call was not found."`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %q in JSON %q", want, s)
		}
	}
	for _, forbidden := range []string{"cause", "connection refused", "DB driver"} {
		if strings.Contains(s, forbidden) {
			t.Errorf("forbidden token %q leaked into JSON %q", forbidden, s)
		}
	}
}

func TestVoipbinErrorErrorMethod(t *testing.T) {
	e := &VoipbinError{
		Status:  StatusPermissionDenied,
		Reason:  "BILLING_ACCESS_DENIED",
		Domain:  "billing-manager",
		Message: "Not allowed.",
	}
	got := e.Error()
	want := "billing-manager: BILLING_ACCESS_DENIED: Not allowed."
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestVoipbinErrorErrorMethodWithCause(t *testing.T) {
	e := &VoipbinError{
		Status:  StatusInternal,
		Reason:  "INTERNAL",
		Domain:  "api-manager",
		Message: "Something went wrong.",
		Cause:   stderrors.New("pq: connection refused"),
	}
	got := e.Error()
	want := "api-manager: INTERNAL: Something went wrong.: pq: connection refused"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestVoipbinErrorErrorMethodNilReceiver(t *testing.T) {
	var e *VoipbinError
	got := e.Error()
	want := "<nil VoipbinError>"
	if got != want {
		t.Errorf("Error() on nil = %q, want %q", got, want)
	}
}

func TestVoipbinErrorUnwrap(t *testing.T) {
	inner := stderrors.New("inner failure")
	e := &VoipbinError{
		Status:  StatusInternal,
		Reason:  "INTERNAL",
		Domain:  "api-manager",
		Message: "wrap test",
		Cause:   inner,
	}

	// errors.Is walks the chain via Unwrap.
	if !stderrors.Is(e, inner) {
		t.Fatalf("errors.Is did not find wrapped cause")
	}

	// errors.As on a wrapped VoipbinError finds it.
	wrapped := fmt.Errorf("context: %w", e)
	var target *VoipbinError
	if !stderrors.As(wrapped, &target) {
		t.Fatalf("errors.As did not recover VoipbinError from wrapped chain")
	}
	if target.Reason != "INTERNAL" {
		t.Errorf("errors.As returned wrong VoipbinError: %v", target)
	}
}

func TestVoipbinErrorWrap(t *testing.T) {
	e := &VoipbinError{Status: StatusInternal, Reason: "INTERNAL", Domain: "api-manager", Message: "x"}
	inner := stderrors.New("boom")
	out := e.Wrap(inner)

	if out != e {
		t.Errorf("Wrap should return the receiver for chaining")
	}
	if e.Cause != inner {
		t.Errorf("Wrap did not set Cause: %v", e.Cause)
	}
}

func TestVoipbinErrorWrapNilReceiver(t *testing.T) {
	var e *VoipbinError
	got := e.Wrap(stderrors.New("x"))
	if got != nil {
		t.Errorf("Wrap on nil receiver = %v, want nil", got)
	}
}
