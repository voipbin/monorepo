package dbhandler

// Scenario tests for the ownership-period READ path (design §6, Phase 2
// of NOJIRA-contact-address-ownership-periods). Uses the real SQLite
// test DB and the actual write-path functions (AddressCreate/
// AddressDelete/AddressClaim/CreateUnresolvedAddress) to build real
// ownership history, then exercises InteractionListByOwnershipPeriods
// and InteractionListUnresolved directly -- proving the four original
// defects this whole design exists to fix are actually resolved, not
// just that the new functions compile.
//
// UUID prefix convention: like address_ownership_test.go, this file
// shares one process-wide SQLite DB (dbTest, main_test.go) across every
// test in the package with no per-test truncation/cleanup. Isolation is
// achieved purely by giving each test function its own customerID/
// contactID UUID prefix (91xxxxxx.. through 98xxxxxx.. in this file;
// address_ownership_test.go separately owns 70xxxxxx../72xxxxxx../
// 73xxxxxx..). Every query in this package is customer_id-scoped, so
// distinct prefixes are sufficient for isolation -- but there is no
// compile-time or lint guard against a future test reusing a prefix and
// silently corrupting another test's fixture. Pick an unused prefix
// (grep the byte before checking one in) when adding a new test here.

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

func mustCreateInteraction(t *testing.T, ctx context.Context, h *handler, id, customerID uuid.UUID, peerType, peerTarget string, tmCreate time.Time) {
	t.Helper()
	i := &interaction.Interaction{
		ID:            id,
		CustomerID:    customerID,
		Direction:     "incoming",
		Peer: commonaddress.Address{Type: commonaddress.Type(peerType), Target: peerTarget},
		Local: commonaddress.Address{Type: "tel", Target: "+155****0000"},
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

// Test_InteractionListUnresolved_SameOwnerReacquire_MissingPeriodSkewSuppressed
// covers design §6.3 round-43's BLOCKER fix directly: contact A registers
// a target, releases it (closed period), then re-registers the SAME
// target but the new row is written WITHOUT a period (simulating an
// old-binary pod, same as insertRawAddressNoPeriod). Round-42's
// owner-scoped-but-not-open-scoped condition would have missed this --
// it finds A's own OLD closed period and does not fire, leaving the
// reacquired row's interactions to wrongly resurface in the unresolved
// queue. Round-43 adds `valid_to IS NULL` to fix exactly this.
func Test_InteractionListUnresolved_SameOwnerReacquire_MissingPeriodSkewSuppressed(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 3, 6, 9, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("95000000-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("95000000-0000-0000-0000-000000000002")
	addrID := uuid.FromStringOrNil("95000000-0000-0000-0000-000000000003")
	target := "+15550005005"

	createTestContact(t, h, ctx, customerID, contactID)

	// A registers, then releases -- produces one CLOSED period for A.
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactID,
	}); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}
	if err := h.AddressDelete(ctx, addrID); err != nil {
		t.Fatalf("AddressDelete() error = %v", err)
	}
	closedRows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(closedRows) != 1 || closedRows[0].ValidTo == nil {
		t.Fatalf("expected exactly one closed period for A before reacquisition, got: %+v", closedRows)
	}

	// A reacquires the SAME target, but the new row is written without
	// a period (old-binary-pod simulation) -- round-42's condition
	// would see A's OLD closed period and wrongly conclude "this owner
	// has a period," missing the reacquisition.
	insertRawAddressNoPeriod(t, ctx, h, customerID, contactID, contact.AddressTypeTel, target)

	interactionID := uuid.FromStringOrNil("95000000-0000-0000-0000-000000000004")
	mustCreateInteraction(t, ctx, h, interactionID, customerID, string(contact.AddressTypeTel), target,
		time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC))

	unresolved, err := h.InteractionListUnresolved(ctx, customerID, 100, "", time.Time{})
	if err != nil {
		t.Fatalf("InteractionListUnresolved() error = %v", err)
	}
	for _, i := range unresolved {
		if i.ID == interactionID {
			t.Errorf("InteractionListUnresolved() resurfaced a same-owner-reacquired, period-less interaction -- round-43 fix NOT working (round-42's owner-scoped-only condition regression)")
		}
	}
}

