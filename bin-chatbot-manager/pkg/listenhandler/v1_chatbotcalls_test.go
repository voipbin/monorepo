package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/pkg/chatbotcallhandler"
)

func Test_processV1ChatbotcallsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseChatbotcalls []*chatbotcall.Chatbotcall

		expectCustomerID uuid.UUID
		expectPageSize   uint64
		expectPageToken  string
		expectFilters    map[string]string
		expectRes        *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/chatbotcalls?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=645e65c8-a773-11ed-b5ae-df76e94347ad&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			[]*chatbotcall.Chatbotcall{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("64b555fe-a773-11ed-9dc7-2fccabe21218"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("6792a0d8-a773-11ed-b28c-c79bf61e95b2"),
					},
				},
			},

			uuid.FromStringOrNil("645e65c8-a773-11ed-b5ae-df76e94347ad"),
			10,
			"2020-05-03 21:35:02.809",
			map[string]string{
				"deleted": "false",
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"64b555fe-a773-11ed-9dc7-2fccabe21218","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"},{"id":"6792a0d8-a773-11ed-b28c-c79bf61e95b2","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				chatbotcallHandler: mockChatbotcall,
			}

			mockChatbotcall.EXPECT().Gets(gomock.Any(), tt.expectCustomerID, tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.responseChatbotcalls, nil)
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

func Test_processV1ChatbotcallsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseChatbotcall *chatbotcall.Chatbotcall

		expectChatbotID     uuid.UUID
		expectActiveflowID  uuid.UUID
		expectReferenceType chatbotcall.ReferenceType
		expectReferenceID   uuid.UUID
		expectGender        chatbotcall.Gender
		expectLanguage      string

		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/chatbotcalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"chatbot_id": "f9e5ec32-ef4d-11ef-80de-8bc376898e49", "reference_type": "call", "reference_id":"fa2471be-ef4d-11ef-80b1-5bee84085737","gender":"female","language":"en-US"}`),
			},

			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("6792a0d8-a773-11ed-b28c-c79bf61e95b2"),
				},
			},

			expectChatbotID:     uuid.FromStringOrNil("f9e5ec32-ef4d-11ef-80de-8bc376898e49"),
			expectActiveflowID:  uuid.Nil,
			expectReferenceType: chatbotcall.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("fa2471be-ef4d-11ef-80b1-5bee84085737"),
			expectGender:        chatbotcall.GenderFemale,
			expectLanguage:      "en-US",
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6792a0d8-a773-11ed-b28c-c79bf61e95b2","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				chatbotcallHandler: mockChatbotcall,
			}

			mockChatbotcall.EXPECT().Start(gomock.Any(), tt.expectChatbotID, tt.expectActiveflowID, tt.expectReferenceType, tt.expectReferenceID, tt.expectGender, tt.expectLanguage).Return(tt.responseChatbotcall, nil)
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

func Test_processV1ChatbotcallsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseChatbotcall *chatbotcall.Chatbotcall

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/chatbotcalls/d9d804d8-ef03-4a23-906c-c192029b19fc",
				Method: sock.RequestMethodDelete,
			},

			&chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d9d804d8-ef03-4a23-906c-c192029b19fc"),
				},
			},

			uuid.FromStringOrNil("d9d804d8-ef03-4a23-906c-c192029b19fc"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d9d804d8-ef03-4a23-906c-c192029b19fc","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				chatbotcallHandler: mockChatbotcall,
			}

			mockChatbotcall.EXPECT().Delete(gomock.Any(), tt.expectID).Return(tt.responseChatbotcall, nil)
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

func Test_processV1ChatbotcallsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseChatbotcall *chatbotcall.Chatbotcall

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/chatbotcalls/3e349bb8-7b31-4533-8e2b-6654ebc84e3e",
				Method: sock.RequestMethodGet,
			},

			&chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("3e349bb8-7b31-4533-8e2b-6654ebc84e3e"),
				},
			},

			uuid.FromStringOrNil("3e349bb8-7b31-4533-8e2b-6654ebc84e3e"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3e349bb8-7b31-4533-8e2b-6654ebc84e3e","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				chatbotcallHandler: mockChatbotcall,
			}

			mockChatbotcall.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseChatbotcall, nil)
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

func Test_processV1ChatbotcallsIDMessagesPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseChatbotcall *chatbotcall.Chatbotcall

		expectID   uuid.UUID
		expectRole chatbotcall.MessageRole
		expectText string
		expectRes  *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/chatbotcalls/e961fcc6-efa1-11ef-8e16-db99776061e2/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"role": "user", "text": "hello world"}`),
			},

			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e961fcc6-efa1-11ef-8e16-db99776061e2"),
				},
			},

			expectID:   uuid.FromStringOrNil("e961fcc6-efa1-11ef-8e16-db99776061e2"),
			expectRole: chatbotcall.MessageRoleUser,
			expectText: "hello world",
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e961fcc6-efa1-11ef-8e16-db99776061e2","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				chatbotcallHandler: mockChatbotcall,
			}

			// mockChatbotcall.EXPECT().ChatMessageByID(gomock.Any(), tt.expectID, tt.expectRole, tt.expectText).Return(tt.responseChatbotcall, nil)
			_, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// if reflect.DeepEqual(res, tt.expectRes) != true {
			// 	t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			// }
		})
	}
}
