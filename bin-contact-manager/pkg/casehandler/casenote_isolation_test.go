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

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_CaseNote_NeverLeaksIntoInteractionList is the design §3.5 /
// implementation-plan-mandated negative test: CaseNote content must
// NEVER appear in the customer-facing Interaction timeline (the
// contact_interactions table, the source of every customer-visible
// message/call record and webhook payload). CaseNote lives in a
// physically separate table (contact_case_notes) with no join or
// projection path into Interaction reads -- this test proves that
// separation holds at the query level, not just "by construction".
func Test_CaseNote_NeverLeaksIntoInteractionList(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9701-9701-9701-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9701-9701-9701-000000000002")
	interactionID := uuid.FromStringOrNil("f1b2c3d4-9701-9701-9701-000000000003")
	noteID := uuid.FromStringOrNil("f1b2c3d4-9701-9701-9701-000000000004")
	secretText := "SECRET_INTERNAL_NOTE_MUST_NEVER_LEAK_a1b2c3"
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	// A real customer-visible Interaction for the same peer/case, so the
	// list path has at least one legitimate row to return.
	i := &interaction.Interaction{
		ID: interactionID, CustomerID: customerID,
		Direction: "incoming", PeerType: "tel", PeerTarget: "+15551700001",
		LocalType: "tel", LocalTarget: "+15551799999",
		ReferenceType: "call", ReferenceID: uuid.Must(uuid.NewV4()),
		CaseID:        &caseID,
		TMInteraction: &now, TMCreate: &now,
	}
	if err := db.InteractionCreate(ctx, i); err != nil {
		t.Fatalf("InteractionCreate() error = %v", err)
	}

	mockUtil.EXPECT().UUIDCreate().Return(noteID)
	mockUtil.EXPECT().TimeNow().Return(&now)
	mockNotify.EXPECT().PublishEvent(ctx, "case_note_created", gomock.Any())
	if _, err := h.CaseNoteCreate(ctx, customerID, caseID, casenote.AuthorTypeAgent, nil, secretText); err != nil {
		t.Fatalf("CaseNoteCreate() error = %v", err)
	}

	// The customer-facing read path: list interactions by peer, exactly
	// as the public Interaction API does.
	items, err := db.InteractionList(ctx, customerID, 20, "", "tel", "+15551700001", nil, time.Time{})
	if err != nil {
		t.Fatalf("InteractionList() error = %v", err)
	}

	if len(items) == 0 {
		t.Fatalf("expected at least the one real Interaction, got 0")
	}
	for _, it := range items {
		if it.ID == noteID {
			t.Errorf("CaseNote id %s leaked into InteractionList() as if it were an Interaction", noteID)
		}
	}
}
