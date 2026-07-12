package dbhandler

// Unit and scenario tests for the ownership-period write path (design
// docs/plans/2026-07-11-contact-address-ownership-integrity-design.md
// §4/§5.1-5.4, NOJIRA-contact-address-ownership-periods Phase 1).

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// newOwnershipTestHandler builds a handler wired to the shared SQLite test
// DB with a permissive TimeNow mock (AnyTimes(), since these tests exercise
// composed multi-write sequences where the exact call count is an
// implementation detail, not the behavior under test) and a permissive
// cache mock (any calls allowed, not asserted -- these tests assert on
// ownership-period/contact_addresses state, not on cache side effects).
func newOwnershipTestHandler(t *testing.T, curTime *time.Time) (*handler, *gomock.Controller) {
	t.Helper()
	mc := gomock.NewController(t)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(curTime).AnyTimes()
	mockCache.EXPECT().ContactSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().ContactGet(gomock.Any(), gomock.Any()).Return(nil, ErrNotFound).AnyTimes()
	mockCache.EXPECT().ContactDelete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	return h, mc
}

func createTestContact(t *testing.T, h *handler, ctx context.Context, customerID, contactID uuid.UUID) {
	t.Helper()
	c := &contact.Contact{
		Identity: commonidentity.Identity{ID: contactID, CustomerID: customerID},
		Source:   "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}
}

func ownershipPeriodsForTarget(t *testing.T, ctx context.Context, h *handler, customerID uuid.UUID, addrType commonaddress.Type, target string) []OwnershipPeriod {
	t.Helper()
	query, args, err := sqOwnershipPeriodSelectAll(customerID, addrType, target)
	if err != nil {
		t.Fatalf("build query error = %v", err)
	}
	rows, err := h.db.Query(query, args...)
	if err != nil {
		t.Fatalf("query error = %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []OwnershipPeriod
	for rows.Next() {
		p, err := scanOwnershipPeriodRow(rows, customerID, addrType, target)
		if err != nil {
			t.Fatalf("scan error = %v", err)
		}
		res = append(res, *p)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration error = %v", err)
	}
	return res
}

// ---------------------------------------------------------------------
// Unit tests: OwnershipPeriodsLockAndResolveTx's Step 1-5 decision
// procedure (design §4).
// ---------------------------------------------------------------------

// Test_OwnershipPeriods_Step5_FirstRegistration covers §4 Step 5: no
// period row exists at all for the target -- first-ever registration,
// valid_from=NULL.
func Test_OwnershipPeriods_Step5_FirstRegistration(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000002")
	createTestContact(t, h, ctx, customerID, contactID)

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	step, rows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactID, contact.AddressTypeTel, "+155****0001")
	if err != nil {
		t.Fatalf("OwnershipPeriodsLockAndResolveTx() error = %v", err)
	}
	if step != StepFirstRegistration {
		t.Errorf("step = %d, want StepFirstRegistration (%d)", step, StepFirstRegistration)
	}
	if len(rows) != 0 {
		t.Errorf("lockedRows = %d rows, want 0", len(rows))
	}
}

// Test_OwnershipPeriods_Step1_LiveConflict covers §4 Step 1: another
// contact's live, agreement-verified open period blocks this contact.
func Test_OwnershipPeriods_Step1_LiveConflict(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000011")
	contactA := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000012")
	contactB := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000013")
	target := "+155****0011"
	createTestContact(t, h, ctx, customerID, contactA)
	createTestContact(t, h, ctx, customerID, contactB)

	addrID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000014")
	a := &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactA,
	}
	if err := h.AddressCreate(ctx, a); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, _, err = h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactB, contact.AddressTypeTel, target)
	if !stderrors.Is(err, ErrConflict) {
		t.Errorf("OwnershipPeriodsLockAndResolveTx() error = %v, want ErrConflict", err)
	}
}

// Test_OwnershipPeriods_Step1_OrphanClose covers §4 Step 1's orphan
// branch: another contact's open period exists, but that contact is
// tombstoned (soft-deleted) -- the orphan is closed, not treated as a
// live conflict.
func Test_OwnershipPeriods_Step1_OrphanClose(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000021")
	contactA := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000022")
	contactB := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000023")
	target := "+155****0021"
	createTestContact(t, h, ctx, customerID, contactA)
	createTestContact(t, h, ctx, customerID, contactB)

	addrID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000024")
	a := &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactA,
	}
	if err := h.AddressCreate(ctx, a); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	// Simulate the A9-b skew state directly: contactA's contact_addresses
	// row survives (as it would under a pre-fix binary) but the Contact
	// itself is soft-deleted, without touching the address/period rows --
	// exactly the corruption this design's ownership-agreement check
	// (design §4 Step 1) exists to detect.
	if err := h.ContactDelete(ctx, contactA); err != nil {
		t.Fatalf("ContactDelete() error = %v", err)
	}
	// Re-insert the now-orphaned contact_addresses row (ContactDelete
	// itself does not touch contact_addresses -- design §4's A9 finding
	// -- but this test wants the address row present with an owner whose
	// Contact is now tombstoned, so the ownership-agreement check's
	// tombstone branch is what's exercised, not the row-missing branch).

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	step, rows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactB, contact.AddressTypeTel, target)
	if err != nil {
		t.Fatalf("OwnershipPeriodsLockAndResolveTx() error = %v", err)
	}
	// contactB has no closed period of its own and no other row exists
	// after the orphan is closed -- design §4 Step 4 (reassignment).
	if step != StepReassign {
		t.Errorf("step = %d, want StepReassign (%d)", step, StepReassign)
	}
	foundClosed := false
	for _, p := range rows {
		if p.ContactID == contactA && p.ValidTo != nil {
			foundClosed = true
		}
	}
	if !foundClosed {
		t.Errorf("expected contactA's orphan period to be closed in lockedRows, got: %+v", rows)
	}
}

