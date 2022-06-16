package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_ConversationV1Setup(t *testing.T) {

	type test struct {
		name string

		customerID    uuid.UUID
		referenceType cvconversation.ReferenceType

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("fdda7b36-ed6c-11ec-921f-03700ba6e155"),
			cvconversation.ReferenceTypeLine,

			"bin-manager.conversation-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/setup",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"fdda7b36-ed6c-11ec-921f-03700ba6e155","reference_type":"line"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
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

			if err := reqHandler.ConversationV1Setup(ctx, tt.customerID, tt.referenceType); err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

		})
	}
}
