package queuehandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
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

		expectFilters map[queue.Field]any
	}{
		{
			name: "normal",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("51813b9e-f08b-11ee-ae42-e79a06af2749"),
			},

			responseQueues: []*queue.Queue{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("fdc57f30-f08d-11ee-8b48-1b3c47cdfa9c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("fe283e18-f08d-11ee-afd2-bf4becedd573"),
					},
				},
			},

			expectFilters: map[queue.Field]any{
				queue.FieldCustomerID: uuid.FromStringOrNil("51813b9e-f08b-11ee-ae42-e79a06af2749"),
				queue.FieldDeleted:    false,
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

			mockDB.EXPECT().QueueList(ctx, uint64(1000), "", tt.expectFilters).Return(tt.responseQueues, nil)

			for _, q := range tt.responseQueues {
				fields := map[queue.Field]any{
					queue.FieldExecute: queue.ExecuteStop,
				}
				mockDB.EXPECT().QueueUpdate(ctx, q.ID, fields).Return(nil)
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
