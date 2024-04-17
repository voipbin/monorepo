package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	cbchatbot "monorepo/bin-chatbot-manager/models/chatbot"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_ChatbotV1ChatbotGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64
		filters    map[string]string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []cbchatbot.Chatbot
	}{
		{
			"normal",

			uuid.FromStringOrNil("83fec56f-8e28-4356-a50c-7641e39ed2df"),
			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"db662396-4449-456c-a6ee-39aa2ec30b55"},{"id":"0ea936d3-c74f-4744-8ca6-44e47178d88a"}]`),
			},

			"bin-manager.chatbot-manager.request",
			&rabbitmqhandler.Request{
				URI:    fmt.Sprintf("/v1/chatbots?page_token=%s&page_size=10&customer_id=83fec56f-8e28-4356-a50c-7641e39ed2df&filter_deleted=false", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method: rabbitmqhandler.RequestMethodGet,
			},
			[]cbchatbot.Chatbot{
				{
					ID: uuid.FromStringOrNil("db662396-4449-456c-a6ee-39aa2ec30b55"),
				},
				{
					ID: uuid.FromStringOrNil("0ea936d3-c74f-4744-8ca6-44e47178d88a"),
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

			res, err := reqHandler.ChatbotV1ChatbotGetsByCustomerID(ctx, tt.customerID, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_ChatbotV1ChatbotGet(t *testing.T) {

	type test struct {
		name      string
		chatbotID uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *cbchatbot.Chatbot
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("d628f462-cf28-47d9-ae37-c604c0ea2863"),

			"bin-manager.chatbot-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/chatbots/d628f462-cf28-47d9-ae37-c604c0ea2863",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"d628f462-cf28-47d9-ae37-c604c0ea2863"}`),
			},
			&cbchatbot.Chatbot{
				ID: uuid.FromStringOrNil("d628f462-cf28-47d9-ae37-c604c0ea2863"),
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

			res, err := reqHandler.ChatbotV1ChatbotGet(ctx, tt.chatbotID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotV1ChatbotCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		chatbotName string
		detail      string
		engineType  cbchatbot.EngineType
		initPrompt  string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cbchatbot.Chatbot
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("eeaf1e90-237a-4da5-a978-a8fc0eb691d0"),
			chatbotName: "test name",
			detail:      "test detail",
			engineType:  cbchatbot.EngineTypeChatGPT,
			initPrompt:  "test init prompt",

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e6248322-de4f-4313-bd89-f9de1c6466a8"}`),
			},

			expectTarget: "bin-manager.chatbot-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/chatbots",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"eeaf1e90-237a-4da5-a978-a8fc0eb691d0","name":"test name","detail":"test detail","engine_type":"chatGPT","init_prompt":"test init prompt"}`),
			},
			expectRes: &cbchatbot.Chatbot{
				ID: uuid.FromStringOrNil("e6248322-de4f-4313-bd89-f9de1c6466a8"),
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

			cf, err := reqHandler.ChatbotV1ChatbotCreate(ctx, tt.customerID, tt.chatbotName, tt.detail, tt.engineType, tt.initPrompt)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_ConferenceV1ChatbotDelete(t *testing.T) {

	tests := []struct {
		name string

		chatbotID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cbchatbot.Chatbot
	}{
		{
			"normal",

			uuid.FromStringOrNil("5d4c38bf-6cd5-4255-950a-9abf52704472"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"5d4c38bf-6cd5-4255-950a-9abf52704472"}`),
			},

			"bin-manager.chatbot-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/chatbots/5d4c38bf-6cd5-4255-950a-9abf52704472",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&cbchatbot.Chatbot{
				ID: uuid.FromStringOrNil("5d4c38bf-6cd5-4255-950a-9abf52704472"),
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

			res, err := reqHandler.ChatbotV1ChatbotDelete(ctx, tt.chatbotID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotV1ChatbotUpdate(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		chatbotName string
		detail      string
		engineType  cbchatbot.EngineType
		initPrompt  string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cbchatbot.Chatbot
	}{
		{
			name: "normal",

			id:          uuid.FromStringOrNil("76380ede-f84a-11ed-a288-2bf54d8b92e6"),
			chatbotName: "test name",
			detail:      "test detail",
			engineType:  cbchatbot.EngineTypeChatGPT,
			initPrompt:  "test init prompt",

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"76380ede-f84a-11ed-a288-2bf54d8b92e6"}`),
			},

			expectTarget: "bin-manager.chatbot-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/chatbots/76380ede-f84a-11ed-a288-2bf54d8b92e6",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test name","detail":"test detail","engine_type":"chatGPT","init_prompt":"test init prompt"}`),
			},
			expectRes: &cbchatbot.Chatbot{
				ID: uuid.FromStringOrNil("76380ede-f84a-11ed-a288-2bf54d8b92e6"),
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

			cf, err := reqHandler.ChatbotV1ChatbotUpdate(ctx, tt.id, tt.chatbotName, tt.detail, tt.engineType, tt.initPrompt)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}
