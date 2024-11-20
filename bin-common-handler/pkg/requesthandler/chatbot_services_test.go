package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cbchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"
	cbservice "monorepo/bin-chatbot-manager/models/service"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_ChatbotV1ServiceTypeChabotcallStart(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		chatbotID      uuid.UUID
		activeflowID   uuid.UUID
		referenceType  cbchatbotcall.ReferenceType
		referenceID    uuid.UUID
		gender         cbchatbotcall.Gender
		language       string
		requestTimeout int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cbservice.Service
	}{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("7654ef82-949f-4f0f-8711-6d1c370537be"),
			chatbotID:      uuid.FromStringOrNil("9469e101-d269-4895-9679-fe49531f7c12"),
			activeflowID:   uuid.FromStringOrNil("db21d8b6-fbab-11ed-8d21-332400f26ee4"),
			referenceType:  cbchatbotcall.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("865089bd-dc1b-45d5-89af-4a09c1d90cea"),
			gender:         "female",
			language:       "en-US",
			requestTimeout: 5000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"134c25c9-c9f9-4800-83bb-b5eaa84bb4ab"}`),
			},

			expectTarget: "bin-manager.chatbot-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/services/type/chatbotcall",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"7654ef82-949f-4f0f-8711-6d1c370537be","chatbot_id":"9469e101-d269-4895-9679-fe49531f7c12","activeflow_id":"db21d8b6-fbab-11ed-8d21-332400f26ee4","reference_type":"call","reference_id":"865089bd-dc1b-45d5-89af-4a09c1d90cea","gender":"female","language":"en-US"}`),
			},
			expectRes: &cbservice.Service{
				ID: uuid.FromStringOrNil("134c25c9-c9f9-4800-83bb-b5eaa84bb4ab"),
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

			cf, err := reqHandler.ChatbotV1ServiceTypeChabotcallStart(ctx, tt.customerID, tt.chatbotID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.gender, tt.language, tt.requestTimeout)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}
