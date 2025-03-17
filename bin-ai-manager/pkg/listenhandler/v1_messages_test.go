package listenhandler

import (
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/chatbotcallhandler"
	"monorepo/bin-ai-manager/pkg/chatbothandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_processV1MessagesGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseMessages []*message.Message

		expectChatbotcallID uuid.UUID
		expectPageSize      uint64
		expectPageToken     string
		expectFilters       map[string]string
		expectRes           *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/messages?page_size=10&page_token=2020-05-03%2021:35:02.809&chatbotcall_id=445110a0-f25d-11ef-9ff1-2f4ea94a72ac&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			responseMessages: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("829d8866-f25d-11ef-9b3a-dbb10220cf40"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("82d92b5a-f25d-11ef-8ee2-db3e98ba4d00"),
					},
				},
			},

			expectChatbotcallID: uuid.FromStringOrNil("445110a0-f25d-11ef-9ff1-2f4ea94a72ac"),
			expectPageSize:      10,
			expectPageToken:     "2020-05-03 21:35:02.809",
			expectFilters: map[string]string{
				"deleted": "false",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"829d8866-f25d-11ef-9b3a-dbb10220cf40","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000"},{"id":"82d92b5a-f25d-11ef-8ee2-db3e98ba4d00","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				chatbotHandler: mockChatbot,
				messageHandler: mockMessage,
			}

			mockMessage.EXPECT().Gets(gomock.Any(), tt.expectChatbotcallID, tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.responseMessages, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1MessagesPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseMessage *message.Message

		expectChatbotcallID uuid.UUID
		expectRole          message.Role
		expectContent       string
		expectRes           *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"chatbotcall_id": "0615dec8-f25e-11ef-b878-0fb5e7ac2aee", "role": "user", "content": "hello world!"}`),
			},

			responseMessage: &message.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("532777f8-f25e-11ef-8618-1f7ef1d18d75"),
				},
			},

			expectChatbotcallID: uuid.FromStringOrNil("0615dec8-f25e-11ef-b878-0fb5e7ac2aee"),
			expectRole:          message.RoleUser,
			expectContent:       "hello world!",
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"532777f8-f25e-11ef-8618-1f7ef1d18d75","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				chatbotHandler: mockChatbot,
				messageHandler: mockMessage,
			}

			mockMessage.EXPECT().Send(gomock.Any(), tt.expectChatbotcallID, tt.expectRole, tt.expectContent).Return(tt.responseMessage, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1MessagesIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseMessage *message.Message

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/messages/e56b73fa-f2c0-11ef-a99b-fb5e1f39d249",
				Method: sock.RequestMethodGet,
			},

			responseMessage: &message.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e56b73fa-f2c0-11ef-a99b-fb5e1f39d249"),
				},
			},

			expectID: uuid.FromStringOrNil("e56b73fa-f2c0-11ef-a99b-fb5e1f39d249"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e56b73fa-f2c0-11ef-a99b-fb5e1f39d249","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				chatbotcallHandler: mockChatbotcall,
				messageHandler:     mockMessage,
			}

			mockMessage.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseMessage, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
