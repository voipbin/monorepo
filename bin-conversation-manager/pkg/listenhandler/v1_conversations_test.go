package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_processV1ConversationsGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		expectPageSize  uint64
		expectPageToken string

		responseFilters       map[string]string
		responseConversations []*conversation.Conversation

		response *sock.Response
	}{
		{
			"normal",

			&sock.Request{
				URI:    "/v1/conversations?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000&filter_customer_id=64a3cbd8-e863-11ec-85de-1bcd09d3872e&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			10,
			"2021-03-01 03:30:17.000000",

			map[string]string{
				"customer_id": "ac03d4ea-7f50-11ec-908d-d39407ab524d",
				"deleted":     "false",
			},
			[]*conversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("645891fe-e863-11ec-b291-9f454e92f1bb"),
						CustomerID: uuid.FromStringOrNil("64a3cbd8-e863-11ec-85de-1bcd09d3872e"),
					},
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"645891fe-e863-11ec-b291-9f454e92f1bb","customer_id":"64a3cbd8-e863-11ec-85de-1bcd09d3872e","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
		{
			"2 results",

			&sock.Request{
				URI:    "/v1/conversations?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000&filter_customer_id=b77be746-e863-11ec-97b0-bb06bbb7db0e&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			10,
			"2021-03-01 03:30:17.000000",

			map[string]string{
				"customer_id": "b77be746-e863-11ec-97b0-bb06bbb7db0e",
				"deleted":     "false",
			},
			[]*conversation.Conversation{
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
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"b7ac843c-e863-11ec-9652-0ff162b38a15","customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000"},{"id":"c45aec8c-e863-11ec-9bae-4fcfe883444a","customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000"}]`),
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

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockConversation.EXPECT().Gets(gomock.Any(), tt.expectPageToken, tt.expectPageSize, tt.responseFilters).Return(tt.responseConversations, nil)
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
				Data:       []byte(`{"id":"73071e00-a29a-11ec-a43a-079fe08ce740","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000"}`),
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

func Test_processV1ConversationsIDMessagesGet(t *testing.T) {

	tests := []struct {
		name string

		expectConversationID uuid.UUID
		expectPageSize       uint64
		expectPageToken      string

		responseMessages []*message.Message

		request  *sock.Request
		response *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("7d83fee0-e866-11ec-bbc3-4b1ea17cf502"),
			10,
			"2021-03-01 03:30:17.000000",

			[]*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("645891fe-e863-11ec-b291-9f454e92f1bb"),
						CustomerID: uuid.FromStringOrNil("64a3cbd8-e863-11ec-85de-1bcd09d3872e"),
					},
				},
			},

			&sock.Request{
				URI:    "/v1/conversations/7d83fee0-e866-11ec-bbc3-4b1ea17cf502/messages?customer_id=64a3cbd8-e863-11ec-85de-1bcd09d3872e&page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"645891fe-e863-11ec-b291-9f454e92f1bb","customer_id":"64a3cbd8-e863-11ec-85de-1bcd09d3872e","conversation_id":"00000000-0000-0000-0000-000000000000","direction":"","status":"","reference_type":"","reference_id":"","transaction_id":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 results",

			uuid.FromStringOrNil("d341b73c-e866-11ec-8b7d-d34da2bad8d5"),
			10,
			"2021-03-01 03:30:17.000000",

			[]*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d373f1e8-e866-11ec-91ef-7711453397a7"),
						CustomerID: uuid.FromStringOrNil("b77be746-e863-11ec-97b0-bb06bbb7db0e"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d3a1f9f8-e866-11ec-a403-07ca24d89997"),
						CustomerID: uuid.FromStringOrNil("b77be746-e863-11ec-97b0-bb06bbb7db0e"),
					},
				},
			},
			&sock.Request{
				URI:    "/v1/conversations/d341b73c-e866-11ec-8b7d-d34da2bad8d5/messages?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d373f1e8-e866-11ec-91ef-7711453397a7","customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","conversation_id":"00000000-0000-0000-0000-000000000000","direction":"","status":"","reference_type":"","reference_id":"","transaction_id":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"d3a1f9f8-e866-11ec-a403-07ca24d89997","customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","conversation_id":"00000000-0000-0000-0000-000000000000","direction":"","status":"","reference_type":"","reference_id":"","transaction_id":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				conversationHandler: mockConversation,
				messageHandler:      mockMessage,
			}

			mockMessage.EXPECT().GetsByConversationID(gomock.Any(), tt.expectConversationID, tt.expectPageToken, tt.expectPageSize).Return(tt.responseMessages, nil)
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

func Test_processV1ConversationsIDMessagesPost(t *testing.T) {

	tests := []struct {
		name string

		expectReqConversationID uuid.UUID
		expectReqText           string
		expectReqMedia          []media.Media

		responseMessage *message.Message

		request  *sock.Request
		response *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("00933876-ec56-11ec-a551-1f012848b901"),
			"hello world",
			[]media.Media{},

			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bb509f64-ec56-11ec-aa8b-374ae78e9b98"),
				},
			},

			&sock.Request{
				URI:      "/v1/conversations/00933876-ec56-11ec-a551-1f012848b901/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"text":"hello world", "medias":[]}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bb509f64-ec56-11ec-aa8b-374ae78e9b98","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","direction":"","status":"","reference_type":"","reference_id":"","transaction_id":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockConversation.EXPECT().MessageSend(gomock.Any(), tt.expectReqConversationID, tt.expectReqText, tt.expectReqMedia).Return(tt.responseMessage, nil)

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

func Test_processV1ConversationsIDPut(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseConversation *conversation.Conversation

		expectConversationID uuid.UUID
		expectName           string
		expectDetail         string
		expectRes            *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations/8d8ab6ae-0074-11ee-80d0-df60c15605d7",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test name", "detail":"test detail"}`),
			},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d8ab6ae-0074-11ee-80d0-df60c15605d7"),
				},
			},

			expectConversationID: uuid.FromStringOrNil("8d8ab6ae-0074-11ee-80d0-df60c15605d7"),
			expectName:           "test name",
			expectDetail:         "test detail",

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8d8ab6ae-0074-11ee-80d0-df60c15605d7","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000"}`),
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

			mockConversation.EXPECT().Update(gomock.Any(), tt.expectConversationID, tt.expectName, tt.expectDetail).Return(tt.responseConversation, nil)

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
