package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/service"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/chatbotcallhandler"
)

func Test_processV1ServicesTypeChatbotcallPost(t *testing.T) {

	type test struct {
		name string

		request *rabbitmqhandler.Request

		responseService *service.Service

		expectCustomerID    uuid.UUID
		expectChatbotID     uuid.UUID
		expectActiveflowID  uuid.UUID
		expectReferenceType chatbotcall.ReferenceType
		expectReferenceID   uuid.UUID
		expectGender        chatbotcall.Gender
		expectLanguage      string

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/services/type/chatbotcall",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"71db8f9c-abde-475e-a060-dc95e63281c3","chatbot_id":"e7f085d0-c7d9-4da4-9992-eda14282cb86","activeflow_id":"80a5199e-fba5-11ed-90aa-6b9821d2ad5b","reference_type":"call","reference_id":"10662882-5ff8-4788-a605-55614dc8d330","gender":"female","language":"en-US"}`),
			},

			responseService: &service.Service{
				ID: uuid.FromStringOrNil("9d5b7e72-2cc9-4868-bfab-c8e758cd5045"),
			},

			expectCustomerID:    uuid.FromStringOrNil("71db8f9c-abde-475e-a060-dc95e63281c3"),
			expectChatbotID:     uuid.FromStringOrNil("e7f085d0-c7d9-4da4-9992-eda14282cb86"),
			expectActiveflowID:  uuid.FromStringOrNil("80a5199e-fba5-11ed-90aa-6b9821d2ad5b"),
			expectReferenceType: chatbotcall.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("10662882-5ff8-4788-a605-55614dc8d330"),
			expectGender:        chatbotcall.GenderFemale,
			expectLanguage:      "en-US",

			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9d5b7e72-2cc9-4868-bfab-c8e758cd5045","type":"","push_actions":null}`),
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

			mockChatbotcall.EXPECT().ServiceStart(gomock.Any(), tt.expectCustomerID, tt.expectChatbotID, tt.expectActiveflowID, tt.expectReferenceType, tt.expectReferenceID, tt.expectGender, tt.expectLanguage).Return(tt.responseService, nil)
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
