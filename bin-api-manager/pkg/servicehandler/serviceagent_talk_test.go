package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_canAccessChat(t *testing.T) {
	tests := []struct {
		name string

		agentID    uuid.UUID
		customerID uuid.UUID
		chatID     uuid.UUID

		responseChat         *tkchat.Chat
		responseParticipants []*tkparticipant.Participant
		expectResult         bool
	}{
		{
			name:       "public talk type - same customer - should allow access",
			agentID:    uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
			customerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
			chatID:     uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
				},
				Type: tkchat.TypeTalk,
			},
			responseParticipants: nil, // TypeTalk doesn't need participants
			expectResult:         true,
		},
		{
			name:       "public talk type - different customer - should deny access",
			agentID:    uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222"),
			customerID: uuid.FromStringOrNil("c2222222-2222-2222-2222-222222222222"),
			chatID:     uuid.FromStringOrNil("d2222222-2222-2222-2222-222222222222"),
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d2222222-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c9999999-9999-9999-9999-999999999999"), // different customer
				},
				Type: tkchat.TypeTalk,
			},
			responseParticipants: []*tkparticipant.Participant{}, // Different customer requires participant check
			expectResult:         false,
		},
		{
			name:       "group type - agent is participant - should allow access",
			agentID:    uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"),
			customerID: uuid.FromStringOrNil("c3333333-3333-3333-3333-333333333333"),
			chatID:     uuid.FromStringOrNil("d3333333-3333-3333-3333-333333333333"),
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d3333333-3333-3333-3333-333333333333"),
					CustomerID: uuid.FromStringOrNil("c3333333-3333-3333-3333-333333333333"),
				},
				Type: tkchat.TypeGroup,
			},
			responseParticipants: []*tkparticipant.Participant{
				{
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"),
					},
				},
			},
			expectResult: true,
		},
		{
			name:       "group type - agent is not participant - should deny access",
			agentID:    uuid.FromStringOrNil("a4444444-4444-4444-4444-444444444444"),
			customerID: uuid.FromStringOrNil("c4444444-4444-4444-4444-444444444444"),
			chatID:     uuid.FromStringOrNil("d4444444-4444-4444-4444-444444444444"),
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d4444444-4444-4444-4444-444444444444"),
					CustomerID: uuid.FromStringOrNil("c4444444-4444-4444-4444-444444444444"),
				},
				Type: tkchat.TypeGroup,
			},
			responseParticipants: []*tkparticipant.Participant{
				{
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("a9999999-9999-9999-9999-999999999999"), // different agent
					},
				},
			},
			expectResult: false,
		},
		{
			name:       "direct type - agent is participant - should allow access",
			agentID:    uuid.FromStringOrNil("a5555555-5555-5555-5555-555555555555"),
			customerID: uuid.FromStringOrNil("c5555555-5555-5555-5555-555555555555"),
			chatID:     uuid.FromStringOrNil("d5555555-5555-5555-5555-555555555555"),
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d5555555-5555-5555-5555-555555555555"),
					CustomerID: uuid.FromStringOrNil("c5555555-5555-5555-5555-555555555555"),
				},
				Type: tkchat.TypeDirect,
			},
			responseParticipants: []*tkparticipant.Participant{
				{
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("a5555555-5555-5555-5555-555555555555"),
					},
				},
			},
			expectResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TalkV1ChatGet(ctx, tt.chatID).Return(tt.responseChat, nil)

			// Mock TalkV1ParticipantList when not (TypeTalk AND same customer)
			if tt.responseChat.Type != tkchat.TypeTalk || tt.responseChat.CustomerID != tt.customerID {
				mockReq.EXPECT().TalkV1ParticipantList(ctx, tt.chatID).Return(tt.responseParticipants, nil)
			}

			res := h.canAccessChat(ctx, tt.agentID, tt.customerID, tt.chatID)
			if res != tt.expectResult {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectResult, res)
			}
		})
	}
}

