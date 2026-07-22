package aicallhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	tmcorrelation "monorepo/bin-timeline-manager/models/correlation"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *aicallHandler) ToolHandle(ctx context.Context, id uuid.UUID, toolID string, toolType message.ToolType, function message.FunctionCall) (map[string]any, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "ToolHandle",
		"aicall_id": id,
		"tool_id":   toolID,
		"tool_type": toolType,
		"function":  function,
	})

	c, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get aicall %s", id)
	}
	log.WithField("aicall", c).Debugf("fetched aicall info.")

	tool := &message.ToolCall{
		ID:       toolID,
		Type:     toolType,
		Function: function,
	}

	// create a message for tool handle request
	toolCallActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, c)
	tmp, errCreate := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionIncoming, message.RoleAssistant, "", []message.ToolCall{*tool}, "",
		messagehandler.WithActiveAIID(toolCallActiveAIID))
	if errCreate != nil {
		return nil, errors.Wrapf(errCreate, "could not create the tool message")
	}
	log.WithField("message", tmp).Debugf("Created the tool message for the actions. message_id: %s", tmp.ID)

	mapFunctions := map[message.FunctionCallName]func(context.Context, *aicall.AIcall, *message.ToolCall) *messageContent{
		message.FunctionCallNameConnectCall:            h.toolHandleConnect,
		message.FunctionCallNameCreateCall:             h.toolHandleCreateCall,
		message.FunctionCallNameGetVariables:           h.toolHandleGetVariables,
		message.FunctionCallNameGetAIcallMessages:      h.toolHandleGetAIcallMessages,
		message.FunctionCallNameSendEmail:              h.toolHandleEmailSend,
		message.FunctionCallNameSendMessage:            h.toolHandleMessageSend,
		message.FunctionCallNameSetVariables:           h.toolHandleSetVariables,
		message.FunctionCallNameStopFlow:               h.toolHandleStop,
		message.FunctionCallNameStopMedia:              h.toolHandleMediaStop,
		message.FunctionCallNameStopService:            h.toolHandleServiceStop,
		message.FunctionCallNameSearchKnowledge:        h.toolHandleSearchKnowledge,
		message.FunctionCallNameGetCorrelation:         h.toolHandleGetCorrelation,
		message.FunctionCallNameGetResource:            h.toolHandleGetResource,
		message.FunctionCallNameDescribeAction:         h.toolHandleDescribeAction,
		message.FunctionCallNameCaseCreate:             h.toolHandleCaseCreate,
		message.FunctionCallNameGetContactInteractions: h.toolHandleGetContactInteractions,
		message.FunctionCallNameGetConversationContent: h.toolHandleGetConversationContent,
	}

	promAIcallToolExecuteTotal.WithLabelValues(string(tool.Function.Name)).Inc()

	var tmpMessageContent *messageContent
	if fn, exists := mapFunctions[tool.Function.Name]; exists {
		tmpMessageContent = fn(ctx, c, tool)
	} else {
		log.Debugf("unknown tool call: %s", tool.Function.Name)
		return nil, fmt.Errorf("unknown tool call: %s", tool.Function.Name)
	}

	msg, err := h.toolCreateResultMessage(ctx, c, tool, tmpMessageContent, toolCallActiveAIID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the tool message")
	}
	log.WithField("message", msg).Debugf("Created the tool response message. message_id: %s", msg.ID)

	res, err := h.unmarshalToolResponse(msg)
	if err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal the tool response message content correctly")
	}

	return res, nil
}

