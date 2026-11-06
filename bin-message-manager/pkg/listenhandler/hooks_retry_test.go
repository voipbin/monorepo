package listenhandler

import (
	"errors"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/pkg/messagehandler"
)

// Test_processV1HooksPost_errorAnswers checks that only a receipt that must be delivered again is answered with 500:
// any other hook error stays a 200, because a redelivered inbound message would be created twice.
func Test_processV1HooksPost_errorAnswers(t *testing.T) {
	tests := []struct {
		name string

		hookErr error

		expectStatus int
	}{
		{"no error", nil, 200},
		{"a plain error", errors.New("number lookup failed"), 200},
		{"a receipt retry", fmt.Errorf("%w: db down", messagehandler.ErrReceiptRetry), 500},
		{"a wrapped receipt retry", fmt.Errorf("could not handle: %w", fmt.Errorf("%w: conflict", messagehandler.ErrReceiptRetry)), 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockMessageHandler := messagehandler.NewMockMessageHandler(mc)
			h := &listenHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				messageHandler: mockMessageHandler,
			}

			mockMessageHandler.EXPECT().Hook(gomock.Any(), "hook.voipbin.net/v1.0/messages/telnyx", []byte("{}")).Return(tt.hookErr)

			res, err := h.processRequest(&sock.Request{
				URI:      "/v1/hooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"hook.voipbin.net/v1.0/messages/telnyx","received_data":"e30="}`),
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}
