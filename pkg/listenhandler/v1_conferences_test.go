package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestProcessV1ConferencesPostTypeConference(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		db:                mockDB,
		reqHandler:        mockReq,
		callHandler:       mockCall,
		conferenceHandler: mockConf,
	}

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		expectReqConf    *conference.Conference
		expectConference *conference.Conference
		expectRes        *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"conference basic",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "type": "conference"}`),
			},
			&conference.Conference{
				UserID: 1,
				Type:   conference.TypeConference,
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("d82ce190-9fe8-11ea-aec8-973901dd28fa"),
				UserID:   1,
				Type:     conference.TypeConference,
				BridgeID: "f1354268-9fe8-11ea-b693-3761800b29d5",
				Timeout:  0,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"d82ce190-9fe8-11ea-aec8-973901dd28fa","user_id":1,"type":"conference","bridge_id":"f1354268-9fe8-11ea-b693-3761800b29d5","status":"","name":"","detail":"","data":null,"timeout":0,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"conference with all items",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "type": "conference", "name": "test conference all items", "detail": "test conference with all tiems detail", "timeout": 180}`),
			},
			&conference.Conference{
				UserID:  1,
				Type:    conference.TypeConference,
				Name:    "test conference all items",
				Detail:  "test conference with all tiems detail",
				Timeout: 180,
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("2a835238-da9c-11ea-bc7b-eb2f57685ad6"),
				UserID:   1,
				Type:     conference.TypeConference,
				BridgeID: "2f84ff66-da9c-11ea-9b90-83a7346c3e97",
				Name:     "test conference all items",
				Detail:   "test conference with all tiems detail",
				Timeout:  180,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"2a835238-da9c-11ea-bc7b-eb2f57685ad6","user_id":1,"type":"conference","bridge_id":"2f84ff66-da9c-11ea-9b90-83a7346c3e97","status":"","name":"test conference all items","detail":"test conference with all tiems detail","data":null,"timeout":180,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"conference with timeout",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "type": "conference", "timeout": 180}`),
			},
			&conference.Conference{
				UserID:  1,
				Type:    conference.TypeConference,
				Timeout: 180,
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("3402e154-da9a-11ea-a52b-2781af28f74d"),
				UserID:   1,
				Type:     conference.TypeConference,
				BridgeID: "3a6486ba-da9a-11ea-8a39-03999d98a404",
				Timeout:  180,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"3402e154-da9a-11ea-a52b-2781af28f74d","user_id":1,"type":"conference","bridge_id":"3a6486ba-da9a-11ea-8a39-03999d98a404","status":"","name":"","detail":"","data":null,"timeout":180,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"conference with name",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "type": "conference", "name": "test conference"}`),
			},
			&conference.Conference{
				UserID: 1,
				Type:   conference.TypeConference,
				Name:   "test conference",
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("9179b768-da9a-11ea-b583-c7592caaa090"),
				UserID:   1,
				Type:     conference.TypeConference,
				BridgeID: "95f5b53a-da9a-11ea-92be-23fad8a8b229",
				Name:     "test conference",
				Timeout:  0,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"9179b768-da9a-11ea-b583-c7592caaa090","user_id":1,"type":"conference","bridge_id":"95f5b53a-da9a-11ea-92be-23fad8a8b229","status":"","name":"test conference","detail":"","data":null,"timeout":0,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"conference with name and detail",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "type": "conference", "name": "test conference", "detail": "test conference detail"}`),
			},
			&conference.Conference{
				UserID: 1,
				Type:   conference.TypeConference,
				Name:   "test conference",
				Detail: "test conference detail",
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("c8fa873a-da9a-11ea-97f0-fff8a6d8aa21"),
				UserID:   1,
				Type:     conference.TypeConference,
				BridgeID: "cdc9898c-da9a-11ea-8b27-c77718b25ab9",
				Name:     "test conference",
				Detail:   "test conference detail",
				Timeout:  0,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"c8fa873a-da9a-11ea-97f0-fff8a6d8aa21","user_id":1,"type":"conference","bridge_id":"cdc9898c-da9a-11ea-8b27-c77718b25ab9","status":"","name":"test conference","detail":"test conference detail","data":null,"timeout":0,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConf.EXPECT().Start(tt.expectReqConf).Return(tt.expectConference, nil)

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

func TestProcessV1ConferencesPostTypeConnect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		db:                mockDB,
		reqHandler:        mockReq,
		callHandler:       mockCall,
		conferenceHandler: mockConf,
	}

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		expectReqConf    *conference.Conference
		expectConference *conference.Conference
		expectRes        *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"connect basic",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "type": "connect"}`),
			},
			&conference.Conference{
				UserID: 1,
				Type:   conference.TypeConnect,
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("d82ce190-9fe8-11ea-aec8-973901dd28fa"),
				UserID:   1,
				Type:     conference.TypeConnect,
				BridgeID: "f1354268-9fe8-11ea-b693-3761800b29d5",
				Timeout:  0,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"d82ce190-9fe8-11ea-aec8-973901dd28fa","user_id":1,"type":"connect","bridge_id":"f1354268-9fe8-11ea-b693-3761800b29d5","status":"","name":"","detail":"","data":null,"timeout":0,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"connect with all items",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "type": "connect", "name": "test conference all items", "detail": "test conference with all tiems detail", "timeout": 180}`),
			},
			&conference.Conference{
				UserID:  1,
				Type:    conference.TypeConnect,
				Name:    "test conference all items",
				Detail:  "test conference with all tiems detail",
				Timeout: 180,
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("2a835238-da9c-11ea-bc7b-eb2f57685ad6"),
				UserID:   1,
				Type:     conference.TypeConnect,
				BridgeID: "2f84ff66-da9c-11ea-9b90-83a7346c3e97",
				Name:     "test conference all items",
				Detail:   "test conference with all tiems detail",
				Timeout:  180,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"2a835238-da9c-11ea-bc7b-eb2f57685ad6","user_id":1,"type":"connect","bridge_id":"2f84ff66-da9c-11ea-9b90-83a7346c3e97","status":"","name":"test conference all items","detail":"test conference with all tiems detail","data":null,"timeout":180,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"conference with timeout",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "type": "connect", "timeout": 180}`),
			},
			&conference.Conference{
				UserID:  1,
				Type:    conference.TypeConnect,
				Timeout: 180,
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("3402e154-da9a-11ea-a52b-2781af28f74d"),
				UserID:   1,
				Type:     conference.TypeConnect,
				BridgeID: "3a6486ba-da9a-11ea-8a39-03999d98a404",
				Timeout:  180,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"3402e154-da9a-11ea-a52b-2781af28f74d","user_id":1,"type":"connect","bridge_id":"3a6486ba-da9a-11ea-8a39-03999d98a404","status":"","name":"","detail":"","data":null,"timeout":180,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"conference with name",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "type": "connect", "name": "test conference connect"}`),
			},
			&conference.Conference{
				UserID: 1,
				Type:   conference.TypeConnect,
				Name:   "test conference connect",
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("9179b768-da9a-11ea-b583-c7592caaa090"),
				UserID:   1,
				Type:     conference.TypeConnect,
				BridgeID: "95f5b53a-da9a-11ea-92be-23fad8a8b229",
				Name:     "test conference connect",
				Timeout:  0,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"9179b768-da9a-11ea-b583-c7592caaa090","user_id":1,"type":"connect","bridge_id":"95f5b53a-da9a-11ea-92be-23fad8a8b229","status":"","name":"test conference connect","detail":"","data":null,"timeout":0,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"conference with name and detail",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "type": "connect", "name": "test conference", "detail": "test conference detail"}`),
			},
			&conference.Conference{
				UserID: 1,
				Type:   conference.TypeConnect,
				Name:   "test conference",
				Detail: "test conference detail",
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("c8fa873a-da9a-11ea-97f0-fff8a6d8aa21"),
				UserID:   1,
				Type:     conference.TypeConnect,
				BridgeID: "cdc9898c-da9a-11ea-8b27-c77718b25ab9",
				Name:     "test conference",
				Detail:   "test conference detail",
				Timeout:  0,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"c8fa873a-da9a-11ea-97f0-fff8a6d8aa21","user_id":1,"type":"connect","bridge_id":"cdc9898c-da9a-11ea-8b27-c77718b25ab9","status":"","name":"test conference","detail":"test conference detail","data":null,"timeout":0,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConf.EXPECT().Start(tt.expectReqConf).Return(tt.expectConference, nil)

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

func TestProcessV1ConferencesIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		db:                mockDB,
		reqHandler:        mockReq,
		callHandler:       mockCall,
		conferenceHandler: mockConf,
	}

	type test struct {
		name    string
		id      uuid.UUID
		request *rabbitmqhandler.Request

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"conference type",
			uuid.FromStringOrNil("cacb6c12-a054-11ea-b1c1-87f3ae0d2b5b"),
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/cacb6c12-a054-11ea-b1c1-87f3ae0d2b5b",
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

			mockConf.EXPECT().Terminate(tt.id, gomock.Any()).Return(nil)

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

func TestProcessV1ConferencesIDCallsIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		db:                mockDB,
		reqHandler:        mockReq,
		callHandler:       mockCall,
		conferenceHandler: mockConf,
	}

	type test struct {
		name         string
		conferenceID uuid.UUID
		callID       uuid.UUID
		request      *rabbitmqhandler.Request

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"conference type",
			uuid.FromStringOrNil("ebabdcaa-a45a-11ea-9bcb-8b169d520839"),
			uuid.FromStringOrNil("55338534-a45a-11ea-8754-838b14c2b227"),
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/ebabdcaa-a45a-11ea-9bcb-8b169d520839/calls/55338534-a45a-11ea-8754-838b14c2b227",
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

			mockConf.EXPECT().Leave(tt.conferenceID, tt.callID).Return(nil)

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

func TestProcessV1ConferencesIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		db:                mockDB,
		reqHandler:        mockReq,
		callHandler:       mockCall,
		conferenceHandler: mockConf,
	}

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		expectConference *conference.Conference
		expectRes        *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"conference basic",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/e2951d7c-ac2d-11ea-8d4b-aff0e70476d6",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				UserID:   1,
				Type:     conference.TypeConference,
				BridgeID: "fea1c22c-ac2d-11ea-8a08-7f5cb36f279a",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","user_id":1,"type":"conference","bridge_id":"fea1c22c-ac2d-11ea-8a08-7f5cb36f279a","status":"","name":"","detail":"","data":null,"timeout":0,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.expectConference.ID).Return(tt.expectConference, nil)

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
