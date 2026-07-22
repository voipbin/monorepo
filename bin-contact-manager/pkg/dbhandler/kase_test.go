package dbhandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_CaseInsert_And_CaseGetByID verifies the basic insert + unlocked-read
// round-trip for the Case entity (Task 3.2), against the real SQLite
// in-memory contact_cases schema.
func Test_CaseInsert_And_CaseGetByID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5001-5001-5001-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5001-5001-5001-000000000002")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	c := &kase.Case{
		ID:            caseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110001"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}

	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	res, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if res.ID != caseID {
		t.Errorf("wrong ID. expect: %s, got: %s", caseID, res.ID)
	}
	if res.CustomerID != customerID {
		t.Errorf("wrong CustomerID. expect: %s, got: %s", customerID, res.CustomerID)
	}
	if res.Status != kase.StatusOpen {
		t.Errorf("wrong Status. expect: %s, got: %s", kase.StatusOpen, res.Status)
	}
	if res.Peer.Target != "+15551110001" {
		t.Errorf("wrong PeerTarget: %s", res.Peer.Target)
	}
}

// Test_CaseUpdateTagIDs_SetOverwriteClear verifies design VOIP-1254's
// CaseUpdateTagIDs round-trip: an initial nil TagIDs, a set of two
// tags, an overwrite to a single different tag, and a clear back to an
// empty slice -- each verified via a fresh CaseGetByID read.
func Test_CaseUpdateTagIDs_SetOverwriteClear(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5090-5090-5090-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5090-5090-5090-000000000002")
	tagID1 := uuid.FromStringOrNil("f1b2c3d4-5090-5090-5090-000000000003")
	tagID2 := uuid.FromStringOrNil("f1b2c3d4-5090-5090-5090-000000000004")
	tagID3 := uuid.FromStringOrNil("f1b2c3d4-5090-5090-5090-000000000005")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	c := &kase.Case{
		ID:            caseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110010"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	// Initial state: nil TagIDs.
	res, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() (initial) error = %v", err)
	}
	if len(res.TagIDs) != 0 {
		t.Errorf("expected empty TagIDs initially, got: %v", res.TagIDs)
	}

	// Set: two tags.
	if err := h.CaseUpdateTagIDs(ctx, customerID, caseID, []uuid.UUID{tagID1, tagID2}); err != nil {
		t.Fatalf("CaseUpdateTagIDs() (set) error = %v", err)
	}
	res, err = h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() (after set) error = %v", err)
	}
	if len(res.TagIDs) != 2 || res.TagIDs[0] != tagID1 || res.TagIDs[1] != tagID2 {
		t.Errorf("expected [%s %s], got: %v", tagID1, tagID2, res.TagIDs)
	}

	// Overwrite: a single different tag.
	if err := h.CaseUpdateTagIDs(ctx, customerID, caseID, []uuid.UUID{tagID3}); err != nil {
		t.Fatalf("CaseUpdateTagIDs() (overwrite) error = %v", err)
	}
	res, err = h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() (after overwrite) error = %v", err)
	}
	if len(res.TagIDs) != 1 || res.TagIDs[0] != tagID3 {
		t.Errorf("expected [%s], got: %v", tagID3, res.TagIDs)
	}

	// Clear: back to empty.
	if err := h.CaseUpdateTagIDs(ctx, customerID, caseID, []uuid.UUID{}); err != nil {
		t.Fatalf("CaseUpdateTagIDs() (clear) error = %v", err)
	}
	res, err = h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() (after clear) error = %v", err)
	}
	if len(res.TagIDs) != 0 {
		t.Errorf("expected empty TagIDs after clear, got: %v", res.TagIDs)
	}
}

