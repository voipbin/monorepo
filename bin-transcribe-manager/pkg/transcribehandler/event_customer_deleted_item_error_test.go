package transcribehandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
)

// Test_EventCUCustomerDeleted_itemDeleteFails checks that a transcribe that cannot be deleted
// does not stop the cleanup: the next transcribe of the customer is still handled.
func Test_EventCUCustomerDeleted_itemDeleteFails(t *testing.T) {
	customerID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000601")
	firstID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000602")
	secondID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000603")
	deletedAt := time.Date(2023, 1, 3, 21, 35, 2, 0, time.UTC)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &transcribeHandler{db: mockDB}
	ctx := context.Background()

	first := &transcribe.Transcribe{Identity: commonidentity.Identity{ID: firstID, CustomerID: customerID}}
	// a transcribe that is deleted already is returned as it is by Delete
	second := &transcribe.Transcribe{Identity: commonidentity.Identity{ID: secondID, CustomerID: customerID}, TMDelete: &deletedAt}

	filters := map[transcribe.Field]any{
		transcribe.FieldCustomerID: customerID,
		transcribe.FieldDeleted:    false,
	}
	mockDB.EXPECT().TranscribeList(ctx, uint64(1000), "", filters).Return([]*transcribe.Transcribe{first, second}, nil)
	mockDB.EXPECT().TranscribeGet(ctx, firstID).Return(nil, stderrors.New("lookup failed"))
	mockDB.EXPECT().TranscribeGet(ctx, secondID).Return(second, nil)

	cu := &cucustomer.Customer{ID: customerID}
	if err := h.EventCUCustomerDeleted(ctx, cu); err != nil {
		t.Errorf("Wrong match. expect: ok (item failures are skipped), got: %v", err)
	}
}
