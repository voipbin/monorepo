package dbhandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// seedResolutionParents inserts a contact + interaction for resolution tests.
// Each caller uses a distinct namespace via the ns UUID prefix bytes.
// Since we ignore ContactCreate errors (contact unique key on id) and
// InteractionCreate is idempotent, repeated calls with the same ns are safe.
func seedResolutionParents(
	t *testing.T,
	h handler,
	ctx context.Context,
	customerID, contactID, interactionID uuid.UUID,
	mockUtil *utilhandler.MockUtilHandler,
	mockCache *cachehandler.MockCacheHandler,
) {
	t.Helper()
	ts := timePtr(time.Date(2026, 6, 28, 8, 0, 0, 0, time.UTC))
	mockUtil.EXPECT().TimeNow().Return(ts)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())

	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "Resolution",
		LastName:  "Test",
		Source:    "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("seedResolutionParents ContactCreate: %v", err)
	}

	iact := &interaction.Interaction{
		ID:            interactionID,
		CustomerID:    customerID,
		Direction:     "incoming",
		PeerType:      "tel",
		PeerTarget:    "+155****0001",
		LocalType:     "tel",
		LocalTarget:   "+155****0001",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("f1b2c3d4-ffff-ffff-ffff-000000000001"),
		TMCreate:      timePtr(time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC)),
	}
	if err := h.InteractionCreate(ctx, iact); err != nil {
		t.Fatalf("seedResolutionParents InteractionCreate: %v", err)
	}
}

func Test_ResolutionCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-1001-1001-1001-000000000001")
	contactID := uuid.FromStringOrNil("f1b2c3d4-1001-1001-1001-000000000002")
	interactionID := uuid.FromStringOrNil("f1b2c3d4-1001-1001-1001-000000000003")
	seedResolutionParents(t, h, ctx, customerID, contactID, interactionID, mockUtil, mockCache)

	r := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-1001-1001-1001-000000000010"),
		CustomerID:     customerID,
		ContactID:      contactID,
		InteractionID:  &interactionID,
		ResolutionType: resolution.ResolutionTypePositive,
		ResolvedByType: resolution.ResolvedByTypeAgent,
		ResolvedByID:   uuid.FromStringOrNil("f1b2c3d4-1001-1001-1001-000000000011"),
		TMCreate:       timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)),
	}

	if err := h.ResolutionCreate(ctx, r); err != nil {
		t.Errorf("ResolutionCreate() error = %v", err)
	}
}

// Test_ResolutionCreate_CaseLevel_NilInteractionID verifies a case-level
// Resolution (InteractionID nil, CaseID set -- contact-case-management
// design §3.3) round-trips through ResolutionCreate against the real
// (SQLite in-memory) contact_resolutions schema, exercising the
// chk_resolution_case_or_interaction CHECK constraint's positive path.
func Test_ResolutionCreate_CaseLevel_NilInteractionID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-1005-1005-1005-000000000001")
	contactID := uuid.FromStringOrNil("f1b2c3d4-1005-1005-1005-000000000002")
	caseID := uuid.FromStringOrNil("f1b2c3d4-1005-1005-1005-000000000003")

	ts := timePtr(time.Date(2026, 6, 28, 8, 0, 0, 0, time.UTC))
	mockUtil.EXPECT().TimeNow().Return(ts)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "CaseResolution",
		LastName:  "Test",
		Source:    "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate: %v", err)
	}

	r := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-1005-1005-1005-000000000010"),
		CustomerID:     customerID,
		ContactID:      contactID,
		InteractionID:  nil,
		CaseID:         &caseID,
		ResolutionType: resolution.ResolutionTypePositive,
		ResolvedByType: resolution.ResolvedByTypeAgent,
		ResolvedByID:   uuid.FromStringOrNil("f1b2c3d4-1005-1005-1005-000000000011"),
		TMCreate:       timePtr(time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)),
	}

	if err := h.ResolutionCreate(ctx, r); err != nil {
		t.Fatalf("ResolutionCreate() error = %v", err)
	}

	res, err := h.ResolutionListByContact(ctx, customerID, contactID)
	if err != nil {
		t.Fatalf("ResolutionListByContact() error = %v", err)
	}
	found := false
	for _, item := range res {
		if item.ID == r.ID {
			found = true
			if item.InteractionID != nil {
				t.Errorf("expected InteractionID nil for case-level resolution, got: %v", *item.InteractionID)
			}
			if item.CaseID == nil || *item.CaseID != caseID {
				t.Errorf("expected CaseID: %v, got: %v", caseID, item.CaseID)
			}
			break
		}
	}
	if !found {
		t.Errorf("ResolutionListByContact() did not return the created case-level resolution")
	}
}

