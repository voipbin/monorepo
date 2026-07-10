package dbhandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_AddressLookupContactIDByTypeTarget verifies the generic
// (type, target) -> contact_id lookup used by Case get-or-create's
// contact auto-match step (design §4 step 2). Unlike ContactLookupByPhone/
// ByEmail (tel/email specific), Case's peer_type can be any
// commonaddress.Type (call reference_type today, others later).
func Test_AddressLookupContactIDByTypeTarget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-6001-6001-6001-000000000001")
	contactID := uuid.FromStringOrNil("f1b2c3d4-6001-6001-6001-000000000002")
	addressID := uuid.FromStringOrNil("f1b2c3d4-6001-6001-6001-000000000003")
	unresolvedAddressID := uuid.FromStringOrNil("f1b2c3d4-6001-6001-6001-000000000004")

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Now())).AnyTimes()
	mockCache.EXPECT().ContactSet(ctx, gomock.Any()).AnyTimes()
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "Address",
		LastName:  "LookupTest",
		Source:    "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	resolvedAddr := &contact.Address{
		Address: commonaddress.Address{
			Type:   commonaddress.TypeTel,
			Target: "+155****9001",
		},
		ID:         addressID,
		CustomerID: customerID,
		ContactID:  contactID,
	}
	if err := h.AddressCreate(ctx, resolvedAddr); err != nil {
		t.Fatalf("AddressCreate(resolved) error = %v", err)
	}

	unresolvedAddr := &contact.Address{
		Address: commonaddress.Address{
			Type:   commonaddress.TypeTel,
			Target: "+155****9002",
		},
		ID:         unresolvedAddressID,
		CustomerID: customerID,
		ContactID:  uuid.Nil,
	}
	if err := h.AddressCreate(ctx, unresolvedAddr); err != nil {
		t.Fatalf("AddressCreate(unresolved) error = %v", err)
	}

	res, err := h.AddressLookupContactIDByTypeTarget(ctx, customerID, commonaddress.TypeTel, "+155****9001")
	if err != nil {
		t.Fatalf("AddressLookupContactIDByTypeTarget() error = %v", err)
	}
	if res != contactID {
		t.Errorf("expected contact_id: %v, got: %v", contactID, res)
	}

	// Unresolved address (contact_id NULL/nil) -> ErrNotFound, same as no
	// address row at all: neither case should attribute a Case's contact_id.
	_, err = h.AddressLookupContactIDByTypeTarget(ctx, customerID, commonaddress.TypeTel, "+15551119002")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for unresolved address, got: %v", err)
	}

	// No matching address at all -> ErrNotFound.
	_, err = h.AddressLookupContactIDByTypeTarget(ctx, customerID, commonaddress.TypeTel, "+19999999999")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for no-match, got: %v", err)
	}
}
