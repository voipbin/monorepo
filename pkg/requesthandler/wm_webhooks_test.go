package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	wmwebhook "gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestWMV1WebhookSend(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		customerID  uuid.UUID
		dataType    wmwebhook.DataType
		messageType string
		messageData []byte

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("d2c2ffe8-825c-11ec-8688-2bebcc3d0013"),
			wmwebhook.DataTypeJSON,
			"application/json",
			[]byte(`{}`),

			"bin-manager.webhook-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/webhooks",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"d2c2ffe8-825c-11ec-8688-2bebcc3d0013","data_type":"application/json","data":{"type":"application/json","data":{}}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.WMV1WebhookSend(ctx, tt.customerID, tt.dataType, tt.messageType, tt.messageData)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