func Test_isParticipantOfTalk(t *testing.T) {
	tests := []struct {
		name string

		agentID uuid.UUID
		chatID  uuid.UUID

		responseParticipants []*tkparticipant.Participant
		expectResult         bool
	}{
		{
			name:    "agent is participant - should return true",
			agentID: uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
			chatID:  uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
			responseParticipants: []*tkparticipant.Participant{
				{
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
					},
				},
			},
			expectResult: true,
		},
		{
			name:    "agent is not participant - should return false",
			agentID: uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222"),
			chatID:  uuid.FromStringOrNil("d2222222-2222-2222-2222-222222222222"),
			responseParticipants: []*tkparticipant.Participant{
				{
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("a9999999-9999-9999-9999-999999999999"), // different agent
					},
				},
			},
			expectResult: false,
		},
		{
			name:                 "no participants - should return false",
			agentID:              uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"),
			chatID:               uuid.FromStringOrNil("d3333333-3333-3333-3333-333333333333"),
			responseParticipants: []*tkparticipant.Participant{},
			expectResult:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TalkV1ParticipantList(ctx, tt.chatID).Return(tt.responseParticipants, nil)

			res := h.isParticipantOfTalk(ctx, tt.agentID, tt.chatID)
			if res != tt.expectResult {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectResult, res)
			}
		})
	}
}