// Test_CaseUpdateTagIDs_CustomerScoped_WrongCustomerNoOp verifies the
// customer_id-scoped UPDATE predicate (mirroring
// caseUpdateContactIDExec's tenant-isolation shape): calling
// CaseUpdateTagIDs with the wrong customerID must not modify the row.
func Test_CaseUpdateTagIDs_CustomerScoped_WrongCustomerNoOp(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	victimCustomerID := uuid.FromStringOrNil("f1b2c3d4-5091-5091-5091-000000000001")
	attackerCustomerID := uuid.FromStringOrNil("f1b2c3d4-5091-5091-5091-000000000002")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5091-5091-5091-000000000003")
	tagID := uuid.FromStringOrNil("f1b2c3d4-5091-5091-5091-000000000004")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	c := &kase.Case{
		ID:            caseID,
		CustomerID:    victimCustomerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110011"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	// Wrong customerID: the UPDATE's WHERE customer_id=? must match
	// zero rows -- no error (Exec succeeds with 0 rows affected), but
	// no mutation either.
	if err := h.CaseUpdateTagIDs(ctx, attackerCustomerID, caseID, []uuid.UUID{tagID}); err != nil {
		t.Fatalf("CaseUpdateTagIDs() (wrong customer) error = %v", err)
	}

	res, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if len(res.TagIDs) != 0 {
		t.Errorf("BUG: cross-tenant CaseUpdateTagIDs call modified the victim's case: %v", res.TagIDs)
	}
}

// Test_CaseInsert_DuplicateOpenPeer_ReturnsConflict verifies the
// uq_case_open_peer race-prevention invariant: a second INSERT for the
// same (customer_id, peer_type, peer_target, reference_type) while the
// first is still 'open' must fail with a detectable conflict error, not
// silently succeed or panic.
func Test_CaseInsert_DuplicateOpenPeer_ReturnsConflict(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5002-5002-5002-000000000001")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	c1 := &kase.Case{
		ID:            uuid.FromStringOrNil("f1b2c3d4-5002-5002-5002-000000000002"),
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110002"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, c1); err != nil {
		t.Fatalf("CaseInsert(c1) error = %v", err)
	}

	c2 := &kase.Case{
		ID:            uuid.FromStringOrNil("f1b2c3d4-5002-5002-5002-000000000003"),
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110002"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	err := h.CaseInsert(ctx, c2)
	if err == nil {
		t.Fatalf("expected a conflict error for duplicate open (customer, peer, reference_type), got nil")
	}
	if err != ErrDuplicate {
		t.Errorf("expected ErrDuplicate, got: %v", err)
	}
}

// Test_CaseGetOpenByPeer verifies the locked-select-for-update helper
// (Task 3.2) used by the get-or-create step 1 lookup. Uses sqlmock (not
// the real SQLite in-memory DB) because SQLite does not support the
// "FOR UPDATE" locking clause -- same convention as
// bin-billing-manager's accountAdjust*WithLedger tests.
func Test_CaseGetOpenByPeer(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: db, cache: mockCache, forUpdateSuffix: "FOR UPDATE"}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5003-5003-5003-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5003-5003-5003-000000000002")
	openedAt := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	rowColumns := []string{
		"id", "customer_id", "peer", "local", "reference_type",
		"name", "detail",
		"contact_id", "owner_type", "owner_id",
		"status", "opened_at", "closed_at", "closed_reason", "closed_by_type", "closed_by_id",
		"previous_case_id", "tag_ids", "tm_create", "tm_update",
	}

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	mock.ExpectQuery("SELECT .* FROM contact_cases WHERE .* FOR UPDATE").
		WithArgs(customerID.Bytes(), string(commonaddress.TypeTel), "+155****0003", "call", string(kase.StatusOpen)).
		WillReturnRows(sqlmock.NewRows(rowColumns).AddRow(
			caseID.Bytes(), customerID.Bytes(), `{"type":"tel","target":"+155****0003"}`, `{}`, "call",
			"", nil,
			nil, nil, nil,
			string(kase.StatusOpen), openedAt, nil, nil, nil, nil,
			nil, nil, openedAt, openedAt,
		))

	res, err := h.CaseGetOpenByPeer(ctx, tx, customerID, commonaddress.TypeTel, "+155****0003", "call")
	if err != nil {
		t.Fatalf("CaseGetOpenByPeer() error = %v", err)
	}
	if res == nil || res.ID != caseID {
		t.Errorf("expected to find the open case %s, got: %v", caseID, res)
	}

	// Non-matching peer -> not found (nil, no error)
	mock.ExpectQuery("SELECT .* FROM contact_cases WHERE .* FOR UPDATE").
		WithArgs(customerID.Bytes(), string(commonaddress.TypeTel), "+19999999999", "call", string(kase.StatusOpen)).
		WillReturnRows(sqlmock.NewRows(rowColumns))

	notFound, err := h.CaseGetOpenByPeer(ctx, tx, customerID, commonaddress.TypeTel, "+19999999999", "call")
	if err != nil {
		t.Fatalf("CaseGetOpenByPeer() (not found) error = %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil for non-matching peer, got: %v", notFound)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Test_CaseUpdateStatusClosed verifies the optimistic WHERE status='open'
// guard (design §5.1): closing an already-closed Case is a no-op (idempotent
// double-close), not an error.
func Test_CaseUpdateStatusClosed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5004-5004-5004-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5004-5004-5004-000000000002")
	agentID := uuid.FromStringOrNil("f1b2c3d4-5004-5004-5004-000000000003")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))
	closedAt := timePtr(time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC))

	c := &kase.Case{
		ID:            caseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110004"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	ok, err := h.CaseUpdateStatusClosed(ctx, customerID, caseID, kase.ClosedReasonAgentClosed, kase.ClosedByTypeAgent, &agentID, closedAt)
	if err != nil {
		t.Fatalf("CaseUpdateStatusClosed() error = %v", err)
	}
	if !ok {
		t.Errorf("expected first close to succeed (rows affected), got false")
	}

	res, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if res.Status != kase.StatusClosed {
		t.Errorf("expected status closed, got: %s", res.Status)
	}
	if res.ClosedReason != kase.ClosedReasonAgentClosed {
		t.Errorf("expected closed_reason agent_closed, got: %s", res.ClosedReason)
	}

	// Double-close: WHERE status='open' guard means the second call
	// affects 0 rows -- idempotent no-op, not an error.
	ok2, err := h.CaseUpdateStatusClosed(ctx, customerID, caseID, kase.ClosedReasonTimeout, kase.ClosedByTypeSystem, nil, closedAt)
	if err != nil {
		t.Fatalf("CaseUpdateStatusClosed() second call error = %v", err)
	}
	if ok2 {
		t.Errorf("expected second (double) close to affect 0 rows, got true")
	}

	// Verify the second call's fields did NOT overwrite the first (still agent_closed).
	res2, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if res2.ClosedReason != kase.ClosedReasonAgentClosed {
		t.Errorf("expected closed_reason to remain agent_closed after no-op double-close, got: %s", res2.ClosedReason)
	}
}

