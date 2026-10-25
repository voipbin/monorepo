package paddlehandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monorepo/bin-billing-manager/models/account"
)

func newTestPaddleHandler(baseURL string) *paddleHandler {
	h := &paddleHandler{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
		priceMap:   map[string]account.PlanType{},
	}
	h.apiKey = "synthetic-test-key"
	return h
}

// Test_CreatePortalSession_request_failures verifies the failures before a response is parsed.
func Test_CreatePortalSession_request_failures(t *testing.T) {
	t.Run("request cannot be built", func(t *testing.T) {
		// a control character makes the URL invalid, so no request is created or sent.
		h := newTestPaddleHandler("http://host\x7f")

		url, err := h.CreatePortalSession(context.Background(), "ctm_synthetic_001")
		if err == nil || !strings.Contains(err.Error(), "could not create request") {
			t.Errorf("expected the request build failure, got: %v", err)
		}
		if url != "" {
			t.Errorf("expected an empty url, got: %s", url)
		}
	})

	t.Run("response body cannot be read", func(t *testing.T) {
		// the server announces more bytes than it sends and closes, so reading the body fails.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("{"))
		}))
		defer server.Close()

		h := newTestPaddleHandler(server.URL)

		url, err := h.CreatePortalSession(context.Background(), "ctm_synthetic_002")
		if err == nil || !strings.Contains(err.Error(), "could not read response body") {
			t.Errorf("expected the body read failure, got: %v", err)
		}
		if url != "" {
			t.Errorf("expected an empty url, got: %s", url)
		}
	})
}
