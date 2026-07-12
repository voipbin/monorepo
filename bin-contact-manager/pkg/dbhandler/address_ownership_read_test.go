package dbhandler

// Scenario tests for the ownership-period READ path (design §6, Phase 2
// of NOJIRA-contact-address-ownership-periods). Uses the real SQLite
// test DB and the actual write-path functions (AddressCreate/
// AddressDelete/AddressClaim) to build real ownership history, then
// exercises InteractionListByOwnershipPeriods and
// InteractionListUnresolved directly -- proving the four original
// defects this whole design exists to fix are actually resolved, not
// just that the new functions compile.

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/interaction"
)

func mustCreateInteraction(t *testing.T, ctx context.Context, h *handler, id, customerID uuid.UUID, peerType, peerTarget string, tmCreate time.Time) {
	t.Helper()
	i := &interaction.Interaction{
		ID:            id,
		CustomerID:    customerID,
		Direction:     "incoming",
		PeerType:      peerType,
		PeerTarget:    peerTarget,
		LocalType:     "tel",
		LocalTarget:   "+155****0000",
		ReferenceType: "call",
		ReferenceID:   uuid.Must(uuid.NewV4()),
		TMCreate:      &tmCreate,
	}
	if err := h.InteractionCreate(ctx, i); err != nil {
		t.Fatalf("InteractionCreate() error = %v", err)
	}
}

// Test_InteractionListByOwnershipPeriods_DeletedTarget_HistoryStillMatches
// is the core "defect #1" regression: today, deleting a phone number from
// a Contact makes its past interactions vanish from that Contact's
// timeline, because STEP1 only lists currently-live contact_addresses
// rows. With the ownership-period rewrite, a CLOSED period (the register
// -> delete lifecycle) still produces a bound covering the interaction's
// original tm_create, so it continues to match.
func Test_InteractionListByOwnershipPeriods_DeletedTarget_HistoryStillMatches(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("91000000-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("91000000-0000-0000-0000-000000000002")
	addrID := uuid.FromStringOrNil("91000000-0000-0000-0000-000000000003")
	target := "+15550001001"

	createTestContact(t, h, ctx, customerID, contactID)

	// Register the number (opens a period), record an interaction while
	// it's live, then delete the number (closes the period). This is
	// the exact register -> interact -> delete sequence design §1 calls
	// out as today's "delete -> history disappears" defect.
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactID,
	}); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	interactionID := uuid.FromStringOrNil("91000000-0000-0000-0000-000000000004")
	mustCreateInteraction(t, ctx, h, interactionID, customerID, string(contact.AddressTypeTel), target,
		time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC))

	if err := h.AddressDelete(ctx, addrID); err != nil {
		t.Fatalf("AddressDelete() error = %v", err)
	}

	periods, err := h.OwnershipPeriodsListByContactID(ctx, contactID)
	if err != nil {
		t.Fatalf("OwnershipPeriodsListByContactID() error = %v", err)
	}
	if len(periods) != 1 {
		t.Fatalf("OwnershipPeriodsListByContactID() len = %d, want 1 (one opened-then-closed period)", len(periods))
	}
	if periods[0].ValidTo == nil {
		t.Fatalf("OwnershipPeriodsListByContactID()[0].ValidTo = nil, want closed (AddressDelete must close the period)")
	}

	bounds := []OwnershipPeriodBound{{Type: periods[0].Type, Target: periods[0].Target, ValidFrom: periods[0].ValidFrom, ValidTo: periods[0].ValidTo}}
	got, err := h.InteractionListByOwnershipPeriods(ctx, customerID, 20, "", "", "", bounds, time.Time{})
	if err != nil {
		t.Fatalf("InteractionListByOwnershipPeriods() error = %v", err)
	}
	found := false
	for _, i := range got {
		if i.ID == interactionID {
			found = true
		}
	}
	if !found {
		t.Errorf("InteractionListByOwnershipPeriods() did not return the pre-delete interaction -- defect #1 (delete -> history disappears) is NOT fixed. got: %+v", got)
	}
}

