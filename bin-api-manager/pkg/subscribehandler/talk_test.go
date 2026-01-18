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
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkparticipant "monorepo/bin-talk-manager/models/participant"
)

func Test_processEventTalk(t *testing.T) {

	tests := []struct {
		name string

		event *sock.Event

		expectTopics []string
	}{
		{
			name: "chat_created event",

			event: &sock.Event{
				Type:      tkchat.EventTypeChatCreated,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					chat := &tkchat.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Type: tkchat.TypeDirect,
						Participants: []*tkparticipant.WebhookMessage{
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
					}
					data, _ := json.Marshal(chat)
					return data
				}(),
			},

			expectTopics: []string{
				// Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chat_created:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
		},
		{
			name: "chat_updated event",

			event: &sock.Event{
				Type:      tkchat.EventTypeChatUpdated,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					chat := &tkchat.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Type:   tkchat.TypeGroup,
						Name:   "Updated Name",
						Detail: "Updated Detail",
						Participants: []*tkparticipant.WebhookMessage{
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
									ID:         uuid.FromStringOrNil("g66d1da0-3ed7-11ef-9208-4bcc069917a3"),
									CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
								},
								Owner: commonidentity.Owner{
									OwnerType: "agent",
									OwnerID:   uuid.FromStringOrNil("ddb5213a-8003-11ec-84ca-9fa226fcda9f"),
								},
							},
						},
					}
					data, _ := json.Marshal(chat)
					return data
				}(),
			},

			expectTopics: []string{
				// Participant 1 - Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Participant 1 - New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chat_updated:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Participant 2 - Old format
				"agent_id:ddb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Participant 2 - New format
				"agent_id:ddb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chat_updated:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
		},
		{
			name: "chat_deleted event",

			event: &sock.Event{
				Type:      tkchat.EventTypeChatDeleted,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					chat := &tkchat.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Type: tkchat.TypeDirect,
						Participants: []*tkparticipant.WebhookMessage{
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
					}
					data, _ := json.Marshal(chat)
					return data
				}(),
			},

			expectTopics: []string{
				// Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chat_deleted:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
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
				mockZMQ.EXPECT().Publish(topic, gomock.Any()).Return(nil)
			}

			ctx := context.Background()

			if err := h.processEventTalk(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_extractResource(t *testing.T) {

	tests := []struct {
		name      string
		eventType string
		expected  string
	}{
		{
			name:      "chat_created",
			eventType: "chat_created",
			expected:  "chat",
		},
		{
			name:      "chat_updated",
			eventType: "chat_updated",
			expected:  "chat",
		},
		{
			name:      "message_created",
			eventType: "message_created",
			expected:  "message",
		},
		{
			name:      "participant_added",
			eventType: "participant_added",
			expected:  "participant",
		},
		{
			name:      "single word",
			eventType: "event",
			expected:  "event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &subscribeHandler{}

			result := h.extractResource(tt.eventType)
			if result != tt.expected {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expected, result)
			}
		})
	}
}

func Test_createTalkTopics(t *testing.T) {

	tests := []struct {
		name       string
		eventType  string
		ownerID    uuid.UUID
		resourceID uuid.UUID
		expected   []string
	}{
		{
			name:       "chat_updated with valid owner",
			eventType:  "chat_updated",
			ownerID:    uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			resourceID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expected: []string{
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chat:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chat_updated:e66d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
		},
		{
			name:       "nil owner returns empty",
			eventType:  "chat_updated",
			ownerID:    uuid.Nil,
			resourceID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expected:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &subscribeHandler{}

			result := h.createTalkTopics(tt.eventType, tt.ownerID, tt.resourceID)

			if len(result) != len(tt.expected) {
				t.Errorf("Wrong number of topics. expect: %d, got: %d", len(tt.expected), len(result))
				return
			}

			for i, topic := range result {
				if topic != tt.expected[i] {
					t.Errorf("Wrong topic at index %d. expect: %s, got: %s", i, tt.expected[i], topic)
				}
			}
		})
	}
}
