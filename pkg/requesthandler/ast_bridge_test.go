package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_AstBridgeGet(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		bridgeID   string
		response   *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectBridge  *cmbridge.Bridge
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"261a2496-dc28-11ea-b3b2-afa07bdffeb2",
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"3e6eec96-fabe-4041-870d-e1daee11aafb","technology":"softmix","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"reference_type=confbridge,reference_id=60d7ee79-78f5-4c86-9d34-4c699e8d5ee7","channels":[],"creationtime":"2020-08-10T22:50:28.085+0000","video_mode":"sfu"}`),
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmqhandler.Request{
				URI:      "/ari/bridges/261a2496-dc28-11ea-b3b2-afa07bdffeb2",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     nil,
			},
			&cmbridge.Bridge{
				ID:   "3e6eec96-fabe-4041-870d-e1daee11aafb",
				Name: "reference_type=confbridge,reference_id=60d7ee79-78f5-4c86-9d34-4c699e8d5ee7",

				// info
				Type:    cmbridge.TypeMixing,
				Tech:    cmbridge.TechSoftmix,
				Class:   "stasis",
				Creator: "Stasis",

				VideoMode: "sfu",
				// VideoSourceID: "",

				ChannelIDs: []string{},

				// conference info
				ReferenceType: cmbridge.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("60d7ee79-78f5-4c86-9d34-4c699e8d5ee7"),

				TMCreate: "",
				TMUpdate: "",
				TMDelete: "",
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

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			res, err := reqHandler.AstBridgeGet(context.Background(), tt.asteriskID, tt.bridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectBridge, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectBridge, res)
			}

		})
	}
}
