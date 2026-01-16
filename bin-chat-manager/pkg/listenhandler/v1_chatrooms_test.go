package listenhandler

import (
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-chat-manager/models/chatroom"
	"monorepo/bin-chat-manager/pkg/chathandler"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
)

func Test_v1ChatroomsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID uuid.UUID
		ownerID    uuid.UUID
		pageToken  string
		pageSize   uint64
		filters    map[chatroom.Field]any

		responseChatrooms []*chatroom.Chatroom

		expectRes *sock.Response
	}{
		{
			"gets by owner id return 1 item",
			&sock.Request{
				URI:      "/v1/chatrooms?page_token=2020-10-10T03:30:17.000000&page_size=10&owner_id=5cc29ca4-3503-11ed-af37-3388a22eea50&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.Nil,
			uuid.FromStringOrNil("5cc29ca4-3503-11ed-af37-3388a22eea50"),
			"2020-10-10T03:30:17.000000",
			10,
			map[chatroom.Field]any{
				chatroom.FieldDeleted: false,
				chatroom.FieldOwnerID: uuid.FromStringOrNil("5cc29ca4-3503-11ed-af37-3388a22eea50"),
			},

			[]*chatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5cedbefc-3503-11ed-a344-aff6ed0bb63f"),
					},
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5cedbefc-3503-11ed-a344-aff6ed0bb63f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"gets by owner id return 2 item",
			&sock.Request{
				URI:      "/v1/chatrooms?page_token=2020-10-10T03:30:17.000000&page_size=10&owner_id=5d1a8cca-3503-11ed-88db-57e51b7f708f&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.Nil,
			uuid.FromStringOrNil("5d1a8cca-3503-11ed-88db-57e51b7f708f"),
			"2020-10-10T03:30:17.000000",
			10,
			map[chatroom.Field]any{
				chatroom.FieldDeleted: false,
				chatroom.FieldOwnerID: uuid.FromStringOrNil("5d1a8cca-3503-11ed-88db-57e51b7f708f"),
			},

			[]*chatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5d479a4e-3503-11ed-9d61-d375835c6b38"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5d74efc6-3503-11ed-bf79-4f76cba0af3c"),
					},
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5d479a4e-3503-11ed-9d61-d375835c6b38","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"5d74efc6-3503-11ed-bf79-4f76cba0af3c","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"gets by customer id return 0 item",
			&sock.Request{
				URI:      "/v1/chatrooms?page_token=2020-10-10T03:30:17.000000&page_size=10&owner_id=5dae1058-3503-11ed-a7d3-df338985d478&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.Nil,
			uuid.FromStringOrNil("5dae1058-3503-11ed-a7d3-df338985d478"),
			"2020-10-10T03:30:17.000000",
			10,
			map[chatroom.Field]any{
				chatroom.FieldDeleted: false,
				chatroom.FieldOwnerID: uuid.FromStringOrNil("5dae1058-3503-11ed-a7d3-df338985d478"),
			},

			[]*chatroom.Chatroom{},

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

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,
			}

			mockChatroom.EXPECT().List(gomock.Any(), tt.pageToken, tt.pageSize, gomock.Any()).Return(tt.responseChatrooms, nil)

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

func Test_v1ChatroomsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		chatroomID uuid.UUID

		responseChatroom *chatroom.Chatroom

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/chatrooms/be0b33ea-3503-11ed-9ea4-d3c16293dae7",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("be0b33ea-3503-11ed-9ea4-d3c16293dae7"),

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("be0b33ea-3503-11ed-9ea4-d3c16293dae7"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"be0b33ea-3503-11ed-9ea4-d3c16293dae7","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,
			}

			mockChatroom.EXPECT().Get(gomock.Any(), tt.chatroomID).Return(tt.responseChatroom, nil)

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

func Test_v1ChatroomsIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		chatroomID   uuid.UUID
		updateName   string
		updateDetail string

		responseChatroom *chatroom.Chatroom

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/chatrooms/d11c222e-bc5b-11ee-940b-d3e8acd4c0d3",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name": "update name", "detail": "update detail"}`),
			},

			uuid.FromStringOrNil("d11c222e-bc5b-11ee-940b-d3e8acd4c0d3"),
			"update name",
			"update detail",

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d11c222e-bc5b-11ee-940b-d3e8acd4c0d3"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d11c222e-bc5b-11ee-940b-d3e8acd4c0d3","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,
			}

			mockChatroom.EXPECT().UpdateBasicInfo(gomock.Any(), tt.chatroomID, tt.updateName, tt.updateDetail).Return(tt.responseChatroom, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_v1ChatroomsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		chatroomID uuid.UUID

		responseChatroom *chatroom.Chatroom
		expectRes        *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/chatrooms/3ec65848-3504-11ed-bf5e-738f1d450725",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("3ec65848-3504-11ed-bf5e-738f1d450725"),

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3ec65848-3504-11ed-bf5e-738f1d450725"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3ec65848-3504-11ed-bf5e-738f1d450725","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			h := &listenHandler{
				sockHandler: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,
			}

			mockChatroom.EXPECT().Delete(gomock.Any(), tt.chatroomID).Return(tt.responseChatroom, nil)

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
