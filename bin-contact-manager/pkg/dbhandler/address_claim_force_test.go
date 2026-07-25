package dbhandler

// Tests for DESIGN.md §4 (v7) -- the `force` parameter extension to
// AddressClaim/addressClaimAttempt. See DESIGN.md §8 for the test plan
// these cases implement.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_AddressClaim_Force_False_UnresolvedTarget_Unaffected verifies that
// force=false against an unresolved (contact_id IS NULL) address behaves
// exactly as the pre-v7 contract -- the claim succeeds and no ownership
// weirdness is introduced. This is the baseline regression guard for
// DESIGN.md §8's "force 없음/false: 기존 계약 100% 유지" bullet.
func Test_AddressClaim_Force_False_UnresolvedTarget_Unaffected(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000002")
	target := "+155****4001"
	createTestContact(t, h, ctx, customerID, contactID)

	addrID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000003")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
	}); err != nil {
		t.Fatalf("AddressCreate(unresolved) error = %v", err)
	}

	if err := h.AddressClaim(ctx, customerID, addrID, contactID, false); err != nil {
		t.Fatalf("AddressClaim(force=false, unresolved) error = %v, want success", err)
	}

	got, err := h.AddressGet(ctx, customerID, addrID)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if got.ContactID != contactID {
		t.Errorf("ContactID = %s, want %s", got.ContactID, contactID)
	}
}

// Test_AddressClaim_Force_False_LiveOwner_Conflict verifies the default
// (force absent/false) contract is unchanged: claiming an address already
// owned by a DIFFERENT, live contact returns ErrConflict.
func Test_AddressClaim_Force_False_LiveOwner_Conflict(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000101")
	ownerID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000102")
	claimerID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000103")
	target := "+155****4002"
	createTestContact(t, h, ctx, customerID, ownerID)
	createTestContact(t, h, ctx, customerID, claimerID)

	addrID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000104")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  ownerID,
	}); err != nil {
		t.Fatalf("AddressCreate(live owner) error = %v", err)
	}

	if err := h.AddressClaim(ctx, customerID, addrID, claimerID, false); err != ErrConflict {
		t.Errorf("AddressClaim(force=false, live owner) error = %v, want ErrConflict", err)
	}

	got, err := h.AddressGet(ctx, customerID, addrID)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if got.ContactID != ownerID {
		t.Errorf("ownership must be undisturbed on conflict: ContactID = %s, want %s", got.ContactID, ownerID)
	}
}

// Test_AddressClaim_Force_True_Unresolved_NoOp verifies force=true against
// an unresolved address is a harmless no-op path -- it behaves the same as
// force=false (DESIGN.md §8: "force가 무해한 no-op으로 동작하는지").
func Test_AddressClaim_Force_True_Unresolved_NoOp(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000201")
	contactID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000202")
	target := "+155****4003"
	createTestContact(t, h, ctx, customerID, contactID)

	addrID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000203")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
	}); err != nil {
		t.Fatalf("AddressCreate(unresolved) error = %v", err)
	}

	if err := h.AddressClaim(ctx, customerID, addrID, contactID, true); err != nil {
		t.Fatalf("AddressClaim(force=true, unresolved) error = %v, want success", err)
	}

	got, err := h.AddressGet(ctx, customerID, addrID)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if got.ContactID != contactID {
		t.Errorf("ContactID = %s, want %s", got.ContactID, contactID)
	}
}

// Test_AddressClaim_Force_True_LiveOwner_ReassignsAndClosesPeriod is the
// core DESIGN.md §4 test: force=true against an address owned by a
// DIFFERENT, live contact must succeed (no ErrConflict), reassign
// ownership to the new contact, and leave the ownership-period history
// table with the previous owner's period CLOSED and the new owner's
// period OPEN (DESIGN.md §8's "DB 레벨로 직접 검증" requirement).
func Test_AddressClaim_Force_True_LiveOwner_ReassignsAndClosesPeriod(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000301")
	prevOwnerID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000302")
	newOwnerID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000303")
	target := "+155****4004"
	createTestContact(t, h, ctx, customerID, prevOwnerID)
	createTestContact(t, h, ctx, customerID, newOwnerID)

	addrID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000304")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  prevOwnerID,
	}); err != nil {
		t.Fatalf("AddressCreate(live owner) error = %v", err)
	}

	if err := h.AddressClaim(ctx, customerID, addrID, newOwnerID, true); err != nil {
		t.Fatalf("AddressClaim(force=true, live owner) error = %v, want success (no ErrConflict)", err)
	}

	got, err := h.AddressGet(ctx, customerID, addrID)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if got.ContactID != newOwnerID {
		t.Errorf("ContactID = %s, want %s (force:true must reassign)", got.ContactID, newOwnerID)
	}

	// DB-level verification of the ownership-period history table:
	// previous owner's period must be CLOSED, new owner's period must be OPEN.
	rows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	var prevClosed, newOpen *OwnershipPeriod
	for i := range rows {
		p := &rows[i]
		if p.ContactID == prevOwnerID && p.ValidTo != nil {
			prevClosed = p
		}
		if p.ContactID == newOwnerID && p.ValidTo == nil {
			newOpen = p
		}
	}
	if prevClosed == nil {
		t.Errorf("expected the previous live owner's open period to be CLOSED via closeOwnOpenPeriodTx, got: %+v", rows)
	}
	if newOpen == nil {
		t.Errorf("expected a new OPEN period for the new owner, got: %+v", rows)
	}
}

