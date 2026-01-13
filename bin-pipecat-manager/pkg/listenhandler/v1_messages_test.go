package listenhandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_processV1MessagesPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseMessage *message.Message

		expectPipecatcallID  uuid.UUID
		expectMessageID      string
		expectMessageText    string
		expectRunImmediately bool
		expectAudioResponse  bool

		expectRes *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"pipecatcall_id":"9bd7ed8e-b3ab-11f0-a12a-d3f1af50fa4a", "message_id": "test message id", "message_text": "Hello, world!", "run_immediately": true, "audio_response": true}`),
			},

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9bee66ae-b3ab-11f0-8bb6-874d8fe199c7"),
				},
			},

			expectPipecatcallID:  uuid.FromStringOrNil("9bd7ed8e-b3ab-11f0-a12a-d3f1af50fa4a"),
			expectMessageID:      "test message id",
			expectMessageText:    "Hello, world!",
			expectRunImmediately: true,
			expectAudioResponse:  true,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9bee66ae-b3ab-11f0-8bb6-874d8fe199c7","customer_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","pipecatcall_reference_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				pipecatcallHandler: mockPipecatcall,
			}
			ctx := context.Background()

			mockPipecatcall.EXPECT().SendMessage(ctx, tt.expectPipecatcallID, tt.expectMessageID, tt.expectMessageText, tt.expectRunImmediately, tt.expectAudioResponse.Return(tt.responseMessage, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
