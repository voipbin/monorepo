package messagehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/messagehandlermessagebird"
)

func Test_Hook(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,

		messageHandlerMessagebird: mockMessagebird,
	}

	tests := []struct {
		name string

		uri  string
		data []byte

		expectToNum string

		responseNumber  *nmnumber.Number
		responseMessage *message.Message
	}{
		{
			"normal",

			"https://hook.voipbin.net/v1.0/hooks/telnyx",
			[]byte(`{
				"data": {
				  "event_type": "message.received",
				  "id": "19539336-11ba-4792-abd8-26d4f8745c4c",
				  "occurred_at": "2022-03-15T16:16:24.073+00:00",
				  "payload": {
					"cc": [],
					"completed_at": null,
					"cost": null,
					"direction": "inbound",
					"encoding": "GSM-7",
					"errors": [],
					"from": {
					  "carrier": "",
					  "line_type": "",
					  "phone_number": "+75973"
					},
					"id": "5d7f9c50-330a-4d7a-9ca8-4157d7a09047",
					"media": [],
					"messaging_profile_id": "40017f8e-49bd-4f16-9e3d-ef103f916228",
					"organization_id": "a506eae0-f72c-449c-bbe5-19ce35f82e0b",
					"parts": 1,
					"received_at": "2022-03-15T16:16:23.466+00:00",
					"record_type": "message",
					"sent_at": null,
					"subject": "",
					"tags": [],
					"text": "pchero21:\nTest message from skype.",
					"to": [
					  {
						"carrier": "Telnyx",
						"line_type": "Wireless",
						"phone_number": "+15734531118",
						"status": "webhook_delivered"
					  }
					],
					"type": "SMS",
					"valid_until": null,
					"webhook_failover_url": null,
					"webhook_url": "https://en7evajwhmqbt.x.pipedream.net"
				  },
				  "record_type": "event"
				},
				"meta": {
				  "attempt": 1,
				  "delivered_to": "https://en7evajwhmqbt.x.pipedream.net"
				}
			  }`),

			"+15734531118",

			&nmnumber.Number{
				ID:     uuid.FromStringOrNil("67afdd50-a65d-11ec-84fa-8b61a2028c6a"),
				Number: "+15734531118",
			},
			&message.Message{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockReq.EXPECT().NMV1NumberGetByNumber(ctx, tt.expectToNum).Return(tt.responseNumber, nil)

			mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, gomock.Any()).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), message.EventTypeMessageCreated, gomock.Any())

			if errHook := h.Hook(ctx, tt.uri, tt.data); errHook != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errHook)
			}
		})
	}
}

func Test_executeMessageFlow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,

		messageHandlerMessagebird: mockMessagebird,
	}

	tests := []struct {
		name string

		m   *message.Message
		num *nmnumber.Number

		expectRes *fmactiveflow.Activeflow
	}{
		{
			"normal",

			&message.Message{
				ID: uuid.FromStringOrNil("1491e9e4-a8b8-11ec-bbe9-4b9389eaa6f7"),
			},
			&nmnumber.Number{
				ID:            uuid.FromStringOrNil("1f2db1da-a8b8-11ec-82b1-2bb474596df1"),
				MessageFlowID: uuid.FromStringOrNil("275a692a-a8b8-11ec-9de7-d39f5b03faec"),
			},

			&fmactiveflow.Activeflow{
				ID: uuid.FromStringOrNil("64447c9a-a8b8-11ec-a544-0b44fb74dc28"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockReq.EXPECT().FMV1ActiveflowCreate(ctx, tt.num.MessageFlowID, fmactiveflow.ReferenceTypeMessage, tt.m.ID).Return(tt.expectRes, nil)
			mockReq.EXPECT().FMV1ActiveflowExecute(ctx, tt.expectRes.ID).Return(nil)

			res, err := h.executeMessageFlow(ctx, tt.m, tt.num)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
