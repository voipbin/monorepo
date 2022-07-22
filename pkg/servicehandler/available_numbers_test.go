package servicehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestAvailableNumberGets(t *testing.T) {

	tests := []struct {
		name        string
		customer    *cscustomer.Customer
		limit       uint64
		countryCode string
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
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

			mockReq.EXPECT().NMV1AvailableNumberGets(gomock.Any(), tt.customer.ID, tt.limit, tt.countryCode).Return([]nmavailablenumber.AvailableNumber{}, nil)

			_, err := h.AvailableNumberGets(tt.customer, tt.limit, tt.countryCode)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
