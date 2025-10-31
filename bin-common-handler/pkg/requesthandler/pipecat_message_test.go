package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
)

func Test_PipecatV1MessageSend(t *testing.T) {

	tests := []struct {
		name string

		hostID         string
		pipecatcallID  uuid.UUID
		messageID      string
		messageText    string
		runImmediately bool
		audioResponse  bool

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *pmmessage.Message

		response *sock.Response
	}{
		{
			name: "normal",

			hostID:         "1.2.3.4",
			pipecatcallID:  uuid.FromStringOrNil("23ea9586-b56f-11f0-aa50-439818383d0b"),
			messageID:      "2461ac48-b56f-11f0-a4fc-67f1da4bde52",
			messageText:    "bruce willis is a ghost",
			runImmediately: true,
			audioResponse:  true,

			expectTarget: "bin-manager.pipecat-manager.request.1.2.3.4",
			expectRequest: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"pipecatcall_id":"23ea9586-b56f-11f0-aa50-439818383d0b","message_id":"2461ac48-b56f-11f0-a4fc-67f1da4bde52","message_text":"bruce willis is a ghost","run_immediately":true,"audio_response":true}`),
			},
			expectRes: &pmmessage.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("248a9fea-b56f-11f0-9119-bfb93aab4dc1"),
				},
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"248a9fea-b56f-11f0-9119-bfb93aab4dc1"}`),
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

			res, err := reqHandler.PipecatV1MessageSend(ctx, tt.hostID, tt.pipecatcallID, tt.messageID, tt.messageText, tt.runImmediately, tt.audioResponse)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %+v, got: %+v", tt.expectRes, res)
			}
		})
	}
}
