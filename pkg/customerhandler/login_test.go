package customerhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/helphandler"
)

func Test_Login(t *testing.T) {

	tests := []struct {
		name string

		username string
		password string

		responseGet *customer.Customer
	}{
		{
			"normal",
			"a13c6c24-7ccc-11ec-86c7-133d05b8ea4e",
			"password1",

			&customer.Customer{
				ID:           uuid.FromStringOrNil("a13c6c24-7ccc-11ec-86c7-133d05b8ea4e"),
				Username:     "a13c6c24-7ccc-11ec-86c7-133d05b8ea4e",
				PasswordHash: "$2a$12$z6fM.TRL7XdYJc7Ea.GGHOCIDe46vWl.h485o5hiid454ASroCOga",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockHelp := helphandler.NewMockHelpHandler(mc)

			h := &customerHandler{
				reqHandler:  mockReq,
				db:          mockDB,
				helpHandler: mockHelp,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerGetByUsername(ctx, tt.username).Return(tt.responseGet, nil)
			mockHelp.EXPECT().HashCheck(tt.password, tt.responseGet.PasswordHash).Return(true)

			_, err := h.Login(ctx, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
