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
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	rmextension "monorepo/bin-registrar-manager/models/extension"
	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentTransferStart(t *testing.T) {
	var (
		customerID      = uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9")
		otherCustomerID = uuid.FromStringOrNil("6e27823d-3b9f-11ef-8a51-f30f1e2ce1e9")
		agentID         = uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a")
		otherAgentID    = uuid.FromStringOrNil("7f38934e-3b9f-11ef-98ac-db226570f09a")
		callID          = uuid.FromStringOrNil("896d70be-3f9f-11ef-bfa4-532d59e85e8c")
		confbridgeID    = uuid.FromStringOrNil("8996f42a-3f9f-11ef-9c18-0b629aa668a1")
		extensionID     = uuid.FromStringOrNil("c5e7f18c-fc5a-4520-8326-e534e2ca0b8f")
		transferID      = uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c021")
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

	extension := func(customer uuid.UUID, deleted *time.Time) *rmextension.Extension {
		return &rmextension.Extension{
			Identity:  commonidentity.Identity{ID: extensionID, CustomerID: customer},
			Extension: "2001",
			TMDelete:  deleted,
		}
	}
	addr := func(typ commonaddress.Type, target string) []commonaddress.Address {
		return []commonaddress.Address{{Type: typ, Target: target}}
	}
	okAddr := addr(commonaddress.TypeExtension, extensionID.String())
	assembled := []commonaddress.Address{{
		Type:       commonaddress.TypeExtension,
		Target:     extensionID.String(),
		TargetName: "2001",
	}}

	errRPC := errors.New("rpc failed")

	tests := []struct {
		name string

		agent        *auth.AuthIdentity
		transferType tmtransfer.Type
		addresses    []commonaddress.Address

		callGet         bool // whether CallV1CallGet is expected
		responseCall    *cmcall.Call
		responseCallErr error

		extGet         bool // whether RegistrarV1ExtensionGet is expected
		responseExt    *rmextension.Extension
		responseExtErr error

		startCalled      bool // whether TransferV1TransferStart is expected
		expectStartType  tmtransfer.Type
		responseStartErr error

		expectErr    error          // matched with errors.Is
		expectStatus cerrors.Status // set when a VoipbinError is expected
	}{
		{
			name:            "blind, normal",
			agent:           agent,
			transferType:    tmtransfer.TypeBlind,
			addresses:       okAddr,
			callGet:         true,
			responseCall:    okCall,
			extGet:          true,
			responseExt:     extension(customerID, nil),
			startCalled:     true,
			expectStartType: tmtransfer.TypeBlind,
		},
		{
			name:            "attended, normal",
			agent:           agent,
			transferType:    tmtransfer.TypeAttended,
			addresses:       okAddr,
			callGet:         true,
			responseCall:    okCall,
			extGet:          true,
			responseExt:     extension(customerID, nil),
			startCalled:     true,
			expectStartType: tmtransfer.TypeAttended,
		},
		{
			name:         "client supplied target_name, name and detail are discarded",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses: []commonaddress.Address{{
				Type: commonaddress.TypeExtension, Target: extensionID.String(),
				TargetName: "9999", Name: "evil", Detail: "evil detail",
			}},
			callGet:         true,
			responseCall:    okCall,
			extGet:          true,
			responseExt:     extension(customerID, nil),
			startCalled:     true,
			expectStartType: tmtransfer.TypeBlind,
		},
		{
			name:            "transfer to the own extension is not blocked",
			agent:           agent,
			transferType:    tmtransfer.TypeBlind,
			addresses:       okAddr,
			callGet:         true,
			responseCall:    okCall,
			extGet:          true,
			responseExt:     extension(customerID, nil),
			startCalled:     true,
			expectStartType: tmtransfer.TypeBlind,
		},
		{
			name:         "accesskey identity is rejected",
			agent:        auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{}),
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			expectErr:    serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:         "direct identity is rejected",
			agent:        auth.NewDirectIdentity(&auth.DirectScope{}),
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			expectErr:    serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:         "empty transfer type",
			agent:        agent,
			transferType: tmtransfer.Type(""),
			addresses:    okAddr,
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "unknown transfer type",
			agent:        agent,
			transferType: tmtransfer.Type("warm"),
			addresses:    okAddr,
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "no transferee address",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    []commonaddress.Address{},
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "two transferee addresses",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    append(append([]commonaddress.Address{}, okAddr...), okAddr...),
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "agent type address",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    addr(commonaddress.TypeAgent, otherAgentID.String()),
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "tel type address",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    addr(commonaddress.TypeTel, "+15550000001"),
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "sip type address",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    addr(commonaddress.TypeSIP, "user@example.com"),
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "extension number instead of the extension id",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    addr(commonaddress.TypeExtension, "2001"),
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "non canonical extension id is rejected",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    addr(commonaddress.TypeExtension, "C5E7F18C-FC5A-4520-8326-E534E2CA0B8F"),
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:            "call lookup error is returned",
			agent:           agent,
			transferType:    tmtransfer.TypeBlind,
			addresses:       okAddr,
			callGet:         true,
			responseCallErr: requesthandler.ErrNotFound,
			expectErr:       requesthandler.ErrNotFound,
		},
		{
			name:         "deleted call",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			callGet:      true,
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{ID: callID, CustomerID: customerID},
				OwnerID:  agentID, Status: cmcall.StatusProgressing, ConfbridgeID: confbridgeID,
				TMDelete: &tmDelete,
			},
			expectErr: serviceerrors.ErrStateInvalid,
		},
		{
			name:         "call of another customer",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			callGet:      true,
			responseCall: call(agentID, otherCustomerID, cmcall.StatusProgressing, confbridgeID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call owned by another agent",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			callGet:      true,
			responseCall: call(otherAgentID, customerID, cmcall.StatusProgressing, confbridgeID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call without an owner",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			callGet:      true,
			responseCall: call(uuid.Nil, customerID, cmcall.StatusProgressing, confbridgeID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name: "agent without an id never matches an ownerless call",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity:   commonidentity.Identity{ID: uuid.Nil, CustomerID: customerID},
				Permission: amagent.PermissionCustomerAgent,
			}),
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			callGet:      true,
			responseCall: call(uuid.Nil, customerID, cmcall.StatusProgressing, confbridgeID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call is not progressing",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			callGet:      true,
			responseCall: call(agentID, customerID, cmcall.StatusRinging, confbridgeID),
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "call without a confbridge",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			callGet:      true,
			responseCall: call(agentID, customerID, cmcall.StatusProgressing, uuid.Nil),
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "extension of another customer",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			callGet:      true,
			responseCall: okCall,
			extGet:       true,
			responseExt:  extension(otherCustomerID, nil),
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:         "deleted extension",
			agent:        agent,
			transferType: tmtransfer.TypeBlind,
			addresses:    okAddr,
			callGet:      true,
			responseCall: okCall,
			extGet:       true,
			responseExt:  extension(customerID, &tmDelete),
			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name:           "extension not found",
			agent:          agent,
			transferType:   tmtransfer.TypeBlind,
			addresses:      okAddr,
			callGet:        true,
			responseCall:   okCall,
			extGet:         true,
			responseExtErr: requesthandler.ErrNotFound,
			expectStatus:   cerrors.StatusInvalidArgument,
		},
		{
			name:           "extension lookup backend failure is not turned into a bad request",
			agent:          agent,
			transferType:   tmtransfer.TypeBlind,
			addresses:      okAddr,
			callGet:        true,
			responseCall:   okCall,
			extGet:         true,
			responseExtErr: errRPC,
			expectErr:      errRPC,
		},
		{
			name:             "transfer-manager failure is returned",
			agent:            agent,
			transferType:     tmtransfer.TypeBlind,
			addresses:        okAddr,
			callGet:          true,
			responseCall:     okCall,
			extGet:           true,
			responseExt:      extension(customerID, nil),
			startCalled:      true,
			expectStartType:  tmtransfer.TypeBlind,
			responseStartErr: errRPC,
			expectErr:        errRPC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.callGet {
				mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(tt.responseCall, tt.responseCallErr)
			}
			if tt.extGet {
				mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(tt.responseExt, tt.responseExtErr)
			}
			var responseTransfer *tmtransfer.Transfer
			if tt.startCalled {
				if tt.responseStartErr == nil {
					responseTransfer = &tmtransfer.Transfer{
						Identity:            commonidentity.Identity{ID: transferID, CustomerID: customerID},
						Type:                tt.expectStartType,
						TransfererCallID:    callID,
						TransfereeAddresses: assembled,
					}
				}
				mockReq.EXPECT().TransferV1TransferStart(ctx, tt.expectStartType, callID, assembled).Return(responseTransfer, tt.responseStartErr)
			}

			res, err := h.ServiceAgentTransferStart(ctx, tt.agent, tt.transferType, callID, tt.addresses)

			if tt.expectErr == nil && tt.expectStatus == "" {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if !reflect.DeepEqual(responseTransfer.ConvertWebhookMessage(), res) {
					t.Errorf("Wrong match.\nexpect: %v\ngot: %v", responseTransfer.ConvertWebhookMessage(), res)
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
			if ve.Status != tt.expectStatus {
				t.Errorf("Wrong status. expect: %v, got: %v", tt.expectStatus, ve.Status)
			}
		})
	}
}

// Test_ServiceAgentTransferStart_extensionRejectionIsIndistinguishable pins that an extension that does not exist,
// belongs to another customer or is deleted gets the very same response, so a caller can not probe other customers.
func Test_ServiceAgentTransferStart_extensionRejectionIsIndistinguishable(t *testing.T) {
	var (
		customerID      = uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9")
		otherCustomerID = uuid.FromStringOrNil("6e27823d-3b9f-11ef-8a51-f30f1e2ce1e9")
		agentID         = uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a")
		callID          = uuid.FromStringOrNil("896d70be-3f9f-11ef-bfa4-532d59e85e8c")
		extensionID     = uuid.FromStringOrNil("c5e7f18c-fc5a-4520-8326-e534e2ca0b8f")
		tmDelete        = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: agentID, CustomerID: customerID},
		Permission: amagent.PermissionCustomerAgent,
	})
	okCall := &cmcall.Call{
		Identity: commonidentity.Identity{ID: callID, CustomerID: customerID},
		OwnerID:  agentID, Status: cmcall.StatusProgressing,
		ConfbridgeID: uuid.FromStringOrNil("8996f42a-3f9f-11ef-9c18-0b629aa668a1"),
	}

	cases := []struct {
		name string
		ext  *rmextension.Extension
		err  error
	}{
		{"not found", nil, requesthandler.ErrNotFound},
		{"other customer", &rmextension.Extension{Identity: commonidentity.Identity{ID: extensionID, CustomerID: otherCustomerID}, Extension: "2001"}, nil},
		{"deleted", &rmextension.Extension{Identity: commonidentity.Identity{ID: extensionID, CustomerID: customerID}, Extension: "2001", TMDelete: &tmDelete}, nil},
	}

	var first *cerrors.VoipbinError
	for _, c := range cases {
		mc := gomock.NewController(t)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(okCall, nil)
		mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(c.ext, c.err)

		_, err := h.ServiceAgentTransferStart(ctx, agent, tmtransfer.TypeBlind, callID,
			[]commonaddress.Address{{Type: commonaddress.TypeExtension, Target: extensionID.String()}})

		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) {
			t.Fatalf("%s: expect VoipbinError, got: %v", c.name, err)
		}
		if first == nil {
			first = ve
			continue
		}
		if ve.Status != first.Status || ve.Reason != first.Reason || ve.Message != first.Message {
			t.Errorf("%s: response differs.\nexpect: %v %v %v\ngot: %v %v %v", c.name,
				first.Status, first.Reason, first.Message, ve.Status, ve.Reason, ve.Message)
		}
		mc.Finish()
	}
}

