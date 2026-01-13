package servicehandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/requesthandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
	"testing"

	gomock "go.uber.org/mock/gomock"
)

func Test_Email(t *testing.T) {
	tests := []struct {
		name string

		uri     string
		message []byte

		expectReq *hmhook.Hook
	}{
		{
			name: "normal",

			uri:     "hook.voipbin.net/v1.0/emails",
			message: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/emails",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},
		},
		{
			name: "sendgrid",

			uri:     "hook.voipbin.net/v1.0/emails/sendgrid",
			message: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/emails/sendgrid",
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

			mockReq.EXPECT().EmailV1Hooks(ctx, tt.expectReq.Return(nil)

			if err := h.Email(ctx, tt.uri, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}

}
