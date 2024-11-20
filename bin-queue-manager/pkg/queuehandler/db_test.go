package queuehandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/pkg/dbhandler"
)

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		pageSize uint64
		token    string
		filters  map[string]string

		responseQueues []*queue.Queue
	}{
		{
			"normal",

			100,
			"2020-05-03 21:35:02.809",
			map[string]string{
				"customer_id": "e2fc1400-d25a-11ec-9cd3-73acb3bb9c85",
				"deleted":     "false",
			},

			[]*queue.Queue{
				{
					ID: uuid.FromStringOrNil("14ca28c8-d25b-11ec-9e2b-3fc096b513f8"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueueGets(ctx, tt.pageSize, tt.token, tt.filters).Return(tt.responseQueues, nil)

			res, err := h.Gets(ctx, tt.pageSize, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueues, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueues, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		queueID uuid.UUID

		responseQueue *queue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("55ea87c6-d25b-11ec-953d-1be008fae5a9"),

			&queue.Queue{
				ID: uuid.FromStringOrNil("55ea87c6-d25b-11ec-953d-1be008fae5a9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)

			res, err := h.Get(ctx, tt.queueID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueue, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueue, res)
			}
		})
	}
}

func Test_dbDelete(t *testing.T) {

	tests := []struct {
		name string

		queueID uuid.UUID

		responseQueue *queue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("8f1edb5a-d25b-11ec-a7ee-6be8a6eba340"),

			&queue.Queue{
				ID: uuid.FromStringOrNil("8f1edb5a-d25b-11ec-a7ee-6be8a6eba340"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().QueueDelete(ctx, tt.queueID).Return(nil)
			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockNotify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueDeleted, tt.responseQueue)

			res, err := h.dbDelete(ctx, tt.queueID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueue, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueue, res)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		queueID        uuid.UUID
		queueName      string
		detail         string
		routingMethod  queue.RoutingMethod
		tagIDs         []uuid.UUID
		waitActions    []fmaction.Action
		waitTimeout    int
		serviceTimeout int

		responseQueue *queue.Queue
	}{
		{
			name: "normal",

			queueID:       uuid.FromStringOrNil("eabefeea-d25b-11ec-b0bd-a33d2b140e8f"),
			queueName:     "test name",
			detail:        "test detail",
			routingMethod: queue.RoutingMethodRandom,
			tagIDs: []uuid.UUID{
				uuid.FromStringOrNil("7fe4d988-4a77-11ee-a4b5-b36894ae13e5"),
				uuid.FromStringOrNil("803c38ae-4a77-11ee-927d-cf749540055f"),
			},
			waitActions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			waitTimeout:    60000,
			serviceTimeout: 6000000,

			responseQueue: &queue.Queue{
				ID: uuid.FromStringOrNil("eabefeea-d25b-11ec-b0bd-a33d2b140e8f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueueSetBasicInfo(
				ctx,
				tt.queueID,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitActions,
				tt.waitTimeout,
				tt.serviceTimeout,
			).Return(nil)
			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockNotify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueUpdated, tt.responseQueue)

			res, err := h.UpdateBasicInfo(
				ctx,
				tt.queueID,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitActions,
				tt.waitTimeout,
				tt.serviceTimeout,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueue, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueue, res)
			}
		})
	}
}

func Test_UpdateTagIDs(t *testing.T) {

	tests := []struct {
		name string

		queueID uuid.UUID
		tagIDs  []uuid.UUID

		responseQueue *queue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("427a0bd4-d25c-11ec-9602-cfbab70ad4a5"),
			[]uuid.UUID{
				uuid.FromStringOrNil("42aaa064-d25c-11ec-b326-6bd63f326abe"),
			},

			&queue.Queue{
				ID: uuid.FromStringOrNil("427a0bd4-d25c-11ec-9602-cfbab70ad4a5"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueueSetTagIDs(ctx, tt.queueID, tt.tagIDs).Return(nil)
			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockNotify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueUpdated, tt.responseQueue)

			res, err := h.UpdateTagIDs(ctx, tt.queueID, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueue, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueue, res)
			}
		})
	}
}

func Test_UpdateRoutingMethod(t *testing.T) {

	tests := []struct {
		name string

		queueID       uuid.UUID
		routingMethod queue.RoutingMethod

		responseQueue *queue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("895d0e0c-d25c-11ec-b633-bf12408d3840"),
			queue.RoutingMethodRandom,

			&queue.Queue{
				ID: uuid.FromStringOrNil("895d0e0c-d25c-11ec-b633-bf12408d3840"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueueSetRoutingMethod(ctx, tt.queueID, tt.routingMethod).Return(nil)
			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockNotify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueUpdated, tt.responseQueue)

			res, err := h.UpdateRoutingMethod(ctx, tt.queueID, tt.routingMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueue, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueue, res)
			}
		})
	}
}

