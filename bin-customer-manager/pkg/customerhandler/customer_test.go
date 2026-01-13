package customerhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

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

func Test_validateCreate(t *testing.T) {
	tests := []struct {
		name string

		email string

		responseCustomer *customer.Customer
		expectedRes      bool

		expectedFilterCustomer map[customer.Field]any
		expectedFilterAgent    map[string]string
	}{
		{
			name: "normal1",

			email: "test@voipbin.net",

			responseCustomer: &customer.Customer{
				ID:       uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			expectedRes: true,

			expectedFilterCustomer: map[customer.Field]any{
				customer.FieldDeleted: false,
				customer.FieldEmail:   "test@voipbin.net",
			},
			expectedFilterAgent: map[string]string{
				"deleted":  "false",
				"username": "test@voipbin.net",
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().EmailIsValid(tt.email).Return(true)
			mockDB.EXPECT().CustomerGets(gomock.Any(), uint64(100), "", tt.expectedFilterCustomer).Return([]*customer.Customer{}, nil)
			mockReq.EXPECT().AgentV1AgentGets(gomock.Any(), "", uint64(100), tt.expectedFilterAgent).Return([]amagent.Agent{}, nil)

			res := h.validateCreate(ctx, tt.email)
			if res != tt.expectedRes {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectedRes, res)
			}
		})
	}
}
