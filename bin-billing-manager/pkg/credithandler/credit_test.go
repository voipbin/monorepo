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

func Test_processAccount(t *testing.T) {

	type test struct {
		name string

		account *account.Account

		responseUUID      uuid.UUID
		responseTimeNow   *time.Time
		responseCreated   bool
		responseTopUpErr  error
		referenceIDExpect uuid.UUID

		expectErr bool
	}

	now := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)
	accountID := uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	customerID := uuid.FromStringOrNil("c1d2e3f4-a5b6-7890-abcd-ef1234567890")
	billingID := uuid.FromStringOrNil("b1b2b3b4-b5b6-7890-abcd-ef1234567890")

	// deterministic reference ID for the test account + month
	expectedRefID := uuid.NewV5(uuid.Nil, accountID.String()+":2026-02")

	tests := []test{
		{
			name: "normal - credit created",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID:         accountID,
					CustomerID: customerID,
				},
			},

			responseUUID:      billingID,
			responseTimeNow:   &now,
			responseCreated:   true,
			responseTopUpErr:  nil,
			referenceIDExpect: expectedRefID,

			expectErr: false,
		},
		{
			name: "already processed - duplicate key",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID:         accountID,
					CustomerID: customerID,
				},
			},

			responseUUID:      billingID,
			responseTimeNow:   &now,
			responseCreated:   false,
			responseTopUpErr:  nil,
			referenceIDExpect: expectedRefID,

			expectErr: false,
		},
		{
			name: "error - db failure",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID:         accountID,
					CustomerID: customerID,
				},
			},

			responseUUID:      billingID,
			responseTimeNow:   &now,
			responseCreated:   false,
			responseTopUpErr:  fmt.Errorf("connection refused"),
			referenceIDExpect: expectedRefID,

			expectErr: true,
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

			// TimeNow is called twice: once for currentYearMonth, once for `now`
			mockUtil.EXPECT().TimeNow().Return(tt.responseTimeNow).Times(2)
			mockUtil.EXPECT().NewV5UUID(uuid.Nil, tt.account.ID.String()+":"+tt.responseTimeNow.Format("2006-01")).Return(tt.referenceIDExpect)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)

			mockDB.EXPECT().BillingCreditTopUp(
				ctx,
				gomock.Any(), // billing struct
				tt.account.ID,
				FreeTierCreditAmount,
			).DoAndReturn(func(_ context.Context, b *billing.Billing, _ uuid.UUID, _ float32) (bool, error) {
				// Verify the billing struct fields
				if b.ID != tt.responseUUID {
					t.Errorf("billing ID mismatch. expect: %v, got: %v", tt.responseUUID, b.ID)
				}
				if b.CustomerID != tt.account.CustomerID {
					t.Errorf("billing CustomerID mismatch. expect: %v, got: %v", tt.account.CustomerID, b.CustomerID)
				}
				if b.AccountID != tt.account.ID {
					t.Errorf("billing AccountID mismatch. expect: %v, got: %v", tt.account.ID, b.AccountID)
				}
				if b.ReferenceType != billing.ReferenceTypeCreditFreeTier {
					t.Errorf("billing ReferenceType mismatch. expect: %v, got: %v", billing.ReferenceTypeCreditFreeTier, b.ReferenceType)
				}
				if b.ReferenceID != tt.referenceIDExpect {
					t.Errorf("billing ReferenceID mismatch. expect: %v, got: %v", tt.referenceIDExpect, b.ReferenceID)
				}
				if b.Status != billing.StatusEnd {
					t.Errorf("billing Status mismatch. expect: %v, got: %v", billing.StatusEnd, b.Status)
				}
				if b.CostPerUnit != 0 {
					t.Errorf("billing CostPerUnit mismatch. expect: 0, got: %v", b.CostPerUnit)
				}
				if b.CostTotal != 0 {
					t.Errorf("billing CostTotal mismatch. expect: 0, got: %v", b.CostTotal)
				}
				if b.BillingUnitCount != 1.0 {
					t.Errorf("billing BillingUnitCount mismatch. expect: 1.0, got: %v", b.BillingUnitCount)
				}
				return tt.responseCreated, tt.responseTopUpErr
			})

			err := h.processAccount(ctx, tt.account)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
