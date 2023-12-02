package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
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
		expectRes     []*chatchatroom.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("f6ff43e2-3776-11ed-ad30-87da66e042fe"),
			10,
			"2020-09-20 03:23:20.995000",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("f6ff43e2-3776-11ed-ad30-87da66e042fe"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},

			[]chatchatroom.Chatroom{
				{
					ID: uuid.FromStringOrNil("3bb948f2-3777-11ed-861d-d79db76202e4"),
				},
			},
			[]*chatchatroom.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("3bb948f2-3777-11ed-861d-d79db76202e4"),
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
			mockReq.EXPECT().ChatV1ChatroomGetsByOwnerID(ctx, tt.ownerID, tt.token, tt.size).Return(tt.response, nil)

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
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("b1698256-3777-11ed-acfe-e7f4e78652c6"),

			&chatchatroom.Chatroom{
				ID:         uuid.FromStringOrNil("b1698256-3777-11ed-acfe-e7f4e78652c6"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatchatroom.WebhookMessage{
				ID:         uuid.FromStringOrNil("b1698256-3777-11ed-acfe-e7f4e78652c6"),
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
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("e592fe04-3777-11ed-8055-3b96646165b9"),

			&chatchatroom.Chatroom{
				ID:         uuid.FromStringOrNil("e592fe04-3777-11ed-8055-3b96646165b9"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatchatroom.WebhookMessage{
				ID:         uuid.FromStringOrNil("e592fe04-3777-11ed-8055-3b96646165b9"),
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
