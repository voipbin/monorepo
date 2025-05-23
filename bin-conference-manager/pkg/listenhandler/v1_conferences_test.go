package listenhandler

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmrecording "monorepo/bin-call-manager/models/recording"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
)

func Test_processV1ConferencesGet(t *testing.T) {

	tests := []struct {
		name      string
		request   *sock.Request
		pageSize  uint64
		pageToken string

		responseFilters     map[string]string
		responseConferences []*conference.Conference
		expectRes           *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/conferences?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_customer_id=24676972-7f49-11ec-bc89-b7d33e9d3ea8",
				Method: sock.RequestMethodGet,
			},
			pageSize:  10,
			pageToken: "2020-05-03 21:35:02.809",

			responseFilters: map[string]string{
				"customer_id": "24676972-7f49-11ec-bc89-b7d33e9d3ea8",
			},
			responseConferences: []*conference.Conference{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("0addf332-9312-11eb-95e8-9b90e44428a0"),
						CustomerID: uuid.FromStringOrNil("24676972-7f49-11ec-bc89-b7d33e9d3ea8"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("cf213904-1e12-11f0-a307-473fcd2945e1"),
						CustomerID: uuid.FromStringOrNil("cf584494-1e12-11f0-87ac-ef9935cfae0b"),
					},
					ConfbridgeID: uuid.FromStringOrNil("cf7cfc9e-1e12-11f0-a5b1-8b47d4f29014"),
					Type:         conference.TypeConference,
					Status:       conference.StatusProgressing,
					Name:         "test",
					Detail:       "test detail",
					Data:         map[string]any{},
					Timeout:      86400,
					PreFlowID:    uuid.FromStringOrNil("cfa62998-1e12-11f0-8a26-eb50997bd60f"),
					PostFlowID:   uuid.FromStringOrNil("cfcaa0de-1e12-11f0-8c6b-63d5cf717773"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0addf332-9312-11eb-95e8-9b90e44428a0","customer_id":"24676972-7f49-11ec-bc89-b7d33e9d3ea8","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"},{"id":"cf213904-1e12-11f0-a307-473fcd2945e1","customer_id":"cf584494-1e12-11f0-87ac-ef9935cfae0b","confbridge_id":"cf7cfc9e-1e12-11f0-a5b1-8b47d4f29014","type":"conference","status":"progressing","name":"test","detail":"test detail","timeout":86400,"pre_flow_id":"cfa62998-1e12-11f0-8a26-eb50997bd60f","post_flow_id":"cfcaa0de-1e12-11f0-8c6b-63d5cf717773","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				utilHandler:       mockUtil,
				sockHandler:       mockSock,
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
		name    string
		request *sock.Request

		responseConference *conference.Conference

		expectID         uuid.UUID
		expectCustomerID uuid.UUID
		expectType       conference.Type
		expectName       string
		expectDetail     string
		expectData       map[string]any
		expectTimeout    int
		expectPreFlowID  uuid.UUID
		expectPostFlowID uuid.UUID
		expectRes        *sock.Response
	}{
		{
			name: "type conference",
			request: &sock.Request{
				URI:      "/v1/conferences",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"6d4ac758-1e13-11f0-8b51-af3b91042ec1", "customer_id": "2375a978-7f4b-11ec-81ed-73f63efd9dd8", "type": "conference", "name": "test", "detail": "test detail", "data": {"key1": "val1"}, "timeout": 86400, "pre_flow_id": "6da1673e-1e13-11f0-a97e-cf4118f48153", "post_flow_id": "6d78895e-1e13-11f0-92a8-f3131cbcdd09"}`),
			},

			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6d4ac758-1e13-11f0-8b51-af3b91042ec1"),
					CustomerID: uuid.FromStringOrNil("2375a978-7f4b-11ec-81ed-73f63efd9dd8"),
				},
			},

			expectID:         uuid.FromStringOrNil("6d4ac758-1e13-11f0-8b51-af3b91042ec1"),
			expectCustomerID: uuid.FromStringOrNil("2375a978-7f4b-11ec-81ed-73f63efd9dd8"),
			expectType:       conference.TypeConference,
			expectName:       "test",
			expectDetail:     "test detail",
			expectData:       map[string]any{"key1": "val1"},
			expectTimeout:    86400,
			expectPreFlowID:  uuid.FromStringOrNil("6da1673e-1e13-11f0-a97e-cf4118f48153"),
			expectPostFlowID: uuid.FromStringOrNil("6d78895e-1e13-11f0-92a8-f3131cbcdd09"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6d4ac758-1e13-11f0-8b51-af3b91042ec1","customer_id":"2375a978-7f4b-11ec-81ed-73f63efd9dd8","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().Create(
				gomock.Any(),
				tt.expectID,
				tt.expectCustomerID,
				tt.expectType,
				tt.expectName,
				tt.expectDetail,
				tt.expectData,
				tt.expectTimeout,
				tt.expectPreFlowID,
				tt.expectPostFlowID,
			).Return(tt.responseConference, nil)
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
		request *sock.Request
		id      uuid.UUID

		responseConference *conference.Conference
		expectRes          *sock.Response
	}{
		{
			"type conference",
			&sock.Request{
				URI:    "/v1/conferences/8d920096-3bf2-11ec-9ff1-87ad93d2f885",
				Method: sock.RequestMethodDelete,
			},

			uuid.FromStringOrNil("8d920096-3bf2-11ec-9ff1-87ad93d2f885"),

			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d920096-3bf2-11ec-9ff1-87ad93d2f885"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8d920096-3bf2-11ec-9ff1-87ad93d2f885","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
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
		name               string
		request            *sock.Request
		responseConference *conference.Conference

		expectedID       uuid.UUID
		expectedName     string
		expectedDetail   string
		expectedData     map[string]any
		expectedTimeout  int
		expectPreFlowID  uuid.UUID
		expectPostFlowID uuid.UUID
		expectedRes      *sock.Response
	}{
		{
			name: "type conference",
			request: &sock.Request{
				URI:      "/v1/conferences/a07e574a-4002-11ec-9c73-a31093777cf0",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name": "test update", "detail": "test detail update", "data": {"key1": "val1"}, "timeout": 86400, "pre_flow_id": "cfc8ec76-1e17-11f0-a8c7-4b7957ebef12", "post_flow_id": "cffbc290-1e17-11f0-ba00-bb8f33143099"}`),
			},
			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a07e574a-4002-11ec-9c73-a31093777cf0"),
				},
			},

			expectedID:       uuid.FromStringOrNil("a07e574a-4002-11ec-9c73-a31093777cf0"),
			expectedName:     "test update",
			expectedDetail:   "test detail update",
			expectedData:     map[string]any{"key1": "val1"},
			expectedTimeout:  86400,
			expectPreFlowID:  uuid.FromStringOrNil("cfc8ec76-1e17-11f0-a8c7-4b7957ebef12"),
			expectPostFlowID: uuid.FromStringOrNil("cffbc290-1e17-11f0-ba00-bb8f33143099"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a07e574a-4002-11ec-9c73-a31093777cf0","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().Update(
				gomock.Any(),
				tt.expectedID,
				tt.expectedName,
				tt.expectedDetail,
				tt.expectedData,
				tt.expectedTimeout,
				tt.expectPreFlowID,
				tt.expectPostFlowID,
			).Return(tt.responseConference, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectedRes, res)
			}

		})
	}
}

