package servicehandler

import (
	"context"
	"fmt"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

func Test_Conversation(t *testing.T) {
	tests := []struct {
		name string

		uri     string
		message []byte

		expectReq *hmhook.Hook
	}{
		{
			name: "normal",

			uri:     "hook.voipbin.net/v1.0/conversation",
			message: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/conversation",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},
		},
		{
			name: "conversation with path",

			uri:     "hook.voipbin.net/v1.0/conversation/customers/id/line",
			message: []byte(`{"test":"data"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/conversation/customers/id/line",
				ReceivedData: []byte(`{"test":"data"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConversationV1Hook(ctx, tt.expectReq).Return(nil)

			if err := h.Conversation(ctx, tt.uri, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Conversation_Error(t *testing.T) {
	tests := []struct {
		name string

		uri     string
		message []byte

		expectReq   *hmhook.Hook
		expectError error
	}{
		{
			name: "request handler error",

			uri:     "hook.voipbin.net/v1.0/conversation",
			message: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/conversation",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},
			expectError: fmt.Errorf("request handler error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConversationV1Hook(ctx, tt.expectReq).Return(tt.expectError)

			if err := h.Conversation(ctx, tt.uri, tt.message); err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}
