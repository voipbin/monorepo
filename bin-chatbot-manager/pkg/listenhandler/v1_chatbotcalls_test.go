package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/chatbotcallhandler"
)

func Test_processV1ChatbotcallsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseChatbotcalls []*chatbotcall.Chatbotcall

		expectCustomerID uuid.UUID
		expectPageSize   uint64
		expectPageToken  string
		expectFilters    map[string]string
		expectRes        *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/chatbotcalls?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=645e65c8-a773-11ed-b5ae-df76e94347ad&filter_deleted=false",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			[]*chatbotcall.Chatbotcall{
				{
					ID: uuid.FromStringOrNil("64b555fe-a773-11ed-9dc7-2fccabe21218"),
				},
				{
					ID: uuid.FromStringOrNil("6792a0d8-a773-11ed-b28c-c79bf61e95b2"),
				},
			},

			uuid.FromStringOrNil("645e65c8-a773-11ed-b5ae-df76e94347ad"),
			10,
			"2020-05-03 21:35:02.809",
			map[string]string{
				"deleted": "false",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"64b555fe-a773-11ed-9dc7-2fccabe21218","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","chatbot_engine_type":"","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","messages":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"6792a0d8-a773-11ed-b28c-c79bf61e95b2","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","chatbot_engine_type":"","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","messages":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)

			h := &listenHandler{
				rabbitSock:         mockSock,
				chatbotcallHandler: mockChatbotcall,
			}

			mockChatbotcall.EXPECT().Gets(gomock.Any(), tt.expectCustomerID, tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.responseChatbotcalls, nil)
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

func Test_processV1ChatbotcallsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseChatbotcall *chatbotcall.Chatbotcall

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/chatbotcalls/d9d804d8-ef03-4a23-906c-c192029b19fc",
				Method: rabbitmqhandler.RequestMethodDelete,
			},

			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("d9d804d8-ef03-4a23-906c-c192029b19fc"),
			},

			uuid.FromStringOrNil("d9d804d8-ef03-4a23-906c-c192029b19fc"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d9d804d8-ef03-4a23-906c-c192029b19fc","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","chatbot_engine_type":"","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","messages":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)

			h := &listenHandler{
				rabbitSock:         mockSock,
				chatbotcallHandler: mockChatbotcall,
			}

			mockChatbotcall.EXPECT().Delete(gomock.Any(), tt.expectID).Return(tt.responseChatbotcall, nil)
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

func Test_processV1ChatbotcallsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseChatbotcall *chatbotcall.Chatbotcall

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/chatbotcalls/3e349bb8-7b31-4533-8e2b-6654ebc84e3e",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("3e349bb8-7b31-4533-8e2b-6654ebc84e3e"),
			},

			uuid.FromStringOrNil("3e349bb8-7b31-4533-8e2b-6654ebc84e3e"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3e349bb8-7b31-4533-8e2b-6654ebc84e3e","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","chatbot_engine_type":"","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","messages":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)

			h := &listenHandler{
				rabbitSock:         mockSock,
				chatbotcallHandler: mockChatbotcall,
			}

			mockChatbotcall.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseChatbotcall, nil)
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
