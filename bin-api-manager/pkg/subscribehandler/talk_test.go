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
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"
)

func Test_processEventTalk(t *testing.T) {

	tests := []struct {
		name string

		event        *sock.Event
		participants []*tkparticipant.Participant

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
					}
					data, _ := json.Marshal(chat)
					return data
				}(),
			},
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
					}
					data, _ := json.Marshal(chat)
					return data
				}(),
			},
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
						ID:         uuid.FromStringOrNil("g66d1da0-3ed7-11ef-9208-4bcc069917a3"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("ddb5213a-8003-11ec-84ca-9fa226fcda9f"),
					},
				},
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
					}
					data, _ := json.Marshal(chat)
					return data
				}(),
			},
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
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &subscribeHandler{
				zmqpubHandler: mockZMQ,
				reqHandler:    mockReq,
			}

			// Parse event data to get chat ID
			var chatMsg tkchat.WebhookMessage
			_ = json.Unmarshal(tt.event.Data, &chatMsg)

			// Mock TalkV1ParticipantList call
			mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), chatMsg.ID).Return(tt.participants, nil)

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

func Test_processEventTalkMessage(t *testing.T) {

	tests := []struct {
		name string

		event        *sock.Event
		participants []*tkparticipant.Participant

		expectTopics []string
	}{
		{
			name: "chatmessage_created with single participant (creator only)",

			event: &sock.Event{
				Type:      tkmessage.EventTypeMessageCreated,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					msg := &tkmessage.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("a11d1da0-3ed7-11ef-9208-4bcc069917a1"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
						},
						ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
						Type:   tkmessage.TypeNormal,
						Text:   "Hello world",
					}
					data, _ := json.Marshal(msg)
					return data
				}(),
			},
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
				// Creator - Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatmessage:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Creator - New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// No additional topics - creator is the only participant, so skip is triggered
			},
		},
		{
			name: "chatmessage_created with multiple participants (creator skipped in participant loop)",

			event: &sock.Event{
				Type:      tkmessage.EventTypeMessageCreated,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					msg := &tkmessage.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("a11d1da0-3ed7-11ef-9208-4bcc069917a1"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
						},
						ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
						Type:   tkmessage.TypeNormal,
						Text:   "Hello team",
					}
					data, _ := json.Marshal(msg)
					return data
				}(),
			},
			participants: []*tkparticipant.Participant{
				{
					// Creator - should be skipped in participant loop
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
					// Other participant - should get topics
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
				// Creator - Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatmessage:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Creator - New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Other participant - Old format
				"agent_id:ddb5213a-8003-11ec-84ca-9fa226fcda9f:chatmessage:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Other participant - New format
				"agent_id:ddb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
		},
		{
			name: "chatmessage_deleted event",

			event: &sock.Event{
				Type:      tkmessage.EventTypeMessageDeleted,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					msg := &tkmessage.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("a11d1da0-3ed7-11ef-9208-4bcc069917a1"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
						},
						ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
						Type:   tkmessage.TypeNormal,
					}
					data, _ := json.Marshal(msg)
					return data
				}(),
			},
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
				// Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatmessage:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatmessage_deleted:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
		},
		{
			name: "chatmessage_reaction_updated event",

			event: &sock.Event{
				Type:      tkmessage.EventTypeMessageReactionUpdated,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					msg := &tkmessage.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("a11d1da0-3ed7-11ef-9208-4bcc069917a1"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
						},
						ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
						Type:   tkmessage.TypeNormal,
					}
					data, _ := json.Marshal(msg)
					return data
				}(),
			},
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
				// Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatmessage:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatmessage_reaction_updated:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
		},
		{
			name: "chatmessage_created with nil chat_id skips participant lookup",

			event: &sock.Event{
				Type:      tkmessage.EventTypeMessageCreated,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					msg := &tkmessage.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("a11d1da0-3ed7-11ef-9208-4bcc069917a1"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
						},
						ChatID: uuid.Nil,
						Type:   tkmessage.TypeNormal,
						Text:   "Message without chat",
					}
					data, _ := json.Marshal(msg)
					return data
				}(),
			},
			participants: nil,

			expectTopics: []string{
				// Creator only - Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatmessage:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Creator only - New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
		},
		{
			name: "chatmessage_created with participant lookup failure continues with creator topics",

			event: &sock.Event{
				Type:      tkmessage.EventTypeMessageCreated,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					msg := &tkmessage.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("a11d1da0-3ed7-11ef-9208-4bcc069917a1"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
						},
						ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
						Type:   tkmessage.TypeNormal,
						Text:   "Message with failed lookup",
					}
					data, _ := json.Marshal(msg)
					return data
				}(),
			},
			participants: nil, // signal to mock an error

			expectTopics: []string{
				// Creator only - Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatmessage:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				// Creator only - New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
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

			// Parse event data to get message info
			var msg tkmessage.WebhookMessage
			_ = json.Unmarshal(tt.event.Data, &msg)

			// Mock TalkV1ParticipantList call only when ChatID is not nil
			if msg.ChatID != uuid.Nil {
				if tt.participants == nil {
					// Mock an error for the participant lookup failure test
					mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), msg.ChatID).Return(nil, fmt.Errorf("rpc error"))
				} else {
					mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), msg.ChatID).Return(tt.participants, nil)
				}
			}

			for _, topic := range tt.expectTopics {
				mockZMQ.EXPECT().Publish(topic, gomock.Any()).Return(nil)
			}

			ctx := context.Background()

			if err := h.processEventTalkMessage(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventTalkParticipant(t *testing.T) {

	tests := []struct {
		name string

		event *sock.Event

		expectTopics []string
	}{
		{
			name: "chatparticipant_added event",

			event: &sock.Event{
				Type:      tkparticipant.EventParticipantAdded,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					p := &tkparticipant.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
						},
						ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
					}
					data, _ := json.Marshal(p)
					return data
				}(),
			},

			expectTopics: []string{
				// Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatparticipant:f66d1da0-3ed7-11ef-9208-4bcc069917a2",
				// New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatparticipant_added:f66d1da0-3ed7-11ef-9208-4bcc069917a2",
			},
		},
		{
			name: "chatparticipant_removed event",

			event: &sock.Event{
				Type:      tkparticipant.EventParticipantRemoved,
				Publisher: "talk-manager",
				DataType:  "application/json",
				Data: func() json.RawMessage {
					p := &tkparticipant.WebhookMessage{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
							CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
						},
						ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
					}
					data, _ := json.Marshal(p)
					return data
				}(),
			},

			expectTopics: []string{
				// Old format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatparticipant:f66d1da0-3ed7-11ef-9208-4bcc069917a2",
				// New format
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatparticipant_removed:f66d1da0-3ed7-11ef-9208-4bcc069917a2",
			},
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

			for _, topic := range tt.expectTopics {
				mockZMQ.EXPECT().Publish(topic, gomock.Any()).Return(nil)
			}

			ctx := context.Background()

			if err := h.processEventTalkParticipant(ctx, tt.event); err != nil {
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
			name:      "chat_deleted",
			eventType: "chat_deleted",
			expected:  "chat",
		},
		{
			name:      "chatmessage_created",
			eventType: "chatmessage_created",
			expected:  "chatmessage",
		},
		{
			name:      "chatmessage_deleted",
			eventType: "chatmessage_deleted",
			expected:  "chatmessage",
		},
		{
			name:      "chatmessage_reaction_updated",
			eventType: "chatmessage_reaction_updated",
			expected:  "chatmessage",
		},
		{
			name:      "chatparticipant_added",
			eventType: "chatparticipant_added",
			expected:  "chatparticipant",
		},
		{
			name:      "chatparticipant_removed",
			eventType: "chatparticipant_removed",
			expected:  "chatparticipant",
		},
		{
			name:      "single word",
			eventType: "event",
			expected:  "event",
		},
		{
			name:      "empty string",
			eventType: "",
			expected:  "",
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
			name:       "chatmessage_created with valid owner",
			eventType:  "chatmessage_created",
			ownerID:    uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			resourceID: uuid.FromStringOrNil("a11d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expected: []string{
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatmessage:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatmessage_created:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
			},
		},
		{
			name:       "chatparticipant_added with valid owner",
			eventType:  "chatparticipant_added",
			ownerID:    uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			resourceID: uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
			expected: []string{
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatparticipant:f66d1da0-3ed7-11ef-9208-4bcc069917a2",
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatparticipant_added:f66d1da0-3ed7-11ef-9208-4bcc069917a2",
			},
		},
		{
			name:       "chatmessage_reaction_updated with valid owner",
			eventType:  "chatmessage_reaction_updated",
			ownerID:    uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			resourceID: uuid.FromStringOrNil("a11d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expected: []string{
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:chatmessage:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
				"agent_id:cdb5213a-8003-11ec-84ca-9fa226fcda9f:talk:chatmessage_reaction_updated:a11d1da0-3ed7-11ef-9208-4bcc069917a1",
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