func Test_UpdateWaitActionsAndTimeouts(t *testing.T) {

	tests := []struct {
		name string

		queueID        uuid.UUID
		waitActions    []fmaction.Action
		waitTimeout    int
		serviceTimeout int

		responseQueue *queue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("bbd57b08-d25c-11ec-9ac2-97f8bfe533ea"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			600000,
			6000000,

			&queue.Queue{
				ID: uuid.FromStringOrNil("bbd57b08-d25c-11ec-9ac2-97f8bfe533ea"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueueSetWaitActionsAndTimeouts(ctx, tt.queueID, tt.waitActions, tt.waitTimeout, tt.serviceTimeout).Return(nil)
			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockNotify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueUpdated, tt.responseQueue)

			res, err := h.UpdateWaitActionsAndTimeouts(ctx, tt.queueID, tt.waitActions, tt.waitTimeout, tt.serviceTimeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueue, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueue, res)
			}
		})
	}
}

func Test_UpdateExecute(t *testing.T) {

	tests := []struct {
		name string

		queueID uuid.UUID
		execute queue.Execute

		responseQueue *queue.Queue
	}{
		{
			"set the execute stop to run",

			uuid.FromStringOrNil("2a4d6dc0-d25d-11ec-96c0-7773a7853963"),
			queue.ExecuteRun,

			&queue.Queue{
				ID:      uuid.FromStringOrNil("2a4d6dc0-d25d-11ec-96c0-7773a7853963"),
				Execute: queue.ExecuteStop,
			},
		},
		{
			"set the execute stop to stop",

			uuid.FromStringOrNil("ee418518-d25d-11ec-85fb-53edf0302f1b"),
			queue.ExecuteStop,

			&queue.Queue{
				ID:      uuid.FromStringOrNil("ee418518-d25d-11ec-85fb-53edf0302f1b"),
				Execute: queue.ExecuteStop,
			},
		},
		{
			"set the execute run to stop",

			uuid.FromStringOrNil("fcb25d48-d25d-11ec-b24f-f351a11e35da"),
			queue.ExecuteStop,

			&queue.Queue{
				ID:      uuid.FromStringOrNil("fcb25d48-d25d-11ec-b24f-f351a11e35da"),
				Execute: queue.ExecuteRun,
			},
		},
		{
			"set the execute run to run",

			uuid.FromStringOrNil("0ab54d9c-d25e-11ec-bd35-fff3fd5a3028"),
			queue.ExecuteRun,

			&queue.Queue{
				ID:      uuid.FromStringOrNil("0ab54d9c-d25e-11ec-bd35-fff3fd5a3028"),
				Execute: queue.ExecuteRun,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)

			if tt.responseQueue.Execute != tt.execute {
				mockDB.EXPECT().QueueSetExecute(ctx, tt.queueID, tt.execute).Return(nil)
				mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
				mockNotify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueUpdated, tt.responseQueue)

				if tt.execute == queue.ExecuteRun && tt.responseQueue.Execute == queue.ExecuteStop {
					mockReq.EXPECT().QueueV1QueueExecuteRun(ctx, tt.queueID, 100)
				}
			}

			res, err := h.UpdateExecute(ctx, tt.queueID, tt.execute)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueue, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueue, res)
			}
		})
	}
}

func Test_RemoveQueuecallID(t *testing.T) {

	tests := []struct {
		name string

		queueID     uuid.UUID
		queuecallID uuid.UUID

		responseQueue *queue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("c0dc4682-d542-11ee-961d-5f6dc6c01c7e"),
			uuid.FromStringOrNil("c169ab4e-d542-11ee-89a9-13ae647f6142"),

			&queue.Queue{
				ID: uuid.FromStringOrNil("c0dc4682-d542-11ee-961d-5f6dc6c01c7e"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().QueueRemoveWaitQueueCall(ctx, tt.queueID, tt.queuecallID).Return(nil)
			mockDB.EXPECT().QueueRemoveServiceQueueCall(ctx, tt.queueID, tt.queuecallID).Return(nil)
			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockNotify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueUpdated, tt.responseQueue)

			res, err := h.RemoveQueuecallID(ctx, tt.queueID, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueue, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueue, res)
			}
		})
	}
}
