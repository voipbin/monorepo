package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencehandler"
)

func TestProcessV1ConferencesGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		conferenceHandler: mockConf,
	}

	tests := []struct {
		name           string
		request        *rabbitmqhandler.Request
		userID         uint64
		pageSize       uint64
		pageToken      string
		conferenceType string
		confs          []*conference.Conference
		expectRes      *rabbitmqhandler.Response
	}{
		{
			"basic",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences?page_size=10&page_token=2020-05-03%2021:35:02.809&user_id=1",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			1,
			10,
			"2020-05-03 21:35:02.809",
			"",
			[]*conference.Conference{
				{
					ID:     uuid.FromStringOrNil("0addf332-9312-11eb-95e8-9b90e44428a0"),
					UserID: 1,
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0addf332-9312-11eb-95e8-9b90e44428a0","user_id":1,"confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"webhook_uri":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"have confbridge and flow id",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences?page_size=10&page_token=2020-05-03%2021:35:02.809&user_id=1",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			1,
			10,
			"2020-05-03 21:35:02.809",
			"",
			[]*conference.Conference{
				{
					ID:           uuid.FromStringOrNil("33b1138a-3bef-11ec-a187-f77a455f3ced"),
					UserID:       1,
					ConfbridgeID: uuid.FromStringOrNil("343ae074-3bef-11ec-b657-db12d3135e42"),
					FlowID:       uuid.FromStringOrNil("49da6378-3bef-11ec-88b6-f31f8c97b61b"),
					Data:         map[string]interface{}{},
					Timeout:      86400,
					PreActions:   []action.Action{},
					PostActions:  []action.Action{},
					CallIDs:      []uuid.UUID{},
					RecordingID:  [16]byte{},
					RecordingIDs: []uuid.UUID{},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"33b1138a-3bef-11ec-a187-f77a455f3ced","user_id":1,"confbridge_id":"343ae074-3bef-11ec-b657-db12d3135e42","flow_id":"49da6378-3bef-11ec-88b6-f31f8c97b61b","type":"","status":"","name":"","detail":"","data":{},"timeout":86400,"pre_actions":[],"post_actions":[],"call_ids":[],"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":[],"webhook_uri":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"have confbridge and with conference type",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences?page_size=10&page_token=2020-05-03%2021:35:02.809&user_id=1&type=conference",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			1,
			10,
			"2020-05-03 21:35:02.809",
			"conference",
			[]*conference.Conference{
				{
					ID:           uuid.FromStringOrNil("c1e0a078-3de6-11ec-ae88-13052faf6ad7"),
					UserID:       1,
					Type:         conference.TypeConference,
					ConfbridgeID: uuid.FromStringOrNil("c21b98ea-3de6-11ec-ab1e-4bcde9e784af"),
					FlowID:       uuid.FromStringOrNil("c234ce0a-3de6-11ec-8807-0b3f00d6e280"),
					Data:         map[string]interface{}{},
					Timeout:      86400,
					PreActions:   []action.Action{},
					PostActions:  []action.Action{},
					CallIDs:      []uuid.UUID{},
					RecordingID:  [16]byte{},
					RecordingIDs: []uuid.UUID{},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c1e0a078-3de6-11ec-ae88-13052faf6ad7","user_id":1,"confbridge_id":"c21b98ea-3de6-11ec-ab1e-4bcde9e784af","flow_id":"c234ce0a-3de6-11ec-8807-0b3f00d6e280","type":"conference","status":"","name":"","detail":"","data":{},"timeout":86400,"pre_actions":[],"post_actions":[],"call_ids":[],"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":[],"webhook_uri":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConf.EXPECT().Gets(gomock.Any(), tt.userID, conference.Type(tt.conferenceType), tt.pageSize, tt.pageToken).Return(tt.confs, nil)
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

func TestProcessV1ConferencesPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		conferenceHandler: mockConf,
	}

	tests := []struct {
		name             string
		request          *rabbitmqhandler.Request
		expectConference *conference.Conference
		expectRes        *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type": "conference", "user_id": 1, "name": "test", "detail": "test detail", "webhook_uri": "test.com", "pre_actions": [{"type":"answer"}], "post_actions": [{"type":"answer"}], "timeout": 86400}`),
			},
			&conference.Conference{
				ID:      uuid.FromStringOrNil("5e0d6cb0-4003-11ec-a7f9-f72079d71f10"),
				UserID:  1,
				Type:    conference.TypeConference,
				Name:    "test",
				Detail:  "test detail",
				Timeout: 86400,
				PreActions: []action.Action{
					{
						Type: "answer",
					},
				},
				PostActions: []action.Action{
					{
						Type: "answer",
					},
				},
				WebhookURI: "test.com",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5e0d6cb0-4003-11ec-a7f9-f72079d71f10","user_id":1,"confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"conference","status":"","name":"test","detail":"test detail","data":null,"timeout":86400,"pre_actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"post_actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"webhook_uri":"test.com","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConf.EXPECT().Create(gomock.Any(), tt.expectConference.Type, tt.expectConference.UserID, tt.expectConference.Name, tt.expectConference.Detail, tt.expectConference.Timeout, tt.expectConference.WebhookURI, tt.expectConference.PreActions, tt.expectConference.PostActions).Return(tt.expectConference, nil)
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
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		conferenceHandler: mockConf,
	}

	tests := []struct {
		name      string
		request   *rabbitmqhandler.Request
		id        uuid.UUID
		expectRes *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/8d920096-3bf2-11ec-9ff1-87ad93d2f885",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			uuid.FromStringOrNil("8d920096-3bf2-11ec-9ff1-87ad93d2f885"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConf.EXPECT().Terminate(gomock.Any(), tt.id).Return(nil)
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

func TestProcessV1ConferencesIDPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		conferenceHandler: mockConf,
	}

	tests := []struct {
		name       string
		request    *rabbitmqhandler.Request
		conference *conference.Conference
		expectRes  *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/a07e574a-4002-11ec-9c73-a31093777cf0",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name": "test update", "detail": "test detail update", "webhook_uri": "test.com", "pre_actions": [{"type":"answer"}], "post_actions": [{"type":"hangup"}], "timeout": 86400}`),
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("a07e574a-4002-11ec-9c73-a31093777cf0"),
				UserID:       1,
				ConfbridgeID: uuid.FromStringOrNil("590b7a70-4005-11ec-882c-cff85956bfd4"),
				FlowID:       uuid.FromStringOrNil("5937a834-4005-11ec-98ca-2770f4d8351a"),
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
				Name:         "test update",
				Detail:       "test detail update",
				Data:         map[string]interface{}{},
				Timeout:      86400,
				PreActions: []action.Action{
					{
						Type: "answer",
					},
				},
				PostActions: []action.Action{
					{
						Type: "hangup",
					},
				},
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{},
				WebhookURI:   "test.com",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a07e574a-4002-11ec-9c73-a31093777cf0","user_id":1,"confbridge_id":"590b7a70-4005-11ec-882c-cff85956bfd4","flow_id":"5937a834-4005-11ec-98ca-2770f4d8351a","type":"conference","status":"progressing","name":"test update","detail":"test detail update","data":{},"timeout":86400,"pre_actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"post_actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"hangup"}],"call_ids":[],"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":[],"webhook_uri":"test.com","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConf.EXPECT().Update(gomock.Any(), tt.conference.ID, tt.conference.Name, tt.conference.Detail, tt.conference.Timeout, tt.conference.WebhookURI, tt.conference.PreActions, tt.conference.PostActions).Return(tt.conference, nil)
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

func TestProcessV1ConferencesIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		conferenceHandler: mockConf,
	}

	tests := []struct {
		name             string
		request          *rabbitmqhandler.Request
		expectConference *conference.Conference
		expectRes        *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/11f067f6-3bf3-11ec-9bca-877deb76639d",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&conference.Conference{
				ID:      uuid.FromStringOrNil("11f067f6-3bf3-11ec-9bca-877deb76639d"),
				UserID:  1,
				Type:    conference.TypeConference,
				Name:    "test",
				Detail:  "test detail",
				Timeout: 86400,
				PreActions: []action.Action{
					{
						Type: "answer",
					},
				},
				PostActions: []action.Action{
					{
						Type: "answer",
					},
				},
				WebhookURI: "test.com",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"11f067f6-3bf3-11ec-9bca-877deb76639d","user_id":1,"confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"conference","status":"","name":"test","detail":"test detail","data":null,"timeout":86400,"pre_actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"post_actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"webhook_uri":"test.com","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConf.EXPECT().Get(gomock.Any(), tt.expectConference.ID).Return(tt.expectConference, nil)
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

func TestProcessV1ConferencesIDCallsIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		conferenceHandler: mockConf,
	}

	tests := []struct {
		name             string
		request          *rabbitmqhandler.Request
		callID           uuid.UUID
		expectConference *conference.Conference
		expectRes        *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/89e95b8c-3bf3-11ec-b6b1-0380a4a12739/calls/8a1fd900-3bf3-11ec-bd15-eb0c54c84612",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			uuid.FromStringOrNil("8a1fd900-3bf3-11ec-bd15-eb0c54c84612"),
			&conference.Conference{
				ID:      uuid.FromStringOrNil("89e95b8c-3bf3-11ec-b6b1-0380a4a12739"),
				UserID:  1,
				Type:    conference.TypeConference,
				Name:    "test",
				Detail:  "test detail",
				Timeout: 86400,
				PreActions: []action.Action{
					{
						Type: "answer",
					},
				},
				PostActions: []action.Action{
					{
						Type: "answer",
					},
				},
				WebhookURI: "test.com",
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("8a1fd900-3bf3-11ec-bd15-eb0c54c84612"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConf.EXPECT().Leave(gomock.Any(), tt.expectConference.ID, tt.callID).Return(nil)
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

func TestProcessV1ConferencesIDCallsIDPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		conferenceHandler: mockConf,
	}

	tests := []struct {
		name         string
		request      *rabbitmqhandler.Request
		conferenceID uuid.UUID
		callID       uuid.UUID
		expectRes    *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/f3dd474c-3bf3-11ec-b9fb-d7835bd4849d/calls/f3fd9268-3bf3-11ec-b5d5-938679a1a8f0",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			uuid.FromStringOrNil("f3dd474c-3bf3-11ec-b9fb-d7835bd4849d"),
			uuid.FromStringOrNil("f3fd9268-3bf3-11ec-b5d5-938679a1a8f0"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockConf.EXPECT().Join(gomock.Any(), tt.conferenceID, tt.callID).Return(nil)
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
