package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestProcessV1ConferencesPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
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
		request *rabbitmq.Request

		expectConference *conference.Conference
		expectRes        *rabbitmq.Response
	}

	tests := []test{
		{
			"conference type",
			uuid.FromStringOrNil("1a94c1e6-982e-11ea-9298-43412daaf0da"),
			&rabbitmq.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmq.RequestMethodPost,
				DataType: "application/json",
				Data:     `{"type": "conference"}`,
			},
			&conference.Conference{
				ID:        uuid.FromStringOrNil("d82ce190-9fe8-11ea-aec8-973901dd28fa"),
				Type:      conference.TypeConference,
				BridgeID:  "f1354268-9fe8-11ea-b693-3761800b29d5",
				BridgeIDs: []string{"f1354268-9fe8-11ea-b693-3761800b29d5"},
			},
			&rabbitmq.Response{
				StatusCode: 200,
				Data:       `{"id":"d82ce190-9fe8-11ea-aec8-973901dd28fa","type":"conference","bridge_id":"f1354268-9fe8-11ea-b693-3761800b29d5","status":"","name":"","detail":"","data":null,"bridge_ids":["f1354268-9fe8-11ea-b693-3761800b29d5"],"call_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConf.EXPECT().Start(conference.TypeConference, nil).Return(tt.expectConference, nil)

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

	mockSock := rabbitmq.NewMockRabbit(mc)
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
		request *rabbitmq.Request

		expectRes *rabbitmq.Response
	}

	tests := []test{
		{
			"conference type",
			uuid.FromStringOrNil("cacb6c12-a054-11ea-b1c1-87f3ae0d2b5b"),
			&rabbitmq.Request{
				URI:      "/v1/conferences/cacb6c12-a054-11ea-b1c1-87f3ae0d2b5b",
				Method:   rabbitmq.RequestMethodDelete,
				DataType: "application/json",
				Data:     "",
			},
			&rabbitmq.Response{
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

	mockSock := rabbitmq.NewMockRabbit(mc)
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
		request      *rabbitmq.Request

		expectRes *rabbitmq.Response
	}

	tests := []test{
		{
			"conference type",
			uuid.FromStringOrNil("ebabdcaa-a45a-11ea-9bcb-8b169d520839"),
			uuid.FromStringOrNil("55338534-a45a-11ea-8754-838b14c2b227"),
			&rabbitmq.Request{
				URI:      "/v1/conferences/ebabdcaa-a45a-11ea-9bcb-8b169d520839/calls/55338534-a45a-11ea-8754-838b14c2b227",
				Method:   rabbitmq.RequestMethodDelete,
				DataType: "application/json",
				Data:     "",
			},
			&rabbitmq.Response{
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

	mockSock := rabbitmq.NewMockRabbit(mc)
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
		request *rabbitmq.Request

		expectConference *conference.Conference
		expectRes        *rabbitmq.Response
	}

	tests := []test{
		{
			"conference type",
			&rabbitmq.Request{
				URI:    "/v1/conferences/e2951d7c-ac2d-11ea-8d4b-aff0e70476d6",
				Method: rabbitmq.RequestMethodGet,
			},
			&conference.Conference{
				ID:        uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Type:      conference.TypeConference,
				BridgeID:  "fea1c22c-ac2d-11ea-8a08-7f5cb36f279a",
				BridgeIDs: []string{"fea1c22c-ac2d-11ea-8a08-7f5cb36f279a"},
			},
			&rabbitmq.Response{
				StatusCode: 200,
				Data:       `{"id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","type":"conference","bridge_id":"fea1c22c-ac2d-11ea-8a08-7f5cb36f279a","status":"","name":"","detail":"","data":null,"bridge_ids":["fea1c22c-ac2d-11ea-8a08-7f5cb36f279a"],"call_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
