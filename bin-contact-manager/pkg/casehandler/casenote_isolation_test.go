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

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_CaseNote_NeverLeaksIntoInteractionList is the design §3.5 /
// implementation-plan-mandated negative test: CaseNote content must
// NEVER appear in the customer-facing Interaction timeline.
//
// Post-PR #1137 (contact_interactions -> peer_events retirement), the
// customer-facing timeline (contacthandler.InteractionList) no longer
// reads any local MySQL table at all -- it proxies
// bin-timeline-manager's peer_events read API over RabbitMQ RPC
// (reqHandler.TimelineV1PeerEventList). CaseNote, in contrast, is
// created and stored entirely in contact_case_notes (MySQL, this
// service's own DB) and published only via the plain
// notifyHandler.PublishEvent() primitive -- never PublishWebhookEvent(),
// and never any call into reqHandler at all.
//
// The isolation guarantee this test proves is therefore now structural,
// not query-level: CaseNoteCreate must never invoke ANY RequestHandler
// RPC (in particular, it must never touch the peer_events pipeline that
// backs InteractionList). mockReq has zero EXPECT() calls configured --
// gomock's default strict mode fails the test immediately if
// CaseNoteCreate calls any unexpected method on it, which is exactly
// the "leaked into the Interaction timeline's RPC path" failure mode
// this test guards against.
func Test_CaseNote_NeverLeaksIntoInteractionList(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc) // zero EXPECT() calls configured -- any call is a hard failure
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9701-9701-9701-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9701-9701-9701-000000000002")
	noteID := uuid.FromStringOrNil("f1b2c3d4-9701-9701-9701-000000000004")
	secretText := "SECRET_INTERNAL_NOTE_MUST_NEVER_LEAK_a1b2c3"
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"}, ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &now, TMCreate: &now, TMUpdate: &now,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().UUIDCreate().Return(noteID)
	mockUtil.EXPECT().TimeNow().Return(&now)
	mockNotify.EXPECT().PublishEvent(ctx, "case_note_created", gomock.Any())
	note, err := h.CaseNoteCreate(ctx, customerID, caseID, casenote.AuthorTypeAgent, nil, secretText)
	if err != nil {
		t.Fatalf("CaseNoteCreate() error = %v", err)
	}
	if note.ID != noteID {
		t.Fatalf("CaseNoteCreate() note.ID = %v, want %v", note.ID, noteID)
	}

	// Verify the note landed only in contact_case_notes, scoped to this
	// case -- never in any structure InteractionList could observe.
	notes, err := db.CaseNoteListByCase(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("CaseNoteListByCase() error = %v", err)
	}
	found := false
	for _, n := range notes {
		if n.ID == noteID {
			found = true
			if n.Text != secretText {
				t.Errorf("CaseNoteListByCase()[note].Text = %q, want %q", n.Text, secretText)
			}
		}
	}
	if !found {
		t.Fatalf("CaseNoteListByCase() did not return the created note %v", noteID)
	}

	// mc.Finish() (deferred above) additionally asserts every mockReq
	// EXPECT() was satisfied -- since none were configured, this passes
	// only if CaseNoteCreate genuinely never called reqHandler. If a
	// future change accidentally routes CaseNote through
	// TimelineV1PeerEventList (or any other RequestHandler RPC), gomock
	// fails this test with "unexpected call" the moment it happens.
}
