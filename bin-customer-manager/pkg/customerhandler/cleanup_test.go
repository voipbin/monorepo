package customerhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func TestCleanupConstants(t *testing.T) {
	if unverifiedMaxAge != time.Hour {
		t.Errorf("unverifiedMaxAge = %v, expected %v", unverifiedMaxAge, time.Hour)
	}
}

func Test_CleanupUnverified(t *testing.T) {
	tests := []struct {
		name string

		responseCustomers []*customer.Customer
		expectUpdateCount int
		expectExpired     int
	}{
		{
			name: "no unverified customers",

			responseCustomers: []*customer.Customer{},
			expectUpdateCount: 0,
			expectExpired:     0,
		},
		{
			name: "one unverified customer - soft deleted",

			responseCustomers: []*customer.Customer{
				{
					ID:            uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
					Email:         "expired@test.com",
					EmailVerified: false,
					Status:        customer.StatusInitial,
				},
			},
			expectUpdateCount: 1,
			expectExpired:     1,
		},
		{
			name: "multiple unverified customers - all soft deleted",

			responseCustomers: []*customer.Customer{
				{
					ID:            uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
					Email:         "expired1@test.com",
					EmailVerified: false,
					Status:        customer.StatusInitial,
				},
				{
					ID:            uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000002"),
					Email:         "expired2@test.com",
					EmailVerified: false,
					Status:        customer.StatusInitial,
				},
			},
			expectUpdateCount: 2,
			expectExpired:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &customerHandler{
				db:          mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerList(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseCustomers, nil)

			for i := 0; i < tt.expectUpdateCount; i++ {
				now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
				mockUtil.EXPECT().TimeNow().Return(&now)

				mockDB.EXPECT().CustomerUpdate(ctx, tt.responseCustomers[i].ID, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ uuid.UUID, fields map[customer.Field]any) error {
						status, ok := fields[customer.FieldStatus]
						if !ok || status != string(customer.StatusExpired) {
							t.Errorf("Expected status=expired, got: %v", status)
						}
						tmDelete, ok := fields[customer.FieldTMDelete]
						if !ok || tmDelete == nil {
							t.Errorf("Expected tm_delete to be set")
						}
						return nil
					},
				)
			}

			expired, err := h.CleanupUnverified(ctx)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if expired != tt.expectExpired {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectExpired, expired)
			}
		})
	}
}

func Test_CleanupUnverified_updateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &customerHandler{
		db:          mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	customers := []*customer.Customer{
		{
			ID:            uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
			Email:         "fail@test.com",
			EmailVerified: false,
			Status:        customer.StatusInitial,
		},
		{
			ID:            uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000002"),
			Email:         "ok@test.com",
			EmailVerified: false,
			Status:        customer.StatusInitial,
		},
	}

	mockDB.EXPECT().CustomerList(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(customers, nil)

	// first customer update fails
	now1 := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now1)
	mockDB.EXPECT().CustomerUpdate(ctx, customers[0].ID, gomock.Any()).Return(fmt.Errorf("db update error"))

	// second customer update succeeds (continues despite first failure)
	now2 := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now2)
	mockDB.EXPECT().CustomerUpdate(ctx, customers[1].ID, gomock.Any()).Return(nil)

	// should not panic, should process both customers, and the failed row
	// must not be counted as expired
	expired, err := h.CleanupUnverified(ctx)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if expired != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", expired)
	}
}

func Test_CleanupUnverified_listError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &customerHandler{
		db:          mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	mockDB.EXPECT().CustomerList(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("db error"))

	// should not panic, should return the list error
	expired, err := h.CleanupUnverified(ctx)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if expired != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", expired)
	}
}
