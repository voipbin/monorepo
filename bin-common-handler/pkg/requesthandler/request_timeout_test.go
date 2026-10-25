package requesthandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// Test_requestTimeout checks the deadline that a wrapper puts on the context of the published
// request. The shared RPC table does not look at the context, so a wrong timeout would pass there.
func Test_requestTimeout(t *testing.T) {
	callID := uuid.FromStringOrNil("d6e7f809-1a2b-4c3d-9e5f-f708192a3b4c")

	tests := []struct {
		name string

		queue    string
		response string
		call     func(h *requestHandler) error

		expectTimeout time.Duration
	}{
		{
			name:     "pcap get uses the long timeout",
			queue:    timelineQueue,
			response: `{"data":""}`,
			call: func(h *requestHandler) error {
				_, err := h.TimelineV1SIPPcapGet(context.Background(), callID, "sip-call-a", "2026-03-04T05:00:00Z", "2026-03-04T06:00:00Z")
				return err
			},
			expectTimeout: 30 * time.Second,
		},
		{
			name:     "sip analysis get uses the default timeout",
			queue:    timelineQueue,
			response: `{"sip_messages":[],"rtcp_stats":null}`,
			call: func(h *requestHandler) error {
				_, err := h.TimelineV1SIPAnalysisGet(context.Background(), callID, "sip-call-a", "2026-03-04T05:00:00Z", "2026-03-04T06:00:00Z")
				return err
			},
			expectTimeout: 3 * time.Second,
		},
		{
			name:     "password forgot passes the timeout of the caller",
			queue:    agentQueue,
			response: `{}`,
			call: func(h *requestHandler) error {
				return h.AgentV1PasswordForgot(context.Background(), 12000, "agent-a@example.com")
			},
			expectTimeout: 12 * time.Second,
		},
		{
			name:     "password reset passes the timeout of the caller",
			queue:    agentQueue,
			response: `{}`,
			call: func(h *requestHandler) error {
				return h.AgentV1PasswordReset(context.Background(), 8000, "reset-token-a", "synthetic-pass-a")
			},
			expectTimeout: 8 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &requestHandler{sock: mockSock}

			var remaining time.Duration
			var hasDeadline bool
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.queue, gomock.Any()).DoAndReturn(
				func(ctx context.Context, _ string, _ *sock.Request) (*sock.Response, error) {
					deadline, ok := ctx.Deadline()
					hasDeadline = ok
					remaining = time.Until(deadline)
					return &sock.Response{StatusCode: 200, DataType: ContentTypeJSON, Data: []byte(tt.response)}, nil
				})

			if err := tt.call(h); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			// the deadline is set right before the publish, so the remaining time is a little
			// below the timeout and never above it.
			if !hasDeadline {
				t.Fatalf("Wrong match. expect: a deadline on the context, got: none")
			}
			if remaining > tt.expectTimeout || remaining < tt.expectTimeout-time.Second {
				t.Errorf("Wrong match. expect the remaining time in (%v, %v], got: %v", tt.expectTimeout-time.Second, tt.expectTimeout, remaining)
			}
		})
	}
}
