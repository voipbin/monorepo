package linehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
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
				ID:     uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
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

func Test_Send(t *testing.T) {

	tests := []struct {
		name string

		conversation *conversation.Conversation
		account      *account.Account
		text         string
		medias       []media.Media

		responseAccount *account.Account
	}{
		{
			name: "normal",

			conversation: &conversation.Conversation{
				ID:          uuid.FromStringOrNil("361e9be2-f134-11ec-961e-9f08635dea9b"),
				CustomerID:  uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
				ReferenceID: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
			},
			account: &account.Account{
				ID:     uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
				Secret: "ba5f0575d826d5b4a052a43145ef1391",
				Token:  "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
			},
			text:   "hi there, This is a test message. :)",
			medias: []media.Media{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			h := lineHandler{}
			ctx := context.Background()

			if err := h.Send(ctx, tt.conversation, tt.account, tt.text, tt.medias); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
