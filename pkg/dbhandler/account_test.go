package dbhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/cachehandler"
)

func Test_AccountSet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		account *account.Account
	}{
		{
			"normal",
			&account.Account{
				ID:         uuid.FromStringOrNil("059dacb6-e427-11ec-a483-1797b05e49b7"),
				LineSecret: "0e3439ee-e427-11ec-8984-9b920f8c9545",
				LineToken:  "0e5aac8c-e427-11ec-aa66-078dc8491983",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().AccountSet(gomock.Any(), gomock.Any()).Return(nil)

			if err := h.AccountSet(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AccountGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		accountID uuid.UUID

		responseAccount *account.Account
	}{
		{
			"normal",

			uuid.FromStringOrNil("9df9c40e-e427-11ec-b9aa-13b03cb8a3c9"),

			&account.Account{
				ID: uuid.FromStringOrNil("9df9c40e-e427-11ec-b9aa-13b03cb8a3c9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().AccountGet(gomock.Any(), gomock.Any()).Return(tt.responseAccount, nil)

			res, err := h.AccountGet(ctx, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseAccount, res)
			}
		})
	}
}