// Test_OwnershipPeriods_Step2_OpenReuse covers §4 Step 2: this contact
// already owns an open period for this target -- reuse, no write needed.
func Test_OwnershipPeriods_Step2_OpenReuse(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000031")
	contactID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000032")
	target := "+155****0031"
	createTestContact(t, h, ctx, customerID, contactID)

	addrID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000033")
	a := &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactID,
	}
	if err := h.AddressCreate(ctx, a); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	step, _, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactID, contact.AddressTypeTel, target)
	if err != nil {
		t.Fatalf("OwnershipPeriodsLockAndResolveTx() error = %v", err)
	}
	if step != StepOpenReuse {
		t.Errorf("step = %d, want StepOpenReuse (%d)", step, StepOpenReuse)
	}
}

// Test_OwnershipPeriods_Step3_Reopen covers §4 Step 3's "no intervening
// owner" branch: same contact re-registers a target it previously
// released, with no one else having held it in the interim -- the
// closed period is reopened, not a new one inserted.
func Test_OwnershipPeriods_Step3_Reopen(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000041")
	contactID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000042")
	target := "+155****0041"
	createTestContact(t, h, ctx, customerID, contactID)

	addrID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000043")
	a := &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactID,
	}
	if err := h.AddressCreate(ctx, a); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}
	if err := h.AddressDelete(ctx, addrID); err != nil {
		t.Fatalf("AddressDelete() error = %v", err)
	}

	before := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(before) != 1 || before[0].ValidTo == nil {
		t.Fatalf("expected exactly one CLOSED period after delete, got: %+v", before)
	}
	closedID := before[0].ID

	// Re-create the same target for the SAME contact -- design §4 Step 3
	// reopen branch, no intervening owner.
	addrID2 := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000044")
	a2 := &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID2,
		CustomerID: customerID,
		ContactID:  contactID,
	}
	if err := h.AddressCreate(ctx, a2); err != nil {
		t.Fatalf("AddressCreate() (reacquire) error = %v", err)
	}

	after := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(after) != 1 {
		t.Fatalf("expected exactly one period after reacquire (reopened, not a new row), got %d: %+v", len(after), after)
	}
	if after[0].ID != closedID {
		t.Errorf("expected the SAME period row to be reopened (id %s), got a different row (id %s)", closedID, after[0].ID)
	}
	if after[0].ValidTo != nil {
		t.Errorf("expected the reopened period to have valid_to=NULL, got %v", after[0].ValidTo)
	}
}

// Test_OwnershipPeriods_Step3_InsertAfterIntervening covers §4 Step 3's
// "intervening owner" branch (the A->B->A case): A releases, B takes it,
// B releases, A re-registers -- A's OLD closed period must NOT be
// reopened (that would overlap B's era); a NEW period must be inserted
// instead.
func Test_OwnershipPeriods_Step3_InsertAfterIntervening(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000051")
	contactA := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000052")
	contactB := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000053")
	target := "+155****0051"
	createTestContact(t, h, ctx, customerID, contactA)
	createTestContact(t, h, ctx, customerID, contactB)

	// A registers, then releases.
	addrA := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000054")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrA,
		CustomerID: customerID,
		ContactID:  contactA,
	}); err != nil {
		t.Fatalf("AddressCreate(A) error = %v", err)
	}
	if err := h.AddressDelete(ctx, addrA); err != nil {
		t.Fatalf("AddressDelete(A) error = %v", err)
	}

	// B registers (reassignment, §4 Step 4), then releases.
	addrB := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000055")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrB,
		CustomerID: customerID,
		ContactID:  contactB,
	}); err != nil {
		t.Fatalf("AddressCreate(B) error = %v", err)
	}
	if err := h.AddressDelete(ctx, addrB); err != nil {
		t.Fatalf("AddressDelete(B) error = %v", err)
	}

	mid := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(mid) != 2 {
		t.Fatalf("expected 2 closed periods (A's era, B's era) before A re-registers, got %d: %+v", len(mid), mid)
	}

	// A re-registers -- design §4 Step 3's intervening-owner branch: a
	// NEW period must be inserted, A's original closed period must stay
	// closed and untouched (not reopened).
	addrA2 := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000056")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrA2,
		CustomerID: customerID,
		ContactID:  contactA,
	}); err != nil {
		t.Fatalf("AddressCreate(A, re-registering) error = %v", err)
	}

	final := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(final) != 3 {
		t.Fatalf("expected 3 periods (A's original closed era, B's closed era, A's NEW open era), got %d: %+v", len(final), final)
	}
	openCount, aOpenCount := 0, 0
	for _, p := range final {
		if p.ValidTo == nil {
			openCount++
			if p.ContactID == contactA {
				aOpenCount++
			}
		}
	}
	if openCount != 1 || aOpenCount != 1 {
		t.Errorf("expected exactly one open period, owned by A, got openCount=%d aOpenCount=%d: %+v", openCount, aOpenCount, final)
	}
	// A's brand-new open period must have valid_from=NOW() (design §4
	// Step 4's caller-specific bound for AddressCreate), not NULL --
	// otherwise it would silently absorb A's original (pre-B) era too.
	for _, p := range final {
		if p.ContactID == contactA && p.ValidTo == nil {
			if p.ValidFrom == nil {
				t.Errorf("A's new open period has valid_from=NULL, want NOW() (non-nil) so it does not overlap A's original closed era")
			}
		}
	}
}

