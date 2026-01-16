package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

func Test_AccountCreate(t *testing.T) {

	type test struct {
		name            string
		account         *account.Account
		responseCurTime string

		expectRes *account.Account
	}

	tests := []test{
		{
			name: "have all",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ec5d6fba-fdf3-11ed-9329-5b12d37e3b82"),
					CustomerID: uuid.FromStringOrNil("876fb2c6-796d-4925-aaf0-570b0a4323bb"),
				},
				Type:   account.TypeLine,
				Name:   "test name",
				Detail: "test detail",
				Secret: "test secret",
				Token:  "test token",
			},
			responseCurTime: "2020-04-18T03:22:17.995000",

			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ec5d6fba-fdf3-11ed-9329-5b12d37e3b82"),
					CustomerID: uuid.FromStringOrNil("876fb2c6-796d-4925-aaf0-570b0a4323bb"),
				},
				Type:   account.TypeLine,
				Name:   "test name",
				Detail: "test detail",
				Secret: "test secret",
				Token:  "test token",

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: commondatabasehandler.DefaultTimeStamp,
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},
		},
		{
			name: "empty",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ec8d1c56-fdf3-11ed-83a6-2bfbd5b33bd6"),
				},
			},
			responseCurTime: "2020-04-18T03:22:17.995000",

			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ec8d1c56-fdf3-11ed-83a6-2bfbd5b33bd6"),
				},

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: commondatabasehandler.DefaultTimeStamp,
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.account.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.account.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created account. call: %v", res)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccountSet(t *testing.T) {
	tests := []struct {
		name    string
		account *account.Account

		id          uuid.UUID
		accountName string
		detail      string
		secret      string
		token       string

		responseCurTime string
		expectRes       *account.Account
	}{
		{
			name: "normal",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("463bfbc0-fee2-11ed-81c2-639d6bd87cf4"),
					CustomerID: uuid.FromStringOrNil("826fdb34-fee2-11ed-ae20-83883bd52100"),
				},
				Type: account.TypeLine,
			},

			id:          uuid.FromStringOrNil("463bfbc0-fee2-11ed-81c2-639d6bd87cf4"),
			accountName: "test name",
			detail:      "test detail",
			secret:      "test secret",
			token:       "test token",

			responseCurTime: "2020-04-18T03:22:17.995000",
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("463bfbc0-fee2-11ed-81c2-639d6bd87cf4"),
					CustomerID: uuid.FromStringOrNil("826fdb34-fee2-11ed-ae20-83883bd52100"),
				},
				Type:     account.TypeLine,
				Name:     "test name",
				Detail:   "test detail",
				Secret:   "test secret",
				Token:    "test token",
				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.AccountSet(ctx, tt.id, tt.accountName, tt.detail, tt.secret, tt.token); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.account.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccountGet(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID

		responseAccount *account.Account
	}{
		{
			"normal",

			uuid.FromStringOrNil("9df9c40e-e427-11ec-b9aa-13b03cb8a3c9"),

			&account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9df9c40e-e427-11ec-b9aa-13b03cb8a3c9"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

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

func Test_AccountList(t *testing.T) {

	tests := []struct {
		name     string
		accounts []*account.Account

		token   string
		limit   uint64
		filters map[account.Field]any

		responseCurTime string
		expectRes       []*account.Account
	}{
		{
			name: "normal",
			accounts: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("151e39da-3e16-11ef-955d-4711f28377ec"),
						CustomerID: uuid.FromStringOrNil("157d41b4-3e16-11ef-a8ed-4b84a868d055"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("15b4e826-3e16-11ef-8cff-47069c33bcae"),
						CustomerID: uuid.FromStringOrNil("157d41b4-3e16-11ef-a8ed-4b84a868d055"),
					},
				},
			},

			token: "2022-06-18 03:22:17.995000",
			limit: 100,
			filters: map[account.Field]any{
				account.FieldDeleted:    false,
				account.FieldCustomerID: uuid.FromStringOrNil("157d41b4-3e16-11ef-a8ed-4b84a868d055"),
			},

			responseCurTime: "2022-04-18 03:22:17.995000",
			expectRes: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("151e39da-3e16-11ef-955d-4711f28377ec"),
						CustomerID: uuid.FromStringOrNil("157d41b4-3e16-11ef-a8ed-4b84a868d055"),
					},
					TMCreate: "2022-04-18 03:22:17.995000",
					TMUpdate: commondatabasehandler.DefaultTimeStamp,
					TMDelete: commondatabasehandler.DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("15b4e826-3e16-11ef-8cff-47069c33bcae"),
						CustomerID: uuid.FromStringOrNil("157d41b4-3e16-11ef-a8ed-4b84a868d055"),
					},
					TMCreate: "2022-04-18 03:22:17.995000",
					TMUpdate: commondatabasehandler.DefaultTimeStamp,
					TMDelete: commondatabasehandler.DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, c := range tt.accounts {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().AccountSet(gomock.Any(), gomock.Any())
				if err := h.AccountCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.AccountList(ctx, tt.limit, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccountDelete(t *testing.T) {

	type test struct {
		name    string
		account *account.Account

		id uuid.UUID

		responseCurTime string

		expectRes *account.Account
	}

	tests := []test{
		{
			name: "have all",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fe975e76-1bdc-11f0-9f72-b3f86287ed78"),
					CustomerID: uuid.FromStringOrNil("876fb2c6-796d-4925-aaf0-570b0a4323bb"),
				},
			},

			id:              uuid.FromStringOrNil("fe975e76-1bdc-11f0-9f72-b3f86287ed78"),
			responseCurTime: "2020-04-18T03:22:17.995000",

			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fe975e76-1bdc-11f0-9f72-b3f86287ed78"),
					CustomerID: uuid.FromStringOrNil("876fb2c6-796d-4925-aaf0-570b0a4323bb"),
				},

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountGet(ctx, tt.account.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if errDelete := h.AccountDelete(ctx, tt.account.ID); errDelete != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDelete)
			}

			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.account.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
