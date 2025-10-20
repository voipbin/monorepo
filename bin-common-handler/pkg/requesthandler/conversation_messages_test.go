package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ConversationV1MessageGet(t *testing.T) {

	type test struct {
		name string

		messageID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *cvmessage.Message
	}

	tests := []test{
		{
			name: "normal",

			messageID: uuid.FromStringOrNil("8a910178-1ae0-11f0-ab95-67636c5f3084"),

			expectQueue: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/messages/8a910178-1ae0-11f0-ab95-67636c5f3084",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"8a910178-1ae0-11f0-ab95-67636c5f3084"}`),
			},
			expectRes: &cvmessage.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8a910178-1ae0-11f0-ab95-67636c5f3084"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConversationV1MessageGet(ctx, tt.messageID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ConversationV1MessageSend(t *testing.T) {

	tests := []struct {
		name string

		conversationID uuid.UUID
		text           string
		medias         []cvmedia.Media

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cvmessage.Message
	}{
		{
			name: "normal",

			conversationID: uuid.FromStringOrNil("e8b821ba-ec61-11ec-a892-ffa25490c095"),
			text:           "hello world.",
			medias:         []cvmedia.Media{},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6d5ed26a-ec62-11ec-9aaa-7b9dc8a28675"}`),
			},

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"conversation_id":"e8b821ba-ec61-11ec-a892-ffa25490c095","text":"hello world."}`),
			},
			expectRes: &cvmessage.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("6d5ed26a-ec62-11ec-9aaa-7b9dc8a28675"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConversationV1MessageSend(ctx, tt.conversationID, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationV1MessageGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[cvmessage.Field]any

		response *sock.Response

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		expectRes     []cvmessage.Message
	}{
		{
			name: "normal",

			pageToken: "2021-03-02 03:23:20.995000",
			pageSize:  10,
			filters: map[cvmessage.Field]any{
				cvmessage.FieldDeleted: false,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"8ad602dc-1ae0-11f0-8b78-57f87f455d8c"},{"id":"8afa8b48-1ae0-11f0-8e81-4715950aa160"}]`),
			},

			expectURL:    "/v1/messages?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/messages?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"deleted":false}`),
			},
			expectRes: []cvmessage.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("8ad602dc-1ae0-11f0-8b78-57f87f455d8c"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("8afa8b48-1ae0-11f0-8e81-4715950aa160"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConversationV1MessageGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationV1MessageCreate(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		customerID     uuid.UUID
		conversationID uuid.UUID
		direction      cvmessage.Direction
		status         cvmessage.Status
		referenceType  cvmessage.ReferenceType
		referenceID    uuid.UUID
		transactionID  string
		text           string
		medias         []cvmedia.Media

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cvmessage.Message
	}{
		{
			name: "normal",

			id:             uuid.FromStringOrNil("3edc5b3c-1bd6-11f0-8371-6725df99009d"),
			customerID:     uuid.FromStringOrNil("8c9e3e90-1acc-11f0-8112-a7bddc5a51fd"),
			conversationID: uuid.FromStringOrNil("55653a04-1ae1-11f0-82c9-473cc412083c"),
			direction:      cvmessage.DirectionIncoming,
			status:         cvmessage.StatusDone,
			referenceType:  cvmessage.ReferenceTypeMessage,
			referenceID:    uuid.FromStringOrNil("559292e2-1ae1-11f0-85f9-1fe5ad4a6e8b"),
			transactionID:  "55ee8b88-1ae1-11f0-9d84-33e1b1016fc7",
			text:           "hello world",
			medias:         []cvmedia.Media{},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"55c101f4-1ae1-11f0-94f2-5b0406640c9b"}`),
			},

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/messages/create",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"3edc5b3c-1bd6-11f0-8371-6725df99009d","customer_id":"8c9e3e90-1acc-11f0-8112-a7bddc5a51fd","conversation_id":"55653a04-1ae1-11f0-82c9-473cc412083c","direction":"incoming","status":"done","reference_type":"message","reference_id":"559292e2-1ae1-11f0-85f9-1fe5ad4a6e8b","transaction_id":"55ee8b88-1ae1-11f0-9d84-33e1b1016fc7","text":"hello world"}`),
			},
			expectRes: &cvmessage.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("55c101f4-1ae1-11f0-94f2-5b0406640c9b"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConversationV1MessageCreate(
				ctx,
				tt.id,
				tt.customerID,
				tt.conversationID,
				tt.direction,
				tt.status,
				tt.referenceType,
				tt.referenceID,
				tt.transactionID,
				tt.text,
				tt.medias,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
