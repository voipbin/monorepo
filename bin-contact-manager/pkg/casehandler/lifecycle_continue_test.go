package casehandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_Continue_ByOwningAgent_CreatesChainedCase verifies design §5.3:
// the owning agent of a closed case can /continue it -- a brand-new
// open Case is created with previous_case_id pointing at the source
// case, same (peer_type, peer_target, reference_type, contact_id), and
// the source case itself is left completely untouched.
func Test_Continue_ByOwningAgent_CreatesChainedCase(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9101-9101-9101-000000000001")
	sourceCaseID := uuid.FromStringOrNil("f1b2c3d4-9101-9101-9101-000000000002")
	newCaseID := uuid.FromStringOrNil("f1b2c3d4-9101-9101-9101-000000000003")
	agentID := uuid.FromStringOrNil("f1b2c3d4-9101-9101-9101-000000000004")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9101-9101-9101-000000000005")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	closedAt := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	source := &kase.Case{
		ID: sourceCaseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551400001", ReferenceType: "call",
		ContactID: &contactID,
		Owner:     commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: agentID},
		Status:    kase.StatusClosed, OpenedAt: &opened, ClosedAt: &closedAt,
		ClosedReason: kase.ClosedReasonAgentClosed, ClosedByType: kase.ClosedByTypeAgent, ClosedByID: &agentID,
		TMCreate: &opened, TMUpdate: &closedAt,
	}
	if err := db.CaseInsert(ctx, source); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newCaseID)

	res, err := h.Continue(ctx, customerID, sourceCaseID, commonidentity.OwnerTypeAgent, agentID, false)
	if err != nil {
		t.Fatalf("Continue() error = %v", err)
	}
	if res.ID != newCaseID {
		t.Errorf("expected new case %s, got: %v", newCaseID, res.ID)
	}
	if res.PreviousCaseID == nil || *res.PreviousCaseID != sourceCaseID {
		t.Errorf("expected previous_case_id: %s, got: %v", sourceCaseID, res.PreviousCaseID)
	}
	if res.PeerTarget != source.PeerTarget || res.ReferenceType != source.ReferenceType {
		t.Errorf("expected new case to inherit peer/reference_type from source")
	}
	if res.ContactID == nil || *res.ContactID != contactID {
		t.Errorf("expected new case to inherit contact_id %s, got: %v", contactID, res.ContactID)
	}
	if res.Status != kase.StatusOpen {
		t.Errorf("expected new case status open, got: %s", res.Status)
	}

	// The source case itself must remain completely untouched.
	reread, err := db.CaseGetByID(ctx, sourceCaseID)
	if err != nil {
		t.Fatalf("CaseGetByID(source) error = %v", err)
	}
	if reread.Status != kase.StatusClosed || reread.ClosedByID == nil || *reread.ClosedByID != agentID {
		t.Errorf("expected source case to remain untouched (closed, closed_by=%s), got: %+v", agentID, reread)
	}
}

// Test_Continue_RequiresSourceClosed verifies /continue rejects a
// still-open source case.
func Test_Continue_RequiresSourceClosed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9102-9102-9102-000000000001")
	openCaseID := uuid.FromStringOrNil("f1b2c3d4-9102-9102-9102-000000000002")
	agentID := uuid.FromStringOrNil("f1b2c3d4-9102-9102-9102-000000000003")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	source := &kase.Case{
		ID: openCaseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551400002", ReferenceType: "call",
		Owner:  commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: agentID},
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, source); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	_, err := h.Continue(ctx, customerID, openCaseID, commonidentity.OwnerTypeAgent, agentID, false)
	if err != ErrCaseNotClosed {
		t.Errorf("expected ErrCaseNotClosed for a still-open source case, got: %v", err)
	}
}

// Test_Continue_RequiresOwningAgentOrAdmin verifies the authorization
// rule: a non-owning agent cannot /continue someone else's closed case;
// an authorized caller (owning agent OR admin/manager, decided by the
// API layer's permission gate and passed in as callerIsAuthorized) can.
func Test_Continue_RequiresOwningAgentOrAdmin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9103-9103-9103-000000000001")
	closedCaseID := uuid.FromStringOrNil("f1b2c3d4-9103-9103-9103-000000000002")
	owningAgentID := uuid.FromStringOrNil("f1b2c3d4-9103-9103-9103-000000000003")
	otherAgentID := uuid.FromStringOrNil("f1b2c3d4-9103-9103-9103-000000000004")
	adminID := uuid.FromStringOrNil("f1b2c3d4-9103-9103-9103-000000000005")
	newCaseID := uuid.FromStringOrNil("f1b2c3d4-9103-9103-9103-000000000006")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	closedAt := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	source := &kase.Case{
		ID: closedCaseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551400003", ReferenceType: "call",
		Owner:  commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: owningAgentID},
		Status: kase.StatusClosed, OpenedAt: &opened, ClosedAt: &closedAt,
		ClosedReason: kase.ClosedReasonAgentClosed, ClosedByType: kase.ClosedByTypeAgent, ClosedByID: &owningAgentID,
		TMCreate: &opened, TMUpdate: &closedAt,
	}
	if err := db.CaseInsert(ctx, source); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	// Non-owning agent, not admin -> rejected, no DB mutation attempted.
	_, err := h.Continue(ctx, customerID, closedCaseID, commonidentity.OwnerTypeAgent, otherAgentID, false)
	if err != ErrCaseContinueForbidden {
		t.Errorf("expected ErrCaseContinueForbidden for a non-owning, non-admin agent, got: %v", err)
	}

	// Admin (callerIsAdmin=true) -> allowed regardless of ownership.
	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newCaseID)
	res, err := h.Continue(ctx, customerID, closedCaseID, commonidentity.OwnerTypeAgent, adminID, true)
	if err != nil {
		t.Fatalf("Continue() (admin) error = %v", err)
	}
	if res.ID != newCaseID {
		t.Errorf("expected new case %s for admin-initiated continue, got: %v", newCaseID, res.ID)
	}
}
