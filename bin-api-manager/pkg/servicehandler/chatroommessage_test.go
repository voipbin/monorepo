package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
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

func Test_ChatroommessageCreate(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		chatroomID uuid.UUID
		message    string
		medias     []chatmedia.Media

		responseChatroom        *chatchatroom.Chatroom
		responseMessagechat     *chatmessagechat.Messagechat
		responseMessagechatroom []chatmessagechatroom.Messagechatroom

		expectSource  commonaddress.Address
		expectFilters map[string]string
		expectRes     *chatmessagechatroom.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
				Name:       "test name",
			},
			chatroomID: uuid.FromStringOrNil("e59dcafa-bbf6-11ee-914f-ab362a70a1cf"),
			message:    "hello world",
			medias: []chatmedia.Media{
				{
					Type: chatmedia.TypeAgent,
					Address: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+123456789",
					},
				},
			},

			responseChatroom: &chatchatroom.Chatroom{
				ID:     uuid.FromStringOrNil("e59dcafa-bbf6-11ee-914f-ab362a70a1cf"),
				ChatID: uuid.FromStringOrNil("2fabac6a-bbf8-11ee-9e21-53afa17d17cb"),
			},
			responseMessagechat: &chatmessagechat.Messagechat{
				ID: uuid.FromStringOrNil("2fe4994e-bbf8-11ee-ba94-137f44f6810a"),
			},
			responseMessagechatroom: []chatmessagechatroom.Messagechatroom{
				{
					ID: uuid.FromStringOrNil("33c28d38-bbf7-11ee-895e-779ed7851af1"),
				},
			},

			expectSource: commonaddress.Address{
				Type:       commonaddress.TypeAgent,
				Target:     "d152e69e-105b-11ee-b395-eb18426de979",
				TargetName: "test name",
			},
			expectFilters: map[string]string{
				"chatroom_id":    "e59dcafa-bbf6-11ee-914f-ab362a70a1cf",
				"messagechat_id": "2fe4994e-bbf8-11ee-ba94-137f44f6810a",
			},
			expectRes: &chatmessagechatroom.WebhookMessage{
				ID: uuid.FromStringOrNil("33c28d38-bbf7-11ee-895e-779ed7851af1"),
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

			mockReq.EXPECT().ChatV1ChatroomGet(ctx, tt.chatroomID).Return(tt.responseChatroom, nil)
			mockReq.EXPECT().ChatV1MessagechatCreate(ctx, tt.agent.CustomerID, tt.responseChatroom.ChatID, tt.expectSource, chatmessagechat.TypeNormal, tt.message, tt.medias).Return(tt.responseMessagechat, nil)

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockReq.EXPECT().ChatV1MessagechatroomGets(ctx, gomock.Any(), uint64(1), tt.expectFilters).Return(tt.responseMessagechatroom, nil)

			res, err := h.ChatroommessageCreate(ctx, tt.agent, tt.chatroomID, tt.message, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
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

		expectFilters map[string]string
		expectRes     []*chatmessagechatroom.WebhookMessage
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

			map[string]string{
				"deleted":     "false",
				"chatroom_id": "a416bb54-3778-11ed-9eb3-4b926f921d68",
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
			mockReq.EXPECT().ChatV1MessagechatroomGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.response, nil)

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
