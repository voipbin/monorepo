package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/chathandler"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/chatroomhandler"
)

func Test_v1ChatroomsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		customerID uuid.UUID
		ownerID    uuid.UUID
		pageToken  string
		pageSize   uint64
		filters    map[string]string

		responseChatrooms []*chatroom.Chatroom

		expectRes *rabbitmqhandler.Response
	}{
		{
			"gets by owner id return 1 item",
			&rabbitmqhandler.Request{
				URI:      "/v1/chatrooms?page_token=2020-10-10T03:30:17.000000&page_size=10&owner_id=5cc29ca4-3503-11ed-af37-3388a22eea50&filter_deleted=false",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.Nil,
			uuid.FromStringOrNil("5cc29ca4-3503-11ed-af37-3388a22eea50"),
			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"deleted":  "false",
				"owner_id": "5cc29ca4-3503-11ed-af37-3388a22eea50",
			},

			[]*chatroom.Chatroom{
				{
					ID: uuid.FromStringOrNil("5cedbefc-3503-11ed-a344-aff6ed0bb63f"),
				},
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5cedbefc-3503-11ed-a344-aff6ed0bb63f","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","onwer_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"gets by owner id return 2 item",
			&rabbitmqhandler.Request{
				URI:      "/v1/chatrooms?page_token=2020-10-10T03:30:17.000000&page_size=10&owner_id=5d1a8cca-3503-11ed-88db-57e51b7f708f&filter_deleted=false",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.Nil,
			uuid.FromStringOrNil("5d1a8cca-3503-11ed-88db-57e51b7f708f"),
			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"deleted":  "false",
				"owner_id": "5d1a8cca-3503-11ed-88db-57e51b7f708f",
			},

			[]*chatroom.Chatroom{
				{
					ID: uuid.FromStringOrNil("5d479a4e-3503-11ed-9d61-d375835c6b38"),
				},
				{
					ID: uuid.FromStringOrNil("5d74efc6-3503-11ed-bf79-4f76cba0af3c"),
				},
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5d479a4e-3503-11ed-9d61-d375835c6b38","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","onwer_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"5d74efc6-3503-11ed-bf79-4f76cba0af3c","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","onwer_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"gets by customer id return 0 item",
			&rabbitmqhandler.Request{
				URI:      "/v1/chatrooms?page_token=2020-10-10T03:30:17.000000&page_size=10&owner_id=5dae1058-3503-11ed-a7d3-df338985d478&filter_deleted=false",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.Nil,
			uuid.FromStringOrNil("5dae1058-3503-11ed-a7d3-df338985d478"),
			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"deleted":  "false",
				"owner_id": "5dae1058-3503-11ed-a7d3-df338985d478",
			},

			[]*chatroom.Chatroom{},

			&rabbitmqhandler.Response{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)

			mockChat := chathandler.NewMockChatHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)

			h := &listenHandler{
				rabbitSock: mockSock,

				chatHandler:     mockChat,
				chatroomHandler: mockChatroom,
			}

			mockChatroom.EXPECT().Gets(gomock.Any(), tt.pageToken, tt.pageSize, tt.filters).Return(tt.responseChatrooms, nil)

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
		request *rabbitmqhandler.Request

		chatroomID uuid.UUID

		responseChatroom *chatroom.Chatroom

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/chatrooms/be0b33ea-3503-11ed-9ea4-d3c16293dae7",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("be0b33ea-3503-11ed-9ea4-d3c16293dae7"),

			&chatroom.Chatroom{
				ID: uuid.FromStringOrNil("be0b33ea-3503-11ed-9ea4-d3c16293dae7"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"be0b33ea-3503-11ed-9ea4-d3c16293dae7","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","onwer_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)

			mockChat := chathandler.NewMockChatHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)

			h := &listenHandler{
				rabbitSock: mockSock,

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
		request *rabbitmqhandler.Request

		chatroomID   uuid.UUID
		updateName   string
		updateDetail string

		responseChatroom *chatroom.Chatroom

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/chatrooms/d11c222e-bc5b-11ee-940b-d3e8acd4c0d3",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name": "update name", "detail": "update detail"}`),
			},

			uuid.FromStringOrNil("d11c222e-bc5b-11ee-940b-d3e8acd4c0d3"),
			"update name",
			"update detail",

			&chatroom.Chatroom{
				ID: uuid.FromStringOrNil("d11c222e-bc5b-11ee-940b-d3e8acd4c0d3"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d11c222e-bc5b-11ee-940b-d3e8acd4c0d3","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","onwer_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)

			mockChat := chathandler.NewMockChatHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)

			h := &listenHandler{
				rabbitSock: mockSock,

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
		request *rabbitmqhandler.Request

		chatroomID uuid.UUID

		responseChatroom *chatroom.Chatroom
		expectRes        *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/chatrooms/3ec65848-3504-11ed-bf5e-738f1d450725",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("3ec65848-3504-11ed-bf5e-738f1d450725"),

			&chatroom.Chatroom{
				ID: uuid.FromStringOrNil("3ec65848-3504-11ed-bf5e-738f1d450725"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3ec65848-3504-11ed-bf5e-738f1d450725","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","onwer_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)

			mockChat := chathandler.NewMockChatHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)

			h := &listenHandler{
				rabbitSock: mockSock,

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
