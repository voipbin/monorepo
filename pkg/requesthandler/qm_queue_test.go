package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestQMV1QueueGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("6cf22a94-7ff1-11ec-9254-5371564adf91"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=6cf22a94-7ff1-11ec-9254-5371564adf91",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"07e42460-6159-11ec-8191-3b89ed95cdb5"}]`),
			},
			[]qmqueue.Queue{
				{
					ID: uuid.FromStringOrNil("07e42460-6159-11ec-8191-3b89ed95cdb5"),
				},
			},
		},
		{
			"2 results",

			uuid.FromStringOrNil("6cf22a94-7ff1-11ec-9254-5371564adf91"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=6cf22a94-7ff1-11ec-9254-5371564adf91",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"08a7c974-6159-11ec-9b3d-0f52d15f98f7"},{"id":"08c9ef2c-6159-11ec-9540-8b38d1cb2283"}]`),
			},
			[]qmqueue.Queue{
				{
					ID: uuid.FromStringOrNil("08a7c974-6159-11ec-9b3d-0f52d15f98f7"),
				},
				{
					ID: uuid.FromStringOrNil("08c9ef2c-6159-11ec-9540-8b38d1cb2283"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueueGets(ctx, tt.customerID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1QueueGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/a2764422-6159-11ec-8d87-975236f7d7b7",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a2764422-6159-11ec-8d87-975236f7d7b7"}`),
			},
			&qmqueue.Queue{
				ID: uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueueGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestQMVQueueCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		customerID     uuid.UUID
		queueName      string
		detail         string
		routingMethod  qmqueue.RoutingMethod
		tagIDs         []uuid.UUID
		waitActions    []fmaction.Action
		timeoutWait    int
		timeoutService int

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("6cf22a94-7ff1-11ec-9254-5371564adf91"),
			"name",
			"detail",
			qmqueue.RoutingMethodRandom,
			[]uuid.UUID{
				uuid.FromStringOrNil("fdbf3fdc-6159-11ec-9263-734d393b9759"),
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			10000,
			100000,

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"6cf22a94-7ff1-11ec-9254-5371564adf91","name":"name","detail":"detail","routing_method":"random","tag_ids":["fdbf3fdc-6159-11ec-9263-734d393b9759"],"wait_actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"wait_timeout":10000,"service_timeout":100000}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"}`),
			},
			&qmqueue.Queue{
				ID: uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueueCreate(ctx, tt.customerID, tt.queueName, tt.detail, tt.routingMethod, tt.tagIDs, tt.waitActions, tt.timeoutWait, tt.timeoutService)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1QueueDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/a2764422-6159-11ec-8d87-975236f7d7b7",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a2764422-6159-11ec-8d87-975236f7d7b7"}`),
			},
			&qmqueue.Queue{
				ID: uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueueDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestQMVQueueUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id        uuid.UUID
		queueName string
		detail    string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("bacc13d4-615a-11ec-a73d-ff4194d49ef7"),
			"name",
			"detail",

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/bacc13d4-615a-11ec-a73d-ff4194d49ef7",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"name","detail":"detail"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bacc13d4-615a-11ec-a73d-ff4194d49ef7"}`),
			},
			&qmqueue.Queue{
				ID: uuid.FromStringOrNil("bacc13d4-615a-11ec-a73d-ff4194d49ef7"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueueUpdate(ctx, tt.id, tt.queueName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQMVQueueUpdateTagIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id     uuid.UUID
		tagIDs []uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
			[]uuid.UUID{
				uuid.FromStringOrNil("2c07e118-615b-11ec-a5cd-0fb1d1ab5c67"),
			},

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/2bdd3418-615b-11ec-80a9-a73788a62c03/tag_ids",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"tag_ids":["2c07e118-615b-11ec-a5cd-0fb1d1ab5c67"]}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2bdd3418-615b-11ec-80a9-a73788a62c03"}`),
			},
			&qmqueue.Queue{
				ID: uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueueUpdateTagIDs(ctx, tt.id, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestQMVQueueUpdateRoutingMethod(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id            uuid.UUID
		routingMethod qmqueue.RoutingMethod

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
			qmqueue.RoutingMethodRandom,

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/2bdd3418-615b-11ec-80a9-a73788a62c03/routing_method",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"routing_method":"random"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2bdd3418-615b-11ec-80a9-a73788a62c03"}`),
			},
			&qmqueue.Queue{
				ID: uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueueUpdateRoutingMethod(ctx, tt.id, tt.routingMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestQMVQueueUpdateActions(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id             uuid.UUID
		waitActions    []fmaction.Action
		timeoutWait    int
		timeoutService int

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			10000,
			100000,
			// qmqueue.RoutingMethodRandom,

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/2bdd3418-615b-11ec-80a9-a73788a62c03/wait_actions",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"wait_actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"wait_timeout":10000,"service_timeout":100000}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2bdd3418-615b-11ec-80a9-a73788a62c03"}`),
			},
			&qmqueue.Queue{
				ID: uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueueUpdateActions(ctx, tt.id, tt.waitActions, tt.timeoutWait, tt.timeoutService)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestQMVQueueCreateQueuecall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id                    uuid.UUID
		referenceType         qmqueuecall.ReferenceType
		referenceID           uuid.UUID
		referenceActiveflowID uuid.UUID
		exitActionID          uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *qmqueuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
			qmqueuecall.ReferenceTypeCall,
			uuid.FromStringOrNil("72beeeac-615c-11ec-bb63-4b76d4878b1d"),
			uuid.FromStringOrNil("8b4d5618-af57-11ec-ba45-7fed62f4b346"),
			uuid.FromStringOrNil("7aa3cc0a-615c-11ec-89fc-3f90491bf4e4"),

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/2bdd3418-615b-11ec-80a9-a73788a62c03/queuecalls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"reference_type":"call","reference_id":"72beeeac-615c-11ec-bb63-4b76d4878b1d","reference_activeflow_id":"8b4d5618-af57-11ec-ba45-7fed62f4b346","exit_action_id":"7aa3cc0a-615c-11ec-89fc-3f90491bf4e4"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"ad7799f4-615c-11ec-b77c-87ab9fdc627c"}`),
			},

			&qmqueuecall.Queuecall{
				ID: uuid.FromStringOrNil("ad7799f4-615c-11ec-b77c-87ab9fdc627c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueueCreateQueuecall(ctx, tt.id, tt.referenceType, tt.referenceID, tt.referenceActiveflowID, tt.exitActionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
