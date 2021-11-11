package listenhandler

import (
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestProcessV1ChannelsIDHealthPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &listenHandler{
		rabbitSock: mockSock,
		db:         mockDB,
		reqHandler: mockReq,
	}

	type test struct {
		name    string
		channel *channel.Channel
		request *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal test",
			&channel.Channel{
				ID:         "f1f90a0a-9844-11ea-8948-5378837e7179",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			&rabbitmqhandler.Request{
				URI:    "/v1/asterisks/42%3A01%3A0a%3Aa4%3A00%3A05/channels/f1f90a0a-9844-11ea-8948-5378837e7179/health-check",
				Method: rabbitmqhandler.RequestMethodPost,
				Data:   []byte(`{"retry_count": 0, "retry_count_max": 2, "delay": 10000}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(tt.channel, nil)
			mockReq.EXPECT().AstChannelGet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID).Return(tt.channel, nil)
			mockReq.EXPECT().CallChannelHealth(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, 10000, 0, 2).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			} else if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}

		})
	}
}
