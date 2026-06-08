package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	fmaction "monorepo/bin-flow-manager/models/action"
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
		message.FunctionCallNameConnectCall:       h.toolHandleConnect,
		message.FunctionCallNameGetVariables:      h.toolHandleGetVariables,
		message.FunctionCallNameGetAIcallMessages: h.toolHandleGetAIcallMessages,
		message.FunctionCallNameSendEmail:         h.toolHandleEmailSend,
		message.FunctionCallNameSendMessage:       h.toolHandleMessageSend,
		message.FunctionCallNameSetVariables:      h.toolHandleSetVariables,
		message.FunctionCallNameStopFlow:          h.toolHandleStop,
		message.FunctionCallNameStopMedia:           h.toolHandleMediaStop,
		message.FunctionCallNameStopService:         h.toolHandleServiceStop,
		message.FunctionCallNameSearchKnowledge:     h.toolHandleSearchKnowledge,
		message.FunctionCallNameGetCorrelation:      h.toolHandleGetCorrelation,
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

// toolHandleGetCorrelation retrieves the correlation graph for a resource and
// returns a human-readable summary. It is an internal diagnostic tool.
//
// Security: the timeline correlation endpoint is not customer-scoped, so an
// arbitrary resource_id could otherwise expose another customer's data (IDOR).
// Ownership is enforced by resolving the correlated activeflow via flow-manager
// and comparing its CustomerID against the calling session's CustomerID. Any
// path that would reveal existence of a resource the caller does not own returns
// the same masking string as a genuine not-found.
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

	corr, err := h.reqHandler.TimelineV1ResourceCorrelationGet(ctx, resourceID)
	if err != nil {
		log.Errorf("Correlation lookup failed. err: %v", err)
		fillFailed(res, fmt.Errorf("correlation lookup failed"))
		return res
	}

	// Resource absent -> canonical not-found.
	if !corr.ResourceFound {
		fillSuccess(res, "correlation", resourceID.String(), msgCorrelationNotFound)
		return res
	}

	// Resource exists but has no activeflow: there is no activeflow to validate
	// ownership against. Only disclose this state for the caller's own session;
	// for a supplied foreign id, mask as not-found.
	if corr.ActiveflowID == uuid.Nil {
		if ownSession {
			fillSuccess(res, "correlation", resourceID.String(), "This resource exists but is not linked to any call flow.")
		} else {
			fillSuccess(res, "correlation", resourceID.String(), msgCorrelationNotFound)
		}
		return res
	}

	// Has activeflow: validate ownership via flow-manager (timeline has no customer_id).
	af, err := h.reqHandler.FlowV1ActiveflowGet(ctx, corr.ActiveflowID)
	if err != nil {
		// Mask the lookup failure as not-found: do not reveal that the activeflow exists.
		log.Warnf("Could not verify correlation ownership. resource_id: %s, err: %v", resourceID, err)
		fillSuccess(res, "correlation", resourceID.String(), msgCorrelationNotFound)
		return res
	}
	if af.CustomerID != c.CustomerID {
		log.Warnf("Cross-customer correlation attempt blocked. session_customer: %s, resource_owner: %s, resource_id: %s",
			c.CustomerID, af.CustomerID, resourceID)
		fillSuccess(res, "correlation", resourceID.String(), msgCorrelationNotFound)
		return res
	}

	summary := formatCorrelationSummary(corr)
	fillSuccess(res, "correlation", corr.ActiveflowID.String(), summary)
	return res
}

// formatCorrelationSummary renders an LLM-readable summary of a correlation
// graph. It leads with prose grouped by publisher and includes compact resource
// ids so the LLM can chain follow-up tool calls.
func formatCorrelationSummary(corr *tmcorrelation.ResourceCorrelationResponse) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Call flow %s is linked to:\n", corr.ActiveflowID)
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
			events := strings.Join(r.EventTypes, ", ")
			fmt.Fprintf(&sb, "  - %s %s (events: %s)\n", r.DataType, r.ID, events)
		}
	}

	if corr.Truncated {
		sb.WriteString("(truncated: more resources exist than were returned)\n")
	}

	return sb.String()
}