// Test_InteractionListByOwnershipPeriods_Reassignment_NoCrossAttribution
// is the "defect #2" regression: when a number is deleted from Contact A
// and re-registered to Contact B, A's past interactions must stay on A's
// timeline (bounded to A's closed period) and must NOT appear on B's
// timeline (B's period only opens after the reassignment).
func Test_InteractionListByOwnershipPeriods_Reassignment_NoCrossAttribution(t *testing.T) {
	target := "+15550002002"
	customerID := uuid.FromStringOrNil("92000000-0000-0000-0000-000000000001")
	contactA := uuid.FromStringOrNil("92000000-0000-0000-0000-000000000002")
	contactB := uuid.FromStringOrNil("92000000-0000-0000-0000-000000000003")
	addrIDA := uuid.FromStringOrNil("92000000-0000-0000-0000-000000000004")
	addrIDB := uuid.FromStringOrNil("92000000-0000-0000-0000-000000000005")

	// A single fixed "now" (design's test convention -- TimeNow mocked
	// once with AnyTimes()) sits between A's interaction and B's
	// interaction; AddressCreate(A)/AddressDelete(A)/AddressCreate(B)
	// all timestamp their period rows at this same instant, which is
	// enough to prove the bound-membership logic: A's period closes at
	// `now`, B's period opens at `now`, A's own interaction (before
	// `now`) must stay out of B's window and B's interaction (after
	// `now`) must stay out of A's window.
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	h, mc := newOwnershipTestHandler(t, &now)
	defer mc.Finish()
	ctx := context.Background()

	createTestContact(t, h, ctx, customerID, contactA)
	createTestContact(t, h, ctx, customerID, contactB)

	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrIDA,
		CustomerID: customerID,
		ContactID:  contactA,
	}); err != nil {
		t.Fatalf("AddressCreate(A) error = %v", err)
	}

	interactionA := uuid.FromStringOrNil("92000000-0000-0000-0000-000000000006")
	mustCreateInteraction(t, ctx, h, interactionA, customerID, string(contact.AddressTypeTel), target,
		now.Add(-1*time.Hour))

	if err := h.AddressDelete(ctx, addrIDA); err != nil {
		t.Fatalf("AddressDelete(A) error = %v", err)
	}

	// Reassignment: B registers the same target (same mocked `now`,
	// consistent with StepReassign's design §4 Step 4 rule that both
	// A's close and B's open share the single locking-read transaction
	// window in a real concurrent scenario).
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrIDB,
		CustomerID: customerID,
		ContactID:  contactB,
	}); err != nil {
		t.Fatalf("AddressCreate(B) error = %v", err)
	}

	interactionB := uuid.FromStringOrNil("92000000-0000-0000-0000-000000000007")
	mustCreateInteraction(t, ctx, h, interactionB, customerID, string(contact.AddressTypeTel), target,
		now.Add(1*time.Hour))

	periodsA, err := h.OwnershipPeriodsListByContactID(ctx, contactA)
	if err != nil {
		t.Fatalf("OwnershipPeriodsListByContactID(A) error = %v", err)
	}
	periodsB, err := h.OwnershipPeriodsListByContactID(ctx, contactB)
	if err != nil {
		t.Fatalf("OwnershipPeriodsListByContactID(B) error = %v", err)
	}
	if len(periodsA) != 1 || len(periodsB) != 1 {
		t.Fatalf("expected exactly one period each for A/B, got A=%d B=%d", len(periodsA), len(periodsB))
	}

	boundsA := []OwnershipPeriodBound{{Type: periodsA[0].Type, Target: periodsA[0].Target, ValidFrom: periodsA[0].ValidFrom, ValidTo: periodsA[0].ValidTo}}
	boundsB := []OwnershipPeriodBound{{Type: periodsB[0].Type, Target: periodsB[0].Target, ValidFrom: periodsB[0].ValidFrom, ValidTo: periodsB[0].ValidTo}}

	gotA, err := h.InteractionListByOwnershipPeriods(ctx, customerID, 20, "", "", "", boundsA, time.Time{})
	if err != nil {
		t.Fatalf("InteractionListByOwnershipPeriods(A) error = %v", err)
	}
	gotB, err := h.InteractionListByOwnershipPeriods(ctx, customerID, 20, "", "", "", boundsB, time.Time{})
	if err != nil {
		t.Fatalf("InteractionListByOwnershipPeriods(B) error = %v", err)
	}

	assertContainsOnly(t, "A's timeline", gotA, interactionA)
	assertContainsOnly(t, "B's timeline", gotB, interactionB)
}

func assertContainsOnly(t *testing.T, label string, got []*interaction.Interaction, want uuid.UUID) {
	t.Helper()
	found := false
	for _, i := range got {
		if i.ID == want {
			found = true
		} else {
			t.Errorf("%s unexpectedly contains interaction %v (cross-attribution defect #2 NOT fixed)", label, i.ID)
		}
	}
	if !found {
		t.Errorf("%s does not contain expected interaction %v", label, want)
	}
}

