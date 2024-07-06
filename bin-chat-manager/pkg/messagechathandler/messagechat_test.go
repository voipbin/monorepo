package messagechathandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-chat-manager/models/chatroom"
	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechat"
	"monorepo/bin-chat-manager/models/messagechatroom"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
	"monorepo/bin-chat-manager/pkg/dbhandler"
	"monorepo/bin-chat-manager/pkg/messagechatroomhandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChat *messagechat.Messagechat
	}{
		{
			"normal",

			uuid.FromStringOrNil("7b4e19b6-32ae-11ed-a6c5-731878cf18dd"),

			&messagechat.Messagechat{
				ID: uuid.FromStringOrNil("7b4e19b6-32ae-11ed-a6c5-731878cf18dd"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messagechatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessagechatGet(ctx, tt.id).Return(tt.responseChat, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		chatroomID uuid.UUID
		token      string
		limit      uint64
		filters    map[string]string

		responseMessagechat []*messagechat.Messagechat
	}{
		{
			"normal",

			uuid.FromStringOrNil("e6358764-32ae-11ed-b8c3-57bb40e5b6e9"),
			"2022-04-18 03:22:17.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			[]*messagechat.Messagechat{
				{
					CustomerID: uuid.FromStringOrNil("e6358764-32ae-11ed-b8c3-57bb40e5b6e9"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messagechatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessagechatGets(ctx, tt.token, tt.limit, tt.filters).Return(tt.responseMessagechat, nil)

			res, err := h.Gets(ctx, tt.token, tt.limit, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseMessagechat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseMessagechat, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		chatID      uuid.UUID
		source      *commonaddress.Address
		messageType messagechat.Type
		text        string
		medias      []media.Media

		responseMessagechat *messagechat.Messagechat
		responseChatroom    []*chatroom.Chatroom

		expectFilters map[string]string
	}{
		{
			"normal",

			uuid.FromStringOrNil("c3d8f7ba-32b7-11ed-838c-8763d177c7c3"),
			uuid.FromStringOrNil("c40c7bda-32b7-11ed-829e-73051197cfc8"),
			&commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000001",
				TargetName: "test target",
				Name:       "test name",
				Detail:     "test detail",
			},
			messagechat.TypeNormal,
			"test message.",
			[]media.Media{},

			&messagechat.Messagechat{
				ID:     uuid.FromStringOrNil("c437982e-32b7-11ed-be19-af03b26e1f0e"),
				ChatID: uuid.FromStringOrNil("c40c7bda-32b7-11ed-829e-73051197cfc8"),
				Source: &commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "test target",
					Name:       "test name",
					Detail:     "test detail",
				},
				Type:   messagechat.TypeNormal,
				Text:   "test message",
				Medias: []media.Media{},
			},
			[]*chatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("41b04940-32b8-11ed-9da3-473f30fded15"),
					},
				},
			},

			map[string]string{
				"chat_id": "c40c7bda-32b7-11ed-829e-73051197cfc8",
				"deleted": "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)
			mockMessagechatroom := messagechatroomhandler.NewMockMessagechatroomHandler(mc)

			h := &messagechatHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,

				chatroomHandler:        mockChatroom,
				messagechatroomHandler: mockMessagechatroom,
			}

			ctx := context.Background()

			// create
			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().MessagechatCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().MessagechatGet(ctx, gomock.Any()).Return(tt.responseMessagechat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessagechat.CustomerID, messagechat.EventTypeMessagechatCreated, tt.responseMessagechat)

			convertType := messagechatroom.ConvertType(tt.responseMessagechat.Type)
			mockChatroom.EXPECT().Gets(ctx, gomock.Any(), gomock.Any(), tt.expectFilters).Return(tt.responseChatroom, nil)
			for _, cr := range tt.responseChatroom {
				mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
				mockMessagechatroom.EXPECT().Create(
					ctx,
					tt.responseMessagechat.CustomerID,
					cr.OwnerID,
					cr.ID,
					tt.responseMessagechat.ID,
					tt.responseMessagechat.Source,
					convertType,
					tt.responseMessagechat.Text,
					tt.responseMessagechat.Medias,
				).Return(&messagechatroom.Messagechatroom{}, nil)
			}

			res, err := h.Create(ctx, tt.customerID, tt.chatID, tt.source, tt.messageType, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseMessagechat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseMessagechat, res)
			}
		})
	}
}

