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

	switch tool.Function.Name {
	case message.FunctionCallNameConnect:
		return h.toolHandleConnect(ctx, c, tool)

	case message.FunctionCallNameMessageSend:
		return h.toolHandleMessageSend(ctx, c, tool)

	default:
		log.Debugf("unknown tool call: %s", tool.Function.Name)
		return nil, fmt.Errorf("unknown tool call: %s", tool.Function.Name)
	}
}

func (h *aicallHandler) toolHandleConnect(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) (map[string]any, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleConnect",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool call connect.")

	var tmpOpt fmaction.OptionConnect
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &tmpOpt); errUnmarshal != nil {
		return nil, errors.Wrapf(errUnmarshal, "could not unmarshal the tool option correctly")
	}

	opt := fmaction.ConvertOption(tmpOpt)
	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeConnect,
			Option: opt,
		},
	}

	result := "success"
	tmpContent := ""
	af, err := h.reqHandler.FlowV1ActiveflowAddActions(ctx, c.ActiveflowID, actions)
	if err != nil {
		result = "failed"
		tmpContent = err.Error()
	}
	log.WithField("activeflow", af).Debugf("Added actions to the activeflow. activeflow_id: %s", c.ActiveflowID)

	content := fmt.Sprintf(`{"result": "%s", "message": "%s"}`, result, tmpContent)
	tmp, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleTool, content, nil, tool.ID)
	if err != nil {
		log.Errorf("Could not create the tool response message correctly. err: %v", err)
		return nil, errors.Wrapf(err, "could not create the tool message")
	}
	log.WithField("message", tmp).Debugf("Created the tool response message. message_id: %s", tmp.ID)

	go func() {
		// this will connect the call right away
		tmp, err := h.reqHandler.AIV1AIcallTerminate(ctx, c.ID)
		if err != nil {
			log.Errorf("Could not terminate the aicall after sending the tool actions. err: %v", err)
			return
		}
		log.WithField("aicall", tmp).Debugf("Terminating the aicall after sending the tool actions. aicall_id: %s", c.ID)
	}()

	res := map[string]any{
		"result":  result,
		"message": tmpContent,
	}

	return res, nil
}

func (h *aicallHandler) toolHandleMessageSend(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) (map[string]any, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleMessageSend",
		"aicall_id": c.ID,
	})

	var tmpOpt fmaction.OptionMessageSend
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &tmpOpt); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the tool option correctly. err: %v", errUnmarshal)
		return nil, errors.Wrapf(errUnmarshal, "could not unmarshal the tool option correctly")
	}

	// send the message right away
	result := "success"
	tmpContent := ""
	msgID := h.utilHandler.UUIDCreate()
	tmpMessage, err := h.reqHandler.MessageV1MessageSend(ctx, msgID, c.CustomerID, tmpOpt.Source, tmpOpt.Destinations, tmpOpt.Text)
	if err != nil {
		log.WithField("tool_call", tool).Errorf("Could not send the message correctly. err: %v", err)
		result = "error"
		tmpContent = err.Error()
	}
	log.WithField("message", tmpMessage).Debugf("Message sent. result: %s, content: %s", result, tmpContent)

	content := fmt.Sprintf(`{"result": "%s", "message": "%s"}`, result, tmpContent)
	tmp, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleTool, content, nil, tool.ID)
	if err != nil {
		log.Errorf("Could not create the tool response message correctly. err: %v", err)
		return nil, errors.Wrapf(err, "could not create the tool message")
	}
	log.WithField("message", tmp).Debugf("Created the tool response message. message_id: %s", tmp.ID)

	res := map[string]any{
		"result":  result,
		"message": tmpContent,
	}

	return res, nil
}