func Test_ResolutionDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-2002-2002-2002-000000000001")
	contactID := uuid.FromStringOrNil("f1b2c3d4-2002-2002-2002-000000000002")
	interactionID := uuid.FromStringOrNil("f1b2c3d4-2002-2002-2002-000000000003")
	seedResolutionParents(t, h, ctx, customerID, contactID, interactionID, mockUtil, mockCache)

	r := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-2002-2002-2002-000000000010"),
		CustomerID:     customerID,
		ContactID:      contactID,
		InteractionID:  &interactionID,
		ResolutionType: resolution.ResolutionTypePositive,
		ResolvedByType: resolution.ResolvedByTypeAgent,
		ResolvedByID:   uuid.FromStringOrNil("f1b2c3d4-2002-2002-2002-000000000011"),
		TMCreate:       timePtr(time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)),
	}
	if err := h.ResolutionCreate(ctx, r); err != nil {
		t.Fatalf("ResolutionCreate() error = %v", err)
	}

	deleteTime := timePtr(time.Date(2026, 6, 28, 23, 0, 0, 0, time.UTC))

	// soft-delete
	mockUtil.EXPECT().TimeNow().Return(deleteTime)
	if err := h.ResolutionDelete(ctx, r.CustomerID, *r.InteractionID, r.ID); err != nil {
		t.Errorf("ResolutionDelete() error = %v", err)
	}

	// delete again → ErrNotFound (already soft-deleted)
	mockUtil.EXPECT().TimeNow().Return(deleteTime)
	if err := h.ResolutionDelete(ctx, r.CustomerID, *r.InteractionID, r.ID); err != ErrNotFound {
		t.Errorf("ResolutionDelete() second delete expected ErrNotFound, got: %v", err)
	}

	// delete non-existent ID → ErrNotFound
	mockUtil.EXPECT().TimeNow().Return(deleteTime)
	if err := h.ResolutionDelete(ctx, r.CustomerID, *r.InteractionID, uuid.FromStringOrNil("f1b2c3d4-2002-ffff-ffff-ffffffffffff")); err != ErrNotFound {
		t.Errorf("ResolutionDelete() non-existent expected ErrNotFound, got: %v", err)
	}

	// cross-tenant: wrong customerID → ErrNotFound (regression guard for ISSUE 1 / BLOCKER fix)
	wrongCustomerID := uuid.FromStringOrNil("f1b2c3d4-2002-dead-beef-000000000000")
	// Insert a fresh resolution with the correct customer to delete
	r2 := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-2002-0000-0000-000000002002"),
		CustomerID:     r.CustomerID,
		ContactID:      r.ContactID,
		InteractionID:  r.InteractionID,
		ResolutionType: resolution.ResolutionTypeNegative,
		ResolvedByType: resolution.ResolvedByTypeAgent,
		ResolvedByID:   uuid.FromStringOrNil("f1b2c3d4-2002-0000-0000-000000003003"),
		TMCreate:       deleteTime,
		TMUpdate:       deleteTime,
	}
	if err := h.ResolutionCreate(ctx, r2); err != nil {
		t.Fatalf("ResolutionCreate(r2) error = %v", err)
	}
	mockUtil.EXPECT().TimeNow().Return(deleteTime)
	if err := h.ResolutionDelete(ctx, wrongCustomerID, *r2.InteractionID, r2.ID); err != ErrNotFound {
		t.Errorf("ResolutionDelete() cross-tenant expected ErrNotFound, got: %v", err)
	}

	// cross-interaction: correct customerID + correct resolutionID + wrong interactionID → ErrNotFound
	r3 := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-2002-0000-0000-000000004004"),
		CustomerID:     r.CustomerID,
		ContactID:      r.ContactID,
		InteractionID:  r.InteractionID,
		ResolutionType: resolution.ResolutionTypePositive,
		ResolvedByType: resolution.ResolvedByTypeSystem,
		ResolvedByID:   resolution.ResolvedByIDSystem,
		TMCreate:       deleteTime,
		TMUpdate:       deleteTime,
	}
	if err := h.ResolutionCreate(ctx, r3); err != nil {
		t.Fatalf("ResolutionCreate(r3) error = %v", err)
	}
	wrongInteractionID := uuid.FromStringOrNil("f1b2c3d4-2002-aaaa-aaaa-000000000000")
	mockUtil.EXPECT().TimeNow().Return(deleteTime)
	if err := h.ResolutionDelete(ctx, r3.CustomerID, wrongInteractionID, r3.ID); err != ErrNotFound {
		t.Errorf("ResolutionDelete() cross-interaction expected ErrNotFound, got: %v", err)
	}
}