// Test_InteractionListUnresolved_Reassignment_MissingPeriodSkewSuppressed
// covers design §6.3 round-42's fix: contact A registers a target,
// releases it (closed period), then a DIFFERENT contact B registers the
// SAME target but the new row is written WITHOUT a period. Round-41's
// target-wide condition ("no period row for this target at all") would
// have missed this -- it sees A's closed period and does not fire,
// wrongly resurfacing B's interactions in the unresolved queue despite
// B's live, normal ownership. Round-42 scopes the condition to the
// OWNER (B), not the target, fixing exactly this.
func Test_InteractionListUnresolved_Reassignment_MissingPeriodSkewSuppressed(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 3, 7, 9, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("96000000-0000-0000-0000-000000000001")
	contactA := uuid.FromStringOrNil("96000000-0000-0000-0000-000000000002")
	contactB := uuid.FromStringOrNil("96000000-0000-0000-0000-000000000003")
	addrIDA := uuid.FromStringOrNil("96000000-0000-0000-0000-000000000004")
	target := "+15550006006"

	createTestContact(t, h, ctx, customerID, contactA)
	createTestContact(t, h, ctx, customerID, contactB)

	// A registers, then releases -- produces one CLOSED period for A.
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrIDA,
		CustomerID: customerID,
		ContactID:  contactA,
	}); err != nil {
		t.Fatalf("AddressCreate(A) error = %v", err)
	}
	if err := h.AddressDelete(ctx, addrIDA); err != nil {
		t.Fatalf("AddressDelete(A) error = %v", err)
	}
	closedRows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(closedRows) != 1 || closedRows[0].ContactID != contactA || closedRows[0].ValidTo == nil {
		t.Fatalf("expected exactly one closed period for A before reassignment, got: %+v", closedRows)
	}

	// B registers the same target, written without a period
	// (old-binary-pod simulation): round-41's target-wide anti-join
	// would see A's closed period and wrongly conclude "this target
	// isn't period-less overall," missing B's own skew.
	insertRawAddressNoPeriod(t, ctx, h, customerID, contactB, contact.AddressTypeTel, target)

	interactionID := uuid.FromStringOrNil("96000000-0000-0000-0000-000000000005")
	mustCreateInteraction(t, ctx, h, interactionID, customerID, string(contact.AddressTypeTel), target,
		time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC))

	unresolved, err := h.InteractionListUnresolved(ctx, customerID, 100, "", time.Time{})
	if err != nil {
		t.Fatalf("InteractionListUnresolved() error = %v", err)
	}
	for _, i := range unresolved {
		if i.ID == interactionID {
			t.Errorf("InteractionListUnresolved() resurfaced a reassigned, period-less interaction -- round-42 owner-scoping fix NOT working (round-41's target-wide condition regression)")
		}
	}
}

