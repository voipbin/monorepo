package listenhandler

import (
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
)

func Test_processV1ChannelsIDHealthPost(t *testing.T) {

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		asteriskID    string
		channelID     string
		retryCount    int
		retryCountMax int
		delay         int
	}{
		{
			"channel id is uuid",

			&rabbitmqhandler.Request{
				URI:    "/v1/channels/f1f90a0a-9844-11ea-8948-5378837e7179/health-check",
				Method: rabbitmqhandler.RequestMethodPost,
				Data:   []byte(`{"retry_count": 0, "retry_count_max": 2, "delay": 10000}`),
			},

			"42:01:0a:a4:00:05",
			"f1f90a0a-9844-11ea-8948-5378837e7179",
			0,
			2,
			10000,
		},
		{
			"channel id is not uuid",

			&rabbitmqhandler.Request{
				URI:    "/v1/channels/asterisk-call-58f54b64c7-d7sv7-1676744879.1115/health-check",
				Method: rabbitmqhandler.RequestMethodPost,
				Data:   []byte(`{"retry_count": 0, "retry_count_max": 2, "delay": 10000}`),
			},

			"42:01:0a:a4:00:05",
			"asterisk-call-58f54b64c7-d7sv7-1676744879.1115",
			0,
			2,
			10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				callHandler:    mockCall,
				channelHandler: mockChannel,
			}

			mockChannel.EXPECT().HealthCheck(gomock.Any(), tt.channelID, tt.retryCount)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			} else if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}

		})
	}
}
