package linehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/accounthandler"
)

func Test_getClient(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID

		responseAccount *account.Account
	}{
		{
			"normal",

			uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),

			&account.Account{
				ID:         uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
				LineSecret: "32bf083c-e4ab-11ec-9e38-6b9bdcde4e32",
				LineToken:  "36d0a6d8-e4ab-11ec-ba26-3bbd1a52af96",
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

			_, err := h.getClient(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Send(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		destination string
		text        string
		medias      []media.Media

		responseAccount *account.Account
	}{
		{
			"normal",

			uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
			"Ud871bcaf7c3ad13d2a0b0d78a42a287f",
			"hi there, This is a test message. :)",
			[]media.Media{},

			&account.Account{
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

			if err := h.Send(ctx, tt.customerID, tt.destination, tt.text, tt.medias); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
