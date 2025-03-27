package requesthandler

import (
	"context"
	"reflect"
	"testing"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_TranscribeV1TranscribeGet(t *testing.T) {

	type test struct {
		name string

		transcribeID uuid.UUID

		response *sock.Response

		expectedTarget  string
		expectedRequest *sock.Request
		expectedRes     *tmtranscribe.Transcribe
	}

	tests := []test{
		{
			name: "normal",

			transcribeID: uuid.FromStringOrNil("32b71878-8093-11ed-8578-775276ea57cf"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"32b71878-8093-11ed-8578-775276ea57cf"}`),
			},

			expectedTarget: "bin-manager.transcribe-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/transcribes/32b71878-8093-11ed-8578-775276ea57cf",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			expectedRes: &tmtranscribe.Transcribe{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("32b71878-8093-11ed-8578-775276ea57cf"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedTarget, tt.expectedRequest).Return(tt.response, nil)

			res, err := h.TranscribeV1TranscribeGet(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_TranscribeV1TranscribeGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *sock.Response

		expectedURL     string
		expectedTarget  string
		expectedRequest *sock.Request
		expectedRes     []tmtranscribe.Transcribe
	}{
		{
			name: "1 item",

			pageToken: "2020-09-20T03:23:20.995000",
			pageSize:  10,
			filters: map[string]string{
				"customer_id": "adddce70-8093-11ed-9a79-530f80f428d8",
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ae0a7cfe-8093-11ed-963d-abb334c8e6d8"}]`),
			},

			expectedURL:    "/v1/transcribes?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			expectedTarget: "bin-manager.transcribe-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/transcribes?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_customer_id=adddce70-8093-11ed-9a79-530f80f428d8",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			expectedRes: []tmtranscribe.Transcribe{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("ae0a7cfe-8093-11ed-963d-abb334c8e6d8"),
					},
				},
			},
		},
		{
			name: "2 items",

			pageToken: "2020-09-20T03:23:20.995000",
			pageSize:  10,
			filters: map[string]string{
				"customer_id": "bb3c9146-8093-11ed-a0df-6fbf1a76cbd3",
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bb6c13bc-8093-11ed-b647-5f3b613e1180"},{"id":"bb8fc46a-8093-11ed-9ea7-9304ab751b40"}]`),
			},

			expectedURL:    "/v1/transcribes?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			expectedTarget: "bin-manager.transcribe-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/transcribes?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_customer_id=bb3c9146-8093-11ed-a0df-6fbf1a76cbd3",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			expectedRes: []tmtranscribe.Transcribe{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("bb6c13bc-8093-11ed-b647-5f3b613e1180"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("bb8fc46a-8093-11ed-9ea7-9304ab751b40"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectedURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectedURL, tt.filters))
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedTarget, tt.expectedRequest).Return(tt.response, nil)

			res, err := h.TranscribeV1TranscribeGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_TranscribeV1TranscribeStart(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		activeflowID  uuid.UUID
		onEndFlowID   uuid.UUID
		referenceType tmtranscribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     tmtranscribe.Direction
		timeout       int

		response *sock.Response

		expectedTarget  string
		expectedRequest *sock.Request
		expectedRes     *tmtranscribe.Transcribe
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("2ab9c63a-8227-11ed-928b-1b90501adbe2"),
			activeflowID:  uuid.FromStringOrNil("d7794d42-0938-11f0-a95d-e3a4c60962f2"),
			onEndFlowID:   uuid.FromStringOrNil("d7b01a98-0938-11f0-85a1-cb7a3f01f80f"),
			referenceType: tmtranscribe.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("2ae8944c-8227-11ed-acb4-c3e23ea3a2a4"),
			language:      "en-US",
			direction:     tmtranscribe.DirectionBoth,
			timeout:       30000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2b13a4ca-8227-11ed-8bad-b7bb9aa7f185"}`),
			},

			expectedTarget: "bin-manager.transcribe-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/transcribes",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"2ab9c63a-8227-11ed-928b-1b90501adbe2","activeflow_id":"d7794d42-0938-11f0-a95d-e3a4c60962f2","on_end_flow_id":"d7b01a98-0938-11f0-85a1-cb7a3f01f80f","reference_type":"call","reference_id":"2ae8944c-8227-11ed-acb4-c3e23ea3a2a4","language":"en-US","direction":"both"}`),
			},
			expectedRes: &tmtranscribe.Transcribe{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("2b13a4ca-8227-11ed-8bad-b7bb9aa7f185"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedTarget, tt.expectedRequest).Return(tt.response, nil)

			res, err := h.TranscribeV1TranscribeStart(ctx, tt.customerID, tt.activeflowID, tt.onEndFlowID, tt.referenceType, tt.referenceID, tt.language, tt.direction, tt.timeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_TranscribeV1TranscribeStop(t *testing.T) {

	tests := []struct {
		name string

		transcribeID uuid.UUID

		response *sock.Response

		expectedTarget  string
		expectedRequest *sock.Request
		expectedRes     *tmtranscribe.Transcribe
	}{
		{
			name: "normal",

			transcribeID: uuid.FromStringOrNil("2622b04a-8228-11ed-98f0-6bfc284cdb95"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2622b04a-8228-11ed-98f0-6bfc284cdb95"}`),
			},

			expectedTarget: "bin-manager.transcribe-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/transcribes/2622b04a-8228-11ed-98f0-6bfc284cdb95/stop",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
			},
			expectedRes: &tmtranscribe.Transcribe{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("2622b04a-8228-11ed-98f0-6bfc284cdb95"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedTarget, tt.expectedRequest).Return(tt.response, nil)

			res, err := h.TranscribeV1TranscribeStop(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_TranscribeV1TranscribeHealthCheck(t *testing.T) {

	tests := []struct {
		name string

		transcribeID uuid.UUID
		delay        int
		retryCount   int

		response *sock.Response

		expectedTarget  string
		expectedRequest *sock.Request
	}{
		{
			name: "normal",

			transcribeID: uuid.FromStringOrNil("273d1fa4-e9ac-46cc-920e-34e163eb0e73"),
			delay:        0,
			retryCount:   3,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			expectedTarget: "bin-manager.transcribe-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/transcribes/273d1fa4-e9ac-46cc-920e-34e163eb0e73/health-check",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"retry_count":3}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedTarget, tt.expectedRequest).Return(tt.response, nil)

			err := h.TranscribeV1TranscribeHealthCheck(ctx, tt.transcribeID, tt.delay, tt.retryCount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
