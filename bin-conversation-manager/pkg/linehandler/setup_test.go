package linehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
)

// note: this test changes the actual test account's webhook address on line.
// need to be careful to test this.
func Test_Setup(t *testing.T) {

	tests := []struct {
		name string

		account *account.Account
	}{
		{
			name: "normal",

			account: &account.Account{
				ID:     uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
				Secret: "ba5f0575d826d5b4a052a43145ef1391",
				Token:  "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			h := lineHandler{}
			ctx := context.Background()

			if err := h.Setup(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
