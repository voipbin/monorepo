package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestCMV1ConfbridgeCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *cmconfbridge.Confbridge
	}

	tests := []test{
		{
			"normal",

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/confbridges",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"700a6ca0-5ba2-11ec-98bd-a3b749617d0b","bridge_id":"70ee9650-5ba2-11ec-bc2a-032ae9e777fe","channel_call_ids":{},"recording_ids":[]}`),
			},

			&cmconfbridge.Confbridge{
				ID:             uuid.FromStringOrNil("700a6ca0-5ba2-11ec-98bd-a3b749617d0b"),
				BridgeID:       "70ee9650-5ba2-11ec-bc2a-032ae9e777fe",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CMV1ConfbridgeCreate(ctx)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
