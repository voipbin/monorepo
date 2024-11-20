package queuehandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/pkg/dbhandler"
)

func Test_EventCUCustomerDeleted(t *testing.T) {
	tests := []struct {
		name string

		customer *cucustomer.Customer

		responseQueues []*queue.Queue

		expectFilters map[string]string
	}{
		{
			name: "normal",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("51813b9e-f08b-11ee-ae42-e79a06af2749"),
			},

			responseQueues: []*queue.Queue{
				{
					ID: uuid.FromStringOrNil("fdc57f30-f08d-11ee-8b48-1b3c47cdfa9c"),
				},
				{
					ID: uuid.FromStringOrNil("fe283e18-f08d-11ee-afd2-bf4becedd573"),
				},
			},

			expectFilters: map[string]string{
				"customer_id": "51813b9e-f08b-11ee-ae42-e79a06af2749",
				"deleted":     "false",
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

			mockDB.EXPECT().QueueGets(ctx, uint64(1000), "", tt.expectFilters).Return(tt.responseQueues, nil)

			for _, q := range tt.responseQueues {
				mockDB.EXPECT().QueueSetExecute(ctx, q.ID, queue.ExecuteStop).Return(nil)
				mockDB.EXPECT().QueueDelete(ctx, q.ID).Return(nil)
				mockDB.EXPECT().QueueGet(ctx, q.ID).Return(q, nil)
				mockNotify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueDeleted, q)
			}

			if errDelete := h.EventCUCustomerDeleted(ctx, tt.customer); errDelete != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDelete)
			}
		})
	}
}
