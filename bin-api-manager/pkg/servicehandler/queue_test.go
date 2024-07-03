package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	qmqueue "monorepo/bin-queue-manager/models/queue"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_QueueGets(t *testing.T) {

	type test struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		expectFilters map[string]string

		responseQueues []qmqueue.Queue
		expectRes      []*qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			"2021-03-01 01:00:00.995000",
			10,

			map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},

			[]qmqueue.Queue{
				{
					ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
				},
			},
			[]*qmqueue.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueueGets(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.responseQueues, nil)

			res, err := h.QueueGets(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res[0], tt.expectRes[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_QueueGet(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent
		id    uuid.UUID

		response  *qmqueue.Queue
		expectRes *qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),

			&qmqueue.Queue{
				ID:         uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&qmqueue.WebhookMessage{
				ID:         uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueueGet(ctx, tt.id).Return(tt.response, nil)

			res, err := h.QueueGet(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueCreate(t *testing.T) {

	type test struct {
		name string

		agent          *amagent.Agent
		queueName      string
		detail         string
		routingMethod  qmqueue.RoutingMethod
		tagIDs         []uuid.UUID
		waitActions    []fmaction.Action
		timeoutWait    int
		timeoutService int

		response  *qmqueue.Queue
		expectRes *qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			"name",
			"detail",
			qmqueue.RoutingMethodRandom,
			[]uuid.UUID{
				uuid.FromStringOrNil("2a743344-6316-11ec-b247-af52c2375309"),
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			100000,
			1000000,

			&qmqueue.Queue{
				ID:         uuid.FromStringOrNil("eb2ee214-6316-11ec-88b2-db9da3dd0931"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&qmqueue.WebhookMessage{
				ID:         uuid.FromStringOrNil("eb2ee214-6316-11ec-88b2-db9da3dd0931"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueueCreate(
				ctx,
				tt.agent.CustomerID,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitActions,
				tt.timeoutWait,
				tt.timeoutService,
			).Return(tt.response, nil)

			res, err := h.QueueCreate(
				ctx,
				tt.agent,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitActions,
				tt.timeoutWait,
				tt.timeoutService,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueDelete(t *testing.T) {

	type test struct {
		name string

		customer *amagent.Agent
		queueID  uuid.UUID

		response  *qmqueue.Queue
		expectRes *qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("6aa878a2-6317-11ec-94b7-c7ba9436173f"),

			&qmqueue.Queue{
				ID:         uuid.FromStringOrNil("6aa878a2-6317-11ec-94b7-c7ba9436173f"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&qmqueue.WebhookMessage{
				ID:         uuid.FromStringOrNil("6aa878a2-6317-11ec-94b7-c7ba9436173f"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueueGet(ctx, tt.queueID).Return(tt.response, nil)
			mockReq.EXPECT().QueueV1QueueDelete(ctx, tt.queueID).Return(tt.response, nil)

			res, err := h.QueueDelete(ctx, tt.customer, tt.queueID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_QueueUpdate(t *testing.T) {

	type test struct {
		name string

		agent          *amagent.Agent
		queueID        uuid.UUID
		queueName      string
		detail         string
		routingMethod  qmqueue.RoutingMethod
		tagIDs         []uuid.UUID
		waitActions    []fmaction.Action
		timeoutWait    int
		timeoutService int

		response  *qmqueue.Queue
		expectRes *qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("116b515e-6391-11ec-a2ab-2b13d87ce328"),
			"name",
			"detail",
			qmqueue.RoutingMethodRandom,
			[]uuid.UUID{
				uuid.FromStringOrNil("4927aacc-4a88-11ee-bc5c-2f5cbe9c7d73"),
				uuid.FromStringOrNil("497d6b06-4a88-11ee-acb4-4b2f731fa7bb"),
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			60000,
			6000000,

			&qmqueue.Queue{
				ID:         uuid.FromStringOrNil("116b515e-6391-11ec-a2ab-2b13d87ce328"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&qmqueue.WebhookMessage{
				ID:         uuid.FromStringOrNil("116b515e-6391-11ec-a2ab-2b13d87ce328"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueueGet(ctx, tt.queueID).Return(tt.response, nil)
			mockReq.EXPECT().QueueV1QueueUpdate(
				ctx,
				tt.queueID,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitActions,
				tt.timeoutWait,
				tt.timeoutService,
			).Return(tt.response, nil)

			res, err := h.QueueUpdate(
				ctx,
				tt.agent,
				tt.queueID,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitActions,
				tt.timeoutWait,
				tt.timeoutService,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueUpdateTagIDs(t *testing.T) {

	type test struct {
		name string

		agent   *amagent.Agent
		queueID uuid.UUID
		tagIDs  []uuid.UUID

		response  *qmqueue.Queue
		expectRes *qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("4f10fcca-6391-11ec-b1a8-cf59a893226a"),
			[]uuid.UUID{
				uuid.FromStringOrNil("50c7c31e-6391-11ec-b1f6-cb24701d7df3"),
			},

			&qmqueue.Queue{
				ID:         uuid.FromStringOrNil("4f10fcca-6391-11ec-b1a8-cf59a893226a"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&qmqueue.WebhookMessage{
				ID:         uuid.FromStringOrNil("4f10fcca-6391-11ec-b1a8-cf59a893226a"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
		{
			"2 items",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("7472d542-6391-11ec-8e92-6f12cb507950"),
			[]uuid.UUID{
				uuid.FromStringOrNil("74963b9a-6391-11ec-84ae-337b926b8136"),
				uuid.FromStringOrNil("74b790d8-6391-11ec-be28-5fd8bcbf3b9c"),
			},

			&qmqueue.Queue{
				ID:         uuid.FromStringOrNil("7472d542-6391-11ec-8e92-6f12cb507950"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&qmqueue.WebhookMessage{
				ID:         uuid.FromStringOrNil("7472d542-6391-11ec-8e92-6f12cb507950"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueueGet(ctx, tt.queueID).Return(tt.response, nil)
			mockReq.EXPECT().QueueV1QueueUpdateTagIDs(ctx, tt.queueID, tt.tagIDs).Return(tt.response, nil)

			res, err := h.QueueUpdateTagIDs(ctx, tt.agent, tt.queueID, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_QueueUpdateRoutingMethod(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		queueID       uuid.UUID
		routingMethod qmqueue.RoutingMethod

		response  *qmqueue.Queue
		expectRes *qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"routing method random",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("af14400a-6391-11ec-baed-7fb98aebe61a"),
			qmqueue.RoutingMethodRandom,

			&qmqueue.Queue{
				ID:         uuid.FromStringOrNil("af14400a-6391-11ec-baed-7fb98aebe61a"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&qmqueue.WebhookMessage{
				ID:         uuid.FromStringOrNil("af14400a-6391-11ec-baed-7fb98aebe61a"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
		{
			"routing method none",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("af2efe86-6391-11ec-8100-c3e8d3057916"),
			qmqueue.RoutingMethodNone,

			&qmqueue.Queue{
				ID:         uuid.FromStringOrNil("af2efe86-6391-11ec-8100-c3e8d3057916"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&qmqueue.WebhookMessage{
				ID:         uuid.FromStringOrNil("af2efe86-6391-11ec-8100-c3e8d3057916"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueueGet(ctx, tt.queueID).Return(tt.response, nil)
			mockReq.EXPECT().QueueV1QueueUpdateRoutingMethod(ctx, tt.queueID, tt.routingMethod).Return(tt.response, nil)

			res, err := h.QueueUpdateRoutingMethod(ctx, tt.agent, tt.queueID, tt.routingMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueUpdateActions(t *testing.T) {

	type test struct {
		name string

		agent          *amagent.Agent
		queueID        uuid.UUID
		waitActions    []fmaction.Action
		timeoutWait    int
		timeoutService int

		response  *qmqueue.Queue
		expectRes *qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"routing method random",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("f4fc8e6a-6391-11ec-bd03-337ff376d96d"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			10000,
			100000,

			&qmqueue.Queue{
				ID:         uuid.FromStringOrNil("f4fc8e6a-6391-11ec-bd03-337ff376d96d"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&qmqueue.WebhookMessage{
				ID:         uuid.FromStringOrNil("f4fc8e6a-6391-11ec-bd03-337ff376d96d"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueueGet(ctx, tt.queueID).Return(tt.response, nil)
			mockReq.EXPECT().QueueV1QueueUpdateActions(ctx, tt.queueID, tt.waitActions, tt.timeoutWait, tt.timeoutService).Return(tt.response, nil)

			res, err := h.QueueUpdateActions(ctx, tt.agent, tt.queueID, tt.waitActions, tt.timeoutWait, tt.timeoutService)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
