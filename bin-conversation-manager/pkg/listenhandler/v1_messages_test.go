package listenhandler

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_processV1MessagesPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseMessage *message.Message

		expectedConversationID uuid.UUID
		expectedText           string
		expectedRes            *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"conversation_id":"f6d0a7a0-1bd3-11f0-9e4d-6f54fac53b80","text":"I'm your father"}`),
			},

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f702762c-1bd3-11f0-8f2b-9f2159c784a9"),
				},
			},

			expectedConversationID: uuid.FromStringOrNil("f6d0a7a0-1bd3-11f0-9e4d-6f54fac53b80"),
			expectedText:           "I'm your father",
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f702762c-1bd3-11f0-8f2b-9f2159c784a9","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().MessageSend(gomock.Any(), tt.expectedConversationID, tt.expectedText, nil).Return(tt.responseMessage, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_processV1MessagesGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		expectFilters    map[message.Field]any
		responseMessages []*message.Message

		expectPageSize   uint64
		expectPageToken  string
		expectedResponse *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/messages?page_size=10&page_token=2021-03-01T03:30:17.000000Z",
				Method:   sock.RequestMethodGet,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"conversation_id":"22f83522-0a74-4a91-813b-1fc45e5bd9fa","deleted":false}`),
			},

			expectFilters: map[message.Field]any{
				message.FieldConversationID: uuid.FromStringOrNil("22f83522-0a74-4a91-813b-1fc45e5bd9fa"),
				message.FieldDeleted:        false,
			},
			responseMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ca8db014-1a1f-407e-9443-54282b975e40"),
					},
				},
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-01T03:30:17.000000Z",
			expectedResponse: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ca8db014-1a1f-407e-9443-54282b975e40","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
		{
			name: "multiple results",

			request: &sock.Request{
				URI:      "/v1/messages?page_size=10&page_token=2021-03-01T03:30:17.000000Z",
				Method:   sock.RequestMethodGet,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"conversation_id":"813840b3-9055-449c-b97b-558a0472f6bb","deleted":false}`),
			},

			expectFilters: map[message.Field]any{
				message.FieldConversationID: uuid.FromStringOrNil("813840b3-9055-449c-b97b-558a0472f6bb"),
				message.FieldDeleted:        false,
			},
			responseMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8956ba14-7ac4-45f6-b0fb-97af3a1d2520"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a72d1b51-aa78-4e86-928d-a18c40da4cac"),
					},
				},
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-01T03:30:17.000000Z",
			expectedResponse: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"8956ba14-7ac4-45f6-b0fb-97af3a1d2520","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null},{"id":"a72d1b51-aa78-4e86-928d-a18c40da4cac","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				utilHandler:         mockUtil,
				sockHandler:         mockSock,
				conversationHandler: mockConversation,
				messageHandler:      mockMessage,
			}

			mockMessage.EXPECT().List(gomock.Any(), tt.expectPageToken, tt.expectPageSize, gomock.Any()).Return(tt.responseMessages, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedResponse, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedResponse, res)
			}
		})
	}
}

func Test_processV1MessagesCreatePost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseMessage *message.Message

		expectedID             uuid.UUID
		expectedCustomerID     uuid.UUID
		expectedConversationID uuid.UUID
		expectedDirection      message.Direction
		expectedStatus         message.Status
		expectedReferenceType  message.ReferenceType
		expectedReferenceID    uuid.UUID
		expectedTransactionID  string
		expectedText           string
		expectedMedias         []media.Media

		expectedRes *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/messages/create",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"34e82c6e-1bcc-11f0-9c76-0fef42a4c318","customer_id":"456609ea-fecc-11ed-a717-5f6984c51794","conversation_id":"586f3930-1adb-11f0-b87e-67f6dad44afa","direction":"incoming","status":"done","reference_type":"line","reference_id":"58a1726a-1adb-11f0-b618-979f6c3070ea","transaction_id":"58caa388-1adb-11f0-a5f0-7f93f53de671","text":"hello world","medias":[]}`),
			},

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("34e82c6e-1bcc-11f0-9c76-0fef42a4c318"),
				},
			},

			expectedID:             uuid.FromStringOrNil("34e82c6e-1bcc-11f0-9c76-0fef42a4c318"),
			expectedCustomerID:     uuid.FromStringOrNil("456609ea-fecc-11ed-a717-5f6984c51794"),
			expectedConversationID: uuid.FromStringOrNil("586f3930-1adb-11f0-b87e-67f6dad44afa"),
			expectedDirection:      message.DirectionIncoming,
			expectedStatus:         message.StatusDone,
			expectedReferenceType:  message.ReferenceTypeLine,
			expectedReferenceID:    uuid.FromStringOrNil("58a1726a-1adb-11f0-b618-979f6c3070ea"),
			expectedTransactionID:  "58caa388-1adb-11f0-a5f0-7f93f53de671",
			expectedText:           "hello world",
			expectedMedias:         []media.Media{},

			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"34e82c6e-1bcc-11f0-9c76-0fef42a4c318","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				accountHandler:      mockAccount,
				conversationHandler: mockConversation,
				messageHandler:      mockMessage,
			}

			mockMessage.EXPECT().Create(
				gomock.Any(),
				tt.expectedID,
				tt.expectedCustomerID,
				tt.expectedConversationID,
				tt.expectedDirection,
				tt.expectedStatus,
				tt.expectedReferenceType,
				tt.expectedReferenceID,
				tt.expectedTransactionID,
				tt.expectedText,
				tt.expectedMedias,
			).Return(tt.responseMessage, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}
