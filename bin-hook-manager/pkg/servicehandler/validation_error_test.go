package servicehandler

import (
	"errors"
	"testing"
)

func Test_ValidationError(t *testing.T) {
	inner := errors.New("signature mismatch")
	var err error = &ValidationError{Err: inner}

	if err.Error() != "signature mismatch" {
		t.Errorf("Wrong match. expect: signature mismatch, got: %s", err.Error())
	}

	if !errors.Is(err, inner) {
		t.Errorf("errors.Is must reach the wrapped error through Unwrap")
	}

	wrapped := errors.Join(errors.New("context"), err)
	var ve *ValidationError
	if !errors.As(wrapped, &ve) {
		t.Fatalf("errors.As must find ValidationError in the chain")
	}
	if ve.Err != inner {
		t.Errorf("Wrong match. expect: %v, got: %v", inner, ve.Err)
	}

	if errors.Is(errors.New("other"), inner) {
		t.Errorf("an unrelated error must not match")
	}
}
