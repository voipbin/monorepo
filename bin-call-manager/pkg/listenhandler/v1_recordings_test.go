package listenhandler

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

func Test_processV1RecordingsGet(t *testing.T) {
	type test struct {
		name      string
		request   *sock.Request
		pageSize  uint64
		pageToken string

		responseFilters    map[string]string
		responseRecordings []*recording.Recording
		expectRes          *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/recordings?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_customer_id=c15af818-7f51-11ec-8eeb-f733ba8df393",
				Method: sock.RequestMethodGet,
			},
			pageSize:  10,
			pageToken: "2020-05-03 21:35:02.809",

			responseFilters: map[string]string{
				"customer_id": "c15af818-7f51-11ec-8eeb-f733ba8df393",
			},
			responseRecordings: []*recording.Recording{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("cfa4d576-6128-11eb-b69b-9f7a738a1ad7"),
						CustomerID: uuid.FromStringOrNil("c15af818-7f51-11ec-8eeb-f733ba8df393"),
					},
					ActiveflowID:  uuid.FromStringOrNil("18d4f7e4-0729-11f0-b836-df8fafa3c2a1"),
					ReferenceType: recording.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
					Status:        recording.StatusEnded,
					Filenames: []string{
						"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"cfa4d576-6128-11eb-b69b-9f7a738a1ad7","customer_id":"c15af818-7f51-11ec-8eeb-f733ba8df393","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"18d4f7e4-0729-11f0-b836-df8fafa3c2a1","reference_type":"call","reference_id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","status":"ended","on_end_flow_id":"00000000-0000-0000-0000-000000000000","filenames":["call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav"]}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &listenHandler{
				utilHandler:      mockUtil,
				sockHandler:      mockSock,
				callHandler:      mockCall,
				recordingHandler: mockRecording,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockRecording.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.responseRecordings, nil)

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

func Test_processV1RecordingsPost(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		responseRecording *recording.Recording

		expectActiveflowID  uuid.UUID
		expectReferenceType recording.ReferenceType
		expectReferenceID   uuid.UUID
		expectFormat        recording.Format
		expectEndOfSilence  int
		expectEndOfKey      string
		expectDuration      int
		expectOnEndFlowID   uuid.UUID

		expectRes *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/recordings",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"activeflow_id": "688101bc-0728-11f0-b710-83da269e6001", "reference_type": "call", "reference_id": "30e259e0-90b5-11ed-9ca7-836b535a4622", "format": "wav", "end_of_silence": 0, "end_of_key": "", "duration": 0, "on_end_flow_id": "c69d28ae-0541-11f0-9eb9-f3479d7a3968"}`),
			},

			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ccf74444-90b5-11ed-958b-4fac7f75981c"),
				},
			},

			expectActiveflowID:  uuid.FromStringOrNil("688101bc-0728-11f0-b710-83da269e6001"),
			expectReferenceType: recording.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("30e259e0-90b5-11ed-9ca7-836b535a4622"),
			expectFormat:        recording.FormatWAV,
			expectEndOfSilence:  0,
			expectEndOfKey:      "",
			expectDuration:      0,
			expectOnEndFlowID:   uuid.FromStringOrNil("c69d28ae-0541-11f0-9eb9-f3479d7a3968"),

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"ccf74444-90b5-11ed-958b-4fac7f75981c","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				recordingHandler: mockRecording,
			}

			mockRecording.EXPECT().Start(gomock.Any(), tt.expectActiveflowID, tt.expectReferenceType, tt.expectReferenceID, tt.expectFormat, tt.expectEndOfSilence, tt.expectEndOfKey, tt.expectDuration, tt.expectOnEndFlowID).Return(tt.responseRecording, nil)
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

func Test_processV1RecordingsIDGet(t *testing.T) {

	type test struct {
		name              string
		request           *sock.Request
		responseRecording *recording.Recording
		expectRes         *sock.Response
	}

	tests := []test{
		{
			name: "basic",
			request: &sock.Request{
				URI:    "/v1/recordings/00c711be-6129-11eb-9404-b73dcf512957",
				Method: sock.RequestMethodGet,
			},
			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00c711be-6129-11eb-9404-b73dcf512957"),
					CustomerID: uuid.FromStringOrNil("d063099a-7f51-11ec-adbd-cf15a2e7ae7d"),
				},
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Status:        recording.StatusEnded,
				Filenames: []string{
					"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"00c711be-6129-11eb-9404-b73dcf512957","customer_id":"d063099a-7f51-11ec-adbd-cf15a2e7ae7d","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"call","reference_id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","status":"ended","on_end_flow_id":"00000000-0000-0000-0000-000000000000","filenames":["call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav"]}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				recordingHandler: mockRecording,
			}

			mockRecording.EXPECT().Get(gomock.Any(), tt.responseRecording.ID).Return(tt.responseRecording, nil)

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

func Test_processV1RecordingsIDDelete(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseRecording *recording.Recording
		expectRes         *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/recordings/3019fe2a-8eba-11ed-809e-bbab8230e905",
				Method: sock.RequestMethodDelete,
			},

			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3019fe2a-8eba-11ed-809e-bbab8230e905"),
					CustomerID: uuid.FromStringOrNil("d063099a-7f51-11ec-adbd-cf15a2e7ae7d"),
				},
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Status:        recording.StatusEnded,
				Filenames: []string{
					"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3019fe2a-8eba-11ed-809e-bbab8230e905","customer_id":"d063099a-7f51-11ec-adbd-cf15a2e7ae7d","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"call","reference_id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","status":"ended","on_end_flow_id":"00000000-0000-0000-0000-000000000000","filenames":["call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav"]}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				recordingHandler: mockRecording,
			}

			mockRecording.EXPECT().Delete(gomock.Any(), tt.responseRecording.ID).Return(tt.responseRecording, nil)

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

func Test_processV1RecordingsIDStopPost(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseRecording *recording.Recording
		expectRes         *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/recordings/2c7e5af4-90d6-11ed-8ba0-c335ddc4049b/stop",
				Method: sock.RequestMethodPost,
			},

			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2c7e5af4-90d6-11ed-8ba0-c335ddc4049b"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2c7e5af4-90d6-11ed-8ba0-c335ddc4049b","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				recordingHandler: mockRecording,
			}

			mockRecording.EXPECT().Stop(gomock.Any(), tt.responseRecording.ID).Return(tt.responseRecording, nil)

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
