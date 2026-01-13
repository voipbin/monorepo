package accounthandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
)

func Test_setup(t *testing.T) {

	tests := []struct {
		name string

		account *account.Account
	}{
		{
			name: "type is line",

			account: &account.Account{
				Type: account.TypeLine,
			},
		},
		{
			name: "type is sms",

			account: &account.Account{
				Type: account.TypeSMS,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			switch tt.account.Type {
			case account.TypeLine:
				mockLine.EXPECT().Setup(ctx, tt.account.Return(nil)
			}

			if err := h.setup(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