// Test_AddressClaim_Force_True_TombstonedOwner_SameAsBefore verifies that
// force has NO effect on the tombstoned-owner repair-in-place path
// (DESIGN.md §4.3: tombstone and force branches are explicit, separate
// code paths using staleRowRepairTx vs closeOwnOpenPeriodTx). Both
// force=true and force=false must behave identically here.
func Test_AddressClaim_Force_True_TombstonedOwner_SameAsBefore(t *testing.T) {
	for i, force := range []bool{true, false} {
		t.Run(boolLabel(force), func(t *testing.T) {
			h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)))
			defer mc.Finish()
			ctx := context.Background()

			customerID := uuid.FromStringOrNil(fmt.Sprintf("74000000-0000-0000-0000-0000000041%02d", i))
			deadOwnerID := uuid.FromStringOrNil(fmt.Sprintf("74000000-0000-0000-0000-0000000042%02d", i))
			newOwnerID := uuid.FromStringOrNil(fmt.Sprintf("74000000-0000-0000-0000-0000000043%02d", i))
			target := fmt.Sprintf("+155****500%d", i)
			createTestContact(t, h, ctx, customerID, deadOwnerID)
			createTestContact(t, h, ctx, customerID, newOwnerID)

			addrID := uuid.FromStringOrNil(fmt.Sprintf("74000000-0000-0000-0000-0000000044%02d", i))
			if err := h.AddressCreate(ctx, &contact.Address{
				Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
				ID:         addrID,
				CustomerID: customerID,
				ContactID:  deadOwnerID,
			}); err != nil {
				t.Fatalf("AddressCreate(dead owner) error = %v", err)
			}

			if err := h.ContactDelete(ctx, deadOwnerID); err != nil {
				t.Fatalf("ContactDelete() error = %v", err)
			}

			if err := h.AddressClaim(ctx, customerID, addrID, newOwnerID, force); err != nil {
				t.Fatalf("AddressClaim(force=%v, tombstoned owner) error = %v, want success via repair-in-place", force, err)
			}

			got, err := h.AddressGet(ctx, customerID, addrID)
			if err != nil {
				t.Fatalf("AddressGet() error = %v", err)
			}
			if got.ContactID != newOwnerID {
				t.Errorf("ContactID = %s, want %s", got.ContactID, newOwnerID)
			}
		})
	}
}

func boolLabel(b bool) string {
	if b {
		return "force_true"
	}
	return "force_false"
}

// Test_AddressList_TargetFilter verifies DESIGN.md §4.6's `target` query
// filter on AddressList -- an exact match on the target column, using the
// same filters-map pattern as the existing `type` filter.
func Test_AddressList_TargetFilter(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000501")
	targetA := "+155****6001"
	targetB := "+155****6002"
	curTime := timePtr(time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC))

	mockUtil.EXPECT().TimeNow().Return(curTime).AnyTimes()
	addrA := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000502")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: targetA},
		ID:         addrA,
		CustomerID: customerID,
	}); err != nil {
		t.Fatalf("AddressCreate(A) error = %v", err)
	}
	addrB := uuid.FromStringOrNil("74000000-0000-0000-0000-000000000503")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: targetB},
		ID:         addrB,
		CustomerID: customerID,
	}); err != nil {
		t.Fatalf("AddressCreate(B) error = %v", err)
	}

	res, err := h.AddressList(ctx, customerID, map[string]any{"target": targetA}, "", 0)
	if err != nil {
		t.Fatalf("AddressList(target=%s) error = %v", targetA, err)
	}
	if len(res) != 1 {
		t.Fatalf("AddressList(target=%s) returned %d rows, want 1: %+v", targetA, len(res), res)
	}
	if res[0].ID != addrA {
		t.Errorf("AddressList(target=%s) returned ID = %v, want %v", targetA, res[0].ID, addrA)
	}
	if res[0].Target != targetA {
		t.Errorf("AddressList(target=%s) returned Target = %v, want %v", targetA, res[0].Target, targetA)
	}
}
