package servicehandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"

	hmhook "monorepo/bin-hook-manager/models/hook"
)

func Test_Message(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
	}

	tests := []struct {
		name string

		uri     string
		message []byte

		expectReq *hmhook.Hook
	}{
		{
			"normal",

			"hook.voipbin.net/v1.0/messages",
			[]byte(`{"key1":"val1"}`),

			&hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/messages",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},
		},
		{
			"message telnyx",

			"hook.voipbin.net/v1.0/messages/telnyx",
			[]byte(`{"key1":"val1"}`),

			&hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/messages/telnyx",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockReq.EXPECT().MessageV1Hook(ctx, tt.expectReq).Return(nil)

			if err := h.Message(ctx, tt.uri, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}

}
