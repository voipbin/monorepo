package customerhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Delete(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		responseCustomer *customer.Customer
	}{
		{
			name: "normal1",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),

			responseCustomer: &customer.Customer{
				ID:       uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomer, nil)

			// dbDelete
			mockDB.EXPECT().CustomerDelete(gomock.Any(), tt.id).Return(nil)
			mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomer, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerDeleted, tt.responseCustomer).Return()

			_, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
