package schedulehandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/dbhandler"
)

func Test_EventCustomerDeleted(t *testing.T) {
	customerID := uuid.FromStringOrNil("a2c9b8fe-6f1c-11f0-96b1-df0b0e5cbd7a")
	curTime := "2026-08-01 03:00:00.000000"

	schedules := []*schedule.Schedule{
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("a3800b8a-6f1c-11f0-b6bd-93a1b21f5a1e"),
				CustomerID: customerID,
			},
			Name: "customer-schedule-1",
		},
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("a3f4b7a2-6f1c-11f0-9d3f-ff8b03bcd0ba"),
				CustomerID: customerID,
			},
			Name: "customer-schedule-2",
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &scheduleHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()

	expectFilters := map[schedule.Field]any{
		schedule.FieldCustomerID: customerID,
		schedule.FieldDeleted:    false,
	}

	mockUtil.EXPECT().TimeGetCurTime().Return(curTime)
	mockDB.EXPECT().ScheduleList(ctx, uint64(eventListLimit), curTime, expectFilters).Return(schedules, nil)

	for _, s := range schedules {
		// Delete: existence check + soft-delete + fetch-back + event
		mockDB.EXPECT().ScheduleGet(ctx, s.ID).Return(s, nil)
		mockDB.EXPECT().ScheduleDelete(ctx, s.ID).Return(nil)
		mockDB.EXPECT().ScheduleGet(ctx, s.ID).Return(s, nil)
		mockNotify.EXPECT().PublishEvent(ctx, schedule.EventTypeScheduleDeleted, s)
	}

	if err := h.EventCustomerDeleted(ctx, &cmcustomer.Customer{ID: customerID}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_EventCustomerDeleted_nilCustomer(t *testing.T) {
	tests := []struct {
		name string

		customer *cmcustomer.Customer
	}{
		{
			name:     "nil customer",
			customer: nil,
		},
		{
			name:     "nil customer id",
			customer: &cmcustomer.Customer{ID: uuid.Nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &scheduleHandler{
				utilHandler: mockUtil,
				db:          mockDB,
			}

			// no db call may happen: the nil-customer platform schedules must
			// stay untouched (design §6)
			if err := h.EventCustomerDeleted(context.Background(), tt.customer); err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_EventCustomerDeleted_listError(t *testing.T) {
	customerID := uuid.FromStringOrNil("b76b3762-6f1c-11f0-9a20-0b5f8b2ea412")
	curTime := "2026-08-01 03:00:00.000000"

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &scheduleHandler{
		utilHandler: mockUtil,
		db:          mockDB,
	}

	ctx := context.Background()

	expectFilters := map[schedule.Field]any{
		schedule.FieldCustomerID: customerID,
		schedule.FieldDeleted:    false,
	}

	mockUtil.EXPECT().TimeGetCurTime().Return(curTime)
	mockDB.EXPECT().ScheduleList(ctx, uint64(eventListLimit), curTime, expectFilters).Return(nil, fmt.Errorf("db error"))

	if err := h.EventCustomerDeleted(ctx, &cmcustomer.Customer{ID: customerID}); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}
