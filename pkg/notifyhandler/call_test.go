package notifyhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

func TestNotifyCall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	exchangeDelay := ""
	exchangeNotify := "bin-manager.call-manager.event"

	h := &notifyHandler{
		sock:           mockSock,
		reqHandler:     mockReq,
		exchangeDelay:  exchangeDelay,
		exchangeNotify: exchangeNotify,
	}

	type test struct {
		name          string
		eventType     EventType
		call          *call.Call
		expectEvent   *rabbitmqhandler.Event
		expectWebhook []byte
	}

	tests := []test{
		{
			"create normal",
			EventTypeCallCreated,
			&call.Call{
				ID:         uuid.FromStringOrNil("14aa3450-84eb-11eb-8285-23e72da33b42"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "48a5446a-e3b1-11ea-b837-83239d9eb45f",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				Destination: address.Address{
					Target: string(action.TypeAnswer),
				},
			},
			&rabbitmqhandler.Event{
				Type:      string(EventTypeCallCreated),
				Publisher: EventPublisher,
				DataType:  dataTypeJSON,
				Data:      []byte(`{"id":"14aa3450-84eb-11eb-8285-23e72da33b42","user_id":0,"asterisk_id":"80:fa:5b:5e:da:81","channel_id":"48a5446a-e3b1-11ea-b837-83239d9eb45f","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
			[]byte{},
		},
		{
			"create webhook uri",
			EventTypeCallCreated,
			&call.Call{
				ID:         uuid.FromStringOrNil("6d06310e-1072-11ec-9606-27a7b382621f"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "72590c44-1072-11ec-85d2-73d06dfabfc2",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				Destination: address.Address{
					Target: string(action.TypeAnswer),
				},
				WebhookURI: "test.com/webhook",
			},
			&rabbitmqhandler.Event{
				Type:      string(EventTypeCallCreated),
				Publisher: EventPublisher,
				DataType:  dataTypeJSON,
				Data:      []byte(`{"id":"6d06310e-1072-11ec-9606-27a7b382621f","user_id":0,"asterisk_id":"80:fa:5b:5e:da:81","channel_id":"72590c44-1072-11ec-85d2-73d06dfabfc2","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
			[]byte(`{"id":"6d06310e-1072-11ec-9606-27a7b382621f","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
		},
		{
			"update normal",
			EventTypeCallUpdated,
			&call.Call{
				ID:         uuid.FromStringOrNil("52678c48-853b-11eb-9693-bbab415f20a4"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "5675714c-853b-11eb-a9a0-8340e19df2d1",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				Destination: address.Address{
					Target: string(action.TypeAnswer),
				},
				WebhookURI: "test.com/webhook",
			},
			&rabbitmqhandler.Event{
				Type:      string(EventTypeCallUpdated),
				Publisher: EventPublisher,
				DataType:  dataTypeJSON,
				Data:      []byte(`{"id":"52678c48-853b-11eb-9693-bbab415f20a4","user_id":0,"asterisk_id":"80:fa:5b:5e:da:81","channel_id":"5675714c-853b-11eb-a9a0-8340e19df2d1","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
			[]byte(`{"id":"52678c48-853b-11eb-9693-bbab415f20a4","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
		},
		{
			"hungup normal",
			EventTypeCallHungup,
			&call.Call{
				ID:         uuid.FromStringOrNil("717b275c-853b-11eb-a029-5f71e2f0a0d3"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "719ad61a-853b-11eb-aad3-0ff51d63724c",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				Destination: address.Address{
					Target: string(action.TypeAnswer),
				},
				WebhookURI: "test.com/webhook",
			},
			&rabbitmqhandler.Event{
				Type:      string(EventTypeCallHungup),
				Publisher: EventPublisher,
				DataType:  dataTypeJSON,
				Data:      []byte(`{"id":"717b275c-853b-11eb-a029-5f71e2f0a0d3","user_id":0,"asterisk_id":"80:fa:5b:5e:da:81","channel_id":"719ad61a-853b-11eb-aad3-0ff51d63724c","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
			[]byte(`{"id":"717b275c-853b-11eb-a029-5f71e2f0a0d3","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"sip-service","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"answer","name":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"incoming","hangup_by":"","hangup_reason":"","webhook_uri":"test.com/webhook","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishExchangeEvent(h.exchangeNotify, "", tt.expectEvent)
			if tt.call.WebhookURI != "" {
				mockReq.EXPECT().WMWebhookPOST("POST", tt.call.WebhookURI, dataTypeJSON, string(tt.eventType), tt.expectWebhook)
			}
			h.NotifyCall(context.Background(), tt.call, tt.eventType)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}
