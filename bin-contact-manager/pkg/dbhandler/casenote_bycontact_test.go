package dbhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_CaseNoteListByContact covers the VOIP-1673 A1 aggregation query:
// notes across all cases of one contact, newest first, tenant scoped,
// soft-deleted notes excluded, paginated by a tm_create cursor.
func Test_CaseNoteListByContact(t *testing.T) {
	resetTestDB(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-000000000001")
	otherCustomerID := uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-000000000003")
	otherContactID := uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-000000000004")
	caseA := uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-0000000000a1")
	caseB := uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-0000000000a2")
	caseC := uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-0000000000a3")
	caseOtherContact := uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-0000000000a4")
	caseOtherCustomer := uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-0000000000a5")
	opened := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	insertCase := func(id, cust uuid.UUID, target string, contact uuid.UUID) {
		t.Helper()
		c := &kase.Case{
			ID: id, CustomerID: cust,
			Peer:          commonaddress.Address{Type: commonaddress.TypeTel, Target: target},
			ReferenceType: "call", Status: kase.StatusOpen,
			OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
		}
		if err := h.CaseInsert(ctx, c); err != nil {
			t.Fatalf("CaseInsert() error = %v", err)
		}
		if err := h.CaseUpdateContactID(ctx, cust, id, contact); err != nil {
			t.Fatalf("CaseUpdateContactID() error = %v", err)
		}
	}
	insertCase(caseA, customerID, "+155****7301", contactID)
	insertCase(caseB, customerID, "+155****7302", contactID)
	insertCase(caseC, customerID, "+155****7303", contactID)
	insertCase(caseOtherContact, customerID, "+155****7304", otherContactID)
	// a row of another customer that carries the same contact_id value
	insertCase(caseOtherCustomer, otherCustomerID, "+155****7305", contactID)

	insertNote := func(idSuffix string, cust, caseID uuid.UUID, minute int) uuid.UUID {
		t.Helper()
		id := uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-0000000001" + idSuffix)
		ts := time.Date(2026, 6, 2, 10, minute, 0, 0, time.UTC)
		if err := h.CaseNoteCreate(ctx, &casenote.CaseNote{
			ID: id, CustomerID: cust, CaseID: caseID,
			AuthorType: casenote.AuthorTypeSystem, Text: "n" + idSuffix, TMCreate: &ts,
		}); err != nil {
			t.Fatalf("CaseNoteCreate() error = %v", err)
		}
		return id
	}
	// interleaved over three cases: minute order 1..6
	n1 := insertNote("01", customerID, caseA, 1)
	n2 := insertNote("02", customerID, caseB, 2)
	n3 := insertNote("03", customerID, caseA, 3)
	n4 := insertNote("04", customerID, caseC, 4)
	n5 := insertNote("05", customerID, caseB, 5)
	nDeleted := insertNote("06", customerID, caseC, 6)
	insertNote("07", customerID, caseOtherContact, 7)
	insertNote("08", otherCustomerID, caseOtherCustomer, 8)
	// a note of this customer attached to a case of the other customer (case side tenant check)
	insertNote("09", customerID, caseOtherCustomer, 9)

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)))
	if err := h.CaseNoteDelete(ctx, customerID, caseC, nDeleted); err != nil {
		t.Fatalf("CaseNoteDelete() error = %v", err)
	}

	// (1)(3) newest first across cases, deleted note and other contact/customer rows excluded
	res, err := h.CaseNoteListByContact(ctx, customerID, contactID, 100, "")
	if err != nil {
		t.Fatalf("CaseNoteListByContact() error = %v", err)
	}
	want := []uuid.UUID{n5, n4, n3, n2, n1}
	if len(res) != len(want) {
		t.Fatalf("expected %d notes, got %d", len(want), len(res))
	}
	for i := range want {
		if res[i].ID != want[i] {
			t.Errorf("index %d: expected %s, got %s", i, want[i], res[i].ID)
		}
		if res[i].CaseID == uuid.Nil {
			t.Errorf("index %d: case_id must be populated", i)
		}
	}

	// (1) page boundaries: size 2 pages with a tm_create cursor, no gap and no duplicate
	var got []uuid.UUID
	token := ""
	for page := 0; page < 10; page++ {
		items, errPage := h.CaseNoteListByContact(ctx, customerID, contactID, 2, token)
		if errPage != nil {
			t.Fatalf("page %d error = %v", page, errPage)
		}
		if len(items) == 0 {
			break
		}
		for _, it := range items {
			got = append(got, it.ID)
		}
		token = items[len(items)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
	}
	if len(got) != len(want) {
		t.Fatalf("paged result count: expected %d, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("paged index %d: expected %s, got %s", i, want[i], got[i])
		}
	}

	// (2) the other customer never sees this customer's rows, even with this customer's contact id
	cross, err := h.CaseNoteListByContact(ctx, otherCustomerID, contactID, 100, "")
	if err != nil {
		t.Fatalf("cross-customer error = %v", err)
	}
	if len(cross) != 1 || cross[0].CustomerID != otherCustomerID {
		t.Errorf("cross-customer list must only contain its own row, got %d rows", len(cross))
	}

	// (3) a contact without cases returns an empty result
	empty, err := h.CaseNoteListByContact(ctx, customerID, uuid.FromStringOrNil("f1b2c3d4-7301-7301-7301-0000000000ff"), 100, "")
	if err != nil {
		t.Fatalf("empty error = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected no notes, got %d", len(empty))
	}

	// (4) re-attributing a case moves its notes to the current contact
	if err := h.CaseUpdateContactID(ctx, customerID, caseB, otherContactID); err != nil {
		t.Fatalf("CaseUpdateContactID(reattribute) error = %v", err)
	}
	after, err := h.CaseNoteListByContact(ctx, customerID, contactID, 100, "")
	if err != nil {
		t.Fatalf("after reattribute error = %v", err)
	}
	if len(after) != 3 || after[0].ID != n4 || after[1].ID != n3 || after[2].ID != n1 {
		t.Errorf("expected notes (n4, n3, n1) after reattribution, got %d rows", len(after))
	}

	// invalid page token
	if _, err := h.CaseNoteListByContact(ctx, customerID, contactID, 10, "not-a-time"); err == nil {
		t.Errorf("expected an error for an invalid page token")
	}
}
