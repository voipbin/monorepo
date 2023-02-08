package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/service"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/chatbotcallhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_processV1ServicesTypeChatbotcallPost(t *testing.T) {

	type test struct {
		name string

		request *rabbitmqhandler.Request

		responseService *service.Service

		expectCustomerID    uuid.UUID
		expectChatbotID     uuid.UUID
		expectReferenceType chatbotcall.ReferenceType
		expectReferenceID   uuid.UUID
		expectGender        chatbotcall.Gender
		expectLanguage      string

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/services/type/chatbotcall",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"71db8f9c-abde-475e-a060-dc95e63281c3","chatbot_id":"e7f085d0-c7d9-4da4-9992-eda14282cb86","reference_type":"call","reference_id":"10662882-5ff8-4788-a605-55614dc8d330","gender":"female","language":"en-US"}`),
			},

			&service.Service{
				ID: uuid.FromStringOrNil("9d5b7e72-2cc9-4868-bfab-c8e758cd5045"),
			},

			uuid.FromStringOrNil("71db8f9c-abde-475e-a060-dc95e63281c3"),
			uuid.FromStringOrNil("e7f085d0-c7d9-4da4-9992-eda14282cb86"),
			chatbotcall.ReferenceTypeCall,
			uuid.FromStringOrNil("10662882-5ff8-4788-a605-55614dc8d330"),
			chatbotcall.GenderFemale,
			"en-US",

			&rabbitmqhandler.Response{
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

			mockChatbotcall.EXPECT().ServiceStart(gomock.Any(), tt.expectCustomerID, tt.expectChatbotID, tt.expectReferenceType, tt.expectReferenceID, tt.expectGender, tt.expectLanguage).Return(tt.responseService, nil)
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
