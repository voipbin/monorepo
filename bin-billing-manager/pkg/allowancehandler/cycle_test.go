package allowancehandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/allowance"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_ProcessAllCycles(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &allowanceHandler{
		db:          mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	tmCreate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	freeAccount := &account.Account{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("aa000002-0000-0000-0000-000000000001"),
		},
		TMCreate: &tmCreate,
	}

	freeFilters := map[account.Field]any{
		account.FieldPlanType: account.PlanTypeFree,
		account.FieldDeleted:  false,
	}
	basicFilters := map[account.Field]any{
		account.FieldPlanType: account.PlanTypeBasic,
		account.FieldDeleted:  false,
	}
	proFilters := map[account.Field]any{
		account.FieldPlanType: account.PlanTypeProfessional,
		account.FieldDeleted:  false,
	}

	// Free plan: returns 1 account, then empty page
	mockDB.EXPECT().AccountList(ctx, uint64(100), "", freeFilters).Return([]*account.Account{freeAccount}, nil)

	// EnsureCurrentCycle for the free account — existing cycle found
	existingAllowance := &allowance.Allowance{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("aa000003-0000-0000-0000-000000000001"),
		},
		AccountID: freeAccount.ID,
	}
	mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, freeAccount.ID).Return(existingAllowance, nil)

	// next page for free: empty
	mockDB.EXPECT().AccountList(ctx, uint64(100), tmCreate.Format(utilhandler.ISO8601Layout), freeFilters).Return([]*account.Account{}, nil)

	// Basic plan: empty
	mockDB.EXPECT().AccountList(ctx, uint64(100), "", basicFilters).Return([]*account.Account{}, nil)

	// Professional plan: empty
	mockDB.EXPECT().AccountList(ctx, uint64(100), "", proFilters).Return([]*account.Account{}, nil)

	_ = now // referenced above via time.Date

	err := h.ProcessAllCycles(ctx)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_processAccountsForPlan_creates_new_cycle(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &allowanceHandler{
		db:          mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	tmCreate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	newID := uuid.FromStringOrNil("bb000001-0000-0000-0000-000000000001")

	acc := &account.Account{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("bb000002-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("bb000003-0000-0000-0000-000000000001"),
		},
		TMCreate: &tmCreate,
	}

	filters := map[account.Field]any{
		account.FieldPlanType: account.PlanTypeBasic,
		account.FieldDeleted:  false,
	}

	// returns 1 account
	mockDB.EXPECT().AccountList(ctx, uint64(100), "", filters).Return([]*account.Account{acc}, nil)

	// EnsureCurrentCycle: no existing cycle → create
	mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, acc.ID).Return(nil, dbhandler.ErrNotFound)
	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newID)
	mockDB.EXPECT().AllowanceCreate(ctx, gomock.Any()).Return(nil)

	// re-read after create
	createdAllowance := &allowance.Allowance{
		Identity: commonidentity.Identity{
			ID:         newID,
			CustomerID: acc.CustomerID,
		},
		AccountID:   acc.ID,
		TokensTotal: 10000, // PlanTypeBasic
		TokensUsed:  0,
	}
	mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, acc.ID).Return(createdAllowance, nil)

	// next page: empty
	mockDB.EXPECT().AccountList(ctx, uint64(100), tmCreate.Format(utilhandler.ISO8601Layout), filters).Return([]*account.Account{}, nil)

	err := h.processAccountsForPlan(ctx, account.PlanTypeBasic)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
