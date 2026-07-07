package dbhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_CaseTagAssignmentCreate_And_ListByCaseID verifies round-trip
// create + list (design §7 round-22 correction): mirrors
// TagAssignmentCreate/ListByContactID exactly, scoped to case_id instead
// of contact_id.
func Test_CaseTagAssignmentCreate_And_ListByCaseID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	caseID := uuid.FromStringOrNil("f1b2c3d4-9801-9801-9801-000000000001")
	tagID1 := uuid.FromStringOrNil("f1b2c3d4-9801-9801-9801-000000000002")
	tagID2 := uuid.FromStringOrNil("f1b2c3d4-9801-9801-9801-000000000003")

	if err := h.CaseTagAssignmentCreate(ctx, caseID, tagID1); err != nil {
		t.Fatalf("CaseTagAssignmentCreate(tagID1) error = %v", err)
	}
	if err := h.CaseTagAssignmentCreate(ctx, caseID, tagID2); err != nil {
		t.Fatalf("CaseTagAssignmentCreate(tagID2) error = %v", err)
	}

	res, err := h.CaseTagAssignmentListByCaseID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseTagAssignmentListByCaseID() error = %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 tag assignments, got %d", len(res))
	}
	found := map[uuid.UUID]bool{res[0]: true, res[1]: true}
	if !found[tagID1] || !found[tagID2] {
		t.Errorf("expected both tag IDs present, got: %v", res)
	}
}

// Test_CaseTagAssignmentDelete_RemovesOnlyThatAssignment verifies delete
// scoping: removing (caseID, tagID1) leaves (caseID, tagID2) intact.
func Test_CaseTagAssignmentDelete_RemovesOnlyThatAssignment(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	caseID := uuid.FromStringOrNil("f1b2c3d4-9802-9802-9802-000000000001")
	tagID1 := uuid.FromStringOrNil("f1b2c3d4-9802-9802-9802-000000000002")
	tagID2 := uuid.FromStringOrNil("f1b2c3d4-9802-9802-9802-000000000003")

	if err := h.CaseTagAssignmentCreate(ctx, caseID, tagID1); err != nil {
		t.Fatalf("CaseTagAssignmentCreate(tagID1) error = %v", err)
	}
	if err := h.CaseTagAssignmentCreate(ctx, caseID, tagID2); err != nil {
		t.Fatalf("CaseTagAssignmentCreate(tagID2) error = %v", err)
	}

	if err := h.CaseTagAssignmentDelete(ctx, caseID, tagID1); err != nil {
		t.Fatalf("CaseTagAssignmentDelete() error = %v", err)
	}

	res, err := h.CaseTagAssignmentListByCaseID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseTagAssignmentListByCaseID() error = %v", err)
	}
	if len(res) != 1 || res[0] != tagID2 {
		t.Errorf("expected only tagID2 to remain, got: %v", res)
	}
}
