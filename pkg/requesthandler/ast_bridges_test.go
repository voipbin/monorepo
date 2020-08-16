package requesthandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

func TestAstBridgeGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request")

	type test struct {
		name       string
		asteriskID string
		bridgeID   string
		response   *rabbitmq.Response

		expectTarget  string
		expectRequest *rabbitmq.Request
		expectBridge  *bridge.Bridge
	}

	tests := []test{
		{
			"normal",
			"00:11:22:33:44:55",
			"261a2496-dc28-11ea-b3b2-afa07bdffeb2",
			&rabbitmq.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       `{"id":"3e6eec96-fabe-4041-870d-e1daee11aafb","technology":"softmix","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"conference_type=conference,conference_id=60d7ee79-78f5-4c86-9d34-4c699e8d5ee7,join=false","channels":[],"creationtime":"2020-08-10T22:50:28.085+0000","video_mode":"sfu"}`,
			},

			"asterisk.00:11:22:33:44:55.request",
			&rabbitmq.Request{
				URI:      "/ari/bridges/261a2496-dc28-11ea-b3b2-afa07bdffeb2",
				Method:   rabbitmq.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     "",
			},
			&bridge.Bridge{
				ID:   "3e6eec96-fabe-4041-870d-e1daee11aafb",
				Name: "conference_type=conference,conference_id=60d7ee79-78f5-4c86-9d34-4c699e8d5ee7,join=false",

				// info
				Type:    bridge.TypeMixing,
				Tech:    bridge.TechSoftmix,
				Class:   "stasis",
				Creator: "Stasis",

				VideoMode: "sfu",
				// VideoSourceID: "",

				ChannelIDs: []string{},

				// conference info
				ConferenceID:   uuid.FromStringOrNil("60d7ee79-78f5-4c86-9d34-4c699e8d5ee7"),
				ConferenceType: conference.TypeConference,
				ConferenceJoin: false,

				TMCreate: "",
				TMUpdate: "",
				TMDelete: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			res, err := reqHandler.AstBridgeGet(tt.asteriskID, tt.bridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectBridge, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectBridge, res)
			}

		})
	}
}
