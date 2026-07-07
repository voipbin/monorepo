package casehandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_ReconcileContact_OverwritesDriftedContactID verifies design
// §3.4's recovery path: case-control's `reconcile-contact` re-runs
// deriveCaseContactID and overwrites Case.contact_id, correcting drift
// (e.g. a bulk import wrote Resolution rows without going through the
// handler, leaving Case.contact_id stale/wrong).
func Test_ReconcileContact_OverwritesDriftedContactID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9901-9901-9901-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9901-9901-9901-000000000002")
	staleContactID := uuid.FromStringOrNil("f1b2c3d4-9901-9901-9901-000000000003")
	correctContactID := uuid.FromStringOrNil("f1b2c3d4-9901-9901-9901-000000000004")
	agentID := uuid.FromStringOrNil("f1b2c3d4-9901-9901-9901-000000000005")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	resTime := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)

	// Case with a stale contact_id (simulating drift from a bulk import
	// that wrote it directly, bypassing deriveCaseContactID).
	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551900001", ReferenceType: "call",
		ContactID: &staleContactID,
		Status:    kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	// The actual source of truth: a case-level positive Resolution
	// pointing at a DIFFERENT (correct) contact_id.
	r := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-9901-9901-9901-000000000006"),
		CustomerID:     customerID,
		ContactID:      correctContactID,
		CaseID:         &caseID,
		ResolutionType: resolution.ResolutionTypePositive,
		ResolvedByType: "agent",
		ResolvedByID:   agentID,
		TMCreate:       &resTime,
	}
	if err := db.ResolutionCreate(ctx, r); err != nil {
		t.Fatalf("ResolutionCreate() error = %v", err)
	}

	if err := h.ReconcileContact(ctx, caseID); err != nil {
		t.Fatalf("ReconcileContact() error = %v", err)
	}

	reread, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if reread.ContactID == nil || *reread.ContactID != correctContactID {
		t.Errorf("expected drift corrected to %s, got: %v", correctContactID, reread.ContactID)
	}

	// Idempotency: running it again produces the same result, no error,
	// no side effects on the second run.
	if err := h.ReconcileContact(ctx, caseID); err != nil {
		t.Fatalf("ReconcileContact() (second run) error = %v", err)
	}
	reread2, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() (second run) error = %v", err)
	}
	if reread2.ContactID == nil || *reread2.ContactID != correctContactID {
		t.Errorf("expected idempotent re-run to leave contact_id at %s, got: %v", correctContactID, reread2.ContactID)
	}
}

// Test_ReconcileContact_ClearsWhenNoActiveResolution verifies the
// nil-derivation branch: a Case with a stale contact_id but no active
// case-level positive Resolution gets its contact_id cleared to NULL.
func Test_ReconcileContact_ClearsWhenNoActiveResolution(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9902-9902-9902-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9902-9902-9902-000000000002")
	staleContactID := uuid.FromStringOrNil("f1b2c3d4-9902-9902-9902-000000000003")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551900002", ReferenceType: "call",
		ContactID: &staleContactID,
		Status:    kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	if err := h.ReconcileContact(ctx, caseID); err != nil {
		t.Fatalf("ReconcileContact() error = %v", err)
	}

	reread, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if reread.ContactID != nil {
		t.Errorf("expected contact_id cleared to nil, got: %v", *reread.ContactID)
	}
}
