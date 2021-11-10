package requesthandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

func TestCFConferenceGet(t *testing.T) {

	type test struct {
		name         string
		conferenceID uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *cfconference.Conference
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("c337c4de-4132-11ec-b076-ab42296b65d5"),

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/c337c4de-4132-11ec-b076-ab42296b65d5",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"c337c4de-4132-11ec-b076-ab42296b65d5","flow_id":"e0e5c2ba-4132-11ec-a38b-c7c6ccec4af6"}`),
			},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("c337c4de-4132-11ec-b076-ab42296b65d5"),
				FlowID: uuid.FromStringOrNil("e0e5c2ba-4132-11ec-a38b-c7c6ccec4af6"),
			},
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:            mockSock,
		exchangeDelay:   "bin-manager.delay",
		queueCall:       "bin-manager.call-manager.request",
		queueFlow:       "bin-manager.flow-manager.request",
		queueTTS:        "bin-manager.tts-manager.request",
		queueRegistrar:  "bin-manager.registrar-manager.request",
		queueConference: "bin-manager.conference-manager.request",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CFConferenceGet(tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
