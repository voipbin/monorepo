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

func Test_GetCurrentCycle(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID

		responseAllowance *allowance.Allowance
		responseErr       error

		expectErr bool
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("a1000001-0000-0000-0000-000000000001"),

			responseAllowance: &allowance.Allowance{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1000002-0000-0000-0000-000000000001"),
				},
				AccountID:   uuid.FromStringOrNil("a1000001-0000-0000-0000-000000000001"),
				TokensTotal: 1000,
				TokensUsed:  200,
			},
			responseErr: nil,

			expectErr: false,
		},
		{
			name: "not found",

			accountID: uuid.FromStringOrNil("a1000001-0000-0000-0000-000000000002"),

			responseAllowance: nil,
			responseErr:       dbhandler.ErrNotFound,

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &allowanceHandler{
				db:          mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, tt.accountID).Return(tt.responseAllowance, tt.responseErr)

			res, err := h.GetCurrentCycle(ctx, tt.accountID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
				return
			}

			if res.ID != tt.responseAllowance.ID {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.responseAllowance.ID, res.ID)
			}
		})
	}
}

func Test_EnsureCurrentCycle_existing(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &allowanceHandler{
		db:          mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("b1000001-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("b1000002-0000-0000-0000-000000000001")

	existingAllowance := &allowance.Allowance{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("b1000003-0000-0000-0000-000000000001"),
			CustomerID: customerID,
		},
		AccountID:   accountID,
		TokensTotal: 1000,
		TokensUsed:  500,
	}

	// existing cycle found
	mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, accountID).Return(existingAllowance, nil)

	res, err := h.EnsureCurrentCycle(ctx, accountID, customerID, account.PlanTypeFree)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}

	if res.ID != existingAllowance.ID {
		t.Errorf("Wrong match. expect: %s, got: %s", existingAllowance.ID, res.ID)
	}
}

func Test_EnsureCurrentCycle_create_new(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &allowanceHandler{
		db:          mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("c1000001-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("c1000002-0000-0000-0000-000000000001")
	newID := uuid.FromStringOrNil("c1000003-0000-0000-0000-000000000001")
	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)

	// no existing cycle
	mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, accountID).Return(nil, dbhandler.ErrNotFound)

	// create new
	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newID)
	mockDB.EXPECT().AllowanceCreate(ctx, gomock.Any()).Return(nil)

	// re-read after create
	createdAllowance := &allowance.Allowance{
		Identity: commonidentity.Identity{
			ID:         newID,
			CustomerID: customerID,
		},
		AccountID:   accountID,
		TokensTotal: 1000, // PlanTypeFree
		TokensUsed:  0,
	}
	mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, accountID).Return(createdAllowance, nil)

	res, err := h.EnsureCurrentCycle(ctx, accountID, customerID, account.PlanTypeFree)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}

	if res.ID != newID {
		t.Errorf("Wrong match. expect: %s, got: %s", newID, res.ID)
	}
}

