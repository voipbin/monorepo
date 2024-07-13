package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	chatchat "monorepo/bin-chat-manager/models/chat"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_ServiceAgentChatroomGets(t *testing.T) {

	type test struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseChatrooms []chatchatroom.Chatroom
		expectFilters     map[string]string
		expectRes         []*chatchatroom.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			size:  100,
			token: "2021-03-01 01:00:00.995000",

			responseChatrooms: []chatchatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("474885c2-3ba1-11ef-9cc2-77ca8259e738"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("487ee210-3ba1-11ef-81ba-0b1288173f42"),
					},
				},
			},
			expectFilters: map[string]string{
				"owner_id":    "5cd8c836-3b9f-11ef-98ac-db226570f09a",
				"customer_id": "5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9",
				"deleted":     "false",
			},
			expectRes: []*chatchatroom.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("474885c2-3ba1-11ef-9cc2-77ca8259e738"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("487ee210-3ba1-11ef-81ba-0b1288173f42"),
					},
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatV1ChatroomGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseChatrooms, nil)

			res, err := h.ServiceAgentChatroomGets(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentChatroomGet(t *testing.T) {

	type test struct {
		name string

		agent      *amagent.Agent
		chatroomID uuid.UUID

		responseChatroom *chatchatroom.Chatroom
		expectRes        *chatchatroom.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			chatroomID: uuid.FromStringOrNil("5c84e452-3ba2-11ef-98d5-eb4f7adf4754"),

			responseChatroom: &chatchatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5c84e452-3ba2-11ef-98d5-eb4f7adf4754"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			expectRes: &chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5c84e452-3ba2-11ef-98d5-eb4f7adf4754"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatV1ChatroomGet(ctx, tt.chatroomID).Return(tt.responseChatroom, nil)

			res, err := h.ServiceAgentChatroomGet(ctx, tt.agent, tt.chatroomID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentChatroomDelete(t *testing.T) {

	type test struct {
		name string

		agent      *amagent.Agent
		chatroomID uuid.UUID

		responseChatroom *chatchatroom.Chatroom
		expectRes        *chatchatroom.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			chatroomID: uuid.FromStringOrNil("52733472-3ba3-11ef-b4fc-2f4b52924d45"),

			responseChatroom: &chatchatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("52733472-3ba3-11ef-b4fc-2f4b52924d45"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			expectRes: &chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("52733472-3ba3-11ef-b4fc-2f4b52924d45"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatV1ChatroomGet(ctx, tt.chatroomID).Return(tt.responseChatroom, nil)
			mockReq.EXPECT().ChatV1ChatroomDelete(ctx, tt.chatroomID).Return(tt.responseChatroom, nil)

			res, err := h.ServiceAgentChatroomDelete(ctx, tt.agent, tt.chatroomID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentChatroomCreate(t *testing.T) {

	type test struct {
		name string

		agent          *amagent.Agent
		participantIDs []uuid.UUID
		chatroomName   string
		detail         string

		responseChat      *chatchat.Chat
		responseCurTime   string
		responseChatrooms []chatchatroom.Chatroom

		expectChatType chatchat.Type
		expectFilters  map[string]string
		expectRes      *chatchatroom.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			participantIDs: []uuid.UUID{
				uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				uuid.FromStringOrNil("03271792-3ba5-11ef-91ad-8b8ad9480711"),
			},
			chatroomName: "test name",
			detail:       "test detail",

			responseChat: &chatchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6aac6da2-3ba7-11ef-adf3-a3455d849cf3"),
				},
			},
			responseCurTime: "",
			responseChatrooms: []chatchatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("52733472-3ba3-11ef-b4fc-2f4b52924d45"),
					},
					Owner: commonidentity.Owner{
						OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					},
				},
			},

			expectChatType: chatchat.TypeNormal,
			expectFilters: map[string]string{
				"deleted":  "false",
				"chat_id":  "6aac6da2-3ba7-11ef-adf3-a3455d849cf3",
				"owner_id": "5cd8c836-3b9f-11ef-98ac-db226570f09a",
			},
			expectRes: &chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("52733472-3ba3-11ef-b4fc-2f4b52924d45"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
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

			for _, participantID := range tt.participantIDs {
				tmpAgent := &amagent.Agent{
					Identity: commonidentity.Identity{
						ID:         participantID,
						CustomerID: tt.agent.CustomerID,
					},
				}
				if participantID == tt.agent.ID {
					continue
				}

				mockReq.EXPECT().AgentV1AgentGet(ctx, participantID).Return(tmpAgent, nil)
			}
			mockReq.EXPECT().ChatV1ChatCreate(ctx, tt.agent.CustomerID, tt.expectChatType, tt.agent.ID, tt.participantIDs, tt.chatroomName, tt.detail).Return(tt.responseChat, nil)

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockReq.EXPECT().ChatV1ChatroomGets(ctx, tt.responseCurTime, uint64(1), tt.expectFilters).Return(tt.responseChatrooms, nil)

			res, err := h.ServiceAgentChatroomCreate(ctx, tt.agent, tt.participantIDs, tt.chatroomName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