func Test_create(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		chatID      uuid.UUID
		source      *commonaddress.Address
		messageType messagechat.Type
		text        string
		medias      []media.Media

		responseMessagechat *messagechat.Messagechat
	}{
		{
			"normal",

			uuid.FromStringOrNil("b4d12a60-32af-11ed-9cda-cb0fba5c8916"),
			uuid.FromStringOrNil("b546cd7e-32af-11ed-8333-e312881a6101"),
			&commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000001",
				TargetName: "test target",
				Name:       "test name",
				Detail:     "test detail",
			},
			messagechat.TypeNormal,
			"test message.",
			[]media.Media{},

			&messagechat.Messagechat{
				ID: uuid.FromStringOrNil("b5a037c4-32af-11ed-aa5c-23f22985f809"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messagechatHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().MessagechatCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().MessagechatGet(ctx, gomock.Any()).Return(tt.responseMessagechat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessagechat.CustomerID, messagechat.EventTypeMessagechatCreated, tt.responseMessagechat)

			res, err := h.create(ctx, tt.customerID, tt.chatID, tt.source, tt.messageType, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseMessagechat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseMessagechat, res)
			}
		})
	}
}

func Test_delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseMessagechat *messagechat.Messagechat
	}{
		{
			"normal",

			uuid.FromStringOrNil("66878cea-32b0-11ed-96f7-9b88b3c4c089"),

			&messagechat.Messagechat{
				ID: uuid.FromStringOrNil("66878cea-32b0-11ed-96f7-9b88b3c4c089"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messagechatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessagechatDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().MessagechatGet(ctx, tt.responseMessagechat.ID).Return(tt.responseMessagechat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessagechat.CustomerID, messagechat.EventTypeMessagechatDeleted, tt.responseMessagechat)

			res, err := h.delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseMessagechat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseMessagechat, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseMessagechat     *messagechat.Messagechat
		responseMessagechatroom []*messagechatroom.Messagechatroom

		expectFilters map[string]string
	}{
		{
			"normal",

			uuid.FromStringOrNil("bf722c96-32c1-11ed-b432-3f40e42890ad"),

			&messagechat.Messagechat{
				ID: uuid.FromStringOrNil("bf722c96-32c1-11ed-b432-3f40e42890ad"),
			},
			[]*messagechatroom.Messagechatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("11131196-32c2-11ed-b4c7-5bfb2b12ea7a"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("11b3578c-32c2-11ed-9759-976f13bc97e9"),
					},
				},
			},

			map[string]string{
				"deleted":        "false",
				"messagechat_id": "bf722c96-32c1-11ed-b432-3f40e42890ad",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockMessagechatroom := messagechatroomhandler.NewMockMessagechatroomHandler(mc)

			h := &messagechatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,

				messagechatroomHandler: mockMessagechatroom,
			}

			ctx := context.Background()

			// delete
			mockDB.EXPECT().MessagechatDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().MessagechatGet(ctx, tt.responseMessagechat.ID).Return(tt.responseMessagechat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessagechat.CustomerID, messagechat.EventTypeMessagechatDeleted, tt.responseMessagechat)

			mockMessagechatroom.EXPECT().Gets(ctx, dbhandler.DefaultTimeStamp, gomock.Any(), tt.expectFilters).Return(tt.responseMessagechatroom, nil)
			for _, mc := range tt.responseMessagechatroom {
				mockMessagechatroom.EXPECT().Delete(ctx, mc.ID).Return(&messagechatroom.Messagechatroom{}, nil)
			}

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseMessagechat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseMessagechat, res)
			}
		})
	}
}
