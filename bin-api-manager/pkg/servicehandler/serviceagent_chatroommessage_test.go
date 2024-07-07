package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_ServiceAgentChatroommessageGet(t *testing.T) {

	type test struct {
		name string

		agent             *amagent.Agent
		chatroomMessageID uuid.UUID

		responseChatroomMessage *chatmessagechatroom.Messagechatroom

		expectRes *chatmessagechatroom.WebhookMessage
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
			chatroomMessageID: uuid.FromStringOrNil("48340710-3ba9-11ef-aa75-2761ac041180"),

			responseChatroomMessage: &chatmessagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("48340710-3ba9-11ef-aa75-2761ac041180"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},

			expectRes: &chatmessagechatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("48340710-3ba9-11ef-aa75-2761ac041180"),
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

			mockReq.EXPECT().ChatV1MessagechatroomGet(ctx, tt.chatroomMessageID).Return(tt.responseChatroomMessage, nil)

			res, err := h.ServiceAgentChatroommessageGet(ctx, tt.agent, tt.chatroomMessageID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentChatroommessageGets(t *testing.T) {

	type test struct {
		name string

		agent      *amagent.Agent
		chatroomID uuid.UUID
		size       uint64
		token      string

		responseChatroom         *chatchatroom.Chatroom
		responseChatroomMessages []chatmessagechatroom.Messagechatroom

		expectFilters map[string]string
		expectRes     []*chatmessagechatroom.WebhookMessage
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
			chatroomID: uuid.FromStringOrNil("dc5d2e98-3baa-11ef-8f73-6791414eb608"),
			size:       100,
			token:      "2021-03-01 01:00:00.995000",

			responseChatroom: &chatchatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dc5d2e98-3baa-11ef-8f73-6791414eb608"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			responseChatroomMessages: []chatmessagechatroom.Messagechatroom{
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
				"chatroom_id": "dc5d2e98-3baa-11ef-8f73-6791414eb608",
				"deleted":     "false",
			},
			expectRes: []*chatmessagechatroom.WebhookMessage{
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

			mockReq.EXPECT().ChatV1ChatroomGet(ctx, tt.chatroomID).Return(tt.responseChatroom, nil)
			mockReq.EXPECT().ChatV1MessagechatroomGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseChatroomMessages, nil)

			res, err := h.ServiceAgentChatroommessageGets(ctx, tt.agent, tt.chatroomID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentChatroommessageCreate(t *testing.T) {

	type test struct {
		name string

		agent      *amagent.Agent
		chatroomID uuid.UUID
		message    string
		medias     []chatmedia.Media

		responseChatroom         *chatchatroom.Chatroom
		responseMessageChat      *chatmessagechat.Messagechat
		responseCurTime          string
		responseMessageChatrooms []chatmessagechatroom.Messagechatroom

		expectSource  commonaddress.Address
		expectFilters map[string]string
		expectRes     *chatmessagechatroom.WebhookMessage
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
				Name:       "test name",
			},
			chatroomID: uuid.FromStringOrNil("69648f42-3bac-11ef-aa57-9fde22132b67"),
			message:    "test message",
			medias: []chatmedia.Media{
				{
					Type: chatmedia.TypeAddress,
					Address: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+123456789",
					},
				},
			},

			responseChatroom: &chatchatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("69648f42-3bac-11ef-aa57-9fde22132b67"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
				ChatID: uuid.FromStringOrNil("8297bd4e-3bad-11ef-bcfc-abe3935be0e0"),
			},
			responseMessageChat: &chatmessagechat.Messagechat{
				ID: uuid.FromStringOrNil("b3d48d38-3bad-11ef-926f-efbf47b2f0f5"),
			},
			responseCurTime: "2021-03-01 01:00:00.995000",
			responseMessageChatrooms: []chatmessagechatroom.Messagechatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b4020650-3bad-11ef-ac6b-f75ee7bb0d97"),
					},
				},
			},

			expectSource: commonaddress.Address{
				Type:       commonaddress.TypeAgent,
				Target:     "5cd8c836-3b9f-11ef-98ac-db226570f09a",
				TargetName: "test name",
			},
			expectFilters: map[string]string{
				"chatroom_id":    "69648f42-3bac-11ef-aa57-9fde22132b67",
				"messagechat_id": "b3d48d38-3bad-11ef-926f-efbf47b2f0f5",
			},
			expectRes: &chatmessagechatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b4020650-3bad-11ef-ac6b-f75ee7bb0d97"),
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

			mockReq.EXPECT().ChatV1ChatroomGet(ctx, tt.chatroomID).Return(tt.responseChatroom, nil)
			mockReq.EXPECT().ChatV1MessagechatCreate(
				ctx,
				tt.agent.CustomerID,
				tt.responseChatroom.ChatID,
				tt.expectSource,
				chatmessagechat.TypeNormal,
				tt.message,
				tt.medias,
			).Return(tt.responseMessageChat, nil)
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockReq.EXPECT().ChatV1MessagechatroomGets(ctx, tt.responseCurTime, uint64(1), tt.expectFilters).Return(tt.responseMessageChatrooms, nil)

			res, err := h.ServiceAgentChatroommessageCreate(ctx, tt.agent, tt.chatroomID, tt.message, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
