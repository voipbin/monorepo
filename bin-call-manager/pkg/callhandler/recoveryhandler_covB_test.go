package callhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func Test_GetRecoveryDetail_covB(t *testing.T) {
	tests := []struct {
		name       string
		useServer  bool
		address    string
		token      string
		statusCode int
		body       string
	}{
		{name: "missing address", token: "token"},
		{name: "missing token", address: "http://127.0.0.1:1"},
		{name: "homer http error", useServer: true, token: "token", statusCode: http.StatusInternalServerError, body: `{}`},
		{name: "no sip messages found", useServer: true, token: "token", statusCode: http.StatusOK, body: `{"data":{"messages":[]}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := tt.address
			client := http.DefaultClient
			if tt.useServer {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				}))
				defer srv.Close()
				address = srv.URL
				client = srv.Client()
			}

			h := &recoveryHandler{
				httpClient:      client,
				homerAPIAddress: address,
				homerAuthToken:  tt.token,
			}

			res, err := h.GetRecoveryDetail(context.Background(), "call-id-1", asteriskRoleUAS)
			if err == nil || res != nil {
				t.Errorf("expected error. res: %v, err: %v", res, err)
			}
		})
	}
}

func Test_GetRecoveryDetail_missingConfigError_covB(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"data":{"messages":[]}}`))
	}))
	defer srv.Close()

	tests := []struct {
		name    string
		address string
		token   string
	}{
		{"address only is empty", "", "token"},
		{"token only is empty", srv.URL, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &recoveryHandler{httpClient: srv.Client(), homerAPIAddress: tt.address, homerAuthToken: tt.token}

			_, err := h.GetRecoveryDetail(context.Background(), "call-id-1", asteriskRoleUAS)
			if err == nil || err.Error() != "missing Homer API address or auth token" {
				t.Errorf("Wrong match. expect: missing Homer API address or auth token, got: %v", err)
			}
			if n := atomic.LoadInt32(&hits); n != 0 {
				t.Errorf("Wrong match. expect: no request, got: %d", n)
			}
		})
	}
}
