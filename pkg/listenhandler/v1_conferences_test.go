package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencehandler"
)

func Test_processV1ConferencesGet(t *testing.T) {

	tests := []struct {
		name      string
		request   *rabbitmqhandler.Request
		pageSize  uint64
		pageToken string

		responseFilters     map[string]string
		responseConferences []*conference.Conference
		expectRes           *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_customer_id=24676972-7f49-11ec-bc89-b7d33e9d3ea8",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			10,
			"2020-05-03 21:35:02.809",

			map[string]string{
				"customer_id": "24676972-7f49-11ec-bc89-b7d33e9d3ea8",
			},
			[]*conference.Conference{
				{
					ID:         uuid.FromStringOrNil("0addf332-9312-11eb-95e8-9b90e44428a0"),
					CustomerID: uuid.FromStringOrNil("24676972-7f49-11ec-bc89-b7d33e9d3ea8"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0addf332-9312-11eb-95e8-9b90e44428a0","customer_id":"24676972-7f49-11ec-bc89-b7d33e9d3ea8","confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"have confbridge and flow id",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_customer_id=3be94c82-7f49-11ec-814e-ff2a9d84a806",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			10,
			"2020-05-03 21:35:02.809",

			map[string]string{
				"customer_id": "3be94c82-7f49-11ec-814e-ff2a9d84a806",
			},
			[]*conference.Conference{
				{
					ID:                uuid.FromStringOrNil("33b1138a-3bef-11ec-a187-f77a455f3ced"),
					CustomerID:        uuid.FromStringOrNil("3be94c82-7f49-11ec-814e-ff2a9d84a806"),
					ConfbridgeID:      uuid.FromStringOrNil("343ae074-3bef-11ec-b657-db12d3135e42"),
					FlowID:            uuid.FromStringOrNil("49da6378-3bef-11ec-88b6-f31f8c97b61b"),
					Data:              map[string]interface{}{},
					Timeout:           86400,
					PreActions:        []fmaction.Action{},
					PostActions:       []fmaction.Action{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TranscribeIDs:     []uuid.UUID{},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"33b1138a-3bef-11ec-a187-f77a455f3ced","customer_id":"3be94c82-7f49-11ec-814e-ff2a9d84a806","confbridge_id":"343ae074-3bef-11ec-b657-db12d3135e42","flow_id":"49da6378-3bef-11ec-88b6-f31f8c97b61b","type":"","status":"","name":"","detail":"","data":{},"timeout":86400,"pre_actions":[],"post_actions":[],"conferencecall_ids":[],"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":[],"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":[],"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"have confbridge and with conference type",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_customer_id=4d4d8ce0-7f49-11ec-a61f-1358990ed631&filter_type=conference",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			10,
			"2020-05-03 21:35:02.809",

			map[string]string{
				"customer_id": "4d4d8ce0-7f49-11ec-a61f-1358990ed631",
				"type":        string(conference.TypeConference),
			},
			[]*conference.Conference{
				{
					ID:                uuid.FromStringOrNil("c1e0a078-3de6-11ec-ae88-13052faf6ad7"),
					CustomerID:        uuid.FromStringOrNil("4d4d8ce0-7f49-11ec-a61f-1358990ed631"),
					Type:              conference.TypeConference,
					ConfbridgeID:      uuid.FromStringOrNil("c21b98ea-3de6-11ec-ab1e-4bcde9e784af"),
					FlowID:            uuid.FromStringOrNil("c234ce0a-3de6-11ec-8807-0b3f00d6e280"),
					Data:              map[string]interface{}{},
					Timeout:           86400,
					PreActions:        []fmaction.Action{},
					PostActions:       []fmaction.Action{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TranscribeIDs:     []uuid.UUID{},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c1e0a078-3de6-11ec-ae88-13052faf6ad7","customer_id":"4d4d8ce0-7f49-11ec-a61f-1358990ed631","confbridge_id":"c21b98ea-3de6-11ec-ab1e-4bcde9e784af","flow_id":"c234ce0a-3de6-11ec-8807-0b3f00d6e280","type":"conference","status":"","name":"","detail":"","data":{},"timeout":86400,"pre_actions":[],"post_actions":[],"conferencecall_ids":[],"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":[],"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":[],"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				utilHandler:       mockUtil,
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockConf.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.responseConferences, nil)
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

func Test_processV1ConferencesPost(t *testing.T) {

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
				Data:     []byte(`{"type": "conference", "customer_id": "2375a978-7f4b-11ec-81ed-73f63efd9dd8", "name": "test", "detail": "test detail", "pre_actions": [{"type":"answer"}], "post_actions": [{"type":"answer"}], "timeout": 86400}`),
			},
			&conference.Conference{
				ID:         uuid.FromStringOrNil("5e0d6cb0-4003-11ec-a7f9-f72079d71f10"),
				CustomerID: uuid.FromStringOrNil("2375a978-7f4b-11ec-81ed-73f63efd9dd8"),
				Type:       conference.TypeConference,
				Name:       "test",
				Detail:     "test detail",
				Timeout:    86400,
				PreActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
				PostActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5e0d6cb0-4003-11ec-a7f9-f72079d71f10","customer_id":"2375a978-7f4b-11ec-81ed-73f63efd9dd8","confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"conference","status":"","name":"test","detail":"test detail","data":null,"timeout":86400,"pre_actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"post_actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().Create(gomock.Any(), tt.expectConference.Type, tt.expectConference.CustomerID, tt.expectConference.Name, tt.expectConference.Detail, tt.expectConference.Timeout, tt.expectConference.PreActions, tt.expectConference.PostActions).Return(tt.expectConference, nil)
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

func Test_processV1ConferencesIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request
		id      uuid.UUID

		responseConference *conference.Conference
		expectRes          *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/8d920096-3bf2-11ec-9ff1-87ad93d2f885",
				Method: rabbitmqhandler.RequestMethodDelete,
			},

			uuid.FromStringOrNil("8d920096-3bf2-11ec-9ff1-87ad93d2f885"),

			&conference.Conference{
				ID: uuid.FromStringOrNil("8d920096-3bf2-11ec-9ff1-87ad93d2f885"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8d920096-3bf2-11ec-9ff1-87ad93d2f885","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseConference, nil)
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

func Test_processV1ConferencesIDPut(t *testing.T) {

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
				Data:     []byte(`{"name": "test update", "detail": "test detail update", "pre_actions": [{"type":"answer"}], "post_actions": [{"type":"hangup"}], "timeout": 86400}`),
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("a07e574a-4002-11ec-9c73-a31093777cf0"),
				CustomerID:   uuid.FromStringOrNil("4fa8d53a-8057-11ec-9e7c-2310213dc857"),
				ConfbridgeID: uuid.FromStringOrNil("590b7a70-4005-11ec-882c-cff85956bfd4"),
				FlowID:       uuid.FromStringOrNil("5937a834-4005-11ec-98ca-2770f4d8351a"),
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
				Name:         "test update",
				Detail:       "test detail update",
				Data:         map[string]interface{}{},
				Timeout:      86400,
				PreActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
				PostActions: []fmaction.Action{
					{
						Type: "hangup",
					},
				},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a07e574a-4002-11ec-9c73-a31093777cf0","customer_id":"4fa8d53a-8057-11ec-9e7c-2310213dc857","confbridge_id":"590b7a70-4005-11ec-882c-cff85956bfd4","flow_id":"5937a834-4005-11ec-98ca-2770f4d8351a","type":"conference","status":"progressing","name":"test update","detail":"test detail update","data":{},"timeout":86400,"pre_actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"post_actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"hangup"}],"conferencecall_ids":[],"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":[],"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().Update(gomock.Any(), tt.conference.ID, tt.conference.Name, tt.conference.Detail, tt.conference.Timeout, tt.conference.PreActions, tt.conference.PostActions).Return(tt.conference, nil)
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

func Test_processV1ConferencesIDGet(t *testing.T) {

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
				ID:         uuid.FromStringOrNil("11f067f6-3bf3-11ec-9bca-877deb76639d"),
				CustomerID: uuid.FromStringOrNil("4fa8d53a-8057-11ec-9e7c-2310213dc857"),
				Type:       conference.TypeConference,
				Name:       "test",
				Detail:     "test detail",
				Timeout:    86400,
				PreActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
				PostActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"11f067f6-3bf3-11ec-9bca-877deb76639d","customer_id":"4fa8d53a-8057-11ec-9e7c-2310213dc857","confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"conference","status":"","name":"test","detail":"test detail","data":null,"timeout":86400,"pre_actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"post_actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

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

func Test_processV1ConferencesIDRecordingIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseConference *conference.Conference

		expectID          uuid.UUID
		expectRecordingID uuid.UUID
		expectRes         *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/81d69286-9091-11ed-8036-5f6887716de3/recording_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"recording_id":"822a2c52-9091-11ed-99a1-5f802877affb"}`),
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("81d69286-9091-11ed-8036-5f6887716de3"),
			},

			uuid.FromStringOrNil("81d69286-9091-11ed-8036-5f6887716de3"),
			uuid.FromStringOrNil("822a2c52-9091-11ed-99a1-5f802877affb"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"81d69286-9091-11ed-8036-5f6887716de3","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().UpdateRecordingID(gomock.Any(), tt.expectID, tt.expectRecordingID).Return(tt.responseConference, nil)
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

func Test_processV1ConferencesIDRecordingStartPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseConference *conference.Conference

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/17ca9f6a-9102-11ed-9c97-1b1670cb9db9/recording_start",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("17ca9f6a-9102-11ed-9c97-1b1670cb9db9"),
			},

			uuid.FromStringOrNil("17ca9f6a-9102-11ed-9c97-1b1670cb9db9"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"17ca9f6a-9102-11ed-9c97-1b1670cb9db9","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().RecordingStart(gomock.Any(), tt.expectID).Return(tt.responseConference, nil)
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

func Test_processV1ConferencesIDRecordingStopPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseConference *conference.Conference

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/18033654-9102-11ed-994e-4b9c733834a5/recording_stop",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("18033654-9102-11ed-994e-4b9c733834a5"),
			},

			uuid.FromStringOrNil("18033654-9102-11ed-994e-4b9c733834a5"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"18033654-9102-11ed-994e-4b9c733834a5","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().RecordingStop(gomock.Any(), tt.expectID).Return(tt.responseConference, nil)
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

func Test_processV1ConferencesIDTranscribeStartPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseConference *conference.Conference

		expectID   uuid.UUID
		expectLang string
		expectRes  *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/95cf180c-98c6-11ed-8330-bb119cab4678/transcribe_start",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"language":"en-US"}`),
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("95cf180c-98c6-11ed-8330-bb119cab4678"),
			},

			uuid.FromStringOrNil("95cf180c-98c6-11ed-8330-bb119cab4678"),
			"en-US",
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"95cf180c-98c6-11ed-8330-bb119cab4678","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().TranscribeStart(gomock.Any(), tt.expectID, tt.expectLang).Return(tt.responseConference, nil)
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

func Test_processV1ConferencesIDTranscribeStopPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseConference *conference.Conference

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/95fdc09e-98c6-11ed-a6a1-ff3648dce452/transcribe_stop",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("95fdc09e-98c6-11ed-a6a1-ff3648dce452"),
			},

			uuid.FromStringOrNil("95fdc09e-98c6-11ed-a6a1-ff3648dce452"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"95fdc09e-98c6-11ed-a6a1-ff3648dce452","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().TranscribeStop(gomock.Any(), tt.expectID).Return(tt.responseConference, nil)
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

func Test_processV1ConferencesIDStopPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseConference *conference.Conference

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/24883eab-931d-4743-bf26-bd867b52127e/stop",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("24883eab-931d-4743-bf26-bd867b52127e"),
			},

			uuid.FromStringOrNil("24883eab-931d-4743-bf26-bd867b52127e"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"24883eab-931d-4743-bf26-bd867b52127e","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().Terminating(gomock.Any(), tt.expectID).Return(tt.responseConference, nil)
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
