package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	chatchat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	chatmedia "gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	chatmessagechat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ChatmessageCreate(t *testing.T) {

	tests := []struct {
		name string

		customer    *cscustomer.Customer
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

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("171d965e-3774-11ed-9ecb-9f5bde29a2a8"),
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
				ID: uuid.FromStringOrNil("17cea962-3774-11ed-88d8-4f22aa82ba39"),
			},
			&chatmessagechat.WebhookMessage{
				ID: uuid.FromStringOrNil("17cea962-3774-11ed-88d8-4f22aa82ba39"),
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

			mockReq.EXPECT().ChatV1MessagechatCreate(ctx, tt.customer.ID, tt.chatID, tt.source, tt.messageType, tt.text, tt.medias).Return(tt.response, nil)

			res, err := h.ChatmessageCreate(ctx, tt.customer, tt.chatID, tt.source, tt.messageType, tt.text, tt.medias)
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

		customer *cscustomer.Customer
		chatID   uuid.UUID
		size     uint64
		token    string

		responseChat *chatchat.Chat
		response     []chatmessagechat.Messagechat
		expectRes    []*chatmessagechat.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("8e963eca-3774-11ed-8c44-0b59cccb48c4"),
			},
			uuid.FromStringOrNil("8ec6c9be-3774-11ed-a626-73312e33dc72"),
			10,
			"2020-09-20 03:23:20.995000",

			&chatchat.Chat{
				ID:         uuid.FromStringOrNil("8ec6c9be-3774-11ed-a626-73312e33dc72"),
				CustomerID: uuid.FromStringOrNil("8e963eca-3774-11ed-8c44-0b59cccb48c4"),
			},
			[]chatmessagechat.Messagechat{
				{
					ID: uuid.FromStringOrNil("b45b1da6-3774-11ed-856d-1b95e72acae8"),
				},
			},
			[]*chatmessagechat.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("b45b1da6-3774-11ed-856d-1b95e72acae8"),
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
			mockReq.EXPECT().ChatV1MessagechatGetsByChatID(ctx, tt.chatID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.ChatmessageGetsByChatID(ctx, tt.customer, tt.chatID, tt.size, tt.token)
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

		customer      *cscustomer.Customer
		chatmessageID uuid.UUID

		response  *chatmessagechat.Messagechat
		expectRes *chatmessagechat.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("dfd40852-3775-11ed-acf9-97e998ed77d3"),
			},
			uuid.FromStringOrNil("e00fa786-3775-11ed-ac3f-f7eb62abd600"),

			&chatmessagechat.Messagechat{
				ID:         uuid.FromStringOrNil("e00fa786-3775-11ed-ac3f-f7eb62abd600"),
				CustomerID: uuid.FromStringOrNil("dfd40852-3775-11ed-acf9-97e998ed77d3"),
			},
			&chatmessagechat.WebhookMessage{
				ID:         uuid.FromStringOrNil("e00fa786-3775-11ed-ac3f-f7eb62abd600"),
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

			mockReq.EXPECT().ChatV1MessagechatGet(ctx, tt.chatmessageID).Return(tt.response, nil)

			res, err := h.ChatmessageGet(ctx, tt.customer, tt.chatmessageID)
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

		customer      *cscustomer.Customer
		messagechatID uuid.UUID

		responseChat *chatmessagechat.Messagechat
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("584fa87c-3776-11ed-bf2c-13f5ab6133a7"),
			},
			uuid.FromStringOrNil("587ecd5a-3776-11ed-b8be-93dc6a90e040"),

			&chatmessagechat.Messagechat{
				ID:         uuid.FromStringOrNil("587ecd5a-3776-11ed-b8be-93dc6a90e040"),
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

			mockReq.EXPECT().ChatV1MessagechatGet(ctx, tt.messagechatID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatV1MessagechatDelete(ctx, tt.messagechatID).Return(tt.responseChat, nil)

			_, err := h.ChatmessageDelete(ctx, tt.customer, tt.messagechatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
