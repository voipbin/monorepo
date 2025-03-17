package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
)

// ChatMessage sends/receives the messages from/to a ai
func (h *aicallHandler) ChatMessage(ctx context.Context, cc *aicall.AIcall, text string) error {
	switch cc.ReferenceType {
	case aicall.ReferenceTypeCall:
		if errChat := h.chatMessageReferenceTypeCall(ctx, cc, text); errChat != nil {
			return errors.Wrap(errChat, "could not handle the chat message")
		}
		return nil

	default:
		return fmt.Errorf("unsupported reference type. reference_type: %s", cc.ReferenceType)
	}
}

// chatMessageActionsHandle handles chat message actions
func (h *aicallHandler) chatMessageActionsHandle(ctx context.Context, cc *aicall.AIcall, actions []fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "chatMessageActionsHandle",
		"aicall":  cc,
		"actions": actions,
	})

	// push the actions
	af, err := h.reqHandler.FlowV1ActiveflowPushActions(ctx, cc.ActiveflowID, actions)
	if err != nil {
		log.Errorf("Could not push the actions to the activeflow. err: %v", err)
		return errors.Wrap(err, "could not push the actions to the activeflow")
	}
	log.WithField("activeflow", af).Debugf("Pushed actions to the activeflow. activeflow_id: %s", af.ID)

	// destroy the confbridge
	tmp, err := h.reqHandler.CallV1ConfbridgeTerminate(ctx, cc.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not terminate the aicall confbridge. err: %v", err)
		return errors.Wrap(err, "could not terminate the aicall confbridge")
	}
	log.WithField("confbridge", tmp).Debugf("Terminated confbridge. confbridge_id: %s", tmp.ID)

	return nil
}

// chatMessageTextHandle handles chat message text
func (h *aicallHandler) chatMessageTextHandle(ctx context.Context, cc *aicall.AIcall, text string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":   "chatMessageTextHandle",
		"aicall": cc,
		"text":   text,
	})

	if errTalk := h.reqHandler.CallV1CallTalk(ctx, cc.ReferenceID, text, string(cc.Gender), cc.Language, 10000); errTalk != nil {
		log.Errorf("Could not talk to the call. err: %v", errTalk)
		return errors.Wrap(errTalk, "could not talk to the call")
	}

	return nil
}

// chatInit sends the chat's init_prompt
func (h *aicallHandler) chatInit(ctx context.Context, cb *ai.AI, cc *aicall.AIcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "chatInit",
		"ai_id":     cb.ID,
		"aicall_id": cc.ID,
	})

	if errSet := h.chatSetVariables(ctx, cc); errSet != nil {
		// we couldn't set the variables, but we can continue
		log.Errorf("Could not set the variables. err: %v", errSet)
	}

	initPrompt := h.chatGetInitPrompt(ctx, cb, cc)
	if initPrompt == "" {
		// has no init prompt. nothing todo
		return nil
	}

	tmp, err := h.reqHandler.AIV1MessageSend(ctx, cc.ID, message.RoleSystem, initPrompt, 30000)
	if err != nil {
		return errors.Wrapf(err, "could not send the init prompt to the ai. aicall_id: %s", cc.ID)
	}

	if errHandle := h.chatMessageHandle(ctx, cc, tmp); errHandle != nil {
		return errors.Wrap(errHandle, "could not handle the chat message")
	}

	return nil
}

