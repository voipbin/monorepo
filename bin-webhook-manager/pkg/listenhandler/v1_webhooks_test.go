package listenhandler

import (
	"encoding/json"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/webhookhandler"
)

func Test_processV1WebhooksPost(t *testing.T) {

	tests := []struct {
		name string

		request    *rabbitmqhandler.Request
		customerID uuid.UUID
		dataType   webhook.DataType
		data       json.RawMessage

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/webhooks",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"2791e4a8-8253-11ec-811a-07cc4bcfbdaa","data_type":"application/json","data":{"type":"call_created","data":{"master_call_id":"00000000-0000-0000-0000-000000000000","direction":"outgoing","hangup_by":"","user_id":1,"id":"cdbe20b4-4f49-4a27-96d5-4dda0ec2134b","action":{"id":"00000000-0000-0000-0000-000000000001","tm_execute":"","type":""},"flow_id":"883f40c7-7f16-48fe-a404-772d9b038808","type":"flow","source":{"target":"+821028286521","name":"","type":"tel"},"tm_update":"","tm_ringing":"","data":null,"recording_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":[],"hangup_reason":"","webhook_uri":"https://endxhr87aa0bkge.m.pipedream.net","tm_progressing":"","conf_id":"00000000-0000-0000-0000-000000000000","tm_hangup":"","asterisk_id":"","status":"dialing","recording_ids":[],"destination":{"name":"","target":"+821021656521","type":"tel"},"channel_id":"8edc9ed7-2ec0-4027-a7b9-3d6eced18afd","tm_create":"2021-03-13 18:18:02.489462"}}}`),
			},
			uuid.FromStringOrNil("2791e4a8-8253-11ec-811a-07cc4bcfbdaa"),
			"application/json",
			[]byte(`{"type":"call_created","data":{"master_call_id":"00000000-0000-0000-0000-000000000000","direction":"outgoing","hangup_by":"","user_id":1,"id":"cdbe20b4-4f49-4a27-96d5-4dda0ec2134b","action":{"id":"00000000-0000-0000-0000-000000000001","tm_execute":"","type":""},"flow_id":"883f40c7-7f16-48fe-a404-772d9b038808","type":"flow","source":{"target":"+821028286521","name":"","type":"tel"},"tm_update":"","tm_ringing":"","data":null,"recording_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":[],"hangup_reason":"","webhook_uri":"https://endxhr87aa0bkge.m.pipedream.net","tm_progressing":"","conf_id":"00000000-0000-0000-0000-000000000000","tm_hangup":"","asterisk_id":"","status":"dialing","recording_ids":[],"destination":{"name":"","target":"+821021656521","type":"tel"},"channel_id":"8edc9ed7-2ec0-4027-a7b9-3d6eced18afd","tm_create":"2021-03-13 18:18:02.489462"}}`),

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
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				rabbitSock: mockSock,
				whHandler:  mockWeb,
			}

			mockWeb.EXPECT().SendWebhookToCustomer(gomock.Any(), tt.customerID, tt.dataType, tt.data).Return(nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1WebhookDestinationsPost(t *testing.T) {

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		customerID uuid.UUID
		uri        string
		method     webhook.MethodType
		dataType   webhook.DataType
		data       json.RawMessage

		expectRes *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			request: &rabbitmqhandler.Request{
				URI:      "/v1/webhook_destinations",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","uri":"test.com","method":"POST","data_type":"application/json","data":"test webhook."}`),
			},

			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			uri:        "test.com",
			method:     webhook.MethodTypePOST,
			dataType:   "application/json",
			data:       []byte(`"test webhook."`),

			expectRes: &rabbitmqhandler.Response{
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
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				rabbitSock: mockSock,
				whHandler:  mockWeb,
			}

			mockWeb.EXPECT().SendWebhookToURI(gomock.Any(), tt.customerID, tt.uri, tt.method, tt.dataType, tt.data).Return(nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
