package conferencehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_CountByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		responseCount int
		expectRes     int
	}{
		{
			name:          "normal",
			customerID:    uuid.FromStringOrNil("b2c3d4e5-0001-0001-0001-000000000001"),
			responseCount: 5,
			expectRes:     5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &conferenceHandler{
				db: mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConferenceCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)

			res, err := h.CountByCustomerID(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