func (h *aicallHandler) chatMessageReferenceTypeCall(ctx context.Context, cc *aicall.AIcall, content string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "chatMessageReferenceTypeCall",
		"aicall_id": cc.ID,
		"content":   content,
	})

	// currently only the reference type call supported
	if cc.ReferenceType != aicall.ReferenceTypeCall {
		log.Errorf("Unsupported reference type. reference_type: %s", cc.ReferenceType)
		return fmt.Errorf("unsupported referencd type")
	}

	// stop the media because chat will talk soon
	if errStop := h.reqHandler.CallV1CallMediaStop(ctx, cc.ReferenceID); errStop != nil {
		log.Errorf("Could not stop the media. err: %v", errStop)
		return errors.Wrap(errStop, "Could not stop the media")
	}

	tmp, err := h.reqHandler.AIV1MessageSend(ctx, cc.ID, message.RoleUser, content, 30000)
	if err != nil {
		return errors.Wrapf(err, "could not send the message to the ai. aicall_id: %s", cc.ID)
	}
	log.WithField("message_id", tmp.ID).Debugf("Sent the message to the ai. aicall_id: %s", cc.ID)

	if errHandle := h.chatMessageHandle(ctx, cc, tmp); errHandle != nil {
		return errors.Wrap(errHandle, "could not handle the chat message")
	}

	return nil
}

func (h *aicallHandler) chatMessageHandle(ctx context.Context, cc *aicall.AIcall, m *message.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "chatMessageHandle",
		"aicall_id": cc.ID,
	})

	if m == nil || len(m.Content) == 0 {
		log.Debugf("Received response with empty content")
		return nil
	}

	switch cc.ReferenceType {
	case aicall.ReferenceTypeCall:
		return h.chatMessageHandleReferenceTypeCall(ctx, cc, m)

	case aicall.ReferenceTypeNone:
		// nothing todo
		return nil

	default:
		return fmt.Errorf("unsupported reference type. reference_type: %s", cc.ReferenceType)

	}
}

func (h *aicallHandler) chatMessageHandleReferenceTypeCall(ctx context.Context, cc *aicall.AIcall, m *message.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "chatMessageHandleReferenceTypeCall",
		"aicall_id": cc.ID,
	})

	// check the response message
	tmpActions := []fmaction.Action{}
	errUnmarshal := json.Unmarshal([]byte(m.Content), &tmpActions)
	if errUnmarshal == nil {
		log.WithField("actions", tmpActions).Debugf("Got a action arrays. len_actions: %d", len(tmpActions))
		if errHandle := h.chatMessageActionsHandle(ctx, cc, tmpActions); errHandle != nil {
			log.Errorf("Could not handle the response actions correctly. err: %v", errHandle)
			return errors.Wrap(errHandle, "could not handle the response actions correctly")
		}
	} else {
		log.WithField("text", m.Content).Debugf("Got an message text. text: %s", m.Content)
		if errHandle := h.chatMessageTextHandle(ctx, cc, m.Content); errHandle != nil {
			log.Errorf("Could not handle the response message text correctly. err: %v", errHandle)
			return errors.Wrap(errHandle, "could not handle the response message text correctly")
		}
	}

	return nil
}

func (h *aicallHandler) chatSetVariables(ctx context.Context, cc *aicall.AIcall) error {
	if cc.ActiveflowID == uuid.Nil {
		// nothing todo
		return nil
	}

	variables := map[string]string{
		variableAIcallID:      cc.ID.String(),
		variableAIID:          cc.AIID.String(),
		variableAIEngineModel: string(cc.AIEngineModel),
		variableConfbridgeID:  cc.ConfbridgeID.String(),
		variableGender:        string(cc.Gender),
		variableLanguage:      cc.Language,
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, cc.ActiveflowID, variables); errSet != nil {
		return errors.Wrap(errSet, "could not set the variables")
	}
	return nil
}

func (h *aicallHandler) chatGetInitPrompt(ctx context.Context, cb *ai.AI, cc *aicall.AIcall) string {
	log := logrus.WithFields(logrus.Fields{
		"func":      "chatGetInitPrompt",
		"ai_id":     cb.ID,
		"aicall_id": cc.ID,
	})

	res := cb.InitPrompt
	if cc.ActiveflowID != uuid.Nil {
		tmp, err := h.reqHandler.FlowV1VariableSubstitute(ctx, cc.ActiveflowID, cb.InitPrompt)
		if err != nil {
			log.Errorf("Could not substitute the init prompt. err: %v", err)
			return res
		} else {
			res = tmp
		}
	}

	return res
}
