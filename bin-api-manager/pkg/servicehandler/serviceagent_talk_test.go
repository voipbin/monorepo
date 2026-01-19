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

		responseChat *tkchat.Chat
		expectResult bool
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
				Type:         tkchat.TypeTalk,
				Participants: []*tkparticipant.Participant{},
			},
			expectResult: true,
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
				Type:         tkchat.TypeTalk,
				Participants: []*tkparticipant.Participant{},
			},
			expectResult: false,
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
				Participants: []*tkparticipant.Participant{
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("p3333333-3333-3333-3333-333333333333"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"),
						},
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
				Participants: []*tkparticipant.Participant{
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("p4444444-4444-4444-4444-444444444444"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("a9999999-9999-9999-9999-999999999999"), // different agent
						},
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
				Participants: []*tkparticipant.Participant{
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("p5555555-5555-5555-5555-555555555555"),
						},
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("a5555555-5555-5555-5555-555555555555"),
						},
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

		responseChat *tkchat.Chat
		expectResult bool
	}{
		{
			name:    "agent is participant - should return true",
			agentID: uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
			chatID:  uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
				},
				Participants: []*tkparticipant.Participant{
					{
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
						},
					},
				},
			},
			expectResult: true,
		},
		{
			name:    "agent is not participant - should return false",
			agentID: uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222"),
			chatID:  uuid.FromStringOrNil("d2222222-2222-2222-2222-222222222222"),
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d2222222-2222-2222-2222-222222222222"),
				},
				Participants: []*tkparticipant.Participant{
					{
						Owner: commonidentity.Owner{
							OwnerType: "agent",
							OwnerID:   uuid.FromStringOrNil("a9999999-9999-9999-9999-999999999999"),
						},
					},
				},
			},
			expectResult: false,
		},
		{
			name:    "no participants - should return false",
			agentID: uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"),
			chatID:  uuid.FromStringOrNil("d3333333-3333-3333-3333-333333333333"),
			responseChat: &tkchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d3333333-3333-3333-3333-333333333333"),
				},
				Participants: []*tkparticipant.Participant{},
			},
			expectResult: false,
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
			token: "2024-01-01 00:00:00.000000",
			responseChats: []*tkchat.Chat{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
						CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
					},
					Type:     tkchat.TypeGroup,
					Name:     "Test Chat",
					TMCreate: "2024-01-01 00:00:00.000000",
					Participants: []*tkparticipant.Participant{
						{
							Owner: commonidentity.Owner{
								OwnerType: "agent",
								OwnerID:   uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
							},
						},
					},
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
					Type:         tkchat.TypeGroup,
					Name:         "Test Chat",
					TMCreate:     "2024-01-01 00:00:00.000000",
					Participants: nil, // participants should be excluded
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
			token: "2024-01-01 00:00:00.000000",
			responseChats: []*tkchat.Chat{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
						CustomerID: uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"),
					},
					Type:     tkchat.TypeTalk,
					Name:     "General Channel",
					TMCreate: "2024-01-01 00:00:00.000000",
					Participants: []*tkparticipant.Participant{
						{
							Owner: commonidentity.Owner{
								OwnerType: "agent",
								OwnerID:   uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222"),
							},
						},
					},
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
					Type:         tkchat.TypeTalk,
					Name:         "General Channel",
					TMCreate:     "2024-01-01 00:00:00.000000",
					Participants: nil, // participants should be excluded
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
