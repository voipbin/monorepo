package accounthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/dbhandler"
)

func Test_IsValidBalanceByCustomerID(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID

		responseAccount *account.Account
		expectRes       bool
	}

	tests := []test{
		{
			name: "account has enough balance",

			customerID: uuid.FromStringOrNil("87abc36a-09f5-11ee-9b1c-d3d80f26eacd"),

			responseAccount: &account.Account{
				ID:       uuid.FromStringOrNil("8802cee4-09f5-11ee-9491-378bbbda7e50"),
				Balance:  10.0,
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			expectRes: true,
		},
		{
			name: "account has no balance but the type is admin",

			customerID: uuid.FromStringOrNil("a0c6f8c4-09f5-11ee-8ad8-4f77e9290706"),

			responseAccount: &account.Account{
				ID:       uuid.FromStringOrNil("a0e702cc-09f5-11ee-a6d0-e77ef7485142"),
				Type:     account.TypeAdmin,
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			expectRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountGetByCustomerID(ctx, tt.customerID).Return(tt.responseAccount, nil)

			res, err := h.IsValidBalanceByCustomerID(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
