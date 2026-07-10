package dbhandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_AddressMigration_CrossTypeSinglePrimary is a regression test for VOIP-1207.
//
// The unified contact_addresses table enforces one-primary-per-contact across
// BOTH tel and email types via UNIQUE(customer_id, primary_contact_uk). The
// handler path (AddressCreate with is_primary=true) should call AddressResetPrimary
// before inserting a new primary (this is done in contacthandler, not dbhandler).
// This test verifies the cross-type single-primary invariant end-to-end:
// after adding a primary tel address and resetting, then adding an email address,
// only the email should be primary.
func Test_AddressMigration_CrossTypeSinglePrimary(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()
	curTime := timePtr(time.Date(2021, 5, 19, 1, 2, 3, 0, time.UTC))

	contactID := uuid.FromStringOrNil("c1c1c1c1-0001-0001-0001-000000000001")
	customerID := uuid.FromStringOrNil("c1c1c1c1-0002-0002-0002-000000000002")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "Cross",
		LastName:  "Primary",
		Source:    "manual",
	}

	// Create contact.
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	// Handler path for adding a primary tel address:
	// reset existing primaries (no-op here, no existing addresses), then create.
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressResetPrimary(ctx, contactID); err != nil {
		t.Fatalf("AddressResetPrimary() error = %v", err)
	}

	telAddr := &contact.Address{
		Address: commonaddress.Address{
			Type: contact.AddressTypeTel,
			Target: "+155****1111",
		},
		ID: uuid.FromStringOrNil("c1c1c1c1-0003-0003-0003-000000000003"),
		CustomerID: customerID,
		ContactID: contactID,
		IsPrimary: true,
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressCreate(ctx, telAddr); err != nil {
		t.Fatalf("AddressCreate(tel) error = %v", err)
	}

	// Tel address should be primary at this point.
	gotTel, err := h.AddressGet(ctx, customerID, telAddr.ID)
	if err != nil {
		t.Fatalf("AddressGet(tel) error = %v", err)
	}
	if !gotTel.IsPrimary {
		t.Fatalf("expected the tel address to be primary after creation")
	}

	// Handler path for adding a primary email address:
	// reset existing primaries (cross-type, must demote the tel), then create.
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressResetPrimary(ctx, contactID); err != nil {
		t.Fatalf("AddressResetPrimary() error = %v", err)
	}

	emailAddr := &contact.Address{
		Address: commonaddress.Address{
			Type: contact.AddressTypeEmail,
			Target: "primary@example.com",
		},
		ID: uuid.FromStringOrNil("c1c1c1c1-0004-0004-0004-000000000004"),
		CustomerID: customerID,
		ContactID: contactID,
		IsPrimary: true,
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressCreate(ctx, emailAddr); err != nil {
		t.Fatalf("AddressCreate(email) error = %v", err)
	}

	// Assert cross-type single primary: the tel must be demoted, the email primary.
	gotTel, err = h.AddressGet(ctx, customerID, telAddr.ID)
	if err != nil {
		t.Fatalf("AddressGet(tel) error = %v", err)
	}
	if gotTel.IsPrimary {
		t.Errorf("expected the tel address to be demoted after adding a primary email, but it is still primary")
	}

	gotEmail, err := h.AddressGet(ctx, customerID, emailAddr.ID)
	if err != nil {
		t.Fatalf("AddressGet(email) error = %v", err)
	}
	if !gotEmail.IsPrimary {
		t.Errorf("expected the email address to be primary, but it is not")
	}
}