// Test_InteractionListUnresolved_ClaimOfUnresolved_ImmediatelyPriorGapAttaches
// covers design §6.3 round-37/38/39: an unresolved (contact_id IS NULL)
// address that sits directly after a prior owner's closed period, once
// claimed, must attach interactions from BOTH the unresolved era AND the
// immediately-prior owner's era (per AddressClaim's valid_from = latest
// closed valid_to rule) -- neither era's interactions should resurface
// in the unresolved queue after the claim.
func Test_InteractionListUnresolved_ClaimOfUnresolved_ImmediatelyPriorGapAttaches(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("97000000-0000-0000-0000-000000000001")
	priorOwner := uuid.FromStringOrNil("97000000-0000-0000-0000-000000000002")
	claimer := uuid.FromStringOrNil("97000000-0000-0000-0000-000000000003")
	target := "+15550007007"

	createTestContact(t, h, ctx, customerID, priorOwner)
	createTestContact(t, h, ctx, customerID, claimer)

	// Prior owner's era: register, interact, release.
	priorAddrID := uuid.FromStringOrNil("97000000-0000-0000-0000-000000000004")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         priorAddrID,
		CustomerID: customerID,
		ContactID:  priorOwner,
	}); err != nil {
		t.Fatalf("AddressCreate(prior owner) error = %v", err)
	}
	priorInteractionID := uuid.FromStringOrNil("97000000-0000-0000-0000-000000000005")
	mustCreateInteraction(t, ctx, h, priorInteractionID, customerID, string(contact.AddressTypeTel), target,
		time.Date(2026, 3, 8, 1, 0, 0, 0, time.UTC))
	if err := h.AddressDelete(ctx, priorAddrID); err != nil {
		t.Fatalf("AddressDelete(prior owner) error = %v", err)
	}

	// Unresolved era: an unresolved row for the same target, with an
	// interaction recorded while it's unresolved.
	unresolvedAddrID := uuid.FromStringOrNil("97000000-0000-0000-0000-000000000006")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         unresolvedAddrID,
		CustomerID: customerID,
		ContactID:  uuid.Nil,
	}); err != nil {
		t.Fatalf("AddressCreate(unresolved) error = %v", err)
	}
	unresolvedInteractionID := uuid.FromStringOrNil("97000000-0000-0000-0000-000000000007")
	mustCreateInteraction(t, ctx, h, unresolvedInteractionID, customerID, string(contact.AddressTypeTel), target,
		time.Date(2026, 3, 8, 2, 0, 0, 0, time.UTC))

	// Sanity: both interactions ARE unresolved before the claim (prior
	// owner's era via row-presence-agnostic period suppression should
	// NOT apply yet since AddressDelete already closed it, and the
	// unresolved row suppresses by presence) -- this pre-claim baseline
	// isn't the point of the test, skip asserting it and go straight to
	// post-claim, which IS the point (round-37/38/39's actual claim).
	if err := h.AddressClaim(ctx, customerID, unresolvedAddrID, claimer); err != nil {
		t.Fatalf("AddressClaim() error = %v", err)
	}

	unresolved, err := h.InteractionListUnresolved(ctx, customerID, 100, "", time.Time{})
	if err != nil {
		t.Fatalf("InteractionListUnresolved() error = %v", err)
	}
	for _, i := range unresolved {
		if i.ID == priorInteractionID {
			t.Errorf("InteractionListUnresolved() resurfaced the immediately-prior-gap interaction after claim -- round-37/38/39 fix NOT working (AddressClaim's valid_from=latest-closed-valid_to rule should have covered it)")
		}
		if i.ID == unresolvedInteractionID {
			t.Errorf("InteractionListUnresolved() resurfaced the claimed-unresolved-era interaction after claim -- ownership-period disjunct should now cover it")
		}
	}

	// And on the POSITIVE side: the claimer's own timeline must now see
	// both eras' interactions via its period bound.
	claimerPeriods, err := h.OwnershipPeriodsListByContactID(ctx, claimer)
	if err != nil {
		t.Fatalf("OwnershipPeriodsListByContactID(claimer) error = %v", err)
	}
	if len(claimerPeriods) != 1 {
		t.Fatalf("expected exactly one period for the claimer, got %d: %+v", len(claimerPeriods), claimerPeriods)
	}
	bounds := []OwnershipPeriodBound{{Type: claimerPeriods[0].Type, Target: claimerPeriods[0].Target, ValidFrom: claimerPeriods[0].ValidFrom, ValidTo: claimerPeriods[0].ValidTo}}
	claimerTimeline, err := h.InteractionListByOwnershipPeriods(ctx, customerID, 20, "", "", "", bounds, time.Time{})
	if err != nil {
		t.Fatalf("InteractionListByOwnershipPeriods(claimer) error = %v", err)
	}
	assertContainsIDs(t, "claimer's timeline", claimerTimeline, unresolvedInteractionID, priorInteractionID)
}

