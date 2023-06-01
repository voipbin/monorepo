package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	cvmedia "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	cvmessage "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_ConversationV1ConversationsGet(t *testing.T) {

	type test struct {
		name string

		conversationID uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *cvconversation.Conversation
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),

			"bin-manager.conversation-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conversations/72179880-ec5f-11ec-920e-c77279756b6d",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d"}`),
			},
			&cvconversation.Conversation{
				ID: uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

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

func Test_ConversationV1ConversationGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes []cvconversation.Conversation
	}{
		{
			"normal",

			uuid.FromStringOrNil("a43e7c74-ec60-11ec-b1af-c73ec1bcf7cd"),
			"2021-03-02 03:23:20.995000",
			10,

			"bin-manager.conversation-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conversations?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10&customer_id=a43e7c74-ec60-11ec-b1af-c73ec1bcf7cd",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"30071608-7e43-11ec-b04a-bb4270e3e223"},{"id":"5ca81a9a-7e43-11ec-b271-5b65823bfdd3"}]`),
			},

			[]cvconversation.Conversation{
				{
					ID: uuid.FromStringOrNil("30071608-7e43-11ec-b04a-bb4270e3e223"),
				},
				{
					ID: uuid.FromStringOrNil("5ca81a9a-7e43-11ec-b271-5b65823bfdd3"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConversationV1ConversationGetsByCustomerID(ctx, tt.customerID, tt.pageToken, tt.pageSize)
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
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *cvmessage.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("e8b821ba-ec61-11ec-a892-ffa25490c095"),
			"hello world.",
			[]cvmedia.Media{},

			"bin-manager.conversation-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conversations/e8b821ba-ec61-11ec-a892-ffa25490c095/messages",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"text":"hello world.","medias":[]}`),
			},
			&rabbitmqhandler.Response{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes []cvmessage.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("a43e7c74-ec60-11ec-b1af-c73ec1bcf7cd"),
			"2021-03-02 03:23:20.995000",
			10,

			"bin-manager.conversation-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conversations/a43e7c74-ec60-11ec-b1af-c73ec1bcf7cd/messages?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			&rabbitmqhandler.Response{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *cvconversation.Conversation
	}{
		{
			name: "normal",

			conversationID:   uuid.FromStringOrNil("1397bde6-007a-11ee-903f-4b1fc025c9a9"),
			conversationName: "test name",
			detail:           "test detail",

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/conversations/1397bde6-007a-11ee-903f-4b1fc025c9a9",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test name","detail":"test detail"}`),
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1397bde6-007a-11ee-903f-4b1fc025c9a9"}`),
			},

			expectRes: &cvconversation.Conversation{
				ID: uuid.FromStringOrNil("1397bde6-007a-11ee-903f-4b1fc025c9a9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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
