package contacthandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_interactionListByContact_NilInteractionIDResolution_NoPanic is the
// explicit nil-guard regression test required by contact-case-management
// design §3.3 (Task 2.3): a case-level Resolution (InteractionID nil,
// CaseID set) present in a contact's resolution list must not panic the
// set-MINUS map keying in interactionListByContact, and must not be
// mistakenly treated as attributing/suppressing a zero-UUID interaction.
func Test_interactionListByContact_NilInteractionIDResolution_NoPanic(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := contactHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   mockUtil,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9001-9001-9001-000000000001")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9001-9001-9001-000000000002")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9001-9001-9001-000000000003")
	realInteractionID := uuid.FromStringOrNil("f1b2c3d4-9001-9001-9001-000000000004")

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(&contact.Contact{
		Identity: commonidentity.Identity{ID: contactID, CustomerID: customerID},
	}, nil)
	mockDB.EXPECT().OwnershipPeriodsListByContactID(ctx, contactID).Return(nil, nil)
	mockDB.EXPECT().MissingPeriodOwnedAddresses(ctx, customerID, contactID).Return(nil, nil)

	// One case-level Resolution (InteractionID nil) mixed with one
	// ordinary interaction-level Resolution -- the panic/mis-keying risk
	// is specifically about the nil one; the non-nil one must still work.
	mockDB.EXPECT().ResolutionListByContact(ctx, customerID, contactID).Return([]*resolution.Resolution{
		{
			ID:             uuid.FromStringOrNil("f1b2c3d4-9001-9001-9001-000000000010"),
			CustomerID:     customerID,
			ContactID:      contactID,
			InteractionID:  nil,
			CaseID:         &caseID,
			ResolutionType: resolution.ResolutionTypePositive,
		},
		{
			ID:             uuid.FromStringOrNil("f1b2c3d4-9001-9001-9001-000000000011"),
			CustomerID:     customerID,
			ContactID:      contactID,
			InteractionID:  &realInteractionID,
			ResolutionType: resolution.ResolutionTypePositive,
		},
	}, nil)
	mockDB.EXPECT().InteractionListByIDs(ctx, customerID, []uuid.UUID{realInteractionID}).Return([]*interaction.Interaction{
		{
			ID:         realInteractionID,
			CustomerID: customerID,
			TMCreate:   func() *time.Time { tm := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC); return &tm }(),
		},
	}, nil)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("interactionListByContact panicked on a nil-InteractionID (case-level) resolution: %v", r)
		}
	}()

	items, _, err := h.interactionListByContact(ctx, logrus.NewEntry(logrus.New()), customerID, contactID, 20, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The nil-InteractionID resolution must not surface as a spurious
	// zero-UUID interaction in the result.
	for _, item := range items {
		if item.ID == uuid.Nil {
			t.Errorf("result contains a zero-UUID interaction -- the nil-InteractionID resolution was mis-keyed")
		}
	}

	// The real interaction-level positive resolution must still surface
	// its interaction (positive-only-path, STEP 5 of the set-MINUS algorithm).
	found := false
	for _, item := range items {
		if item.ID == realInteractionID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected the real interaction-level positive resolution's interaction to surface, got: %v", items)
	}
}

// Test_interactionListByContact_BoundsComposition_STEP1STEP2Wiring is a
// non-trivial exercise of interaction_read.go's bounds-composition glue
// (STEP1/STEP2, interaction_read.go's periods+skewed -> bounds loop):
// unlike Test_interactionListByContact_NilInteractionIDResolution_NoPanic,
// which stubs OwnershipPeriodsListByContactID/MissingPeriodOwnedAddresses
// to (nil, nil) and never exercises this composition at all (bounds stays
// empty, InteractionListByOwnershipPeriods is never even called), this
// test supplies one real closed period AND one real missing-period-skew
// address, and asserts InteractionListByOwnershipPeriods is called with
// exactly the bounds the composition loop is supposed to build from them
// (design §6.2: periods -> bounded entries, skewed -> unbounded entries).
func Test_interactionListByContact_BoundsComposition_STEP1STEP2Wiring(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := contactHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   mockUtil,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9002-9002-9002-000000000001")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9002-9002-9002-000000000002")

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(&contact.Contact{
		Identity: commonidentity.Identity{ID: contactID, CustomerID: customerID},
	}, nil)

	validFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	validTo := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	mockDB.EXPECT().OwnershipPeriodsListByContactID(ctx, contactID).Return([]dbhandler.OwnershipPeriod{
		{
			ID:         uuid.FromStringOrNil("f1b2c3d4-9002-9002-9002-000000000010"),
			CustomerID: customerID,
			ContactID:  contactID,
			Type:       "tel",
			Target:     "+15559991001",
			ValidFrom:  &validFrom,
			ValidTo:    &validTo,
		},
	}, nil)
	mockDB.EXPECT().MissingPeriodOwnedAddresses(ctx, customerID, contactID).Return([]dbhandler.MissingPeriodAddress{
		{Type: "email", Target: "skewed@example.com"},
	}, nil)

	// The composition MUST produce exactly two bounds: the period entry
	// carrying its real [validFrom, validTo) window, and the skewed
	// entry carrying an UNBOUNDED (nil, nil) window -- asserted via
	// gomock.Eq's structural match on OwnershipPeriodBound, not a
	// gomock.Any() passthrough, so this test actually fails if the
	// composition loop drops a field or gets the bound shape wrong.
	wantBounds := []dbhandler.OwnershipPeriodBound{
		{Type: "tel", Target: "+15559991001", ValidFrom: &validFrom, ValidTo: &validTo},
		{Type: "email", Target: "skewed@example.com", ValidFrom: nil, ValidTo: nil},
	}
	interactionID := uuid.FromStringOrNil("f1b2c3d4-9002-9002-9002-000000000011")
	mockDB.EXPECT().InteractionListByOwnershipPeriods(ctx, customerID, interactionInternalCap, "", "", "", wantBounds, time.Time{}).
		Return([]*interaction.Interaction{
			{ID: interactionID, CustomerID: customerID, PeerType: "tel", PeerTarget: "+15559991001"},
		}, nil)

	mockDB.EXPECT().ResolutionListByContact(ctx, customerID, contactID).Return(nil, nil)

	items, _, err := h.interactionListByContact(ctx, logrus.NewEntry(logrus.New()), customerID, contactID, 20, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].ID != interactionID {
		t.Errorf("expected exactly the one interaction InteractionListByOwnershipPeriods returned, got: %+v", items)
	}
}
