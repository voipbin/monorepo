package requesthandler

import (
	"context"
	"testing"

	hmhook "monorepo/bin-hook-manager/models/hook"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_ConversationV1Hook(t *testing.T) {

	tests := []struct {
		name string

		hookMessage *hmhook.Hook

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			name: "normal",

			hookMessage: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/conversation/customers/7a008138-ea75-11ec-a1ab-83428342ec10/line",
				ReceivedData: []byte(`{"destination": "U11298214116e3afbad432b5794a6d3a0"}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/hooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"hook.voipbin.net/v1.0/conversation/customers/7a008138-ea75-11ec-a1ab-83428342ec10/line","received_data":"eyJkZXN0aW5hdGlvbiI6ICJVMTEyOTgyMTQxMTZlM2FmYmFkNDMyYjU3OTRhNmQzYTAifQ=="}`),
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

			if err := reqHandler.ConversationV1Hook(ctx, tt.hookMessage); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
