package errors

import (
	"encoding/json"
	"testing"

	"monorepo/bin-common-handler/models/sock"
)

func TestFromResponseTypedError(t *testing.T) {
	payload := &VoipbinError{
		Status:  StatusNotFound,
		Reason:  "CALL_NOT_FOUND",
		Domain:  "call-manager",
		Message: "The call was not found.",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal setup: %v", err)
	}
	resp := &sock.Response{
		StatusCode: 404,
		DataType:   DataTypeVoipbinError,
		Data:       data,
	}

	got := FromResponse(resp)
	if got == nil {
		t.Fatal("FromResponse returned nil for a typed error response")
	}
	if got.Status != StatusNotFound || got.Reason != "CALL_NOT_FOUND" {
		t.Errorf("wrong VoipbinError: %+v", got)
	}
}

func TestFromResponseSuccess(t *testing.T) {
	resp := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       json.RawMessage(`{"ok":true}`),
	}
	if got := FromResponse(resp); got != nil {
		t.Errorf("FromResponse on success should return nil, got %+v", got)
	}
}

func TestFromResponseErrorWithoutTypedDataType(t *testing.T) {
	// Legacy manager — error code but no typed body.
	resp := &sock.Response{
		StatusCode: 500,
		DataType:   "application/json",
		Data:       json.RawMessage(`{"message":"legacy"}`),
	}
	if got := FromResponse(resp); got != nil {
		t.Errorf("FromResponse without DataTypeVoipbinError must return nil, got %+v", got)
	}
}

func TestFromResponseMalformedData(t *testing.T) {
	resp := &sock.Response{
		StatusCode: 500,
		DataType:   DataTypeVoipbinError,
		Data:       json.RawMessage(`{not json`),
	}
	// Must not panic and must fall back to nil so the caller uses its own fallback path.
	if got := FromResponse(resp); got != nil {
		t.Errorf("malformed Data must return nil, got %+v", got)
	}
}

func TestFromResponseNil(t *testing.T) {
	if got := FromResponse(nil); got != nil {
		t.Errorf("nil response must return nil, got %+v", got)
	}
}
