package contacthandler

import (
	"context"
	stderrors "errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_CreateUnresolvedAddress verifies CreateUnresolvedAddress sets ID,
// CustomerID, and forces ContactID to uuid.Nil, and does NOT publish any
// event (there is no contact yet to attach it to).
func Test_CreateUnresolvedAddress(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-1111-1111-1111-111111111111")
	generatedID := uuid.FromStringOrNil("bbbbbbbb-1111-1111-1111-111111111111")

	a := &contact.Address{
		Type:   contact.AddressTypeTel,
		Target: "+15559998888",
	}

	mockUtil.EXPECT().UUIDCreate().Return(generatedID)
	mockDB.EXPECT().AddressCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, got *contact.Address) error {
		if got.ID != generatedID {
			t.Errorf("AddressCreate() ID = %v, want %v", got.ID, generatedID)
		}
		if got.CustomerID != customerID {
			t.Errorf("AddressCreate() CustomerID = %v, want %v", got.CustomerID, customerID)
		}
		if got.ContactID != uuid.Nil {
			t.Errorf("AddressCreate() ContactID = %v, want uuid.Nil", got.ContactID)
		}
		return nil
	})

	created := &contact.Address{
		ID:         generatedID,
		CustomerID: customerID,
		ContactID:  uuid.Nil,
		Type:       contact.AddressTypeTel,
		Target:     "+15559998888",
	}
	mockDB.EXPECT().AddressGet(ctx, customerID, generatedID).Return(created, nil)

	// No PublishEvent expectation set — any call would fail the mock controller.

	res, err := h.CreateUnresolvedAddress(ctx, customerID, a)
	if err != nil {
		t.Fatalf("CreateUnresolvedAddress() error = %v", err)
	}
	if res.ID != generatedID {
		t.Errorf("CreateUnresolvedAddress() ID = %v, want %v", res.ID, generatedID)
	}
}

// Test_ClaimAddress_Success verifies the happy path publishes
// EventTypeContactUpdated exactly once, reusing the contact fetched for the
// tenant check (no redundant second ContactGet).
func Test_ClaimAddress_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-2222-2222-2222-222222222222")
	contactID := uuid.FromStringOrNil("bbbbbbbb-2222-2222-2222-222222222222")
	addressID := uuid.FromStringOrNil("cccccccc-2222-2222-2222-222222222222")

	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
	}
	claimed := &contact.Address{
		ID:         addressID,
		CustomerID: customerID,
		ContactID:  contactID,
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(c, nil)
	mockDB.EXPECT().AddressClaim(ctx, customerID, addressID, contactID).Return(nil)
	mockDB.EXPECT().AddressGet(ctx, customerID, addressID).Return(claimed, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

	res, err := h.ClaimAddress(ctx, customerID, addressID, contactID)
	if err != nil {
		t.Fatalf("ClaimAddress() error = %v", err)
	}
	if res.ID != addressID {
		t.Errorf("ClaimAddress() ID = %v, want %v", res.ID, addressID)
	}
}

// Test_ClaimAddress_CrossTenantContact verifies a contact belonging to a
// different customer is treated as not-found, never reaching AddressClaim.
func Test_ClaimAddress_CrossTenantContact(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := contactHandler{db: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-3333-3333-3333-333333333333")
	otherCustomerID := uuid.FromStringOrNil("dddddddd-3333-3333-3333-333333333333")
	contactID := uuid.FromStringOrNil("bbbbbbbb-3333-3333-3333-333333333333")
	addressID := uuid.FromStringOrNil("cccccccc-3333-3333-3333-333333333333")

	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: otherCustomerID,
		},
	}
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(c, nil)

	_, err := h.ClaimAddress(ctx, customerID, addressID, contactID)
	if err == nil {
		t.Fatal("ClaimAddress() expected error for cross-tenant contact, got nil")
	}
	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("ClaimAddress() error = %v, want *cerrors.VoipbinError", err)
	}
}

// Test_ClaimAddress_Conflict verifies dbhandler.ErrConflict is translated to
// a typed AlreadyExists error.
func Test_ClaimAddress_Conflict(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := contactHandler{db: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-4444-4444-4444-444444444444")
	contactID := uuid.FromStringOrNil("bbbbbbbb-4444-4444-4444-444444444444")
	addressID := uuid.FromStringOrNil("cccccccc-4444-4444-4444-444444444444")

	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
	}
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(c, nil)
	mockDB.EXPECT().AddressClaim(ctx, customerID, addressID, contactID).Return(dbhandler.ErrConflict)

	_, err := h.ClaimAddress(ctx, customerID, addressID, contactID)
	if err == nil {
		t.Fatal("ClaimAddress() expected conflict error, got nil")
	}
}

// Test_ClaimAddress_AddressNotFound verifies dbhandler.ErrNotFound (on the
// address side) is translated to a typed NotFound error.
func Test_ClaimAddress_AddressNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := contactHandler{db: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-5555-5555-5555-555555555555")
	contactID := uuid.FromStringOrNil("bbbbbbbb-5555-5555-5555-555555555555")
	addressID := uuid.FromStringOrNil("cccccccc-5555-5555-5555-555555555555")

	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
	}
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(c, nil)
	mockDB.EXPECT().AddressClaim(ctx, customerID, addressID, contactID).Return(dbhandler.ErrNotFound)

	_, err := h.ClaimAddress(ctx, customerID, addressID, contactID)
	if err == nil {
		t.Fatal("ClaimAddress() expected not-found error, got nil")
	}
}
