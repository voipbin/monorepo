package requesthandler

import (
	"encoding/json"
	amagent "monorepo/bin-agent-manager/models/agent"
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
