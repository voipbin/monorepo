package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
)

func Test_processV1ConfbridgePost(t *testing.T) {

	type test struct {
		name           string
		request        *rabbitmqhandler.Request
		confbridgeType confbridge.Type

		confbridge *confbridge.Confbridge
		expectRes  *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal type connect",
			&rabbitmqhandler.Request{
				URI:      "/v1/confbridges",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"connect"}`),
			},
			confbridge.TypeConnect,

			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
				Type:           confbridge.TypeConnect,
				BridgeID:       "73453fa8-3609-11ec-af18-075139856086",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				TMCreate:       "",
				TMUpdate:       "",
				TMDelete:       "",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"68e9edd8-3609-11ec-ad76-b72fa8f57f23","type":"connect","bridge_id":"73453fa8-3609-11ec-af18-075139856086","channel_call_ids":{},"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":[],"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"normal type conference",
			&rabbitmqhandler.Request{
				URI:      "/v1/confbridges",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"conference"}`),
			},
			confbridge.TypeConference,

			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("7a995638-977d-11ec-bd1d-6f78844899df"),
				Type:           confbridge.TypeConference,
				BridgeID:       "7b7c4d9e-977d-11ec-96d3-0780fcb609eb",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				TMCreate:       "",
				TMUpdate:       "",
				TMDelete:       "",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"7a995638-977d-11ec-bd1d-6f78844899df","type":"conference","bridge_id":"7b7c4d9e-977d-11ec-96d3-0780fcb609eb","channel_call_ids":{},"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":[],"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
			}

			mockConfbridge.EXPECT().Create(gomock.Any(), tt.confbridgeType).Return(tt.confbridge, nil)

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

func Test_processV1ConfbridgesIDDelete(t *testing.T) {

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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
			}

			mockConfbridge.EXPECT().Terminate(gomock.Any(), tt.id).Return(nil)

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
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
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

func TestProcessV1ConfbridgesIDCallsIDPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
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
				Method:   rabbitmqhandler.RequestMethodPost,
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

			mockConfbridge.EXPECT().Join(gomock.All(), tt.confbridgeID, tt.callID).Return(nil)

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

func Test_processV1ConfbridgesIDExternalMediaPost(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		expectConfbridgeID   uuid.UUID
		expectExternalHost   string
		expectEncapsulation  string
		expectTransport      string
		expectConnectionType string
		expectFormat         string
		expectDirection      string

		responseConfbridge *confbridge.Confbridge
		expectRes          *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/confbridges/594c42fe-97ce-11ed-8d9f-ab7694f63546/external-media",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"external_host":"127.0.0.1:8080","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both"}`),
			},

			uuid.FromStringOrNil("594c42fe-97ce-11ed-8d9f-ab7694f63546"),
			"127.0.0.1:8080",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("594c42fe-97ce-11ed-8d9f-ab7694f63546"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"594c42fe-97ce-11ed-8d9f-ab7694f63546","type":"","bridge_id":"","channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
			}

			mockConfbridge.EXPECT().ExternalMediaStart(gomock.Any(), tt.expectConfbridgeID, tt.expectExternalHost, tt.expectEncapsulation, tt.expectTransport, tt.expectConnectionType, tt.expectFormat, tt.expectDirection).Return(tt.responseConfbridge, nil)

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

func Test_processV1ConfbridgesIDExternalMediaDelete(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		expectConfbridgeID uuid.UUID

		responseConfbridge *confbridge.Confbridge
		expectRes          *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/confbridges/0a3c7a34-97cf-11ed-8adf-4b1653edac02/external-media",
				Method:   rabbitmqhandler.RequestMethodDelete,
			},

			uuid.FromStringOrNil("0a3c7a34-97cf-11ed-8adf-4b1653edac02"),

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("0a3c7a34-97cf-11ed-8adf-4b1653edac02"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0a3c7a34-97cf-11ed-8adf-4b1653edac02","type":"","bridge_id":"","channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
			}

			mockConfbridge.EXPECT().ExternalMediaStop(gomock.Any(), tt.expectConfbridgeID).Return(tt.responseConfbridge, nil)

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
