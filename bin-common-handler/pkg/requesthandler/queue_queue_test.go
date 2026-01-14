package requesthandler

import (
	"context"
	"reflect"
	"testing"

	qmqueue "monorepo/bin-queue-manager/models/queue"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_QueueV1QueueGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[qmqueue.Field]any

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     []qmqueue.Queue
	}{
		{
			"normal",

			"2020-09-20T03:23:20.995000",
			10,
			map[qmqueue.Field]any{
				qmqueue.FieldDeleted: false,
			},

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queues?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"deleted":false}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"07e42460-6159-11ec-8191-3b89ed95cdb5"}]`),
			},
			[]qmqueue.Queue{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("07e42460-6159-11ec-8191-3b89ed95cdb5"),
					},
				},
			},
		},
		{
			"2 results",

			"2020-09-20T03:23:20.995000",
			10,
			map[qmqueue.Field]any{
				qmqueue.FieldDeleted: false,
			},

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queues?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"deleted":false}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"08a7c974-6159-11ec-9b3d-0f52d15f98f7"},{"id":"08c9ef2c-6159-11ec-9540-8b38d1cb2283"}]`),
			},
			[]qmqueue.Queue{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("08a7c974-6159-11ec-9b3d-0f52d15f98f7"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("08c9ef2c-6159-11ec-9540-8b38d1cb2283"),
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
			h := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.QueueV1QueueGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueueGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queues/a2764422-6159-11ec-8d87-975236f7d7b7",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a2764422-6159-11ec-8d87-975236f7d7b7"}`),
			},
			&qmqueue.Queue{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QueueV1QueueGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueueGetAgents(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		status amagent.Status

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     []amagent.Agent
	}{
		{
			"none",

			uuid.FromStringOrNil("2f31ae1a-b4a2-11ec-9c56-97b273d77408"),
			amagent.StatusNone,

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queues/2f31ae1a-b4a2-11ec-9c56-97b273d77408/agents",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			Data:     []byte(`{"status":""}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2faae1f4-b4a2-11ec-a519-77ff3160d5e2"}]`),
			},
			[]amagent.Agent{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("2faae1f4-b4a2-11ec-a519-77ff3160d5e2"),
					},
				},
			},
		},
		{
			"available",

			uuid.FromStringOrNil("2fdd4374-b4a2-11ec-929d-5b6756eada32"),
			amagent.StatusAvailable,

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queues/2fdd4374-b4a2-11ec-929d-5b6756eada32/agents",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			Data:     []byte(`{"status":"available"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"300c0740-b4a2-11ec-9751-632ef9ed0b46"}]`),
			},
			[]amagent.Agent{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("300c0740-b4a2-11ec-9751-632ef9ed0b46"),
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
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
		filters := map[amagent.Field]any{
			amagent.FieldStatus: tt.status,
		}

		res, err := reqHandler.QueueV1QueueGetAgents(ctx, tt.id, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueueCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		queueName      string
		detail         string
		routingMethod  qmqueue.RoutingMethod
		tagIDs         []uuid.UUID
		waitFlowID     uuid.UUID
		timeoutWait    int
		timeoutService int

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *qmqueue.Queue
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("6cf22a94-7ff1-11ec-9254-5371564adf91"),
			queueName:     "name",
			detail:        "detail",
			routingMethod: qmqueue.RoutingMethodRandom,
			tagIDs: []uuid.UUID{
				uuid.FromStringOrNil("fdbf3fdc-6159-11ec-9263-734d393b9759"),
			},
			waitFlowID:     uuid.FromStringOrNil("f4e6c374-2066-11f0-99fe-73d349835ba0"),
			timeoutWait:    10000,
			timeoutService: 100000,

			expectTarget: "bin-manager.queue-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/queues",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"6cf22a94-7ff1-11ec-9254-5371564adf91","name":"name","detail":"detail","routing_method":"random","tag_ids":["fdbf3fdc-6159-11ec-9263-734d393b9759"],"wait_flow_id":"f4e6c374-2066-11f0-99fe-73d349835ba0","wait_timeout":10000,"service_timeout":100000}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"}`),
			},
			expectRes: &qmqueue.Queue{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
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
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.QueueV1QueueCreate(
				ctx,
				tt.customerID,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitFlowID,
				tt.timeoutWait,
				tt.timeoutService,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueueDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queues/a2764422-6159-11ec-8d87-975236f7d7b7",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a2764422-6159-11ec-8d87-975236f7d7b7"}`),
			},
			&qmqueue.Queue{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QueueV1QueueDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_QueueV1QueueUpdate(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		queueName      string
		detail         string
		routingMethod  qmqueue.RoutingMethod
		tagIDs         []uuid.UUID
		waitFlowID     uuid.UUID
		waitTimeout    int
		serviceTimeout int

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *qmqueue.Queue
	}{
		{
			name: "normal",

			id:            uuid.FromStringOrNil("bacc13d4-615a-11ec-a73d-ff4194d49ef7"),
			queueName:     "name",
			detail:        "detail",
			routingMethod: qmqueue.RoutingMethodRandom,
			tagIDs: []uuid.UUID{
				uuid.FromStringOrNil("5c4085f4-4a81-11ee-a137-b7953610070d"),
				uuid.FromStringOrNil("5ca5f42a-4a81-11ee-b6ba-7b5ab1c95600"),
			},
			waitFlowID:     uuid.FromStringOrNil("7f3bfe40-2067-11f0-ab87-cb53bd94634b"),
			waitTimeout:    60000,
			serviceTimeout: 6000000,

			expectTarget: "bin-manager.queue-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/queues/bacc13d4-615a-11ec-a73d-ff4194d49ef7",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"name","detail":"detail","routing_method":"random","tag_ids":["5c4085f4-4a81-11ee-a137-b7953610070d","5ca5f42a-4a81-11ee-b6ba-7b5ab1c95600"],"wait_flow_id":"7f3bfe40-2067-11f0-ab87-cb53bd94634b","wait_timeout":60000,"service_timeout":6000000}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bacc13d4-615a-11ec-a73d-ff4194d49ef7"}`),
			},
			expectRes: &qmqueue.Queue{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("bacc13d4-615a-11ec-a73d-ff4194d49ef7"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QueueV1QueueUpdate(ctx, tt.id, tt.queueName, tt.detail, tt.routingMethod, tt.tagIDs, tt.waitFlowID, tt.waitTimeout, tt.serviceTimeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueueUpdateTagIDs(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		tagIDs []uuid.UUID

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
			[]uuid.UUID{
				uuid.FromStringOrNil("2c07e118-615b-11ec-a5cd-0fb1d1ab5c67"),
			},

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queues/2bdd3418-615b-11ec-80a9-a73788a62c03/tag_ids",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"tag_ids":["2c07e118-615b-11ec-a5cd-0fb1d1ab5c67"]}`),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2bdd3418-615b-11ec-80a9-a73788a62c03"}`),
			},
			&qmqueue.Queue{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QueueV1QueueUpdateTagIDs(ctx, tt.id, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_QueueV1QueueUpdateRoutingMethod(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		routingMethod qmqueue.RoutingMethod

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
			qmqueue.RoutingMethodRandom,

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queues/2bdd3418-615b-11ec-80a9-a73788a62c03/routing_method",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"routing_method":"random"}`),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2bdd3418-615b-11ec-80a9-a73788a62c03"}`),
			},
			&qmqueue.Queue{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("2bdd3418-615b-11ec-80a9-a73788a62c03"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QueueV1QueueUpdateRoutingMethod(ctx, tt.id, tt.routingMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_QueueV1QueueCreateQueuecall(t *testing.T) {

	tests := []struct {
		name string

		id                    uuid.UUID
		referenceType         qmqueuecall.ReferenceType
		referenceID           uuid.UUID
		referenceActiveflowID uuid.UUID
		exitActionID          uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

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
			&sock.Request{
				URI:      "/v1/queues/2bdd3418-615b-11ec-80a9-a73788a62c03/queuecalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"reference_type":"call","reference_id":"72beeeac-615c-11ec-bb63-4b76d4878b1d","reference_activeflow_id":"8b4d5618-af57-11ec-ba45-7fed62f4b346","exit_action_id":"7aa3cc0a-615c-11ec-89fc-3f90491bf4e4"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"ad7799f4-615c-11ec-b77c-87ab9fdc627c"}`),
			},

			&qmqueuecall.Queuecall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("ad7799f4-615c-11ec-b77c-87ab9fdc627c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QueueV1QueueCreateQueuecall(ctx, tt.id, tt.referenceType, tt.referenceID, tt.referenceActiveflowID, tt.exitActionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueueExecuteRun(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		executeDelay int

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("4ab73968-d197-11ec-ab6a-17c76533c5d6"),
			1000,

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queues/4ab73968-d197-11ec-ab6a-17c76533c5d6/execute_run",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			if tt.executeDelay == DelayNow {
				mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			} else {
				mockSock.EXPECT().RequestPublishWithDelay(tt.expectTarget, tt.expectRequest, tt.executeDelay).Return(nil)
			}

			if err := reqHandler.QueueV1QueueExecuteRun(ctx, tt.id, tt.executeDelay); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_QueueV1QueueUpdateExecute(t *testing.T) {

	tests := []struct {
		name string

		id      uuid.UUID
		execute qmqueue.Execute

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *qmqueue.Queue
	}{
		{
			"run",

			uuid.FromStringOrNil("7f72fd14-d263-11ec-8a58-ef9e846046ae"),
			qmqueue.ExecuteRun,

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queues/7f72fd14-d263-11ec-8a58-ef9e846046ae/execute",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"execute":"run"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7f72fd14-d263-11ec-8a58-ef9e846046ae"}`),
			},

			&qmqueue.Queue{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("7f72fd14-d263-11ec-8a58-ef9e846046ae"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QueueV1QueueUpdateExecute(ctx, tt.id, tt.execute)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
