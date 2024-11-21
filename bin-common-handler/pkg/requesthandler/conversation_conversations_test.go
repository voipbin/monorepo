package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ConversationV1ConversationsGet(t *testing.T) {

	type test struct {
		name string

		conversationID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *cvconversation.Conversation
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),

			"bin-manager.conversation-manager.request",
			&sock.Request{
				URI:      "/v1/conversations/72179880-ec5f-11ec-920e-c77279756b6d",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d"}`),
			},
			&cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
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

			res, err := reqHandler.ConversationV1ConversationGet(ctx, tt.conversationID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ConversationV1ConversationGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []cvconversation.Conversation
	}{
		{
			"normal",

			"2021-03-02 03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/conversations?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
			"bin-manager.conversation-manager.request",
			&sock.Request{
				URI:      "/v1/conversations?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"30071608-7e43-11ec-b04a-bb4270e3e223"},{"id":"5ca81a9a-7e43-11ec-b271-5b65823bfdd3"}]`),
			},

			[]cvconversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("30071608-7e43-11ec-b04a-bb4270e3e223"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5ca81a9a-7e43-11ec-b271-5b65823bfdd3"),
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

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConversationV1ConversationGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
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
			"normal",

			uuid.FromStringOrNil("e8b821ba-ec61-11ec-a892-ffa25490c095"),
			"hello world.",
			[]cvmedia.Media{},

			"bin-manager.conversation-manager.request",
			&sock.Request{
				URI:      "/v1/conversations/e8b821ba-ec61-11ec-a892-ffa25490c095/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"text":"hello world.","medias":[]}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6d5ed26a-ec62-11ec-9aaa-7b9dc8a28675"}`),
			},

			&cvmessage.Message{
				ID: uuid.FromStringOrNil("6d5ed26a-ec62-11ec-9aaa-7b9dc8a28675"),
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

func Test_ConversationV1ConversationMessageGetsByConversationID(t *testing.T) {

	tests := []struct {
		name string

		conversationID uuid.UUID
		pageToken      string
		pageSize       uint64

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []cvmessage.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("a43e7c74-ec60-11ec-b1af-c73ec1bcf7cd"),
			"2021-03-02 03:23:20.995000",
			10,

			"bin-manager.conversation-manager.request",
			&sock.Request{
				URI:      "/v1/conversations/a43e7c74-ec60-11ec-b1af-c73ec1bcf7cd/messages?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"1489cedc-ec63-11ec-995f-9361e44de4ab"},{"id":"14b9f530-ec63-11ec-961e-2fc971635023"}]`),
			},

			[]cvmessage.Message{
				{
					ID: uuid.FromStringOrNil("1489cedc-ec63-11ec-995f-9361e44de4ab"),
				},
				{
					ID: uuid.FromStringOrNil("14b9f530-ec63-11ec-961e-2fc971635023"),
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

			res, err := reqHandler.ConversationV1ConversationMessageGetsByConversationID(ctx, tt.conversationID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationV1ConversationUpdate(t *testing.T) {

	tests := []struct {
		name string

		conversationID   uuid.UUID
		conversationName string
		detail           string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cvconversation.Conversation
	}{
		{
			name: "normal",

			conversationID:   uuid.FromStringOrNil("1397bde6-007a-11ee-903f-4b1fc025c9a9"),
			conversationName: "test name",
			detail:           "test detail",

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conversations/1397bde6-007a-11ee-903f-4b1fc025c9a9",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test name","detail":"test detail"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1397bde6-007a-11ee-903f-4b1fc025c9a9"}`),
			},

			expectRes: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1397bde6-007a-11ee-903f-4b1fc025c9a9"),
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

			res, err := reqHandler.ConversationV1ConversationUpdate(ctx, tt.conversationID, tt.conversationName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
