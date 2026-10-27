package activeflowhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

// depMocks holds the strict mocks of the handler. A call that a test does not
// register fails the test.
type depMocks struct {
	req      *requesthandler.MockRequestHandler
	stack    *stackmaphandler.MockStackmapHandler
	db       *dbhandler.MockDBHandler
	variable *variablehandler.MockVariableHandler
	action   *actionhandler.MockActionHandler
	util     *utilhandler.MockUtilHandler
}

func newDepTestHandler(mc *gomock.Controller) (*activeflowHandler, *depMocks) {
	m := &depMocks{
		req:      requesthandler.NewMockRequestHandler(mc),
		stack:    stackmaphandler.NewMockStackmapHandler(mc),
		db:       dbhandler.NewMockDBHandler(mc),
		variable: variablehandler.NewMockVariableHandler(mc),
		action:   actionhandler.NewMockActionHandler(mc),
		util:     utilhandler.NewMockUtilHandler(mc),
	}

	h := &activeflowHandler{
		utilHandler:     m.util,
		db:              m.db,
		reqHandler:      m.req,
		notifyHandler:   notifyhandler.NewMockNotifyHandler(mc),
		actionHandler:   m.action,
		variableHandler: m.variable,
		stackmapHandler: m.stack,
	}

	return h, m
}

var (
	depTargetStackID  = uuid.FromStringOrNil("d3b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c01")
	depTargetActionID = uuid.FromStringOrNil("d3b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c02")
	depFalseTargetID  = uuid.FromStringOrNil("d3b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c03")
	depServiceID      = uuid.FromStringOrNil("d3b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c04")
	depConfbridgeID   = uuid.FromStringOrNil("d3b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c05")
	depFlowID         = uuid.FromStringOrNil("d3b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c06")
	depQueueID        = uuid.FromStringOrNil("d3b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c07")
	depConferenceID   = uuid.FromStringOrNil("d3b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c08")

	errDepInjected = errors.New("synthetic dependency failure")
)

// depPushActions is returned by the mocked services as the actions to push.
var depPushActions = []action.Action{{ID: depTargetActionID, Type: action.TypeAnswer}}

