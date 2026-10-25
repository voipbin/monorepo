package actionhandler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
)

func Test_ActionFetchGet_request(t *testing.T) {
	activeflowID := uuid.FromStringOrNil("41712ed0-ce50-11ec-a29f-b3616bd154d6")
	referenceID := uuid.FromStringOrNil("549d358a-fbfc-11ea-a625-43073fda56b9")

	var gotMethod, gotContentType string
	var gotBody map[string]string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		_, _ = w.Write([]byte(`[{"type":"answer"},{"type":"hangup"}]`))
	}))
	defer ts.Close()

	h := &actionHandler{}
	act := &action.Action{
		ID:     uuid.FromStringOrNil("6e2a0cee-fba2-11ea-a469-a350f2dad844"),
		Option: map[string]any{"event_url": ts.URL, "event_method": "POST"},
	}

	res, err := h.ActionFetchGet(context.Background(), act, activeflowID, referenceID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectedRes := []action.Action{{Type: action.TypeAnswer}, {Type: action.TypeHangup}}
	if !reflect.DeepEqual(res, expectedRes) {
		t.Errorf("Wrong result. expect: %v, got: %v", expectedRes, res)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("Wrong method. expect: POST, got: %s", gotMethod)
	}
	if gotContentType != "application/json; charset=utf-8" {
		t.Errorf("Wrong content type. got: %s", gotContentType)
	}
	expectedBody := map[string]string{"activeflow_id": activeflowID.String(), "reference_id": referenceID.String()}
	if !reflect.DeepEqual(gotBody, expectedBody) {
		t.Errorf("Wrong request body. expect: %v, got: %v", expectedBody, gotBody)
	}
}

func Test_ActionFetchGet_error(t *testing.T) {
	tests := []struct {
		name string

		handler func(w http.ResponseWriter, r *http.Request) // nil means the url is not served
		option  func(url string) map[string]any

		expectedInErr string
	}{
		{
			name:          "option cannot be marshaled",
			option:        func(url string) map[string]any { return map[string]any{"event_url": make(chan int)} },
			expectedInErr: "could not marshal the option",
		},
		{
			name:          "option has a wrong type",
			option:        func(url string) map[string]any { return map[string]any{"event_url": 5} },
			expectedInErr: "cannot unmarshal number",
		},
		{
			name:          "request cannot be created from the method",
			option:        func(url string) map[string]any { return map[string]any{"event_url": url, "event_method": "BAD METHOD"} },
			expectedInErr: "invalid method",
		},
		{
			name:          "request cannot be sent to an unsupported scheme",
			option:        func(url string) map[string]any { return map[string]any{"event_url": "bogus://host/path"} },
			expectedInErr: "unsupported protocol scheme",
		},
		{
			name: "remote answers with a non-200 status",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`[{"type":"answer"}]`))
			},
			option:        func(url string) map[string]any { return map[string]any{"event_url": url} },
			expectedInErr: "status return. stauts: 500",
		},
		{
			name: "remote answers with a body that is cut short",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "100")
				_, _ = w.Write([]byte(`[{"type"`))
			},
			option:        func(url string) map[string]any { return map[string]any{"event_url": url} },
			expectedInErr: "unexpected EOF",
		},
		{
			name: "remote answers with a body that is not an action list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"type":"answer"}`))
			},
			option:        func(url string) map[string]any { return map[string]any{"event_url": url} },
			expectedInErr: "cannot unmarshal object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := ""
			if tt.handler != nil {
				ts := httptest.NewServer(http.HandlerFunc(tt.handler))
				defer ts.Close()
				url = ts.URL
			}

			h := &actionHandler{}
			act := &action.Action{
				ID:     uuid.FromStringOrNil("6e2a0cee-fba2-11ea-a469-a350f2dad844"),
				Option: tt.option(url),
			}

			res, err := h.ActionFetchGet(context.Background(), act, uuid.Nil, uuid.Nil)
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectedInErr) {
				t.Errorf("Wrong error. expect to contain: %q, got: %v", tt.expectedInErr, err)
			}
		})
	}
}
