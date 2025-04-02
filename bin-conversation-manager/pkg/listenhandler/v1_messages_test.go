package listenhandler

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_processV1MessagesGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		expectConversationID uuid.UUID
		expectPageSize       uint64
		expectPageToken      string

		responseMessages []*message.Message

		response *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:    "/v1/messages?conversation_id=22f83522-0a74-4a91-813b-1fc45e5bd9fa&page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: sock.RequestMethodGet,
			},

			expectConversationID: uuid.FromStringOrNil("22f83522-0a74-4a91-813b-1fc45e5bd9fa"),
			expectPageSize:       10,
			expectPageToken:      "2021-03-01 03:30:17.000000",

			responseMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ca8db014-1a1f-407e-9443-54282b975e40"),
					},
				},
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ca8db014-1a1f-407e-9443-54282b975e40","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","direction":"","status":"","reference_type":"","reference_id":"","transaction_id":"","source":null,"text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name: "multiple results",

			request: &sock.Request{
				URI:    "/v1/messages?conversation_id=813840b3-9055-449c-b97b-558a0472f6bb&page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: sock.RequestMethodGet,
			},

			expectConversationID: uuid.FromStringOrNil("813840b3-9055-449c-b97b-558a0472f6bb"),
			expectPageSize:       10,
			expectPageToken:      "2021-03-01 03:30:17.000000",

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

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"8956ba14-7ac4-45f6-b0fb-97af3a1d2520","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","direction":"","status":"","reference_type":"","reference_id":"","transaction_id":"","source":null,"text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"a72d1b51-aa78-4e86-928d-a18c40da4cac","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","direction":"","status":"","reference_type":"","reference_id":"","transaction_id":"","source":null,"text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
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
