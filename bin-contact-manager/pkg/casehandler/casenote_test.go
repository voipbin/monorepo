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
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_CaseNoteCreate_NeverUsesPublishWebhookEvent is the design §3.5
// / §7 mandatory negative test: case_note_created MUST be published via
// the plain notifyHandler.PublishEvent() primitive -- NEVER
// PublishWebhookEvent(). CaseNote is internal, agent-facing content that
// must never reach a customer webhook.
func Test_CaseNoteCreate_NeverUsesPublishWebhookEvent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000002")
	authorID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000003")
	noteID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000004")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().UUIDCreate().Return(noteID)
	mockUtil.EXPECT().TimeNow().Return(&now)

	// The mandatory negative assertion: PublishEvent exactly once,
	// PublishWebhookEvent NEVER. gomock fails the test if
	// PublishWebhookEvent is called without a matching EXPECT (Times(0)
	// makes any call a hard failure, not just an unasserted no-op).
	mockNotify.EXPECT().PublishEvent(ctx, "case_note_created", gomock.Any()).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	res, err := h.CaseNoteCreate(ctx, customerID, caseID, casenote.AuthorTypeAgent, &authorID, "customer confirmed the outage is resolved")
	if err != nil {
		t.Fatalf("CaseNoteCreate() error = %v", err)
	}
	if res.ID != noteID || res.CaseID != caseID || res.Text != "customer confirmed the outage is resolved" {
		t.Errorf("unexpected CaseNote result: %+v", res)
	}
}

// Test_CaseNoteDelete_PublishesCaseNoteDeletedViaPlainEvent verifies
// delete also uses the plain PublishEvent, never PublishWebhookEvent.
func Test_CaseNoteDelete_PublishesCaseNoteDeletedViaPlainEvent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9602-9602-9602-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9602-9602-9602-000000000002")
	noteID := uuid.FromStringOrNil("f1b2c3d4-9602-9602-9602-000000000003")
	createTime := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().UUIDCreate().Return(noteID)
	mockUtil.EXPECT().TimeNow().Return(&createTime)
	mockNotify.EXPECT().PublishEvent(ctx, "case_note_created", gomock.Any())
	if _, err := h.CaseNoteCreate(ctx, customerID, caseID, casenote.AuthorTypeAgent, nil, "note text"); err != nil {
		t.Fatalf("CaseNoteCreate() error = %v", err)
	}

	mockNotify.EXPECT().PublishEvent(ctx, "case_note_deleted", gomock.Any()).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	if err := h.CaseNoteDelete(ctx, customerID, caseID, noteID); err != nil {
		t.Fatalf("CaseNoteDelete() error = %v", err)
	}

	res, err := h.CaseNoteListByCase(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("CaseNoteListByCase() error = %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected the deleted note to be excluded from the list, got: %+v", res)
	}
}