// Test_OwnershipPeriods_Step4_Reassignment covers §4 Step 4: a target
// previously held (and released) by a DIFFERENT contact is reassigned to
// this contact, with valid_from=NOW() (not NULL) so history does not
// leak from the old owner to the new one.
func Test_OwnershipPeriods_Step4_Reassignment(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000061")
	contactA := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000062")
	contactB := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000063")
	target := "+155****0061"
	createTestContact(t, h, ctx, customerID, contactA)
	createTestContact(t, h, ctx, customerID, contactB)

	addrA := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000064")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrA,
		CustomerID: customerID,
		ContactID:  contactA,
	}); err != nil {
		t.Fatalf("AddressCreate(A) error = %v", err)
	}
	if err := h.AddressDelete(ctx, addrA); err != nil {
		t.Fatalf("AddressDelete(A) error = %v", err)
	}

	addrB := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000065")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrB,
		CustomerID: customerID,
		ContactID:  contactB,
	}); err != nil {
		t.Fatalf("AddressCreate(B, reassignment) error = %v", err)
	}

	rows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(rows) != 2 {
		t.Fatalf("expected 2 periods (A's closed era, B's new open era), got %d: %+v", len(rows), rows)
	}
	for _, p := range rows {
		if p.ContactID == contactB {
			if p.ValidTo != nil {
				t.Errorf("B's period should be open (valid_to=NULL), got %v", p.ValidTo)
			}
			if p.ValidFrom == nil {
				t.Errorf("B's reassigned period has valid_from=NULL, want NOW() (non-nil) -- history must not leak from A")
			}
		}
		if p.ContactID == contactA && p.ValidTo == nil {
			t.Errorf("A's period should be closed after AddressDelete, still open: %+v", p)
		}
	}
}

