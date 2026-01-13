package listenhandler

import (
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechat"
	"monorepo/bin-chat-manager/pkg/chathandler"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
	"monorepo/bin-chat-manager/pkg/messagechathandler"
)

func Test_v1MessagechatsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID  uuid.UUID
		chatID      uuid.UUID
		source      *commonaddress.Address
		messageType messagechat.Type
		text        string
		medias      []media.Media

		responseMessagechat *messagechat.Messagechat

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/messagechats",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"9ab4fa2e-3504-11ed-b3a3-53fb5b1fecb9","chat_id":"a0c05828-3504-11ed-9ad6-639abfa992b7","source":{"type":"tel","target":"+821100000001"},"message_type":"normal","text":"test text","medias":[]}`),
			},

			uuid.FromStringOrNil("9ab4fa2e-3504-11ed-b3a3-53fb5b1fecb9"),
			uuid.FromStringOrNil("a0c05828-3504-11ed-9ad6-639abfa992b7"),
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			messagechat.TypeNormal,
			"test text",
			[]media.Media{},

			&messagechat.Messagechat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("63c6f11a-3505-11ed-be2a-7bbff41a9a6c"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"63c6f11a-3505-11ed-be2a-7bbff41a9a6c","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"media is null",
			&sock.Request{
				URI:      "/v1/messagechats",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"9ab4fa2e-3504-11ed-b3a3-53fb5b1fecb9","chat_id":"a0c05828-3504-11ed-9ad6-639abfa992b7","source":{"type":"tel","target":"+821100000001"},"message_type":"normal","text":"test text","medias":null}`),
			},

			uuid.FromStringOrNil("9ab4fa2e-3504-11ed-b3a3-53fb5b1fecb9"),
			uuid.FromStringOrNil("a0c05828-3504-11ed-9ad6-639abfa992b7"),
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			messagechat.TypeNormal,
			"test text",
			[]media.Media{},

			&messagechat.Messagechat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("be7c686a-3cc1-11ed-b98a-cb05fdbf5ebc"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"be7c686a-3cc1-11ed-b98a-cb05fdbf5ebc","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			mockChat := chathandler.NewMockChatHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)
			mockMessagechat := messagechathandler.NewMockMessagechatHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,

				messagechatHandler: mockMessagechat,
			}

			mockMessagechat.EXPECT().Create(
				gomock.Any(),
				tt.customerID,
				tt.chatID,
				tt.source,
				tt.messageType,
				tt.text,
				tt.medias,
			.Return(tt.responseMessagechat, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_v1MessagechatsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		chatID    uuid.UUID
		pageToken string
		pageSize  uint64

		responseMessagechats []*messagechat.Messagechat

		expectFilters map[messagechat.Field]any
		expectRes     *sock.Response
	}{
		{
			"gets by chat id return 1 item",
			&sock.Request{
				URI:      "/v1/messagechats?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_chat_id=1209ea7a-3506-11ed-9c39-83b3c3ded5a4&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("1209ea7a-3506-11ed-9c39-83b3c3ded5a4"),
			"2020-10-10T03:30:17.000000",
			10,

			[]*messagechat.Messagechat{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("123c4966-3506-11ed-be0c-f7d1f54f9992"),
					},
				},
			},

			map[messagechat.Field]any{
				messagechat.FieldChatID:  uuid.FromStringOrNil("1209ea7a-3506-11ed-9c39-83b3c3ded5a4"),
				messagechat.FieldDeleted: false,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"123c4966-3506-11ed-be0c-f7d1f54f9992","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"gets by chat id return 2 item",
			&sock.Request{
				URI:      "/v1/messagechats?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_chat_id=6728bcac-3506-11ed-87e1-6b1453c7790c&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("6728bcac-3506-11ed-87e1-6b1453c7790c"),
			"2020-10-10T03:30:17.000000",
			10,

			[]*messagechat.Messagechat{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("68b5594a-3506-11ed-9414-73dd9e1d2cca"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("68e6aebe-3506-11ed-9fd0-635331039efa"),
					},
				},
			},

			map[messagechat.Field]any{
				messagechat.FieldChatID:  uuid.FromStringOrNil("6728bcac-3506-11ed-87e1-6b1453c7790c"),
				messagechat.FieldDeleted: false,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"68b5594a-3506-11ed-9414-73dd9e1d2cca","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"68e6aebe-3506-11ed-9fd0-635331039efa","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"gets by chat id return 0 item",
			&sock.Request{
				URI:      "/v1/messagechats?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_chat_id=925dfbf8-3506-11ed-b4aa-439c6be5c723&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("925dfbf8-3506-11ed-b4aa-439c6be5c723"),
			"2020-10-10T03:30:17.000000",
			10,

			[]*messagechat.Messagechat{},

			map[messagechat.Field]any{
				messagechat.FieldChatID:  uuid.FromStringOrNil("925dfbf8-3506-11ed-b4aa-439c6be5c723"),
				messagechat.FieldDeleted: false,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			mockChat := chathandler.NewMockChatHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)
			mockMessagechat := messagechathandler.NewMockMessagechatHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,

				messagechatHandler: mockMessagechat,
			}

			mockMessagechat.EXPECT().Gets(gomock.Any(), tt.pageToken, tt.pageSize, gomock.Any().Return(tt.responseMessagechats, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1MessagechatsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		chatID uuid.UUID

		responseMessagechat *messagechat.Messagechat

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/messagechats/cf9f32fc-3506-11ed-97f5-07ccb6f809de",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("cf9f32fc-3506-11ed-97f5-07ccb6f809de"),

			&messagechat.Messagechat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cf9f32fc-3506-11ed-97f5-07ccb6f809de"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"cf9f32fc-3506-11ed-97f5-07ccb6f809de","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			mockChat := chathandler.NewMockChatHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)
			mockMessagechat := messagechathandler.NewMockMessagechatHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,

				messagechatHandler: mockMessagechat,
			}

			mockMessagechat.EXPECT().Get(gomock.Any(), tt.chatID.Return(tt.responseMessagechat, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1MessagechatsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		messagechatID uuid.UUID

		responseMessagechat *messagechat.Messagechat
		expectRes           *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/messagechats/26a0a8c4-3507-11ed-8ced-e36d2e15f350",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("26a0a8c4-3507-11ed-8ced-e36d2e15f350"),

			&messagechat.Messagechat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("26a0a8c4-3507-11ed-8ced-e36d2e15f350"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"26a0a8c4-3507-11ed-8ced-e36d2e15f350","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			mockChat := chathandler.NewMockChatHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)
			mockMessagechat := messagechathandler.NewMockMessagechatHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,

				messagechatHandler: mockMessagechat,
			}

			mockMessagechat.EXPECT().Delete(gomock.Any(), tt.messagechatID.Return(tt.responseMessagechat, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
