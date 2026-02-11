package queuehandler

import (
	"context"
	"reflect"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		queueName      string
		detail         string
		routingMethod  queue.RoutingMethod
		tagIDs         []uuid.UUID
		waitFlowID     uuid.UUID
		waitTimeout    int
		serviceTimeout int

		responseUUID  uuid.UUID
		responseQueue *queue.Queue

		expectQueue *queue.Queue
		expectRes   *queue.Queue
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("1ed812a6-7f56-11ec-82c1-8bb47b0f9d98"),
			queueName:     "test name",
			detail:        "tes detail",
			routingMethod: queue.RoutingMethodRandom,
			tagIDs: []uuid.UUID{
				uuid.FromStringOrNil("074b6e1e-60e6-11ec-9dc5-4bc92b81a572"),
			},
			waitFlowID:     uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30"),
			waitTimeout:    100000,
			serviceTimeout: 1000000,

			responseUUID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
			responseQueue: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
			},

			expectQueue: &queue.Queue{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
					CustomerID: uuid.FromStringOrNil("1ed812a6-7f56-11ec-82c1-8bb47b0f9d98"),
				},
				Name:          "test name",
				Detail:        "test detail",
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("074b6e1e-60e6-11ec-9dc5-4bc92b81a572"),
				},
				Execute:             queue.ExecuteStop,
				WaitFlowID:          uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30"),
				WaitTimeout:         100000,
				ServiceTimeout:      1000000,
				WaitQueuecallIDs:    []uuid.UUID{},
				ServiceQueuecallIDs: []uuid.UUID{},
				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
			},
			expectRes: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, tt.customerID, bmaccount.ResourceTypeQueue).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().QueueCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().QueueGet(ctx, gomock.Any()).Return(tt.responseQueue, nil)

			res, err := h.Create(
				ctx,
				tt.customerID,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitFlowID,
				tt.waitTimeout,
				tt.serviceTimeout,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
