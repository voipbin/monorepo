package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	fmaction "monorepo/bin-flow-manager/models/action"

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
	tmp, errCreate := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionIncoming, message.RoleAssistant, "", []message.ToolCall{*tool}, "")
	if errCreate != nil {
		return nil, errors.Wrapf(errCreate, "could not create the tool message")
	}
	log.WithField("message", tmp).Debugf("Created the tool message for the actions. message_id: %s", tmp.ID)

	var tmpMessageContent *messageContent

	mapFunctions := map[message.FunctionCallName]func(context.Context, *aicall.AIcall, *message.ToolCall) *messageContent{
		message.FunctionCallNameConnect:     h.toolHandleConnect,
		message.FunctionCallNameEmailSend:   h.toolHandleEmailSend,
		message.FunctionCallNameMediaStop:   h.toolHandleMediaStop,
		message.FunctionCallNameMessageSend: h.toolHandleMessageSend,
		message.FunctionCallNameServiceStop: h.toolHandleServiceStop,
		message.FunctionCallNameStop:        h.toolHandleStop,
	}

	if fn, exists := mapFunctions[tool.Function.Name]; exists {
		tmpMessageContent = fn(ctx, c, tool)
	} else {
		log.Debugf("unknown tool call: %s", tool.Function.Name)
		return nil, fmt.Errorf("unknown tool call: %s", tool.Function.Name)
	}

	msg, err := h.toolCreateResultMessage(ctx, c, tool, tmpMessageContent)
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
) (*message.Message, error) {

	content, err := json.Marshal(tmpContent)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the tool result content")
	}

	tmp, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleTool, string(content), nil, tool.ID)
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
	log.Debugf("handling tool stop.")

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