// Test_InteractionListUnresolved_ClaimOfUnresolved_NonAdjacentGapResurfaces
// covers design §6.3 round-39's correction: "matching today's observable
// behavior" only extends to the IMMEDIATELY PRIOR gap, not to every
// earlier gap in a multi-era history. An older, non-adjacent gap
// (an A-to-B gap preceding a B-to-claimer gap) is NOT covered by the
// claim-created period and DOES resurface in the unresolved queue once
// the claim removes the row-presence suppression that had covered it
// unconditionally. This is accepted design behavior (§4), pinned here
// so a future change cannot silently alter it either way.
//
// Uses a hand-rolled, strictly increasing TimeNow() sequence (rather than
// newOwnershipTestHandler's single fixed-time convention) because this
// test needs each era's period boundary to land at a DIFFERENT instant --
// a single shared "now" (as every other test in this file uses) would
// collapse every period to the same boundary and make the eras
// indistinguishable by timestamp, which is exactly what this test needs
// to tell apart.
func Test_InteractionListUnresolved_ClaimOfUnresolved_NonAdjacentGapResurfaces(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := newMonotonicTimeUtilHandler(t, mc, time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC))
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().ContactSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().ContactGet(gomock.Any(), gomock.Any()).Return(nil, ErrNotFound).AnyTimes()
	mockCache.EXPECT().ContactDelete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("98000000-0000-0000-0000-000000000001")
	ownerA := uuid.FromStringOrNil("98000000-0000-0000-0000-000000000002")
	ownerB := uuid.FromStringOrNil("98000000-0000-0000-0000-000000000003")
	claimer := uuid.FromStringOrNil("98000000-0000-0000-0000-000000000004")
	target := "+15550008008"

	createTestContactNoTimeNow(t, h, ctx, customerID, ownerA)
	createTestContactNoTimeNow(t, h, ctx, customerID, ownerB)
	createTestContactNoTimeNow(t, h, ctx, customerID, claimer)

	// Era A: register, interact (strictly between A's create and
	// delete instants, guaranteed by the monotonic clock), release --
	// the OLDER, non-adjacent gap.
	addrA := uuid.FromStringOrNil("98000000-0000-0000-0000-000000000005")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrA,
		CustomerID: customerID,
		ContactID:  ownerA,
	}); err != nil {
		t.Fatalf("AddressCreate(A) error = %v", err)
	}
	interactionA := uuid.FromStringOrNil("98000000-0000-0000-0000-000000000006")
	mustCreateInteraction(t, ctx, h, interactionA, customerID, string(contact.AddressTypeTel), target,
		mockUtil.peek())
	if err := h.AddressDelete(ctx, addrA); err != nil {
		t.Fatalf("AddressDelete(A) error = %v", err)
	}
	aClosed := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(aClosed) != 1 || aClosed[0].ValidTo == nil {
		t.Fatalf("expected exactly one closed period for A, got: %+v", aClosed)
	}

	// Era B: register, release -- the IMMEDIATELY PRIOR gap (this is
	// the one the claim's valid_from will reach back to).
	addrB := uuid.FromStringOrNil("98000000-0000-0000-0000-000000000007")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrB,
		CustomerID: customerID,
		ContactID:  ownerB,
	}); err != nil {
		t.Fatalf("AddressCreate(B) error = %v", err)
	}
	if err := h.AddressDelete(ctx, addrB); err != nil {
		t.Fatalf("AddressDelete(B) error = %v", err)
	}
	bRows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	var bClosedValidTo *time.Time
	for _, p := range bRows {
		if p.ContactID == ownerB && p.ValidTo != nil {
			bClosedValidTo = p.ValidTo
		}
	}
	if bClosedValidTo == nil {
		t.Fatalf("expected a closed period for B, got: %+v", bRows)
	}

	// Unresolved era, then claimed.
	unresolvedAddrID := uuid.FromStringOrNil("98000000-0000-0000-0000-000000000008")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         unresolvedAddrID,
		CustomerID: customerID,
		ContactID:  uuid.Nil,
	}); err != nil {
		t.Fatalf("AddressCreate(unresolved) error = %v", err)
	}
	if err := h.AddressClaim(ctx, customerID, unresolvedAddrID, claimer); err != nil {
		t.Fatalf("AddressClaim() error = %v", err)
	}

	claimerPeriods, err := h.OwnershipPeriodsListByContactID(ctx, claimer)
	if err != nil {
		t.Fatalf("OwnershipPeriodsListByContactID(claimer) error = %v", err)
	}
	if len(claimerPeriods) != 1 || claimerPeriods[0].ValidFrom == nil {
		t.Fatalf("expected exactly one bounded period for the claimer, got: %+v", claimerPeriods)
	}
	if !claimerPeriods[0].ValidFrom.Equal(*bClosedValidTo) {
		t.Fatalf("claimer's valid_from = %v, want %v (B's valid_to, the IMMEDIATELY prior gap) -- test setup assumption violated, cannot proceed", claimerPeriods[0].ValidFrom, bClosedValidTo)
	}

	unresolved, err := h.InteractionListUnresolved(ctx, customerID, 100, "", time.Time{})
	if err != nil {
		t.Fatalf("InteractionListUnresolved() error = %v", err)
	}
	found := false
	for _, i := range unresolved {
		if i.ID == interactionA {
			found = true
		}
	}
	if !found {
		t.Errorf("InteractionListUnresolved() did NOT resurface the older, non-adjacent-gap interaction after claim -- round-39's accepted (documented) behavior regressed: it should resurface once the row-presence suppression that covered it unconditionally is removed by the claim")
	}
}

