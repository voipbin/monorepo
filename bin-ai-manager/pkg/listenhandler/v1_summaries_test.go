package listenhandler

import (
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/summaryhandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_processV1SummariesGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseSummaries []*summary.Summary

		expectPageSize  uint64
		expectPageToken string
		expectFilters   map[string]string
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/summaries?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_customer_id=fa74c67c-0baa-11f0-9d9b-f79be2dd6ee6&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			responseSummaries: []*summary.Summary{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("faa75204-0baa-11f0-8aaa-831a1ee94d5d"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("facc2566-0baa-11f0-b71b-1bbb47f50b3b"),
					},
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-05-03 21:35:02.809",
			expectFilters: map[string]string{
				"deleted":     "false",
				"customer_id": "fa74c67c-0baa-11f0-9d9b-f79be2dd6ee6",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"faa75204-0baa-11f0-8aaa-831a1ee94d5d","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"},{"id":"facc2566-0baa-11f0-b71b-1bbb47f50b3b","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSummary := summaryhandler.NewMockSummaryHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				summaryHandler: mockSummary,
			}

			mockSummary.EXPECT().Gets(gomock.Any(), tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.responseSummaries, nil)
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

func Test_processV1SummariesPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseSummary *summary.Summary

		expectedCustomerID    uuid.UUID
		expectedActiveflowID  uuid.UUID
		expectedOnEndFlowID   uuid.UUID
		expectedReferenceType summary.ReferenceType
		expectedReferenceID   uuid.UUID
		expectedLanguage      string
		expectedRes           *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/summaries",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "62eb1516-0bac-11f0-a9f1-dbac9c204aa7", "activeflow_id": "62817d0e-0bac-11f0-8772-1f86bc7d4822", "on_end_flow_id": "813df6be-0bde-11f0-98b7-1b88d5c94e92", "reference_type": "recording", "reference_id": "62a5dea6-0bac-11f0-aed5-67b4506078c7", "language": "en-US"}`),
			},

			responseSummary: &summary.Summary{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("62c6a992-0bac-11f0-8ae6-db52e5aaf23d"),
				},
			},

			expectedCustomerID:    uuid.FromStringOrNil("62eb1516-0bac-11f0-a9f1-dbac9c204aa7"),
			expectedActiveflowID:  uuid.FromStringOrNil("62817d0e-0bac-11f0-8772-1f86bc7d4822"),
			expectedOnEndFlowID:   uuid.FromStringOrNil("813df6be-0bde-11f0-98b7-1b88d5c94e92"),
			expectedReferenceType: summary.ReferenceType(summary.ReferenceTypeRecording),
			expectedReferenceID:   uuid.FromStringOrNil("62a5dea6-0bac-11f0-aed5-67b4506078c7"),
			expectedLanguage:      "en-US",
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"62c6a992-0bac-11f0-8ae6-db52e5aaf23d","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSummary := summaryhandler.NewMockSummaryHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				summaryHandler: mockSummary,
			}

			mockSummary.EXPECT().Start(gomock.Any(), tt.expectedCustomerID, tt.expectedActiveflowID, tt.expectedOnEndFlowID, tt.expectedReferenceType, tt.expectedReferenceID, tt.expectedLanguage).Return(tt.responseSummary, nil)
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

func Test_processV1SummariesIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseSummary *summary.Summary

		expectedID  uuid.UUID
		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/summaries/4520a2ac-0bad-11f0-a428-679c2b6f1888",
				Method: sock.RequestMethodGet,
			},

			responseSummary: &summary.Summary{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("4520a2ac-0bad-11f0-a428-679c2b6f1888"),
				},
			},

			expectedID: uuid.FromStringOrNil("4520a2ac-0bad-11f0-a428-679c2b6f1888"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4520a2ac-0bad-11f0-a428-679c2b6f1888","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSummary := summaryhandler.NewMockSummaryHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				summaryHandler: mockSummary,
			}

			mockSummary.EXPECT().Get(gomock.Any(), tt.expectedID).Return(tt.responseSummary, nil)
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

func Test_processV1SummariesIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseSummary *summary.Summary

		expectedID  uuid.UUID
		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/summaries/93f1f214-0bad-11f0-980a-bba7be7d0493",
				Method: sock.RequestMethodDelete,
			},

			responseSummary: &summary.Summary{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("93f1f214-0bad-11f0-980a-bba7be7d0493"),
				},
			},

			expectedID: uuid.FromStringOrNil("93f1f214-0bad-11f0-980a-bba7be7d0493"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"93f1f214-0bad-11f0-980a-bba7be7d0493","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSummary := summaryhandler.NewMockSummaryHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				summaryHandler: mockSummary,
			}

			mockSummary.EXPECT().Delete(gomock.Any(), tt.expectedID).Return(tt.responseSummary, nil)
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
