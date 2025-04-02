package linehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-conversation-manager/models/account"
)

func Test_getClient(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		account    *account.Account
	}{
		{
			name: "normal",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
				},
				Secret: "32bf083c-e4ab-11ec-9e38-6b9bdcde4e32",
				Token:  "36d0a6d8-e4ab-11ec-ba26-3bbd1a52af96",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			h := lineHandler{}

			ctx := context.Background()

			_, err := h.getClient(ctx, tt.account)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