type messageContent struct {
	ToolCallID   string `json:"tool_call_id"`
	Result       string `json:"result"`
	Message      string `json:"message"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

func (h *aicallHandler) toolCreateResultMessage(
	ctx context.Context,
	c *aicall.AIcall,
	tool *message.ToolCall,
	tmpContent *messageContent,
	activeAIID uuid.UUID,
) (*message.Message, error) {

	content, err := json.Marshal(tmpContent)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the tool result content")
	}

	tmp, err := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, string(content), nil, tool.ID,
		messagehandler.WithActiveAIID(activeAIID))
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the tool message")
	}
	return tmp, nil
}

func newToolResult(toolID string) *messageContent {
	return &messageContent{ToolCallID: toolID}
}

func fillFailed(mc *messageContent, err error) {
	mc.Result = "failed"
	mc.Message = err.Error()
}

func fillSuccess(mc *messageContent, rType, rID, msg string) {
	mc.Result = "success"
	mc.ResourceType = rType
	mc.ResourceID = rID
	mc.Message = msg
}

func (h *aicallHandler) unmarshalToolResponse(tmp *message.Message) (map[string]any, error) {
	var res map[string]any
	if err := json.Unmarshal([]byte(tmp.Content), &res); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal the tool response message content")
	}
	return res, nil
}

func (h *aicallHandler) toolHandleConnect(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleConnect",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool connect.")

	res := newToolResult(tool.ID)
	if c.ReferenceType != aicall.ReferenceTypeCall {
		fillFailed(res, fmt.Errorf("connect_call is only supported for call reference type"))
		return res
	}

	var tmpOpt fmaction.OptionConnect
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &tmpOpt); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	opt := fmaction.ConvertOption(tmpOpt)
	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeConnect,
			Option: opt,
		},
	}

	af, err := h.reqHandler.FlowV1ActiveflowAddActions(ctx, c.ActiveflowID, actions)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	log.WithField("activeflow", af).Debugf("Added actions to the activeflow. activeflow_id: %s", c.ActiveflowID)
	fillSuccess(res, "activeflow", af.ID.String(), "Added connect action successfully.")

	go func() {
		// this will connect the call right away
		tmp, err := h.reqHandler.AIV1AIcallTerminate(context.Background(), c.ID)
		if err != nil {
			log.Errorf("Could not terminate the aicall after sending the tool actions. err: %v", err)
			return
		}
		log.WithField("aicall", tmp).Debugf("Terminating the aicall after sending the tool actions. aicall_id: %s", c.ID)
	}()

	return res
}

// errCouldNotResolveFlow is the single masked error used for both flow-not-found and
// cross-customer-flow paths in toolHandleCreateCall, so the tool cannot be used as a
// flow-existence oracle. Bare sentinel (no stack, no %w wrap) keeps both paths byte-identical.
var errCouldNotResolveFlow = stderrors.New("could not resolve flow")

// toolHandleCreateCall places a NEW, INDEPENDENT outbound call that is NOT bridged to the
// current session and does NOT terminate the current AIcall (contrast with toolHandleConnect).
// The originated call runs its own flow_id; the current AI session continues.
func (h *aicallHandler) toolHandleCreateCall(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleCreateCall",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool create_call.")

	res := newToolResult(tool.ID)

	// 1. parse args
	var args struct {
		FlowID       uuid.UUID               `json:"flow_id"`
		Actions      []fmaction.Action       `json:"actions,omitempty"`
		Source       *commonaddress.Address  `json:"source,omitempty"`
		Destinations []commonaddress.Address `json:"destinations"`
		Anonymous    string                  `json:"anonymous,omitempty"`
		// Variables is parsed as map[string]any then coerced to map[string]string, because an
		// LLM frequently emits non-string scalar values (e.g. {"campaign_id": 123}). Unmarshaling
		// straight into map[string]string would fail and abort the whole tool call.
		Variables map[string]any `json:"variables,omitempty"`
	}
	// Use a json.Decoder with UseNumber() so JSON numbers in the Variables map[string]any field
	// arrive as json.Number (preserving large-integer precision) rather than float64. UseNumber
	// only affects numbers decoded into interface{}, so typed fields (FlowID uuid.UUID, etc.)
	// still unmarshal correctly.
	dec := json.NewDecoder(strings.NewReader(tool.Function.Arguments))
	dec.UseNumber()
	if errUnmarshal := dec.Decode(&args); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	// flow_id XOR actions: exactly one must be provided. An empty actions array ([]) counts as
	// "not provided" (len check), so it falls into the neither-provided rejection.
	hasFlowID := args.FlowID != uuid.Nil
	hasActions := len(args.Actions) > 0
	switch {
	case hasFlowID && hasActions:
		fillFailed(res, fmt.Errorf("provide either flow_id or actions, not both"))
		return res
	case !hasFlowID && !hasActions:
		fillFailed(res, fmt.Errorf("either flow_id or actions is required"))
		return res
	}

	if len(args.Destinations) == 0 {
		fillFailed(res, fmt.Errorf("at least one destination is required"))
		return res
	}
	// input hygiene: an empty destination target would be silently skipped by call-manager.
	for _, d := range args.Destinations {
		if d.Target == "" {
			fillFailed(res, fmt.Errorf("destination target must not be empty"))
			return res
		}
	}

	// coerce LLM-supplied variables (map[string]any) to map[string]string. Scalar values are
	// stringified; non-scalar values are skipped. Final sanitization (reserved-key drop, size
	// caps) is enforced by flow-manager, not here.
	variables := fmvariable.NewVariablesFromMap(args.Variables)

	// 2. resolve the target flow.
	//    - actions path: assemble an ephemeral (persist=false, cache-only, TTL-expiring) flow
	//      from the LLM-supplied actions. The actions are passed through unchanged: the LLM may
	//      include id/next_id to express non-linear control flow (goto/branch/condition_* carry
	//      their jump targets inside option referencing actions by id), so resetting ids would
	//      dangle those targets. flow-manager ValidateActions (via FlowV1FlowCreate) rejects
	//      unknown action types and invalid anonymous values. Composition limits (count,
	//      recursion, loop) are enforced at the flow-manager activeflow layer, not here.
	//      The ephemeral flow is owned by c.CustomerID by construction, so no ownership check is
	//      needed and there is no caller-supplied flow id to leak as an existence oracle.
	//    - flow_id path: SECURITY: flow ownership (IDOR prevention). Both not-found and
	//      cross-customer return the same byte-identical masked error so the tool is not a
	//      flow-existence oracle.
	var targetFlowID uuid.UUID
	if hasActions {
		f, err := h.reqHandler.FlowV1FlowCreate(
			ctx, c.CustomerID, fmflow.TypeFlow, "tmp", "tmp flow for ai create_call action assembly", args.Actions, uuid.Nil, false)
		if err != nil {
			log.Errorf("Could not create the ephemeral flow for create_call actions. err: %v", err)
			fillFailed(res, err)
			return res
		}
		log.WithField("flow_id", f.ID).Debugf("Created ephemeral flow for create_call actions. flow_id: %s, action_count: %d", f.ID, len(args.Actions))
		targetFlowID = f.ID
	} else {
		f, err := h.reqHandler.FlowV1FlowGet(ctx, args.FlowID)
		if err != nil || f == nil {
			log.Errorf("Could not get the flow. flow_id: %s, err: %v", args.FlowID, err)
			fillFailed(res, errCouldNotResolveFlow)
			return res
		}
		if f.CustomerID != c.CustomerID {
			log.Warnf("Flow does not belong to the customer. flow_id: %s, flow_customer_id: %s, customer_id: %s", args.FlowID, f.CustomerID, c.CustomerID)
			fillFailed(res, errCouldNotResolveFlow)
			return res
		}
		targetFlowID = args.FlowID
	}

	// 3. originate. masterCallID = uuid.Nil (no bridge). The 8th bool (executeNextMasterOnHangup)
	//    is false: it only governs whether a MASTER call's flow advances when a chained call hangs
	//    up, which is irrelevant with no master. The originated call self-runs its own flow on
	//    answer via call-manager (status.go updateStatusProgressing -> ActionNext).
	src := commonaddress.Address{}
	if args.Source != nil {
		src = *args.Source
	}
	calls, groupcalls, err := h.reqHandler.CallV1CallsCreate(
		ctx, c.CustomerID, targetFlowID, uuid.Nil, &src, args.Destinations, false, false, args.Anonymous, nil, variables)
	// CONTRACT: CreateCallsOutgoing returns an error ONLY when ALL destinations fail (nil,nil,err).
	// Partial/full success returns (calls,groupcalls,nil). There is no (err + non-empty slices) case,
	// so a single err!=nil check is correct and total.
	if err != nil {
		log.Errorf("Could not create outgoing calls for create_call. flow_id: %s, err: %v", targetFlowID, err)
		fillFailed(res, err)
		return res
	}

	// 4. return originated ids for tracking.
	//    Per-destination failures are swallowed by call-manager (logged + continue), so the only
	//    honest partial signal is requested-vs-created count. Surface it explicitly.
	//    INVARIANT (load-bearing): each destination yields at most one leg (one call OR one
	//    groupcall; fan-out is encapsulated inside a single groupcall). So created <= requested.
	type createCallResult struct {
		CallIDs      []string `json:"call_ids"`
		GroupcallIDs []string `json:"groupcall_ids"`
		Requested    int      `json:"requested"`
		Created      int      `json:"created"`
		Partial      bool     `json:"partial,omitempty"`
	}
	out := createCallResult{
		CallIDs:      []string{},
		GroupcallIDs: []string{},
		Requested:    len(args.Destinations),
	}
	for _, cc := range calls {
		out.CallIDs = append(out.CallIDs, cc.ID.String())
	}
	for _, gc := range groupcalls {
		out.GroupcallIDs = append(out.GroupcallIDs, gc.ID.String())
	}
	out.Created = len(out.CallIDs) + len(out.GroupcallIDs)
	out.Partial = out.Created < out.Requested

	body, errMarshal := json.Marshal(out)
	if errMarshal != nil {
		fillFailed(res, errMarshal)
		return res
	}

	// primary handle + correct resource type (groupcall-only must not be tagged "call").
	// INVARIANT: err==nil guarantees Created>=1 per CreateCallsOutgoing's contract (it errors
	// only when ALL destinations fail), so primaryID is always populated here.
	primaryID, primaryType := "", "call"
	if len(calls) > 0 {
		primaryID = calls[0].ID.String()
	} else if len(groupcalls) > 0 {
		primaryID, primaryType = groupcalls[0].ID.String(), "groupcall"
	}

	log.WithFields(logrus.Fields{
		"flow_id":   targetFlowID,
		"requested": out.Requested,
		"created":   out.Created,
	}).Debugf("Created independent outgoing call(s) for create_call. primary_id: %s", primaryID)
	fillSuccess(res, primaryType, primaryID, string(body))

	return res
}

func (h *aicallHandler) toolHandleMessageSend(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleMessageSend",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool message_send.")

	res := newToolResult(tool.ID)

	var tmpOpt fmaction.OptionMessageSend
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &tmpOpt); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	msgID := h.utilHandler.UUIDCreate()
	tmpRes, err := h.reqHandler.MessageV1MessageSend(ctx, msgID, c.CustomerID, tmpOpt.Source, tmpOpt.Destinations, tmpOpt.Text)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	fillSuccess(res, "message", tmpRes.ID.String(), "Message sent successfully.")

	return res
}

func (h *aicallHandler) toolHandleEmailSend(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleEmailSend",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool email_send.")

	res := newToolResult(tool.ID)

	var tmpOpt fmaction.OptionEmailSend
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &tmpOpt); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	tmpRes, err := h.reqHandler.EmailV1EmailSend(
		ctx,
		c.CustomerID,
		c.ActiveflowID,
		tmpOpt.Destinations,
		tmpOpt.Subject,
		tmpOpt.Content,
		tmpOpt.Attachments,
	)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	fillSuccess(res, "email", tmpRes.ID.String(), "Email sent successfully.")

	return res
}

// caseCreateCRMIneligiblePeerTypes duplicates
// bin-flow-manager/pkg/activeflowhandler's crmIneligiblePeerTypes (design
// VOIP-1243 §5.2/§6.3) -- the original is unexported in bin-contact-manager
// and cannot be imported.
var caseCreateCRMIneligiblePeerTypes = map[commonaddress.Type]struct{}{
	commonaddress.TypeNone:       {}, // zero-value/unknown-direction peer -- never a real contact (VOIP-1243 round-review fix)
	commonaddress.TypeAgent:      {},
	commonaddress.TypeAI:         {},
	commonaddress.TypeAITeam:     {},
	commonaddress.TypeConference: {},
	commonaddress.TypeExtension:  {},
	commonaddress.TypeSIP:        {},
	"web_session":                {}, // synthetic type; not in commonaddress.Type enum
}

// isCRMEligiblePeer reports whether the given peer address type can ever
// represent an external, re-identifiable contact. Duplicated from
// contacthandler.isCRMEligiblePeer / bin-flow-manager's copy (design
// VOIP-1243 §6.3).
func isCRMEligiblePeer(peerType commonaddress.Type) bool {
	_, ineligible := caseCreateCRMIneligiblePeerTypes[peerType]
	return !ineligible
}

// deriveEndpointsForCase resolves which address is the remote peer and
// which is our local endpoint based on the call direction. Mirrors
// bin-flow-manager's deriveEndpointsForCase / contacthandler.deriveEndpoints
// (design VOIP-1243 §6.3).
func deriveEndpointsForCase(direction string, source, dest commonaddress.Address) (peer commonaddress.Address, self commonaddress.Address) {
	switch direction {
	case "incoming":
		return source, dest
	case "outgoing":
		return dest, source
	default:
		return commonaddress.Address{}, commonaddress.Address{}
	}
}

// deriveCaseEndpointsForAIcall derives the case peer/self/reference_type
// for an AIcall, mirroring bin-flow-manager's actionHandleCaseCreate
// derivation logic (design VOIP-1243 §6.3). supported=false means "no
// defined derivation for this reference type" -- a deliberate scope
// limit, not an error, and is reported to the LLM via fillSuccess. A
// non-nil err means a genuine downstream RPC failure (CallV1CallGet /
// ConversationV1ConversationGet erroring) and MUST be reported via
// fillFailed per design §6.3/§8 -- these two outcomes are NOT the same
// and must not be collapsed into a single bool (round-review fix).
func (h *aicallHandler) deriveCaseEndpointsForAIcall(ctx context.Context, c *aicall.AIcall) (peer commonaddress.Address, self commonaddress.Address, referenceType string, supported bool, err error) {
	switch c.ReferenceType {
	case aicall.ReferenceTypeCall:
		cl, errGet := h.reqHandler.CallV1CallGet(ctx, c.ReferenceID)
		if errGet != nil {
			return commonaddress.Address{}, commonaddress.Address{}, "", true, errGet
		}
		peer, self = deriveEndpointsForCase(string(cl.Direction), cl.Source, cl.Destination)
		return peer, self, "call", true, nil

	case aicall.ReferenceTypeConversation:
		cv, errGet := h.reqHandler.ConversationV1ConversationGet(ctx, c.ReferenceID)
		if errGet != nil {
			return commonaddress.Address{}, commonaddress.Address{}, "", true, errGet
		}
		return cv.Peer, cv.Self, "conversation_message", true, nil

	default:
		return commonaddress.Address{}, commonaddress.Address{}, "", false, nil
	}
}

// toolHandleCaseCreate handles tool call case_create: creates a new
// contact CRM case for the AIcall's current call/conversation reference.
// Per design VOIP-1243 §6.3/§8: CRM-ineligibility is reported via
// fillSuccess (not a failure -- symmetric with the Flow action's silent
// skip). A ContactV1CaseCreate error (AlreadyExists/Unavailable/other) IS
// reported to the LLM via fillFailed, unlike the Flow action which only
// logs -- the tool caller (the LLM) has no other way to learn the
// attempt did not produce a new case.
func (h *aicallHandler) toolHandleCaseCreate(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleCaseCreate",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool case_create.")

	res := newToolResult(tool.ID)

	var tmpOpt fmaction.OptionCaseCreate
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &tmpOpt); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	peer, self, referenceType, supported, errDerive := h.deriveCaseEndpointsForAIcall(ctx, c)
	if errDerive != nil {
		fillFailed(res, errDerive)
		return res
	}
	if !supported {
		fillSuccess(res, "case", "", "No case was created: this reference type does not support case tracking.")
		return res
	}

	if !isCRMEligiblePeer(peer.Type) {
		fillSuccess(res, "case", "", fmt.Sprintf("No case was created: peer type %s is not eligible for CRM case tracking.", peer.Type))
		return res
	}

	// Activeflow-scoped dedup (design VOIP-1243 §3.5): check the
	// activeflow's own variable store BEFORE calling ContactV1CaseCreate.
	if v, errVar := h.reqHandler.FlowV1VariableGet(ctx, c.ActiveflowID); errVar == nil && v != nil {
		if existingCaseID, ok := v.Variables["contact_case_id"]; ok && existingCaseID != "" {
			fillSuccess(res, "case", existingCaseID, "A case already exists for this call/conversation; no new case was created.")
			return res
		}
	}

	peerTarget, errNormalize := commonaddress.NormalizeTarget(peer.Type, peer.Target)
	if errNormalize != nil {
		log.WithError(errNormalize).Warnf("could not normalize peer target; using raw value. peer_type: %s", peer.Type)
		peerTarget = peer.Target
	}
	peerAddr := peer
	peerAddr.Target = peerTarget // override with the normalized value; TargetName/Name/Detail pass through unchanged

	created, errCreate := h.reqHandler.ContactV1CaseCreate(ctx, c.CustomerID, self, peerAddr, referenceType, tmpOpt.Name, tmpOpt.Detail)
	if errCreate != nil {
		fillFailed(res, errCreate)
		return res
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, c.ActiveflowID, map[string]string{"contact_case_id": created.ID.String()}); errSet != nil {
		log.Errorf("could not set contact_case_id variable. err: %v", errSet) // best-effort; the Case itself was created successfully regardless
	}

	if tmpOpt.Note != "" {
		if _, errNote := h.reqHandler.ContactV1CaseNoteCreate(ctx, c.CustomerID, created.ID, "ai", nil, tmpOpt.Note); errNote != nil {
			log.Errorf("could not create initial case note. err: %v", errNote) // best-effort
		}
	}

	fillSuccess(res, "case", created.ID.String(), "Case created successfully.")

	return res
}

func (h *aicallHandler) toolHandleServiceStop(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleServiceStop",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool service_stop.")

	res := newToolResult(tool.ID)
	if errStop := h.serviceStop(ctx, c); errStop != nil {
		fillFailed(res, errStop)
		return res
	}

	fillSuccess(res, "service", c.ID.String(), "Service stopped successfully.")

	return res
}

func (h *aicallHandler) toolHandleStop(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleStop",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool stop.")

	res := newToolResult(tool.ID)
	tmpActiveflow, err := h.reqHandler.FlowV1ActiveflowStop(ctx, c.ActiveflowID)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	log.WithField("activeflow", tmpActiveflow).Debugf("Stopped the activeflow. activeflow_id: %s", c.ActiveflowID)
	fillSuccess(res, "activeflow", tmpActiveflow.ID.String(), "Activeflow stopped successfully.")

	return res
}

func (h *aicallHandler) toolHandleMediaStop(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleMediaStop",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool media_stop.")

	res := newToolResult(tool.ID)
	if c.ReferenceType != aicall.ReferenceTypeCall {
		fillFailed(res, fmt.Errorf("invalid reference type for media stop"))
		return res
	}

	log.Debugf("Stopping call media playing. call_id: %s", c.ReferenceID)
	if errStop := h.reqHandler.CallV1CallMediaStop(ctx, c.ReferenceID); errStop != nil {
		fillFailed(res, errStop)
		return res
	}

	log.Debugf("Stopped the call media playing. call_id: %s", c.ReferenceID)
	fillSuccess(res, "call", c.ReferenceID.String(), "Call media stopped successfully.")

	return res
}

func (h *aicallHandler) toolHandleSetVariables(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleSetVariables",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool set_variables.")

	res := newToolResult(tool.ID)

	if c.ActiveflowID == uuid.Nil {
		fillFailed(res, fmt.Errorf("no activeflow associated with this aicall"))
		return res
	}

	req := struct {
		Variables map[string]string `json:"variables"`
	}{}

	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &req); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	log.Debugf("Setting the activeflow variables. activeflow_id: %s", c.ActiveflowID)
	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, c.ActiveflowID, req.Variables); errSet != nil {
		fillFailed(res, errSet)
		return res
	}

	log.Debugf("Set activeflow variables successfully. activeflow_id: %s", c.ActiveflowID)
	fillSuccess(res, "activeflow", c.ActiveflowID.String(), "Variables set successfully.")

	return res
}

func (h *aicallHandler) toolHandleGetVariables(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetVariables",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_variables.")

	res := newToolResult(tool.ID)

	if c.ActiveflowID == uuid.Nil {
		fillFailed(res, fmt.Errorf("no activeflow associated with this aicall"))
		return res
	}

	tmp, err := h.reqHandler.FlowV1VariableGet(ctx, c.ActiveflowID)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	tmpMessage, err := json.Marshal(tmp.Variables)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	log.Debugf("Got activeflow variables successfully. activeflow_id: %s", c.ActiveflowID)
	fillSuccess(res, "activeflow", c.ActiveflowID.String(), string(tmpMessage))

	return res
}

func (h *aicallHandler) toolHandleGetAIcallMessages(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetAIcallMessages",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_aicall_messages.")

	res := newToolResult(tool.ID)

	req := struct {
		AICallID uuid.UUID `json:"aicall_id"`
	}{}

	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &req); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	// get ai call info
	tmp, err := h.Get(ctx, req.AICallID)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	if tmp.CustomerID != c.CustomerID {
		fillFailed(res, fmt.Errorf("aicall customer id does not match"))
		return res
	}

	messages, err := h.messageHandler.List(ctx, 1000, "", map[message.Field]any{
		message.FieldAIcallID: tmp.ID,
	})
	if err != nil {
		fillFailed(res, err)
		return res
	}

	tmpMessage, err := json.Marshal(messages)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	log.Debugf("Got aicall messages successfully. aicall_id: %s", c.ID)
	fillSuccess(res, "messages", tmp.ID.String(), string(tmpMessage))

	return res
}

func (h *aicallHandler) toolHandleSearchKnowledge(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleSearchKnowledge",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool search_knowledge.")

	res := newToolResult(tc.ID)

	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		fillFailed(res, err)
		return res
	}

	tmpAI, _, _, err := h.resolveAI(ctx, c.AssistanceType, c.AssistanceID)
	if err != nil {
		log.Errorf("Could not resolve AI. err: %v", err)
		fillFailed(res, fmt.Errorf("could not retrieve AI configuration"))
		return res
	}
	log.WithField("ai", tmpAI).Debugf("Retrieved AI info. ai_id: %s", tmpAI.ID)

	if tmpAI.RagID == uuid.Nil {
		fillFailed(res, fmt.Errorf("no knowledge base is configured for this assistant"))
		return res
	}

	ragRes, err := h.reqHandler.RagV1RagQuery(ctx, tmpAI.RagID, args.Query, 5)
	if err != nil {
		log.Errorf("RAG query failed. err: %v", err)
		fillFailed(res, fmt.Errorf("knowledge base search failed"))
		return res
	}
	log.Debugf("RAG query completed. rag_id: %s, source_count: %d", tmpAI.RagID, len(ragRes.Sources))

	if len(ragRes.Sources) == 0 {
		fillSuccess(res, "rag", tmpAI.RagID.String(), "No relevant information found in the knowledge base.")
		return res
	}

	var sb strings.Builder
	for i, s := range ragRes.Sources {
		fmt.Fprintf(&sb, "[Source %d: \"%s\" > \"%s\" (relevance: %.2f)]\n",
			i+1, s.DocumentName, s.SectionTitle, s.RelevanceScore)
		sb.WriteString(s.Text)
		sb.WriteString("\n\n")
	}

	fillSuccess(res, "rag", tmpAI.RagID.String(), sb.String())
	return res
}

// msgCorrelationNotFound is the single masking string used for every
// "you cannot see this" path so that genuine-absent, exists-but-not-owned,
// and ownership-lookup-failure are all byte-identical. This prevents the tool
// from being used as a cross-customer existence oracle.
const msgCorrelationNotFound = "No events found for this resource."

// ErrCorrelationNotAccessible is the single sentinel covering every outcome where
// the caller may not see the correlation: genuinely absent, exists-but-cross-customer,
// ownership-lookup failure, and foreign-id-with-no-activeflow. Callers MUST collapse
// it to the byte-identical msgCorrelationNotFound so the tool cannot be used as a
// cross-customer existence oracle. It deliberately does NOT distinguish the four
// causes — that is the security property.
//
// Uses stdlib errors.New (aliased stderrors): a sentinel needs no stack trace, and
// identity comparison via stderrors.Is is all that matters.
var ErrCorrelationNotAccessible = stderrors.New("correlation not accessible")

// resolveCorrelation fetches the correlation graph for resourceID and validates that
// the caller (callerCustomerID) owns it.
//
// Returns:
//   - (corr, nil)                         : access granted. corr is non-nil.
//     corr.ActiveflowID may be uuid.Nil only when ownSession is true (caller's own
//     unlinked resource).
//   - (corr, ErrCorrelationNotAccessible) : caller may not see this. corr is non-nil
//     but MUST NOT be exposed; the caller masks and ignores corr entirely.
//   - (nil, <wrapped err>)                : transient/infra failure (e.g. timeline RPC
//     down). corr is nil. Existence is unknown; caller reports a tool failure.
//
// CONTRACT: corr is meaningful ONLY when err == nil. On any non-nil error the caller
// MUST NOT dereference corr (it is nil for the transient case). The handler enforces
// this by returning inside the error block before reading corr.
//
// ownSession must be true iff the caller did not supply a resource_id (target is the
// caller's own activeflow, owned by definition).
//
// Security: the timeline correlation endpoint is not customer-scoped, so an arbitrary
// resource_id could otherwise expose another customer's data (IDOR). Ownership is
// enforced by resolving the correlated activeflow via flow-manager and comparing its
// CustomerID against the caller's. Failures that could reveal a resource's existence
// fail closed (return the sentinel -> masked).
func (h *aicallHandler) resolveCorrelation(
	ctx context.Context,
	callerCustomerID uuid.UUID,
	resourceID uuid.UUID,
	ownSession bool,
) (*tmcorrelation.Correlation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "resolveCorrelation",
		"customer_id": callerCustomerID,
		"resource_id": resourceID,
	})

	corr, err := h.reqHandler.TimelineV1CorrelationGet(ctx, resourceID)
	if err != nil {
		// Existence unknown -> genuine tool failure, NOT a mask. corr is nil.
		return nil, errors.Wrap(err, "correlation lookup failed")
	}

	// Resource absent -> caller may not see it.
	if !corr.ResourceFound {
		return corr, ErrCorrelationNotAccessible
	}

	// Resource exists but has no activeflow: there is no activeflow to validate
	// ownership against. Only disclose this state for the caller's own session;
	// for a supplied foreign id, mask.
	if corr.ActiveflowID == uuid.Nil {
		if ownSession {
			return corr, nil
		}
		return corr, ErrCorrelationNotAccessible
	}

	// Has activeflow: validate ownership via flow-manager (timeline has no customer_id).
	af, err := h.reqHandler.FlowV1ActiveflowGet(ctx, corr.ActiveflowID)
	if err != nil {
		// Mask the lookup failure: do not reveal that the activeflow exists. Fail closed.
		log.Warnf("Could not verify correlation ownership. err: %v", err)
		return corr, ErrCorrelationNotAccessible
	}
	if af.CustomerID != callerCustomerID {
		log.Warnf("Cross-customer correlation attempt blocked. resource_owner: %s", af.CustomerID)
		return corr, ErrCorrelationNotAccessible
	}

	return corr, nil
}

// toolHandleGetCorrelation retrieves the correlation graph for a resource and
// returns a human-readable summary. It is an internal diagnostic tool.
//
// Ownership validation and existence-oracle masking are delegated to
// resolveCorrelation. Every "cannot see this" path collapses to the single
// msgCorrelationNotFound emission site below.
//
// CRITICAL: both branches inside the `if err` block MUST return. Falling through
// would (a) for a cross-customer error expose a foreign activeflow summary (IDOR),
// and (b) for the transient case dereference a nil corr (panic). The returns are
// load-bearing.
func (h *aicallHandler) toolHandleGetCorrelation(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetCorrelation",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_correlation.")

	res := newToolResult(tc.ID)

	var args struct {
		ResourceID string `json:"resource_id"`
	}
	// resource_id is optional; ignore unmarshal errors and fall back to own session.
	_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

	// ownSession is true when no id is supplied: the target is the caller's own
	// activeflow, which is owned by definition.
	ownSession := args.ResourceID == ""
	resourceID := c.ActiveflowID
	if !ownSession {
		parsed, err := uuid.FromString(args.ResourceID)
		if err != nil {
			fillFailed(res, fmt.Errorf("invalid resource_id"))
			return res
		}
		resourceID = parsed
	}
	if resourceID == uuid.Nil {
		fillFailed(res, fmt.Errorf("no resource_id available"))
		return res
	}

	corr, err := h.resolveCorrelation(ctx, c.CustomerID, resourceID, ownSession)
	if err != nil {
		if stderrors.Is(err, ErrCorrelationNotAccessible) {
			// Single masking site for ALL not-accessible paths.
			fillSuccess(res, "correlation", resourceID.String(), msgCorrelationNotFound)
			return res
		}
		// Transient/infra failure: existence unknown, report tool failure.
		log.Errorf("Correlation lookup failed. err: %v", err)
		fillFailed(res, fmt.Errorf("correlation lookup failed"))
		return res
	}

	// err == nil below: corr is safe to read.
	// Own-session unlinked resource gets the disclosure message.
	if corr.ActiveflowID == uuid.Nil {
		fillSuccess(res, "correlation", resourceID.String(), "This resource exists but is not linked to any activeflow.")
		return res
	}

	summary := formatCorrelationSummary(corr)
	fillSuccess(res, "correlation", corr.ActiveflowID.String(), summary)
	return res
}

// correlationResourceLabel derives a human/LLM-readable resource-type label
// from a resource's event types (e.g. "call_created" → "call",
// "aicall_status_progressing" → "aicall"). The event envelope's data_type is
// always "application/json" (the content type), so the event-type prefix is
// the only usable label source. Empty event types fall back to the neutral
// label "resource".
func correlationResourceLabel(eventTypes []string) string {
	if len(eventTypes) == 0 || eventTypes[0] == "" {
		return "resource"
	}
	first := eventTypes[0]
	if idx := strings.Index(first, "_"); idx > 0 {
		return first[:idx]
	}
	if strings.HasPrefix(first, "_") {
		return "resource"
	}
	return first
}

// formatCorrelationSummary renders an LLM-readable summary of a correlation
// graph. It leads with prose grouped by publisher and includes compact resource
// ids so the LLM can chain follow-up tool calls.
func formatCorrelationSummary(corr *tmcorrelation.Correlation) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Activeflow %s is linked to:\n", corr.ActiveflowID)
	if len(corr.Resources) == 0 {
		sb.WriteString("- (no correlated resources)\n")
	}

	for _, group := range corr.Resources {
		if group == nil {
			continue
		}
		fmt.Fprintf(&sb, "- %s: %d resource(s)\n", group.Publisher, len(group.Resources))
		for _, r := range group.Resources {
			if r == nil {
				continue
			}
			if len(r.EventTypes) > 0 {
				fmt.Fprintf(&sb, "  - %s %s (events: %s)\n", correlationResourceLabel(r.EventTypes), r.ID, strings.Join(r.EventTypes, ", "))
			} else {
				fmt.Fprintf(&sb, "  - %s %s\n", correlationResourceLabel(r.EventTypes), r.ID)
			}
		}
	}

	if corr.Truncated {
		sb.WriteString("(truncated: more resources exist than were returned)\n")
	}

	return sb.String()
}
