package accounthandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_TopUpDue(t *testing.T) {

	now := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	dueAccount := &account.Account{
		Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001"), CustomerID: customerID},
		PlanType:    account.PlanTypeFree,
		TmNextTopUp: &past,
		TMCreate:    &now,
	}
	notDueAccount := &account.Account{
		Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002"), CustomerID: customerID},
		PlanType:    account.PlanTypeFree,
		TmNextTopUp: &future,
		TMCreate:    &now,
	}
	casSkipAccount := &account.Account{
		Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000003"), CustomerID: customerID},
		PlanType:    account.PlanTypeFree,
		TmNextTopUp: &past,
		TMCreate:    &now,
	}
	errorAccount := &account.Account{
		Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000004"), CustomerID: customerID},
		PlanType:    account.PlanTypeFree,
		TmNextTopUp: &past,
		TMCreate:    &now,
	}
	noPlanAccount := &account.Account{
		Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000005"), CustomerID: customerID},
		PlanType:    account.PlanType("unknown"),
		TmNextTopUp: &past,
		TMCreate:    &now,
	}

	t.Run("pagination across two pages with due, not-due, cas-skip, per-row error, and unknown-plan accounts", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)

		h := &accountHandler{
			utilHandler: mockUtil,
			db:          mockDB,
		}
		ctx := context.Background()

		filters := map[account.Field]any{
			account.FieldDeleted: false,
		}

		mockUtil.EXPECT().TimeNow().Return(&now)

		// page 1: exactly 500 not-due accounts — a full page, forcing TopUpDue to
		// request a second page. None of these are due, so no AccountTopUpTokens
		// calls are expected for this page.
		page1 := make([]*account.Account, 500)
		for i := range page1 {
			page1[i] = &account.Account{
				Identity:    commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
				PlanType:    account.PlanTypeFree,
				TmNextTopUp: &future,
				TMCreate:    &now,
			}
		}
		mockDB.EXPECT().AccountList(ctx, uint64(500), "", filters).Return(page1, nil)

		// page 2: fewer than 500 rows, ends pagination. Exercises due, cas-skip,
		// per-row-error, and unknown-plan-type accounts.
		page2 := []*account.Account{dueAccount, notDueAccount, casSkipAccount, errorAccount, noPlanAccount}
		pageToken := now.Format(time.RFC3339Nano)
		mockDB.EXPECT().AccountList(ctx, uint64(500), pageToken, filters).Return(page2, nil)

		mockDB.EXPECT().AccountTopUpTokens(ctx, dueAccount.ID, customerID, int64(100), string(account.PlanTypeFree)).Return(true, nil)
		mockDB.EXPECT().AccountTopUpTokens(ctx, casSkipAccount.ID, customerID, int64(100), string(account.PlanTypeFree)).Return(false, nil)
		mockDB.EXPECT().AccountTopUpTokens(ctx, errorAccount.ID, customerID, int64(100), string(account.PlanTypeFree)).Return(false, fmt.Errorf("db error"))

		processed, failed, err := h.TopUpDue(ctx)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if processed != 1 {
			t.Errorf("expected processed=1, got: %d", processed)
		}
		if failed != 1 {
			t.Errorf("expected failed=1, got: %d", failed)
		}
	})

	t.Run("AccountList failure returns error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)

		h := &accountHandler{
			utilHandler: mockUtil,
			db:          mockDB,
		}
		ctx := context.Background()

		filters := map[account.Field]any{
			account.FieldDeleted: false,
		}

		mockUtil.EXPECT().TimeNow().Return(&now)
		mockDB.EXPECT().AccountList(ctx, uint64(500), "", filters).Return(nil, fmt.Errorf("list failed"))

		processed, failed, err := h.TopUpDue(ctx)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if processed != 0 || failed != 0 {
			t.Errorf("expected processed=0, failed=0, got processed=%d, failed=%d", processed, failed)
		}
	})

	t.Run("empty list returns zero counts", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)

		h := &accountHandler{
			utilHandler: mockUtil,
			db:          mockDB,
		}
		ctx := context.Background()

		filters := map[account.Field]any{
			account.FieldDeleted: false,
		}

		mockUtil.EXPECT().TimeNow().Return(&now)
		mockDB.EXPECT().AccountList(ctx, uint64(500), "", filters).Return(nil, nil)

		processed, failed, err := h.TopUpDue(ctx)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if processed != 0 || failed != 0 {
			t.Errorf("expected processed=0, failed=0, got processed=%d, failed=%d", processed, failed)
		}
	})
}
