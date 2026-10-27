package accounthandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

// Test_Create_storeWriteFails verifies that the stored record decides the outcome of a creation:
// when the write reports a failure but the record can be read back, the account is returned and announced.
func Test_Create_storeWriteFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := accountHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5a000001-0000-0000-0000-000000000001")
	accountID := uuid.FromStringOrNil("5a000002-0000-0000-0000-000000000001")
	stored := &account.Account{Identity: commonidentity.Identity{ID: accountID, CustomerID: customerID}, Name: "stored"}

	mockUtil.EXPECT().UUIDCreate().Return(accountID)
	mockDB.EXPECT().AccountCreate(ctx, gomock.Any()).Return(fmt.Errorf("write failed"))
	mockDB.EXPECT().AccountGet(ctx, accountID).Return(stored, nil)
	mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountCreated, stored)

	res, err := h.Create(ctx, customerID, "stored", "detail", account.PaymentTypePrepaid, account.PaymentMethodNone)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != stored {
		t.Errorf("Wrong match. expect the stored account, got: %+v", res)
	}
}
