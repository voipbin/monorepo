package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	chatchat "monorepo/bin-chat-manager/models/chat"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ChatroomGetsByOwnerID(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		ownerID uuid.UUID
		size    uint64
		token   string

		responseAgent *amagent.Agent
		response      []chatchatroom.Chatroom

		expectFilters map[string]string
		expectRes     []*chatchatroom.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			10,
			"2020-09-20 03:23:20.995000",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			[]chatchatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3bb948f2-3777-11ed-861d-d79db76202e4"),
					},
				},
			},

			map[string]string{
				"deleted":  "false",
				"owner_id": "d152e69e-105b-11ee-b395-eb18426de979",
			},
			[]*chatchatroom.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3bb948f2-3777-11ed-861d-d79db76202e4"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.ownerID).Return(tt.responseAgent, nil)
			mockReq.EXPECT().ChatV1ChatroomGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.response, nil)

			res, err := h.ChatroomGetsByOwnerID(ctx, tt.agent, tt.ownerID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatroomGet(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		chatroomID uuid.UUID

		response  *chatchatroom.Chatroom
		expectRes *chatchatroom.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("b1698256-3777-11ed-acfe-e7f4e78652c6"),

			&chatchatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1698256-3777-11ed-acfe-e7f4e78652c6"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1698256-3777-11ed-acfe-e7f4e78652c6"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatV1ChatroomGet(ctx, tt.chatroomID).Return(tt.response, nil)

			res, err := h.ChatroomGet(ctx, tt.agent, tt.chatroomID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_chatroomGetByChatIDAndOwnerID(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		chatID  uuid.UUID
		ownerID uuid.UUID

		responseChatrooms []chatchatroom.Chatroom

		expectFilters map[string]string
		expectRes     *chatchatroom.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			chatID:  uuid.FromStringOrNil("731a0de8-bc02-11ee-9606-1b8395e94244"),
			ownerID: uuid.FromStringOrNil("734e22c2-bc02-11ee-b7b0-9f9f9f508f1b"),

			responseChatrooms: []chatchatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("737e2fda-bc02-11ee-8642-6b9fa97d4d3b"),
					},
				},
			},

			expectFilters: map[string]string{
				"deleted":  "false",
				"chat_id":  "731a0de8-bc02-11ee-9606-1b8395e94244",
				"owner_id": "734e22c2-bc02-11ee-b7b0-9f9f9f508f1b",
			},
			expectRes: &chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("737e2fda-bc02-11ee-8642-6b9fa97d4d3b"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
				dbHandler:   mockDB,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockReq.EXPECT().ChatV1ChatroomGets(ctx, gomock.Any(), uint64(1), tt.expectFilters).Return(tt.responseChatrooms, nil)

			res, err := h.chatroomGetByChatIDAndOwnerID(ctx, tt.agent, tt.chatID, tt.ownerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatroomDelete(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		chatroomID uuid.UUID

		responseChat *chatchatroom.Chatroom
		expectRes    *chatchatroom.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("e592fe04-3777-11ed-8055-3b96646165b9"),

			&chatchatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e592fe04-3777-11ed-8055-3b96646165b9"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e592fe04-3777-11ed-8055-3b96646165b9"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ChatV1ChatroomGet(ctx, tt.chatroomID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatV1ChatroomDelete(ctx, tt.chatroomID).Return(tt.responseChat, nil)

			res, err := h.ChatroomDelete(ctx, tt.agent, tt.chatroomID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}

		})
	}
}

func Test_ChatroomCreate(t *testing.T) {

	tests := []struct {
		name string

		agent          *amagent.Agent
		participantIDs []uuid.UUID
		chatroomName   string
		detail         string

		responseAgents    []*amagent.Agent
		responseChat      *chatchat.Chat
		responseChatrooms []chatchatroom.Chatroom

		expectType    chatchat.Type
		expectFilters map[string]string
		expectRes     *chatchatroom.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			participantIDs: []uuid.UUID{
				uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				uuid.FromStringOrNil("afc242b4-bc03-11ee-88ad-4b5f74ea0f36"),
			},
			chatroomName: "test name",
			detail:       "test detail",

			responseAgents: []*amagent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("afc242b4-bc03-11ee-88ad-4b5f74ea0f36"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
				},
			},
			responseChat: &chatchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3eb7bbde-bc04-11ee-933b-f38aea616771"),
				},
			},
			responseChatrooms: []chatchatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3eed1842-bc04-11ee-8e09-ef9647ed120e"),
					},
				},
			},

			expectType: chatchat.TypeNormal,
			expectFilters: map[string]string{
				"deleted":  "false",
				"chat_id":  "3eb7bbde-bc04-11ee-933b-f38aea616771",
				"owner_id": "d152e69e-105b-11ee-b395-eb18426de979",
			},
			expectRes: &chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3eed1842-bc04-11ee-8e09-ef9647ed120e"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
				dbHandler:   mockDB,
			}
			ctx := context.Background()

			i := 0
			for _, participantID := range tt.participantIDs {
				if participantID == tt.agent.ID {
					continue
				}
				mockReq.EXPECT().AgentV1AgentGet(ctx, participantID).Return(tt.responseAgents[i], nil)
				i++
			}
			mockReq.EXPECT().ChatV1ChatCreate(ctx, tt.agent.CustomerID, tt.expectType, tt.agent.ID, tt.participantIDs, tt.chatroomName, tt.detail).Return(tt.responseChat, nil)

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockReq.EXPECT().ChatV1ChatroomGets(ctx, gomock.Any(), uint64(1), tt.expectFilters).Return(tt.responseChatrooms, nil)

			res, err := h.ChatroomCreate(ctx, tt.agent, tt.participantIDs, tt.chatroomName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatroomUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		chatroomID uuid.UUID
		chatName   string
		detail     string

		responseChatroom *chatchatroom.Chatroom
		expectRes        *chatchatroom.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("999dc43a-bc63-11ee-9462-77cf3a1394d7"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("996add0e-bc63-11ee-86e3-cb1450c93e7b"),
			"update name",
			"update detail",

			&chatchatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("996add0e-bc63-11ee-86e3-cb1450c93e7b"),
					CustomerID: uuid.FromStringOrNil("999dc43a-bc63-11ee-9462-77cf3a1394d7"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				},
			},
			&chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("996add0e-bc63-11ee-86e3-cb1450c93e7b"),
					CustomerID: uuid.FromStringOrNil("999dc43a-bc63-11ee-9462-77cf3a1394d7"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ChatV1ChatroomGet(ctx, tt.chatroomID).Return(tt.responseChatroom, nil)
			mockReq.EXPECT().ChatV1ChatroomUpdateBasicInfo(ctx, tt.chatroomID, tt.chatName, tt.detail).Return(tt.responseChatroom, nil)

			res, err := h.ChatroomUpdateBasicInfo(ctx, tt.agent, tt.chatroomID, tt.chatName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
