package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
)

func Test_processV1ConfbridgePost(t *testing.T) {

	type test struct {
		name           string
		request        *sock.Request
		customerID     uuid.UUID
		confbridgeType confbridge.Type

		confbridge *confbridge.Confbridge
		expectRes  *sock.Response
	}

	tests := []test{
		{
			"normal type connect",
			&sock.Request{
				URI:      "/v1/confbridges",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a09c9c80-98f5-11ed-a7d4-eb729c335ae0","type":"connect"}`),
			},
			uuid.FromStringOrNil("a09c9c80-98f5-11ed-a7d4-eb729c335ae0"),
			confbridge.TypeConnect,

			&confbridge.Confbridge{
				ID:         uuid.FromStringOrNil("68e9edd8-3609-11ec-ad76-b72fa8f57f23"),
				CustomerID: uuid.FromStringOrNil("a09c9c80-98f5-11ed-a7d4-eb729c335ae0"),
				Type:       confbridge.TypeConnect,
				BridgeID:   "73453fa8-3609-11ec-af18-075139856086",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"68e9edd8-3609-11ec-ad76-b72fa8f57f23","customer_id":"a09c9c80-98f5-11ed-a7d4-eb729c335ae0","type":"connect","status":"","bridge_id":"73453fa8-3609-11ec-af18-075139856086","flags":null,"channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"normal type conference",
			&sock.Request{
				URI:      "/v1/confbridges",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"b405bcb6-98f5-11ed-8c1e-439c0a428f05","type":"conference"}`),
			},
			uuid.FromStringOrNil("b405bcb6-98f5-11ed-8c1e-439c0a428f05"),
			confbridge.TypeConference,

			&confbridge.Confbridge{
				ID:         uuid.FromStringOrNil("7a995638-977d-11ec-bd1d-6f78844899df"),
				CustomerID: uuid.FromStringOrNil("b405bcb6-98f5-11ed-8c1e-439c0a428f05"),
				Type:       confbridge.TypeConference,
				BridgeID:   "7b7c4d9e-977d-11ec-96d3-0780fcb609eb",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7a995638-977d-11ec-bd1d-6f78844899df","customer_id":"b405bcb6-98f5-11ed-8c1e-439c0a428f05","type":"conference","status":"","bridge_id":"7b7c4d9e-977d-11ec-96d3-0780fcb609eb","flags":null,"channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
			}

			mockConfbridge.EXPECT().Create(gomock.Any(), tt.customerID, tt.confbridgeType).Return(tt.confbridge, nil)

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
		request *sock.Request

		responseConfbridge *confbridge.Confbridge
		expectRes          *sock.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("8bb55e1c-36de-11ec-9c31-afdc9b633856"),
			&sock.Request{
				URI:    "/v1/confbridges/8bb55e1c-36de-11ec-9c31-afdc9b633856",
				Method: sock.RequestMethodDelete,
			},

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("8bb55e1c-36de-11ec-9c31-afdc9b633856"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8bb55e1c-36de-11ec-9c31-afdc9b633856","customer_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","bridge_id":"","flags":null,"channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
			}

			mockConfbridge.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseConfbridge, nil)

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

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

	h := &listenHandler{
		sockHandler:       mockSock,
		callHandler:       mockCall,
		confbridgeHandler: mockConfbridge,
	}

	type test struct {
		name         string
		confbridgeID uuid.UUID
		callID       uuid.UUID
		request      *sock.Request

		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("d0ae517a-36e0-11ec-9f1a-1f801a7883e1"),
			uuid.FromStringOrNil("d0ec7f40-36e0-11ec-9854-b716be01ecc0"),
			&sock.Request{
				URI:      "/v1/confbridges/d0ae517a-36e0-11ec-9f1a-1f801a7883e1/calls/d0ec7f40-36e0-11ec-9854-b716be01ecc0",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},
			&sock.Response{
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

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

	h := &listenHandler{
		sockHandler:       mockSock,
		callHandler:       mockCall,
		confbridgeHandler: mockConfbridge,
	}

	type test struct {
		name         string
		confbridgeID uuid.UUID
		callID       uuid.UUID
		request      *sock.Request

		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("d0ae517a-36e0-11ec-9f1a-1f801a7883e1"),
			uuid.FromStringOrNil("d0ec7f40-36e0-11ec-9854-b716be01ecc0"),
			&sock.Request{
				URI:      "/v1/confbridges/d0ae517a-36e0-11ec-9f1a-1f801a7883e1/calls/d0ec7f40-36e0-11ec-9854-b716be01ecc0",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     nil,
			},
			&sock.Response{
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
		request *sock.Request

		expectExternalMediaID uuid.UUID
		expectConfbridgeID    uuid.UUID
		expectExternalHost    string
		expectEncapsulation   externalmedia.Encapsulation
		expectTransport       externalmedia.Transport
		expectConnectionType  string
		expectFormat          string
		expectDirection       string

		responseConfbridge *confbridge.Confbridge
		expectRes          *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/confbridges/594c42fe-97ce-11ed-8d9f-ab7694f63546/external-media",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"external_media_id":"a8ede278-b332-11ef-9ee0-4f6311cf9409","external_host":"127.0.0.1:8080","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both"}`),
			},

			uuid.FromStringOrNil("a8ede278-b332-11ef-9ee0-4f6311cf9409"),
			uuid.FromStringOrNil("594c42fe-97ce-11ed-8d9f-ab7694f63546"),
			"127.0.0.1:8080",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&confbridge.Confbridge{
				ID:         uuid.FromStringOrNil("594c42fe-97ce-11ed-8d9f-ab7694f63546"),
				CustomerID: uuid.FromStringOrNil("474e1f30-98f7-11ed-8c35-dfd5f4d2a313"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"594c42fe-97ce-11ed-8d9f-ab7694f63546","customer_id":"474e1f30-98f7-11ed-8c35-dfd5f4d2a313","type":"","status":"","bridge_id":"","flags":null,"channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
			}

			mockConfbridge.EXPECT().ExternalMediaStart(gomock.Any(), tt.expectConfbridgeID, tt.expectExternalMediaID, tt.expectExternalHost, tt.expectEncapsulation, tt.expectTransport, tt.expectConnectionType, tt.expectFormat, tt.expectDirection).Return(tt.responseConfbridge, nil)

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
		request *sock.Request

		expectConfbridgeID uuid.UUID

		responseConfbridge *confbridge.Confbridge
		expectRes          *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/confbridges/0a3c7a34-97cf-11ed-8adf-4b1653edac02/external-media",
				Method: sock.RequestMethodDelete,
			},

			uuid.FromStringOrNil("0a3c7a34-97cf-11ed-8adf-4b1653edac02"),

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("0a3c7a34-97cf-11ed-8adf-4b1653edac02"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0a3c7a34-97cf-11ed-8adf-4b1653edac02","customer_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","bridge_id":"","flags":null,"channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
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

func Test_processV1ConfbridgesIDRecordingStartPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		responseConfbridge *confbridge.Confbridge

		expectID           uuid.UUID
		expectFormat       recording.Format
		expectEndOfSilence int
		expectEndOfKey     string
		expectDuration     int

		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/confbridges/99cc7924-996e-11ed-bc44-6fda69332002/recording_start",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"format": "wav", "end_of_silence": 1000, "end_of_key": "#", "duration": 86400}`),
			},

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("99cc7924-996e-11ed-bc44-6fda69332002"),
			},

			uuid.FromStringOrNil("99cc7924-996e-11ed-bc44-6fda69332002"),
			recording.FormatWAV,
			1000,
			"#",
			86400,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"99cc7924-996e-11ed-bc44-6fda69332002","customer_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","bridge_id":"","flags":null,"channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				confbridgeHandler: mockConfbridge,
			}

			mockConfbridge.EXPECT().RecordingStart(gomock.Any(), tt.expectID, tt.expectFormat, tt.expectEndOfSilence, tt.expectEndOfKey, tt.expectDuration).Return(tt.responseConfbridge, nil)
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

func Test_processV1ConfbridgesIDRecordingStopPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		responseCall *confbridge.Confbridge

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/confbridges/9a30f390-996e-11ed-8b2b-133b9632eea1/recording_stop",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("9a30f390-996e-11ed-8b2b-133b9632eea1"),
			},

			uuid.FromStringOrNil("9a30f390-996e-11ed-8b2b-133b9632eea1"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9a30f390-996e-11ed-8b2b-133b9632eea1","customer_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","bridge_id":"","flags":null,"channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				confbridgeHandler: mockConfbridge,
			}
			mockConfbridge.EXPECT().RecordingStop(gomock.Any(), tt.expectID).Return(tt.responseCall, nil)
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

func Test_processV1ConfbridgesIDFlagsPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		responseConfbridge *confbridge.Confbridge

		expectID   uuid.UUID
		expectFlag confbridge.Flag
		expectRes  *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/confbridges/68e83474-d7b7-11ed-a7fb-5b38b6216d42/flags",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"flag":"no_auto_leave"}`),
			},

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("68e83474-d7b7-11ed-a7fb-5b38b6216d42"),
			},

			uuid.FromStringOrNil("68e83474-d7b7-11ed-a7fb-5b38b6216d42"),
			confbridge.FlagNoAutoLeave,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"68e83474-d7b7-11ed-a7fb-5b38b6216d42","customer_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","bridge_id":"","flags":null,"channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				confbridgeHandler: mockConfbridge,
			}
			mockConfbridge.EXPECT().FlagAdd(gomock.Any(), tt.expectID, tt.expectFlag).Return(tt.responseConfbridge, nil)
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