func Test_ResolutionListByInteraction(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-3003-3003-3003-000000000001")
	contactID := uuid.FromStringOrNil("f1b2c3d4-3003-3003-3003-000000000002")
	interactionID := uuid.FromStringOrNil("f1b2c3d4-3003-3003-3003-000000000003")
	seedResolutionParents(t, h, ctx, customerID, contactID, interactionID, mockUtil, mockCache)

	r1 := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-3003-3003-3003-000000000010"),
		CustomerID:     customerID,
		ContactID:      contactID,
		InteractionID:  &interactionID,
		ResolutionType: resolution.ResolutionTypePositive,
		ResolvedByType: resolution.ResolvedByTypeAgent,
		ResolvedByID:   uuid.FromStringOrNil("f1b2c3d4-3003-3003-3003-000000000011"),
		TMCreate:       timePtr(time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)),
	}
	r2 := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-3003-3003-3003-000000000020"),
		CustomerID:     customerID,
		ContactID:      contactID,
		InteractionID:  &interactionID,
		ResolutionType: resolution.ResolutionTypeNegative,
		ResolvedByType: resolution.ResolvedByTypeSystem,
		ResolvedByID:   resolution.ResolvedByIDSystem,
		TMCreate:       timePtr(time.Date(2026, 6, 28, 13, 0, 0, 0, time.UTC)),
	}
	if err := h.ResolutionCreate(ctx, r1); err != nil {
		t.Fatalf("ResolutionCreate(r1) error = %v", err)
	}
	if err := h.ResolutionCreate(ctx, r2); err != nil {
		t.Fatalf("ResolutionCreate(r2) error = %v", err)
	}

	// Soft-delete r2; active only list should not include it
	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2026, 6, 28, 23, 0, 0, 0, time.UTC)))
	if err := h.ResolutionDelete(ctx, r2.CustomerID, *r2.InteractionID, r2.ID); err != nil {
		t.Fatalf("ResolutionDelete(r2) error = %v", err)
	}

	res, err := h.ResolutionListByInteraction(ctx, customerID, interactionID)
	if err != nil {
		t.Fatalf("ResolutionListByInteraction() error = %v", err)
	}

	// No soft-deleted rows
	for _, item := range res {
		if item.TMDelete != nil {
			t.Errorf("ResolutionListByInteraction() returned soft-deleted resolution %v", item.ID)
		}
	}

	// r1 should appear
	found := false
	for _, item := range res {
		if item.ID == r1.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ResolutionListByInteraction() r1 not found (len=%d)", len(res))
	}
}

func Test_ResolutionListByContact(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-4004-4004-4004-000000000001")
	contactID := uuid.FromStringOrNil("f1b2c3d4-4004-4004-4004-000000000002")
	interactionID := uuid.FromStringOrNil("f1b2c3d4-4004-4004-4004-000000000003")
	seedResolutionParents(t, h, ctx, customerID, contactID, interactionID, mockUtil, mockCache)

	r1 := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-4004-4004-4004-000000000010"),
		CustomerID:     customerID,
		ContactID:      contactID,
		InteractionID:  &interactionID,
		ResolutionType: resolution.ResolutionTypePositive,
		ResolvedByType: resolution.ResolvedByTypeAgent,
		ResolvedByID:   uuid.FromStringOrNil("f1b2c3d4-4004-4004-4004-000000000011"),
		TMCreate:       timePtr(time.Date(2026, 6, 28, 14, 0, 0, 0, time.UTC)),
	}
	r2 := &resolution.Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-4004-4004-4004-000000000020"),
		CustomerID:     customerID,
		ContactID:      contactID,
		InteractionID:  &interactionID,
		ResolutionType: resolution.ResolutionTypeNegative,
		ResolvedByType: resolution.ResolvedByTypeSystem,
		ResolvedByID:   resolution.ResolvedByIDSystem,
		TMCreate:       timePtr(time.Date(2026, 6, 28, 15, 0, 0, 0, time.UTC)),
	}
	if err := h.ResolutionCreate(ctx, r1); err != nil {
		t.Fatalf("ResolutionCreate(r1) error = %v", err)
	}
	if err := h.ResolutionCreate(ctx, r2); err != nil {
		t.Fatalf("ResolutionCreate(r2) error = %v", err)
	}

	// Soft-delete r2
	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2026, 6, 28, 23, 0, 0, 0, time.UTC)))
	if err := h.ResolutionDelete(ctx, r2.CustomerID, *r2.InteractionID, r2.ID); err != nil {
		t.Fatalf("ResolutionDelete(r2) error = %v", err)
	}

	res, err := h.ResolutionListByContact(ctx, customerID, contactID)
	if err != nil {
		t.Fatalf("ResolutionListByContact() error = %v", err)
	}

	// No soft-deleted rows
	for _, item := range res {
		if item.TMDelete != nil {
			t.Errorf("ResolutionListByContact() returned soft-deleted resolution %v", item.ID)
		}
	}

	// r1 should appear
	found := false
	for _, item := range res {
		if item.ID == r1.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ResolutionListByContact() r1 not found (len=%d)", len(res))
	}
}
