package listenhandler

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
)

func Test_processV1TranscribesPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		responseTranscribe *transcribe.Transcribe

		expectCustomerID    uuid.UUID
		expectActiveflowID  uuid.UUID
		expectOnEndFlowID   uuid.UUID
		expectReferenceType transcribe.ReferenceType
		expectReferenceID   uuid.UUID
		expectLanguage      string
		expectDirection     transcribe.Direction
		expectRes           *sock.Response
	}

	tests := []test{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/transcribes",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"10a7593a-9693-11ed-b4b7-7b48322d6a8d","activeflow_id":"586e7c60-0925-11f0-85f7-ffa1f967c6d0","on_end_flow_id":"58aedc4c-0925-11f0-a2ea-b36d37f1d67c","reference_type":"call","reference_id":"112d907c-9693-11ed-a72c-8fa9ccd046a7","language":"en-US","direction":"both"}`),
			},

			responseTranscribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1162e178-9693-11ed-9bcf-974fbfeb1ea3"),
				},
			},

			expectCustomerID:    uuid.FromStringOrNil("10a7593a-9693-11ed-b4b7-7b48322d6a8d"),
			expectActiveflowID:  uuid.FromStringOrNil("586e7c60-0925-11f0-85f7-ffa1f967c6d0"),
			expectOnEndFlowID:   uuid.FromStringOrNil("58aedc4c-0925-11f0-a2ea-b36d37f1d67c"),
			expectReferenceType: transcribe.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("112d907c-9693-11ed-a72c-8fa9ccd046a7"),
			expectLanguage:      "en-US",
			expectDirection:     transcribe.DirectionBoth,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1162e178-9693-11ed-9bcf-974fbfeb1ea3","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","host_id":"00000000-0000-0000-0000-000000000000","language":"","direction":"","streaming_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				reqHandler:        mockReq,
				transcribeHandler: mockTranscribe,
			}

			mockTranscribe.EXPECT().Start(gomock.Any(), tt.expectCustomerID, tt.expectActiveflowID, tt.expectOnEndFlowID, tt.expectReferenceType, tt.expectReferenceID, tt.expectLanguage, tt.expectDirection).Return(tt.responseTranscribe, nil)
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

func Test_processV1TranscribesGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageSize  uint64
		pageToken string

		responseFilters     map[string]string
		responseTranscribes []*transcribe.Transcribe
		expectRes           *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/transcribes?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_customer_id=079ffd84-7f68-11ed-ae05-430c9b75ab3b",
				Method: sock.RequestMethodGet,
			},

			pageSize:  10,
			pageToken: "2020-05-03 21:35:02.809",

			responseFilters: map[string]string{
				"customer_id": "079ffd84-7f68-11ed-ae05-430c9b75ab3b",
			},
			responseTranscribes: []*transcribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("0710ac06-7f68-11ed-b2cd-877b6dca8ac7"),
						CustomerID: uuid.FromStringOrNil("079ffd84-7f68-11ed-ae05-430c9b75ab3b"),
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0710ac06-7f68-11ed-b2cd-877b6dca8ac7","customer_id":"079ffd84-7f68-11ed-ae05-430c9b75ab3b","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","host_id":"00000000-0000-0000-0000-000000000000","language":"","direction":"","streaming_ids":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name: "2 items",
			request: &sock.Request{
				URI:    "/v1/transcribes?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_customer_id=871275ba-7f68-11ed-a6e2-dbc6d9a383d9",
				Method: sock.RequestMethodGet,
			},

			pageSize:  10,
			pageToken: "2020-05-03 21:35:02.809",

			responseFilters: map[string]string{
				"customer_id": "871275ba-7f68-11ed-a6e2-dbc6d9a383d9",
			},
			responseTranscribes: []*transcribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("873a8eec-7f68-11ed-9c2b-5f1311cc5a88"),
						CustomerID: uuid.FromStringOrNil("871275ba-7f68-11ed-a6e2-dbc6d9a383d9"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("876112b0-7f68-11ed-bf8c-074e301a66da"),
						CustomerID: uuid.FromStringOrNil("871275ba-7f68-11ed-a6e2-dbc6d9a383d9"),
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"873a8eec-7f68-11ed-9c2b-5f1311cc5a88","customer_id":"871275ba-7f68-11ed-a6e2-dbc6d9a383d9","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","host_id":"00000000-0000-0000-0000-000000000000","language":"","direction":"","streaming_ids":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"876112b0-7f68-11ed-bf8c-074e301a66da","customer_id":"871275ba-7f68-11ed-a6e2-dbc6d9a383d9","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","host_id":"00000000-0000-0000-0000-000000000000","language":"","direction":"","streaming_ids":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				utilHandler:       mockUtil,
				transcribeHandler: mockTranscribe,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockTranscribe.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.responseTranscribes, nil)
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

