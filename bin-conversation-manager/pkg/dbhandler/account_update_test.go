package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

func Test_AccountUpdate(t *testing.T) {
	createTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)
	updateTime := time.Date(2020, 4, 19, 4, 23, 18, 996000000, time.UTC)

	tests := []struct {
		name string

		account *account.Account
		fields  map[account.Field]any

		expectRes *account.Account
	}{
		{
			name: "updates the given columns and stamps tm_update",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000a0001"),
					CustomerID: uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000c0001"),
				},
				Type:   account.TypeLine,
				Name:   "before name",
				Detail: "before detail",
				Secret: "before secret",
				Token:  "before token",
			},
			fields: map[account.Field]any{
				account.FieldName:   "after name",
				account.FieldDetail: "after detail",
				account.FieldToken:  "after token",
			},

			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000a0001"),
					CustomerID: uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000c0001"),
				},
				Type:   account.TypeLine,
				Name:   "after name",
				Detail: "after detail",
				Secret: "before secret",
				Token:  "after token",

				TMCreate: &createTime,
				TMUpdate: &updateTime,
			},
		},
		{
			name: "uuid column is stored and read back",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000a0002"),
					CustomerID: uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000c0002"),
				},
				Type: account.TypeLine,
				Name: "flow account",
			},
			fields: map[account.Field]any{
				account.FieldMessageFlowID: uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000f0002"),
			},

			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000a0002"),
					CustomerID: uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000c0002"),
				},
				Type:          account.TypeLine,
				Name:          "flow account",
				MessageFlowID: uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000f0002"),

				TMCreate: &createTime,
				TMUpdate: &updateTime,
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

			mockUtil.EXPECT().TimeNow().Return(&createTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			// the update stamps tm_update and refreshes the cache from the stored row.
			mockUtil.EXPECT().TimeNow().Return(&updateTime)
			mockCache.EXPECT().AccountSet(ctx, tt.expectRes).Return(nil)
			if err := h.AccountUpdate(ctx, tt.account.ID, tt.fields); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.account.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.account.ID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccountUpdate_noFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no clock, cache or database interaction is expected for an empty update.
	h := handler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          dbTest,
		cache:       cachehandler.NewMockCacheHandler(mc),
	}

	if err := h.AccountUpdate(context.Background(), uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000a0003"), map[account.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_AccountUpdate_unknownColumn(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	updateTime := time.Date(2020, 4, 19, 4, 23, 18, 996000000, time.UTC)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	// the cache must not be touched when the statement fails.
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       cachehandler.NewMockCacheHandler(mc),
	}

	mockUtil.EXPECT().TimeNow().Return(&updateTime)
	err := h.AccountUpdate(context.Background(), uuid.FromStringOrNil("b1000000-0000-0000-0000-0000000a0004"), map[account.Field]any{
		account.Field("no_such_column"): "x",
	})
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}