// Test_CaseUpdateTMUpdate verifies the tm_update bump helper used at the
// end of the get-or-create transaction (design §4 step 4).
func Test_CaseUpdateTMUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5005-5005-5005-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5005-5005-5005-000000000002")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))
	newUpdateTime := timePtr(time.Date(2026, 6, 28, 13, 0, 0, 0, time.UTC))

	c := &kase.Case{
		ID:            caseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110005"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	if err := h.CaseUpdateTMUpdate(ctx, caseID, newUpdateTime); err != nil {
		t.Fatalf("CaseUpdateTMUpdate() error = %v", err)
	}

	res, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if res.TMUpdate == nil || !res.TMUpdate.Equal(*newUpdateTime) {
		t.Errorf("expected tm_update: %v, got: %v", newUpdateTime, res.TMUpdate)
	}
}

// Test_CaseUpdateContactID verifies the contact_id denormalized-cache
// update helper (design §3.4).
func Test_CaseUpdateContactID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5006-5006-5006-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5006-5006-5006-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-5006-5006-5006-000000000003")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	c := &kase.Case{
		ID:            caseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110006"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	if err := h.CaseUpdateContactID(ctx, customerID, caseID, contactID); err != nil {
		t.Fatalf("CaseUpdateContactID() error = %v", err)
	}

	res, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if res.ContactID == nil || *res.ContactID != contactID {
		t.Errorf("expected contact_id: %v, got: %v", contactID, res.ContactID)
	}
}

// Test_CaseUpdateOwner verifies the Case.Owner write helper (square-talk
// Cases menu design §3.2): asserts the persisted owner_type column
// equals commonidentity.OwnerTypeAgent, not merely that owner_id
// changed.
func Test_CaseUpdateOwner(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5020-5020-5020-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5020-5020-5020-000000000002")
	ownerID := uuid.FromStringOrNil("f1b2c3d4-5020-5020-5020-000000000003")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	c := &kase.Case{
		ID:            caseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0020"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	if err := h.CaseUpdateOwner(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, ownerID); err != nil {
		t.Fatalf("CaseUpdateOwner() error = %v", err)
	}

	res, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if res.OwnerType != commonidentity.OwnerTypeAgent {
		t.Errorf("expected owner_type: %v, got: %v", commonidentity.OwnerTypeAgent, res.OwnerType)
	}
	if res.OwnerID != ownerID {
		t.Errorf("expected owner_id: %v, got: %v", ownerID, res.OwnerID)
	}
}

