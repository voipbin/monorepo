package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_QueuecallGets(t *testing.T) {

	type test struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		responseQueuecalls []qmqueuecall.Queuecall
		expectFilters map[qmqueuecall.Field]any
		expectRes          []*qmqueuecall.WebhookMessage
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

			[]qmqueuecall.Queuecall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("cccf3e1a-6413-11ec-9874-afa5340c4843"),
					},
				},
			},
			map[qmqueuecall.Field]any{
				qmqueuecall.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				qmqueuecall.FieldDeleted:    false,
			},
			[]*qmqueuecall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("cccf3e1a-6413-11ec-9874-afa5340c4843"),
					},
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

			mockReq.EXPECT().QueueV1QueuecallList(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.responseQueuecalls, nil)

			res, err := h.QueuecallList(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res[0], tt.expectRes[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_QueuecallGet(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent
		id    uuid.UUID

		response  *qmqueuecall.Queuecall
		expectRes *qmqueuecall.WebhookMessage
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
			uuid.FromStringOrNil("cd268152-6413-11ec-8e49-4bc7bcc6d465"),

			&qmqueuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cd268152-6413-11ec-8e49-4bc7bcc6d465"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&qmqueuecall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cd268152-6413-11ec-8e49-4bc7bcc6d465"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueuecallGet(ctx, tt.id).Return(tt.response, nil)

			res, err := h.QueuecallGet(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueuecallDelete(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent
		id    uuid.UUID

		response  *qmqueuecall.Queuecall
		expectRes *qmqueuecall.WebhookMessage
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
			uuid.FromStringOrNil("00043d94-6414-11ec-9c13-eb81c8c76e8d"),

			&qmqueuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00043d94-6414-11ec-9c13-eb81c8c76e8d"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&qmqueuecall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00043d94-6414-11ec-9c13-eb81c8c76e8d"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueuecallGet(ctx, tt.id).Return(tt.response, nil)
			mockReq.EXPECT().QueueV1QueuecallDelete(ctx, tt.id).Return(tt.response, nil)

			res, err := h.QueuecallDelete(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueuecallKick(t *testing.T) {

	type test struct {
		name        string
		agent       *amagent.Agent
		queuecallID uuid.UUID

		responseQueuecall *qmqueuecall.Queuecall
		expectRes         *qmqueuecall.WebhookMessage
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
			uuid.FromStringOrNil("968e4108-bcbd-11ed-bbf4-2740d4b50a8e"),

			&qmqueuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("968e4108-bcbd-11ed-bbf4-2740d4b50a8e"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&qmqueuecall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("968e4108-bcbd-11ed-bbf4-2740d4b50a8e"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueuecallGet(ctx, tt.queuecallID).Return(tt.responseQueuecall, nil)
			mockReq.EXPECT().QueueV1QueuecallKick(ctx, tt.queuecallID).Return(tt.responseQueuecall, nil)

			res, err := h.QueuecallKick(ctx, tt.agent, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueuecallKickByReferenceID(t *testing.T) {

	type test struct {
		name string

		agent       *amagent.Agent
		referenceID uuid.UUID

		responseQueuecall *qmqueuecall.Queuecall
		expectRes         *qmqueuecall.WebhookMessage
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
			uuid.FromStringOrNil("b3fef3c2-bcbd-11ed-b53a-77ef688fa9c5"),

			&qmqueuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b418e2d2-bcbd-11ed-ab24-6b4bb882a906"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&qmqueuecall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b418e2d2-bcbd-11ed-ab24-6b4bb882a906"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueuecallGetByReferenceID(ctx, tt.referenceID).Return(tt.responseQueuecall, nil)
			mockReq.EXPECT().QueueV1QueuecallKick(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)

			res, err := h.QueuecallKickByReferenceID(ctx, tt.agent, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
