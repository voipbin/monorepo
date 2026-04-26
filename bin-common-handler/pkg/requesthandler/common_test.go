package requesthandler

import (
	"encoding/json"
	stderrors "errors"
	amagent "monorepo/bin-agent-manager/models/agent"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func Test_converError_error(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int

		expectedCauseError error
	}{
		{
			name:               "mapped 4xx status code returns sentinel error",
			statusCode:         http.StatusBadRequest, // 400
			expectedCauseError: ErrBadRequest,
		},
		{
			name:               "mapped 5xx status code returns sentinel error",
			statusCode:         http.StatusInternalServerError, // 500
			expectedCauseError: ErrInternal,
		},
		{
			name:               "unmapped 4xx status code returns default fmt error",
			statusCode:         499,
			expectedCauseError: ErrUnknown,
		},
		{
			name:               "unmapped 5xx status code returns default fmt error",
			statusCode:         599,
			expectedCauseError: ErrUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := getResponseStatusCodeError(tt.statusCode)

			if errors.Cause(err) != tt.expectedCauseError {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectedCauseError, errors.Cause(err))
			}
		})
	}
}

func Test_parseResponse(t *testing.T) {
	tests := []struct {
		name string
		resp *sock.Response
		out  any

		expectedRes any
	}{
		{
			name:        "normal",
			resp:        &sock.Response{StatusCode: 200, Data: json.RawMessage(`{"name":"Alice"}`)},
			out:         &amagent.Agent{},
			expectedRes: &amagent.Agent{Name: "Alice"},
		},
		{
			name:        "response is nil",
			resp:        nil,
			out:         amagent.Agent{},
			expectedRes: amagent.Agent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errParse := parseResponse(tt.resp, tt.out); errParse != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", errParse)
			}

			if !reflect.DeepEqual(tt.out, tt.expectedRes) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectedRes, tt.out)
			}
		})
	}
}

