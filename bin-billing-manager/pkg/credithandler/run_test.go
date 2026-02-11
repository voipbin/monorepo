package credithandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_ProcessAll(t *testing.T) {

	type test struct {
		name string

		// mock responses per AccountList call (pages)
		accountPages [][]*account.Account
		accountErr   error // if set, first AccountList returns this error

		// per-account BillingCreditTopUp responses (indexed by account ID string)
		topUpResults map[string]struct {
			created bool
			err     error
		}

		expectErr bool
	}

	now := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)
	tmCreate1 := time.Date(2026, 2, 10, 8, 0, 0, 0, time.UTC)
	tmCreate2 := time.Date(2026, 2, 9, 7, 0, 0, 0, time.UTC)
	tmCreate3 := time.Date(2026, 2, 8, 6, 0, 0, 0, time.UTC)

	acc1 := &account.Account{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001"),
		},
		PlanType: account.PlanTypeFree,
		TMCreate: &tmCreate1,
	}
	acc2 := &account.Account{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000002"),
			CustomerID: uuid.FromStringOrNil("c2000000-0000-0000-0000-000000000002"),
		},
		PlanType: account.PlanTypeFree,
		TMCreate: &tmCreate2,
	}
	acc3 := &account.Account{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a3000000-0000-0000-0000-000000000003"),
			CustomerID: uuid.FromStringOrNil("c3000000-0000-0000-0000-000000000003"),
		},
		PlanType: account.PlanTypeFree,
		TMCreate: &tmCreate3,
	}

	tests := []test{
		{
			name: "empty - no accounts",

			accountPages: [][]*account.Account{
				{}, // empty page
			},

			expectErr: false,
		},
		{
			name: "single page - two accounts",

			accountPages: [][]*account.Account{
				{acc1, acc2},
				{}, // empty second page signals end
			},

			topUpResults: map[string]struct {
				created bool
				err     error
			}{
				acc1.ID.String(): {created: true, err: nil},
				acc2.ID.String(): {created: true, err: nil},
			},

			expectErr: false,
		},
		{
			name: "multiple pages",

			accountPages: [][]*account.Account{
				{acc1, acc2},
				{acc3},
				{}, // empty third page signals end
			},

			topUpResults: map[string]struct {
				created bool
				err     error
			}{
				acc1.ID.String(): {created: true, err: nil},
				acc2.ID.String(): {created: false, err: nil}, // duplicate
				acc3.ID.String(): {created: true, err: nil},
			},

			expectErr: false,
		},
		{
			name: "account list error",

			accountPages: nil,
			accountErr:   fmt.Errorf("database connection lost"),

			expectErr: true,
		},
		{
			name: "one account fails - continues to next",

			accountPages: [][]*account.Account{
				{acc1, acc2},
				{}, // empty
			},

			topUpResults: map[string]struct {
				created bool
				err     error
			}{
				acc1.ID.String(): {created: false, err: fmt.Errorf("timeout")},
				acc2.ID.String(): {created: true, err: nil},
			},

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &handler{
				db:          mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			expectedFilters := map[account.Field]any{
				account.FieldPlanType: account.PlanTypeFree,
			}

			if tt.accountErr != nil {
				// first AccountList call returns error
				mockDB.EXPECT().AccountList(ctx, uint64(100), "", expectedFilters).Return(nil, tt.accountErr)
			} else {
				// set up page expectations
				token := ""
				for _, page := range tt.accountPages {
					currentToken := token
					if len(page) > 0 {
						// next token is the last account's TMCreate
						token = page[len(page)-1].TMCreate.Format(utilhandler.ISO8601Layout)
					}

					mockDB.EXPECT().AccountList(ctx, uint64(100), currentToken, expectedFilters).Return(page, nil)

					// set up processAccount expectations for each account in this page
					for _, acc := range page {
						// processAccount calls: TimeNow (x2), NewV5UUID, UUIDCreate, BillingCreditTopUp
						mockUtil.EXPECT().TimeNow().Return(&now)
						mockUtil.EXPECT().NewV5UUID(uuid.Nil, acc.ID.String()+":"+now.Format("2006-01")).Return(uuid.NewV5(uuid.Nil, acc.ID.String()+":2026-02"))
						mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"))

						result := tt.topUpResults[acc.ID.String()]
						mockDB.EXPECT().BillingCreditTopUp(ctx, gomock.Any(), acc.ID, FreeTierCreditAmount).
							DoAndReturn(func(_ context.Context, b *billing.Billing, _ uuid.UUID, _ float32) (bool, error) {
								if b.ReferenceType != billing.ReferenceTypeCreditFreeTier {
									t.Errorf("wrong reference type. expect: %v, got: %v", billing.ReferenceTypeCreditFreeTier, b.ReferenceType)
								}
								return result.created, result.err
							})
					}

					// if this page is empty, don't expect more calls
					if len(page) == 0 {
						break
					}
				}
			}

			err := h.ProcessAll(ctx)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
