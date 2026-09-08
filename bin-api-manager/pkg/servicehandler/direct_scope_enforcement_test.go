package servicehandler

import (
	stderrors "errors"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	ammessage "monorepo/bin-ai-manager/models/message"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	wcmessage "monorepo/bin-webchat-manager/models/message"
	wcsession "monorepo/bin-webchat-manager/models/session"
	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// These tests cover the enforcement branches that make a direct token usable
// for exactly one resource. Each one asserts the property a public-link holder
// would otherwise be able to break.

var (
	dseCustomerID = uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	dseAIID       = uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001")
	dseAllowedID  = uuid.FromStringOrNil("beefbeef-0000-0000-0000-000000000001")
	dseOtherID    = uuid.FromStringOrNil("deaddead-0000-0000-0000-000000000002")
)

func dseDirect(allowed uuid.UUID) *auth.AuthIdentity {
	return auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           dseCustomerID,
		ResourceType:         "ai",
		ResourceID:           dseAIID,
		AllowedResourceTypes: []string{"aicall"},
		AllowedResourceID:    allowed,
	})
}

func dseHandler(mc *gomock.Controller) (serviceHandler, *requesthandler.MockRequestHandler) {
	mockReq := requesthandler.NewMockRequestHandler(mc)
	return serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: utilhandler.NewMockUtilHandler(mc),
	}, mockReq
}

// Test_AIcallCreate_direct_bindsAndForcesReference covers three things at once
// because they all happen in the same branch:
//
//  1. the new aicall takes the token's assigned id,
//  2. reference_type is forced to none, and
//  3. reference_id is forced to Nil.
//
// (2) and (3) are separately load-bearing. Forcing only the type would still
// leave reference_id non-zero, which keeps the generated active_reference_key
// populated and the unique-index collision surface live. Forcing only the id
// would still route into startReferenceTypeContactCase, whose reuse path calls
// AIcallGetByReferenceID -- a lookup with no customer filter.
func Test_AIcallCreate_direct_bindsAndForcesReference(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := dseHandler(mc)
	ctx := t.Context()

	af := &fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("aaaa0000-0000-0000-0000-000000000001")}}
	mockReq.EXPECT().AIV1AIGet(ctx, dseAIID).Return(&amai.AI{
		Identity: commonidentity.Identity{ID: dseAIID, CustomerID: dseCustomerID},
	}, nil)
	mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, gomock.Any(), dseCustomerID, gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(af, nil)

	// The caller asks for contact_case with someone else's reference id.
	// Both must be discarded.
	mockReq.EXPECT().AIV1AIcallStart(
		ctx,
		dseAllowedID,
		amaicall.AssistanceTypeAI,
		dseAIID,
		af.ID,
		amaicall.ReferenceTypeNone,
		uuid.Nil,
	).Return(&amaicall.AIcall{Identity: commonidentity.Identity{ID: dseAllowedID}}, nil)

	res, err := h.AIcallCreate(ctx, dseDirect(dseAllowedID), amaicall.AssistanceTypeAI, dseAIID,
		amaicall.ReferenceTypeContactCase, dseOtherID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != dseAllowedID {
		t.Errorf("Wrong aicall id. expect: %v, got: %v", dseAllowedID, res.ID)
	}
}

// An unbound token must not reach the create path at all; otherwise the
// overwrite would persist uuid.Nil as a real resource id.
func Test_AIcallCreate_direct_unboundIsRefused(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := dseHandler(mc)
	ctx := t.Context()

	mockReq.EXPECT().AIV1AIGet(ctx, dseAIID).Return(&amai.AI{
		Identity: commonidentity.Identity{ID: dseAIID, CustomerID: dseCustomerID},
	}, nil)

	_, err := h.AIcallCreate(ctx, dseDirect(uuid.Nil), amaicall.AssistanceTypeAI, dseAIID,
		amaicall.ReferenceTypeNone, uuid.Nil)
	if !stderrors.Is(err, serviceerrors.ErrPermissionDenied) {
		t.Errorf("Wrong match. expect: permission denied, got: %v", err)
	}
}

// Listing is the only way a visitor could learn another visitor's resource id,
// and there is no target to compare or overwrite. It is refused outright.
func Test_AIcallGetsByCustomerID_direct_isRefused(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := serviceHandler{
		reqHandler:  requesthandler.NewMockRequestHandler(mc),
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: mockUtil,
	}
	// The default page token is filled in before the authorization switch.
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-09-08T00:00:00.000000Z")

	_, err := h.AIcallGetsByCustomerID(t.Context(), dseDirect(dseAllowedID), 10, "")
	if !stderrors.Is(err, serviceerrors.ErrPermissionDenied) {
		t.Errorf("Wrong match. expect: permission denied, got: %v", err)
	}
}

