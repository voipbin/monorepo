package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/cachehandler"
)

func Test_AccountCreate(t *testing.T) {

	type test struct {
		name string

		account *account.Account

		responseCurTime *time.Time
		expectRes       *account.Account
	}

	tmCreate1 := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	tmCreate2 := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			"have all",
			&account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ecdb856-0600-11ee-b746-d3ef5adc8ef7"),
					CustomerID: uuid.FromStringOrNil("6efc4a5e-0600-11ee-9aca-57553e6045e7"),
				},
				Name:          "test name",
				Detail:        "test detail",
				BalanceCredit: 9999000,
				PaymentType:   account.PaymentTypeNone,
				PaymentMethod: account.PaymentMethodNone,
			},

			&tmCreate1,
			&account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ecdb856-0600-11ee-b746-d3ef5adc8ef7"),
					CustomerID: uuid.FromStringOrNil("6efc4a5e-0600-11ee-9aca-57553e6045e7"),
				},
				Name:          "test name",
				Detail:        "test detail",
				BalanceCredit: 9999000,
				PaymentType:   account.PaymentTypeNone,
				PaymentMethod: account.PaymentMethodNone,
				TMCreate:      &tmCreate1,
				TMUpdate:      nil,
				TMDelete:      nil,
			},
		},
		{
			"empty",

			&account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9b219300-0600-11ee-bd0c-db57aac06783"),
				},
			},

			&tmCreate2,
			&account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9b219300-0600-11ee-bd0c-db57aac06783"),
				},
				TMCreate: &tmCreate2,
				TMUpdate: nil,
				TMDelete: nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.account.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(context.Background(), tt.account.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccountList(t *testing.T) {

	type test struct {
		name     string
		accounts []*account.Account

		filters map[account.Field]any

		responseCurTime *time.Time
		expectRes       []*account.Account
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			accounts: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("99a99eb2-f3d7-11ee-8c0a-f7457252a2f8"),
						CustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("99ceaf40-f3d7-11ee-b8bb-97bb778dce9e"),
						CustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
					},
				},
			},

			filters: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
			},

			responseCurTime: &tmCreate,
			expectRes: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("99a99eb2-f3d7-11ee-8c0a-f7457252a2f8"),
						CustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
					},
					TMCreate: &tmCreate,
					TMUpdate: nil,
					TMDelete: nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("99ceaf40-f3d7-11ee-b8bb-97bb778dce9e"),
						CustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
					},
					TMCreate: &tmCreate,
					TMUpdate: nil,
					TMDelete: nil,
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

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, c := range tt.accounts {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().AccountSet(ctx, gomock.Any())
				_ = h.AccountCreate(ctx, c)
			}

			res, err := h.AccountList(ctx, uint64(1000), utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_AccountListByCustomerID(t *testing.T) {

	type test struct {
		name     string
		accounts []*account.Account

		customerID uuid.UUID
		size       uint64

		responseCurTime *time.Time
		expectRes       []*account.Account
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			accounts: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("1d6fcb5a-06ca-11ee-96c1-bb6797183957"),
						CustomerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("21e3a2a6-06ca-11ee-a265-73b6edfdaf51"),
						CustomerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
					},
				},
			},

			customerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
			size:       10,

			responseCurTime: &tmCreate,
			expectRes: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("21e3a2a6-06ca-11ee-a265-73b6edfdaf51"),
						CustomerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
					},
					TMCreate: &tmCreate,
					TMUpdate: nil,
					TMDelete: nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("1d6fcb5a-06ca-11ee-96c1-bb6797183957"),
						CustomerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
					},
					TMCreate: &tmCreate,
					TMUpdate: nil,
					TMDelete: nil,
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

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, c := range tt.accounts {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().AccountSet(ctx, gomock.Any())
				_ = h.AccountCreate(ctx, c)
			}

			res, err := h.AccountListByCustomerID(ctx, tt.customerID, tt.size, utilhandler.TimeGetCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) < len(tt.accounts) {
				t.Errorf("Wrong match. expect: bigger than %d, got: %d", len(tt.accounts), len(res))
			}
		})
	}
}

func Test_AccountUpdate(t *testing.T) {

	type test struct {
		name    string
		account *account.Account

		id     uuid.UUID
		fields map[account.Field]any

		responseCurTime *time.Time
		expectRes       *account.Account
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("697786c6-06cc-11ee-88f6-a79092bb719c"),
				},
			},

			id: uuid.FromStringOrNil("697786c6-06cc-11ee-88f6-a79092bb719c"),
			fields: map[account.Field]any{
				account.FieldName:   "test name",
				account.FieldDetail: "test detail",
			},

			responseCurTime: &tmCreate,
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("697786c6-06cc-11ee-88f6-a79092bb719c"),
				},
				Name:     "test name",
				Detail:   "test detail",
				TMCreate: &tmCreate,
				TMUpdate: &tmCreate,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// AccountAddBalance, AccountSubtractBalance, and AccountSubtractBalanceWithCheck now use
// SELECT ... FOR UPDATE which is not supported by SQLite. These functions are covered by
// mock-based tests in pkg/accounthandler/db_test.go (same pattern as BillingConsumeAndRecord
// and AccountTopUpTokens).

func Test_AccountUpdatePaymentInfo(t *testing.T) {

	type test struct {
		name    string
		account *account.Account

		id     uuid.UUID
		fields map[account.Field]any

		responseCurTime *time.Time
		expectRes       *account.Account
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5d0dd0e2-06cd-11ee-a292-3bc4c472124e"),
				},
			},

			id: uuid.FromStringOrNil("5d0dd0e2-06cd-11ee-a292-3bc4c472124e"),
			fields: map[account.Field]any{
				account.FieldPaymentType:   account.PaymentTypePrepaid,
				account.FieldPaymentMethod: account.PaymentMethodCreditCard,
			},

			responseCurTime: &tmCreate,
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5d0dd0e2-06cd-11ee-a292-3bc4c472124e"),
				},
				PaymentType:   account.PaymentTypePrepaid,
				PaymentMethod: account.PaymentMethodCreditCard,
				TMCreate:      &tmCreate,
				TMUpdate:      &tmCreate,
				TMDelete:      nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
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

		responseCurTime *time.Time
		expectRes       *account.Account
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3d9d1d2c-06d1-11ee-a149-033de1ce53d7"),
				},
			},

			id: uuid.FromStringOrNil("3d9d1d2c-06d1-11ee-a149-033de1ce53d7"),

			responseCurTime: &tmCreate,
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3d9d1d2c-06d1-11ee-a149-033de1ce53d7"),
				},
				TMCreate: &tmCreate,
				TMUpdate: &tmCreate,
				TMDelete: &tmCreate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