// Test_CaseUpdateOwner_CrossTenant verifies that a cross-tenant
// customer_id mismatch affects 0 rows (defense-in-depth): the case's
// owner fields are left untouched.
func Test_CaseUpdateOwner_CrossTenant(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	victimCustomerID := uuid.FromStringOrNil("f1b2c3d4-5021-5021-5021-000000000001")
	attackerCustomerID := uuid.FromStringOrNil("f1b2c3d4-5021-5021-5021-000000000002")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5021-5021-5021-000000000003")
	ownerID := uuid.FromStringOrNil("f1b2c3d4-5021-5021-5021-000000000004")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	c := &kase.Case{
		ID:            caseID,
		CustomerID:    victimCustomerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0021"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	if err := h.CaseUpdateOwner(ctx, attackerCustomerID, caseID, commonidentity.OwnerTypeAgent, ownerID); err != nil {
		t.Fatalf("CaseUpdateOwner() error = %v", err)
	}

	res, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if res.OwnerID == ownerID {
		t.Errorf("expected owner_id unchanged (cross-tenant no-op), got: %v", res.OwnerID)
	}
}

// Test_CaseClearContactID verifies the contact_id revert-to-NULL
// helper (design VOIP-1253).
func Test_CaseClearContactID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5012-5012-5012-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5012-5012-5012-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-5012-5012-5012-000000000003")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	c := &kase.Case{
		ID:            caseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110008"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	if err := h.CaseUpdateContactID(ctx, customerID, caseID, contactID); err != nil {
		t.Fatalf("CaseUpdateContactID() error = %v", err)
	}

	if err := h.CaseClearContactID(ctx, customerID, caseID); err != nil {
		t.Fatalf("CaseClearContactID() error = %v", err)
	}

	res, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if res.ContactID != nil {
		t.Errorf("expected nil contact_id after clear, got: %v", *res.ContactID)
	}
}

// Test_CaseClearContactID_CrossTenant verifies that clearing with a
// non-matching customer_id is a scoped no-op (defense-in-depth): the
// case's contact_id is left untouched.
func Test_CaseClearContactID_CrossTenant(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	victimCustomerID := uuid.FromStringOrNil("f1b2c3d4-5013-5013-5013-000000000001")
	attackerCustomerID := uuid.FromStringOrNil("f1b2c3d4-5013-5013-5013-000000000002")
	caseID := uuid.FromStringOrNil("f1b2c3d4-5013-5013-5013-000000000003")
	contactID := uuid.FromStringOrNil("f1b2c3d4-5013-5013-5013-000000000004")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	c := &kase.Case{
		ID:            caseID,
		CustomerID:    victimCustomerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110009"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}
	if err := h.CaseUpdateContactID(ctx, victimCustomerID, caseID, contactID); err != nil {
		t.Fatalf("CaseUpdateContactID() error = %v", err)
	}

	// The exec-based Update primitive is a scoped no-op (0 rows
	// affected) for a non-matching customer_id, not an ErrNotFound --
	// it returns nil, mirroring caseUpdateContactIDExec's own
	// behavior. The upstream caller (casehandler.UpdateContact) is
	// responsible for the ownership check via verifyCaseOwnership.
	if err := h.CaseClearContactID(ctx, attackerCustomerID, caseID); err != nil {
		t.Fatalf("CaseClearContactID() error = %v", err)
	}

	res, err := h.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if res.ContactID == nil || *res.ContactID != contactID {
		t.Errorf("BUG: cross-tenant CaseClearContactID call cleared the victim's contact_id: %v", res.ContactID)
	}
}

// Test_CaseListUnresolved verifies the idx_case_unresolved-backed list
// (design §6): Cases with contact_id IS NULL, scoped to customer.
func Test_CaseListUnresolved(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5007-5007-5007-000000000001")
	unresolvedCaseID := uuid.FromStringOrNil("f1b2c3d4-5007-5007-5007-000000000002")
	resolvedCaseID := uuid.FromStringOrNil("f1b2c3d4-5007-5007-5007-000000000003")
	contactID := uuid.FromStringOrNil("f1b2c3d4-5007-5007-5007-000000000004")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	unresolved := &kase.Case{
		ID:            unresolvedCaseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110007"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	resolved := &kase.Case{
		ID:            resolvedCaseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110008"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		ContactID:     &contactID,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, unresolved); err != nil {
		t.Fatalf("CaseInsert(unresolved) error = %v", err)
	}
	if err := h.CaseInsert(ctx, resolved); err != nil {
		t.Fatalf("CaseInsert(resolved) error = %v", err)
	}

	res, err := h.CaseListUnresolved(ctx, customerID)
	if err != nil {
		t.Fatalf("CaseListUnresolved() error = %v", err)
	}

	foundUnresolved := false
	foundResolved := false
	for _, item := range res {
		if item.ID == unresolvedCaseID {
			foundUnresolved = true
		}
		if item.ID == resolvedCaseID {
			foundResolved = true
		}
	}
	if !foundUnresolved {
		t.Errorf("expected unresolved case to appear in CaseListUnresolved()")
	}
	if foundResolved {
		t.Errorf("expected resolved case (contact_id set) to NOT appear in CaseListUnresolved()")
	}
}

// Test_CaseListByOwner verifies the idx_case_owner-backed "my cases" list
// (design §7).
func Test_CaseListByOwner(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5010-5010-5010-000000000001")
	ownedCaseID := uuid.FromStringOrNil("f1b2c3d4-5010-5010-5010-000000000002")
	unownedCaseID := uuid.FromStringOrNil("f1b2c3d4-5010-5010-5010-000000000003")
	agentID := uuid.FromStringOrNil("f1b2c3d4-5010-5010-5010-000000000004")
	openedAt := timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC))

	owned := &kase.Case{
		ID:            ownedCaseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110009"},
		ReferenceType: "call",
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   agentID,
		},
		Status:   kase.StatusOpen,
		OpenedAt: openedAt,
		TMCreate: openedAt,
		TMUpdate: openedAt,
	}
	unowned := &kase.Case{
		ID:            unownedCaseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110010"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      openedAt,
		TMCreate:      openedAt,
		TMUpdate:      openedAt,
	}
	if err := h.CaseInsert(ctx, owned); err != nil {
		t.Fatalf("CaseInsert(owned) error = %v", err)
	}
	if err := h.CaseInsert(ctx, unowned); err != nil {
		t.Fatalf("CaseInsert(unowned) error = %v", err)
	}

	res, err := h.CaseListByOwner(ctx, customerID, commonidentity.OwnerTypeAgent, agentID)
	if err != nil {
		t.Fatalf("CaseListByOwner() error = %v", err)
	}

	foundOwned := false
	foundUnowned := false
	for _, item := range res {
		if item.ID == ownedCaseID {
			foundOwned = true
		}
		if item.ID == unownedCaseID {
			foundUnowned = true
		}
	}
	if !foundOwned {
		t.Errorf("expected owned case to appear in CaseListByOwner()")
	}
	if foundUnowned {
		t.Errorf("expected unowned case to NOT appear in CaseListByOwner()")
	}
}

