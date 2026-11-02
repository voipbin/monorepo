package dbhandler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/contactstar"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

const starTokenLayout = "2006-01-02T15:04:05.000000Z"

var (
	starCustomerID      = uuid.FromStringOrNil("c1a0b100-1673-4b02-0000-000000000001")
	starOtherCustomerID = uuid.FromStringOrNil("c1a0b100-1673-4b02-0000-000000000002")
	starAgentA          = uuid.FromStringOrNil("c1a0b100-1673-4b02-0000-0000000000a1")
	starAgentB          = uuid.FromStringOrNil("c1a0b100-1673-4b02-0000-0000000000a2")
)

// starContactID returns a deterministic contact id for index n.
func starContactID(n int) uuid.UUID {
	return uuid.FromStringOrNil(fmt.Sprintf("c1a0b100-1673-4b02-0001-%012d", n))
}

func starRowID(n int) uuid.UUID {
	return uuid.FromStringOrNil(fmt.Sprintf("c1a0b100-1673-4b02-0002-%012d", n))
}

// starInsertContact inserts a bare contact row. minute orders contacts by tm_create;
// deleted marks it soft-deleted.
func starInsertContact(t *testing.T, id, customerID uuid.UUID, minute int, deleted bool) {
	t.Helper()

	ts := time.Date(2026, 6, 1, 10, minute, 0, 0, time.UTC)
	var tmDelete any
	if deleted {
		tmDelete = ts.Add(time.Hour)
	}
	if _, err := dbTest.Exec(
		`INSERT INTO contact_contacts (id, customer_id, first_name, last_name, display_name, company, job_title, source, external_id, notes, tm_create, tm_delete)
		VALUES (?, ?, 'f', 'l', 'd', 'c', 'j', 'manual', '', '', ?, ?)`,
		id.Bytes(), customerID.Bytes(), ts, tmDelete,
	); err != nil {
		t.Fatalf("could not insert the contact. err: %v", err)
	}
}

// starCreate stars contactID for the agent at the given minute and returns the stored row.
func starCreate(t *testing.T, h *handler, mockUtil *utilhandler.MockUtilHandler, n int, customerID, agentID, contactID uuid.UUID, minute int) *contactstar.ContactStar {
	t.Helper()

	ts := time.Date(2026, 6, 2, 10, minute, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&ts)

	res, err := h.ContactStarCreate(context.Background(), &contactstar.ContactStar{
		Identity:  commonidentity.Identity{ID: starRowID(n), CustomerID: customerID},
		AgentID:   agentID,
		ContactID: contactID,
	})
	if err != nil {
		t.Fatalf("ContactStarCreate() error = %v", err)
	}

	return res
}

func starNewHandler(t *testing.T) (*handler, *utilhandler.MockUtilHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	return &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}, mockUtil
}

func starCount(t *testing.T) int {
	t.Helper()

	var n int
	if err := dbTest.QueryRow(`SELECT COUNT(*) FROM contact_stars`).Scan(&n); err != nil {
		t.Fatalf("could not count the stars. err: %v", err)
	}

	return n
}

func Test_ContactStarCreate_Idempotent(t *testing.T) {
	resetTestDB(t)
	h, mockUtil := starNewHandler(t)

	first := starCreate(t, h, mockUtil, 1, starCustomerID, starAgentA, starContactID(1), 1)
	if first.ID != starRowID(1) || first.TMCreate == nil {
		t.Fatalf("Wrong match. got: %+v", first)
	}

	// the second call returns the existing row: same id, same tm_create, no extra row
	later := time.Date(2026, 6, 2, 11, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&later)
	second, err := h.ContactStarCreate(context.Background(), &contactstar.ContactStar{
		Identity:  commonidentity.Identity{ID: starRowID(2), CustomerID: starCustomerID},
		AgentID:   starAgentA,
		ContactID: starContactID(1),
	})
	if err != nil {
		t.Fatalf("ContactStarCreate() second error = %v", err)
	}
	if second.ID != first.ID || !second.TMCreate.Equal(*first.TMCreate) {
		t.Errorf("Wrong match. expect the first row, got: %+v", second)
	}
	if n := starCount(t); n != 1 {
		t.Errorf("Wrong match. expect: 1 row, got: %d", n)
	}

	// another agent starring the same contact is a separate row
	other := starCreate(t, h, mockUtil, 3, starCustomerID, starAgentB, starContactID(1), 5)
	if other.ID == first.ID || starCount(t) != 2 {
		t.Errorf("Wrong match. expect an independent row for agent B, got: %+v", other)
	}
}

