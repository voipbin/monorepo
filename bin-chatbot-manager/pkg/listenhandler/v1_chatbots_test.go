package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
)

func Test_processV1ChatbotsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseChatbots []*chatbot.Chatbot

		expectCustomerID uuid.UUID
		expectPageSize   uint64
		expectPageToken  string
		expectFilters    map[string]string
		expectRes        *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/chatbots?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=24676972-7f49-11ec-bc89-b7d33e9d3ea8&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			[]*chatbot.Chatbot{
				{
					ID: uuid.FromStringOrNil("0b61dcbe-a770-11ed-bab4-2fc1dac66672"),
				},
				{
					ID: uuid.FromStringOrNil("0bbe1dee-a770-11ed-b455-cbb60d5dd90b"),
				},
			},

			uuid.FromStringOrNil("24676972-7f49-11ec-bc89-b7d33e9d3ea8"),
			10,
			"2020-05-03 21:35:02.809",
			map[string]string{
				"deleted": "false",
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0b61dcbe-a770-11ed-bab4-2fc1dac66672","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","init_prompt":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"0bbe1dee-a770-11ed-b455-cbb60d5dd90b","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","init_prompt":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				chatbotHandler: mockChatbot,
			}

			mockChatbot.EXPECT().Gets(gomock.Any(), tt.expectCustomerID, tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.responseChatbots, nil)
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

func Test_processV1ChatbotsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseChatbot *chatbot.Chatbot

		expectCustomerID uuid.UUID
		expectName       string
		expectDetail     string
		expectEngineType chatbot.EngineType
		expectInitPrompt string
		expectRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/chatbots",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "58e7502c-a770-11ed-9b86-7fabe2dba847", "name": "test name", "detail": "test detail", "engine_type":"chatGPT", "init_prompt": "test init prompt"}`),
			},

			responseChatbot: &chatbot.Chatbot{
				ID: uuid.FromStringOrNil("59230ca2-a770-11ed-b5dd-2783587ed477"),
			},

			expectCustomerID: uuid.FromStringOrNil("58e7502c-a770-11ed-9b86-7fabe2dba847"),
			expectName:       "test name",
			expectDetail:     "test detail",
			expectEngineType: chatbot.EngineTypeChatGPT,
			expectInitPrompt: "test init prompt",
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"59230ca2-a770-11ed-b5dd-2783587ed477","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","init_prompt":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				chatbotHandler: mockChatbot,
			}

			mockChatbot.EXPECT().Create(gomock.Any(), tt.expectCustomerID, tt.expectName, tt.expectDetail, tt.expectEngineType, tt.expectInitPrompt).Return(tt.responseChatbot, nil)
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

func Test_processV1ChatbotsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseChatbot *chatbot.Chatbot

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/chatbots/de740384-a770-11ed-afab-5f9c8a447889",
				Method: sock.RequestMethodGet,
			},

			&chatbot.Chatbot{
				ID: uuid.FromStringOrNil("de740384-a770-11ed-afab-5f9c8a447889"),
			},

			uuid.FromStringOrNil("de740384-a770-11ed-afab-5f9c8a447889"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"de740384-a770-11ed-afab-5f9c8a447889","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","init_prompt":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				chatbotHandler: mockChatbot,
			}

			mockChatbot.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseChatbot, nil)
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

func Test_processV1ChatbotsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseChatbot *chatbot.Chatbot

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/chatbots/de99e522-a770-11ed-a0ab-5b39ee2db203",
				Method: sock.RequestMethodDelete,
			},

			&chatbot.Chatbot{
				ID: uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203"),
			},

			uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"de99e522-a770-11ed-a0ab-5b39ee2db203","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","init_prompt":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				chatbotHandler: mockChatbot,
			}

			mockChatbot.EXPECT().Delete(gomock.Any(), tt.expectID).Return(tt.responseChatbot, nil)
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

func Test_processV1ChatbotsIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseChatbot *chatbot.Chatbot

		expectID         uuid.UUID
		expectName       string
		expectDetail     string
		expectEngineType chatbot.EngineType
		expectInitPrompt string
		expectRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/chatbots/fa4d3b6a-f82f-11ed-9176-d32f5705e10c",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"new name","detail":"new detail","engine_type":"chatGPT","init_prompt":"new prompt"}`),
			},

			responseChatbot: &chatbot.Chatbot{
				ID: uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
			},

			expectID:         uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
			expectName:       "new name",
			expectDetail:     "new detail",
			expectEngineType: chatbot.EngineTypeChatGPT,
			expectInitPrompt: "new prompt",

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"fa4d3b6a-f82f-11ed-9176-d32f5705e10c","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","init_prompt":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				chatbotHandler: mockChatbot,
			}

			mockChatbot.EXPECT().Update(gomock.Any(), tt.expectID, tt.expectName, tt.expectDetail, tt.expectEngineType, tt.expectInitPrompt).Return(tt.responseChatbot, nil)
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
