package listenhandler

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/webhookhandler"
)

func TestProcessV1WebhooksPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockWeb := webhookhandler.NewMockWebhookHandler(mc)

	h := &listenHandler{
		rabbitSock: mockSock,
		whHandler:  mockWeb,
	}

	type test struct {
		name          string
		request       *rabbitmqhandler.Request
		expectRes     *rabbitmqhandler.Response
		expectWebhook *webhook.Webhook
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/webhooks",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"method":"POST","webhook_uri":"http://test.com/webhook","data_type":"application/json","data":{"type":"call_created","data":{"master_call_id": "00000000-0000-0000-0000-000000000000","direction": "outgoing","hangup_by": "","user_id": 1,"id": "cdbe20b4-4f49-4a27-96d5-4dda0ec2134b","action": {"id": "00000000-0000-0000-0000-000000000001","tm_execute": "","type": ""},"flow_id": "883f40c7-7f16-48fe-a404-772d9b038808","type": "flow","source": {"target": "+821028286521","name": "","type": "tel"},"tm_update": "","tm_ringing": "","data": null,"recording_id": "00000000-0000-0000-0000-000000000000","chained_call_ids": [],"hangup_reason": "","webhook_uri": "https://endxhr87aa0bkge.m.pipedream.net","tm_progressing": "","conf_id": "00000000-0000-0000-0000-000000000000","tm_hangup": "","asterisk_id": "","status": "dialing","recording_ids": [],"destination": {"name": "","target": "+821021656521","type": "tel"},"channel_id": "8edc9ed7-2ec0-4027-a7b9-3d6eced18afd","tm_create": "2021-03-13 18:18:02.489462"}}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
			&webhook.Webhook{
				Method:     webhook.MethodTypePOST,
				WebhookURI: "http://test.com/webhook",
				DataType:   webhook.DataTypeJSON,
				Data:       []byte(`{"type":"call_created","data":{"master_call_id": "00000000-0000-0000-0000-000000000000","direction": "outgoing","hangup_by": "","user_id": 1,"id": "cdbe20b4-4f49-4a27-96d5-4dda0ec2134b","action": {"id": "00000000-0000-0000-0000-000000000001","tm_execute": "","type": ""},"flow_id": "883f40c7-7f16-48fe-a404-772d9b038808","type": "flow","source": {"target": "+821028286521","name": "","type": "tel"},"tm_update": "","tm_ringing": "","data": null,"recording_id": "00000000-0000-0000-0000-000000000000","chained_call_ids": [],"hangup_reason": "","webhook_uri": "https://endxhr87aa0bkge.m.pipedream.net","tm_progressing": "","conf_id": "00000000-0000-0000-0000-000000000000","tm_hangup": "","asterisk_id": "","status": "dialing","recording_ids": [],"destination": {"name": "","target": "+821021656521","type": "tel"},"channel_id": "8edc9ed7-2ec0-4027-a7b9-3d6eced18afd","tm_create": "2021-03-13 18:18:02.489462"}}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockWeb.EXPECT().SendWebhook(tt.expectWebhook).Return(nil)
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
