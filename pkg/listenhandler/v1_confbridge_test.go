package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestProcessV1ConfbridgePost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		db:                mockDB,
		reqHandler:        mockReq,
		callHandler:       mockCall,
		confbridgeHandler: mockConfbridge,
	}

	type test struct {
		name         string
		conferenceID uuid.UUID
		request      *rabbitmqhandler.Request

		confbridge *confbridge.Confbridge
		expectRes  *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("2676edc0-3609-11ec-b533-bfe73425643b"),
			&rabbitmqhandler.Request{
				URI:      "/v1/confbridges",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"conference_id": "2676edc0-3609-11ec-b533-bfe73425643b"}`),
			},
			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
				ConferenceID:   uuid.FromStringOrNil("2676edc0-3609-11ec-b533-bfe73425643b"),
				BridgeID:       "73453fa8-3609-11ec-af18-075139856086",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				TMCreate:       "",
				TMUpdate:       "",
				TMDelete:       "",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"68e9edd8-3609-11ec-ad76-b72fa8f57f23","conference_id":"2676edc0-3609-11ec-b533-bfe73425643b","bridge_id":"73453fa8-3609-11ec-af18-075139856086","channel_call_ids":{},"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConfbridge.EXPECT().Create(gomock.Any(), tt.conferenceID).Return(tt.confbridge, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1ConfbridgesIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		db:                mockDB,
		reqHandler:        mockReq,
		callHandler:       mockCall,
		confbridgeHandler: mockConfbridge,
	}

	type test struct {
		name    string
		id      uuid.UUID
		request *rabbitmqhandler.Request

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("8bb55e1c-36de-11ec-9c31-afdc9b633856"),
			&rabbitmqhandler.Request{
				URI:      "/v1/confbridges/8bb55e1c-36de-11ec-9c31-afdc9b633856",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConfbridge.EXPECT().Terminate(tt.id).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match. exepct: 200, got: %v", res)
			}
		})
	}
}

func TestProcessV1ConfbridgesIDCallsIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		db:                mockDB,
		reqHandler:        mockReq,
		callHandler:       mockCall,
		confbridgeHandler: mockConfbridge,
	}

	type test struct {
		name         string
		confbridgeID uuid.UUID
		callID       uuid.UUID
		request      *rabbitmqhandler.Request

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("d0ae517a-36e0-11ec-9f1a-1f801a7883e1"),
			uuid.FromStringOrNil("d0ec7f40-36e0-11ec-9854-b716be01ecc0"),
			&rabbitmqhandler.Request{
				URI:      "/v1/confbridges/d0ae517a-36e0-11ec-9f1a-1f801a7883e1/calls/d0ec7f40-36e0-11ec-9854-b716be01ecc0",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConfbridge.EXPECT().Kick(gomock.All(), tt.confbridgeID, tt.callID).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match. exepct: 200, got: %v", res)
			}
		})
	}
}
