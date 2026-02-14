package servicehandler

import (
	"context"
	"fmt"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

func Test_Message(t *testing.T) {
	tests := []struct {
		name string

		uri     string
		message []byte

		expectReq *hmhook.Hook
	}{
		{
			name: "normal",

			uri:     "hook.voipbin.net/v1.0/messages",
			message: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/messages",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},
		},
		{
			name: "message telnyx",

			uri:     "hook.voipbin.net/v1.0/messages/telnyx",
			message: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/messages/telnyx",
				ReceivedData: []byte(`{"key1":"val1"}`),
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

			mockReq.EXPECT().MessageV1Hook(ctx, tt.expectReq).Return(nil)

			if err := h.Message(ctx, tt.uri, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}

}

func Test_Message_Error(t *testing.T) {
	tests := []struct {
		name string

		uri     string
		message []byte

		expectReq   *hmhook.Hook
		expectError error
	}{
		{
			name: "request handler error",

			uri:     "hook.voipbin.net/v1.0/messages",
			message: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/messages",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},
			expectError: fmt.Errorf("message hook error"),
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

			mockReq.EXPECT().MessageV1Hook(ctx, tt.expectReq).Return(tt.expectError)

			if err := h.Message(ctx, tt.uri, tt.message); err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}
