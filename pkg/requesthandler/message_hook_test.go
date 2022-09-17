package requesthandler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	hmhook "gitlab.com/voipbin/bin-manager/hook-manager.git/models/hook"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_MessageV1Hook(t *testing.T) {

	tests := []struct {
		name string

		hookMessage *hmhook.Hook

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			&hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/messages/telnyx",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},

			"bin-manager.message-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/hooks",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"hook.voipbin.net/v1.0/messages/telnyx","received_data":"eyJrZXkxIjoidmFsMSJ9"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.MessageV1Hook(ctx, tt.hookMessage); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
