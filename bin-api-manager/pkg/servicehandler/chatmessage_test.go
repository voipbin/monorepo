package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	chatchat "monorepo/bin-chat-manager/models/chat"
	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ChatmessageCreate(t *testing.T) {

	tests := []struct {
		name string

		agent       *amagent.Agent
		chatID      uuid.UUID
		source      commonaddress.Address
		messageType chatmessagechat.Type
		text        string
		medias      []chatmedia.Media

		response  *chatmessagechat.Messagechat
		expectRes *chatmessagechat.WebhookMessage
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
			uuid.FromStringOrNil("1768c58e-3774-11ed-ac88-3b7ca9a452f4"),
			commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			chatmessagechat.TypeNormal,
			"test text",
			[]chatmedia.Media{},

			&chatmessagechat.Messagechat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("17cea962-3774-11ed-88d8-4f22aa82ba39"),
				},
			},
			&chatmessagechat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("17cea962-3774-11ed-88d8-4f22aa82ba39"),
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

			mockReq.EXPECT().ChatV1MessagechatCreate(ctx, tt.agent.CustomerID, tt.chatID, tt.source, tt.messageType, tt.text, tt.medias).Return(tt.response, nil)

			res, err := h.ChatmessageCreate(ctx, tt.agent, tt.chatID, tt.source, tt.messageType, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_ChatmessageGetsByChatID(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		chatID uuid.UUID
		size   uint64
		token  string

		responseChat *chatchat.Chat
		response     []chatmessagechat.Messagechat

		expectFilters map[string]string
		expectRes     []*chatmessagechat.WebhookMessage
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
			uuid.FromStringOrNil("8ec6c9be-3774-11ed-a626-73312e33dc72"),
			10,
			"2020-09-20 03:23:20.995000",

			&chatchat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8ec6c9be-3774-11ed-a626-73312e33dc72"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			[]chatmessagechat.Messagechat{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b45b1da6-3774-11ed-856d-1b95e72acae8"),
					},
				},
			},

			map[string]string{
				"deleted": "false",
				"chat_id": "8ec6c9be-3774-11ed-a626-73312e33dc72",
			},
			[]*chatmessagechat.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b45b1da6-3774-11ed-856d-1b95e72acae8"),
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

			mockReq.EXPECT().ChatV1ChatGet(ctx, tt.chatID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatV1MessagechatGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.response, nil)

			res, err := h.ChatmessageGetsByChatID(ctx, tt.agent, tt.chatID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatmessageGet(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		chatmessageID uuid.UUID

		response  *chatmessagechat.Messagechat
		expectRes *chatmessagechat.WebhookMessage
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
			uuid.FromStringOrNil("e00fa786-3775-11ed-ac3f-f7eb62abd600"),

			&chatmessagechat.Messagechat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e00fa786-3775-11ed-ac3f-f7eb62abd600"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&chatmessagechat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e00fa786-3775-11ed-ac3f-f7eb62abd600"),
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

			mockReq.EXPECT().ChatV1MessagechatGet(ctx, tt.chatmessageID).Return(tt.response, nil)

			res, err := h.ChatmessageGet(ctx, tt.agent, tt.chatmessageID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatmessageDelete(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		messagechatID uuid.UUID

		responseChat *chatmessagechat.Messagechat
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
			uuid.FromStringOrNil("587ecd5a-3776-11ed-b8be-93dc6a90e040"),

			&chatmessagechat.Messagechat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("587ecd5a-3776-11ed-b8be-93dc6a90e040"),
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

			mockReq.EXPECT().ChatV1MessagechatGet(ctx, tt.messagechatID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatV1MessagechatDelete(ctx, tt.messagechatID).Return(tt.responseChat, nil)

			_, err := h.ChatmessageDelete(ctx, tt.agent, tt.messagechatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