func Test_actionHandle_dependencyError(t *testing.T) {
	ctx := context.Background()

	// expectGetActionFail registers a failing lookup of the false target.
	expectGetActionFail := func(m *depMocks, af *activeflow.Activeflow, targetID uuid.UUID) {
		m.stack.EXPECT().GetAction(af.StackMap, af.CurrentStackID, targetID, false).Return(stack.IDEmpty, nil, errDepInjected)
	}
	// expectForward registers a successful lookup of the false target, and
	// returns the stack id and the action id the activeflow must be forwarded to.
	expectForward := func(m *depMocks, af *activeflow.Activeflow, targetID uuid.UUID) {
		m.stack.EXPECT().GetAction(af.StackMap, af.CurrentStackID, targetID, false).Return(depTargetStackID, &action.Action{ID: depTargetActionID}, nil)
	}
	expectUpdateFail := func(m *depMocks, af *activeflow.Activeflow) {
		m.db.EXPECT().ActiveflowUpdate(ctx, af.ID, gomock.Any()).Return(errDepInjected)
	}
	expectPushFail := func(m *depMocks, af *activeflow.Activeflow, stackID uuid.UUID, actions []action.Action) {
		m.stack.EXPECT().PushStackByActions(af.StackMap, stackID, actions, af.CurrentStackID, af.CurrentAction.ID).Return(nil, errDepInjected)
	}

	tests := []struct {
		name string

		refType    activeflow.ReferenceType
		actionType action.Type
		option     map[string]any
		call       func(h *activeflowHandler, af *activeflow.Activeflow) error
		setup      func(m *depMocks, af *activeflow.Activeflow)

		// expectCause is matched with errors.Is. expectMessage is used for errors that
		// do not carry the cause.
		expectCause   error
		expectMessage string
		// expectForwardStack and expectForwardAction are the position the activeflow
		// must point to when the failure is reported. They stay empty when the
		// activeflow must not move.
		expectForwardStack  uuid.UUID
		expectForwardAction uuid.UUID
	}{
		{
			name:       "goto: original action lookup fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeGoto,
			option:     map[string]any{"target_id": depTargetActionID.String(), "loop_count": 2},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleGoto(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				expectGetActionFail(m, af, af.CurrentAction.ID)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not get the original action.",
		},
		{
			name:       "goto: target lookup fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeGoto,
			option:     map[string]any{"target_id": depTargetActionID.String(), "loop_count": 2},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleGoto(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.stack.EXPECT().GetAction(af.StackMap, af.CurrentStackID, af.CurrentAction.ID, false).Return(af.CurrentStackID, &action.Action{ID: af.CurrentAction.ID}, nil)
				expectGetActionFail(m, af, depTargetActionID)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not find the goto target action.",
		},
		{
			name:       "goto: progress update fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeGoto,
			option:     map[string]any{"target_id": depTargetActionID.String(), "loop_count": 2},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleGoto(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.stack.EXPECT().GetAction(af.StackMap, af.CurrentStackID, af.CurrentAction.ID, false).Return(af.CurrentStackID, &action.Action{ID: af.CurrentAction.ID}, nil)
				expectForward(m, af, depTargetActionID)
				expectUpdateFail(m, af)
			},
			expectCause:        errDepInjected,
			expectMessage:      "could not update the active flow after appending the patched actions.",
			expectForwardStack: depTargetStackID, expectForwardAction: depTargetActionID,
		},
		{
			name:       "fetch: pushing the fetched actions fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeFetch,
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleFetch(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.action.EXPECT().ActionFetchGet(ctx, &af.CurrentAction, af.ID, af.ReferenceID).Return(depPushActions, nil)
				expectPushFail(m, af, uuid.Nil, depPushActions)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not push the actions to the stack.",
		},
		{
			name:       "fetch_flow: pushing the actions of the flow fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeFetchFlow,
			option:     map[string]any{"flow_id": depFlowID.String()},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleFetchFlow(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().FlowV1FlowGet(ctx, depFlowID).Return(&flow.Flow{
					Identity: commonidentity.Identity{ID: depFlowID, CustomerID: af.CustomerID},
					Actions:  depPushActions,
				}, nil)
				expectPushFail(m, af, uuid.Nil, depPushActions)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not push the actions to the stack",
		},
		{
			name:       "condition_call_digits: reading the digits fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionCallDigits,
			option:     map[string]any{"length": 3, "false_target_id": depFalseTargetID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionCallDigits(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGetDigits(ctx, af.ReferenceID).Return("", errDepInjected)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not get digits.",
		},
		{
			name:       "condition_call_digits: false target lookup fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionCallDigits,
			option:     map[string]any{"length": 3, "false_target_id": depFalseTargetID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionCallDigits(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGetDigits(ctx, af.ReferenceID).Return("12", nil)
				expectGetActionFail(m, af, depFalseTargetID)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not find false target action.",
		},
		{
			name:       "condition_call_digits: progress update fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionCallDigits,
			option:     map[string]any{"length": 3, "false_target_id": depFalseTargetID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionCallDigits(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGetDigits(ctx, af.ReferenceID).Return("12", nil)
				expectForward(m, af, depFalseTargetID)
				expectUpdateFail(m, af)
			},
			expectCause:        errDepInjected,
			expectMessage:      "could not update the active flow after appending the patched actions.",
			expectForwardStack: depTargetStackID, expectForwardAction: depTargetActionID,
		},
		{
			name:       "condition_call_status: reading the call fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionCallStatus,
			option:     map[string]any{"status": string(cmcall.StatusProgressing), "false_target_id": depFalseTargetID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionCallStatus(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(nil, errDepInjected)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not get call.",
		},
		{
			name:       "condition_call_status: false target lookup fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionCallStatus,
			option:     map[string]any{"status": string(cmcall.StatusProgressing), "false_target_id": depFalseTargetID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionCallStatus(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(&cmcall.Call{Status: cmcall.StatusRinging}, nil)
				expectGetActionFail(m, af, depFalseTargetID)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not find false target action.",
		},
		{
			name:       "condition_call_status: progress update fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionCallStatus,
			option:     map[string]any{"status": string(cmcall.StatusProgressing), "false_target_id": depFalseTargetID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionCallStatus(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(&cmcall.Call{Status: cmcall.StatusRinging}, nil)
				expectForward(m, af, depFalseTargetID)
				expectUpdateFail(m, af)
			},
			expectCause:        errDepInjected,
			expectMessage:      "could not update the activeflow after appended the patched actions.",
			expectForwardStack: depTargetStackID, expectForwardAction: depTargetActionID,
		},
		{
			// the weekday is outside the range of a real weekday, so the condition never
			// matches no matter what the current time is.
			name:       "condition_datetime: false target lookup fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionDatetime,
			option:     map[string]any{"condition": "==", "weekdays": []int{7}, "false_target_id": depFalseTargetID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionDatetime(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				expectGetActionFail(m, af, depFalseTargetID)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not find false target action.",
		},
		{
			name:       "condition_datetime: progress update fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionDatetime,
			option:     map[string]any{"condition": "==", "weekdays": []int{7}, "false_target_id": depFalseTargetID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionDatetime(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				expectForward(m, af, depFalseTargetID)
				expectUpdateFail(m, af)
			},
			expectCause:        errDepInjected,
			expectMessage:      "could not update the active flow after appending the patched actions.",
			expectForwardStack: depTargetStackID, expectForwardAction: depTargetActionID,
		},
		{
			// a variable that is not a number can never satisfy a numeric condition.
			name:       "condition_variable: unparsable number goes to the false target and lookup fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionVariable,
			option: map[string]any{
				"condition": "==", "variable": "not-a-number", "value_type": string(action.OptionConditionVariableTypeNumber),
				"value_number": 1, "false_target_id": depFalseTargetID.String(),
			},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionVariable(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				expectGetActionFail(m, af, depFalseTargetID)
			},
			expectCause: errDepInjected,
		},
		{
			name:       "condition_variable: progress update fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionVariable,
			option: map[string]any{
				"condition": "==", "variable": "not-a-number", "value_type": string(action.OptionConditionVariableTypeNumber),
				"value_number": 1, "false_target_id": depFalseTargetID.String(),
			},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionVariable(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				expectForward(m, af, depFalseTargetID)
				expectUpdateFail(m, af)
			},
			expectCause:        errDepInjected,
			expectMessage:      "could not update the active flow after appending the patched actions",
			expectForwardStack: depTargetStackID, expectForwardAction: depTargetActionID,
		},
		{
			name:       "conference_join: non call reference is rejected before starting the service",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeConferenceJoin,
			option:     map[string]any{"conference_id": depConferenceID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConferenceJoin(ctx, af)
			},
			setup:         func(m *depMocks, af *activeflow.Activeflow) {},
			expectMessage: "wrong reference type. reference_type: conversation",
		},
		{
			name:       "conference_join: starting the service fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConferenceJoin,
			option:     map[string]any{"conference_id": depConferenceID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConferenceJoin(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().ConferenceV1ServiceTypeConferencecallStart(ctx, af.ID, depConferenceID, gomock.Any(), af.ReferenceID).Return(nil, errDepInjected)
			},
			expectCause:   errDepInjected,
			expectMessage: "Could not start the service.",
		},
		{
			name:       "conference_join: pushing the service actions fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConferenceJoin,
			option:     map[string]any{"conference_id": depConferenceID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConferenceJoin(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().ConferenceV1ServiceTypeConferencecallStart(ctx, af.ID, depConferenceID, gomock.Any(), af.ReferenceID).Return(&service.Service{ID: depServiceID, PushActions: depPushActions}, nil)
				expectPushFail(m, af, depServiceID, depPushActions)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not push the actions",
		},
		{
			name:       "connect: creating the confbridge fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConnect,
			option:     map[string]any{"source": map[string]any{"type": "tel", "target": "+821100000001"}},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleConnect(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1ConfbridgeCreate(ctx, af.CustomerID, af.ID, cmconfbridge.ReferenceTypeCall, af.ReferenceID, cmconfbridge.TypeConnect).Return(nil, errDepInjected)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not create a confbridge for connect",
		},
		{
			// the cause is kept opaque on purpose, so only the message is compared.
			name:       "connect: creating the temporary flow fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConnect,
			option:     map[string]any{"source": map[string]any{"type": "tel", "target": "+821100000001"}},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleConnect(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1ConfbridgeCreate(ctx, af.CustomerID, af.ID, cmconfbridge.ReferenceTypeCall, af.ReferenceID, cmconfbridge.TypeConnect).Return(&cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: depConfbridgeID}}, nil)
				m.req.EXPECT().FlowV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "tmp", "tmp flow for action connect", gomock.Any(), uuid.Nil, false).Return(nil, errDepInjected)
			},
			expectMessage: "could not create a call flow. err: synthetic dependency failure",
		},
		{
			name:       "connect: creating the outgoing call fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConnect,
			option:     map[string]any{"source": map[string]any{"type": "tel", "target": "+821100000001"}},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleConnect(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1ConfbridgeCreate(ctx, af.CustomerID, af.ID, cmconfbridge.ReferenceTypeCall, af.ReferenceID, cmconfbridge.TypeConnect).Return(&cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: depConfbridgeID}}, nil)
				m.req.EXPECT().FlowV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "tmp", "tmp flow for action connect", gomock.Any(), uuid.Nil, false).Return(&flow.Flow{Identity: commonidentity.Identity{ID: depFlowID, CustomerID: af.CustomerID}}, nil)
				m.req.EXPECT().CallV1CallsCreate(ctx, af.CustomerID, depFlowID, af.ReferenceID, gomock.Any(), gomock.Any(), false, true, "", nil, nil).Return(nil, nil, errDepInjected)
			},
			expectCause: errDepInjected,
		},
		{
			name:       "connect: no outgoing call is created",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConnect,
			option:     map[string]any{"source": map[string]any{"type": "tel", "target": "+821100000001"}},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleConnect(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1ConfbridgeCreate(ctx, af.CustomerID, af.ID, cmconfbridge.ReferenceTypeCall, af.ReferenceID, cmconfbridge.TypeConnect).Return(&cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: depConfbridgeID}}, nil)
				m.req.EXPECT().FlowV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "tmp", "tmp flow for action connect", gomock.Any(), uuid.Nil, false).Return(&flow.Flow{Identity: commonidentity.Identity{ID: depFlowID, CustomerID: af.CustomerID}}, nil)
				m.req.EXPECT().CallV1CallsCreate(ctx, af.CustomerID, depFlowID, af.ReferenceID, gomock.Any(), gomock.Any(), false, true, "", nil, nil).Return([]*cmcall.Call{}, []*cmgroupcall.Groupcall{}, nil)
			},
			expectMessage: "could not create any outgoing calls",
		},
		{
			name:       "connect: pushing the join action fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConnect,
			option:     map[string]any{"source": map[string]any{"type": "tel", "target": "+821100000001"}},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleConnect(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1ConfbridgeCreate(ctx, af.CustomerID, af.ID, cmconfbridge.ReferenceTypeCall, af.ReferenceID, cmconfbridge.TypeConnect).Return(&cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: depConfbridgeID}}, nil)
				m.req.EXPECT().FlowV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "tmp", "tmp flow for action connect", gomock.Any(), uuid.Nil, false).Return(&flow.Flow{Identity: commonidentity.Identity{ID: depFlowID, CustomerID: af.CustomerID}}, nil)
				m.req.EXPECT().CallV1CallsCreate(ctx, af.CustomerID, depFlowID, af.ReferenceID, gomock.Any(), gomock.Any(), false, true, "", nil, nil).Return([]*cmcall.Call{{Identity: commonidentity.Identity{ID: depServiceID}}}, nil, nil)
				m.util.EXPECT().UUIDCreate().Return(depTargetActionID)
				m.stack.EXPECT().PushStackByActions(af.StackMap, uuid.Nil, gomock.Any(), af.CurrentStackID, af.CurrentAction.ID).Return(nil, errDepInjected)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not push the actions",
		},
		{
			name:       "queue_join: starting the service fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeQueueJoin,
			option:     map[string]any{"queue_id": depQueueID.String()},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleQueueJoin(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().QueueV1ServiceTypeQueuecallStart(ctx, depQueueID, af.ID, qmqueuecall.ReferenceType(qmqueuecall.ReferenceTypeCall), af.ReferenceID).Return(nil, errDepInjected)
			},
			expectCause:   errDepInjected,
			expectMessage: "Could not start the service.",
		},
		{
			name:       "queue_join: pushing the service actions fails before the queuecall is updated",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeQueueJoin,
			option:     map[string]any{"queue_id": depQueueID.String()},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleQueueJoin(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().QueueV1ServiceTypeQueuecallStart(ctx, depQueueID, af.ID, qmqueuecall.ReferenceType(qmqueuecall.ReferenceTypeCall), af.ReferenceID).Return(&service.Service{ID: depServiceID, PushActions: depPushActions}, nil)
				expectPushFail(m, af, depServiceID, depPushActions)
			},
			expectCause: errDepInjected,
		},
		{
			name:       "queue_join: updating the queuecall status fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeQueueJoin,
			option:     map[string]any{"queue_id": depQueueID.String()},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleQueueJoin(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().QueueV1ServiceTypeQueuecallStart(ctx, depQueueID, af.ID, qmqueuecall.ReferenceType(qmqueuecall.ReferenceTypeCall), af.ReferenceID).Return(&service.Service{ID: depServiceID, PushActions: depPushActions}, nil)
				m.stack.EXPECT().PushStackByActions(af.StackMap, depServiceID, depPushActions, af.CurrentStackID, af.CurrentAction.ID).Return(&stack.Stack{ID: depServiceID, Actions: depPushActions}, nil)
				m.db.EXPECT().ActiveflowUpdate(ctx, af.ID, gomock.Any()).Return(nil)
				m.req.EXPECT().QueueV1QueuecallUpdateStatusWaiting(ctx, depServiceID).Return(nil, errDepInjected)
			},
			expectCause:         errDepInjected,
			expectMessage:       "Could not update the queuecall status to waiting.",
			expectForwardStack:  depServiceID,
			expectForwardAction: depTargetActionID,
		},
		{
			name:       "transcribe_recording: reading the call fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeTranscribeRecording,
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleTranscribeRecording(ctx, af)
			},
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(nil, errDepInjected)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not get the call.",
		},
		{
			name:       "branch: reading the variables fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeBranch,
			option:     map[string]any{"default_target_id": depFalseTargetID.String()},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleBranch(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.variable.EXPECT().Get(ctx, af.ID).Return(nil, errDepInjected)
			},
			expectCause: errDepInjected,
		},
		{
			name:       "branch: target lookup fails after the variable is reset",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeBranch,
			option:     map[string]any{"default_target_id": depFalseTargetID.String()},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleBranch(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.variable.EXPECT().Get(ctx, af.ID).Return(&variable.Variable{ID: af.ID, Variables: map[string]string{}}, nil)
				m.variable.EXPECT().SetVariable(ctx, af.ID, map[string]string{action.OptionBranchVariableDefault: ""}).Return(nil)
				expectGetActionFail(m, af, depFalseTargetID)
			},
			expectCause: errDepInjected,
		},
		{
			name:       "message_send: sending fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeMessageSend,
			option:     map[string]any{"source": map[string]any{"type": "tel", "target": "+821100000001"}, "text": "synthetic text"},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleMessageSend(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().MessageV1MessageSend(ctx, uuid.Nil, af.CustomerID, gomock.Any(), gomock.Any(), "synthetic text").Return(nil, errDepInjected)
			},
			expectCause: errDepInjected,
		},
		{
			name:       "call: creating the flow fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeCall,
			option:     map[string]any{"actions": []map[string]any{{"type": "answer"}}},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleCall(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().FlowV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "", "", gomock.Any(), uuid.Nil, false).Return(nil, errDepInjected)
			},
			expectMessage: "could not create a call flow. err: synthetic dependency failure",
		},
		{
			name:       "call: creating the outgoing call fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeCall,
			option:     map[string]any{"flow_id": depFlowID.String()},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleCall(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallsCreate(ctx, af.CustomerID, depFlowID, uuid.Nil, gomock.Any(), gomock.Any(), false, false, "", nil, nil).Return(nil, nil, errDepInjected)
			},
			expectCause: errDepInjected,
		},
		{
			name:       "variable_set: storing the variable fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeVariableSet,
			option:     map[string]any{"key": "synthetic_key", "value": "synthetic_value"},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleVariableSet(ctx, af) },
			setup: func(m *depMocks, af *activeflow.Activeflow) {
				m.variable.EXPECT().SetVariable(ctx, af.ID, map[string]string{"synthetic_key": "synthetic_value"}).Return(errDepInjected)
			},
			expectCause:   errDepInjected,
			expectMessage: "could not set varialbe.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newDepTestHandler(mc)
			af := newExecuteErrorTestActiveflow(tt.refType, tt.actionType, tt.option)
			tt.setup(m, af)

			err := tt.call(h, af)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if tt.expectCause != nil && !errors.Is(err, tt.expectCause) {
				t.Errorf("Wrong match. expect: %v in the chain, got: %v", tt.expectCause, err)
			}
			if tt.expectMessage != "" && !strings.Contains(err.Error(), tt.expectMessage) {
				t.Errorf("Wrong match. expect message: %s, got: %v", tt.expectMessage, err)
			}

			if af.ForwardStackID != tt.expectForwardStack || af.ForwardActionID != tt.expectForwardAction {
				t.Errorf("Wrong match. expect forward: %s %s, got: %s %s", tt.expectForwardStack, tt.expectForwardAction, af.ForwardStackID, af.ForwardActionID)
			}
		})
	}
}