func Test_ServiceAgentTalkChatList(t *testing.T) {
	tests := []struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseChats []*tkchat.Chat
		expectFilters map[string]any
		expectRes     []*tkchat.WebhookMessage
	}{
		{
			name: "normal - returns joined chats without participants",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
				},
			},
			size:  10,
			token: "2024-01-01T00:00:00.000000Z",
			responseChats: []*tkchat.Chat{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
						CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
					},
					Type:     tkchat.TypeGroup,
					Name:     "Test Chat",
					TMCreate: "2024-01-01T00:00:00.000000Z",
				},
			},
			expectFilters: map[string]any{
				"owner_type": "agent",
				"owner_id":   "a1111111-1111-1111-1111-111111111111",
				"deleted":    false,
			},
			expectRes: []*tkchat.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
						CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
					},
					Type:     tkchat.TypeGroup,
					Name:     "Test Chat",
					TMCreate: "2024-01-01T00:00:00.000000Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().TalkV1ChatList(ctx, tt.expectFilters, tt.token, tt.size).Return(tt.responseChats, nil)

			res, err := h.ServiceAgentTalkChatList(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentTalkChannelList(t *testing.T) {
	tests := []struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseChats []*tkchat.Chat
		expectFilters map[string]any
		expectRes     []*tkchat.WebhookMessage
	}{
		{
			name: "normal - returns public channels without participants",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
				},
			},
			size:  10,
			token: "2024-01-01T00:00:00.000000Z",
			responseChats: []*tkchat.Chat{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
						CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
					},
					Type:     tkchat.TypeTalk,
					Name:     "General Channel",
					TMCreate: "2024-01-01T00:00:00.000000Z",
				},
			},
			expectFilters: map[string]any{
				"customer_id": "c1111111-1111-1111-1111-111111111111",
				"type":        "talk",
				"deleted":     false,
			},
			expectRes: []*tkchat.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
						CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
					},
					Type:     tkchat.TypeTalk,
					Name:     "General Channel",
					TMCreate: "2024-01-01T00:00:00.000000Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().TalkV1ChatList(ctx, tt.expectFilters, tt.token, tt.size).Return(tt.responseChats, nil)

			res, err := h.ServiceAgentTalkChannelList(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_canAddParticipant(t *testing.T) {
	tests := []struct {
		name string

		agent     *amagent.Agent
		chatID    uuid.UUID
		ownerType string
		ownerID   uuid.UUID

		responseChat         *tkchat.Chat
		responseParticipants []*tkparticipant.Participant
		expectResult         bool
	}{
		{
			name: "existing participant adding another agent to group chat - should succeed",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
				},
			},
			chatID:    uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222"), // different agent
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
				},
				Type: tkchat.TypeGroup,
			},
			responseParticipants: []*tkparticipant.Participant{
				{
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"), // agent is participant
					},
				},
			},
			expectResult: true,
		},
		{
			name: "existing participant adding another agent to talk-type chat - should succeed",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"),
					CustomerID: uuid.FromStringOrNil("c3333333-3333-3333-3333-333333333333"),
				},
			},
			chatID:    uuid.FromStringOrNil("d3333333-3333-3333-3333-333333333333"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("a4444444-4444-4444-4444-444444444444"), // different agent
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d3333333-3333-3333-3333-333333333333"),
					CustomerID: uuid.FromStringOrNil("c3333333-3333-3333-3333-333333333333"),
				},
				Type: tkchat.TypeTalk,
			},
			responseParticipants: []*tkparticipant.Participant{
				{
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"), // agent is participant
					},
				},
			},
			expectResult: true,
		},
		{
			name: "non-participant agent adding themselves to talk-type chat - should succeed",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a5555555-5555-5555-5555-555555555555"),
					CustomerID: uuid.FromStringOrNil("c5555555-5555-5555-5555-555555555555"),
				},
			},
			chatID:    uuid.FromStringOrNil("d5555555-5555-5555-5555-555555555555"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("a5555555-5555-5555-5555-555555555555"), // same agent (self-join)
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d5555555-5555-5555-5555-555555555555"),
					CustomerID: uuid.FromStringOrNil("c5555555-5555-5555-5555-555555555555"),
				},
				Type: tkchat.TypeTalk,
			},
			responseParticipants: []*tkparticipant.Participant{},
			expectResult:         true,
		},
		{
			name: "non-participant agent trying to add someone else to talk-type chat - should fail",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a6666666-6666-6666-6666-666666666666"),
					CustomerID: uuid.FromStringOrNil("c6666666-6666-6666-6666-666666666666"),
				},
			},
			chatID:    uuid.FromStringOrNil("d6666666-6666-6666-6666-666666666666"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("a7777777-7777-7777-7777-777777777777"), // different agent
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d6666666-6666-6666-6666-666666666666"),
					CustomerID: uuid.FromStringOrNil("c6666666-6666-6666-6666-666666666666"),
				},
				Type: tkchat.TypeTalk,
			},
			responseParticipants: []*tkparticipant.Participant{},
			expectResult:         false,
		},
		{
			name: "non-participant agent trying to add anyone to group chat - should fail",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a8888888-8888-8888-8888-888888888888"),
					CustomerID: uuid.FromStringOrNil("c8888888-8888-8888-8888-888888888888"),
				},
			},
			chatID:    uuid.FromStringOrNil("d8888888-8888-8888-8888-888888888888"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("a9999999-9999-9999-9999-999999999999"), // different agent
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d8888888-8888-8888-8888-888888888888"),
					CustomerID: uuid.FromStringOrNil("c8888888-8888-8888-8888-888888888888"),
				},
				Type: tkchat.TypeGroup,
			},
			responseParticipants: []*tkparticipant.Participant{},
			expectResult:         false,
		},
		{
			name: "non-participant agent trying to add themselves to group chat - should fail",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
					CustomerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
				},
			},
			chatID:    uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), // same agent (self-join)
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd"),
					CustomerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
				},
				Type: tkchat.TypeGroup,
			},
			responseParticipants: []*tkparticipant.Participant{},
			expectResult:         false,
		},
		{
			name: "non-participant agent from different customer trying to join talk-type chat - should fail",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
					CustomerID: uuid.FromStringOrNil("c9999999-9999-9999-9999-999999999999"), // different customer
				},
			},
			chatID:    uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), // same agent
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
					CustomerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"), // different customer
				},
				Type: tkchat.TypeTalk,
			},
			responseParticipants: []*tkparticipant.Participant{},
			expectResult:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TalkV1ChatGet(ctx, tt.chatID).Return(tt.responseChat, nil)
			mockReq.EXPECT().TalkV1ParticipantList(ctx, tt.chatID).Return(tt.responseParticipants, nil)

			res := h.canAddParticipant(ctx, tt.agent, tt.chatID, tt.ownerType, tt.ownerID)
			if res != tt.expectResult {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectResult, res)
			}
		})
	}
}

