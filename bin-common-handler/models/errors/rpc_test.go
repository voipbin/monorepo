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

func TestToResponse(t *testing.T) {
	e := NotFound("call-manager", "CALL_NOT_FOUND", "The call was not found.")
	resp, err := ToResponse(e)
	if err != nil {
		t.Fatalf("ToResponse returned error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("wrong StatusCode: %d", resp.StatusCode)
	}
	if resp.DataType != DataTypeVoipbinError {
		t.Errorf("wrong DataType: %s", resp.DataType)
	}

	// Round-trip: FromResponse must recover the original.
	got := FromResponse(resp)
	if got == nil || got.Status != StatusNotFound || got.Reason != "CALL_NOT_FOUND" {
		t.Errorf("round-trip failed: %+v", got)
	}
}

func TestToResponseAllStatuses(t *testing.T) {
	tests := []struct {
		status Status
		http   int
	}{
		{StatusInvalidArgument, 400},
		{StatusUnauthenticated, 401},
		{StatusPaymentRequired, 402},
		{StatusPermissionDenied, 403},
		{StatusNotFound, 404},
		{StatusAlreadyExists, 409},
		{StatusFailedPrecondition, 409},
		{StatusResourceExhausted, 429},
		{StatusUnavailable, 503},
		{StatusInternal, 500},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			e := &VoipbinError{Status: tt.status, Reason: "X", Domain: "d", Message: "m"}
			resp, err := ToResponse(e)
			if err != nil {
				t.Fatalf("ToResponse failed: %v", err)
			}
			if resp.StatusCode != tt.http {
				t.Errorf("wrong StatusCode for %s: got %d want %d", tt.status, resp.StatusCode, tt.http)
			}
		})
	}
}

func TestToResponseNil(t *testing.T) {
	if _, err := ToResponse(nil); err == nil {
		t.Errorf("ToResponse(nil) must return an error")
	}
}

func TestHTTPStatusFor(t *testing.T) {
	tests := []struct {
		status Status
		want   int
	}{
		{StatusInvalidArgument, 400},
		{StatusUnauthenticated, 401},
		{StatusPaymentRequired, 402},
		{StatusPermissionDenied, 403},
		{StatusNotFound, 404},
		{StatusAlreadyExists, 409},
		{StatusFailedPrecondition, 409},
		{StatusResourceExhausted, 429},
		{StatusUnavailable, 503},
		{StatusInternal, 500},
		{Status("UNKNOWN"), 500},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := HTTPStatusFor(tt.status); got != tt.want {
				t.Errorf("got %d want %d", got, tt.want)
			}
		})
	}
}
