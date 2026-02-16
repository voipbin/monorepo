package customerhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Freeze(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		responseCustomerGet *customer.Customer

		expectDBFreeze  bool
		expectDBGet2    bool
		expectPublish   bool
		expectErr       bool
		responseCustomerGet2 *customer.Customer
	}{
		{
			name: "active customer - success",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusActive,
			},
			expectDBFreeze: true,
			expectDBGet2:   true,
			expectPublish:  true,
			expectErr:      false,
			responseCustomerGet2: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusFrozen,
			},
		},
		{
			name: "already frozen - idempotent",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusFrozen,
			},
			expectDBFreeze: false,
			expectDBGet2:   false,
			expectPublish:  false,
			expectErr:      false,
		},
		{
			name: "deleted customer - error",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusDeleted,
			},
			expectDBFreeze: false,
			expectDBGet2:   false,
			expectPublish:  false,
			expectErr:      true,
		},
		{
			name: "deleted customer with tm_delete - error",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: func() *customer.Customer {
				tmDelete := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				return &customer.Customer{
					ID:       uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
					Status:   customer.StatusActive,
					TMDelete: &tmDelete,
				}
			}(),
			expectDBFreeze: false,
			expectDBGet2:   false,
			expectPublish:  false,
			expectErr:      true,
		},
		{
			name: "get customer fails - error",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: nil,
			expectDBFreeze:      false,
			expectDBGet2:        false,
			expectPublish:       false,
			expectErr:           true,
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

			// initial Get
			if tt.responseCustomerGet == nil {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf("not found"))
			} else {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGet, nil)
			}

			if tt.expectDBFreeze {
				mockDB.EXPECT().CustomerFreeze(gomock.Any(), tt.id).Return(nil)
			}

			if tt.expectDBGet2 {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGet2, nil)
			}

			if tt.expectPublish {
				mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerFrozen, tt.responseCustomerGet2).Return()
			}

			res, err := h.Freeze(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if res == nil {
				t.Errorf("Expected result, got nil")
			}
		})
	}
}

func Test_Freeze_DBFreezeError(t *testing.T) {
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

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
		ID:     id,
		Status: customer.StatusActive,
	}, nil)
	mockDB.EXPECT().CustomerFreeze(gomock.Any(), id).Return(fmt.Errorf("db error"))

	_, err := h.Freeze(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func Test_Freeze_GetAfterFreezeError(t *testing.T) {
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

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
		ID:     id,
		Status: customer.StatusActive,
	}, nil)
	mockDB.EXPECT().CustomerFreeze(gomock.Any(), id).Return(nil)
	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(nil, fmt.Errorf("not found"))

	_, err := h.Freeze(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func Test_Recover(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		responseCustomerGet *customer.Customer

		expectDBRecover bool
		expectDBGet2    bool
		expectPublish   bool
		expectErr       bool
		responseCustomerGet2 *customer.Customer
	}{
		{
			name: "frozen customer - success",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusFrozen,
			},
			expectDBRecover: true,
			expectDBGet2:    true,
			expectPublish:   true,
			expectErr:       false,
			responseCustomerGet2: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusActive,
			},
		},
		{
			name: "active customer - error",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusActive,
			},
			expectDBRecover: false,
			expectDBGet2:    false,
			expectPublish:   false,
			expectErr:       true,
		},
		{
			name: "deleted customer - error",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusDeleted,
			},
			expectDBRecover: false,
			expectDBGet2:    false,
			expectPublish:   false,
			expectErr:       true,
		},
		{
			name:                "get customer fails - error",
			id:                  uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: nil,
			expectDBRecover:     false,
			expectDBGet2:        false,
			expectPublish:       false,
			expectErr:           true,
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

			// initial Get
			if tt.responseCustomerGet == nil {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf("not found"))
			} else {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGet, nil)
			}

			if tt.expectDBRecover {
				mockDB.EXPECT().CustomerRecover(gomock.Any(), tt.id).Return(nil)
			}

			if tt.expectDBGet2 {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGet2, nil)
			}

			if tt.expectPublish {
				mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerRecovered, tt.responseCustomerGet2).Return()
			}

			res, err := h.Recover(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if res == nil {
				t.Errorf("Expected result, got nil")
			}
		})
	}
}

func Test_Recover_DBRecoverError(t *testing.T) {
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

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
		ID:     id,
		Status: customer.StatusFrozen,
	}, nil)
	mockDB.EXPECT().CustomerRecover(gomock.Any(), id).Return(fmt.Errorf("db error"))

	_, err := h.Recover(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func Test_Recover_GetAfterRecoverError(t *testing.T) {
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

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
		ID:     id,
		Status: customer.StatusFrozen,
	}, nil)
	mockDB.EXPECT().CustomerRecover(gomock.Any(), id).Return(nil)
	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(nil, fmt.Errorf("not found"))

	_, err := h.Recover(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
