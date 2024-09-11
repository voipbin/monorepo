package requesthandler

import (
	"context"
	"testing"

	hmhook "monorepo/bin-hook-manager/models/hook"

	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_MessageV1Hook(t *testing.T) {

	tests := []struct {
		name string

		hookMessage *hmhook.Hook

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
	}{
		{
			"normal",

			&hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/messages/telnyx",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},

			"bin-manager.message-manager.request",
			&sock.Request{
				URI:      "/v1/hooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"hook.voipbin.net/v1.0/messages/telnyx","received_data":"eyJrZXkxIjoidmFsMSJ9"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.MessageV1Hook(ctx, tt.hookMessage); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