func Test_processV1ConferencesIDGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseConference *conference.Conference

		expectedID  uuid.UUID
		expectedRes *sock.Response
	}{
		{
			name: "type conference",
			request: &sock.Request{
				URI:    "/v1/conferences/11f067f6-3bf3-11ec-9bca-877deb76639d",
				Method: sock.RequestMethodGet,
			},
			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11f067f6-3bf3-11ec-9bca-877deb76639d"),
				},
			},

			expectedID: uuid.FromStringOrNil("11f067f6-3bf3-11ec-9bca-877deb76639d"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"11f067f6-3bf3-11ec-9bca-877deb76639d","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().Get(gomock.Any(), tt.expectedID).Return(tt.responseConference, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectedRes, res)
			}

		})
	}
}

func Test_processV1ConferencesIDRecordingIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseConference *conference.Conference

		expectID          uuid.UUID
		expectRecordingID uuid.UUID
		expectRes         *sock.Response
	}{
		{
			"type conference",
			&sock.Request{
				URI:      "/v1/conferences/81d69286-9091-11ed-8036-5f6887716de3/recording_id",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"recording_id":"822a2c52-9091-11ed-99a1-5f802877affb"}`),
			},
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("81d69286-9091-11ed-8036-5f6887716de3"),
				},
			},

			uuid.FromStringOrNil("81d69286-9091-11ed-8036-5f6887716de3"),
			uuid.FromStringOrNil("822a2c52-9091-11ed-99a1-5f802877affb"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"81d69286-9091-11ed-8036-5f6887716de3","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
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
		request *sock.Request

		responseConference *conference.Conference

		expectedID           uuid.UUID
		expectedActiveflowID uuid.UUID
		expectedFormat       cmrecording.Format
		expectedDuration     int
		expectedOnEndFlowID  uuid.UUID
		expectedRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/conferences/17ca9f6a-9102-11ed-9c97-1b1670cb9db9/recording_start",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"activeflow_id":"174670b0-075b-11f0-8de9-ebaa8ca77a57","format":"wav","duration":600,"on_end_flow_id":"b2f2d696-055f-11f0-8b66-b75440b1ede2"}`),
			},

			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("17ca9f6a-9102-11ed-9c97-1b1670cb9db9"),
				},
			},

			expectedID:           uuid.FromStringOrNil("17ca9f6a-9102-11ed-9c97-1b1670cb9db9"),
			expectedActiveflowID: uuid.FromStringOrNil("174670b0-075b-11f0-8de9-ebaa8ca77a57"),
			expectedFormat:       cmrecording.FormatWAV,
			expectedDuration:     600,
			expectedOnEndFlowID:  uuid.FromStringOrNil("b2f2d696-055f-11f0-8b66-b75440b1ede2"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"17ca9f6a-9102-11ed-9c97-1b1670cb9db9","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().RecordingStart(gomock.Any(), tt.expectedID, tt.expectedActiveflowID, tt.expectedFormat, tt.expectedDuration, tt.expectedOnEndFlowID).Return(tt.responseConference, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_processV1ConferencesIDRecordingStopPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseConference *conference.Conference

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"type conference",
			&sock.Request{
				URI:      "/v1/conferences/18033654-9102-11ed-994e-4b9c733834a5/recording_stop",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("18033654-9102-11ed-994e-4b9c733834a5"),
				},
			},

			uuid.FromStringOrNil("18033654-9102-11ed-994e-4b9c733834a5"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"18033654-9102-11ed-994e-4b9c733834a5","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
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
		request *sock.Request

		responseConference *conference.Conference

		expectID   uuid.UUID
		expectLang string
		expectRes  *sock.Response
	}{
		{
			"type conference",
			&sock.Request{
				URI:      "/v1/conferences/95cf180c-98c6-11ed-8330-bb119cab4678/transcribe_start",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"language":"en-US"}`),
			},
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("95cf180c-98c6-11ed-8330-bb119cab4678"),
				},
			},

			uuid.FromStringOrNil("95cf180c-98c6-11ed-8330-bb119cab4678"),
			"en-US",
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"95cf180c-98c6-11ed-8330-bb119cab4678","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
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
		request *sock.Request

		responseConference *conference.Conference

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"type conference",
			&sock.Request{
				URI:      "/v1/conferences/95fdc09e-98c6-11ed-a6a1-ff3648dce452/transcribe_stop",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("95fdc09e-98c6-11ed-a6a1-ff3648dce452"),
				},
			},

			uuid.FromStringOrNil("95fdc09e-98c6-11ed-a6a1-ff3648dce452"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"95fdc09e-98c6-11ed-a6a1-ff3648dce452","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
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
		request *sock.Request

		responseConference *conference.Conference

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			name: "type conference",
			request: &sock.Request{
				URI:      "/v1/conferences/24883eab-931d-4743-bf26-bd867b52127e/stop",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},
			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("24883eab-931d-4743-bf26-bd867b52127e"),
				},
			},

			expectID: uuid.FromStringOrNil("24883eab-931d-4743-bf26-bd867b52127e"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"24883eab-931d-4743-bf26-bd867b52127e","customer_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pre_flow_id":"00000000-0000-0000-0000-000000000000","post_flow_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
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
