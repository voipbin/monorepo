package listenhandler

import (
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-chat-manager/models/messagechatroom"
	"monorepo/bin-chat-manager/pkg/chathandler"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
	"monorepo/bin-chat-manager/pkg/messagechatroomhandler"
)

func Test_v1MessagechatroomsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		chatroomID uuid.UUID
		pageToken  string
		pageSize   uint64
		filters    map[string]string

		responseMessagechatrooms []*messagechatroom.Messagechatroom

		expectRes *sock.Response
	}{
		{
			"gets by chatroom id return 1 item",
			&sock.Request{
				URI:      "/v1/messagechatrooms?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_chatroom_id=d260ac86-3507-11ed-9594-cfe9c21b2ca3&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("d260ac86-3507-11ed-9594-cfe9c21b2ca3"),
			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"deleted":     "false",
				"chatroom_id": "d260ac86-3507-11ed-9594-cfe9c21b2ca3",
			},

			[]*messagechatroom.Messagechatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d3162c64-3507-11ed-afa2-cbb3e904a7b3"),
					},
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d3162c64-3507-11ed-afa2-cbb3e904a7b3","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"gets by chatroom id return 2 item",
			&sock.Request{
				URI:      "/v1/messagechatrooms?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_chatroom_id=3b683b7c-3508-11ed-97bb-9be24437edc2&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("3b683b7c-3508-11ed-97bb-9be24437edc2"),
			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"deleted":     "false",
				"chatroom_id": "3b683b7c-3508-11ed-97bb-9be24437edc2",
			},

			[]*messagechatroom.Messagechatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3bacdc46-3508-11ed-8422-1766fe775e1a"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3c985216-3508-11ed-b760-5bd81cf30313"),
					},
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"3bacdc46-3508-11ed-8422-1766fe775e1a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"3c985216-3508-11ed-b760-5bd81cf30313","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"gets by chat id return 0 item",
			&sock.Request{
				URI:      "/v1/messagechatrooms?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_chatroom_id=4829801e-3503-11ed-aecf-4f1b13ce1b9f&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("4829801e-3503-11ed-aecf-4f1b13ce1b9f"),
			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"deleted":     "false",
				"chatroom_id": "4829801e-3503-11ed-aecf-4f1b13ce1b9f",
			},

			[]*messagechatroom.Messagechatroom{},

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

			mockMessagechatroom := messagechatroomhandler.NewMockMessagechatroomHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,

				messagechatroomHandler: mockMessagechatroom,
			}

			mockMessagechatroom.EXPECT().Gets(gomock.Any(), tt.pageToken, tt.pageSize, tt.filters).Return(tt.responseMessagechatrooms, nil)

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

func Test_v1MessagechatroomsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		messagechatroomID uuid.UUID

		responseMessagechatroom *messagechatroom.Messagechatroom

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/messagechatrooms/92c89948-3508-11ed-9b4f-77544678aa39",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("92c89948-3508-11ed-9b4f-77544678aa39"),

			&messagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("92c89948-3508-11ed-9b4f-77544678aa39"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"92c89948-3508-11ed-9b4f-77544678aa39","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockMessagechatroom := messagechatroomhandler.NewMockMessagechatroomHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,

				messagechatroomHandler: mockMessagechatroom,
			}

			mockMessagechatroom.EXPECT().Get(gomock.Any(), tt.messagechatroomID).Return(tt.responseMessagechatroom, nil)

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

func Test_v1MessagechatroomsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		messagechatroomID uuid.UUID

		responseMessagechatroom *messagechatroom.Messagechatroom
		expectRes               *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/messagechatrooms/ce2b7f32-3508-11ed-8a20-a32e4374af3f",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("ce2b7f32-3508-11ed-8a20-a32e4374af3f"),

			&messagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ce2b7f32-3508-11ed-8a20-a32e4374af3f"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"ce2b7f32-3508-11ed-8a20-a32e4374af3f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockMessagechatroom := messagechatroomhandler.NewMockMessagechatroomHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,

				messagechatroomHandler: mockMessagechatroom,
			}

			mockMessagechatroom.EXPECT().Delete(gomock.Any(), tt.messagechatroomID).Return(tt.responseMessagechatroom, nil)

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