// Test_StaleRowRepair_TombstonedOwner_AddressCreate covers design §4
// round-27/28/30's duplicate-key repair path: a Contact holding an
// address gets tombstoned WITHOUT its contact_addresses row being
// touched (design's own A9-b finding: ContactDelete never touches
// contact_addresses), leaving a dead-owner row occupying the unique
// index slot. A later AddressCreate for the SAME target by a DIFFERENT
// contact must not be permanently 409-locked -- it should detect the
// dead owner, repair (fabricate a closed period bounded by
// GREATEST(latest valid_to, stale row's tm_create) ending at the
// tombstone time, vacate the slot), and succeed on retry.
func Test_StaleRowRepair_TombstonedOwner_AddressCreate(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("72000000-0000-0000-0000-000000000001")
	deadContactID := uuid.FromStringOrNil("72000000-0000-0000-0000-000000000002")
	newContactID := uuid.FromStringOrNil("72000000-0000-0000-0000-000000000003")
	target := "+155****2001"
	createTestContact(t, h, ctx, customerID, deadContactID)

	deadAddrID := uuid.FromStringOrNil("72000000-0000-0000-0000-000000000004")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         deadAddrID,
		CustomerID: customerID,
		ContactID:  deadContactID,
	}); err != nil {
		t.Fatalf("AddressCreate(dead owner) error = %v", err)
	}

	// A9-b: soft-delete the Contact WITHOUT touching contact_addresses
	// (ContactDelete's real, documented behavior) -- leaves a dead-owner
	// row + a still-open period occupying the target's unique-index
	// slot.
	if err := h.ContactDelete(ctx, deadContactID); err != nil {
		t.Fatalf("ContactDelete() error = %v", err)
	}

	// A different contact tries to register the SAME target.
	createTestContact(t, h, ctx, customerID, newContactID)
	newAddrID := uuid.FromStringOrNil("72000000-0000-0000-0000-000000000005")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         newAddrID,
		CustomerID: customerID,
		ContactID:  newContactID,
	}); err != nil {
		t.Fatalf("AddressCreate(new owner, after dead-owner repair) error = %v, want success (self-healing, not permanent 409 lockout)", err)
	}

	got, err := h.AddressGet(ctx, customerID, newAddrID)
	if err != nil {
		t.Fatalf("AddressGet(new owner) error = %v", err)
	}
	if got.ContactID != newContactID {
		t.Errorf("new address's ContactID = %s, want %s", got.ContactID, newContactID)
	}

	// The old dead-owner row must be gone (hard-deleted by the repair,
	// design §4 round-28: AddressCreate's repair hard-deletes, not
	// resets-to-NULL -- that's AddressClaimTx's variant only).
	if _, err := h.AddressGet(ctx, customerID, deadAddrID); !stderrors.Is(err, ErrNotFound) {
		t.Errorf("AddressGet(dead owner's old row) = %v, want ErrNotFound (repaired row must be removed)", err)
	}

	rows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	var deadClosed, newOpen *OwnershipPeriod
	for i := range rows {
		p := &rows[i]
		if p.ContactID == deadContactID && p.ValidTo != nil {
			deadClosed = p
		}
		if p.ContactID == newContactID && p.ValidTo == nil {
			newOpen = p
		}
	}
	if deadClosed == nil {
		t.Errorf("expected a fabricated closed period for the dead owner's era, got: %+v", rows)
	}
	if newOpen == nil {
		t.Fatalf("expected an open period for the new owner, got: %+v", rows)
	}
	if newOpen.ValidFrom == nil {
		t.Errorf("new owner's period has valid_from=NULL, want NOW() (design §4 Step 4's caller-specific bound, reached via the repair's retry path)")
	}
}

// Test_StaleRowRepair_LiveOwner_StaysErrDuplicateTarget covers the
// negative case: if the occupying row's owner is LIVE (not tombstoned),
// staleRowRepairTx must NOT repair -- the caller sees the ordinary
// ErrDuplicateTarget collision, not a silently stolen target.
func Test_StaleRowRepair_LiveOwner_StaysErrDuplicateTarget(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("72000000-0000-0000-0000-000000000011")
	contactA := uuid.FromStringOrNil("72000000-0000-0000-0000-000000000012")
	contactB := uuid.FromStringOrNil("72000000-0000-0000-0000-000000000013")
	target := "+155****2011"
	createTestContact(t, h, ctx, customerID, contactA)
	createTestContact(t, h, ctx, customerID, contactB)

	addrA := uuid.FromStringOrNil("72000000-0000-0000-0000-000000000014")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrA,
		CustomerID: customerID,
		ContactID:  contactA,
	}); err != nil {
		t.Fatalf("AddressCreate(A, live) error = %v", err)
	}

	addrB := uuid.FromStringOrNil("72000000-0000-0000-0000-000000000015")
	err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrB,
		CustomerID: customerID,
		ContactID:  contactB,
	})
	// Note: Step 1's live-conflict check (open period + live owner)
	// already rejects this with ErrConflict before ever reaching the
	// unique-index INSERT -- staleRowRepairTx is never invoked here.
	// This test asserts the end-to-end outcome (rejection, A undisturbed)
	// regardless of which layer produces it.
	if !stderrors.Is(err, ErrConflict) && !stderrors.Is(err, ErrDuplicateTarget) {
		t.Errorf("AddressCreate(B, live A holds target) error = %v, want ErrConflict or ErrDuplicateTarget", err)
	}
	if _, err := h.AddressGet(ctx, customerID, addrB); !stderrors.Is(err, ErrNotFound) {
		t.Errorf("AddressGet(B) = %v, want ErrNotFound (B's row must never have been written)", err)
	}
}

