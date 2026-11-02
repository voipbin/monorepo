package casehandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func newCaseNoteByContactHandler(t *testing.T) (*caseHandler, dbhandler.DBHandler, *cachehandler.MockCacheHandler) {
	t.Helper()
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	mockCache.EXPECT().ContactGet(gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound).AnyTimes()
	mockCache.EXPECT().ContactSet(gomock.Any(), gomock.Any()).AnyTimes()
	mockCache.EXPECT().ContactDelete(gomock.Any(), gomock.Any()).AnyTimes()
	return &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}, db, mockCache
}

// Test_CaseNoteListByContact verifies the VOIP-1673 A1 handler: ownership
// check (404 for another customer's or a deleted contact, no existence
// leak), cross-case newest-first listing with a page token, empty result.
func Test_CaseNoteListByContact(t *testing.T) {
	h, db, _ := newCaseNoteByContactHandler(t)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7401-7401-7401-000000000001")
	otherCustomerID := uuid.FromStringOrNil("f1b2c3d4-7401-7401-7401-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-7401-7401-7401-000000000003")
	emptyContactID := uuid.FromStringOrNil("f1b2c3d4-7401-7401-7401-000000000004")
	foreignContactID := uuid.FromStringOrNil("f1b2c3d4-7401-7401-7401-000000000005")
	deletedContactID := uuid.FromStringOrNil("f1b2c3d4-7401-7401-7401-000000000006")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	for _, c := range []struct {
		id, cust uuid.UUID
	}{{contactID, customerID}, {emptyContactID, customerID}, {foreignContactID, otherCustomerID}, {deletedContactID, customerID}} {
		ct := &contact.Contact{Identity: commonidentity.Identity{ID: c.id, CustomerID: c.cust}}
		if err := db.ContactCreate(ctx, ct); err != nil {
			t.Fatalf("ContactCreate() error = %v", err)
		}
	}
	if err := db.ContactDelete(ctx, deletedContactID); err != nil {
		t.Fatalf("ContactDelete() error = %v", err)
	}

	caseA := uuid.FromStringOrNil("f1b2c3d4-7401-7401-7401-0000000000a1")
	caseB := uuid.FromStringOrNil("f1b2c3d4-7401-7401-7401-0000000000a2")
	for i, id := range []uuid.UUID{caseA, caseB} {
		c := &kase.Case{
			ID: id, CustomerID: customerID,
			Peer:          commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****740" + string(rune('1'+i))},
			ReferenceType: "call", Status: kase.StatusOpen, OpenedAt: &now, TMCreate: &now, TMUpdate: &now,
		}
		if err := db.CaseInsert(ctx, c); err != nil {
			t.Fatalf("CaseInsert() error = %v", err)
		}
		if err := db.CaseUpdateContactID(ctx, customerID, id, contactID); err != nil {
			t.Fatalf("CaseUpdateContactID() error = %v", err)
		}
	}
	var ids []uuid.UUID
	for i := 0; i < 5; i++ {
		id := uuid.FromStringOrNil("f1b2c3d4-7401-7401-7401-00000000010" + string(rune('0'+i)))
		ts := time.Date(2026, 6, 28, 13, i, 0, 0, time.UTC)
		cid := caseA
		if i%2 == 1 {
			cid = caseB
		}
		if err := db.CaseNoteCreate(ctx, &casenote.CaseNote{ID: id, CustomerID: customerID, CaseID: cid, AuthorType: casenote.AuthorTypeSystem, Text: "t", TMCreate: &ts}); err != nil {
			t.Fatalf("CaseNoteCreate() error = %v", err)
		}
		ids = append([]uuid.UUID{id}, ids...) // newest first
	}

	// page 1 (size 2): newest two across both cases, token present
	items, token, err := h.CaseNoteListByContact(ctx, customerID, contactID, 2, "")
	if err != nil {
		t.Fatalf("page 1 error = %v", err)
	}
	if len(items) != 2 || items[0].ID != ids[0] || items[1].ID != ids[1] {
		t.Fatalf("page 1 mismatch: %v", items)
	}
	if token == "" {
		t.Fatalf("expected a next page token")
	}
	// page 2 and 3 using the token
	var got []uuid.UUID
	for _, it := range items {
		got = append(got, it.ID)
	}
	for token != "" {
		var page []*casenote.CaseNote
		page, token, err = h.CaseNoteListByContact(ctx, customerID, contactID, 2, token)
		if err != nil {
			t.Fatalf("next page error = %v", err)
		}
		for _, it := range page {
			got = append(got, it.ID)
		}
	}
	if len(got) != len(ids) {
		t.Fatalf("expected %d notes over all pages, got %d", len(ids), len(got))
	}
	for i := range ids {
		if got[i] != ids[i] {
			t.Errorf("index %d: expected %s, got %s", i, ids[i], got[i])
		}
	}

	// size 0 falls back to the default size
	def, _, err := h.CaseNoteListByContact(ctx, customerID, contactID, 0, "")
	if err != nil || len(def) != len(ids) {
		t.Errorf("default size: err=%v len=%d", err, len(def))
	}

	// a contact with no cases returns an empty list and no token
	empty, tok, err := h.CaseNoteListByContact(ctx, customerID, emptyContactID, 10, "")
	if err != nil || len(empty) != 0 || tok != "" {
		t.Errorf("empty contact: err=%v len=%d token=%q", err, len(empty), tok)
	}

	// another customer's contact, a deleted contact and an unknown id are all 404
	for name, id := range map[string]uuid.UUID{
		"foreign": foreignContactID,
		"deleted": deletedContactID,
		"unknown": uuid.FromStringOrNil("f1b2c3d4-7401-7401-7401-0000000000ff"),
	} {
		_, _, err := h.CaseNoteListByContact(ctx, customerID, id, 10, "")
		var ve *cerrors.VoipbinError
		if !stderrors.As(err, &ve) || ve.Status != cerrors.StatusNotFound {
			t.Errorf("%s: expected a NotFound error, got %v", name, err)
		}
	}
}
