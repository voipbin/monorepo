package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

// Test_ConversationUpdate_Metadata verifies that a map-based
// ConversationUpdate call carrying a bare (non-pointer) conversation.Metadata
// value round-trips correctly through the DB: PrepareFields' map path
// auto-detects the struct kind and JSON-marshals it, and the row read
// back correctly unmarshals into the *pointer* Conversation.Metadata
// field. This is Phase 1 Task 1.6 of the contact-case-management
// implementation plan (UpdateMetadata RPC).
func Test_ConversationUpdate_Metadata(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-0009-0009-0009-000000000001")

	conv := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("f1b2c3d4-0009-0009-0009-000000000002"),
			CustomerID: uuid.FromStringOrNil("f1b2c3d4-0009-0009-0009-000000000003"),
		},
		Type: conversation.TypeMessage,
		Self: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110000"},
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15552220000"},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	createTime := func() *time.Time { t := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC); return &t }()
	mockUtil.EXPECT().TimeNow().Return(createTime)
	mockCache.EXPECT().ConversationSet(ctx, gomock.Any())
	if err := h.ConversationCreate(ctx, conv); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	updateTime := func() *time.Time { t := time.Date(2026, 7, 7, 10, 1, 0, 0, time.UTC); return &t }()
	mockUtil.EXPECT().TimeNow().Return(updateTime)
	mockCache.EXPECT().ConversationSet(ctx, gomock.Any()).Return(nil)

	fields := map[conversation.Field]any{
		conversation.FieldMetadata: conversation.Metadata{ContactCaseID: &caseID},
	}
	if err := h.ConversationUpdate(ctx, conv.ID, fields); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.conversationGetFromDB(ctx, conv.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if res.Metadata == nil || res.Metadata.ContactCaseID == nil {
		t.Fatalf("expected Metadata.ContactCaseID to be set, got: %+v", res.Metadata)
	}
	if !reflect.DeepEqual(*res.Metadata.ContactCaseID, caseID) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", caseID, *res.Metadata.ContactCaseID)
	}
}
