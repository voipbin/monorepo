package requesthandler

import (
	"context"
	"testing"

	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_WebhookV1WebhookSend(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		dataType    wmwebhook.DataType
		messageType string
		messageData []byte

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("d2c2ffe8-825c-11ec-8688-2bebcc3d0013"),
			wmwebhook.DataTypeJSON,
			"application/json",
			[]byte(`{}`),

			"bin-manager.webhook-manager.request",
			&sock.Request{
				URI:      "/v1/webhooks",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"d2c2ffe8-825c-11ec-8688-2bebcc3d0013","data_type":"application/json","data":{"type":"application/json","data":{}}}`),
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

			err := reqHandler.WebhookV1WebhookSend(ctx, tt.customerID, tt.dataType, tt.messageType, tt.messageData)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_WebhookV1WebhookDestinationSend(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		destination string
		method      wmwebhook.MethodType
		dataType    wmwebhook.DataType
		data        []byte

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("d2c2ffe8-825c-11ec-8688-2bebcc3d0013"),
			destination: "test.com",
			method:      wmwebhook.MethodTypePOST,
			dataType:    wmwebhook.DataTypeJSON,

			expectTarget: "bin-manager.webhook-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/webhook_destinations",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"d2c2ffe8-825c-11ec-8688-2bebcc3d0013","uri":"test.com","method":"POST","data_type":"application/json","data":"test webhook."}`),
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

			a := []byte(`"test webhook."`)
			err := reqHandler.WebhookV1WebhookSendToDestination(ctx, tt.customerID, tt.destination, tt.method, tt.dataType, a)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
