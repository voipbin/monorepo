package errors

import (
	"encoding/json"
	stderrors "errors"
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
