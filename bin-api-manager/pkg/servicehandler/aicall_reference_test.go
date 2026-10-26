package servicehandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"
	cmkase "monorepo/bin-contact-manager/models/kase"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const (
	refTestCustomer = "a0000000-0001-11f0-8000-000000000001"
	refTestOther    = "a0000000-0002-11f0-8000-000000000001"
	refTestRef      = "a0000000-0003-11f0-8000-000000000001"
	refTestAI       = "a0000000-0004-11f0-8000-000000000001"
	refTestAgent    = "a0000000-0005-11f0-8000-000000000001"
)

func refDeleteTime(deleted bool) *time.Time {
	if !deleted {
		return nil
	}
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return &ts
}

func Test_aicallReferenceValidate(t *testing.T) {
	customerID := uuid.FromStringOrNil(refTestCustomer)
	otherID := uuid.FromStringOrNil(refTestOther)
	refID := uuid.FromStringOrNil(refTestRef)

	typedNotFound := cerrors.NotFound(commonoutline.ServiceNameCallManager, "CALL_NOT_FOUND", "call not found")
	typedDenied := cerrors.PermissionDenied(commonoutline.ServiceNameCallManager, "DENIED", "denied")

	// getErr/ownerID/deleted describe what the backend returns for the lookup.
	tests := []struct {
		name string

		referenceType amaicall.ReferenceType
		referenceID   uuid.UUID

		lookup  bool // whether a backend lookup is expected
		getErr  error
		ownerID uuid.UUID
		deleted bool

		expectID  uuid.UUID
		expectErr error // matched with errors.Is; nil means success
	}{
		// none
		{name: "none with nil id", referenceType: amaicall.ReferenceTypeNone, referenceID: uuid.Nil, expectID: uuid.Nil},
		{name: "none with non-nil id forces nil", referenceType: amaicall.ReferenceTypeNone, referenceID: refID, expectID: uuid.Nil},

		// unknown types
		{name: "task is not allowed", referenceType: amaicall.ReferenceTypeTask, referenceID: refID, expectErr: serviceerrors.ErrInvalidArgument},
		{name: "unknown type", referenceType: amaicall.ReferenceType("bogus"), referenceID: refID, expectErr: serviceerrors.ErrInvalidArgument},
		{name: "different case", referenceType: amaicall.ReferenceType("Call"), referenceID: refID, expectErr: serviceerrors.ErrInvalidArgument},
		{name: "padded", referenceType: amaicall.ReferenceType(" call"), referenceID: refID, expectErr: serviceerrors.ErrInvalidArgument},

		// nil id with a typed reference
		{name: "call with nil id", referenceType: amaicall.ReferenceTypeCall, referenceID: uuid.Nil, expectErr: serviceerrors.ErrInvalidArgument},
		{name: "conversation with nil id", referenceType: amaicall.ReferenceTypeConversation, referenceID: uuid.Nil, expectErr: serviceerrors.ErrInvalidArgument},
		{name: "contact_case with nil id", referenceType: amaicall.ReferenceTypeContactCase, referenceID: uuid.Nil, expectErr: serviceerrors.ErrInvalidArgument},

		// call
		{name: "call owned", referenceType: amaicall.ReferenceTypeCall, referenceID: refID, lookup: true, ownerID: customerID, expectID: refID},
		{name: "call foreign", referenceType: amaicall.ReferenceTypeCall, referenceID: refID, lookup: true, ownerID: otherID, expectErr: serviceerrors.ErrNotFound},
		{name: "call deleted", referenceType: amaicall.ReferenceTypeCall, referenceID: refID, lookup: true, ownerID: customerID, deleted: true, expectErr: serviceerrors.ErrNotFound},
		{name: "call missing typed", referenceType: amaicall.ReferenceTypeCall, referenceID: refID, lookup: true, getErr: typedNotFound, expectErr: serviceerrors.ErrNotFound},
		{name: "call missing bare", referenceType: amaicall.ReferenceTypeCall, referenceID: refID, lookup: true, getErr: requesthandler.ErrNotFound, expectErr: serviceerrors.ErrNotFound},
		{name: "call backend failure", referenceType: amaicall.ReferenceTypeCall, referenceID: refID, lookup: true, getErr: requesthandler.ErrInternal, expectErr: requesthandler.ErrInternal},
		{name: "call typed non-not-found propagates", referenceType: amaicall.ReferenceTypeCall, referenceID: refID, lookup: true, getErr: typedDenied, expectErr: typedDenied},

		// conversation
		{name: "conversation owned", referenceType: amaicall.ReferenceTypeConversation, referenceID: refID, lookup: true, ownerID: customerID, expectID: refID},
		{name: "conversation foreign", referenceType: amaicall.ReferenceTypeConversation, referenceID: refID, lookup: true, ownerID: otherID, expectErr: serviceerrors.ErrNotFound},
		{name: "conversation deleted", referenceType: amaicall.ReferenceTypeConversation, referenceID: refID, lookup: true, ownerID: customerID, deleted: true, expectErr: serviceerrors.ErrNotFound},
		{name: "conversation missing typed", referenceType: amaicall.ReferenceTypeConversation, referenceID: refID, lookup: true, getErr: typedNotFound, expectErr: serviceerrors.ErrNotFound},
		{name: "conversation missing bare", referenceType: amaicall.ReferenceTypeConversation, referenceID: refID, lookup: true, getErr: requesthandler.ErrNotFound, expectErr: serviceerrors.ErrNotFound},
		{name: "conversation backend failure", referenceType: amaicall.ReferenceTypeConversation, referenceID: refID, lookup: true, getErr: requesthandler.ErrInternal, expectErr: requesthandler.ErrInternal},

		// contact_case
		{name: "case owned", referenceType: amaicall.ReferenceTypeContactCase, referenceID: refID, lookup: true, ownerID: customerID, expectID: refID},
		{name: "case foreign", referenceType: amaicall.ReferenceTypeContactCase, referenceID: refID, lookup: true, ownerID: otherID, expectErr: serviceerrors.ErrNotFound},
		{name: "case missing typed", referenceType: amaicall.ReferenceTypeContactCase, referenceID: refID, lookup: true, getErr: typedNotFound, expectErr: serviceerrors.ErrNotFound},
		{name: "case missing bare", referenceType: amaicall.ReferenceTypeContactCase, referenceID: refID, lookup: true, getErr: requesthandler.ErrNotFound, expectErr: serviceerrors.ErrNotFound},
		{name: "case backend failure", referenceType: amaicall.ReferenceTypeContactCase, referenceID: refID, lookup: true, getErr: requesthandler.ErrInternal, expectErr: requesthandler.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.lookup {
				tmDelete := refDeleteTime(tt.deleted)
				switch tt.referenceType {
				case amaicall.ReferenceTypeCall:
					if tt.getErr != nil {
						mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(nil, tt.getErr)
					} else {
						mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(&cmcall.Call{
							Identity: commonidentity.Identity{ID: tt.referenceID, CustomerID: tt.ownerID},
							TMDelete: tmDelete,
						}, nil)
					}
				case amaicall.ReferenceTypeConversation:
					if tt.getErr != nil {
						mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.referenceID).Return(nil, tt.getErr)
					} else {
						mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.referenceID).Return(&cvconversation.Conversation{
							Identity: commonidentity.Identity{ID: tt.referenceID, CustomerID: tt.ownerID},
							TMDelete: tmDelete,
						}, nil)
					}
				case amaicall.ReferenceTypeContactCase:
					if tt.getErr != nil {
						mockReq.EXPECT().ContactV1CaseGet(ctx, customerID, tt.referenceID).Return(nil, tt.getErr)
					} else {
						mockReq.EXPECT().ContactV1CaseGet(ctx, customerID, tt.referenceID).Return(&cmkase.Case{
							ID: tt.referenceID, CustomerID: tt.ownerID,
						}, nil)
					}
				}
			}

			res, err := h.aicallReferenceValidate(ctx, customerID, tt.referenceType, tt.referenceID)
			if tt.expectErr == nil {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if res != tt.expectID {
					t.Errorf("Wrong match. expect: %s, got: %s", tt.expectID, res)
				}
				return
			}

			if err == nil {
				t.Fatalf("Wrong match. expect: %v, got: ok", tt.expectErr)
			}
			if !stderrors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if res != uuid.Nil {
				t.Errorf("Wrong match. expect: nil id on error, got: %s", res)
			}
		})
	}
}

