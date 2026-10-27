package webhookhandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/activeflowhandler"
)

// Test_sendWebhookToActiveflow_skipsExtraDelivery covers the paths on which the per-activeflow delivery
// is skipped. They return before a delivery goroutine is started.
func Test_sendWebhookToActiveflow_skipsExtraDelivery(t *testing.T) {
	activeflowID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name string

		data json.RawMessage

		// expectGet is true when the destination must be resolved with activeflowID.
		expectGet   bool
		responseErr error
	}{
		{
			name: "malformed envelope is not resolved",
			data: []byte(`{"type":"call_updated","data":`),
		},
		{
			name:        "resolver failure skips the extra delivery",
			data:        []byte(`{"type":"call_updated","data":{"activeflow_id":"11111111-1111-1111-1111-111111111111"}}`),
			expectGet:   true,
			responseErr: errors.New("resolver down"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockActiveflow := activeflowhandler.NewMockActiveflowHandler(mc)
			st := &stubTransport{responses: []func() (*http.Response, error){stubStatus(200)}}
			h := &webhookHandler{activeflowHandler: mockActiveflow, httpClient: &http.Client{Transport: st}}

			ctx := context.Background()
			if tt.expectGet {
				mockActiveflow.EXPECT().Get(ctx, activeflowID).Return(nil, tt.responseErr)
			}

			h.sendWebhookToActiveflow(ctx, "application/json", tt.data, "")

			if st.calls() != 0 {
				t.Errorf("Wrong match. expect: no request, got: %d", st.calls())
			}
		})
	}
}

// chanTransport hands every request it receives to a channel and answers 200.
type chanTransport struct {
	requests chan *http.Request
	bodies   chan string
}

func (c *chanTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body string
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}
	c.requests <- req
	c.bodies <- body

	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}, nil
}

// Test_SendWebhookToURI_accountLookupFails verifies that a failing signing-secret lookup does not block
// the delivery: the request goes out with its body and without a signature.
func Test_SendWebhookToURI_accountLookupFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e")
	data := json.RawMessage(`{"type":"call_updated","data":{"type":"call"}}`)

	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockTopicNotify := notifyhandler.NewMockNotifyHandler(mc)
	ct := &chanTransport{requests: make(chan *http.Request, 1), bodies: make(chan string, 1)}
	h := &webhookHandler{
		topicNotifyHandler: mockTopicNotify,
		accoutHandler:      mockAccount,
		httpClient:         &http.Client{Transport: ct},
	}

	ctx := context.Background()
	mockAccount.EXPECT().Get(ctx, customerID).Return((*account.Account)(nil), errors.New("account lookup failed"))

	if err := h.SendWebhookToURI(ctx, customerID, publicTestURI, webhook.MethodTypePOST, "application/json", data); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case req := <-ct.requests:
		if got := req.Header.Get(signatureHeader); got != "" {
			t.Errorf("Wrong match. expect: unsigned request, got signature: %q", got)
		}
		if req.Method != "POST" || req.URL.String() != publicTestURI {
			t.Errorf("Wrong match. got: %s %s", req.Method, req.URL)
		}
		if got := <-ct.bodies; got != string(data) {
			t.Errorf("Wrong match. expect body: %q, got: %q", string(data), got)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("The request was not delivered")
	}
}