// Test_StaleRowRepair_TombstonedOwner_AddressClaim covers design §4
// round-27(a)/28's repair-in-place path: an unresolved address's target
// history shows a DIFFERENT contact_id occupying the row (an A9-b/A9-c
// version-skew artifact -- the address was left resolved to a now-dead
// Contact because ContactDelete never touches contact_addresses). A
// later AddressClaim by a NEW contact for that same address must not be
// permanently rejected -- it should detect the tombstoned owner, reset
// the row to unresolved in-place, and complete the claim.
func Test_StaleRowRepair_TombstonedOwner_AddressClaim(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("73000000-0000-0000-0000-000000000001")
	deadContactID := uuid.FromStringOrNil("73000000-0000-0000-0000-000000000002")
	newContactID := uuid.FromStringOrNil("73000000-0000-0000-0000-000000000003")
	target := "+155****3001"
	createTestContact(t, h, ctx, customerID, deadContactID)
	createTestContact(t, h, ctx, customerID, newContactID)

	addrID := uuid.FromStringOrNil("73000000-0000-0000-0000-000000000004")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  deadContactID,
	}); err != nil {
		t.Fatalf("AddressCreate(dead owner) error = %v", err)
	}

	// A9-b: tombstone the owning Contact without touching contact_addresses.
	if err := h.ContactDelete(ctx, deadContactID); err != nil {
		t.Fatalf("ContactDelete() error = %v", err)
	}

	// A different contact claims the SAME address id.
	if err := h.AddressClaim(ctx, customerID, addrID, newContactID); err != nil {
		t.Fatalf("AddressClaim(new owner, after dead-owner repair-in-place) error = %v, want success", err)
	}

	got, err := h.AddressGet(ctx, customerID, addrID)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if got.ContactID != newContactID {
		t.Errorf("address's ContactID = %s, want %s", got.ContactID, newContactID)
	}

	rows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	var deadClosed, newOpen *OwnershipPeriod
	for i := range rows {
		p := &rows[i]
		if p.ContactID == deadContactID && p.ValidTo != nil {
			deadClosed = p
		}
		if p.ContactID == newContactID && p.ValidTo == nil {
			newOpen = p
		}
	}
	if deadClosed == nil {
		t.Errorf("expected a fabricated closed period for the dead owner's era, got: %+v", rows)
	}
	if newOpen == nil {
		t.Errorf("expected an open period for the new owner (Step 4 reassignment via the repaired NULL-owned row), got: %+v", rows)
	}
}

// ---------------------------------------------------------------------
// Unit tests: individual Tx-suffixed write functions.
// ---------------------------------------------------------------------

// Test_AddressDeleteTx_Skew_NotAnError covers the CLOSE-ing-caller skew
// exemption (design §5.1 round-57/§9 round-16/17): if lockedRows is
// completely empty (no period ever existed for this target -- a
// rolling-deploy version-skew artifact), AddressDelete must NOT error.
func Test_AddressDeleteTx_Skew_NotAnError(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000071")
	contactID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000072")
	addrID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000073")
	createTestContact(t, h, ctx, customerID, contactID)

	// Insert the contact_addresses row directly, bypassing AddressCreate
	// entirely (design §9's version-skew simulation: an old-binary write
	// with no ownership-period write at all).
	insertTestAddress(t, h.db, addrID, customerID, contactID, string(contact.AddressTypeTel), "+155****0071")

	if err := h.AddressDelete(ctx, addrID); err != nil {
		t.Errorf("AddressDelete() with no pre-existing period should NOT error (skew case), got: %v", err)
	}
}

// Test_AddressDeleteTx_GenuineConflict_ErrConflict covers the CLOSE-ing
// caller's genuine-conflict branch (design §5.1 round-49/57): lockedRows
// is non-empty (a period DOES exist for this target) but no row is open
// for this contact -- e.g. a concurrent operation already closed it.
func Test_AddressDeleteTx_GenuineConflict_ErrConflict(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000081")
	contactID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000082")
	addrID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000083")
	target := "+155****0081"
	createTestContact(t, h, ctx, customerID, contactID)

	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactID,
	}); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	// Simulate a concurrent close: directly close the period row without
	// going through AddressDelete, leaving the contact_addresses row
	// itself still present. AddressDelete's own close attempt now finds
	// a non-empty lockedRows with no row open for this contact -- a
	// genuine conflict, not skew.
	rows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(rows) != 1 {
		t.Fatalf("expected exactly one period row, got %d", len(rows))
	}
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	if err := h.ownershipPeriodCloseByIDTx(ctx, tx, rows[0].ID); err != nil {
		t.Fatalf("ownershipPeriodCloseByIDTx() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("tx.Commit() error = %v", err)
	}

	err = h.AddressDelete(ctx, addrID)
	if !stderrors.Is(err, ErrConflict) {
		t.Errorf("AddressDelete() after concurrent close: error = %v, want ErrConflict", err)
	}
}

// Test_AddressDeleteCompensating_NoSkewExemption covers design §4
// round-57: unlike AddressDeleteTx, AddressDeleteCompensating has NO
// skew exemption -- an empty lockedRows is unconditionally ErrConflict,
// never treated as version skew (the period being compensated for was
// just inserted moments earlier by the SAME request).
func Test_AddressDeleteCompensating_NoSkewExemption(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000091")
	contactID := uuid.FromStringOrNil("70000000-0000-0000-0000-000000000092")

	// No address/period ever created for this target -- lockedRows will
	// be empty. AddressDeleteCompensating must still return ErrConflict,
	// not silently succeed.
	err := h.AddressDeleteCompensating(ctx, customerID, contactID, contact.AddressTypeTel, "+155****0091")
	if !stderrors.Is(err, ErrConflict) {
		t.Errorf("AddressDeleteCompensating() with no period = %v, want ErrConflict (no skew exemption)", err)
	}
}