// Test_CaseGetLastClosedByPeer verifies the previous_case_id chaining
// lookup (design §4, fresh-insert path): finds the most recently closed
// Case for a given peer, or nil if none exists.
func Test_CaseGetLastClosedByPeer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-5011-5011-5011-000000000001")
	olderClosedID := uuid.FromStringOrNil("f1b2c3d4-5011-5011-5011-000000000002")
	newerClosedID := uuid.FromStringOrNil("f1b2c3d4-5011-5011-5011-000000000003")

	older := &kase.Case{
		ID:            olderClosedID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110011"},
		ReferenceType: "call",
		Status:        kase.StatusClosed,
		OpenedAt:      timePtr(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)),
		ClosedAt:      timePtr(time.Date(2026, 6, 27, 11, 0, 0, 0, time.UTC)),
		ClosedReason:  kase.ClosedReasonAgentClosed,
		TMCreate:      timePtr(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)),
		TMUpdate:      timePtr(time.Date(2026, 6, 27, 11, 0, 0, 0, time.UTC)),
	}
	newer := &kase.Case{
		ID:            newerClosedID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110011"},
		ReferenceType: "call",
		Status:        kase.StatusClosed,
		OpenedAt:      timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)),
		ClosedAt:      timePtr(time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)),
		ClosedReason:  kase.ClosedReasonAgentClosed,
		TMCreate:      timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)),
		TMUpdate:      timePtr(time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)),
	}
	if err := h.CaseInsert(ctx, older); err != nil {
		t.Fatalf("CaseInsert(older) error = %v", err)
	}
	if err := h.CaseInsert(ctx, newer); err != nil {
		t.Fatalf("CaseInsert(newer) error = %v", err)
	}

	res, err := h.CaseGetLastClosedByPeer(ctx, customerID, commonaddress.TypeTel, "+15551110011", "call")
	if err != nil {
		t.Fatalf("CaseGetLastClosedByPeer() error = %v", err)
	}
	if res == nil || res.ID != newerClosedID {
		t.Errorf("expected the more recently closed case %s, got: %v", newerClosedID, res)
	}

	// No closed case for a peer that never had one -> nil, no error.
	none, err := h.CaseGetLastClosedByPeer(ctx, customerID, commonaddress.TypeTel, "+19999999999", "call")
	if err != nil {
		t.Fatalf("CaseGetLastClosedByPeer() (none) error = %v", err)
	}
	if none != nil {
		t.Errorf("expected nil for a peer with no closed case, got: %v", none)
	}
}
