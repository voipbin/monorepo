package contacthandler

// Regression tests for A9-b: a soft-deleted Contact must not accept new
// address writes. interactionListByContact (interaction_read.go) has
// always treated a soft-deleted Contact (TMDelete != nil) as not-found;
// AddAddress/UpdateAddress/RemoveAddress/ClaimAddress never adopted the
// same check, so a customer could keep registering/editing/removing/
// claiming addresses against a Contact that had already been deleted.

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func deletedTestContact(contactID, customerID uuid.UUID) *contact.Contact {
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		TMDelete: &deletedAt,
	}
}

func assertNotFound(t *testing.T, err error, method string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s() expected error for soft-deleted contact, got nil", method)
	}
	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("%s() error = %v, want *cerrors.VoipbinError", method, err)
	}
}

// Test_AddAddress_DeletedContact_NotFound verifies AddAddress rejects a
// write against a soft-deleted Contact instead of silently succeeding.
func Test_AddAddress_DeletedContact_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := contactHandler{db: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-9000-9000-9000-000000000001")
	contactID := uuid.FromStringOrNil("bbbbbbbb-9000-9000-9000-000000000001")

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(deletedTestContact(contactID, customerID), nil)

	_, err := h.AddAddress(ctx, contactID, &contact.Address{
		Address: commonaddress.Address{Type: contact.AddressTypeTel, Target: "+15551230001"},
	})
	assertNotFound(t, err, "AddAddress")
}

// Test_UpdateAddress_DeletedContact_NotFound verifies UpdateAddress
// rejects a write against a soft-deleted Contact.
func Test_UpdateAddress_DeletedContact_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := contactHandler{db: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-9000-9000-9000-000000000002")
	contactID := uuid.FromStringOrNil("bbbbbbbb-9000-9000-9000-000000000002")
	addressID := uuid.FromStringOrNil("cccccccc-9000-9000-9000-000000000002")

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(deletedTestContact(contactID, customerID), nil)

	_, err := h.UpdateAddress(ctx, contactID, addressID, map[string]any{"name": "updated"})
	assertNotFound(t, err, "UpdateAddress")
}

// Test_RemoveAddress_DeletedContact_NotFound verifies RemoveAddress
// rejects a write against a soft-deleted Contact.
func Test_RemoveAddress_DeletedContact_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := contactHandler{db: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-9000-9000-9000-000000000003")
	contactID := uuid.FromStringOrNil("bbbbbbbb-9000-9000-9000-000000000003")
	addressID := uuid.FromStringOrNil("cccccccc-9000-9000-9000-000000000003")

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(deletedTestContact(contactID, customerID), nil)

	_, err := h.RemoveAddress(ctx, contactID, addressID)
	assertNotFound(t, err, "RemoveAddress")
}

// Test_ClaimAddress_DeletedContact_NotFound verifies ClaimAddress
// rejects a claim against a soft-deleted Contact, instead of letting
// AddressClaim proceed and silently reattach an address to a Contact
// that no longer exists from the customer's perspective.
func Test_ClaimAddress_DeletedContact_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := contactHandler{db: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-9000-9000-9000-000000000004")
	contactID := uuid.FromStringOrNil("bbbbbbbb-9000-9000-9000-000000000004")
	addressID := uuid.FromStringOrNil("cccccccc-9000-9000-9000-000000000004")

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(deletedTestContact(contactID, customerID), nil)

	_, err := h.ClaimAddress(ctx, customerID, addressID, contactID)
	assertNotFound(t, err, "ClaimAddress")
}
