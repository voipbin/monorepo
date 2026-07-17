package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/zmqpubhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	requesthandler "monorepo/bin-common-handler/pkg/requesthandler"
	tkparticipant "monorepo/bin-talk-manager/models/participant"
)

func Test_processEventWebhookManagerWebhookPublished(t *testing.T) {

	type test struct {
		name string

		request *sock.Event

		// For chat events that need participant fan-out
		chatID       uuid.UUID
		participants []*tkparticipant.Participant
		participantErr error

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
						"tm_create": "2022-07-23T15:02:35.763577Z",
						"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
						"tm_update": "2022-07-23T15:02:35.763577Z",
						"reference_id": "167e552a-a739-473d-b7a7-245b258a4af0",
						"tm_delete": "9999-01-01T00:00:00.000000Z"
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
			expectEvent: `{"data":{"current_action":{"id":"00000000-0000-0000-0000-000000000001","next_id":"00000000-0000-0000-0000-000000000000","type":""},"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","flow_id":"d78245f0-8239-4298-b27b-1e3fc2185ba0","forward_action_id":"00000000-0000-0000-0000-000000000000","id":"d5d70d9d-85ad-4ffc-90c1-fdda37c046b0","reference_id":"167e552a-a739-473d-b7a7-245b258a4af0","reference_type":"call","tm_create":"2022-07-23T15:02:35.763577Z","tm_delete":"9999-01-01T00:00:00.000000Z","tm_update":"2022-07-23T15:02:35.763577Z"},"type":"activeflow_created"}`,
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
		{
			name: "aimessage_created with aicall_id",

			request: &sock.Event{
				Type:      "webhook_published",
				Publisher: "webhook-manager",
				DataType:  "application/json",
				Data: json.RawMessage([]byte(`{
					"data": {
					  "data": {
						"id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
						"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
						"aicall_id": "c3d4e5f6-a1b2-7890-abcd-1234567890ef"
					  },
					  "type": "aimessage_created"
					},
					"data_type": "application/json",
					"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b"
				  }`)),
			},

			expectTopics: []string{
				// Old format uses aicall_id instead of aimessage_id
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:aicall:c3d4e5f6-a1b2-7890-abcd-1234567890ef",
				// New format (service-namespaced)
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:webhook:aimessage_created:a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			},
			expectEvent: `{"data":{"aicall_id":"c3d4e5f6-a1b2-7890-abcd-1234567890ef","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"},"type":"aimessage_created"}`,
		},
		{
			name: "aimessage_intermediate with aicall_id",

			request: &sock.Event{
				Type:      "webhook_published",
				Publisher: "webhook-manager",
				DataType:  "application/json",
				Data: json.RawMessage([]byte(`{
					"data": {
					  "data": {
						"id": "b2c3d4e5-f6a1-7890-abcd-234567890ef1",
						"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
						"aicall_id": "c3d4e5f6-a1b2-7890-abcd-1234567890ef"
					  },
					  "type": "aimessage_intermediate"
					},
					"data_type": "application/json",
					"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b"
				  }`)),
			},

			expectTopics: []string{
				// Old format uses aicall_id instead of aimessage_id
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:aicall:c3d4e5f6-a1b2-7890-abcd-1234567890ef",
				// New format (service-namespaced)
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:webhook:aimessage_intermediate:b2c3d4e5-f6a1-7890-abcd-234567890ef1",
			},
			expectEvent: `{"data":{"aicall_id":"c3d4e5f6-a1b2-7890-abcd-1234567890ef","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","id":"b2c3d4e5-f6a1-7890-abcd-234567890ef1"},"type":"aimessage_intermediate"}`,
		},
		{
			name: "chatmessage_created with participants",

			request: &sock.Event{
				Type:      "webhook_published",
				Publisher: "webhook-manager",
				DataType:  "application/json",
				Data: json.RawMessage([]byte(`{
					"data": {
					  "data": {
						"id": "a11d1da0-3ed7-11ef-9208-4bcc069917a1",
						"customer_id": "550e8400-e29b-41d4-a716-446655440000",
						"owner_id": "cdb5213a-8003-11ec-84ca-9fa226fcda9f",
						"chat_id": "e66d1da0-3ed7-11ef-9208-4bcc069917a1"
					  },
					  "type": "chatmessage_created"
					},
					"data_type": "application/json",
					"customer_id": "550e8400-e29b-41d4-a716-446655440000"
				  }`)),
			},

			chatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			participants: []*tkparticipant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b22d1da0-3ed7-11ef-9208-4bcc069917a3"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("ddb5213a-8003-11ec-84ca-9fa226fcda9f"),
					},
				},
			},

			expectTopics: []string{
				// Old format - customer chat topic
				"customer_id:550e8400-e29b-41d4-a716-446655440000:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Old format - owner chat topic
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Fan-out other participant - old format
				"agent_id:ddb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Fan-out other participant - new format
				"agent_id:ddb5213a-8003-11ec-84ca-9fa226fcda9f:webhook:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format - customer
				"customer_id:550e8400-e29b-41d4-a716-446655440000:webhook:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format - owner
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:webhook:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
			expectEvent: `{"data":{"chat_id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1","customer_id":"550e8400-e29b-41d4-a716-446655440000","id":"a11d1da0-3ed7-11ef-9208-4bcc069917a1","owner_id":"cdb5213a-8003-11ec-84ca-9fa226fcda9f"},"type":"chatmessage_created"}`,
		},
		{
			name: "chat_created with participants (chatID fallback to d.ID)",

			request: &sock.Event{
				Type:      "webhook_published",
				Publisher: "webhook-manager",
				DataType:  "application/json",
				Data: json.RawMessage([]byte(`{
					"data": {
					  "data": {
						"id": "e66d1da0-3ed7-11ef-9208-4bcc069917a1",
						"customer_id": "550e8400-e29b-41d4-a716-446655440000"
					  },
					  "type": "chat_created"
					},
					"data_type": "application/json",
					"customer_id": "550e8400-e29b-41d4-a716-446655440000"
				  }`)),
			},

			chatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			participants: []*tkparticipant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					},
				},
			},

			expectTopics: []string{
				// Old format - customer chat topic
				"customer_id:550e8400-e29b-41d4-a716-446655440000:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Fan-out participant - old format (d.OwnerID is nil, so all participants get topics)
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Fan-out participant - new format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:webhook:chat_created:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format - customer
				"customer_id:550e8400-e29b-41d4-a716-446655440000:webhook:chat_created:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
			expectEvent: `{"data":{"customer_id":"550e8400-e29b-41d4-a716-446655440000","id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1"},"type":"chat_created"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockZMQ := zmqpubhandler.NewMockZMQPubHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &subscribeHandler{
				zmqpubHandler: mockZMQ,
				reqHandler:    mockReq,
			}

			// Set up participant list mock for chat events
			if tt.chatID != uuid.Nil {
				if tt.participantErr != nil {
					mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), tt.chatID).Return(nil, tt.participantErr)
				} else {
					mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), tt.chatID).Return(tt.participants, nil)
				}
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

		// For chat events that need participant fan-out
		chatID       uuid.UUID
		participants []*tkparticipant.Participant
		participantErr error

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
			name:        "aimessage_created with aicall_id uses aicall topic",
			messageType: "aimessage_created",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				AIcallID: uuid.FromStringOrNil("c3d4e5f6-a1b2-7890-abcd-1234567890ef"),
			},
			expectTopics: []string{
				// Old format uses aicall_id
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:aicall:c3d4e5f6-a1b2-7890-abcd-1234567890ef",
				// New format
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:test-manager:aimessage_created:a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			},
			expectError: false,
		},
		{
			name:        "aimessage_intermediate with aicall_id uses aicall topic",
			messageType: "aimessage_intermediate",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2c3d4e5-f6a1-7890-abcd-234567890ef1"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				AIcallID: uuid.FromStringOrNil("c3d4e5f6-a1b2-7890-abcd-1234567890ef"),
			},
			expectTopics: []string{
				// Old format uses aicall_id
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:aicall:c3d4e5f6-a1b2-7890-abcd-1234567890ef",
				// New format
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:test-manager:aimessage_intermediate:b2c3d4e5-f6a1-7890-abcd-234567890ef1",
			},
			expectError: false,
		},
		{
			name:        "webchat_message_created uses webchat_session topic scoped by session_id",
			messageType: "webchat_message_created",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e1e1e1e1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				SessionID: uuid.FromStringOrNil("f2f2f2f2-2222-2222-2222-222222222222"),
			},
			expectTopics: []string{
				// webchat case: scoped by session_id, not message id
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:webchat_session:f2f2f2f2-2222-2222-2222-222222222222",
				// New format still uses the message's own id
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:test-manager:webchat_message_created:e1e1e1e1-1111-1111-1111-111111111111",
			},
			expectError: false,
		},
		{
			name:        "webchat_session_ended uses webchat_session topic scoped by the session's own id",
			messageType: "webchat_session_ended",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					// A Session's own webhook payload has no separate
					// SessionID field -- its own Identity.ID IS the
					// session id (VOIP-1265 §9.3).
					ID:         uuid.FromStringOrNil("f2f2f2f2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
			},
			expectTopics: []string{
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:webchat_session:f2f2f2f2-2222-2222-2222-222222222222",
				"customer_id:5e4a0680-804e-11ec-8477-2fea5968d85b:test-manager:webchat_session_ended:f2f2f2f2-2222-2222-2222-222222222222",
			},
			expectError: false,
		},
		{
			name:        "chatmessage_created with chat_id and participants",
			messageType: "chatmessage_created",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a11d1da0-3ed7-11ef-9208-4bcc069917a1"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			},

			chatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			participants: []*tkparticipant.Participant{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
					},
					Owner: commonidentity.Owner{
						OwnerID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b22d1da0-3ed7-11ef-9208-4bcc069917a3"),
					},
					Owner: commonidentity.Owner{
						OwnerID: uuid.FromStringOrNil("ddb5213a-8003-11ec-84ca-9fa226fcda9f"),
					},
				},
			},

			expectTopics: []string{
				// Old format - customer chat
				"customer_id:550e8400-e29b-41d4-a716-446655440000:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Old format - owner chat
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Fan-out other participant - old format
				"agent_id:ddb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Fan-out other participant - new format
				"agent_id:ddb5213a-8003-11ec-84ca-9fa226fcda9f:test-manager:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format - customer
				"customer_id:550e8400-e29b-41d4-a716-446655440000:test-manager:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format - owner
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:test-manager:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
			expectError: false,
		},
		{
			name:        "chat_created with chatID nil fallback to d.ID",
			messageType: "chat_created",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
			},

			chatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			participants: []*tkparticipant.Participant{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
					},
					Owner: commonidentity.Owner{
						OwnerID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					},
				},
			},

			expectTopics: []string{
				// Old format - customer chat (chatID = d.ID)
				"customer_id:550e8400-e29b-41d4-a716-446655440000:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Fan-out participant - old format (d.OwnerID is nil, so no skip)
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Fan-out participant - new format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:test-manager:chat_created:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format - customer
				"customer_id:550e8400-e29b-41d4-a716-446655440000:test-manager:chat_created:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
			expectError: false,
		},
		{
			name:        "chatparticipant_added with chat_id",
			messageType: "chatparticipant_added",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			},

			chatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			participants: []*tkparticipant.Participant{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
					},
					Owner: commonidentity.Owner{
						OwnerID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b22d1da0-3ed7-11ef-9208-4bcc069917a3"),
					},
					Owner: commonidentity.Owner{
						OwnerID: uuid.FromStringOrNil("ddb5213a-8003-11ec-84ca-9fa226fcda9f"),
					},
				},
			},

			expectTopics: []string{
				// Old format - customer chat
				"customer_id:550e8400-e29b-41d4-a716-446655440000:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Old format - owner chat
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Fan-out other participant - old format
				"agent_id:ddb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Fan-out other participant - new format
				"agent_id:ddb5213a-8003-11ec-84ca-9fa226fcda9f:test-manager:chatparticipant_added:f66d1da0-3ed7-11ef-9208-4bcc069917a2",
				// New format - customer
				"customer_id:550e8400-e29b-41d4-a716-446655440000:test-manager:chatparticipant_added:f66d1da0-3ed7-11ef-9208-4bcc069917a2",
				// New format - owner
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:test-manager:chatparticipant_added:f66d1da0-3ed7-11ef-9208-4bcc069917a2",
			},
			expectError: false,
		},
		{
			name:        "chatmessage_created with participant lookup failure",
			messageType: "chatmessage_created",
			data: &commonWebhookData{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a11d1da0-3ed7-11ef-9208-4bcc069917a1"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			},

			chatID:         uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			participantErr: fmt.Errorf("rpc error"),

			expectTopics: []string{
				// Old format - customer chat
				"customer_id:550e8400-e29b-41d4-a716-446655440000:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Old format - owner chat
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// No fan-out topics (participant lookup failed)
				// New format - customer
				"customer_id:550e8400-e29b-41d4-a716-446655440000:test-manager:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format - owner
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:test-manager:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &subscribeHandler{
				reqHandler: mockReq,
			}

			// Set up participant list mock for chat events
			if tt.chatID != uuid.Nil {
				if tt.participantErr != nil {
					mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), tt.chatID).Return(nil, tt.participantErr)
				} else {
					mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), tt.chatID).Return(tt.participants, nil)
				}
			}

			ctx := context.Background()

			topics, err := h.createTopics(ctx, tt.messageType, tt.data, "test-manager")
			if (err != nil) != tt.expectError {
				t.Errorf("Unexpected error. expect: %v, got: %v", tt.expectError, err)
			}

			if len(topics) != len(tt.expectTopics) {
				t.Errorf("Unexpected number of topics. expect: %d, got: %d\nexpect: %v\ngot: %v", len(tt.expectTopics), len(topics), tt.expectTopics, topics)
			}

			for i, topic := range topics {
				if topic != tt.expectTopics[i] {
					t.Errorf("Unexpected topic at index %d. expect: %s, got: %s", i, tt.expectTopics[i], topic)
				}
			}
		})
	}
}

// Test_processEventWebhookManagerRoutingKeyedEvent is a regression test for a production bug
// found 2026-07-15 (post-envelope-fix verification): events arriving via the NEW topic exchange
// (m.Publisher == "webhook-manager", m.Type == the REAL resource event type e.g. "call_created")
// were silently discarded by processEvent's switch statement, which only matched
// m.Type == "webhook_published" (the OLD fanout path's fixed constant). The AMQP message reached
// this pod's queue correctly, but was never handed to zmqpubHandler.Publish, so it never reached
// a connected websocket client. This test asserts the new case actually publishes the topics a
// real websocket client would have subscribed to (the OLD-FORMAT prefix, e.g.
// "customer_id:<id>:call", which zmqSub.Subscribe uses as a ZMQ prefix filter).
func Test_processEventWebhookManagerRoutingKeyedEvent(t *testing.T) {
	customerID := uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b")
	ownerID := uuid.FromStringOrNil("62005165-7592-4ff7-9076-55bf491023f2")
	callID := uuid.FromStringOrNil("d499c4f4-2f07-488b-a5f7-4b49c00e9a2a")

	// m.Data here is the ALREADY-UNWRAPPED resource object (bin-webhook-manager's
	// publishRoutingKeyedEvent does the envelope unwrap at publish time) -- NOT the doubly-nested
	// {"type":...,"data":...} envelope that processEventWebhookManagerWebhookPublished expects.
	data := json.RawMessage(fmt.Sprintf(
		`{"id":"%s","customer_id":"%s","owner_id":"%s","owner_type":"agent"}`,
		callID, customerID, ownerID,
	))

	event := &sock.Event{
		Type:      "call_created",
		Publisher: "webhook-manager",
		DataType:  "application/json",
		Data:      data,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockZMQ := zmqpubhandler.NewMockZMQPubHandler(mc)
	h := &subscribeHandler{zmqpubHandler: mockZMQ}

	expectedOldFormatCustomerTopic := fmt.Sprintf("customer_id:%s:call:%s", customerID, callID)
	expectedOldFormatOwnerTopic := fmt.Sprintf("agent_id:%s:call:%s", ownerID, callID)

	mockZMQ.EXPECT().Publish(expectedOldFormatCustomerTopic, string(data)).Return(nil)
	mockZMQ.EXPECT().Publish(expectedOldFormatOwnerTopic, string(data)).Return(nil)
	// NEW format topics also get published (createTopics always emits both) -- accept any
	// remaining Publish calls for those, this test's focus is the OLD-FORMAT compatibility.
	mockZMQ.EXPECT().Publish(gomock.Any(), string(data)).Return(nil).AnyTimes()

	err := h.processEventWebhookManagerRoutingKeyedEvent(context.Background(), event)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// Test_processEventWebhookManagerRoutingKeyedEvent_WrongEventTypeFormat verifies the error path:
// an event type with no underscore segment cannot be split into a resource/verb pair and must
// return an error rather than silently producing zero topics.
func Test_processEventWebhookManagerRoutingKeyedEvent_WrongEventTypeFormat(t *testing.T) {
	event := &sock.Event{
		Type:      "malformed",
		Publisher: "webhook-manager",
		DataType:  "application/json",
		Data:      json.RawMessage(`{"id":"9d0966c0-da98-11ee-97df-1786497422fb"}`),
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockZMQ := zmqpubhandler.NewMockZMQPubHandler(mc)
	h := &subscribeHandler{zmqpubHandler: mockZMQ}
	// No Publish call expected at all.

	err := h.processEventWebhookManagerRoutingKeyedEvent(context.Background(), event)
	if err == nil {
		t.Error("Expected an error for a malformed event type, got nil")
	}
}