// Test_InteractionListUnresolved_PastOwnerPeriod_NotUnresolved is the
// "defect #4" regression for a target's PAST owner: once a period is
// closed (target deleted, never reassigned), that owner's old
// interactions must NOT resurface in the unresolved queue just because
// the live contact_addresses row is gone -- design §6.3's first
// disjunct (ownership-period match) must still suppress them.
func Test_InteractionListUnresolved_PastOwnerPeriod_NotUnresolved(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("93000000-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("93000000-0000-0000-0000-000000000002")
	addrID := uuid.FromStringOrNil("93000000-0000-0000-0000-000000000003")
	target := "+15550003003"

	createTestContact(t, h, ctx, customerID, contactID)

	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactID,
	}); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	interactionID := uuid.FromStringOrNil("93000000-0000-0000-0000-000000000004")
	mustCreateInteraction(t, ctx, h, interactionID, customerID, string(contact.AddressTypeTel), target,
		time.Date(2026, 3, 4, 8, 0, 0, 0, time.UTC))

	if err := h.AddressDelete(ctx, addrID); err != nil {
		t.Fatalf("AddressDelete() error = %v", err)
	}

	unresolved, err := h.InteractionListUnresolved(ctx, customerID, 100, "", time.Time{})
	if err != nil {
		t.Fatalf("InteractionListUnresolved() error = %v", err)
	}
	for _, i := range unresolved {
		if i.ID == interactionID {
			t.Errorf("InteractionListUnresolved() resurfaced a past-owner-period interaction after target deletion -- defect #4 NOT fixed")
		}
	}
}

// Test_InteractionListUnresolved_LiveOwnedNoPeriod_MissingPeriodSkewSuppressed
// covers design §6.3's round-40/42/43 missing-period-skew guard directly:
// a live contact_addresses row with NO period row of its own (simulating
// an old-binary pod's AddressCreate that predates this rewire, or any
// out-of-band insert) must NOT have its interactions resurface in the
// unresolved queue just because it lacks a period.
func Test_InteractionListUnresolved_LiveOwnedNoPeriod_MissingPeriodSkewSuppressed(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 3, 5, 9, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("94000000-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("94000000-0000-0000-0000-000000000002")
	target := "+15550004004"

	createTestContact(t, h, ctx, customerID, contactID)

	// Simulate the skew: insert a live contact_addresses row directly
	// (bypassing AddressCreate/OwnershipPeriodsLockAndResolveTx), so no
	// period row is ever written for it -- exactly the degraded state
	// an old-binary pod's AddressCreate would leave behind.
	insertRawAddressNoPeriod(t, ctx, h, customerID, contactID, contact.AddressTypeTel, target)

	interactionID := uuid.FromStringOrNil("94000000-0000-0000-0000-000000000003")
	mustCreateInteraction(t, ctx, h, interactionID, customerID, string(contact.AddressTypeTel), target,
		time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC))

	unresolved, err := h.InteractionListUnresolved(ctx, customerID, 100, "", time.Time{})
	if err != nil {
		t.Fatalf("InteractionListUnresolved() error = %v", err)
	}
	for _, i := range unresolved {
		if i.ID == interactionID {
			t.Errorf("InteractionListUnresolved() resurfaced a live-owned, period-less interaction -- missing-period-skew guard NOT working")
		}
	}

	// And STEP1's mirror-image guard: the contact's OWN timeline must
	// still see it via the missing-period-skew unbounded bound (design
	// §6.2 round-41-43/47), not lose it entirely.
	skewed, err := h.MissingPeriodOwnedAddresses(ctx, customerID, contactID)
	if err != nil {
		t.Fatalf("MissingPeriodOwnedAddresses() error = %v", err)
	}
	found := false
	for _, s := range skewed {
		if s.Type == string(contact.AddressTypeTel) && s.Target == target {
			found = true
		}
	}
	if !found {
		t.Errorf("MissingPeriodOwnedAddresses() did not report the period-less owned row -- STEP1's own-timeline guard NOT working. got: %+v", skewed)
	}
}

// insertRawAddressNoPeriod inserts a contact_addresses row directly via
// AddressCreateTx's underlying INSERT, bypassing the ownership-period
// write entirely -- reproducing the missing-period-skew degraded state
// without needing an actual old binary.
func insertRawAddressNoPeriod(t *testing.T, ctx context.Context, h *handler, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) {
	t.Helper()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	a := &contact.Address{
		Address:    commonaddress.Address{Type: addrType, Target: target},
		ID:         uuid.Must(uuid.NewV4()),
		CustomerID: customerID,
		ContactID:  contactID,
	}
	if err := h.addressInsertTx(ctx, tx, a); err != nil {
		t.Fatalf("addressInsertTx() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit error = %v", err)
	}
}