// Test_ServiceAgentTransferStart_addressIsBuiltFromLookup pins that the address passed on carries the values of the
// looked-up extension, not the values the client sent.
func Test_ServiceAgentTransferStart_addressIsBuiltFromLookup(t *testing.T) {
	var (
		customerID  = uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9")
		agentID     = uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a")
		callID      = uuid.FromStringOrNil("896d70be-3f9f-11ef-bfa4-532d59e85e8c")
		requestedID = uuid.FromStringOrNil("c5e7f18c-fc5a-4520-8326-e534e2ca0b8f")
		storedID    = uuid.FromStringOrNil("d6f8a29d-0d6b-4631-9437-f645f3db1c90")
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

	mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(&cmcall.Call{
		Identity: commonidentity.Identity{ID: callID, CustomerID: customerID},
		OwnerID:  agentID, Status: cmcall.StatusProgressing,
		ConfbridgeID: uuid.FromStringOrNil("8996f42a-3f9f-11ef-9c18-0b629aa668a1"),
	}, nil)
	mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, requestedID).Return(&rmextension.Extension{
		Identity:  commonidentity.Identity{ID: storedID, CustomerID: customerID},
		Extension: "2001",
	}, nil)
	mockReq.EXPECT().TransferV1TransferStart(ctx, tmtransfer.TypeBlind, callID, []commonaddress.Address{
		{Type: commonaddress.TypeExtension, Target: storedID.String(), TargetName: "2001"},
	}).Return(&tmtransfer.Transfer{}, nil)

	if _, err := h.ServiceAgentTransferStart(ctx, agent, tmtransfer.TypeBlind, callID, []commonaddress.Address{
		{Type: commonaddress.TypeExtension, Target: requestedID.String(), TargetName: "9999"},
	}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
