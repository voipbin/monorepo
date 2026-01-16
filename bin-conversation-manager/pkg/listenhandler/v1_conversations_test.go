package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_processV1ConversationsGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		expectPageSize  uint64
		expectPageToken string
		expectFields    map[conversation.Field]any

		responseConversations []*conversation.Conversation

		response *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method:   sock.RequestMethodGet,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"64a3cbd8-e863-11ec-85de-1bcd09d3872e","deleted":false}`),
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-01 03:30:17.000000",
			expectFields: map[conversation.Field]any{
				conversation.FieldCustomerID: uuid.FromStringOrNil("64a3cbd8-e863-11ec-85de-1bcd09d3872e"),
				conversation.FieldDeleted:    false,
			},

			responseConversations: []*conversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("645891fe-e863-11ec-b291-9f454e92f1bb"),
						CustomerID: uuid.FromStringOrNil("64a3cbd8-e863-11ec-85de-1bcd09d3872e"),
					},
				},
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"645891fe-e863-11ec-b291-9f454e92f1bb","customer_id":"64a3cbd8-e863-11ec-85de-1bcd09d3872e","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{}}]`),
			},
		},
		{
			name: "2 results",

			request: &sock.Request{
				URI:      "/v1/conversations?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method:   sock.RequestMethodGet,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","deleted":false}`),
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-01 03:30:17.000000",
			expectFields: map[conversation.Field]any{
				conversation.FieldCustomerID: uuid.FromStringOrNil("b77be746-e863-11ec-97b0-bb06bbb7db0e"),
				conversation.FieldDeleted:    false,
			},

			responseConversations: []*conversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b7ac843c-e863-11ec-9652-0ff162b38a15"),
						CustomerID: uuid.FromStringOrNil("b77be746-e863-11ec-97b0-bb06bbb7db0e"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c45aec8c-e863-11ec-9bae-4fcfe883444a"),
						CustomerID: uuid.FromStringOrNil("b77be746-e863-11ec-97b0-bb06bbb7db0e"),
					},
				},
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"b7ac843c-e863-11ec-9652-0ff162b38a15","customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{}},{"id":"c45aec8c-e863-11ec-9bae-4fcfe883444a","customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{}}]`),
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

			h := &listenHandler{
				sockHandler:         mockSock,
				utilHandler:         mockUtil,
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().List(gomock.Any(), tt.expectPageToken, tt.expectPageSize, gomock.Any()).Return(tt.responseConversations, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

func Test_processV1ConversationsPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseConversation *conversation.Conversation

		expectedCustomerID uuid.UUID
		expectedName       string
		expectedDetail     string
		expectedType       conversation.Type
		expectedDialogID   string
		expectedSelf       commonaddress.Address
		expectedPeer       commonaddress.Address
		expectedRes        *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"456609ea-fecc-11ed-a717-5f6984c51794","name":"test name","detail":"test detail","type":"line","dialog_id":"b5404340-1acf-11f0-941a-633dfb3b6be3","self":{"type":"line","target":"b589c6f0-1acf-11f0-b1ad-a32b39bc73a2"},"peer":{"type":"line","target":"b5b28d6a-1acf-11f0-86ed-cb4575eb8b11"}}`),
			},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b5dc1f5e-1acf-11f0-8f6f-7f18c79eb9a2"),
				},
			},

			expectedCustomerID: uuid.FromStringOrNil("456609ea-fecc-11ed-a717-5f6984c51794"),
			expectedName:       "test name",
			expectedDetail:     "test detail",
			expectedType:       conversation.TypeLine,
			expectedDialogID:   "b5404340-1acf-11f0-941a-633dfb3b6be3",
			expectedSelf: commonaddress.Address{
				Type:   commonaddress.TypeLine,
				Target: "b589c6f0-1acf-11f0-b1ad-a32b39bc73a2",
			},
			expectedPeer: commonaddress.Address{
				Type:   commonaddress.TypeLine,
				Target: "b5b28d6a-1acf-11f0-86ed-cb4575eb8b11",
			},

			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b5dc1f5e-1acf-11f0-8f6f-7f18c79eb9a2","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{}}`),
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

			h := &listenHandler{
				sockHandler:         mockSock,
				accountHandler:      mockAccount,
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().Create(
				gomock.Any(),
				tt.expectedCustomerID,
				tt.expectedName,
				tt.expectedDetail,
				tt.expectedType,
				tt.expectedDialogID,
				tt.expectedSelf,
				tt.expectedPeer,
			).Return(tt.responseConversation, nil)
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

func Test_processV1ConversationsIDGet(t *testing.T) {

	tests := []struct {
		name string

		expectID uuid.UUID

		resultData *conversation.Conversation

		responseConversation *sock.Request
		response             *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("73071e00-a29a-11ec-a43a-079fe08ce740"),

			&conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("73071e00-a29a-11ec-a43a-079fe08ce740"),
				},
			},

			&sock.Request{
				URI:    "/v1/conversations/73071e00-a29a-11ec-a43a-079fe08ce740",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"73071e00-a29a-11ec-a43a-079fe08ce740","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{}}`),
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

			mockConversation.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.resultData, nil)
			res, err := h.processRequest(tt.responseConversation)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}

func Test_processV1ConversationsIDPut(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseConversation *conversation.Conversation

		expectedConversationID uuid.UUID
		expectedFields         map[conversation.Field]any
		expectRes              *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations/8d8ab6ae-0074-11ee-80d0-df60c15605d7",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test name", "detail":"test detail", "account_id": "a3f340b4-21ec-11f0-9b2a-f70f3bf0b3be"}`),
			},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d8ab6ae-0074-11ee-80d0-df60c15605d7"),
				},
			},

			expectedConversationID: uuid.FromStringOrNil("8d8ab6ae-0074-11ee-80d0-df60c15605d7"),
			expectedFields: map[conversation.Field]any{
				conversation.FieldName:      "test name",
				conversation.FieldDetail:    "test detail",
				conversation.FieldAccountID: uuid.FromStringOrNil("a3f340b4-21ec-11f0-9b2a-f70f3bf0b3be"),
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8d8ab6ae-0074-11ee-80d0-df60c15605d7","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{}}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConversation := conversationhandler.NewMockConversationHandler(mc)

			h := &listenHandler{
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().Update(gomock.Any(), tt.expectedConversationID, gomock.Any()).Return(tt.responseConversation, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