// newMonotonicTimeUtilHandler returns a mock UtilHandler whose TimeNow()
// advances by one minute on every call, starting at start -- unlike
// newOwnershipTestHandler's single fixed instant (AnyTimes() returning
// the same pointer always), this lets a test construct ownership periods
// with genuinely distinct, strictly increasing boundaries without having
// to predict the exact number of internal TimeNow() calls a given write
// makes (which varies by which §4 step it resolves to). peek() returns
// the timestamp the NEXT call will produce, without consuming it -- used
// by callers that need to record an interaction at a time guaranteed to
// fall strictly between two writes.
type monotonicTimeUtilHandler struct {
	*utilhandler.MockUtilHandler
	next time.Time
}

func (m *monotonicTimeUtilHandler) peek() time.Time {
	return m.next
}

func newMonotonicTimeUtilHandler(t *testing.T, mc *gomock.Controller, start time.Time) *monotonicTimeUtilHandler {
	t.Helper()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	m := &monotonicTimeUtilHandler{MockUtilHandler: mockUtil, next: start}
	mockUtil.EXPECT().TimeNow().DoAndReturn(func() *time.Time {
		tm := m.next
		m.next = m.next.Add(time.Minute)
		return &tm
	}).AnyTimes()
	return m
}

// createTestContactNoTimeNow mirrors createTestContact but does not
// consume a TimeNow() call budget -- ContactCreate does not call
// TimeNow() (confirmed against createTestContact's existing usage
// pattern, which shares the same AnyTimes() mock as every other write in
// its test), so this exists only to make that non-consumption explicit
// at call sites using the strict, monotonic-clock newMonotonicTimeUtilHandler
// mock above, where an unexpected TimeNow() call would still succeed but
// would shift every subsequent timestamp by one unintended tick.
func createTestContactNoTimeNow(t *testing.T, h *handler, ctx context.Context, customerID, contactID uuid.UUID) {
	t.Helper()
	c := &contact.Contact{
		Identity: commonidentity.Identity{ID: contactID, CustomerID: customerID},
		Source:   "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}
}

func assertContainsIDs(t *testing.T, label string, got []*interaction.Interaction, want ...uuid.UUID) {
	t.Helper()
	gotSet := make(map[uuid.UUID]bool, len(got))
	for _, i := range got {
		gotSet[i.ID] = true
	}
	for _, w := range want {
		if !gotSet[w] {
			t.Errorf("%s missing expected interaction %v (got: %d items)", label, w, len(got))
		}
	}
}