func Test_ConsumeTokens(t *testing.T) {

	type test struct {
		name string

		accountID    uuid.UUID
		tokensNeeded int
		creditPerUnit float32
		tokenPerUnit  int

		responseCycle         *allowance.Allowance
		responseTokensConsumed int
		responseCreditCharged  float32

		expectTokensConsumed int
		expectCreditCharged  float32
		expectErr            bool
	}

	tests := []test{
		{
			name: "case A - enough tokens",

			accountID:    uuid.FromStringOrNil("d1000001-0000-0000-0000-000000000001"),
			tokensNeeded: 5,
			creditPerUnit: 0.0045,
			tokenPerUnit:  1,

			responseCycle: &allowance.Allowance{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1000002-0000-0000-0000-000000000001"),
				},
				AccountID:   uuid.FromStringOrNil("d1000001-0000-0000-0000-000000000001"),
				TokensTotal: 1000,
				TokensUsed:  100,
			},
			responseTokensConsumed: 5,
			responseCreditCharged:  0,

			expectTokensConsumed: 5,
			expectCreditCharged:  0,
			expectErr:            false,
		},
		{
			name: "case B - partial tokens, credit overflow",

			accountID:    uuid.FromStringOrNil("d1000001-0000-0000-0000-000000000002"),
			tokensNeeded: 10,
			creditPerUnit: 0.0045,
			tokenPerUnit:  1,

			responseCycle: &allowance.Allowance{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1000002-0000-0000-0000-000000000002"),
				},
				AccountID:   uuid.FromStringOrNil("d1000001-0000-0000-0000-000000000002"),
				TokensTotal: 1000,
				TokensUsed:  995,
			},
			responseTokensConsumed: 5,
			responseCreditCharged:  0.0225, // 5 overflow units * 0.0045

			expectTokensConsumed: 5,
			expectCreditCharged:  0.0225,
			expectErr:            false,
		},
		{
			name: "case C - no tokens, all credit",

			accountID:    uuid.FromStringOrNil("d1000001-0000-0000-0000-000000000003"),
			tokensNeeded: 10,
			creditPerUnit: 0.008,
			tokenPerUnit:  10,

			responseCycle: &allowance.Allowance{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1000002-0000-0000-0000-000000000003"),
				},
				AccountID:   uuid.FromStringOrNil("d1000001-0000-0000-0000-000000000003"),
				TokensTotal: 1000,
				TokensUsed:  1000,
			},
			responseTokensConsumed: 0,
			responseCreditCharged:  0.008,

			expectTokensConsumed: 0,
			expectCreditCharged:  0.008,
			expectErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &allowanceHandler{
				db:          mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, tt.accountID).Return(tt.responseCycle, nil)
			mockDB.EXPECT().AllowanceConsumeTokens(ctx, tt.responseCycle.ID, tt.accountID, tt.tokensNeeded, tt.creditPerUnit, tt.tokenPerUnit).Return(tt.responseTokensConsumed, tt.responseCreditCharged, nil)

			tokensConsumed, creditCharged, err := h.ConsumeTokens(ctx, tt.accountID, tt.tokensNeeded, tt.creditPerUnit, tt.tokenPerUnit)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
				return
			}

			if tokensConsumed != tt.expectTokensConsumed {
				t.Errorf("Wrong tokensConsumed. expect: %d, got: %d", tt.expectTokensConsumed, tokensConsumed)
			}
			if creditCharged != tt.expectCreditCharged {
				t.Errorf("Wrong creditCharged. expect: %f, got: %f", tt.expectCreditCharged, creditCharged)
			}
		})
	}
}

func Test_ConsumeTokens_no_cycle(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &allowanceHandler{
		db:          mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("e1000001-0000-0000-0000-000000000001")

	mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, accountID).Return(nil, dbhandler.ErrNotFound)

	_, _, err := h.ConsumeTokens(ctx, accountID, 5, 0.0045, 1)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_computeCycleDates(t *testing.T) {

	tests := []struct {
		name string

		now time.Time

		expectStart time.Time
		expectEnd   time.Time
	}{
		{
			name: "middle of month",

			now: time.Date(2026, 2, 14, 15, 30, 0, 0, time.UTC),

			expectStart: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			expectEnd:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "first of month",

			now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),

			expectStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			expectEnd:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "last of month",

			now: time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),

			expectStart: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
			expectEnd:   time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "february non-leap year",

			now: time.Date(2027, 2, 28, 12, 0, 0, 0, time.UTC),

			expectStart: time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC),
			expectEnd:   time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := computeCycleDates(tt.now)
			if !start.Equal(tt.expectStart) {
				t.Errorf("Wrong cycleStart. expect: %v, got: %v", tt.expectStart, start)
			}
			if !end.Equal(tt.expectEnd) {
				t.Errorf("Wrong cycleEnd. expect: %v, got: %v", tt.expectEnd, end)
			}
		})
	}
}

func Test_AddTokens(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID
		amount    int

		responseCycle     *allowance.Allowance
		responseCycleErr  error
		responseUpdated   *allowance.Allowance
		responseUpdateErr error
		responseGetErr    error

		expectErr bool
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000001"),
			amount:    500,

			responseCycle: &allowance.Allowance{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa000002-0000-0000-0000-000000000001"),
				},
				AccountID:   uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000001"),
				TokensTotal: 1000,
				TokensUsed:  200,
			},
			responseCycleErr:  nil,
			responseUpdateErr: nil,
			responseUpdated: &allowance.Allowance{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa000002-0000-0000-0000-000000000001"),
				},
				AccountID:   uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000001"),
				TokensTotal: 1500,
				TokensUsed:  200,
			},
			responseGetErr: nil,

			expectErr: false,
		},
		{
			name: "no current cycle",

			accountID: uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000002"),
			amount:    500,

			responseCycle:    nil,
			responseCycleErr: dbhandler.ErrNotFound,

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &allowanceHandler{
				db:          mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, tt.accountID).Return(tt.responseCycle, tt.responseCycleErr)

			if tt.responseCycleErr == nil {
				mockDB.EXPECT().AllowanceUpdate(ctx, tt.responseCycle.ID, map[allowance.Field]any{
					allowance.FieldTokensTotal: tt.responseCycle.TokensTotal + tt.amount,
				}).Return(tt.responseUpdateErr)

				if tt.responseUpdateErr == nil {
					mockDB.EXPECT().AllowanceGet(ctx, tt.responseCycle.ID).Return(tt.responseUpdated, tt.responseGetErr)
				}
			}

			res, err := h.AddTokens(ctx, tt.accountID, tt.amount)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
				return
			}

			if res.TokensTotal != tt.responseUpdated.TokensTotal {
				t.Errorf("Wrong TokensTotal. expect: %d, got: %d", tt.responseUpdated.TokensTotal, res.TokensTotal)
			}
		})
	}
}

