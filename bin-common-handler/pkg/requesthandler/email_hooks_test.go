package requesthandler

import (
	"context"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
	"testing"

	"go.uber.org/mock/gomock"
)

func Test_EmailV1Hooks(t *testing.T) {

	tests := []struct {
		name string

		hookMessage *hmhook.Hook

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
	}{
		{
			name: "normal",

			hookMessage: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/email/sendgrid",
				ReceivedData: []byte(`{"voipbin_message_id": "12a4b8b0-007a-11f0-a49b-6fa21c3b2cc3"}`),
			},

			expectTarget: "bin-manager.email-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/hooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"hook.voipbin.net/v1.0/email/sendgrid","received_data":"eyJ2b2lwYmluX21lc3NhZ2VfaWQiOiAiMTJhNGI4YjAtMDA3YS0xMWYwLWE0OWItNmZhMjFjM2IyY2MzIn0="}`),
			},
			response: &sock.Response{
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

			if err := reqHandler.EmailV1Hooks(ctx, tt.hookMessage); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
