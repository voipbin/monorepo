package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentCallParticipantAdd(t *testing.T) {
	var (
		customerID      = uuid.FromStringOrNil("a1b2c3d4-3b9f-11ef-8a51-f30f1e2ce101")
		otherCustomerID = uuid.FromStringOrNil("a1b2c3d4-3b9f-11ef-8a51-f30f1e2ce102")
		agentID         = uuid.FromStringOrNil("a1b2c3d4-3b9f-11ef-98ac-db226570f103")
		otherAgentID    = uuid.FromStringOrNil("a1b2c3d4-3b9f-11ef-98ac-db226570f104")
		callID          = uuid.FromStringOrNil("a1b2c3d4-3f9f-11ef-bfa4-532d59e85e05")
		confbridgeID    = uuid.FromStringOrNil("a1b2c3d4-3f9f-11ef-9c18-0b629aa668a6")
		extensionID     = uuid.FromStringOrNil("c5e7f18c-fc5a-4520-8326-e534e2ca0b8f")
		groupcallID     = uuid.FromStringOrNil("a1b2c3d4-8286-11ed-8875-378ced61c027")
		tmDelete        = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: agentID, CustomerID: customerID},
		Permission: amagent.PermissionCustomerAgent,
	})

	call := func(owner, customer uuid.UUID, status cmcall.Status, confbridge uuid.UUID) *cmcall.Call {
		return &cmcall.Call{
			Identity:     commonidentity.Identity{ID: callID, CustomerID: customer},
			OwnerID:      owner,
			Status:       status,
			ConfbridgeID: confbridge,
		}
	}
	okCall := call(agentID, customerID, cmcall.StatusProgressing, confbridgeID)

	confbridge := func(n int) *cmconfbridge.Confbridge {
		ids := map[string]uuid.UUID{}
		for i := 0; i < n; i++ {
			ids[string(rune('a'+i))] = uuid.Must(uuid.NewV4())
		}
		return &cmconfbridge.Confbridge{
			Identity:       commonidentity.Identity{ID: confbridgeID, CustomerID: customerID},
			ChannelCallIDs: ids,
		}
	}

	extension := func(customer uuid.UUID, deleted *time.Time) *rmextension.Extension {
		return &rmextension.Extension{
			Identity:  commonidentity.Identity{ID: extensionID, CustomerID: customer},
			Extension: "2002",
			TMDelete:  deleted,
		}
	}
	okAddr := commonaddress.Address{Type: commonaddress.TypeExtension, Target: extensionID.String()}
	assembled := commonaddress.Address{
		Type:       commonaddress.TypeExtension,
		Target:     extensionID.String(),
		TargetName: "2002",
	}

	errRPC := errors.New("rpc failed")

	tests := []struct {
		name string

		agent   *auth.AuthIdentity
		address commonaddress.Address

		callGet         bool
		responseCall    *cmcall.Call
		responseCallErr error

		confbridgeGet         bool
		responseConfbridge    *cmconfbridge.Confbridge
		responseConfbridgeErr error

		extGet         bool
		responseExt    *rmextension.Extension
		responseExtErr error

		addCalled      bool
		responseAddErr error

		expectErr    error          // matched with errors.Is
		expectStatus cerrors.Status // set when a VoipbinError is expected
		expectReason string
	}{
		{
			name:    "normal",
			agent:   agent,
			address: okAddr,

			callGet:            true,
			responseCall:       okCall,
			confbridgeGet:      true,
			responseConfbridge: confbridge(2),
			extGet:             true,
			responseExt:        extension(customerID, nil),
			addCalled:          true,
		},
		{
			name:  "client supplied target_name, name and detail are discarded",
			agent: agent,
			address: commonaddress.Address{
				Type: commonaddress.TypeExtension, Target: extensionID.String(),
				TargetName: "9999", Name: "evil", Detail: "evil detail",
			},

			callGet:            true,
			responseCall:       okCall,
			confbridgeGet:      true,
			responseConfbridge: confbridge(2),
			extGet:             true,
			responseExt:        extension(customerID, nil),
			addCalled:          true,
		},
		{
			name:      "accesskey identity is rejected",
			agent:     auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{}),
			address:   okAddr,
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:      "direct identity is rejected",
			agent:     auth.NewDirectIdentity(&auth.DirectScope{}),
			address:   okAddr,
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:            "call lookup error is returned",
			agent:           agent,
			address:         okAddr,
			callGet:         true,
			responseCallErr: errRPC,
			expectErr:       errRPC,
		},
		{
			name:         "call of another customer",
			agent:        agent,
			address:      okAddr,
			callGet:      true,
			responseCall: call(agentID, otherCustomerID, cmcall.StatusProgressing, confbridgeID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call owned by another agent",
			agent:        agent,
			address:      okAddr,
			callGet:      true,
			responseCall: call(otherAgentID, customerID, cmcall.StatusProgressing, confbridgeID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call without an owner",
			agent:        agent,
			address:      okAddr,
			callGet:      true,
			responseCall: call(uuid.Nil, customerID, cmcall.StatusProgressing, confbridgeID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "agent type address",
			agent:        agent,
			address:      commonaddress.Address{Type: commonaddress.TypeAgent, Target: extensionID.String()},
			callGet:      true,
			responseCall: okCall,
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "INVALID_PARTICIPANT_ADDRESS",
		},
		{
			name:         "tel type address",
			agent:        agent,
			address:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230002"},
			callGet:      true,
			responseCall: okCall,
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "INVALID_PARTICIPANT_ADDRESS",
		},
		{
			name:         "sip type address",
			agent:        agent,
			address:      commonaddress.Address{Type: commonaddress.TypeSIP, Target: "sip:a@203.0.113.1"},
			callGet:      true,
			responseCall: okCall,
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "INVALID_PARTICIPANT_ADDRESS",
		},
		{
			name:         "extension number instead of the extension id",
			agent:        agent,
			address:      commonaddress.Address{Type: commonaddress.TypeExtension, Target: "2002"},
			callGet:      true,
			responseCall: okCall,
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "INVALID_PARTICIPANT_ADDRESS",
		},
		{
			name:         "non canonical extension id is rejected",
			agent:        agent,
			address:      commonaddress.Address{Type: commonaddress.TypeExtension, Target: "C5E7F18C-FC5A-4520-8326-E534E2CA0B8F"},
			callGet:      true,
			responseCall: okCall,
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "INVALID_PARTICIPANT_ADDRESS",
		},
		{
			name:         "call is not progressing",
			agent:        agent,
			address:      okAddr,
			callGet:      true,
			responseCall: call(agentID, customerID, cmcall.StatusRinging, confbridgeID),
			expectStatus: cerrors.StatusFailedPrecondition,
			expectReason: "CALL_NOT_JOINABLE",
		},
		{
			name:         "call without a confbridge",
			agent:        agent,
			address:      okAddr,
			callGet:      true,
			responseCall: call(agentID, customerID, cmcall.StatusProgressing, uuid.Nil),
			expectStatus: cerrors.StatusFailedPrecondition,
			expectReason: "CALL_NOT_JOINABLE",
		},
		{
			name:                  "confbridge lookup error is returned",
			agent:                 agent,
			address:               okAddr,
			callGet:               true,
			responseCall:          okCall,
			confbridgeGet:         true,
			responseConfbridgeErr: errRPC,
			expectErr:             errRPC,
		},
		{
			name:               "call already has three parties",
			agent:              agent,
			address:            okAddr,
			callGet:            true,
			responseCall:       okCall,
			confbridgeGet:      true,
			responseConfbridge: confbridge(3),
			expectStatus:       cerrors.StatusFailedPrecondition,
			expectReason:       "PARTICIPANT_COUNT_INVALID",
		},
		{
			name:               "call with one party only",
			agent:              agent,
			address:            okAddr,
			callGet:            true,
			responseCall:       okCall,
			confbridgeGet:      true,
			responseConfbridge: confbridge(1),
			expectStatus:       cerrors.StatusFailedPrecondition,
			expectReason:       "PARTICIPANT_COUNT_INVALID",
		},
		{
			name:               "extension of another customer",
			agent:              agent,
			address:            okAddr,
			callGet:            true,
			responseCall:       okCall,
			confbridgeGet:      true,
			responseConfbridge: confbridge(2),
			extGet:             true,
			responseExt:        extension(otherCustomerID, nil),
			expectStatus:       cerrors.StatusInvalidArgument,
			expectReason:       "INVALID_PARTICIPANT_ADDRESS",
		},
		{
			name:               "deleted extension",
			agent:              agent,
			address:            okAddr,
			callGet:            true,
			responseCall:       okCall,
			confbridgeGet:      true,
			responseConfbridge: confbridge(2),
			extGet:             true,
			responseExt:        extension(customerID, &tmDelete),
			expectStatus:       cerrors.StatusInvalidArgument,
			expectReason:       "INVALID_PARTICIPANT_ADDRESS",
		},
		{
			name:               "extension not found",
			agent:              agent,
			address:            okAddr,
			callGet:            true,
			responseCall:       okCall,
			confbridgeGet:      true,
			responseConfbridge: confbridge(2),
			extGet:             true,
			responseExtErr:     requesthandler.ErrNotFound,
			expectStatus:       cerrors.StatusInvalidArgument,
			expectReason:       "INVALID_PARTICIPANT_ADDRESS",
		},
		{
			name:               "extension lookup backend failure is not turned into a bad request",
			agent:              agent,
			address:            okAddr,
			callGet:            true,
			responseCall:       okCall,
			confbridgeGet:      true,
			responseConfbridge: confbridge(2),
			extGet:             true,
			responseExtErr:     errRPC,
			expectErr:          errRPC,
		},
		{
			name:               "call-manager failure is returned",
			agent:              agent,
			address:            okAddr,
			callGet:            true,
			responseCall:       okCall,
			confbridgeGet:      true,
			responseConfbridge: confbridge(2),
			extGet:             true,
			responseExt:        extension(customerID, nil),
			addCalled:          true,
			responseAddErr:     errRPC,
			expectErr:          errRPC,
		},
		{
			name:               "call-manager conflict keeps its reason",
			agent:              agent,
			address:            okAddr,
			callGet:            true,
			responseCall:       okCall,
			confbridgeGet:      true,
			responseConfbridge: confbridge(2),
			extGet:             true,
			responseExt:        extension(customerID, nil),
			addCalled:          true,
			responseAddErr:     cerrors.FailedPrecondition("call-manager", "PARTICIPANT_ADD_IN_PROGRESS", "busy"),
			expectStatus:       cerrors.StatusFailedPrecondition,
			expectReason:       "PARTICIPANT_ADD_IN_PROGRESS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			// the order is part of the contract: the call and its owner, the call state, the participant count,
			// then the extension, and call-manager last. A rejected request never reaches the later steps.
			var calls []any
			if tt.callGet {
				calls = append(calls, mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(tt.responseCall, tt.responseCallErr))
			}
			if tt.confbridgeGet {
				calls = append(calls, mockReq.EXPECT().CallV1ConfbridgeGet(ctx, confbridgeID).Return(tt.responseConfbridge, tt.responseConfbridgeErr))
			}
			if tt.extGet {
				calls = append(calls, mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(tt.responseExt, tt.responseExtErr))
			}
			var responseGroupcall *cmgroupcall.Groupcall
			if tt.addCalled {
				if tt.responseAddErr == nil {
					responseGroupcall = &cmgroupcall.Groupcall{
						Identity: commonidentity.Identity{ID: groupcallID, CustomerID: customerID},
						CallIDs:  []uuid.UUID{uuid.FromStringOrNil("a1b2c3d4-8286-11ed-8875-378ced61c028")},
					}
				}
				calls = append(calls, mockReq.EXPECT().CallV1CallAddParticipant(ctx, callID, assembled).Return(responseGroupcall, tt.responseAddErr))
			}
			gomock.InOrder(calls...)

			res, err := h.ServiceAgentCallParticipantAdd(ctx, tt.agent, callID, tt.address)

			if tt.expectErr == nil && tt.expectStatus == "" {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if !reflect.DeepEqual(responseGroupcall.ConvertWebhookMessage(), res) {
					t.Errorf("Wrong match.\nexpect: %v\ngot: %v", responseGroupcall.ConvertWebhookMessage(), res)
				}
				return
			}

			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result on error, got: %v", res)
			}
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				var ve *cerrors.VoipbinError
				if errors.As(err, &ve) && ve.Status == cerrors.StatusInvalidArgument {
					t.Errorf("Wrong error. must not be a bad request: %v", err)
				}
				return
			}

			var ve *cerrors.VoipbinError
			if !errors.As(err, &ve) {
				t.Fatalf("Wrong error. expect: VoipbinError, got: %v", err)
			}
			if ve.Status != tt.expectStatus || ve.Reason != tt.expectReason {
				t.Errorf("Wrong match. expect: %v/%v, got: %v/%v", tt.expectStatus, tt.expectReason, ve.Status, ve.Reason)
			}
		})
	}
}

