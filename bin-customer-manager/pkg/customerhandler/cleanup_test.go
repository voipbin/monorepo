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
	if unverifiedMaxAge != 72*time.Hour {
		t.Errorf("unverifiedMaxAge = %v, expected %v", unverifiedMaxAge, 72*time.Hour)
	}
	// Pinned exactly, not only relative to the window: the verification email
	// copy promises "expires in 24 hours", so reverting the TTL to 1h would keep
	// the relational assertion below green while making that copy a lie.
	if emailVerifyTokenTTL != 24*time.Hour {
		t.Errorf("emailVerifyTokenTTL = %v, expected %v", emailVerifyTokenTTL, 24*time.Hour)
	}
	// The window must stay strictly longer than the link TTL, otherwise a user
	// whose link expired has no interval in which to ask for a new one.
	if emailVerifyTokenTTL >= unverifiedMaxAge {
		t.Errorf("emailVerifyTokenTTL = %v, expected less than unverifiedMaxAge %v", emailVerifyTokenTTL, unverifiedMaxAge)
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

			h := &customerHandler{
				db: mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerList(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseCustomers, nil)

			for i := 0; i < tt.expectUpdateCount; i++ {
				mockDB.EXPECT().CustomerUpdate(ctx, tt.responseCustomers[i].ID, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ uuid.UUID, fields map[customer.Field]any) error {
						status, ok := fields[customer.FieldStatus]
						if !ok || status != string(customer.StatusExpired) {
							t.Errorf("Expected status=expired, got: %v", status)
						}
						if _, ok := fields[customer.FieldTMDelete]; ok {
							t.Errorf("Expected tm_delete to not be part of the update")
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

	h := &customerHandler{
		db: mockDB,
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
	mockDB.EXPECT().CustomerUpdate(ctx, customers[0].ID, gomock.Any()).Return(fmt.Errorf("db update error"))

	// second customer update succeeds (continues despite first failure)
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

func Test_CleanupUnverified_DoesNotSetTMDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{
		db: mockDB,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("8f2b7f2c-1c4a-4a3b-9d61-0f2b8a1c7e11")

	mockDB.EXPECT().CustomerList(ctx, uint64(100), gomock.Any(), map[customer.Field]any{
		customer.FieldEmailVerified: false,
		customer.FieldDeleted:       false,
		customer.FieldStatus:        string(customer.StatusInitial),
	}).Return([]*customer.Customer{{ID: customerID, Email: "a@test.com"}}, nil)

	// tm_delete must NOT be part of the update. Leaving it unset is what keeps the
	// customer_deleted cascade reachable and restores the documented invariant
	// "tm_delete set <=> status=deleted" from migration dafeedbccfa5.
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, map[customer.Field]any{
		customer.FieldStatus: string(customer.StatusExpired),
	}).Return(nil)

	got, err := h.CleanupUnverified(ctx)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if got != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", got)
	}
}
