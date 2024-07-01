package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"

	chatchat "monorepo/bin-chat-manager/models/chat"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ChatCreate(t *testing.T) {

	tests := []struct {
		name string

		agent          *amagent.Agent
		chatType       chatchat.Type
		ownerID        uuid.UUID
		participantIDs []uuid.UUID
		chatName       string
		detail         string

		response  *chatchat.Chat
		expectRes *chatchat.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			chatchat.TypeNormal,
			uuid.FromStringOrNil("f125112e-376f-11ed-8107-67a724b24bf1"),
			[]uuid.UUID{
				uuid.FromStringOrNil("f15899a4-376f-11ed-8d03-ab6928dc54a5"),
				uuid.FromStringOrNil("f1877be8-376f-11ed-8578-5bd2154bc9e0"),
			},
			"test name",
			"test detail",

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("f1b77320-376f-11ed-9a81-3f5fa945b36b"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("f1b77320-376f-11ed-9a81-3f5fa945b36b"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ChatV1ChatCreate(ctx, tt.agent.CustomerID, tt.chatType, tt.ownerID, tt.participantIDs, tt.chatName, tt.detail).Return(tt.response, nil)

			res, err := h.ChatCreate(ctx, tt.agent, tt.chatType, tt.ownerID, tt.participantIDs, tt.chatName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_ChatGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		response []chatchat.Chat

		expectFilters map[string]string
		expectRes     []*chatchat.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]chatchat.Chat{
				{
					ID: uuid.FromStringOrNil("077992fa-3771-11ed-ba07-13550523bc69"),
				},
			},

			map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},
			[]*chatchat.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("077992fa-3771-11ed-ba07-13550523bc69"),
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

			mockReq.EXPECT().ChatV1ChatGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.response, nil)

			res, err := h.ChatGetsByCustomerID(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatGet(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		chatID uuid.UUID

		response  *chatchat.Chat
		expectRes *chatchat.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("53cf13e6-3771-11ed-8c41-5f1bcf653b18"),

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("53cf13e6-3771-11ed-8c41-5f1bcf653b18"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("53cf13e6-3771-11ed-8c41-5f1bcf653b18"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ChatV1ChatGet(ctx, tt.chatID).Return(tt.response, nil)

			res, err := h.ChatGet(ctx, tt.agent, tt.chatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatDelete(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		chatID uuid.UUID

		responseChat *chatchat.Chat
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ChatV1ChatGet(ctx, tt.chatID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatV1ChatDelete(ctx, tt.chatID).Return(tt.responseChat, nil)

			_, err := h.ChatDelete(ctx, tt.agent, tt.chatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_ChatUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		agent    *amagent.Agent
		chatID   uuid.UUID
		chatName string
		detail   string

		responseChat *chatchat.Chat
		expectRes    *chatchat.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
			"update name",
			"update detail",

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ChatV1ChatGet(ctx, tt.chatID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatV1ChatUpdateBasicInfo(ctx, tt.chatID, tt.chatName, tt.detail).Return(tt.responseChat, nil)

			res, err := h.ChatUpdateBasicInfo(ctx, tt.agent, tt.chatID, tt.chatName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ChatUpdateOwnerID(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		chatID  uuid.UUID
		ownerID uuid.UUID

		responseChat *chatchat.Chat
		expectRes    *chatchat.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("eee4287a-3772-11ed-9f41-b3f8e184a4c1"),
			uuid.FromStringOrNil("ef0cc3f2-3772-11ed-a9b8-8bf05018295c"),

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("eee4287a-3772-11ed-9f41-b3f8e184a4c1"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("eee4287a-3772-11ed-9f41-b3f8e184a4c1"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ChatV1ChatGet(ctx, tt.chatID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatV1ChatUpdateRoomOwnerID(ctx, tt.chatID, tt.ownerID).Return(tt.responseChat, nil)

			res, err := h.ChatUpdateRoomOwnerID(ctx, tt.agent, tt.chatID, tt.ownerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ChatAddParticipantID(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		chatID        uuid.UUID
		participantID uuid.UUID

		responseChat *chatchat.Chat
		expectRes    *chatchat.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("2626ffe2-3773-11ed-9d5c-0bd9c532f572"),
			uuid.FromStringOrNil("266684d2-3773-11ed-891f-d76283b1a5a3"),

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("2626ffe2-3773-11ed-9d5c-0bd9c532f572"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("2626ffe2-3773-11ed-9d5c-0bd9c532f572"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ChatV1ChatGet(ctx, tt.chatID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatV1ChatAddParticipantID(ctx, tt.chatID, tt.participantID).Return(tt.responseChat, nil)

			res, err := h.ChatAddParticipantID(ctx, tt.agent, tt.chatID, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ChatRemoveParticipantID(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		chatID        uuid.UUID
		participantID uuid.UUID

		responseChat *chatchat.Chat
		expectRes    *chatchat.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("4a4100c6-3773-11ed-b5a8-ef19d4af83c0"),
			uuid.FromStringOrNil("4a77ca02-3773-11ed-bdb7-47ea979defcf"),

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("4a4100c6-3773-11ed-b5a8-ef19d4af83c0"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("4a4100c6-3773-11ed-b5a8-ef19d4af83c0"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ChatV1ChatGet(ctx, tt.chatID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatV1ChatRemoveParticipantID(ctx, tt.chatID, tt.participantID).Return(tt.responseChat, nil)

			res, err := h.ChatRemoveParticipantID(ctx, tt.agent, tt.chatID, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