// Test_ServiceAgentCallParticipantAdd_extensionRejectionIsIndistinguishable pins that an extension that does not
// exist, belongs to another customer or is deleted gets the very same response, so a caller can not probe other
// customers.
func Test_ServiceAgentCallParticipantAdd_extensionRejectionIsIndistinguishable(t *testing.T) {
	var (
		customerID      = uuid.FromStringOrNil("a1b2c3d4-3b9f-11ef-8a51-f30f1e2ce101")
		otherCustomerID = uuid.FromStringOrNil("a1b2c3d4-3b9f-11ef-8a51-f30f1e2ce102")
		agentID         = uuid.FromStringOrNil("a1b2c3d4-3b9f-11ef-98ac-db226570f103")
		callID          = uuid.FromStringOrNil("a1b2c3d4-3f9f-11ef-bfa4-532d59e85e05")
		confbridgeID    = uuid.FromStringOrNil("a1b2c3d4-3f9f-11ef-9c18-0b629aa668a6")
		extensionID     = uuid.FromStringOrNil("c5e7f18c-fc5a-4520-8326-e534e2ca0b8f")
		tmDelete        = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: agentID, CustomerID: customerID},
		Permission: amagent.PermissionCustomerAgent,
	})
	okCall := &cmcall.Call{
		Identity: commonidentity.Identity{ID: callID, CustomerID: customerID},
		OwnerID:  agentID, Status: cmcall.StatusProgressing, ConfbridgeID: confbridgeID,
	}
	two := &cmconfbridge.Confbridge{ChannelCallIDs: map[string]uuid.UUID{"a": uuid.Must(uuid.NewV4()), "b": uuid.Must(uuid.NewV4())}}

	cases := []struct {
		name string
		ext  *rmextension.Extension
		err  error
	}{
		{"not found", nil, requesthandler.ErrNotFound},
		{"other customer", &rmextension.Extension{Identity: commonidentity.Identity{ID: extensionID, CustomerID: otherCustomerID}, Extension: "2002"}, nil},
		{"deleted", &rmextension.Extension{Identity: commonidentity.Identity{ID: extensionID, CustomerID: customerID}, Extension: "2002", TMDelete: &tmDelete}, nil},
	}

	var first *cerrors.VoipbinError
	for _, c := range cases {
		mc := gomock.NewController(t)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(okCall, nil)
		mockReq.EXPECT().CallV1ConfbridgeGet(ctx, confbridgeID).Return(two, nil)
		mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(c.ext, c.err)

		_, err := h.ServiceAgentCallParticipantAdd(ctx, agent, callID,
			commonaddress.Address{Type: commonaddress.TypeExtension, Target: extensionID.String()})

		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) {
			t.Fatalf("%s: expect VoipbinError, got: %v", c.name, err)
		}
		if first == nil {
			first = ve
			mc.Finish()
			continue
		}
		if ve.Status != first.Status || ve.Reason != first.Reason || ve.Message != first.Message {
			t.Errorf("%s: response differs.\nexpect: %v %v %v\ngot: %v %v %v", c.name,
				first.Status, first.Reason, first.Message, ve.Status, ve.Reason, ve.Message)
		}
		mc.Finish()
	}
}

// Test_ServiceAgentTransferStart_extensionBuilder_unchanged pins that the transfer keeps its own reason after the
// extension lookup was shared with the participant add.
func Test_ServiceAgentTransferStart_extensionReasonUnchanged(t *testing.T) {
	var (
		customerID  = uuid.FromStringOrNil("a1b2c3d4-3b9f-11ef-8a51-f30f1e2ce101")
		agentID     = uuid.FromStringOrNil("a1b2c3d4-3b9f-11ef-98ac-db226570f103")
		extensionID = uuid.FromStringOrNil("c5e7f18c-fc5a-4520-8326-e534e2ca0b8f")
	)
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: agentID, CustomerID: customerID},
		Permission: amagent.PermissionCustomerAgent,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(nil, requesthandler.ErrNotFound)

	_, err := h.serviceAgentTransferBuildAddresses(ctx, agent, extensionID)
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Reason != "INVALID_TRANSFEREE_ADDRESS" || ve.Message != "The transferee extension was not found." {
		t.Errorf("Wrong match. expect: INVALID_TRANSFEREE_ADDRESS, got: %v", err)
	}
}
