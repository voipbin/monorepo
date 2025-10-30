package aicallhandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *aicallHandler) Start(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*aicall.AIcall, error) {

	c, err := h.aiHandler.Get(ctx, aiID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get ai info")
	}

	switch referenceType {
	case aicall.ReferenceTypeCall:
		return h.startReferenceTypeCall(ctx, c, activeflowID, referenceID, gender, language)

	case aicall.ReferenceTypeConversation:
		return h.startReferenceTypeConversation(ctx, c, activeflowID, referenceID, gender, language)

	case aicall.ReferenceTypeNone:
		return h.startReferenceTypeNone(ctx, c, gender, language)

	default:
		return nil, fmt.Errorf("unsupported reference type")
	}
}

// // startResume starts an aicall with an existing aicall.
// // It is used to continue a previously interrupted or paused session.
// func (h *aicallHandler) startResume(ctx context.Context, activeflowID uuid.UUID) (*aicall.AIcall, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":          "startResume",
// 		"activeflow_id": activeflowID,
// 	})

// 	// aicall get by activeflow id
// 	filters := map[string]string{
// 		"activeflow_id": activeflowID.String(),
// 		"deleted":       "false",
// 	}

// 	tmps, err := h.Gets(ctx, 1, "", filters)
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "could not get the aicall info. activeflow_id: %s", activeflowID)
// 	} else if len(tmps) == 0 {
// 		return nil, errors.New("could not get the aicall info. activeflow_id: %s")
// 	}
// 	cc := tmps[0]
// 	log.WithField("aicall", cc).Debugf("Found the aicall. aicall_id: %s", cc.ID)

// 	if cc.ReferenceType != aicall.ReferenceTypeCall {
// 		return nil, errors.New("could not resume the aicall. reference type is not call")
// 	}

// 	// cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, cmcustomer.IDAIManager, activeflowID, cmconfbridge.ReferenceTypeAI, cc.ID, cmconfbridge.TypeConference)
// 	// if err != nil {
// 	// 	log.Errorf("Could not create confbridge. err: %v", err)
// 	// 	return nil, errors.Wrap(err, "Could not create confbridge")
// 	// }

// 	// // update aicall's confbridge info
// 	// res, err := h.UpdateStatusResuming(ctx, cc.ID, cb.ID)
// 	// if err != nil {
// 	// 	return nil, errors.Wrapf(err, "could not update the status to resuming. aicall_id: %s", cc.ID)
// 	// }

// 	return res, nil
// }

// startReferenceTypeCall starts a new aicall with reference type call
func (h *aicallHandler) startReferenceTypeCall(
	ctx context.Context,
	c *ai.AI,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startNew",
		"ai":            c,
		"activeflow_id": activeflowID,
	})
	log.Debugf("Starting a new aicall")

	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, cmcustomer.IDAIManager, activeflowID, cmconfbridge.ReferenceTypeAI, c.ID, cmconfbridge.TypeConference)
	if err != nil {
		log.Errorf("Could not create confbridge. err: %v", err)
		return nil, errors.Wrap(err, "Could not create confbridge")
	}

	// create ai call
	pipecatcallID := h.utilHandler.UUIDCreate()
	res, err := h.Create(ctx, c, activeflowID, aicall.ReferenceTypeCall, referenceID, cb.ID, pipecatcallID, gender, language)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create aicall.")
	}
	log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)

	if errSet := h.setVariables(ctx, res); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the variables for aicall. aicall_id: %s", res.ID)
	}

	messages, err := h.getPipecatcallMessages(ctx, c, activeflowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the messages for pipecatcall")
	}

	ttsType := h.getPipecatcallTTSType(c.TTSType)
	ttsVoiceID, err := h.getPipecatcallVoiceID(c.TTSType, c.TTSVoiceID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the tts voice id for pipecatcall")
	}
	sttType := h.getSTTType(c.STTType)

	pc, err := h.reqHandler.PipecatV1PipecatcallStart(
		ctx,
		res.PipecatcallID,
		c.CustomerID,
		activeflowID,
		pmpipecatcall.ReferenceTypeAICall,
		res.ID,
		pmpipecatcall.LLMType(c.EngineModel),
		messages,
		sttType,
		ttsType,
		ttsVoiceID,
	)
	if err != nil {
		log.Errorf("Could not start pipecatcall. err: %v", err)
		return nil, errors.Wrap(err, "could not start pipecatcall")
	}
	log.WithField("pipecatcall", pc).Debugf("Started pipecatcall. pipecatcall_id: %s", pc.ID)

	// update pipecatcall info
	h.UpdatePipecatcallID(ctx, res.ID, pc.ID)

	return res, nil
}

