package servicehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_AvailableNumberGets(t *testing.T) {

	tests := []struct {
		name        string
		agent       *amagent.Agent
		limit       uint64
		countryCode string
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().NumberV1AvailableNumberGets(ctx, tt.agent.CustomerID, tt.limit, tt.countryCode).Return([]nmavailablenumber.AvailableNumber{}, nil)

			_, err := h.AvailableNumberGets(ctx, tt.agent, tt.limit, tt.countryCode)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
