package messagehandler

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

func (h *messageHandler) toolMessageHandle(ctx context.Context, cc *aicall.AIcall, toolCall *message.ToolCall) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolMessageHandle",
		"tool_call": toolCall,
	})

	switch toolCall.Function.Name {
	case string(fmaction.TypeConnect):
		return h.toolMessageHandleConnect(ctx, cc, toolCall)

	case string(fmaction.TypeMessageSend):
		return h.toolMessageHandleMessageSend(ctx, cc, toolCall)

	default:
		log.WithField("tool_call", toolCall).Warnf("Unsupported action type received: %s", toolCall.Function.Name)
		return false, fmt.Errorf("unsupported action type: %s", toolCall.Function.Name)
	}
}

// toolMessageHandleConnect
// returns terminate, sendResponse, error
func (h *messageHandler) toolMessageHandleConnect(ctx context.Context, cc *aicall.AIcall, toolCall *message.ToolCall) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolMessageHandleConnect",
		"tool_call": toolCall,
	})

	var tmpOpt fmaction.OptionConnect
	if errUnmarshal := json.Unmarshal([]byte(toolCall.Function.Arguments), &tmpOpt); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the tool option correctly. err: %v", errUnmarshal)
		return false, errors.Wrapf(errUnmarshal, "could not unmarshal the tool option correctly")
	}

	opt := fmaction.ConvertOption(tmpOpt)
	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeConnect,
			Option: opt,
		},
	}

	af, err := h.reqHandler.FlowV1ActiveflowAddActions(ctx, cc.ActiveflowID, actions)
	if err != nil {
		log.WithField("actions", actions).Errorf("Could not add actions to the activeflow. err: %v", err)
		return false, errors.Wrapf(err, "could not add actions to the activeflow. activeflow_id: %s", cc.ActiveflowID)
	}
	log.WithField("activeflow", af).Debugf("Added actions to the activeflow. activeflow_id: %s", cc.ActiveflowID)

	tmp, err := h.Create(ctx, uuid.Nil, cc.CustomerID, cc.ID, message.DirectionOutgoing, message.RoleTool, `{"result": "success"}`, nil, toolCall.ID)
	if err != nil {
		log.Errorf("Could not create the tool response message correctly. err: %v", err)
		return false, errors.Wrapf(err, "could not create the tool message")
	}
	log.WithField("message", tmp).Debugf("Created the tool response message. message_id: %s", tmp.ID)

	return true, nil
}

// toolMessageHandleMessageSend
// returns terminate, sendResponse, error
func (h *messageHandler) toolMessageHandleMessageSend(ctx context.Context, cc *aicall.AIcall, toolCall *message.ToolCall) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolMessageHandleMessageSend",
		"tool_call": toolCall,
	})

	var tmpOpt fmaction.OptionMessageSend
	if errUnmarshal := json.Unmarshal([]byte(toolCall.Function.Arguments), &tmpOpt); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the tool option correctly. err: %v", errUnmarshal)
		return false, errors.Wrapf(errUnmarshal, "could not unmarshal the tool option correctly")
	}

	// send the message right away
	result := "success"
	tmpContent := ""
	tmpMessage, err := h.reqHandler.MessageV1MessageSend(ctx, uuid.Nil, cc.CustomerID, tmpOpt.Source, tmpOpt.Destinations, tmpOpt.Text)
	if err != nil {
		log.WithField("tool_call", toolCall).Errorf("Could not send the message correctly. err: %v", err)
		result = "error"
		tmpContent = "{}"
	} else {
		log.WithField("message", tmpMessage).Infof("Sent the message. message_id: %s", tmpMessage.ID)
		tmpRes, errUnmarshal := json.Marshal(tmpMessage)
		if errUnmarshal != nil {
			log.Errorf("Could not marshal the sent message correctly. err: %v", errUnmarshal)
			tmpContent = "{}"
		} else {
			tmpContent = string(tmpRes)
		}
	}
	log.Debugf("Message sent. result: %s, content: %s", result, tmpContent)

	content := fmt.Sprintf(`{"result": "%s", "message": %s}`, result, tmpContent)
	tmp, err := h.Create(ctx, uuid.Nil, cc.CustomerID, cc.ID, message.DirectionOutgoing, message.RoleTool, content, nil, toolCall.ID)
	if err != nil {
		log.Errorf("Could not create the tool response message correctly. err: %v", err)
		return false, errors.Wrapf(err, "could not create the tool message")
	}
	log.WithField("message", tmp).Debugf("Created the tool response message. message_id: %s", tmp.ID)

	return false, nil
}