// Test_aicallReferenceValidate_notFoundIsIndistinguishable pins that a
// missing, a deleted and a foreign resource produce the very same error value,
// and that it carries no backend error in its chain.
func Test_aicallReferenceValidate_notFoundIsIndistinguishable(t *testing.T) {
	customerID := uuid.FromStringOrNil(refTestCustomer)
	otherID := uuid.FromStringOrNil(refTestOther)
	refID := uuid.FromStringOrNil(refTestRef)
	typedNotFound := cerrors.NotFound(commonoutline.ServiceNameConversationManager, "CONVERSATION_NOT_FOUND", "not found")

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	now := refDeleteTime(true)
	mockReq.EXPECT().ConversationV1ConversationGet(ctx, refID).Return(nil, typedNotFound)
	_, errMissing := h.aicallReferenceValidate(ctx, customerID, amaicall.ReferenceTypeConversation, refID)

	mockReq.EXPECT().ConversationV1ConversationGet(ctx, refID).Return(&cvconversation.Conversation{Identity: commonidentity.Identity{ID: refID, CustomerID: customerID}, TMDelete: now}, nil)
	_, errDeleted := h.aicallReferenceValidate(ctx, customerID, amaicall.ReferenceTypeConversation, refID)

	mockReq.EXPECT().ConversationV1ConversationGet(ctx, refID).Return(&cvconversation.Conversation{Identity: commonidentity.Identity{ID: refID, CustomerID: otherID}}, nil)
	_, errForeign := h.aicallReferenceValidate(ctx, customerID, amaicall.ReferenceTypeConversation, refID)

	for name, err := range map[string]error{"missing": errMissing, "deleted": errDeleted, "foreign": errForeign} {
		if err != serviceerrors.ErrNotFound {
			t.Errorf("Wrong match. %s: expect the bare ErrNotFound, got: %v", name, err)
		}
		var ve *cerrors.VoipbinError
		if stderrors.As(err, &ve) {
			t.Errorf("Wrong match. %s: the chain must not carry a backend VoipbinError", name)
		}
	}
}

