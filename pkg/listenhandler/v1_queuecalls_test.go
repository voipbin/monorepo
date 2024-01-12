package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallhandler"
)

func Test_processV1QueuecallsGet(t *testing.T) {

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		customerID uuid.UUID
		pageSize   uint64
		pageToken  string
		filters    map[string]string

		queuecalls []*queuecall.Queuecall

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/queuecalls?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=f9f94078-7f54-11ec-8387-9fe49204286f&filter_deleted=false&filter_queue_id=d885e09e-b14e-11ee-95f5-37ef89c64b7a&filter_status=waiting",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			uuid.FromStringOrNil("f9f94078-7f54-11ec-8387-9fe49204286f"),
			10,
			"2020-05-03 21:35:02.809",
			map[string]string{
				"deleted":  "false",
				"queue_id": "d885e09e-b14e-11ee-95f5-37ef89c64b7a",
				"status":   "waiting",
			},

			[]*queuecall.Queuecall{
				{
					ID:         uuid.FromStringOrNil("4b46ad9c-6152-11ec-a4a6-7b3b226046a5"),
					CustomerID: uuid.FromStringOrNil("f9f94078-7f54-11ec-8387-9fe49204286f"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"4b46ad9c-6152-11ec-a4a6-7b3b226046a5","customer_id":"f9f94078-7f54-11ec-8387-9fe49204286f","queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","exit_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_end":"","tm_delete":""}]`),
			},
		},
		{
			"2 items",
			&rabbitmqhandler.Request{
				URI:    "/v1/queuecalls?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=13529ca4-7f55-11ec-b445-c3f90a718170&filter_deleted=false",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			uuid.FromStringOrNil("13529ca4-7f55-11ec-b445-c3f90a718170"),
			10,
			"2020-05-03 21:35:02.809",
			map[string]string{
				"deleted": "false",
			},

			[]*queuecall.Queuecall{
				{
					ID:         uuid.FromStringOrNil("4ca0c722-6152-11ec-a0ad-1be04f100fff"),
					CustomerID: uuid.FromStringOrNil("13529ca4-7f55-11ec-b445-c3f90a718170"),
				},
				{
					ID:         uuid.FromStringOrNil("4cc9430a-6152-11ec-9295-d783a3ffb68e"),
					CustomerID: uuid.FromStringOrNil("13529ca4-7f55-11ec-b445-c3f90a718170"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"4ca0c722-6152-11ec-a0ad-1be04f100fff","customer_id":"13529ca4-7f55-11ec-b445-c3f90a718170","queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","exit_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_end":"","tm_delete":""},{"id":"4cc9430a-6152-11ec-9295-d783a3ffb68e","customer_id":"13529ca4-7f55-11ec-b445-c3f90a718170","queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","exit_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_end":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				rabbitSock: mockSock,

				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().GetsByCustomerID(gomock.Any(), tt.customerID, tt.pageSize, tt.pageToken, tt.filters).Return(tt.queuecalls, nil)
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

		request *rabbitmqhandler.Request

		id        uuid.UUID
		queuecall *queuecall.Queuecall

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/queuecalls/0bc84788-6153-11ec-b08a-d74a5a04d995",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			uuid.FromStringOrNil("0bc84788-6153-11ec-b08a-d74a5a04d995"),
			&queuecall.Queuecall{
				ID:         uuid.FromStringOrNil("0bc84788-6153-11ec-b08a-d74a5a04d995"),
				CustomerID: uuid.FromStringOrNil("2ff5fe64-7f55-11ec-8c3c-83bef268c5ed"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0bc84788-6153-11ec-b08a-d74a5a04d995","customer_id":"2ff5fe64-7f55-11ec-8c3c-83bef268c5ed","queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","exit_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_end":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				rabbitSock: mockSock,

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

		request *rabbitmqhandler.Request

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		expectRes         *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecalls/4a76400a-60ab-11ec-aeb8-eb262d80acf1",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("4a76400a-60ab-11ec-aeb8-eb262d80acf1"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("4a76400a-60ab-11ec-aeb8-eb262d80acf1"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4a76400a-60ab-11ec-aeb8-eb262d80acf1","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","exit_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_end":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
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

		request *rabbitmqhandler.Request

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		expectRes         *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/queuecalls/319fec77-0843-4207-8c6a-65bf067e4bac/kick",
				Method: rabbitmqhandler.RequestMethodPost,
			},

			uuid.FromStringOrNil("319fec77-0843-4207-8c6a-65bf067e4bac"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("319fec77-0843-4207-8c6a-65bf067e4bac"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"319fec77-0843-4207-8c6a-65bf067e4bac","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","exit_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_end":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
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

func Test_processV1QueuecallsReferenceIDIDKickPost(t *testing.T) {

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		referenceID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		expectRes         *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/queuecalls/reference_id/c8794c51-4e4d-46ef-bfa1-5220f66aea87/kick",
				Method: rabbitmqhandler.RequestMethodPost,
			},

			uuid.FromStringOrNil("c8794c51-4e4d-46ef-bfa1-5220f66aea87"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("b1d4d172-52e3-4927-bf10-77eafebd19d8"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b1d4d172-52e3-4927-bf10-77eafebd19d8","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","exit_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_end":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
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

		request *rabbitmqhandler.Request

		referenceID       uuid.UUID
		responseQueuecall *queuecall.Queuecall

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/queuecalls/reference_id/b5f73c26-bcb7-11ed-af77-e397b8122b09",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			uuid.FromStringOrNil("b5f73c26-bcb7-11ed-af77-e397b8122b09"),
			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("b673a022-bcb7-11ed-8212-6fef4fabe382"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b673a022-bcb7-11ed-8212-6fef4fabe382","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","exit_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_end":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				rabbitSock: mockSock,

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
