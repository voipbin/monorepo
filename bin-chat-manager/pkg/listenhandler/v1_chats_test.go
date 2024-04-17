package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-chat-manager/models/chat"
	"monorepo/bin-chat-manager/pkg/chathandler"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
)

func Test_v1ChatsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		customerID     uuid.UUID
		chatType       chat.Type
		ownerID        uuid.UUID
		participantIDs []uuid.UUID
		chatName       string
		detail         string
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"e0ab1d48-31d9-11ed-bbc5-27681ded85a1","type":"normal","owner_id":"5f66bb7e-31da-11ed-ae71-377183eb19a3","participant_ids":["5f66bb7e-31da-11ed-ae71-377183eb19a3","6ebc6880-31da-11ed-8e95-a3bc92af9795"],"name":"test name","detail":"test detail"}`),
			},

			uuid.FromStringOrNil("e0ab1d48-31d9-11ed-bbc5-27681ded85a1"),
			chat.TypeNormal,
			uuid.FromStringOrNil("5f66bb7e-31da-11ed-ae71-377183eb19a3"),
			[]uuid.UUID{
				uuid.FromStringOrNil("5f66bb7e-31da-11ed-ae71-377183eb19a3"),
				uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
			},
			"test name",
			"test detail",
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

			mockChat.EXPECT().Create(
				gomock.Any(),
				tt.customerID,
				tt.chatType,
				tt.ownerID,
				tt.participantIDs,
				tt.chatName,
				tt.detail,
			).Return(&chat.Chat{}, nil)

			if _, err := h.processRequest(tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_v1ChatsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		pageToken string
		pageSize  uint64
		filters   map[string]string

		responseChats []*chat.Chat

		expectRes *rabbitmqhandler.Response
	}{
		{
			"gets by customer id return 1 item",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=0c21c67c-31dd-11ed-9f27-cb7cefee3726&filter_deleted=false&filter_participant_ids=1cc7bc1a-b95a-11ee-9129-5771eb8762a7,1cf5a562-b95a-11ee-ac01-7b660d7215d6",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"deleted":         "false",
				"customer_id":     "0c21c67c-31dd-11ed-9f27-cb7cefee3726",
				"participant_ids": "1cc7bc1a-b95a-11ee-9129-5771eb8762a7,1cf5a562-b95a-11ee-ac01-7b660d7215d6",
			},

			[]*chat.Chat{
				{
					ID: uuid.FromStringOrNil("6eb6134c-31dd-11ed-9f6c-f7fa9148cdb6"),
				},
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"6eb6134c-31dd-11ed-9f6c-f7fa9148cdb6","customer_id":"00000000-0000-0000-0000-000000000000","type":"","owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"gets by customer id return 2 item",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=472ad5d2-31de-11ed-8f3b-7fbd0e2b1f81&filter_deleted=false",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"deleted":     "false",
				"customer_id": "472ad5d2-31de-11ed-8f3b-7fbd0e2b1f81",
			},

			[]*chat.Chat{
				{
					ID: uuid.FromStringOrNil("47769404-31de-11ed-9ded-470bba65e75d"),
				},
				{
					ID: uuid.FromStringOrNil("47b18d0c-31de-11ed-9cfe-afbd2262ab42"),
				},
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"47769404-31de-11ed-9ded-470bba65e75d","customer_id":"00000000-0000-0000-0000-000000000000","type":"","owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"47b18d0c-31de-11ed-9cfe-afbd2262ab42","customer_id":"00000000-0000-0000-0000-000000000000","type":"","owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"gets by customer id return 0 item",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=77a3e140-31de-11ed-b4d0-3323833e9231&filter_deleted=false",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"deleted":     "false",
				"customer_id": "77a3e140-31de-11ed-b4d0-3323833e9231",
			},

			[]*chat.Chat{},

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

			mockChat.EXPECT().Gets(gomock.Any(), tt.pageToken, tt.pageSize, tt.filters).Return(tt.responseChats, nil)

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

func Test_v1ChatsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		chatID uuid.UUID

		responseChat *chat.Chat

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/13b7f120-31df-11ed-8214-63c85c3c8ecf",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("13b7f120-31df-11ed-8214-63c85c3c8ecf"),

			&chat.Chat{
				ID: uuid.FromStringOrNil("13b7f120-31df-11ed-8214-63c85c3c8ecf"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"13b7f120-31df-11ed-8214-63c85c3c8ecf","customer_id":"00000000-0000-0000-0000-000000000000","type":"","owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockChat.EXPECT().Get(gomock.Any(), tt.chatID).Return(tt.responseChat, nil)

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

func Test_v1ChatsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		chatID uuid.UUID

		responseChat *chat.Chat
		expectRes    *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/d9967376-31df-11ed-ba8f-e376c0c4f1fc",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("d9967376-31df-11ed-ba8f-e376c0c4f1fc"),

			&chat.Chat{
				ID: uuid.FromStringOrNil("d9967376-31df-11ed-ba8f-e376c0c4f1fc"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d9967376-31df-11ed-ba8f-e376c0c4f1fc","customer_id":"00000000-0000-0000-0000-000000000000","type":"","owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockChat.EXPECT().Delete(gomock.Any(), tt.chatID).Return(tt.responseChat, nil)

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

func Test_v1ChatsIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		chatID       uuid.UUID
		updateName   string
		updateDetail string

		responseChat *chat.Chat

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/f340e606-31f0-11ed-ae93-eba096967cda",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name": "update name", "detail": "update detail"}`),
			},

			uuid.FromStringOrNil("f340e606-31f0-11ed-ae93-eba096967cda"),
			"update name",
			"update detail",

			&chat.Chat{
				ID: uuid.FromStringOrNil("b3f5c700-31f0-11ed-b1a2-bb3854582a08"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b3f5c700-31f0-11ed-b1a2-bb3854582a08","customer_id":"00000000-0000-0000-0000-000000000000","type":"","owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockChat.EXPECT().UpdateBasicInfo(gomock.Any(), tt.chatID, tt.updateName, tt.updateDetail).Return(tt.responseChat, nil)

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

func Test_v1ChatsIDOwnerIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		chatID  uuid.UUID
		ownerID uuid.UUID

		responseChat *chat.Chat

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/b3f5c700-31f0-11ed-b1a2-bb3854582a08/owner_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"owner_id": "b45cd102-31f0-11ed-9cd9-3fa1a0f883ef"}`),
			},

			uuid.FromStringOrNil("b3f5c700-31f0-11ed-b1a2-bb3854582a08"),
			uuid.FromStringOrNil("b45cd102-31f0-11ed-9cd9-3fa1a0f883ef"),

			&chat.Chat{
				ID: uuid.FromStringOrNil("b3f5c700-31f0-11ed-b1a2-bb3854582a08"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b3f5c700-31f0-11ed-b1a2-bb3854582a08","customer_id":"00000000-0000-0000-0000-000000000000","type":"","owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockChat.EXPECT().UpdateOwnerID(gomock.Any(), tt.chatID, tt.ownerID).Return(tt.responseChat, nil)

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

func Test_v1ChatsIDParticipantIDsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		chatID        uuid.UUID
		participantID uuid.UUID

		responseChat *chat.Chat

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/4131ccfa-31e1-11ed-8ae8-ef5f171c9c8e/participant_ids",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"participant_id": "417cdce0-31e1-11ed-b9f2-e3db93506530"}`),
			},

			uuid.FromStringOrNil("4131ccfa-31e1-11ed-8ae8-ef5f171c9c8e"),
			uuid.FromStringOrNil("417cdce0-31e1-11ed-b9f2-e3db93506530"),

			&chat.Chat{
				ID: uuid.FromStringOrNil("4131ccfa-31e1-11ed-8ae8-ef5f171c9c8e"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4131ccfa-31e1-11ed-8ae8-ef5f171c9c8e","customer_id":"00000000-0000-0000-0000-000000000000","type":"","owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockChat.EXPECT().AddParticipantID(gomock.Any(), tt.chatID, tt.participantID).Return(tt.responseChat, nil)

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

func Test_v1ChatsIDParticipantIDsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		chatID        uuid.UUID
		participantID uuid.UUID

		responseChat *chat.Chat

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/1a023c40-31e2-11ed-a0dd-3770f2201744/participant_ids/1a3923e0-31e2-11ed-93b4-cb9081f9b4ee",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("1a023c40-31e2-11ed-a0dd-3770f2201744"),
			uuid.FromStringOrNil("1a3923e0-31e2-11ed-93b4-cb9081f9b4ee"),

			&chat.Chat{
				ID: uuid.FromStringOrNil("1a023c40-31e2-11ed-a0dd-3770f2201744"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1a023c40-31e2-11ed-a0dd-3770f2201744","customer_id":"00000000-0000-0000-0000-000000000000","type":"","owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockChat.EXPECT().RemoveParticipantID(gomock.Any(), tt.chatID, tt.participantID).Return(tt.responseChat, nil)

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
