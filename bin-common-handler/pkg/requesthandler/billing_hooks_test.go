package requesthandler

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

func Test_BillingV1PaddleHook(t *testing.T) {

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
				ReceviedURI:  "hook.voipbin.net/v1.0/billing/paddle",
				ReceivedData: []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`),
			},

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/hooks/paddle",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"hook.voipbin.net/v1.0/billing/paddle","received_data":"eyJldmVudF9pZCI6ImV2dF8wMDEiLCJldmVudF90eXBlIjoidHJhbnNhY3Rpb24uY29tcGxldGVkIn0=","received_method":"","received_signature":""}`),
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

			if err := reqHandler.BillingV1PaddleHook(ctx, tt.hookMessage); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