func Test_ContactStarCreate_InsertErrorWithoutRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock. err: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := &handler{utilHandler: mockUtil, db: db}

	now := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now)

	// the insert fails for a reason other than a duplicate, and the re-read finds no row:
	// the original error must be returned, not swallowed.
	boom := errors.New("boom")
	mock.ExpectExec("INSERT INTO contact_stars").WillReturnError(boom)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err = h.ContactStarCreate(context.Background(), &contactstar.ContactStar{
		Identity:  commonidentity.Identity{ID: starRowID(1), CustomerID: starCustomerID},
		AgentID:   starAgentA,
		ContactID: starContactID(1),
	})
	if !errors.Is(err, boom) {
		t.Errorf("Wrong match. expect the insert error, got: %v", err)
	}
}

func Test_ContactStarDelete(t *testing.T) {
	resetTestDB(t)
	h, mockUtil := starNewHandler(t)
	ctx := context.Background()

	starCreate(t, h, mockUtil, 1, starCustomerID, starAgentA, starContactID(1), 1)
	starCreate(t, h, mockUtil, 2, starCustomerID, starAgentB, starContactID(1), 2)
	starCreate(t, h, mockUtil, 3, starOtherCustomerID, starAgentA, starContactID(3), 3)
	// the same agent also stars another contact, which must survive the unstar of contact 1
	starCreate(t, h, mockUtil, 4, starCustomerID, starAgentA, starContactID(2), 4)

	// another customer's call does not remove the star of this customer
	if err := h.ContactStarDelete(ctx, starOtherCustomerID, starAgentB, starContactID(1)); err != nil {
		t.Fatalf("ContactStarDelete() error = %v", err)
	}
	if n := starCount(t); n != 4 {
		t.Fatalf("Wrong match. expect: 4 rows untouched, got: %d", n)
	}

	// removes only the (customer, agent, contact) row, and repeating it is not an error
	for i := 0; i < 2; i++ {
		if err := h.ContactStarDelete(ctx, starCustomerID, starAgentA, starContactID(1)); err != nil {
			t.Fatalf("ContactStarDelete() call %d error = %v", i, err)
		}
	}
	if n := starCount(t); n != 3 {
		t.Errorf("Wrong match. expect: 3 rows, got: %d", n)
	}
}

func Test_ContactListStarred(t *testing.T) {
	resetTestDB(t)
	h, mockUtil := starNewHandler(t)
	ctx := context.Background()

	// contacts 1..5 live; 6 is soft-deleted; 7 belongs to another customer
	for n := 1; n <= 5; n++ {
		starInsertContact(t, starContactID(n), starCustomerID, n, false)
	}
	starInsertContact(t, starContactID(6), starCustomerID, 6, true)
	starInsertContact(t, starContactID(7), starOtherCustomerID, 7, false)
	starInsertContact(t, starContactID(8), starCustomerID, 8, false) // starred only by agent B

	// agent A stars in the REVERSE order of the contacts' tm_create, so a star-time sort differs
	// from the contact tm_create sort the endpoint documents.
	for i, n := range []int{5, 4, 3, 2, 1, 6} {
		starCreate(t, h, mockUtil, 10+i, starCustomerID, starAgentA, starContactID(n), i+1)
	}
	// the same agent id carrying stars of ANOTHER customer must not leak through: a star of the
	// other customer on the other customer's contact (7), and a star of the other customer on a
	// contact of this customer (9, mismatched customer on the join).
	starInsertContact(t, starContactID(9), starCustomerID, 9, false)
	starCreate(t, h, mockUtil, 20, starOtherCustomerID, starAgentA, starContactID(7), 20)
	starCreate(t, h, mockUtil, 22, starOtherCustomerID, starAgentA, starContactID(9), 22)
	starCreate(t, h, mockUtil, 21, starCustomerID, starAgentB, starContactID(8), 21)

	ids := func(res []uuid.UUID) string { return fmt.Sprint(res) }
	collect := func(agentID uuid.UUID, size uint64, token string) ([]uuid.UUID, string) {
		t.Helper()
		items, err := h.ContactListStarred(ctx, starCustomerID, agentID, size, token)
		if err != nil {
			t.Fatalf("ContactListStarred() error = %v", err)
		}
		res := []uuid.UUID{}
		for _, c := range items {
			res = append(res, c.ID)
		}
		next := ""
		if len(items) > 0 {
			next = items[len(items)-1].TMCreate.UTC().Format(starTokenLayout)
		}
		return res, next
	}

	// (2) agent isolation, (3)(4) deleted and other-customer contacts excluded, contact tm_create DESC
	got, _ := collect(starAgentA, 100, "")
	want := []uuid.UUID{starContactID(5), starContactID(4), starContactID(3), starContactID(2), starContactID(1)}
	if ids(got) != ids(want) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", want, got)
	}
	gotB, _ := collect(starAgentB, 100, "")
	if ids(gotB) != ids([]uuid.UUID{starContactID(8)}) {
		t.Errorf("Wrong match. agent B expect only contact 8, got: %v", gotB)
	}
	if gotC, _ := collect(uuid.FromStringOrNil("c1a0b100-1673-4b02-0000-0000000000a3"), 100, ""); len(gotC) != 0 {
		t.Errorf("Wrong match. an agent without stars expects an empty list, got: %v", gotC)
	}

	// (4) page boundary: the token is the last contact's tm_create, the same key as the sort
	var paged []uuid.UUID
	token := ""
	for page := 0; page < 10; page++ {
		items, next := collect(starAgentA, 2, token)
		if len(items) == 0 {
			break
		}
		paged = append(paged, items...)
		token = next
	}
	if ids(paged) != ids(want) {
		t.Errorf("Wrong match. paged result must have no gap and no duplicate.\nexpect: %v\ngot: %v", want, paged)
	}

	// an unparsable token is a distinct error, not a SQL failure
	if _, err := h.ContactListStarred(ctx, starCustomerID, starAgentA, 10, "not-a-time"); !errors.Is(err, ErrInvalidPageToken) {
		t.Errorf("Wrong match. expect: ErrInvalidPageToken, got: %v", err)
	}
}

