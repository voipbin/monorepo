package apierror

import (
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

func TestEnvelopeFor_FullVoipbinError_OmitsDomain(t *testing.T) {
	e := cerrors.NotFound(commonoutline.ServiceNameCallManager, "CALL_NOT_FOUND", "The call was not found.")

	body := EnvelopeFor(e, "req-abc-123")

	outer, ok := body["error"].(gin.H)
	if !ok {
		t.Fatalf("body.error is not gin.H: %+v", body)
	}
	if outer["status"] != "NOT_FOUND" {
		t.Errorf("status = %v, want NOT_FOUND", outer["status"])
	}
	if outer["reason"] != "CALL_NOT_FOUND" {
		t.Errorf("reason = %v, want CALL_NOT_FOUND", outer["reason"])
	}
	if outer["message"] != "The call was not found." {
		t.Errorf("message = %v", outer["message"])
	}
	if outer["request_id"] != "req-abc-123" {
		t.Errorf("request_id = %v", outer["request_id"])
	}
	if _, present := outer["domain"]; present {
		t.Errorf("domain key MUST be absent, got: %v", outer["domain"])
	}
	if _, present := outer["details"]; present {
		t.Errorf("details key MUST be absent for empty Details, got: %v", outer["details"])
	}
}

func TestEnvelopeFor_DetailsIncluded(t *testing.T) {
	e := cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_FIELD", "Validation failed.")
	e.Details = []map[string]any{
		{"field": "destination", "issue": "must be E.164"},
	}

	body := EnvelopeFor(e, "req-1")

	outer := body["error"].(gin.H)
	details, ok := outer["details"].([]map[string]any)
	if !ok {
		t.Fatalf("details key missing or wrong type: %+v", outer)
	}
	if len(details) != 1 || details[0]["field"] != "destination" {
		t.Errorf("details payload not preserved: %+v", details)
	}
	if _, present := outer["domain"]; present {
		t.Errorf("domain key MUST be absent")
	}
}

func TestEnvelopeFor_NilDetails_OmitsKey(t *testing.T) {
	e := cerrors.NotFound(commonoutline.ServiceNameCallManager, "CALL_NOT_FOUND", "x")
	e.Details = nil

	body := EnvelopeFor(e, "req-1")

	outer := body["error"].(gin.H)
	if _, present := outer["details"]; present {
		t.Errorf("details key MUST be omitted when nil")
	}
}

func TestEnvelopeFor_NilVoipbinError_FallsBackToInternal(t *testing.T) {
	body := EnvelopeFor(nil, "req-fallback")

	outer := body["error"].(gin.H)
	if outer["status"] != "INTERNAL" {
		t.Errorf("fallback status = %v, want INTERNAL", outer["status"])
	}
	if outer["reason"] != "INTERNAL" {
		t.Errorf("fallback reason = %v, want INTERNAL", outer["reason"])
	}
	if outer["request_id"] != "req-fallback" {
		t.Errorf("request_id not preserved: %v", outer["request_id"])
	}
	if _, present := outer["domain"]; present {
		t.Errorf("domain key MUST be absent on fallback")
	}
}

func TestEnvelopeFor_EmptyRequestID(t *testing.T) {
	e := cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "x")

	body := EnvelopeFor(e, "")

	outer := body["error"].(gin.H)
	if outer["request_id"] != "" {
		t.Errorf("expected empty request_id, got %v", outer["request_id"])
	}
}