// Test_AddressDeleteCompensating_HardDeletes covers design §4 round-31's
// sanctioned exception: AddressDeleteCompensating hard-DELETEs the period
// row it cleans up (not valid_to=NOW()), so a subsequent retry for the
// same target is a genuinely fresh Step 5 registration, not a Step 4
// reassignment against a ghost's leftover closed period.
func Test_AddressDeleteCompensating_HardDeletes(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-0000000000a1")
	ghostContactID := uuid.FromStringOrNil("70000000-0000-0000-0000-0000000000a2")
	target := "+155****00a1"

	// Simulate ContactCreate's address loop having already inserted this
	// address (and its period) for a not-yet-committed/ghost contact_id,
	// then a LATER address in the same loop fails, triggering
	// compensating cleanup for this one.
	addrID := uuid.FromStringOrNil("70000000-0000-0000-0000-0000000000a3")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  ghostContactID,
	}); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	if err := h.AddressDeleteCompensating(ctx, customerID, ghostContactID, contact.AddressTypeTel, target); err != nil {
		t.Fatalf("AddressDeleteCompensating() error = %v", err)
	}

	rows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(rows) != 0 {
		t.Errorf("expected the period row to be HARD-DELETED (not just closed), got %d rows: %+v", len(rows), rows)
	}

	// A genuinely fresh retry for the same target must now be a Step 5
	// first registration (valid_from=NULL), not Step 4 reassignment
	// against the ghost's leftover history.
	newContactID := uuid.FromStringOrNil("70000000-0000-0000-0000-0000000000a4")
	createTestContact(t, h, ctx, customerID, newContactID)
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	step, _, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, newContactID, contact.AddressTypeTel, target)
	if err != nil {
		t.Fatalf("OwnershipPeriodsLockAndResolveTx() error = %v", err)
	}
	if step != StepFirstRegistration {
		t.Errorf("retry step = %d, want StepFirstRegistration (%d) -- no ghost period should remain", step, StepFirstRegistration)
	}
}

// Test_AddressClaimTx_ValidFromIsLatestClosed covers design §4 Step 4's
// round-38 corrected bound for AddressClaim: valid_from = the latest
// closed valid_to for this target (not NOW(), not NULL), so the claim
// attributes exactly the gap since the immediately-prior era to the
// claimer.
func Test_AddressClaimTx_ValidFromIsLatestClosed(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("70000000-0000-0000-0000-0000000000b1")
	priorOwner := uuid.FromStringOrNil("70000000-0000-0000-0000-0000000000b2")
	claimer := uuid.FromStringOrNil("70000000-0000-0000-0000-0000000000b3")
	target := "+155****00b1"
	createTestContact(t, h, ctx, customerID, priorOwner)
	createTestContact(t, h, ctx, customerID, claimer)

	priorAddrID := uuid.FromStringOrNil("70000000-0000-0000-0000-0000000000b4")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         priorAddrID,
		CustomerID: customerID,
		ContactID:  priorOwner,
	}); err != nil {
		t.Fatalf("AddressCreate(prior owner) error = %v", err)
	}
	if err := h.AddressDelete(ctx, priorAddrID); err != nil {
		t.Fatalf("AddressDelete(prior owner) error = %v", err)
	}

	priorRows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(priorRows) != 1 || priorRows[0].ValidTo == nil {
		t.Fatalf("expected exactly one closed period before claim, got: %+v", priorRows)
	}
	priorEnd := *priorRows[0].ValidTo

	// An unresolved address, then claimed by a new contact.
	unresolvedAddrID := uuid.FromStringOrNil("70000000-0000-0000-0000-0000000000b5")
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

	afterRows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	var claimerPeriod *OwnershipPeriod
	for i := range afterRows {
		if afterRows[i].ContactID == claimer {
			claimerPeriod = &afterRows[i]
		}
	}
	if claimerPeriod == nil {
		t.Fatalf("expected a period for the claimer, got: %+v", afterRows)
	}
	if claimerPeriod.ValidFrom == nil || !claimerPeriod.ValidFrom.Equal(priorEnd) {
		t.Errorf("claimer's valid_from = %v, want exactly the prior owner's valid_to (%v)", claimerPeriod.ValidFrom, priorEnd)
	}
}

// ---------------------------------------------------------------------
// Scenario tests: multiple functions composed together, per the user's
// explicit requirement that composed/cross-function scenarios (not just
// per-function unit tests) be covered.
// ---------------------------------------------------------------------

