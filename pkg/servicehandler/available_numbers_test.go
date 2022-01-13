package servicehandler

import (
	"testing"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
)

func TestAvailableNumberGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name        string
		user        *user.User
		limit       uint64
		countryCode string
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			10,
			"US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().NMV1AvailableNumberGets(gomock.Any(), tt.user.ID, tt.limit, tt.countryCode).Return([]nmavailablenumber.AvailableNumber{}, nil)

			_, err := h.AvailableNumberGets(tt.user, tt.limit, tt.countryCode)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
