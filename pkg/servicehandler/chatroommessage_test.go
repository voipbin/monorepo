package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	chatmessagechatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ChatroommessageGet(t *testing.T) {

	tests := []struct {
		name string

		customer          *cscustomer.Customer
		chatroommessageID uuid.UUID

		response  *chatmessagechatroom.Messagechatroom
		expectRes *chatmessagechatroom.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("dfd40852-3775-11ed-acf9-97e998ed77d3"),
			},
			uuid.FromStringOrNil("78605f6a-3778-11ed-a0bc-f7087bf490a3"),

			&chatmessagechatroom.Messagechatroom{
				ID:         uuid.FromStringOrNil("78605f6a-3778-11ed-a0bc-f7087bf490a3"),
				CustomerID: uuid.FromStringOrNil("dfd40852-3775-11ed-acf9-97e998ed77d3"),
			},
			&chatmessagechatroom.WebhookMessage{
				ID:         uuid.FromStringOrNil("78605f6a-3778-11ed-a0bc-f7087bf490a3"),
				CustomerID: uuid.FromStringOrNil("dfd40852-3775-11ed-acf9-97e998ed77d3"),
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

			res, err := h.ChatroommessageGet(ctx, tt.customer, tt.chatroommessageID)
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

		customer   *cscustomer.Customer
		chatroomID uuid.UUID
		size       uint64
		token      string

		responseChatroom *chatchatroom.Chatroom
		response         []chatmessagechatroom.Messagechatroom
		expectRes        []*chatmessagechatroom.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("8e963eca-3774-11ed-8c44-0b59cccb48c4"),
			},
			uuid.FromStringOrNil("a416bb54-3778-11ed-9eb3-4b926f921d68"),
			10,
			"2020-09-20 03:23:20.995000",

			&chatchatroom.Chatroom{
				ID:         uuid.FromStringOrNil("a416bb54-3778-11ed-9eb3-4b926f921d68"),
				CustomerID: uuid.FromStringOrNil("8e963eca-3774-11ed-8c44-0b59cccb48c4"),
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

			res, err := h.ChatroommessageGetsByChatroomID(ctx, tt.customer, tt.chatroomID, tt.size, tt.token)
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

		customer          *cscustomer.Customer
		messagechatroomID uuid.UUID

		responseChatroommessage *chatmessagechatroom.Messagechatroom
		expectRes               *chatmessagechatroom.WebhookMessage
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("584fa87c-3776-11ed-bf2c-13f5ab6133a7"),
			},
			uuid.FromStringOrNil("f091186c-3778-11ed-afad-ef1c157d091e"),

			&chatmessagechatroom.Messagechatroom{
				ID:         uuid.FromStringOrNil("f091186c-3778-11ed-afad-ef1c157d091e"),
				CustomerID: uuid.FromStringOrNil("584fa87c-3776-11ed-bf2c-13f5ab6133a7"),
			},
			&chatmessagechatroom.WebhookMessage{
				ID:         uuid.FromStringOrNil("f091186c-3778-11ed-afad-ef1c157d091e"),
				CustomerID: uuid.FromStringOrNil("584fa87c-3776-11ed-bf2c-13f5ab6133a7"),
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

			res, err := h.ChatroommessageDelete(ctx, tt.customer, tt.messagechatroomID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