// Test_AImessageCreate_direct_ignoresCallerAIcallID is the property that makes
// the body/query paths safe without a per-route comparison: the id the caller
// sends is not consulted, so there is no check to forget.
func Test_AImessageCreate_direct_ignoresCallerAIcallID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := dseHandler(mc)
	ctx := t.Context()

	owned := &amaicall.AIcall{Identity: commonidentity.Identity{ID: dseAllowedID, CustomerID: dseCustomerID}}
	// Both the authorization read and the send must target the assignment,
	// never dseOtherID.
	mockReq.EXPECT().AIV1AIcallGet(ctx, dseAllowedID).Return(owned, nil)
	mockReq.EXPECT().AIV1MessageSend(ctx, dseAllowedID, ammessage.RoleUser, "hello", true, false, 30000).
		Return(&ammessage.Message{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("ffff0000-0000-0000-0000-000000000001")}}, nil)

	if _, err := h.AImessageCreate(ctx, dseDirect(dseAllowedID), dseOtherID, ammessage.RoleUser, "hello"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_AImessageCreate_direct_unboundIsRefused(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := dseHandler(mc)

	_, err := h.AImessageCreate(t.Context(), dseDirect(uuid.Nil), dseOtherID, ammessage.RoleUser, "hello")
	if !stderrors.Is(err, serviceerrors.ErrPermissionDenied) {
		t.Errorf("Wrong match. expect: permission denied, got: %v", err)
	}
}

func Test_AImessageGetsByAIcallID_direct_ignoresCallerAIcallID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: mockUtil,
	}
	ctx := t.Context()

	owned := &amaicall.AIcall{Identity: commonidentity.Identity{ID: dseAllowedID, CustomerID: dseCustomerID}}
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-09-08T00:00:00.000000Z")
	mockReq.EXPECT().AIV1AIcallGet(ctx, dseAllowedID).Return(owned, nil)
	// The list RPC must be keyed on the assignment, not on what the caller asked for.
	mockReq.EXPECT().AIV1MessageGetsByAIcallID(ctx, dseAllowedID, gomock.Any(), uint64(100), gomock.Any()).
		Return(nil, nil)

	if _, err := h.AImessageGetsByAIcallID(ctx, dseDirect(dseAllowedID), dseOtherID, 0, ""); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

var (
	dseWidgetID  = uuid.FromStringOrNil("cccc0000-0000-0000-0000-000000000001")
	dseSessionID = uuid.FromStringOrNil("beefbeef-0000-0000-0000-000000000009")
)

func dseDirectWebchat(allowed uuid.UUID) *auth.AuthIdentity {
	return auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           dseCustomerID,
		ResourceType:         "webchat_widget",
		ResourceID:           dseWidgetID,
		AllowedResourceTypes: []string{"webchat_session"},
		AllowedResourceID:    allowed,
	})
}

// The session a visitor creates must take the token's assigned id. Without
// this, every visitor of the same widget gets a server-chosen id and nothing
// downstream can tell them apart.
func Test_WebchatSessionCreate_direct_bindsSessionID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := dseHandler(mc)
	ctx := t.Context()

	w := &wcwidget.Widget{Identity: commonidentity.Identity{ID: dseWidgetID, CustomerID: dseCustomerID}}
	mockReq.EXPECT().WebchatV1WidgetGet(ctx, dseWidgetID).Return(w, nil)
	mockReq.EXPECT().WebchatV1SessionCreate(ctx, dseSessionID, dseCustomerID, dseWidgetID, "", "").
		Return(&wcsession.Session{Identity: commonidentity.Identity{ID: dseSessionID, CustomerID: dseCustomerID}}, nil)

	res, err := h.WebchatSessionCreate(ctx, dseDirectWebchat(dseSessionID), dseWidgetID, "", "")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != dseSessionID {
		t.Errorf("Wrong session id. expect: %v, got: %v", dseSessionID, res.ID)
	}
}

func Test_WebchatSessionCreate_direct_unboundIsRefused(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := dseHandler(mc)
	ctx := t.Context()

	w := &wcwidget.Widget{Identity: commonidentity.Identity{ID: dseWidgetID, CustomerID: dseCustomerID}}
	mockReq.EXPECT().WebchatV1WidgetGet(ctx, dseWidgetID).Return(w, nil)

	_, err := h.WebchatSessionCreate(ctx, dseDirectWebchat(uuid.Nil), dseWidgetID, "", "")
	if !stderrors.Is(err, serviceerrors.ErrPermissionDenied) {
		t.Errorf("Wrong match. expect: permission denied, got: %v", err)
	}
}

// A visitor posting into another visitor's session id must land in their own.
// This is the webchat half of the cross-visitor isolation property.
func Test_WebchatMessageCreate_direct_ignoresCallerSessionID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := dseHandler(mc)
	ctx := t.Context()

	owned := &wcsession.Session{
		Identity: commonidentity.Identity{ID: dseSessionID, CustomerID: dseCustomerID},
		WidgetID: dseWidgetID,
		Status:   wcsession.StatusActive,
	}
	w := &wcwidget.Widget{Identity: commonidentity.Identity{ID: dseWidgetID, CustomerID: dseCustomerID}}

	// Every downstream call must reference the assignment, never dseOtherID.
	mockReq.EXPECT().WebchatV1SessionGet(ctx, dseSessionID).Return(owned, nil)
	mockReq.EXPECT().WebchatV1WidgetGet(ctx, dseWidgetID).Return(w, nil)
	mockReq.EXPECT().WebchatV1MessageCreate(ctx, dseCustomerID, dseSessionID, gomock.Any(), gomock.Any(), "hi").
		Return(&wcmessage.Message{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("dddd0000-0000-0000-0000-000000000001")}}, nil)

	if _, err := h.WebchatMessageCreate(ctx, dseDirectWebchat(dseSessionID), dseOtherID, wcmessage.DirectionInbound, "hi"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}
