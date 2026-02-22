package customerhandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func TestCleanupConstants(t *testing.T) {
	if cleanupInterval != 15*time.Minute {
		t.Errorf("cleanupInterval = %v, expected %v", cleanupInterval, 15*time.Minute)
	}
	if unverifiedMaxAge != time.Hour {
		t.Errorf("unverifiedMaxAge = %v, expected %v", unverifiedMaxAge, time.Hour)
	}
}

func Test_cleanupUnverified(t *testing.T) {
	tests := []struct {
		name string

		responseCustomers []*customer.Customer
		expectUpdate      bool
	}{
		{
			name: "no unverified customers",

			responseCustomers: []*customer.Customer{},
			expectUpdate:      false,
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
			expectUpdate: true,
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

			if tt.expectUpdate {
				now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
				mockUtil.EXPECT().TimeNow().Return(&now)

				mockDB.EXPECT().CustomerUpdate(ctx, tt.responseCustomers[0].ID, gomock.Any()).DoAndReturn(
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

			h.cleanupUnverified(ctx)
		})
	}
}
