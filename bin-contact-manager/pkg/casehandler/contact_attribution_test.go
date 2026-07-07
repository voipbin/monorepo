package casehandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_DeriveCaseContactID_ActivePositive verifies design §3.4's
// deriveCaseContactID: an active (not soft-deleted), case-level,
// positive Resolution (interaction_id IS NULL) derives that contact_id.
func Test_DeriveCaseContactID_ActivePositive(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9201-9201-9201-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9201-9201-9201-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9201-9201-9201-000000000003")
	agentID := uuid.FromStringOrNil("f1b2c3d4-9201-9201-9201-000000000004")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	r := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-9201-9201-9201-000000000005"),
		CustomerID:     customerID,
		ContactID:      contactID,
		CaseID:         &caseID,
		ResolutionType: resolution.ResolutionTypePositive,
		ResolvedByType: "agent",
		ResolvedByID:   agentID,
		TMCreate:       &now,
	}
	if err := db.ResolutionCreate(ctx, r); err != nil {
		t.Fatalf("ResolutionCreate() error = %v", err)
	}

	got, err := h.deriveCaseContactID(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("deriveCaseContactID() error = %v", err)
	}
	if got == nil || *got != contactID {
		t.Errorf("expected contact_id: %s, got: %v", contactID, got)
	}
}

// Test_DeriveCaseContactID_NoActiveResolution verifies the "none"
// branch: an unresolved case (no Resolution rows at all) derives nil.
func Test_DeriveCaseContactID_NoActiveResolution(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	caseID := uuid.FromStringOrNil("f1b2c3d4-9202-9202-9202-000000000001")
	customerID := uuid.FromStringOrNil("f1b2c3d4-9202-9202-9202-000000000099")

	got, err := h.deriveCaseContactID(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("deriveCaseContactID() error = %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for a case with no case-level positive resolution, got: %v", *got)
	}
}

// Test_DeriveCaseContactID_IgnoresInteractionScopedAndSoftDeleted
// verifies deriveCaseContactID only counts CASE-level (interaction_id
// IS NULL), active (tm_delete IS NULL) positive resolutions -- an
// interaction-scoped override and a soft-deleted case-level resolution
// must both be ignored.
func Test_DeriveCaseContactID_IgnoresInteractionScopedAndSoftDeleted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9203-9203-9203-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9203-9203-9203-000000000002")
	interactionID := uuid.FromStringOrNil("f1b2c3d4-9203-9203-9203-000000000003")
	wrongContactID := uuid.FromStringOrNil("f1b2c3d4-9203-9203-9203-000000000004")
	deletedContactID := uuid.FromStringOrNil("f1b2c3d4-9203-9203-9203-000000000005")
	agentID := uuid.FromStringOrNil("f1b2c3d4-9203-9203-9203-000000000006")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2026, 6, 28, 13, 0, 0, 0, time.UTC)

	// Interaction-scoped positive resolution (case_id nil) -- must NOT
	// count toward the case-level derivation.
	interactionScoped := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-9203-9203-9203-000000000007"),
		CustomerID:     customerID,
		ContactID:      wrongContactID,
		InteractionID:  &interactionID,
		ResolutionType: resolution.ResolutionTypePositive,
		ResolvedByType: "agent",
		ResolvedByID:   agentID,
		TMCreate:       &now,
	}
	if err := db.ResolutionCreate(ctx, interactionScoped); err != nil {
		t.Fatalf("ResolutionCreate(interactionScoped) error = %v", err)
	}

	// Soft-deleted case-level positive resolution -- must NOT count.
	softDeleted := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-9203-9203-9203-000000000008"),
		CustomerID:     customerID,
		ContactID:      deletedContactID,
		CaseID:         &caseID,
		ResolutionType: resolution.ResolutionTypePositive,
		ResolvedByType: "agent",
		ResolvedByID:   agentID,
		TMCreate:       &now,
		TMDelete:       &deletedAt,
	}
	if err := db.ResolutionCreate(ctx, softDeleted); err != nil {
		t.Fatalf("ResolutionCreate(softDeleted) error = %v", err)
	}

	got, err := h.deriveCaseContactID(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("deriveCaseContactID() error = %v", err)
	}
	if got != nil {
		t.Errorf("expected nil (neither resolution should count), got: %v", *got)
	}
}
