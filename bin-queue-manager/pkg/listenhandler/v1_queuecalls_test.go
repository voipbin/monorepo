package listenhandler

import (
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/queuecallhandler"
)

func Test_processV1QueuecallsGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		pageSize        uint64
		pageToken       string
		responseFilters map[queuecall.Field]any

		queuecalls []*queuecall.Queuecall

		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/queuecalls?page_size=10&page_token=2020-05-03T21:35:02.809Z",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"f9f94078-7f54-11ec-8387-9fe49204286f","deleted":false,"queue_id":"d885e09e-b14e-11ee-95f5-37ef89c64b7a","status":"waiting"}`),
			},

			pageSize:  10,
			pageToken: "2020-05-03T21:35:02.809Z",

			responseFilters: map[queuecall.Field]any{
				queuecall.FieldCustomerID: uuid.FromStringOrNil("f9f94078-7f54-11ec-8387-9fe49204286f"),
				queuecall.FieldDeleted:    false,
				queuecall.FieldQueueID:    uuid.FromStringOrNil("d885e09e-b14e-11ee-95f5-37ef89c64b7a"),
				queuecall.FieldStatus:     queuecall.StatusWaiting,
			},

			queuecalls: []*queuecall.Queuecall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("4b46ad9c-6152-11ec-a4a6-7b3b226046a5"),
						CustomerID: uuid.FromStringOrNil("f9f94078-7f54-11ec-8387-9fe49204286f"),
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"4b46ad9c-6152-11ec-a4a6-7b3b226046a5","customer_id":"f9f94078-7f54-11ec-8387-9fe49204286f","queue_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{},"service_agent_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
		{
			name: "2 items",
			request: &sock.Request{
				URI:      "/v1/queuecalls?page_size=10&page_token=2020-05-03T21:35:02.809Z",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"13529ca4-7f55-11ec-b445-c3f90a718170","deleted":false}`),
			},

			pageSize:  10,
			pageToken: "2020-05-03T21:35:02.809Z",
			responseFilters: map[queuecall.Field]any{
				queuecall.FieldCustomerID: uuid.FromStringOrNil("13529ca4-7f55-11ec-b445-c3f90a718170"),
				queuecall.FieldDeleted:    false,
			},

			queuecalls: []*queuecall.Queuecall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("4ca0c722-6152-11ec-a0ad-1be04f100fff"),
						CustomerID: uuid.FromStringOrNil("13529ca4-7f55-11ec-b445-c3f90a718170"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("4cc9430a-6152-11ec-9295-d783a3ffb68e"),
						CustomerID: uuid.FromStringOrNil("13529ca4-7f55-11ec-b445-c3f90a718170"),
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"4ca0c722-6152-11ec-a0ad-1be04f100fff","customer_id":"13529ca4-7f55-11ec-b445-c3f90a718170","queue_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{},"service_agent_id":"00000000-0000-0000-0000-000000000000"},{"id":"4cc9430a-6152-11ec-9295-d783a3ffb68e","customer_id":"13529ca4-7f55-11ec-b445-c3f90a718170","queue_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{},"service_agent_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.queuecalls, nil)
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

func Test_processV1QueuecallsIDGet(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		id        uuid.UUID
		queuecall *queuecall.Queuecall

		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/queuecalls/0bc84788-6153-11ec-b08a-d74a5a04d995",
				Method: sock.RequestMethodGet,
			},

			id: uuid.FromStringOrNil("0bc84788-6153-11ec-b08a-d74a5a04d995"),
			queuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0bc84788-6153-11ec-b08a-d74a5a04d995"),
					CustomerID: uuid.FromStringOrNil("2ff5fe64-7f55-11ec-8c3c-83bef268c5ed"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0bc84788-6153-11ec-b08a-d74a5a04d995","customer_id":"2ff5fe64-7f55-11ec-8c3c-83bef268c5ed","queue_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{},"service_agent_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().Get(gomock.Any(), tt.id).Return(tt.queuecall, nil)
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

func Test_processV1QueuescallsIDDelete(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		expectRes         *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/queuecalls/4a76400a-60ab-11ec-aeb8-eb262d80acf1",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			queuecallID: uuid.FromStringOrNil("4a76400a-60ab-11ec-aeb8-eb262d80acf1"),

			responseQueuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4a76400a-60ab-11ec-aeb8-eb262d80acf1"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4a76400a-60ab-11ec-aeb8-eb262d80acf1","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{},"service_agent_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().Delete(gomock.Any(), tt.queuecallID).Return(tt.responseQueuecall, nil)

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

func Test_processV1QueuecallsIDKickPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		expectRes         *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/queuecalls/319fec77-0843-4207-8c6a-65bf067e4bac/kick",
				Method: sock.RequestMethodPost,
			},

			queuecallID: uuid.FromStringOrNil("319fec77-0843-4207-8c6a-65bf067e4bac"),

			responseQueuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("319fec77-0843-4207-8c6a-65bf067e4bac"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"319fec77-0843-4207-8c6a-65bf067e4bac","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{},"service_agent_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().Kick(gomock.Any(), tt.queuecallID).Return(tt.responseQueuecall, nil)

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

func Test_processV1QueuecallsIDHealthCheckPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		queuecallID uuid.UUID
		retryCount  int

		responseQueuecall *queuecall.Queuecall
		expectRes         *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/queuecalls/4a85453c-d534-11ee-8746-a3727fdf5678/health-check",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"retry_count": 1}`),
			},

			uuid.FromStringOrNil("4a85453c-d534-11ee-8746-a3727fdf5678"),
			1,

			&queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4a85453c-d534-11ee-8746-a3727fdf5678"),
				},
			},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().HealthCheck(gomock.Any(), tt.queuecallID, tt.retryCount)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

			time.Sleep(time.Millisecond * 100)

		})
	}
}

func Test_processV1QueuecallsReferenceIDIDKickPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		referenceID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		expectRes         *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/queuecalls/reference_id/c8794c51-4e4d-46ef-bfa1-5220f66aea87/kick",
				Method: sock.RequestMethodPost,
			},

			referenceID: uuid.FromStringOrNil("c8794c51-4e4d-46ef-bfa1-5220f66aea87"),

			responseQueuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b1d4d172-52e3-4927-bf10-77eafebd19d8"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b1d4d172-52e3-4927-bf10-77eafebd19d8","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{},"service_agent_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().KickByReferenceID(gomock.Any(), tt.referenceID).Return(tt.responseQueuecall, nil)

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

func Test_processV1QueuecallsReferenceIDIDGet(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		referenceID       uuid.UUID
		responseQueuecall *queuecall.Queuecall

		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/queuecalls/reference_id/b5f73c26-bcb7-11ed-af77-e397b8122b09",
				Method: sock.RequestMethodGet,
			},

			referenceID: uuid.FromStringOrNil("b5f73c26-bcb7-11ed-af77-e397b8122b09"),
			responseQueuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b673a022-bcb7-11ed-8212-6fef4fabe382"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b673a022-bcb7-11ed-8212-6fef4fabe382","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{},"service_agent_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().GetByReferenceID(gomock.Any(), tt.referenceID).Return(tt.responseQueuecall, nil)
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

func Test_processV1QueuecallsIDStatusWaitingPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		expectedQueuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		expectedRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/queuecalls/7c9e9cae-d1ca-11ec-a81e-0baaef8ce608/status_waiting",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			responseQueuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7c9e9cae-d1ca-11ec-a81e-0baaef8ce608"),
				},
			},

			expectedQueuecallID: uuid.FromStringOrNil("7c9e9cae-d1ca-11ec-a81e-0baaef8ce608"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7c9e9cae-d1ca-11ec-a81e-0baaef8ce608","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{},"service_agent_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().UpdateStatusWaiting(gomock.Any(), tt.expectedQueuecallID).Return(tt.responseQueuecall, nil)

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
