package subscribehandler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/zmqpubhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
)

func Test_processEventWebhookManagerWebhookPublished(t *testing.T) {

	type test struct {
		name string

		request *sock.Event

		expectTopics []string
		expectEvent  string
	}

	tests := []test{
		{
			name: "customer level normal",

			request: &sock.Event{
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

			expectTopics: []string{
				// Old format (backward compatible)
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:activeflow:d5d70d9d-85ad-4ffc-90c1-fdda37c046b0",
				// New format (service-namespaced)
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:webhook:activeflow_created:d5d70d9d-85ad-4ffc-90c1-fdda37c046b0",
			},
			expectEvent: `{"data":{"current_action":{"id":"00000000-0000-0000-0000-000000000001","next_id":"00000000-0000-0000-0000-000000000000","type":""},"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","flow_id":"d78245f0-8239-4298-b27b-1e3fc2185ba0","forward_action_id":"00000000-0000-0000-0000-000000000000","id":"d5d70d9d-85ad-4ffc-90c1-fdda37c046b0","reference_id":"167e552a-a739-473d-b7a7-245b258a4af0","reference_type":"call","tm_create":"2022-07-23 15:02:35.763577","tm_delete":"9999-01-01 00:00:00.000000","tm_update":"2022-07-23 15:02:35.763577"},"type":"activeflow_created"}`,
		},
		{
			name: "owner level normal",

			request: &sock.Event{
				Type:      "webhook_published",
				Publisher: "webhook-manager",
				DataType:  "application/json",
				Data: json.RawMessage([]byte(`{
					"data": {
					  "data": {
						"id": "7b0966c0-da98-11ee-97df-1786497422fb",
						"customer_id": "7b6cc58a-da98-11ee-b114-3fb0f4ae9318",
						"owner_id": "7b3d4468-da98-11ee-a5f0-e32e2e370da3"
					  },
					  "type": "chatroom_created"
					},
					"data_type": "application/json",
					"customer_id": "7b6cc58a-da98-11ee-b114-3fb0f4ae9318"
				  }`)),
			},

			expectTopics: []string{
				// Old format (backward compatible)
				"customer_id:7b6cc58a-da98-11ee-b114-3fb0f4ae9318:chatroom:7b0966c0-da98-11ee-97df-1786497422fb",
				"agent_id:7b3d4468-da98-11ee-a5f0-e32e2e370da3:chatroom:7b0966c0-da98-11ee-97df-1786497422fb",
				// New format (service-namespaced)
				"customer_id:7b6cc58a-da98-11ee-b114-3fb0f4ae9318:webhook:chatroom_created:7b0966c0-da98-11ee-97df-1786497422fb",
				"agent_id:7b3d4468-da98-11ee-a5f0-e32e2e370da3:webhook:chatroom_created:7b0966c0-da98-11ee-97df-1786497422fb",
			},
			expectEvent: `{"data":{"customer_id":"7b6cc58a-da98-11ee-b114-3fb0f4ae9318","id":"7b0966c0-da98-11ee-97df-1786497422fb","owner_id":"7b3d4468-da98-11ee-a5f0-e32e2e370da3"},"type":"chatroom_created"}`,
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

			for _, topic := range tt.expectTopics {
				mockZMQ.EXPECT().Publish(topic, tt.expectEvent)
			}

			ctx := context.Background()

			if err := h.processEventWebhookManagerWebhookPublished(ctx, tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_createTopics(t *testing.T) {

	type test struct {
		name string

		messageType string
		data        *commonWebhookData

		expectTopics []string
		expectError  bool
	}

	tests := []test{
		{
			name:        "have id, customer_id",
			messageType: "activeflow_created",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d5d70d9d-85ad-4ffc-90c1-fdda37c046b0"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
			},
			expectTopics: []string{
				// Old format
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:activeflow:d5d70d9d-85ad-4ffc-90c1-fdda37c046b0",
				// New format
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:test-manager:activeflow_created:d5d70d9d-85ad-4ffc-90c1-fdda37c046b0",
			},
			expectError: false,
		},
		{
			name:        "have id, customer_id, owner_id",
			messageType: "chatroom_created",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7b0966c0-da98-11ee-97df-1786497422fb"),
					CustomerID: uuid.FromStringOrNil("7b6cc58a-da98-11ee-b114-3fb0f4ae9318"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("7b3d4468-da98-11ee-a5f0-e32e2e370da3"),
				},
			},
			expectTopics: []string{
				// Old format
				"customer_id:7b6cc58a-da98-11ee-b114-3fb0f4ae9318:chatroom:7b0966c0-da98-11ee-97df-1786497422fb",
				"agent_id:7b3d4468-da98-11ee-a5f0-e32e2e370da3:chatroom:7b0966c0-da98-11ee-97df-1786497422fb",
				// New format
				"customer_id:7b6cc58a-da98-11ee-b114-3fb0f4ae9318:test-manager:chatroom_created:7b0966c0-da98-11ee-97df-1786497422fb",
				"agent_id:7b3d4468-da98-11ee-a5f0-e32e2e370da3:test-manager:chatroom_created:7b0966c0-da98-11ee-97df-1786497422fb",
			},
			expectError: false,
		},
		{
			name:        "have id, owner_id",
			messageType: "task_created",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8c0966c0-da98-11ee-97df-1786497422fb"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("8c3d4468-da98-11ee-a5f0-e32e2e370da3"),
				},
			},
			expectTopics: []string{
				// Old format
				"agent_id:8c3d4468-da98-11ee-a5f0-e32e2e370da3:task:8c0966c0-da98-11ee-97df-1786497422fb",
				// New format
				"agent_id:8c3d4468-da98-11ee-a5f0-e32e2e370da3:test-manager:task_created:8c0966c0-da98-11ee-97df-1786497422fb",
			},
			expectError: false,
		},
		{
			name:        "invalid message type",
			messageType: "invalidtype",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9d0966c0-da98-11ee-97df-1786497422fb"),
					CustomerID: uuid.FromStringOrNil("9d6cc58a-da98-11ee-b114-3fb0f4ae9318"),
				},
			},
			expectTopics: []string{},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &subscribeHandler{}

			topics, err := h.createTopics(tt.messageType, tt.data, "test-manager")
			if (err != nil) != tt.expectError {
				t.Errorf("Unexpected error. expect: %v, got: %v", tt.expectError, err)
			}

			if len(topics) != len(tt.expectTopics) {
				t.Errorf("Unexpected number of topics. expect: %d, got: %d", len(tt.expectTopics), len(topics))
			}

			for i, topic := range topics {
				if topic != tt.expectTopics[i] {
					t.Errorf("Unexpected topic. expect: %s, got: %s", tt.expectTopics[i], topic)
				}
			}
		})
	}
}