func Test_parseResponse_typedVoipbinError(t *testing.T) {
	// Typed VoipbinError envelope — DataType set, valid JSON body, 4xx
	// status. parseResponse must return the typed error so errors.As
	// recovers it at the api-manager edge.
	typed := &cerrors.VoipbinError{
		Status:  cerrors.StatusNotFound,
		Reason:  "RESOURCE_NOT_FOUND",
		Domain:  "billing-manager",
		Message: "Account not found.",
	}
	body, err := json.Marshal(typed)
	if err != nil {
		t.Fatalf("marshal typed error: %v", err)
	}

	resp := &sock.Response{
		StatusCode: http.StatusNotFound,
		DataType:   cerrors.DataTypeVoipbinError,
		Data:       body,
	}

	got := parseResponse(resp, nil)
	if got == nil {
		t.Fatalf("expected typed error, got nil")
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(got, &ve) {
		t.Fatalf("expected *VoipbinError via errors.As, got %T: %v", got, got)
	}
	if ve.Status != cerrors.StatusNotFound {
		t.Errorf("Status mismatch. expected: %s, got: %s", cerrors.StatusNotFound, ve.Status)
	}
	if ve.Reason != "RESOURCE_NOT_FOUND" {
		t.Errorf("Reason mismatch. expected: RESOURCE_NOT_FOUND, got: %s", ve.Reason)
	}
	if ve.Domain != "billing-manager" {
		t.Errorf("Domain mismatch. expected: billing-manager, got: %s", ve.Domain)
	}
}

func Test_parseResponse_typedFallsThroughOnMalformedJSON(t *testing.T) {
	// DataType claims VoipbinError but Data is malformed — FromResponse
	// returns nil and parseResponse must fall through to the legacy
	// HttpStatusErrorMap path so callers still get a usable sentinel.
	resp := &sock.Response{
		StatusCode: http.StatusNotFound,
		DataType:   cerrors.DataTypeVoipbinError,
		Data:       json.RawMessage(`{"status":`), // truncated
	}

	got := parseResponse(resp, nil)
	if got == nil {
		t.Fatalf("expected error, got nil")
	}

	var ve *cerrors.VoipbinError
	if stderrors.As(got, &ve) {
		t.Errorf("expected legacy sentinel, got typed error: %v", got)
	}
	if errors.Cause(got) != ErrNotFound {
		t.Errorf("expected legacy ErrNotFound, got: %v", errors.Cause(got))
	}
}

func Test_parseResponse_typedErrorSurvivesPkgErrorsWrap(t *testing.T) {
	// Many call sites in this package wrap parseResponse's return with
	// pkg/errors.Wrapf for context — and the api-manager translator
	// recovers the typed error via stderrors.As. This test guards the
	// pkg/errors.Wrapf -> stderrors.As contract: pkg/errors v0.9.1+ does
	// implement Unwrap(), but if that ever changes (or the dep gets
	// downgraded) this test catches the regression at the seam.
	typed := &cerrors.VoipbinError{
		Status:  cerrors.StatusInvalidArgument,
		Reason:  "INVALID_STATUS",
		Domain:  "billing-manager",
		Message: "Status is not allowed.",
	}
	body, err := json.Marshal(typed)
	if err != nil {
		t.Fatalf("marshal typed error: %v", err)
	}

	resp := &sock.Response{
		StatusCode: http.StatusBadRequest,
		DataType:   cerrors.DataTypeVoipbinError,
		Data:       body,
	}

	parsed := parseResponse(resp, nil)
	if parsed == nil {
		t.Fatalf("expected typed error from parseResponse, got nil")
	}

	wrapped := errors.Wrapf(parsed, "outer context: extra info")

	var ve *cerrors.VoipbinError
	if !stderrors.As(wrapped, &ve) {
		t.Fatalf("errors.As failed to recover *VoipbinError through pkg/errors.Wrapf chain. wrapped=%v", wrapped)
	}
	if ve.Reason != "INVALID_STATUS" {
		t.Errorf("Reason mismatch after unwrap. expected: INVALID_STATUS, got: %s", ve.Reason)
	}
}

func Test_parseResponse_legacyUnchangedWhenDataTypeAbsent(t *testing.T) {
	// No DataType header — FromResponse returns nil, legacy path fires.
	// This guards against regressing non-migrated managers.
	resp := &sock.Response{
		StatusCode: http.StatusInternalServerError,
		Data:       json.RawMessage(`{"some":"body"}`),
	}

	got := parseResponse(resp, nil)
	if got == nil {
		t.Fatalf("expected error, got nil")
	}

	var ve *cerrors.VoipbinError
	if stderrors.As(got, &ve) {
		t.Errorf("expected legacy sentinel, got typed error: %v", got)
	}
	if errors.Cause(got) != ErrInternal {
		t.Errorf("expected legacy ErrInternal, got: %v", errors.Cause(got))
	}
}

func Test_parseResponse_error(t *testing.T) {
	tests := []struct {
		name string
		resp *sock.Response
		out  any

		expectedRes string
	}{
		{
			name:        "out is not a pointer",
			resp:        &sock.Response{StatusCode: 200, Data: json.RawMessage(`{"name":"Alice"}`)},
			out:         amagent.Agent{},
			expectedRes: "out must be a pointer",
		},
		{
			name:        "invalid JSON",
			resp:        &sock.Response{StatusCode: 200, Data: json.RawMessage(`{"name":`)},
			out:         amagent.Agent{},
			expectedRes: "out must be a pointer, got agent.Agent",
		},
		{
			name:        "status code error",
			resp:        &sock.Response{StatusCode: 500, Data: json.RawMessage(`{"name":"Alice"}`)},
			out:         amagent.Agent{},
			expectedRes: ErrInternal.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := parseResponse(tt.resp, tt.out)
			if res == nil {
				t.Errorf("Wrong match. expected: error, got: ok")
			}

			if !strings.Contains(res.Error(), tt.expectedRes) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectedRes, res.Error())
			}
		})
	}
}