// startReferenceTypeConversation starts a new aicall with reference type conversation
func (h *aicallHandler) startReferenceTypeConversation(
	ctx context.Context,
	c *ai.AI,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeConversation",
		"ai":            c,
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
	})

	// get existing aicall info
	res, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		// aicall not found, create a new one
		pipecatcallID := h.utilHandler.UUIDCreate()
		res, err = h.Create(ctx, c, activeflowID, aicall.ReferenceTypeConversation, referenceID, uuid.Nil, pipecatcallID, gender, language)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create aicall. activeflow_id: %s", activeflowID)
		}
	}
	log.WithField("aicall", res).Debugf("Found the aicall. aicall_id: %s", res.ID)

	// get conversation message
	vars, err := h.reqHandler.FlowV1VariableGet(ctx, activeflowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the activeflow variables. activeflow_id: %s", activeflowID)
	}

	content, ok := vars.Variables["voipbin.conversation_message.text"]
	if !ok {
		return nil, errors.New("could not get the conversation message text from the activeflow variables")
	}

	// send the message
	m, err := h.messageHandler.Send(ctx, res.ID, message.RoleUser, content, false)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message to the ai. aicall_id: %s", res.ID)
	}
	log.WithField("message", m).Debugf("Sent the message to the ai. aicall_id: %s", res.ID)

	return res, nil
}

// startReferenceTypeNone starts a new aicall with no reference
// this is used to test the aicall
func (h *aicallHandler) startReferenceTypeNone(
	ctx context.Context,
	c *ai.AI,
	gender aicall.Gender,
	language string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "startReferenceTypeNone",
		"ai":   c,
	})

	pipecatcallID := h.utilHandler.UUIDCreate()
	tmp, err := h.Create(ctx, c, uuid.Nil, aicall.ReferenceTypeNone, uuid.Nil, uuid.Nil, pipecatcallID, gender, language)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create aicall.")
	}
	log.WithField("aicall", tmp).Debugf("Created aicall. aicall_id: %s", tmp.ID)

	if errSet := h.setVariables(ctx, tmp); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the variables for aicall. aicall_id: %s", tmp.ID)
	}

	res, err := h.UpdateStatus(ctx, tmp.ID, aicall.StatusProgressing)
	if err != nil {
		log.Errorf("Could not update the status to start. err: %v", err)
		return nil, errors.Wrapf(err, "Could not update the status to start. aicall_id: %s", tmp.ID)
	}

	return res, nil
}

func (h *aicallHandler) getPipecatcallMessages(ctx context.Context, c *ai.AI, activeflowID uuid.UUID) ([]map[string]any, error) {
	res := []map[string]any{
		{
			"role":    "system",
			"content": defaultCommonSystemPrompt,
		},
	}

	initPromptMessage := h.chatGetInitPrompt(ctx, c, activeflowID)
	initPrompt := map[string]any{
		"role":    "system",
		"content": initPromptMessage,
	}
	res = append(res, initPrompt)

	// retrieve previous messages
	tmpMessages, err := h.messageHandler.Gets(ctx, c.ID, 100, "", map[string]string{})
	if err != nil {
		return nil, errors.Wrap(err, "Could not get messages")
	}
	if len(tmpMessages) > 0 {
		// reverse the messages to have the correct order
		for i, j := 0, len(tmpMessages)-1; i < j; i, j = i+1, j-1 {
			tmpMessages[i], tmpMessages[j] = tmpMessages[j], tmpMessages[i]
		}

		for _, m := range tmpMessages {
			res = append(res, map[string]any{
				"role":    m.Role,
				"content": m.Content,
			})
		}
	}

	return res, nil
}

func (h *aicallHandler) getSTTType(sttType ai.STTType) pmpipecatcall.STTType {

	if sttType != ai.STTTypeNone {
		return pmpipecatcall.STTType(sttType)
	}

	return pmpipecatcall.STTType(defaultSTTType)
}

func (h *aicallHandler) getTTSType(ttsType ai.TTSType) ai.TTSType {
	if ttsType != ai.TTSTypeNone {
		return ttsType
	}

	return defaultTTSType
}

func (h *aicallHandler) getPipecatcallTTSType(ttsType ai.TTSType) pmpipecatcall.TTSType {
	tmp := h.getTTSType(ttsType)

	return pmpipecatcall.TTSType(tmp)
}

func (h *aicallHandler) getPipecatcallVoiceID(ttsType ai.TTSType, voiceID string) (string, error) {
	if voiceID != "" {
		return voiceID, nil
	}

	// note: we need to do this because the ttsType could be none
	tmpTTSType := h.getTTSType(ttsType)

	res, ok := mapDefaultTTSVoiceIDByTTSType[tmpTTSType]
	if !ok {
		return "", fmt.Errorf("unsupported tts type: %s", ttsType)
	}

	return res, nil
}
