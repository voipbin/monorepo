package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/conversationhandler"
)

func Test_processV1SetupPost(t *testing.T) {

	tests := []struct {
		name string

		expectReqCustomerID uuid.UUID
		expectReferenceType conversation.ReferenceType

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("fdca8fb4-a22b-11ec-8894-7bfd708fa894"),
			conversation.ReferenceTypeLine,

			&rabbitmqhandler.Request{
				URI:      "/v1/setup",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"fdca8fb4-a22b-11ec-8894-7bfd708fa894", "reference_type": "line"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConversation := conversationhandler.NewMockConversationHandler(mc)

			h := &listenHandler{
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().Setup(gomock.Any(), tt.expectReqCustomerID, tt.expectReferenceType).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}
