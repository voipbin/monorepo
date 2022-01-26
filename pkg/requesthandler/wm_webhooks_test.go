package requesthandler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

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

		webhookMethod string
		webhookURI    string
		dataType      string
		messageType   string
		messageData   []byte

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			"POST",
			"test.com",
			"application/json",
			"application/json",
			[]byte(`{}`),

			"bin-manager.webhook-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/webhooks",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"method":"POST","webhook_uri":"test.com","data_type":"application/json","data":{"type":"application/json","data":{}}}`),
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

			err := reqHandler.WMV1WebhookSend(ctx, tt.webhookMethod, tt.webhookURI, tt.dataType, tt.messageType, tt.messageData)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
