package notifyhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestCallCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	exchangeDelay := ""
	exchangeNotify := "bin-manager.call-manager.event"

	h := &notifyHandler{
		sock:           mockSock,
		exchangeDelay:  exchangeDelay,
		exchangeNotify: exchangeNotify,
	}

	type test struct {
		name        string
		call        *call.Call
		expectEvent *rabbitmqhandler.Event
	}

	tests := []test{
		{
			"empty option",
			&call.Call{
				ID:         uuid.FromStringOrNil("14aa3450-84eb-11eb-8285-23e72da33b42"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "48a5446a-e3b1-11ea-b837-83239d9eb45f",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				Destination: call.Address{
					Target: string(action.TypeAnswer),
				},
			},
			&rabbitmqhandler.Event{
				Type:      string(eventTypeCallCreated),
				Publisher: eventPublisher,
				DataType:  dataTypeJSON,
				Data:      []byte(`{"id":"14aa3450-84eb-11eb-8285-23e72da33b42","user_id":0,"asterisk_id":"80:fa:5b:5e:da:81","channel_id":"48a5446a-e3b1-11ea-b837-83239d9eb45f","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishExchangeEvent(h.exchangeNotify, "", tt.expectEvent)
			h.CallCreate(tt.call)
		})
	}
}
