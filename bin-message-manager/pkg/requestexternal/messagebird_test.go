package requestexternal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_MessagebirdSendMessage_canceledContext(t *testing.T) {
	h := NewRequestExternal(staticToken("token"), staticToken("token"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// the request carries the caller's context, so a canceled context fails before any network use
	_, err := h.MessagebirdSendMessage(ctx, "+15550100", []string{"+15550101"}, "hello")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("MessagebirdSendMessage() error = %v, want it to wrap context.Canceled", err)
	}
}

// Test_MessagebirdSendMessage_status checks the answers: a non 2xx answer is a ProviderError that carries the error
// code and description only, never the raw body, and a 2xx answer is parsed.
func Test_MessagebirdSendMessage_status(t *testing.T) {
	tests := []struct {
		name string

		status int
		body   string

		expectErr    bool
		expectDetail string
	}{
		{"201 is accepted", 201, `{"id":"mb-1","recipients":{"items":[{"recipient":15550100002,"status":"scheduled","messagePartCount":2}]}}`, false, ""},
		{"422 carries the reason", 422, `{"errors":[{"code":9,"description":"recipient 15550100002 is not allowed","parameter":"recipients"}]}`, true, "code 9 recipient <num> is not allowed"},
		{"422 with a body that is not json", 422, `<html>15550100002</html>`, true, ""},
		{"500", 500, `{}`, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			old := messagebirdMessagesURI
			messagebirdMessagesURI = server.URL
			defer func() { messagebirdMessagesURI = old }()

			h := NewRequestExternal(staticToken("fake-token-mb"), staticToken("fake-token-tn"))
			res, err := h.MessagebirdSendMessage(context.Background(), "+15550100001", []string{"+15550100002"}, "hello")
			if !tt.expectErr {
				if err != nil || res == nil || res.ID != "mb-1" || res.Recipients.Items[0].MessagePartCount != 2 {
					t.Errorf("Wrong match. expect: ok, got: res: %v, err: %v", res, err)
				}
				return
			}

			var pe *ProviderError
			if err == nil || !errors.As(err, &pe) || pe.StatusCode != tt.status || pe.Provider != providerMessagebird {
				t.Fatalf("Wrong match. expect: ProviderError %d, got: %v", tt.status, err)
			}
			if pe.Detail != tt.expectDetail {
				t.Errorf("Wrong detail. expect: %q, got: %q", tt.expectDetail, pe.Detail)
			}
			if strings.Contains(err.Error(), "15550100002") || strings.Contains(err.Error(), "<html>") {
				t.Errorf("The raw body must not be in the error. err: %v", err)
			}
		})
	}
}
