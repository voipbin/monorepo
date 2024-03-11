package requesthandler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_CallV1ChannelHealth(t *testing.T) {

	tests := []struct {
		name string

		channelID  string
		delay      int
		retryCount int

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			"09574818-df60-11ee-a381-479b576528f0",
			0,
			1,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/channels/09574818-df60-11ee-a381-479b576528f0/health-check",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"retry_count":1}`),
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

			err := reqHandler.CallV1ChannelHealth(ctx, tt.channelID, tt.delay, tt.retryCount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
