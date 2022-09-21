package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	chatchat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ChatCreate(t *testing.T) {

	tests := []struct {
		name string

		customer       *cscustomer.Customer
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

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("f0d20d08-376f-11ed-9a7a-dbc21a700b21"),
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
				ID: uuid.FromStringOrNil("f1b77320-376f-11ed-9a81-3f5fa945b36b"),
			},
			&chatchat.WebhookMessage{
				ID: uuid.FromStringOrNil("f1b77320-376f-11ed-9a81-3f5fa945b36b"),
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

			mockReq.EXPECT().ChatV1ChatCreate(ctx, tt.customer.ID, tt.chatType, tt.ownerID, tt.participantIDs, tt.chatName, tt.detail).Return(tt.response, nil)

			res, err := h.ChatCreate(ctx, tt.customer, tt.chatType, tt.ownerID, tt.participantIDs, tt.chatName, tt.detail)
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

		customer *cscustomer.Customer
		size     uint64
		token    string

		response  []chatchat.Chat
		expectRes []*chatchat.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("040422b6-3771-11ed-801b-27518c703c82"),
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]chatchat.Chat{
				{
					ID: uuid.FromStringOrNil("077992fa-3771-11ed-ba07-13550523bc69"),
				},
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

			mockReq.EXPECT().ChatV1ChatGetsByCustomerID(ctx, tt.customer.ID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.ChatGetsByCustomerID(ctx, tt.customer, tt.size, tt.token)
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

		customer *cscustomer.Customer
		chatID   uuid.UUID

		response  *chatchat.Chat
		expectRes *chatchat.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("539c78aa-3771-11ed-ab19-379f45ca7efc"),
			},
			uuid.FromStringOrNil("53cf13e6-3771-11ed-8c41-5f1bcf653b18"),

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("53cf13e6-3771-11ed-8c41-5f1bcf653b18"),
				CustomerID: uuid.FromStringOrNil("539c78aa-3771-11ed-ab19-379f45ca7efc"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("53cf13e6-3771-11ed-8c41-5f1bcf653b18"),
				CustomerID: uuid.FromStringOrNil("539c78aa-3771-11ed-ab19-379f45ca7efc"),
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

			res, err := h.ChatGet(ctx, tt.customer, tt.chatID)
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

		customer *cscustomer.Customer
		chatID   uuid.UUID

		responseChat *chatchat.Chat
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
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

			_, err := h.ChatDelete(ctx, tt.customer, tt.chatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_ChatUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		chatID   uuid.UUID
		chatName string
		detail   string

		responseChat *chatchat.Chat
		expectRes    *chatchat.WebhookMessage
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("9a38b138-3772-11ed-a262-bb1543f2a312"),
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
			"update name",
			"update detail",

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("9a38b138-3772-11ed-a262-bb1543f2a312"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("9a38b138-3772-11ed-a262-bb1543f2a312"),
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

			res, err := h.ChatUpdateBasicInfo(ctx, tt.customer, tt.chatID, tt.chatName, tt.detail)
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

		customer *cscustomer.Customer
		chatID   uuid.UUID
		ownerID  uuid.UUID

		responseChat *chatchat.Chat
		expectRes    *chatchat.WebhookMessage
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("eebfbad0-3772-11ed-8c02-7faceae786c9"),
			},
			uuid.FromStringOrNil("eee4287a-3772-11ed-9f41-b3f8e184a4c1"),
			uuid.FromStringOrNil("ef0cc3f2-3772-11ed-a9b8-8bf05018295c"),

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("eee4287a-3772-11ed-9f41-b3f8e184a4c1"),
				CustomerID: uuid.FromStringOrNil("eebfbad0-3772-11ed-8c02-7faceae786c9"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("eee4287a-3772-11ed-9f41-b3f8e184a4c1"),
				CustomerID: uuid.FromStringOrNil("eebfbad0-3772-11ed-8c02-7faceae786c9"),
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
			mockReq.EXPECT().ChatV1ChatUpdateOwnerID(ctx, tt.chatID, tt.ownerID).Return(tt.responseChat, nil)

			res, err := h.ChatUpdateOwnerID(ctx, tt.customer, tt.chatID, tt.ownerID)
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

		customer      *cscustomer.Customer
		chatID        uuid.UUID
		participantID uuid.UUID

		responseChat *chatchat.Chat
		expectRes    *chatchat.WebhookMessage
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("25e7d402-3773-11ed-9908-5bbbf25364f9"),
			},
			uuid.FromStringOrNil("2626ffe2-3773-11ed-9d5c-0bd9c532f572"),
			uuid.FromStringOrNil("266684d2-3773-11ed-891f-d76283b1a5a3"),

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("2626ffe2-3773-11ed-9d5c-0bd9c532f572"),
				CustomerID: uuid.FromStringOrNil("25e7d402-3773-11ed-9908-5bbbf25364f9"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("2626ffe2-3773-11ed-9d5c-0bd9c532f572"),
				CustomerID: uuid.FromStringOrNil("25e7d402-3773-11ed-9908-5bbbf25364f9"),
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

			res, err := h.ChatAddParticipantID(ctx, tt.customer, tt.chatID, tt.participantID)
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

		customer      *cscustomer.Customer
		chatID        uuid.UUID
		participantID uuid.UUID

		responseChat *chatchat.Chat
		expectRes    *chatchat.WebhookMessage
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("4a0a7966-3773-11ed-8b12-afa69a2caf8d"),
			},
			uuid.FromStringOrNil("4a4100c6-3773-11ed-b5a8-ef19d4af83c0"),
			uuid.FromStringOrNil("4a77ca02-3773-11ed-bdb7-47ea979defcf"),

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("4a4100c6-3773-11ed-b5a8-ef19d4af83c0"),
				CustomerID: uuid.FromStringOrNil("4a0a7966-3773-11ed-8b12-afa69a2caf8d"),
			},
			&chatchat.WebhookMessage{
				ID:         uuid.FromStringOrNil("4a4100c6-3773-11ed-b5a8-ef19d4af83c0"),
				CustomerID: uuid.FromStringOrNil("4a0a7966-3773-11ed-8b12-afa69a2caf8d"),
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

			res, err := h.ChatRemoveParticipantID(ctx, tt.customer, tt.chatID, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
