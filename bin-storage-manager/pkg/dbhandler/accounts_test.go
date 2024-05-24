package dbhandler

import (
	"context"
	"fmt"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/pkg/cachehandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_AccountCreate(t *testing.T) {

	tests := []struct {
		name string
		file *account.Account

		responseCurTime string
		expectRes       *account.Account
	}{
		{
			"normal",
			&account.Account{
				ID:         uuid.FromStringOrNil("7e1d4424-198d-11ef-8962-5b7dcd4e37e8"),
				CustomerID: uuid.FromStringOrNil("7e4d9caa-198d-11ef-a42b-abbbe058dea6"),

				TotalFileCount: 1,
				TotalFileSize:  1024,
			},

			"2024-05-18 03:22:17.995000",
			&account.Account{
				ID:         uuid.FromStringOrNil("7e1d4424-198d-11ef-8962-5b7dcd4e37e8"),
				CustomerID: uuid.FromStringOrNil("7e4d9caa-198d-11ef-a42b-abbbe058dea6"),

				TotalFileCount: 1,
				TotalFileSize:  1024,

				TMCreate: "2024-05-18 03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
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
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.file); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.file.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.file.ID)
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

	tests := []struct {
		name     string
		accounts []account.Account

		size    uint64
		filters map[string]string

		responseCurTime string
		expectRes       []*account.Account
	}{
		{
			"normal",
			[]account.Account{
				{
					ID:         uuid.FromStringOrNil("5227c0ce-198d-11ef-b06c-a744dbfcde74"),
					CustomerID: uuid.FromStringOrNil("52642a32-198d-11ef-ae41-7f16629231ae"),
				},
				{
					ID:         uuid.FromStringOrNil("529e6936-198d-11ef-afd6-57ccd5f20689"),
					CustomerID: uuid.FromStringOrNil("52642a32-198d-11ef-ae41-7f16629231ae"),
				},
			},

			10,
			map[string]string{
				"customer_id": "52642a32-198d-11ef-ae41-7f16629231ae",
				"deleted":     "false",
			},

			"2024-05-16 03:22:17.995000",
			[]*account.Account{
				{
					ID:         uuid.FromStringOrNil("5227c0ce-198d-11ef-b06c-a744dbfcde74"),
					CustomerID: uuid.FromStringOrNil("52642a32-198d-11ef-ae41-7f16629231ae"),
					TMCreate:   "2024-05-16 03:22:17.995000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("529e6936-198d-11ef-afd6-57ccd5f20689"),
					CustomerID: uuid.FromStringOrNil("52642a32-198d-11ef-ae41-7f16629231ae"),
					TMCreate:   "2024-05-16 03:22:17.995000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
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
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for _, account := range tt.accounts {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().AccountSet(ctx, gomock.Any())
				if err := h.AccountCreate(ctx, &account); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.AccountGets(ctx, utilhandler.TimeGetCurTime(), tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_AccountIncreaseFile(t *testing.T) {

	tests := []struct {
		name    string
		account *account.Account

		id        uuid.UUID
		filecount int64
		filesize  int64

		responseCurTime string
		expectRes       *account.Account
	}{
		{
			name: "normal",
			account: &account.Account{
				ID:         uuid.FromStringOrNil("e7f3e69e-198f-11ef-9c52-47220c3b173b"),
				CustomerID: uuid.FromStringOrNil("7e4d9caa-198d-11ef-a42b-abbbe058dea6"),

				TotalFileCount: 1,
				TotalFileSize:  1024,
			},

			id:        uuid.FromStringOrNil("e7f3e69e-198f-11ef-9c52-47220c3b173b"),
			filecount: 1,
			filesize:  1024,

			responseCurTime: "2024-05-18 03:22:17.995000",
			expectRes: &account.Account{
				ID:         uuid.FromStringOrNil("e7f3e69e-198f-11ef-9c52-47220c3b173b"),
				CustomerID: uuid.FromStringOrNil("7e4d9caa-198d-11ef-a42b-abbbe058dea6"),

				TotalFileCount: 2,
				TotalFileSize:  2048,

				TMCreate: "2024-05-18 03:22:17.995000",
				TMUpdate: "2024-05-18 03:22:17.995000",
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
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if errIncrease := h.AccountIncreaseFile(ctx, tt.id, tt.filecount, tt.filesize); errIncrease != nil {
				t.Errorf("Wrong match. expect: ok, got: got: %v", errIncrease)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.account.ID).Return(nil, fmt.Errorf(""))
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

func Test_AccountDelete(t *testing.T) {

	tests := []struct {
		name    string
		account *account.Account

		id uuid.UUID

		responseCurTime string
		expectRes       *account.Account
	}{
		{
			name: "normal",
			account: &account.Account{
				ID:         uuid.FromStringOrNil("874be486-198f-11ef-9772-675746b370f6"),
				CustomerID: uuid.FromStringOrNil("7e4d9caa-198d-11ef-a42b-abbbe058dea6"),
			},

			id: uuid.FromStringOrNil("874be486-198f-11ef-9772-675746b370f6"),

			responseCurTime: "2024-05-18 03:22:17.995000",
			expectRes: &account.Account{
				ID:         uuid.FromStringOrNil("874be486-198f-11ef-9772-675746b370f6"),
				CustomerID: uuid.FromStringOrNil("7e4d9caa-198d-11ef-a42b-abbbe058dea6"),

				TMCreate: "2024-05-18 03:22:17.995000",
				TMUpdate: "2024-05-18 03:22:17.995000",
				TMDelete: "2024-05-18 03:22:17.995000",
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
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if errDelete := h.AccountDelete(ctx, tt.id); errDelete != nil {
				t.Errorf("Wrong match. expect: ok, got: got: %v", errDelete)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.account.ID).Return(nil, fmt.Errorf(""))
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