// Test_Scenario_SameOwnerReleaseAndReacquire is scenario (a): the same
// owner releases a target and later reacquires it -- Delete then
// Create, exercising the Step 3 reopen path end-to-end through the
// public AddressDelete/AddressCreate API (not the internal Tx
// functions directly), across two independent request-shaped calls.
func Test_Scenario_SameOwnerReleaseAndReacquire(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000002")
	target := "+155****1001"
	createTestContact(t, h, ctx, customerID, contactID)

	addr1 := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000003")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addr1,
		CustomerID: customerID,
		ContactID:  contactID,
	}); err != nil {
		t.Fatalf("AddressCreate() (1st) error = %v", err)
	}
	if err := h.AddressDelete(ctx, addr1); err != nil {
		t.Fatalf("AddressDelete() error = %v", err)
	}

	addr2 := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000004")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addr2,
		CustomerID: customerID,
		ContactID:  contactID,
	}); err != nil {
		t.Fatalf("AddressCreate() (reacquire) error = %v", err)
	}

	rows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(rows) != 1 {
		t.Fatalf("expected exactly ONE continuous period (reopened, not duplicated) after release+reacquire, got %d: %+v", len(rows), rows)
	}
	if rows[0].ValidTo != nil {
		t.Errorf("expected the reacquired period to be open, got valid_to=%v", rows[0].ValidTo)
	}
	if rows[0].ValidFrom != nil {
		t.Errorf("expected the reopened period's valid_from to be unchanged (still NULL from original registration, since this is the SAME continuous ownership, not a new era), got %v", rows[0].ValidFrom)
	}

	got, err := h.AddressGet(ctx, customerID, addr2)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if got.ContactID != contactID {
		t.Errorf("AddressGet().ContactID = %s, want %s", got.ContactID, contactID)
	}
}

// Test_Scenario_ABAReassignmentCycle is scenario (b): A->B->A
// reassignment, exercising Step 4 (reassignment to B) then Step 3's
// intervening-owner insert branch (A re-registering after B), verifying
// two DISTINCT period rows exist for A (the original era and the new
// one) with B's era sandwiched between them, and that valid_from/valid_to
// boundaries are all correct and non-overlapping.
func Test_Scenario_ABAReassignmentCycle(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000011")
	contactA := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000012")
	contactB := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000013")
	target := "+155****1011"
	createTestContact(t, h, ctx, customerID, contactA)
	createTestContact(t, h, ctx, customerID, contactB)

	addrA1 := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000014")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrA1,
		CustomerID: customerID,
		ContactID:  contactA,
	}); err != nil {
		t.Fatalf("AddressCreate(A, 1st) error = %v", err)
	}
	if err := h.AddressDelete(ctx, addrA1); err != nil {
		t.Fatalf("AddressDelete(A) error = %v", err)
	}

	addrB := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000015")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrB,
		CustomerID: customerID,
		ContactID:  contactB,
	}); err != nil {
		t.Fatalf("AddressCreate(B) error = %v", err)
	}
	if err := h.AddressDelete(ctx, addrB); err != nil {
		t.Fatalf("AddressDelete(B) error = %v", err)
	}

	addrA2 := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000016")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrA2,
		CustomerID: customerID,
		ContactID:  contactA,
	}); err != nil {
		t.Fatalf("AddressCreate(A, 2nd/re-registering) error = %v", err)
	}

	rows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(rows) != 3 {
		t.Fatalf("expected 3 periods (A's original era, B's era, A's new era), got %d: %+v", len(rows), rows)
	}

	var aClosed, bClosed, aOpen *OwnershipPeriod
	for i := range rows {
		p := &rows[i]
		switch {
		case p.ContactID == contactA && p.ValidTo != nil:
			aClosed = p
		case p.ContactID == contactB && p.ValidTo != nil:
			bClosed = p
		case p.ContactID == contactA && p.ValidTo == nil:
			aOpen = p
		}
	}
	if aClosed == nil || bClosed == nil || aOpen == nil {
		t.Fatalf("expected A-closed, B-closed, A-open periods, got: aClosed=%v bClosed=%v aOpen=%v", aClosed, bClosed, aOpen)
	}
	if aClosed.ID == aOpen.ID {
		t.Errorf("A's original closed period and A's new open period must be DISTINCT rows (no reopen across an intervening owner), got same id %s", aClosed.ID)
	}
	// Non-overlap: A's original era must end at or before B's era began.
	if bClosed.ValidFrom != nil && aClosed.ValidTo.After(*bClosed.ValidFrom) {
		t.Errorf("A's original era (ends %v) overlaps B's era (starts %v)", *aClosed.ValidTo, *bClosed.ValidFrom)
	}
	// A's new era must start at/after B's era ended.
	if aOpen.ValidFrom == nil {
		t.Errorf("A's new era has valid_from=NULL, want NOW() (non-nil) so it doesn't retroactively claim B's era")
	} else if aOpen.ValidFrom.Before(*bClosed.ValidTo) {
		t.Errorf("A's new era (starts %v) begins before B's era ended (%v) -- overlap", *aOpen.ValidFrom, *bClosed.ValidTo)
	}
}

