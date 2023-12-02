package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	chatmessagechatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ChatroommessageGet(t *testing.T) {

	tests := []struct {
		name string

		agent             *amagent.Agent
		chatroommessageID uuid.UUID

		response  *chatmessagechatroom.Messagechatroom
		expectRes *chatmessagechatroom.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("78605f6a-3778-11ed-a0bc-f7087bf490a3"),

			&chatmessagechatroom.Messagechatroom{
				ID:         uuid.FromStringOrNil("78605f6a-3778-11ed-a0bc-f7087bf490a3"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatmessagechatroom.WebhookMessage{
				ID:         uuid.FromStringOrNil("78605f6a-3778-11ed-a0bc-f7087bf490a3"),
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

			mockReq.EXPECT().ChatV1MessagechatroomGet(ctx, tt.chatroommessageID).Return(tt.response, nil)

			res, err := h.ChatroommessageGet(ctx, tt.agent, tt.chatroommessageID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatroommessageGetsByChatroomID(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		chatroomID uuid.UUID
		size       uint64
		token      string

		responseChatroom *chatchatroom.Chatroom
		response         []chatmessagechatroom.Messagechatroom
		expectRes        []*chatmessagechatroom.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("a416bb54-3778-11ed-9eb3-4b926f921d68"),
			10,
			"2020-09-20 03:23:20.995000",

			&chatchatroom.Chatroom{
				ID:         uuid.FromStringOrNil("a416bb54-3778-11ed-9eb3-4b926f921d68"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			[]chatmessagechatroom.Messagechatroom{
				{
					ID: uuid.FromStringOrNil("a442c686-3778-11ed-a13a-83c9ddd14c70"),
				},
			},
			[]*chatmessagechatroom.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("a442c686-3778-11ed-a13a-83c9ddd14c70"),
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
			mockReq.EXPECT().ChatV1MessagechatroomGetsByChatroomID(ctx, tt.chatroomID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.ChatroommessageGetsByChatroomID(ctx, tt.agent, tt.chatroomID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatroommessageDelete(t *testing.T) {

	tests := []struct {
		name string

		agent             *amagent.Agent
		messagechatroomID uuid.UUID

		responseChatroommessage *chatmessagechatroom.Messagechatroom
		expectRes               *chatmessagechatroom.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("f091186c-3778-11ed-afad-ef1c157d091e"),

			&chatmessagechatroom.Messagechatroom{
				ID:         uuid.FromStringOrNil("f091186c-3778-11ed-afad-ef1c157d091e"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatmessagechatroom.WebhookMessage{
				ID:         uuid.FromStringOrNil("f091186c-3778-11ed-afad-ef1c157d091e"),
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

			mockReq.EXPECT().ChatV1MessagechatroomGet(ctx, tt.messagechatroomID).Return(tt.responseChatroommessage, nil)
			mockReq.EXPECT().ChatV1MessagechatroomDelete(ctx, tt.messagechatroomID).Return(tt.responseChatroommessage, nil)

			res, err := h.ChatroommessageDelete(ctx, tt.agent, tt.messagechatroomID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
