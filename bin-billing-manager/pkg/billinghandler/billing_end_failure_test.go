package billinghandler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_NewBillingHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	res := NewBillingHandler(mockReq, mockDB, mockNotify, mockAccount)
	h, ok := res.(*billingHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *billingHandler, got: %T", res)
	}

	if h.utilHandler == nil || h.reqHandler != mockReq || h.db != mockDB || h.notifyHandler != mockNotify || h.accountHandler != mockAccount {
		t.Errorf("Wrong match. every dependency must be wired. got: %+v", h)
	}
}

// Test_BillingStart_immediateEndFails verifies that a failure while finalizing an immediate-end
// billing is reported to the caller after the billing record has been created.
func Test_BillingStart_immediateEndFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := billingHandler{
		utilHandler:    mockUtil,
		db:             mockDB,
		notifyHandler:  mockNotify,
		accountHandler: mockAccount,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5b000001-0000-0000-0000-000000000001")
	accountID := uuid.FromStringOrNil("5b000002-0000-0000-0000-000000000001")
	referenceID := uuid.FromStringOrNil("5b000003-0000-0000-0000-000000000001")
	billingID := uuid.FromStringOrNil("5b000004-0000-0000-0000-000000000001")
	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 0, time.UTC)

	created := &billing.Billing{
		Identity:      commonidentity.Identity{ID: billingID, CustomerID: customerID},
		AccountID:     accountID,
		Status:        billing.StatusProgressing,
		ReferenceType: billing.ReferenceTypeSMS,
		ReferenceID:   referenceID,
		CostType:      billing.CostTypeSMS,
	}

	mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeSMS, referenceID).Return(nil, dbhandler.ErrNotFound)
	mockAccount.EXPECT().GetByCustomerID(ctx, customerID).Return(&account.Account{Identity: commonidentity.Identity{ID: accountID, CustomerID: customerID}}, nil)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)
	mockDB.EXPECT().BillingCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().BillingGet(ctx, billingID).Return(created, nil)
	mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, created)

	// the atomic consume-and-record step fails, so no update event may be published
	mockDB.EXPECT().BillingConsumeAndRecord(ctx, created, accountID, 1, 0, billing.GetCostInfo(billing.CostTypeSMS), &tmBillingStart).Return(nil, fmt.Errorf("store unavailable"))

	err := h.BillingStart(ctx, customerID, billing.ReferenceTypeSMS, referenceID, billing.CostTypeSMS, &tmBillingStart, &commonaddress.Address{Target: "+10000000001"}, &commonaddress.Address{Target: "+10000000002"})
	if err == nil || !strings.Contains(err.Error(), "could not end the billing") {
		t.Errorf("Wrong match. expect the end failure, got: %v", err)
	}
}