func Test_ContactStarList(t *testing.T) {
	resetTestDB(t)
	h, mockUtil := starNewHandler(t)
	ctx := context.Background()

	starInsertContact(t, starContactID(1), starCustomerID, 1, false)
	starInsertContact(t, starContactID(2), starCustomerID, 2, false)
	starInsertContact(t, starContactID(3), starCustomerID, 3, true)

	// the same agent id also starred a contact of ANOTHER customer: it must not be listed for this customer
	starInsertContact(t, starContactID(7), starOtherCustomerID, 7, false)
	starCreate(t, h, mockUtil, 30, starOtherCustomerID, starAgentA, starContactID(7), 30)

	// star times are 1, 2, 3 minutes; contact 3 is soft-deleted
	starCreate(t, h, mockUtil, 1, starCustomerID, starAgentA, starContactID(1), 1)
	starCreate(t, h, mockUtil, 2, starCustomerID, starAgentA, starContactID(2), 2)
	starCreate(t, h, mockUtil, 3, starCustomerID, starAgentA, starContactID(3), 3)
	starCreate(t, h, mockUtil, 4, starCustomerID, starAgentB, starContactID(1), 4)

	collect := func(agentID uuid.UUID, size uint64, token string) []*contactstar.ContactStar {
		t.Helper()
		items, err := h.ContactStarList(ctx, starCustomerID, agentID, size, token)
		if err != nil {
			t.Fatalf("ContactStarList() error = %v", err)
		}
		return items
	}

	// newest star first, the soft-deleted contact's star and the other agent's star are excluded
	got := collect(starAgentA, 100, "")
	if len(got) != 2 || got[0].ContactID != starContactID(2) || got[1].ContactID != starContactID(1) {
		t.Fatalf("Wrong match. got: %+v", got)
	}

	// the token key is the STAR row's tm_create (not the contact's)
	page1 := collect(starAgentA, 1, "")
	if len(page1) != 1 || page1[0].ContactID != starContactID(2) {
		t.Fatalf("Wrong match. page1: %+v", page1)
	}
	page2 := collect(starAgentA, 1, page1[0].TMCreate.UTC().Format(starTokenLayout))
	if len(page2) != 1 || page2[0].ContactID != starContactID(1) {
		t.Fatalf("Wrong match. page2: %+v", page2)
	}
	if page3 := collect(starAgentA, 1, page2[0].TMCreate.UTC().Format(starTokenLayout)); len(page3) != 0 {
		t.Errorf("Wrong match. page3 expect empty, got: %+v", page3)
	}

	if _, err := h.ContactStarList(ctx, starCustomerID, starAgentA, 10, "bad"); !errors.Is(err, ErrInvalidPageToken) {
		t.Errorf("Wrong match. expect: ErrInvalidPageToken, got: %v", err)
	}
}

func Test_ContactStarDeleteByCustomerID(t *testing.T) {
	resetTestDB(t)
	h, mockUtil := starNewHandler(t)

	starCreate(t, h, mockUtil, 1, starCustomerID, starAgentA, starContactID(1), 1)
	starCreate(t, h, mockUtil, 2, starCustomerID, starAgentB, starContactID(2), 2)
	starCreate(t, h, mockUtil, 3, starOtherCustomerID, starAgentA, starContactID(3), 3)

	if err := h.ContactStarDeleteByCustomerID(context.Background(), starCustomerID); err != nil {
		t.Fatalf("ContactStarDeleteByCustomerID() error = %v", err)
	}

	if n := starCount(t); n != 1 {
		t.Errorf("Wrong match. expect: only the other customer's row, got: %d rows", n)
	}
}
