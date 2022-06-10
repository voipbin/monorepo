package linehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/accounthandler"
)

func Test_Setup(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID

		responseAccount *account.Account
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),

			responseAccount: &account.Account{
				ID:         uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
				LineSecret: "ba5f0575d826d5b4a052a43145ef1391",
				LineToken:  "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAccount := accounthandler.NewMockAccountHandler(mc)
			h := lineHandler{
				accountHandler: mockAccount,
			}

			ctx := context.Background()

			mockAccount.EXPECT().Get(ctx, tt.customerID).Return(tt.responseAccount, nil)

			if err := h.Setup(ctx, tt.customerID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
