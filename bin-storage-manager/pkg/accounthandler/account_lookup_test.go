package accounthandler

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-storage-manager/models/account"
)

func Test_Create_lookupFailureIsReturned(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("d41e2a10-0000-0000-0000-000000000001")
	filters := map[account.Field]any{
		account.FieldDeleted:    false,
		account.FieldCustomerID: customerID,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockDB, _ := newTestAccountHandler(mc)

	// nothing else may be called: no id, no insert, no event.
	mockDB.EXPECT().AccountList(ctx, "", uint64(1), filters).Return(nil, errTestDB)

	res, err := h.Create(ctx, customerID)
	if !stderrors.Is(err, errTestDB) || res != nil {
		t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTestDB, res, err)
	}
}

func Test_ValidateFileInfoByCustomerID_missingAccountIsAnError(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("d41e2a10-0000-0000-0000-000000000002")
	filters := map[account.Field]any{
		account.FieldDeleted:    false,
		account.FieldCustomerID: customerID,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockDB, _ := newTestAccountHandler(mc)
	mockDB.EXPECT().AccountList(ctx, "", uint64(1), filters).Return([]*account.Account{}, nil)

	res, err := h.ValidateFileInfoByCustomerID(ctx, customerID, 1, 1)
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: nil/error, got: %v/%v", res, err)
	}
}