func Test_ServiceAgentTalkParticipantCreate(t *testing.T) {
	tests := []struct {
		name string

		agent     *amagent.Agent
		chatID    uuid.UUID
		ownerType string
		ownerID   uuid.UUID

		responseChat         *tkchat.Chat
		responseParticipants []*tkparticipant.Participant
		responseParticipant  *tkparticipant.Participant
		expectError          bool
	}{
		{
			name: "existing participant adding another agent - should succeed",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
				},
			},
			chatID:    uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222"),
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
				},
				Type: tkchat.TypeGroup,
			},
			responseParticipants: []*tkparticipant.Participant{
				{
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"), // agent is participant
					},
				},
			},
			responseParticipant: &tkparticipant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("p2222222-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
				},
				ChatID: uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222"),
				},
			},
			expectError: false,
		},
		{
			name: "non-participant agent joining talk-type chat - should succeed",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"),
					CustomerID: uuid.FromStringOrNil("c3333333-3333-3333-3333-333333333333"),
				},
			},
			chatID:    uuid.FromStringOrNil("d3333333-3333-3333-3333-333333333333"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"), // self-join
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d3333333-3333-3333-3333-333333333333"),
					CustomerID: uuid.FromStringOrNil("c3333333-3333-3333-3333-333333333333"),
				},
				Type: tkchat.TypeTalk,
			},
			responseParticipants: []*tkparticipant.Participant{}, // Non-participant self-joining
			responseParticipant: &tkparticipant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("p3333333-3333-3333-3333-333333333333"),
					CustomerID: uuid.FromStringOrNil("c3333333-3333-3333-3333-333333333333"),
				},
				ChatID: uuid.FromStringOrNil("d3333333-3333-3333-3333-333333333333"),
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"),
				},
			},
			expectError: false,
		},
		{
			name: "non-participant agent trying to add someone else - should fail",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a4444444-4444-4444-4444-444444444444"),
					CustomerID: uuid.FromStringOrNil("c4444444-4444-4444-4444-444444444444"),
				},
			},
			chatID:    uuid.FromStringOrNil("d4444444-4444-4444-4444-444444444444"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("a5555555-5555-5555-5555-555555555555"), // different agent
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d4444444-4444-4444-4444-444444444444"),
					CustomerID: uuid.FromStringOrNil("c4444444-4444-4444-4444-444444444444"),
				},
				Type: tkchat.TypeGroup,
			},
			responseParticipants: []*tkparticipant.Participant{}, // Not a participant
			responseParticipant:  nil,
			expectError:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			// Mock the chat get call for permission check
			mockReq.EXPECT().TalkV1ChatGet(ctx, tt.chatID).Return(tt.responseChat, nil)
			mockReq.EXPECT().TalkV1ParticipantList(ctx, tt.chatID).Return(tt.responseParticipants, nil)

			// Only expect participant create call if not expecting error
			if !tt.expectError {
				mockReq.EXPECT().TalkV1ParticipantCreate(ctx, tt.chatID, tt.ownerType, tt.ownerID).Return(tt.responseParticipant, nil)
			}

			res, err := h.ServiceAgentTalkParticipantCreate(ctx, tt.agent, tt.chatID, tt.ownerType, tt.ownerID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			} else {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				if res == nil {
					t.Errorf("Wrong match. expect: participant, got: nil")
				}
			}
		})
	}
}
