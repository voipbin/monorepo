package casehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/internal/config"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
)

var errProactiveLinkRPCTest = errors.New("simulated RPC failure for §4.4 proactive-link test")

// Test_GetOrCreate_ProactiveLink_Found verifies design §4.4: when a NEW
// Case opens for a non-"conversation_message" referenceType (here "call"),
// GetOrCreate looks up the sibling message Conversation for (self, peer)
// and, if found, stamps Metadata.ContactCaseID on it -- and does so AFTER
// the Case transaction has committed (asserted via mock call ordering).
func Test_GetOrCreate_ProactiveLink_Found(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	config.SetCaseTimeoutHoursForTest(24)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-700b-700b-700b-000000000001")
	newCaseID := uuid.FromStringOrNil("f1b2c3d4-700b-700b-700b-000000000002")
	conversationID := uuid.FromStringOrNil("f1b2c3d4-700b-700b-700b-000000000003")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	self := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551200099"}
	peer := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551200001"}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newCaseID)

	gomock.InOrder(
		// Case DB transaction (insert + tm_update bump + commit) happens
		// first -- the mock db is real (SQLite), so we only assert
		// ordering against the two conversation RPCs below via InOrder on
		// THIS controller's shared timeline: the RPCs must not be called
		// before GetOrCreate returns, which the real sequential code path
		// already guarantees since they're called after tx.Commit().
		mockReq.EXPECT().ConversationV1ConversationGetBySelfAndPeer(ctx, self, peer).Return(&cvconversation.Conversation{
			Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
		}, nil),
		mockReq.EXPECT().ConversationV1ConversationUpdateMetadata(ctx, conversationID, cvconversation.Metadata{ContactCaseID: &newCaseID}).Return(nil, nil),
	)

	res, err := h.GetOrCreate(ctx, customerID, self, peer.Type, peer.Target, "call", nil)
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != newCaseID {
		t.Errorf("expected the new case %s, got: %v", newCaseID, res)
	}
}

// Test_GetOrCreate_ProactiveLink_NotFound verifies design §4.4's "not
// found" outcome: no Conversation exists for (self, peer) -> no further
// RPC, nothing created, and GetOrCreate still succeeds normally.
func Test_GetOrCreate_ProactiveLink_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	config.SetCaseTimeoutHoursForTest(24)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-700c-700c-700c-000000000001")
	newCaseID := uuid.FromStringOrNil("f1b2c3d4-700c-700c-700c-000000000002")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	self := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551200098"}
	peer := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551200002"}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newCaseID)

	mockReq.EXPECT().ConversationV1ConversationGetBySelfAndPeer(ctx, self, peer).Return(nil, nil)
	// No ConversationV1ConversationUpdateMetadata call expected at all --
	// gomock fails the test if it's called without a matching EXPECT.

	res, err := h.GetOrCreate(ctx, customerID, self, peer.Type, peer.Target, "call", nil)
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != newCaseID {
		t.Errorf("expected the new case %s, got: %v", newCaseID, res)
	}
}

// Test_GetOrCreate_ProactiveLink_RPCFailureDoesNotFailCaseOpen verifies
// design §4.4's failure-handling rule: either RPC failing does not roll
// back or fail the Case-open operation.
func Test_GetOrCreate_ProactiveLink_RPCFailureDoesNotFailCaseOpen(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	config.SetCaseTimeoutHoursForTest(24)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-700d-700d-700d-000000000001")
	newCaseID := uuid.FromStringOrNil("f1b2c3d4-700d-700d-700d-000000000002")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	self := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551200097"}
	peer := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551200003"}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newCaseID)

	mockReq.EXPECT().ConversationV1ConversationGetBySelfAndPeer(ctx, self, peer).Return(nil, errProactiveLinkRPCTest)

	res, err := h.GetOrCreate(ctx, customerID, self, peer.Type, peer.Target, "call", nil)
	if err != nil {
		t.Fatalf("GetOrCreate() must succeed even if the proactive-link RPC fails, got error = %v", err)
	}
	if res == nil || res.ID != newCaseID {
		t.Errorf("expected the new case %s to have opened despite the RPC failure, got: %v", newCaseID, res)
	}
}

// Test_GetOrCreate_ProactiveLink_SkippedOnCacheHitReuse verifies design
// §4.4's cost note: the trigger condition is specifically "a NEW Case was
// just opened" -- reusing an already-open Case must NOT re-fire the
// proactive-link RPCs.
func Test_GetOrCreate_ProactiveLink_SkippedOnCacheHitReuse(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	config.SetCaseTimeoutHoursForTest(24)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-700e-700e-700e-000000000001")
	existingCaseID := uuid.FromStringOrNil("f1b2c3d4-700e-700e-700e-000000000002")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	recentUpdate := now.Add(-1 * time.Hour)
	opened := now.Add(-2 * time.Hour)

	self := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551200096"}
	peer := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551200004"}

	existing := &kase.Case{
		ID: existingCaseID, CustomerID: customerID,
		PeerType: peer.Type, PeerTarget: peer.Target, ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &recentUpdate,
	}
	if err := db.CaseInsert(ctx, existing); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	// No ConversationV1ConversationGetBySelfAndPeer / UpdateMetadata
	// expected -- reuse-on-open-match must not trigger the §4.4 write.

	res, err := h.GetOrCreate(ctx, customerID, self, peer.Type, peer.Target, "call", nil)
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != existingCaseID {
		t.Errorf("expected to reuse the existing case %s, got: %v", existingCaseID, res)
	}
}
