package aicallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
)

// ChatMessage sends/receives the messages from/to an ai
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

	var tmp *message.Message
	var err error
	switch cc.ReferenceType {
	case aicall.ReferenceTypeCall:
		tmp, err = h.messageHandler.StreamingSend(ctx, cc.ID, message.RoleSystem, initPrompt, true)
		if err != nil {
			return errors.Wrapf(err, "could not send the init prompt to the ai. aicall_id: %s", cc.ID)
		}

	default:
		tmp, err = h.messageHandler.Send(ctx, cc.ID, message.RoleSystem, initPrompt, true)
		if err != nil {
			return errors.Wrapf(err, "could not send the init prompt to the ai. aicall_id: %s", cc.ID)
		}
	}
	log.WithField("message", tmp).Debugf("Response message from the ai for init prompt. aicall_id: %s", cc.ID)

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
	if errStop := h.reqHandler.TTSV1StreamingSayStop(ctx, cc.TTSStreamingPodID, cc.TTSStreamingID); errStop != nil {
		log.Errorf("Could not stop the tts streaming. err: %v", errStop)
		return errors.Wrap(errStop, "Could not stop the tts streaming")
	}

	tmp, err := h.messageHandler.StreamingSend(ctx, cc.ID, message.RoleUser, content, true)
	if err != nil {
		return errors.Wrapf(err, "could not send the message to the ai. aicall_id: %s", cc.ID)
	}
	log.WithField("message", tmp).Debugf("Response message from the ai. aicall_id: %s", cc.ID)

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
