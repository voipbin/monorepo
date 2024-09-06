package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	cbchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ChatbotV1ChatbotcallGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64
		filters    map[string]string

		response *rabbitmqhandler.Response

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		expectResult  []cbchatbotcall.Chatbotcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("ccf7720e-4838-4f97-bb61-3021e14c185a"),
			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c3ac26c7-567c-4230-aaf8-d19b6fde4d6c"},{"id":"eb36875a-0d7a-4a8f-92a9-7551f4f29fd6"}]`),
			},

			"/v1/chatbotcalls?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&customer_id=ccf7720e-4838-4f97-bb61-3021e14c185a",
			"bin-manager.chatbot-manager.request",
			&sock.Request{
				URI:    fmt.Sprintf("/v1/chatbotcalls?page_token=%s&page_size=10&customer_id=ccf7720e-4838-4f97-bb61-3021e14c185a&filter_deleted=false", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method: sock.RequestMethodGet,
			},
			[]cbchatbotcall.Chatbotcall{
				{
					ID: uuid.FromStringOrNil("c3ac26c7-567c-4230-aaf8-d19b6fde4d6c"),
				},
				{
					ID: uuid.FromStringOrNil("eb36875a-0d7a-4a8f-92a9-7551f4f29fd6"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			h := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: h,
			}
			ctx := context.Background()

			h.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ChatbotV1ChatbotcallGetsByCustomerID(ctx, tt.customerID, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_ChatbotV1ChatbotcallGet(t *testing.T) {

	type test struct {
		name string

		chatbotcallID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *rabbitmqhandler.Response
		expectRes *cbchatbotcall.Chatbotcall
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("d3937170-ee3b-40d0-8b81-4261e5bb5ba4"),

			"bin-manager.chatbot-manager.request",
			&sock.Request{
				URI:    "/v1/chatbotcalls/d3937170-ee3b-40d0-8b81-4261e5bb5ba4",
				Method: sock.RequestMethodGet,
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"d3937170-ee3b-40d0-8b81-4261e5bb5ba4"}`),
			},
			&cbchatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("d3937170-ee3b-40d0-8b81-4261e5bb5ba4"),
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

			res, err := reqHandler.ChatbotV1ChatbotcallGet(ctx, tt.chatbotcallID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotV1ChatbotcallDelete(t *testing.T) {

	tests := []struct {
		name string

		chatbotcallID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cbchatbotcall.Chatbotcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("6078c492-25e6-4f31-baa0-2fef98379db7"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"6078c492-25e6-4f31-baa0-2fef98379db7"}`),
			},

			"bin-manager.chatbot-manager.request",
			&sock.Request{
				URI:    "/v1/chatbotcalls/6078c492-25e6-4f31-baa0-2fef98379db7",
				Method: sock.RequestMethodDelete,
			},
			&cbchatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("6078c492-25e6-4f31-baa0-2fef98379db7"),
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

			res, err := reqHandler.ChatbotV1ChatbotcallDelete(ctx, tt.chatbotcallID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
