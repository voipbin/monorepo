package activeflowhandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmpushring "monorepo/bin-call-manager/models/pushring"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
)

func Test_actionHandleConnect_pushRing(t *testing.T) {
	var (
		customerID   = uuid.FromStringOrNil("5a000000-0000-4000-8000-000000000001")
		referenceID  = uuid.FromStringOrNil("5a000000-0000-4000-8000-000000000002")
		confbridgeID = uuid.FromStringOrNil("5a000000-0000-4000-8000-000000000003")
		flowID       = uuid.FromStringOrNil("5a000000-0000-4000-8000-000000000004")
		sessionID    = uuid.FromStringOrNil("5a000000-0000-4000-8000-000000000005")
		actionID     = uuid.FromStringOrNil("5a000000-0000-4000-8000-000000000006")
		joinID       = uuid.FromStringOrNil("5a000000-0000-4000-8000-000000000007")
		stackID      = uuid.FromStringOrNil("5a000000-0000-4000-8000-000000000008")
		callID       = uuid.FromStringOrNil("5a000000-0000-4000-8000-000000000009")
	)

	source := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"}
	extension := commonaddress.Address{Type: commonaddress.TypeExtension, Target: "5a000000-0000-4000-8000-0000000000e1", TargetName: "2000"}
	tel := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550199"}

	notRegistered := cerrors.FailedPrecondition(commonoutline.ServiceNameCallManager, cmgroupcall.ReasonExtensionNotRegistered, "not registered")
	otherTyped := cerrors.FailedPrecondition(commonoutline.ServiceNameCallManager, "SOMETHING_ELSE", "other")
	otherDomain := cerrors.FailedPrecondition(commonoutline.ServiceNameRegistrarManager, cmgroupcall.ReasonExtensionNotRegistered, "other domain")

	tests := []struct {
		name string

		destinations []commonaddress.Address
		relayReason  bool

		expectStart      bool
		responseStart    *cmpushring.StartResult
		responseStartErr error

		responseCalls    []*cmcall.Call
		responseGroups   []*cmgroupcall.Groupcall
		responseCallsErr error

		expectClose  bool
		expectErr    bool
		expectPushed bool
	}{
		{
			name:         "a tel destination never starts a session",
			destinations: []commonaddress.Address{tel},
			responseCalls: []*cmcall.Call{
				{Identity: commonidentity.Identity{ID: callID}},
			},
			expectPushed: true,
		},
		{
			name:         "several destinations never start a session",
			destinations: []commonaddress.Address{extension, tel},
			responseCalls: []*cmcall.Call{
				{Identity: commonidentity.Identity{ID: callID}},
			},
			expectPushed: true,
		},
		{
			name:           "a single extension without a session runs as before",
			destinations:   []commonaddress.Address{extension},
			expectStart:    true,
			responseStart:  &cmpushring.StartResult{},
			responseGroups: []*cmgroupcall.Groupcall{{Identity: commonidentity.Identity{ID: callID}}},
			expectPushed:   true,
		},
		{
			name:             "a start error runs as before",
			destinations:     []commonaddress.Address{extension},
			expectStart:      true,
			responseStartErr: errors.New("unknown route"),
			responseGroups:   []*cmgroupcall.Groupcall{{Identity: commonidentity.Identity{ID: callID}}},
			expectPushed:     true,
		},
		{
			name:             "a start error does not turn a not registered extension into a push only call",
			destinations:     []commonaddress.Address{extension},
			expectStart:      true,
			responseStartErr: errors.New("timeout"),
			responseCallsErr: notRegistered,
			expectErr:        true,
		},
		{
			name:           "a session with registered contacts runs the legs",
			destinations:   []commonaddress.Address{extension},
			expectStart:    true,
			responseStart:  &cmpushring.StartResult{Started: true, SessionID: sessionID, SIPRegistered: true},
			responseGroups: []*cmgroupcall.Groupcall{{Identity: commonidentity.Identity{ID: callID}}},
			expectPushed:   true,
		},
		{
			name:             "push only: no registered contact and the app can answer",
			destinations:     []commonaddress.Address{extension},
			expectStart:      true,
			responseStart:    &cmpushring.StartResult{Started: true, SessionID: sessionID},
			responseCallsErr: notRegistered,
			expectPushed:     true,
		},
		{
			name:             "push only with the relay reason and no call does not read a call",
			destinations:     []commonaddress.Address{extension},
			relayReason:      true,
			expectStart:      true,
			responseStart:    &cmpushring.StartResult{Started: true, SessionID: sessionID},
			responseCallsErr: notRegistered,
			expectPushed:     true,
		},
		{
			name:             "push only: the error of another reason closes the session",
			destinations:     []commonaddress.Address{extension},
			expectStart:      true,
			responseStart:    &cmpushring.StartResult{Started: true, SessionID: sessionID},
			responseCallsErr: otherTyped,
			expectClose:      true,
			expectErr:        true,
		},
		{
			name:             "push only: the same reason of another service closes the session",
			destinations:     []commonaddress.Address{extension},
			expectStart:      true,
			responseStart:    &cmpushring.StartResult{Started: true, SessionID: sessionID},
			responseCallsErr: otherDomain,
			expectClose:      true,
			expectErr:        true,
		},
		{
			name:             "push only: an untyped error closes the session",
			destinations:     []commonaddress.Address{extension},
			expectStart:      true,
			responseStart:    &cmpushring.StartResult{Started: true, SessionID: sessionID},
			responseCallsErr: errors.New("rpc down"),
			expectClose:      true,
			expectErr:        true,
		},
		{
			name:             "registered contacts and a not registered error closes the session",
			destinations:     []commonaddress.Address{extension},
			expectStart:      true,
			responseStart:    &cmpushring.StartResult{Started: true, SessionID: sessionID, SIPRegistered: true},
			responseCallsErr: notRegistered,
			expectClose:      true,
			expectErr:        true,
		},
		{
			name:             "an error without a session is returned as it is",
			destinations:     []commonaddress.Address{extension},
			expectStart:      true,
			responseStart:    &cmpushring.StartResult{},
			responseCallsErr: notRegistered,
			expectErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				reqHandler:      mockReq,
				actionHandler:   mockAction,
				stackmapHandler: mockStack,
			}

			af := &activeflow.Activeflow{
				Identity:       commonidentity.Identity{ID: uuid.FromStringOrNil("5a000000-0000-4000-8000-0000000000aa"), CustomerID: customerID},
				ReferenceID:    referenceID,
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   actionID,
					Type: action.TypeConnect,
					Option: action.ConvertOption(action.OptionConnect{
						Source:       source,
						Destinations: tt.destinations,
						RelayReason:  tt.relayReason,
					}),
				},
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, customerID, af.ID, cmconfbridge.ReferenceTypeCall, referenceID, cmconfbridge.TypeConnect).
				Return(&cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: confbridgeID}}, nil)
			mockReq.EXPECT().FlowV1FlowCreate(ctx, customerID, flow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).
				Return(&flow.Flow{Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID}}, nil)

			if tt.expectStart {
				// the session gets the temporary flow, the confbridge, the master call and the source of the connect.
				mockReq.EXPECT().CallV1PushRingStart(ctx, customerID, tt.destinations[0], confbridgeID, referenceID, flowID, source).
					Return(tt.responseStart, tt.responseStartErr)
			}

			mockReq.EXPECT().CallV1CallsCreate(ctx, customerID, flowID, referenceID, &source, tt.destinations, false, gomock.Any(), "", nil, nil).
				Return(tt.responseCalls, tt.responseGroups, tt.responseCallsErr)

			if tt.expectClose {
				mockReq.EXPECT().CallV1PushRingClose(ctx, sessionID).Return(nil)
			}

			if tt.expectPushed {
				mockUtil.EXPECT().UUIDCreate().Return(joinID).AnyTimes()
				mockStack.EXPECT().PushStackByActions(gomock.Any(), uuid.Nil, gomock.Any(), stack.IDMain, actionID).
					Return(&stack.Stack{ID: stackID, Actions: []action.Action{{ID: joinID, Type: action.TypeConfbridgeJoin}}}, nil)
				mockDB.EXPECT().ActiveflowUpdate(ctx, af.ID, gomock.Any()).Return(nil)
			}

			err := h.actionHandleConnect(ctx, af)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

// The close of a session is best effort: a failing close request does not hide the original error.
func Test_actionHandleConnect_pushRing_closeFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &activeflowHandler{reqHandler: mockReq}

	sessionID := uuid.FromStringOrNil("5a000000-0000-4000-8000-000000000005")
	mockReq.EXPECT().CallV1PushRingClose(gomock.Any(), sessionID).Return(errors.New("rpc down"))

	h.connectPushRingClose(context.Background(), &connectPushRing{started: true, sessionID: sessionID})
	h.connectPushRingClose(context.Background(), &connectPushRing{})
	h.connectPushRingClose(context.Background(), nil)
}

func Test_connectPushRing_pushOnlyAllowed(t *testing.T) {
	var nilRing *connectPushRing
	if nilRing.pushOnlyAllowed(errors.New("x")) {
		t.Errorf("a nil session must not allow the push only call")
	}
}