func Test_processV1ConfbridgesIDFlagsDelete(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		responseConfbridge *confbridge.Confbridge

		expectID   uuid.UUID
		expectFlag confbridge.Flag
		expectRes  *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/confbridges/4e812144-d7b8-11ed-b8c5-cb2f9c49fce6/flags",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     []byte(`{"flag":"no_auto_leave"}`),
			},

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("4e812144-d7b8-11ed-b8c5-cb2f9c49fce6"),
			},

			uuid.FromStringOrNil("4e812144-d7b8-11ed-b8c5-cb2f9c49fce6"),
			confbridge.FlagNoAutoLeave,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4e812144-d7b8-11ed-b8c5-cb2f9c49fce6","customer_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","bridge_id":"","flags":null,"channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				confbridgeHandler: mockConfbridge,
			}
			mockConfbridge.EXPECT().FlagRemove(gomock.Any(), tt.expectID, tt.expectFlag).Return(tt.responseConfbridge, nil)
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

func Test_processV1ConfbridgesIDTerminatePost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		responseConfbridge *confbridge.Confbridge

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/confbridges/78e67099-943b-4067-88b6-337a245acbf1/terminate",
				Method: sock.RequestMethodPost,
			},

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("78e67099-943b-4067-88b6-337a245acbf1"),
			},

			uuid.FromStringOrNil("78e67099-943b-4067-88b6-337a245acbf1"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"78e67099-943b-4067-88b6-337a245acbf1","customer_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","bridge_id":"","flags":null,"channel_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				confbridgeHandler: mockConfbridge,
			}
			mockConfbridge.EXPECT().Terminating(gomock.Any(), tt.expectID).Return(tt.responseConfbridge, nil)
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

func Test_processV1ConfbridgesIDRingPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		responseConfbridge *confbridge.Confbridge

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/confbridges/df5f86c0-db8b-11ed-a3a9-2bfab0c1cf79/ring",
				Method: sock.RequestMethodPost,
			},

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("df5f86c0-db8b-11ed-a3a9-2bfab0c1cf79"),
			},

			uuid.FromStringOrNil("df5f86c0-db8b-11ed-a3a9-2bfab0c1cf79"),

			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				confbridgeHandler: mockConfbridge,
			}
			mockConfbridge.EXPECT().Ring(gomock.Any(), tt.expectID).Return(nil)
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

func Test_processV1ConfbridgesIDAnswerPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		responseConfbridge *confbridge.Confbridge

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/confbridges/dfb6fc20-db8b-11ed-a49e-9f25af48b85e/answer",
				Method: sock.RequestMethodPost,
			},

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("dfb6fc20-db8b-11ed-a49e-9f25af48b85e"),
			},

			uuid.FromStringOrNil("dfb6fc20-db8b-11ed-a49e-9f25af48b85e"),

			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				confbridgeHandler: mockConfbridge,
			}
			mockConfbridge.EXPECT().Answer(gomock.Any(), tt.expectID).Return(nil)
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