func Test_processV1TranscribesIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseTranscribe *transcribe.Transcribe
		expectRes          *sock.Response
	}{
		{
			name: "basic",
			request: &sock.Request{
				URI:    "/v1/transcribes/06db1ed2-7f69-11ed-a6fe-83fb6c80964d",
				Method: sock.RequestMethodGet,
			},
			responseTranscribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("06db1ed2-7f69-11ed-a6fe-83fb6c80964d"),
					CustomerID: uuid.FromStringOrNil("ab0fb69e-7f50-11ec-b0d3-2b4311e649e0"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"06db1ed2-7f69-11ed-a6fe-83fb6c80964d","customer_id":"ab0fb69e-7f50-11ec-b0d3-2b4311e649e0","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","host_id":"00000000-0000-0000-0000-000000000000","language":"","direction":"","streaming_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				transcribeHandler: mockTranscribe,
			}

			mockTranscribe.EXPECT().Get(gomock.Any(), tt.responseTranscribe.ID).Return(tt.responseTranscribe, nil)

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

func Test_processV1TranscribesIDDelete(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		customerID uuid.UUID

		request            *sock.Request
		responseTranscribe *transcribe.Transcribe

		expectRes *sock.Response
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("a4f388dc-86ab-11ec-8d14-9bd962288757"),
			customerID: uuid.FromStringOrNil("45afd578-7ffe-11ec-9430-3bdf65368563"),

			request: &sock.Request{
				URI:    "/v1/transcribes/a4f388dc-86ab-11ec-8d14-9bd962288757",
				Method: sock.RequestMethodDelete,
			},
			responseTranscribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4f388dc-86ab-11ec-8d14-9bd962288757"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a4f388dc-86ab-11ec-8d14-9bd962288757","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","host_id":"00000000-0000-0000-0000-000000000000","language":"","direction":"","streaming_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				reqHandler:        mockReq,
				transcribeHandler: mockTranscribe,
			}

			mockTranscribe.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseTranscribe, nil)

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

func Test_processV1TranscribesIDStopPost(t *testing.T) {

	type test struct {
		name string

		transcribeID uuid.UUID
		request      *sock.Request

		responseTranscribe *transcribe.Transcribe
		expectRes          *sock.Response
	}

	tests := []test{
		{
			name: "normal",

			transcribeID: uuid.FromStringOrNil("06b55408-821c-11ed-980a-cf31e1861a1f"),
			request: &sock.Request{
				URI:      "/v1/transcribes/06b55408-821c-11ed-980a-cf31e1861a1f/stop",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(``),
			},

			responseTranscribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("06b55408-821c-11ed-980a-cf31e1861a1f"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"06b55408-821c-11ed-980a-cf31e1861a1f","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","host_id":"00000000-0000-0000-0000-000000000000","language":"","direction":"","streaming_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				reqHandler:        mockReq,
				transcribeHandler: mockTranscribe,
			}

			mockTranscribe.EXPECT().Stop(gomock.Any(), tt.transcribeID).Return(tt.responseTranscribe, nil)
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

func Test_processV1TranscribesIDHealthCheckPost(t *testing.T) {

	type test struct {
		name string

		id         uuid.UUID
		retryCount int

		request *sock.Request
	}

	tests := []test{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("e04a0326-5c94-446e-bafb-1d53aa310420"),
			retryCount: 0,

			request: &sock.Request{
				URI:    "/v1/transcribes/e04a0326-5c94-446e-bafb-1d53aa310420/health-check",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"retry_count": 0}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				transcribeHandler: mockTranscribe,
			}

			mockTranscribe.EXPECT().HealthCheck(gomock.Any(), tt.id, tt.retryCount)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			} else if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}