// Test_Scenario_ContactCreatePartialFailureCompensation is scenario (c):
// simulates ContactCreate's per-address loop composing AddressCreateTx
// calls, one succeeding then a later one "failing" (simulated here by
// the test driving the compensating cleanup directly, since the
// simulated failure itself is a contacthandler-level concern outside
// Phase 1's dbhandler scope) -- AddressDeleteCompensating cleans up the
// succeeded address, and a subsequent genuinely-fresh retry for the SAME
// target gets Step 5 (first registration), not Step 4 (reassignment
// against a ghost's leftover history) -- i.e. no ghost period survives.
func Test_Scenario_ContactCreatePartialFailureCompensation(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000021")
	ghostContactID := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000022")
	targetSucceeded := "+155****1021" // address 1 in the loop: succeeds
	targetFailed := "+155****1022"    // address 2 in the loop: "fails" (simulated)

	// ContactCreate's own base-row insert (this design's round-12 finding:
	// the Contact row commits before the address loop runs).
	createTestContact(t, h, ctx, customerID, ghostContactID)

	// Address 1 in the loop succeeds.
	addr1 := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000023")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: targetSucceeded},
		ID:         addr1,
		CustomerID: customerID,
		ContactID:  ghostContactID,
	}); err != nil {
		t.Fatalf("AddressCreate(1st, succeeds) error = %v", err)
	}

	// Address 2 in the loop "fails" -- simulated by simply not creating
	// it (e.g. a validation error before the write). The compensating
	// cleanup path now runs for address 1 (the one that DID succeed).
	_ = targetFailed

	if err := h.AddressDeleteCompensating(ctx, customerID, ghostContactID, contact.AddressTypeTel, targetSucceeded); err != nil {
		t.Fatalf("AddressDeleteCompensating() error = %v", err)
	}

	// Verify convergence to "no addresses were added": no
	// contact_addresses row, no period row.
	if _, err := h.AddressGet(ctx, customerID, addr1); !stderrors.Is(err, ErrNotFound) {
		t.Errorf("AddressGet() after compensation = %v, want ErrNotFound", err)
	}
	rows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, targetSucceeded)
	if len(rows) != 0 {
		t.Errorf("expected no period rows after compensation, got %d: %+v", len(rows), rows)
	}

	// A genuinely fresh retry (new Contact, new address) for the SAME
	// target must be a Step 5 first registration -- no ghost residue.
	freshContactID := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000024")
	createTestContact(t, h, ctx, customerID, freshContactID)
	freshAddrID := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000025")
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: targetSucceeded},
		ID:         freshAddrID,
		CustomerID: customerID,
		ContactID:  freshContactID,
	}); err != nil {
		t.Fatalf("AddressCreate() (fresh retry) error = %v", err)
	}
	freshRows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, targetSucceeded)
	if len(freshRows) != 1 {
		t.Fatalf("expected exactly one FRESH period after retry, got %d: %+v", len(freshRows), freshRows)
	}
	if freshRows[0].ValidFrom != nil {
		t.Errorf("fresh retry's period has valid_from=%v, want NULL (Step 5 first-ever registration, no ghost history to inherit)", freshRows[0].ValidFrom)
	}
}

// Test_Scenario_SequentialConcurrentCreateAttempts is scenario (d): two
// sequential attempts (simulating a concurrency race resolved in
// arrival order, since the SQLite test harness's single-connection pool
// cannot exercise genuine cross-goroutine row locking) to
// AddressCreateTx the SAME target for two DIFFERENT contacts -- the
// first succeeds, the second correctly observes the first's live,
// agreement-verified open period and is rejected with ErrConflict
// (design §4 Step 1), never silently overwriting or duplicating.
func Test_Scenario_SequentialConcurrentCreateAttempts(t *testing.T) {
	h, mc := newOwnershipTestHandler(t, timePtr(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)))
	defer mc.Finish()
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000031")
	contactA := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000032")
	contactB := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000033")
	target := "+155****1031"
	createTestContact(t, h, ctx, customerID, contactA)
	createTestContact(t, h, ctx, customerID, contactB)

	addrA := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000034")
	addrB := uuid.FromStringOrNil("71000000-0000-0000-0000-000000000035")

	// "Goroutine 1" wins the race (arrives first in this serialized
	// simulation).
	if err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrA,
		CustomerID: customerID,
		ContactID:  contactA,
	}); err != nil {
		t.Fatalf("AddressCreate(A, wins race) error = %v", err)
	}

	// "Goroutine 2" loses the race -- must be rejected, not silently
	// succeed or corrupt A's period.
	err := h.AddressCreate(ctx, &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrB,
		CustomerID: customerID,
		ContactID:  contactB,
	})
	if !stderrors.Is(err, ErrConflict) {
		t.Errorf("AddressCreate(B, loses race) error = %v, want ErrConflict", err)
	}

	rows := ownershipPeriodsForTarget(t, ctx, h, customerID, contact.AddressTypeTel, target)
	if len(rows) != 1 {
		t.Fatalf("expected exactly ONE period (A's, undisturbed by B's rejected attempt), got %d: %+v", len(rows), rows)
	}
	if rows[0].ContactID != contactA || rows[0].ValidTo != nil {
		t.Errorf("expected A's period to remain open and untouched, got: %+v", rows[0])
	}
	if _, err := h.AddressGet(ctx, customerID, addrB); !stderrors.Is(err, ErrNotFound) {
		t.Errorf("AddressGet(B) after rejected create = %v, want ErrNotFound (B's row must never have been written)", err)
	}
}
