package dbhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_CaseNoteCreate_And_CaseNoteListByCase verifies round-trip create
// + list (design §3.5). Active notes (tm_delete IS NULL) are ordered by
// tm_create.
func Test_CaseNoteCreate_And_CaseNoteListByCase(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9501-9501-9501-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9501-9501-9501-000000000002")
	authorID := uuid.FromStringOrNil("f1b2c3d4-9501-9501-9501-000000000003")
	id1 := uuid.FromStringOrNil("f1b2c3d4-9501-9501-9501-000000000004")
	id2 := uuid.FromStringOrNil("f1b2c3d4-9501-9501-9501-000000000005")
	t1 := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)

	n1 := &casenote.CaseNote{
		ID: id1, CustomerID: customerID, CaseID: caseID,
		AuthorType: casenote.AuthorTypeAgent, AuthorID: &authorID,
		Text: "first note", TMCreate: &t1,
	}
	n2 := &casenote.CaseNote{
		ID: id2, CustomerID: customerID, CaseID: caseID,
		AuthorType: casenote.AuthorTypeAgent, AuthorID: &authorID,
		Text: "second note", TMCreate: &t2,
	}
	if err := h.CaseNoteCreate(ctx, n1); err != nil {
		t.Fatalf("CaseNoteCreate(n1) error = %v", err)
	}
	if err := h.CaseNoteCreate(ctx, n2); err != nil {
		t.Fatalf("CaseNoteCreate(n2) error = %v", err)
	}

	res, err := h.CaseNoteListByCase(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("CaseNoteListByCase() error = %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(res))
	}
	if res[0].ID != id1 || res[1].ID != id2 {
		t.Errorf("expected notes ordered by tm_create (id1, id2), got: %s, %s", res[0].ID, res[1].ID)
	}
	if res[0].Text != "first note" || res[1].Text != "second note" {
		t.Errorf("unexpected note text: %+v", res)
	}
}

// Test_CaseNoteDelete_SoftDeletesAndExcludesFromList verifies the
// soft-delete via tm_delete (design §3.5's explicit choice, mirroring
// contact_resolutions' retraction pattern): a deleted note is excluded
// from CaseNoteListByCase.
func Test_CaseNoteDelete_SoftDeletesAndExcludesFromList(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9502-9502-9502-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9502-9502-9502-000000000002")
	id := uuid.FromStringOrNil("f1b2c3d4-9502-9502-9502-000000000003")
	created := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	deleted := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)

	n := &casenote.CaseNote{
		ID: id, CustomerID: customerID, CaseID: caseID,
		AuthorType: casenote.AuthorTypeAgent,
		Text:       "to be deleted", TMCreate: &created,
	}
	if err := h.CaseNoteCreate(ctx, n); err != nil {
		t.Fatalf("CaseNoteCreate() error = %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&deleted)
	if err := h.CaseNoteDelete(ctx, customerID, caseID, id); err != nil {
		t.Fatalf("CaseNoteDelete() error = %v", err)
	}

	res, err := h.CaseNoteListByCase(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("CaseNoteListByCase() error = %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected the deleted note to be excluded, got: %+v", res)
	}
}

// Test_CaseNoteDelete_NotFound verifies deleting a non-existent (or
// already-deleted, or cross-tenant) note returns ErrNotFound.
func Test_CaseNoteDelete_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9503-9503-9503-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9503-9503-9503-000000000002")
	ghostID := uuid.FromStringOrNil("f1b2c3d4-9503-9503-9503-000000000099")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeNow().Return(&now)
	err := h.CaseNoteDelete(ctx, customerID, caseID, ghostID)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