func Test_AIcallCreate_reference(t *testing.T) {
	customerID := uuid.FromStringOrNil(refTestCustomer)
	otherID := uuid.FromStringOrNil(refTestOther)
	refID := uuid.FromStringOrNil(refTestRef)
	aiID := uuid.FromStringOrNil(refTestAI)

	adminAgent := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil(refTestAgent), CustomerID: customerID},
		Permission: amagent.PermissionCustomerAdmin,
	})
	superAgent := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil(refTestAgent), CustomerID: customerID},
		Permission: amagent.PermissionProjectSuperAdmin,
	})
	plainAgent := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil(refTestAgent), CustomerID: customerID},
		Permission: amagent.PermissionNone,
	})
	accesskey := auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{
		ID:         uuid.FromStringOrNil(refTestAgent),
		CustomerID: customerID,
	})

	tests := []struct {
		name string

		agent         *auth.AuthIdentity
		aiCustomer    uuid.UUID
		referenceType amaicall.ReferenceType
		referenceID   uuid.UUID

		callOwner  uuid.UUID // when set, CallV1CallGet is expected and returns this owner
		expectErr  error
		expectFlow bool // the activeflow and the start call happen only on success
	}{
		{name: "owned call", agent: adminAgent, aiCustomer: customerID, referenceType: amaicall.ReferenceTypeCall, referenceID: refID, callOwner: customerID, expectFlow: true},
		{name: "accesskey owned call", agent: accesskey, aiCustomer: customerID, referenceType: amaicall.ReferenceTypeCall, referenceID: refID, callOwner: customerID, expectFlow: true},
		{name: "foreign call", agent: adminAgent, aiCustomer: customerID, referenceType: amaicall.ReferenceTypeCall, referenceID: refID, callOwner: otherID, expectErr: serviceerrors.ErrNotFound},
		{name: "unknown type", agent: adminAgent, aiCustomer: customerID, referenceType: amaicall.ReferenceTypeTask, referenceID: refID, expectErr: serviceerrors.ErrInvalidArgument},
		{name: "none with a reference id", agent: adminAgent, aiCustomer: customerID, referenceType: amaicall.ReferenceTypeNone, referenceID: refID, expectFlow: true},
		// the reference is compared to the assistance owner, not to the caller
		{name: "super admin, assistance of another customer, call of the caller", agent: superAgent, aiCustomer: otherID, referenceType: amaicall.ReferenceTypeCall, referenceID: refID, callOwner: customerID, expectErr: serviceerrors.ErrNotFound},
		{name: "super admin, assistance of another customer, call of the assistance owner", agent: superAgent, aiCustomer: otherID, referenceType: amaicall.ReferenceTypeCall, referenceID: refID, callOwner: otherID, expectFlow: true},
		// permission is checked before any reference lookup
		{name: "no permission", agent: plainAgent, aiCustomer: customerID, referenceType: amaicall.ReferenceTypeCall, referenceID: refID, expectErr: serviceerrors.ErrPermissionDenied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB, utilHandler: mockUtil}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIGet(ctx, aiID).Return(&amai.AI{Identity: commonidentity.Identity{ID: aiID, CustomerID: tt.aiCustomer}}, nil)

			if tt.callOwner != uuid.Nil {
				mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(&cmcall.Call{Identity: commonidentity.Identity{ID: tt.referenceID, CustomerID: tt.callOwner}}, nil)
			}

			// Without an expectation gomock fails on an unexpected call, which
			// is what pins "a refusal creates nothing".
			if tt.expectFlow {
				af := &fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a0000000-0006-11f0-8000-000000000001")}}
				mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, tt.aiCustomer, uuid.Nil, fmactiveflow.ReferenceTypeAPI, uuid.Nil, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(af, nil)

				expectRefID := tt.referenceID
				if tt.referenceType == amaicall.ReferenceTypeNone {
					expectRefID = uuid.Nil
				}
				mockReq.EXPECT().AIV1AIcallStart(ctx, uuid.Nil, amaicall.AssistanceTypeAI, aiID, af.ID, tt.referenceType, expectRefID).Return(&amaicall.AIcall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a0000000-0007-11f0-8000-000000000001")}}, nil)
			}

			_, err := h.AIcallCreate(ctx, tt.agent, amaicall.AssistanceTypeAI, aiID, tt.referenceType, tt.referenceID)
			if tt.expectErr == nil {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if !stderrors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_ServiceAgentAIcallCreate_reference(t *testing.T) {
	customerID := uuid.FromStringOrNil(refTestCustomer)
	otherID := uuid.FromStringOrNil(refTestOther)
	refID := uuid.FromStringOrNil(refTestRef)
	aiID := uuid.FromStringOrNil(refTestAI)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil(refTestAgent), CustomerID: customerID},
		Permission: amagent.PermissionCustomerAgent,
	})

	tests := []struct {
		name string

		assistanceID  uuid.UUID
		referenceType amaicall.ReferenceType
		referenceID   uuid.UUID

		lookupType amaicall.ReferenceType // which backend lookup is expected, if any
		owner      uuid.UUID
		expectErr  error
		expectFlow bool
	}{
		{name: "owned call", assistanceID: aiID, referenceType: amaicall.ReferenceTypeCall, referenceID: refID, lookupType: amaicall.ReferenceTypeCall, owner: customerID, expectFlow: true},
		{name: "foreign call", assistanceID: aiID, referenceType: amaicall.ReferenceTypeCall, referenceID: refID, lookupType: amaicall.ReferenceTypeCall, owner: otherID, expectErr: serviceerrors.ErrNotFound},
		{name: "owned conversation", assistanceID: aiID, referenceType: amaicall.ReferenceTypeConversation, referenceID: refID, lookupType: amaicall.ReferenceTypeConversation, owner: customerID, expectFlow: true},
		{name: "foreign conversation", assistanceID: aiID, referenceType: amaicall.ReferenceTypeConversation, referenceID: refID, lookupType: amaicall.ReferenceTypeConversation, owner: otherID, expectErr: serviceerrors.ErrNotFound},
		{name: "foreign contact_case", assistanceID: aiID, referenceType: amaicall.ReferenceTypeContactCase, referenceID: refID, lookupType: amaicall.ReferenceTypeContactCase, owner: otherID, expectErr: serviceerrors.ErrNotFound},
		{name: "unknown type", assistanceID: aiID, referenceType: amaicall.ReferenceTypeTask, referenceID: refID, expectErr: serviceerrors.ErrInvalidArgument},
		{name: "none with a reference id", assistanceID: aiID, referenceType: amaicall.ReferenceTypeNone, referenceID: refID, expectFlow: true},
		// the 400 for a missing assistance id is kept for every type but contact_case
		{name: "none without assistance id", assistanceID: uuid.Nil, referenceType: amaicall.ReferenceTypeNone, referenceID: refID, expectErr: serviceerrors.ErrInvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB, utilHandler: mockUtil}
			ctx := context.Background()

			switch tt.lookupType {
			case amaicall.ReferenceTypeCall:
				mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(&cmcall.Call{Identity: commonidentity.Identity{ID: tt.referenceID, CustomerID: tt.owner}}, nil)
			case amaicall.ReferenceTypeConversation:
				mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.referenceID).Return(&cvconversation.Conversation{Identity: commonidentity.Identity{ID: tt.referenceID, CustomerID: tt.owner}}, nil)
			case amaicall.ReferenceTypeContactCase:
				mockReq.EXPECT().ContactV1CaseGet(ctx, customerID, tt.referenceID).Return(&cmkase.Case{ID: tt.referenceID, CustomerID: tt.owner}, nil)
			}

			if tt.expectFlow {
				mockReq.EXPECT().AIV1AIGet(ctx, aiID).Return(&amai.AI{Identity: commonidentity.Identity{ID: aiID, CustomerID: customerID}}, nil)
				af := &fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a0000000-0006-11f0-8000-000000000001")}}
				mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, customerID, uuid.Nil, fmactiveflow.ReferenceTypeAPI, uuid.Nil, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(af, nil)

				expectRefID := tt.referenceID
				if tt.referenceType == amaicall.ReferenceTypeNone {
					expectRefID = uuid.Nil
				}
				mockReq.EXPECT().AIV1AIcallStart(ctx, uuid.Nil, amaicall.AssistanceTypeAI, aiID, af.ID, tt.referenceType, expectRefID).Return(&amaicall.AIcall{
					Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil("a0000000-0007-11f0-8000-000000000001")},
					ActiveflowID: af.ID,
				}, nil)
			}

			_, err := h.ServiceAgentAIcallCreate(ctx, agent, amaicall.AssistanceTypeAI, tt.assistanceID, tt.referenceType, tt.referenceID)
			if tt.expectErr == nil {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if !stderrors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
