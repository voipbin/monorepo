package listenhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-webhook-manager/pkg/webhookhandler"
)

func Test_Run(t *testing.T) {
	tests := []struct {
		name string

		queueCreateErr error
		consumeErr     error

		expectConsume     bool
		expectErrContains string
	}{
		{
			name:              "queue create failure returns an error without consuming",
			queueCreateErr:    errors.New("boom"),
			expectErrContains: "could not declare the queue",
		},
		{
			name:          "success starts the RPC consumer",
			expectConsume: true,
		},
		{
			name:          "consumer failure does not change the result of Run",
			consumeErr:    errors.New("consume failed"),
			expectConsume: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockWebhook := webhookhandler.NewMockWebhookHandler(mc)
			h := NewListenHandler(mockSock, mockWebhook)

			mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(tt.queueCreateErr).Times(1)

			consumed := make(chan struct{})
			if tt.expectConsume {
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", "webhook-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
					func(_ context.Context, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgRPC) error {
						close(consumed)
						return tt.consumeErr
					}).Times(1)
			}

			err := h.Run("test-queue", "")
			if tt.expectErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErrContains) {
					t.Errorf("Wrong match. expect error containing %q, got: %v", tt.expectErrContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: nil, got: %v", err)
			}

			select {
			case <-consumed:
			case <-time.After(5 * time.Second):
				t.Fatalf("ConsumeRPC was not called")
			}
		})
	}
}
