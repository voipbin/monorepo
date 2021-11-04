package requesthandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

func TestCFConferenceDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:            mockSock,
		exchangeDelay:   "bin-manager.delay",
		queueCall:       "bin-manager.call-manager.request",
		queueFlow:       "bin-manager.flow-manager.request",
		queueConference: "bin-manager.conference-manager.request",
	}

	type test struct {
		name string

		conferenceID  uuid.UUID
		response      *rabbitmqhandler.Response
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("2d9227a4-3d17-11ec-ab43-cfdad30eccdf"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/2d9227a4-3d17-11ec-ab43-cfdad30eccdf",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
				Data:     []byte(``),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CFConferenceDelete(tt.conferenceID); err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
		})
	}
}

func TestCFConferenceGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:            mockSock,
		exchangeDelay:   "bin-manager.delay",
		queueCall:       "bin-manager.call-manager.request",
		queueFlow:       "bin-manager.flow-manager.request",
		queueConference: "bin-manager.conference-manager.request",
	}

	type test struct {
		name string

		conferenceID     uuid.UUID
		response         *rabbitmqhandler.Response
		expectTarget     string
		expectRequest    *rabbitmqhandler.Request
		expectConference *cfconference.Conference
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("2d9227a4-3d17-11ec-ab43-cfdad30eccdf"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"eef5c414-3d17-11ec-8db4-83a569efe7a7","name":"test","user_id":1}`),
			},
			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/2d9227a4-3d17-11ec-ab43-cfdad30eccdf",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(``),
			},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("eef5c414-3d17-11ec-8db4-83a569efe7a7"),
				UserID: 1,
				Name:   "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			cf, err := reqHandler.CFConferenceGet(tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConference, cf)
			}
		})
	}
}

func TestCFConferenceCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:            mockSock,
		exchangeDelay:   "bin-manager.delay",
		queueCall:       "bin-manager.call-manager.request",
		queueFlow:       "bin-manager.flow-manager.request",
		queueConference: "bin-manager.conference-manager.request",
	}

	type test struct {
		name string

		response         *rabbitmqhandler.Response
		expectTarget     string
		expectRequest    *rabbitmqhandler.Request
		expectConference *cfconference.Conference
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"04432fd6-3d19-11ec-8ad9-43e6162f0953","name":"test","detail":"test detail","user_id":1,"timeout":86400000,"type":"connect"}`),
			},
			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"connect","user_id":1,"name":"test","detail":"test detail","timeout":86400000,"webhook_uri":"","data":null,"pre_actions":null,"post_actions":null}`),
			},
			&cfconference.Conference{
				ID:      uuid.FromStringOrNil("04432fd6-3d19-11ec-8ad9-43e6162f0953"),
				UserID:  1,
				Type:    cfconference.TypeConnect,
				Name:    "test",
				Detail:  "test detail",
				Timeout: 86400000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			cf, err := reqHandler.CFConferenceCreate(tt.expectConference.UserID, tt.expectConference.Type, tt.expectConference.Name, tt.expectConference.Detail, tt.expectConference.Timeout)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConference, cf)
			}
		})
	}
}
