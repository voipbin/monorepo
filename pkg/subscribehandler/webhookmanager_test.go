package subscribehandler

import (
	"context"
	"encoding/json"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/zmqpubhandler"
)

func Test_processEventWebhookManagerWebhookPublished(t *testing.T) {

	type test struct {
		name string

		request *rabbitmqhandler.Event

		expectTopic string
		expectEvent string
	}

	tests := []test{
		{
			"normal",

			&rabbitmqhandler.Event{
				Type:      "webhook_published",
				Publisher: "webhook-manager",
				DataType:  "application/json",
				Data: json.RawMessage([]byte(`{
					"data": {
					  "data": {
						"forward_action_id": "00000000-0000-0000-0000-000000000000",
						"current_action": {
						  "type": "",
						  "next_id": "00000000-0000-0000-0000-000000000000",
						  "id": "00000000-0000-0000-0000-000000000001"
						},
						"reference_type": "call",
						"id": "d5d70d9d-85ad-4ffc-90c1-fdda37c046b0",
						"flow_id": "d78245f0-8239-4298-b27b-1e3fc2185ba0",
						"tm_create": "2022-07-23 15:02:35.763577",
						"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
						"tm_update": "2022-07-23 15:02:35.763577",
						"reference_id": "167e552a-a739-473d-b7a7-245b258a4af0",
						"tm_delete": "9999-01-01 00:00:00.000000"
					  },
					  "type": "activeflow_created"
					},
					"data_type": "application/json",
					"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b"
				  }`)),
			},

			"5e4a0680-804e-11ec-8477-2fea5968d85b:activeflow:d5d70d9d-85ad-4ffc-90c1-fdda37c046b0",
			`{"data":{"current_action":{"id":"00000000-0000-0000-0000-000000000001","next_id":"00000000-0000-0000-0000-000000000000","type":""},"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","flow_id":"d78245f0-8239-4298-b27b-1e3fc2185ba0","forward_action_id":"00000000-0000-0000-0000-000000000000","id":"d5d70d9d-85ad-4ffc-90c1-fdda37c046b0","reference_id":"167e552a-a739-473d-b7a7-245b258a4af0","reference_type":"call","tm_create":"2022-07-23 15:02:35.763577","tm_delete":"9999-01-01 00:00:00.000000","tm_update":"2022-07-23 15:02:35.763577"},"type":"activeflow_created"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockZMQ := zmqpubhandler.NewMockZMQPubHandler(mc)

			h := &subscribeHandler{
				zmqpubHandler: mockZMQ,
			}

			mockZMQ.EXPECT().Publish(tt.expectTopic, tt.expectEvent)

			ctx := context.Background()

			if err := h.processEventWebhookManagerWebhookPublished(ctx, tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
