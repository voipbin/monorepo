package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

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

		responseCurTime string
		expectRes       *account.Account
	}

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
				Type:          account.TypeNormal,
				Balance:       99.99,
				PaymentType:   account.PaymentTypeNone,
				PaymentMethod: account.PaymentMethodNone,
			},

			"2023-06-07 03:22:17.995000",
			&account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ecdb856-0600-11ee-b746-d3ef5adc8ef7"),
					CustomerID: uuid.FromStringOrNil("6efc4a5e-0600-11ee-9aca-57553e6045e7"),
				},
				Name:          "test name",
				Detail:        "test detail",
				Type:          account.TypeNormal,
				Balance:       99.99,
				PaymentType:   account.PaymentTypeNone,
				PaymentMethod: account.PaymentMethodNone,
				TMCreate:      "2023-06-07 03:22:17.995000",
				TMUpdate:      DefaultTimeStamp,
				TMDelete:      DefaultTimeStamp,
			},
		},
		{
			"empty",

			&account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9b219300-0600-11ee-bd0c-db57aac06783"),
				},
			},

			"2020-04-18 03:22:17.995000",
			&account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9b219300-0600-11ee-bd0c-db57aac06783"),
				},
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.account.ID.Return(nil, fmt.Errorf(""))
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

func Test_AccountGets(t *testing.T) {

	type test struct {
		name     string
		accounts []*account.Account

		filters map[account.Field]any

		responseCurTime string
		expectRes       []*account.Account
	}

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

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("99a99eb2-f3d7-11ee-8c0a-f7457252a2f8"),
						CustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
					},
					TMCreate: "2023-06-08 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("99ceaf40-f3d7-11ee-b8bb-97bb778dce9e"),
						CustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
					},
					TMCreate: "2023-06-08 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
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
				mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
				mockCache.EXPECT().AccountSet(ctx, gomock.Any())
				_ = h.AccountCreate(ctx, c)
			}

			res, err := h.AccountGets(ctx, uint64(1000), utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_AccountGetsByCustomerID(t *testing.T) {

	type test struct {
		name     string
		accounts []*account.Account

		customerID uuid.UUID
		size       uint64

		responseCurTime string
		expectRes       []*account.Account
	}

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

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("21e3a2a6-06ca-11ee-a265-73b6edfdaf51"),
						CustomerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
					},
					TMCreate: "2023-06-08 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("1d6fcb5a-06ca-11ee-96c1-bb6797183957"),
						CustomerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
					},
					TMCreate: "2023-06-08 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
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
				mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
				mockCache.EXPECT().AccountSet(ctx, gomock.Any())
				_ = h.AccountCreate(ctx, c)
			}

			res, err := h.AccountGetsByCustomerID(ctx, tt.customerID, tt.size, utilhandler.TimeGetCurTime())
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

		responseCurTime string
		expectRes       *account.Account
	}

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

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("697786c6-06cc-11ee-88f6-a79092bb719c"),
				},
				Name:     "test name",
				Detail:   "test detail",
				TMCreate: "2023-06-08 03:22:17.995000",
				TMUpdate: "2023-06-08 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.id.Return(nil, fmt.Errorf(""))
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

func Test_AccountAddBalance(t *testing.T) {

	type test struct {
		name    string
		account *account.Account

		accountID uuid.UUID
		balance   float32

		responseCurTime string
		expectRes       *account.Account
	}

	tests := []test{
		{
			name: "normal",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c05e0eba-09bf-11ee-867c-13d325e0d976"),
					CustomerID: uuid.FromStringOrNil("1a547210-06cd-11ee-bf06-abb9387009e2"),
				},
				Balance: 20.0,
			},

			accountID: uuid.FromStringOrNil("c05e0eba-09bf-11ee-867c-13d325e0d976"),
			balance:   888.88,

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c05e0eba-09bf-11ee-867c-13d325e0d976"),
					CustomerID: uuid.FromStringOrNil("1a547210-06cd-11ee-bf06-abb9387009e2"),
				},
				Balance:  908.88,
				TMCreate: "2023-06-08 03:22:17.995000",
				TMUpdate: "2023-06-08 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountAddBalance(ctx, tt.accountID, tt.balance); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.accountID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccountSubtractBalance(t *testing.T) {

	type test struct {
		name    string
		account *account.Account

		accountID uuid.UUID
		balance   float32

		responseCurTime string
		expectRes       *account.Account
	}

	tests := []test{
		{
			name: "normal",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2788b4ce-07b7-11ee-acdb-07679240a451"),
					CustomerID: uuid.FromStringOrNil("0d9c3274-09c0-11ee-a384-1f58f10e9a62"),
				},
				Balance: 20.0,
			},

			accountID: uuid.FromStringOrNil("2788b4ce-07b7-11ee-acdb-07679240a451"),
			balance:   8.88,

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2788b4ce-07b7-11ee-acdb-07679240a451"),
					CustomerID: uuid.FromStringOrNil("0d9c3274-09c0-11ee-a384-1f58f10e9a62"),
				},
				Balance:  11.12,
				TMCreate: "2023-06-08 03:22:17.995000",
				TMUpdate: "2023-06-08 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountSubtractBalance(ctx, tt.accountID, tt.balance); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.accountID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccountUpdatePaymentInfo(t *testing.T) {

	type test struct {
		name    string
		account *account.Account

		id     uuid.UUID
		fields map[account.Field]any

		responseCurTime string
		expectRes       *account.Account
	}

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

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5d0dd0e2-06cd-11ee-a292-3bc4c472124e"),
				},
				PaymentType:   account.PaymentTypePrepaid,
				PaymentMethod: account.PaymentMethodCreditCard,
				TMCreate:      "2023-06-08 03:22:17.995000",
				TMUpdate:      "2023-06-08 03:22:17.995000",
				TMDelete:      DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.id.Return(nil, fmt.Errorf(""))
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

		responseCurTime string
		expectRes       *account.Account
	}

	tests := []test{
		{
			name: "normal",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3d9d1d2c-06d1-11ee-a149-033de1ce53d7"),
				},
			},

			id: uuid.FromStringOrNil("3d9d1d2c-06d1-11ee-a149-033de1ce53d7"),

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3d9d1d2c-06d1-11ee-a149-033de1ce53d7"),
				},
				TMCreate: "2023-06-08 03:22:17.995000",
				TMUpdate: "2023-06-08 03:22:17.995000",
				TMDelete: "2023-06-08 03:22:17.995000",
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.id.Return(nil, fmt.Errorf(""))
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
