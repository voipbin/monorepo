package casehandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	tmtag "monorepo/bin-tag-manager/models/tag"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_CaseTagAdd_ValidatesTagExistsThenAssigns verifies design §7
// round-22: assigning a tag to a Case validates the tag_id exists via
// bin-tag-manager's existing TagV1TagGet (no other tag-manager
// interaction needed -- tag-manager itself is unchanged), then creates
// the case-scoped assignment row.
func Test_CaseTagAdd_ValidatesTagExistsThenAssigns(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9803-9803-9803-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9803-9803-9803-000000000002")
	tagID := uuid.FromStringOrNil("f1b2c3d4-9803-9803-9803-000000000003")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551800001", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockReq.EXPECT().TagV1TagGet(ctx, tagID).Return(&tmtag.Tag{}, nil)

	if err := h.CaseTagAdd(ctx, customerID, caseID, tagID); err != nil {
		t.Fatalf("CaseTagAdd() error = %v", err)
	}

	tags, err := h.CaseTagList(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("CaseTagList() error = %v", err)
	}
	if len(tags) != 1 || tags[0] != tagID {
		t.Errorf("expected [%s], got: %v", tagID, tags)
	}
}

// Test_CaseTagAdd_RejectsNonExistentTag verifies the tag_id existence
// check actually gates the assignment -- a tag that TagV1TagGet reports
// as not found must be rejected without creating an assignment row.
func Test_CaseTagAdd_RejectsNonExistentTag(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9804-9804-9804-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9804-9804-9804-000000000002")
	ghostTagID := uuid.FromStringOrNil("f1b2c3d4-9804-9804-9804-000000000099")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551800002", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockReq.EXPECT().TagV1TagGet(ctx, ghostTagID).Return(nil, dbhandler.ErrNotFound)

	err := h.CaseTagAdd(ctx, customerID, caseID, ghostTagID)
	if err != ErrTagNotFound {
		t.Errorf("expected ErrTagNotFound, got: %v", err)
	}

	tags, err := h.CaseTagList(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("CaseTagList() error = %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected no assignment created, got: %v", tags)
	}
}

// Test_CaseTagRemove_DeletesAssignment verifies unassignment.
func Test_CaseTagRemove_DeletesAssignment(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9805-9805-9805-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9805-9805-9805-000000000002")
	tagID := uuid.FromStringOrNil("f1b2c3d4-9805-9805-9805-000000000003")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551800003", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockReq.EXPECT().TagV1TagGet(ctx, tagID).Return(&tmtag.Tag{}, nil)
	if err := h.CaseTagAdd(ctx, customerID, caseID, tagID); err != nil {
		t.Fatalf("CaseTagAdd() error = %v", err)
	}

	if err := h.CaseTagRemove(ctx, customerID, caseID, tagID); err != nil {
		t.Fatalf("CaseTagRemove() error = %v", err)
	}

	tags, err := h.CaseTagList(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("CaseTagList() error = %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected no tags remaining, got: %v", tags)
	}
}

// Test_CaseTagAdd_CrossTenant_MustNotAssignToOtherTenantsCase is the
// round-2 review's defect regression guard: CaseTagAdd/Remove/List
// accepted a customerID parameter but never verified the target Case
// actually belongs to that customer -- contact_case_tag_assignments has
// no customer_id column of its own, so an attacker who knew (or
// guessed) another tenant's case_id could tag, untag, or list tags on
// it. Fixed by verifyCaseOwnership gating all three methods.
func Test_CaseTagAdd_CrossTenant_MustNotAssignToOtherTenantsCase(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	victimCustomerID := uuid.FromStringOrNil("f1b2c3d4-9806-9806-9806-000000000001")
	attackerCustomerID := uuid.FromStringOrNil("f1b2c3d4-9806-9806-9806-000000000002")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9806-9806-9806-000000000003")
	tagID := uuid.FromStringOrNil("f1b2c3d4-9806-9806-9806-000000000004")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	victimCase := &kase.Case{
		ID: caseID, CustomerID: victimCustomerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****0004", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, victimCase); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	// No TagV1TagGet EXPECT is set: the ownership check must reject
	// BEFORE the tag-existence RPC is even called.
	if err := h.CaseTagAdd(ctx, attackerCustomerID, caseID, tagID); err != dbhandler.ErrNotFound {
		t.Errorf("expected ErrNotFound (tenant isolation must hide existence), got: %v", err)
	}
	if err := h.CaseTagRemove(ctx, attackerCustomerID, caseID, tagID); err != dbhandler.ErrNotFound {
		t.Errorf("expected ErrNotFound for CaseTagRemove, got: %v", err)
	}
	if _, err := h.CaseTagList(ctx, attackerCustomerID, caseID); err != dbhandler.ErrNotFound {
		t.Errorf("expected ErrNotFound for CaseTagList, got: %v", err)
	}

	// The victim's case must have zero tag assignments -- the attacker's
	// calls must never have reached the DB write/read at all.
	tags, err := h.CaseTagList(ctx, victimCustomerID, caseID)
	if err != nil {
		t.Fatalf("CaseTagList() (victim) error = %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("BUG: attacker's cross-tenant CaseTagAdd call created an assignment on the victim's case: %v", tags)
	}
}