func Test_SubtractTokens(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID
		amount    int

		responseCycle     *allowance.Allowance
		responseCycleErr  error
		responseUpdated   *allowance.Allowance
		responseUpdateErr error
		responseGetErr    error

		expectErr bool
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("ab000001-0000-0000-0000-000000000001"),
			amount:    300,

			responseCycle: &allowance.Allowance{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab000002-0000-0000-0000-000000000001"),
				},
				AccountID:   uuid.FromStringOrNil("ab000001-0000-0000-0000-000000000001"),
				TokensTotal: 1000,
				TokensUsed:  200,
			},
			responseCycleErr:  nil,
			responseUpdateErr: nil,
			responseUpdated: &allowance.Allowance{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab000002-0000-0000-0000-000000000001"),
				},
				AccountID:   uuid.FromStringOrNil("ab000001-0000-0000-0000-000000000001"),
				TokensTotal: 700,
				TokensUsed:  200,
			},
			responseGetErr: nil,

			expectErr: false,
		},
		{
			name: "would go negative",

			accountID: uuid.FromStringOrNil("ab000001-0000-0000-0000-000000000002"),
			amount:    1500,

			responseCycle: &allowance.Allowance{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab000002-0000-0000-0000-000000000002"),
				},
				AccountID:   uuid.FromStringOrNil("ab000001-0000-0000-0000-000000000002"),
				TokensTotal: 1000,
				TokensUsed:  200,
			},
			responseCycleErr: nil,

			expectErr: true,
		},
		{
			name: "no current cycle",

			accountID: uuid.FromStringOrNil("ab000001-0000-0000-0000-000000000003"),
			amount:    300,

			responseCycle:    nil,
			responseCycleErr: dbhandler.ErrNotFound,

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &allowanceHandler{
				db:          mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().AllowanceGetCurrentByAccountID(ctx, tt.accountID).Return(tt.responseCycle, tt.responseCycleErr)

			if tt.responseCycleErr == nil {
				newTotal := tt.responseCycle.TokensTotal - tt.amount
				if newTotal >= 0 {
					mockDB.EXPECT().AllowanceUpdate(ctx, tt.responseCycle.ID, map[allowance.Field]any{
						allowance.FieldTokensTotal: newTotal,
					}).Return(tt.responseUpdateErr)

					if tt.responseUpdateErr == nil {
						mockDB.EXPECT().AllowanceGet(ctx, tt.responseCycle.ID).Return(tt.responseUpdated, tt.responseGetErr)
					}
				}
			}

			res, err := h.SubtractTokens(ctx, tt.accountID, tt.amount)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
				return
			}

			if res.TokensTotal != tt.responseUpdated.TokensTotal {
				t.Errorf("Wrong TokensTotal. expect: %d, got: %d", tt.responseUpdated.TokensTotal, res.TokensTotal)
			}
		})
	}
}

func Test_ListByAccountID(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &allowanceHandler{
		db:          mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("f1000001-0000-0000-0000-000000000001")

	expected := []*allowance.Allowance{
		{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("f1000002-0000-0000-0000-000000000001"),
			},
			AccountID: accountID,
		},
	}

	filters := map[allowance.Field]any{
		allowance.FieldAccountID: accountID,
		allowance.FieldDeleted:   false,
	}
	mockDB.EXPECT().AllowanceList(ctx, uint64(10), "", filters).Return(expected, nil)

	res, err := h.ListByAccountID(ctx, accountID, 10, "")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}

	if len(res) != 1 {
		t.Errorf("Wrong result count. expect: 1, got: %d", len(res))
	}
}
