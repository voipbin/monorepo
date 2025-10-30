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
	a *ai.AI,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startNew",
		"ai":            a,
		"activeflow_id": activeflowID,
	})
	log.Debugf("Starting a new aicall")

	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, cmcustomer.IDAIManager, activeflowID, cmconfbridge.ReferenceTypeAI, a.ID, cmconfbridge.TypeConference)
	if err != nil {
		log.Errorf("Could not create confbridge. err: %v", err)
		return nil, errors.Wrap(err, "Could not create confbridge")
	}

	// generate pipecatcall id
	pipecatcallID := h.utilHandler.UUIDCreate()
	log.Debugf("Generated pipecatcall_id: %s", pipecatcallID)

	// create ai call
	res, err := h.Create(ctx, a, activeflowID, aicall.ReferenceTypeCall, referenceID, cb.ID, pipecatcallID, gender, language)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create aicall.")
	}
	log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)

	// set activeflow variables
	if errSet := h.setActiveflowVariables(ctx, res); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the activeflow variables for aicall. aicall_id: %s", res.ID)
	}
	log.Debugf("Set activeflow variables for aicall. aicall_id: %s", res.ID)

	// start initial messages
	if errInitMessages := h.startInitMessages(ctx, a, res); errInitMessages != nil {
		return nil, errors.Wrapf(errInitMessages, "could not start initial messages for aicall. aicall_id: %s", res.ID)
	}
	log.Debugf("Initialized messages for aicall. aicall_id: %s", res.ID)

	// start pipecatcall
	tmpPipecatcall, err := h.startPipecatcall(ctx, a, res)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", tmpPipecatcall).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	return res, nil
}

// startReferenceTypeConversation starts a new aicall with reference type conversation
func (h *aicallHandler) startReferenceTypeConversation(
	ctx context.Context,
	a *ai.AI,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeConversation",
		"ai":            a,
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
	})

	// get conversation message
	vars, err := h.reqHandler.FlowV1VariableGet(ctx, activeflowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the activeflow variables. activeflow_id: %s", activeflowID)
	}

	content, ok := vars.Variables["voipbin.conversation_message.text"]
	if !ok {
		return nil, errors.New("could not get the conversation message text from the activeflow variables")
	}

	// get existing aicall info
	res, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Debugf("Could not get the aicall by reference id. Creating a new aicall. reference_id: %s. err: %v", referenceID, err)
		res, err = h.startReferenceTypeConversationNew(ctx, a, activeflowID, referenceID, gender, language, content)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create aicall. activeflow_id: %s", activeflowID)
		}
	}
	log.WithField("aicall", res).Debugf("Found the aicall. aicall_id: %s", res.ID)

	tmp, err := h.startPipecatcall(ctx, a, res)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", tmp).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

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

	if errSet := h.setActiveflowVariables(ctx, tmp); errSet != nil {
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

	// retrieve previous messages
	tmpMessages, err := h.messageHandler.Gets(ctx, c.ID, 100, "", map[string]string{})
	if err != nil {
		return nil, errors.Wrap(err, "Could not get messages")
	}

	res := []map[string]any{}
	if len(tmpMessages) > 0 {
		// reverse the messages to have the correct order
		for i, j := 0, len(tmpMessages)-1; i < j; i, j = i+1, j-1 {
			tmpMessages[i], tmpMessages[j] = tmpMessages[j], tmpMessages[i]
		}

		for _, m := range tmpMessages {
			res = append(res, map[string]any{
				"role":    string(m.Role),
				"content": string(m.Content),
			})
		}
	}

	return res, nil
}

func (h *aicallHandler) getPipecatcallSTTType(sttType ai.STTType) pmpipecatcall.STTType {

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

func (h *aicallHandler) getPipecatcallTTSType(c *aicall.AIcall, ttsType ai.TTSType) pmpipecatcall.TTSType {
	if c.ReferenceType != aicall.ReferenceTypeCall {
		return pmpipecatcall.TTSType("")
	}

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

func (h *aicallHandler) startPipecatcall(ctx context.Context, a *ai.AI, c *aicall.AIcall) (*pmpipecatcall.Pipecatcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "startPipecatcall",
		"aicall_id": c.ID,
	})

	// get messages for pipecatcall
	messages, err := h.getPipecatcallMessages(ctx, a, c.ActiveflowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the messages for pipecatcall")
	}
	log.Debugf("Got %d messages for pipecatcall", len(messages))

	// get tts and stt types for pipecatcall
	ttsType := h.getPipecatcallTTSType(c, a.TTSType)
	ttsVoiceID, err := h.getPipecatcallVoiceID(a.TTSType, a.TTSVoiceID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the tts voice id for pipecatcall")
	}
	sttType := h.getPipecatcallSTTType(a.STTType)
	log.Debugf("Determined variables. sttType: %s, ttsType: %s, ttsVoiceID: %s for pipecatcall", sttType, ttsType, ttsVoiceID)

	res, err := h.reqHandler.PipecatV1PipecatcallStart(
		ctx,
		c.PipecatcallID,
		c.CustomerID,
		c.ActiveflowID,
		pmpipecatcall.ReferenceTypeAICall,
		c.ID,
		pmpipecatcall.LLMType(a.EngineModel),
		messages,
		sttType,
		ttsType,
		ttsVoiceID,
	)
	if err != nil {
		log.Errorf("Could not start pipecatcall. err: %v", err)
		return nil, errors.Wrap(err, "could not start pipecatcall")
	}
	log.WithField("pipecatcall", res).Debugf("Started pipecatcall. pipecatcall_id: %s", res.ID)

	return res, nil
}

func (h *aicallHandler) startReferenceTypeConversationNew(
	ctx context.Context,
	a *ai.AI,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
	messageText string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeConversationNew",
		"ai":            a,
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
	})

	pipecatcallID := h.utilHandler.UUIDCreate()
	res, err := h.Create(ctx, a, activeflowID, aicall.ReferenceTypeConversation, referenceID, uuid.Nil, pipecatcallID, gender, language)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create aicall. activeflow_id: %s", activeflowID)
	}

	if errInitMessages := h.startInitMessages(ctx, a, res); errInitMessages != nil {
		return nil, errors.Wrapf(errInitMessages, "could not start initial messages for aicall. aicall_id: %s", res.ID)
	}
	log.Debugf("Initialized messages for aicall. aicall_id: %s", res.ID)

	m, err := h.messageHandler.Create(ctx, a.CustomerID, res.ID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the message to the ai. aicall_id: %s", res.ID)
	}
	log.WithField("message", m).Debugf("Created the start message to the ai. aicall_id: %s", res.ID)

	tmp, err := h.startPipecatcall(ctx, a, res)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", tmp).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	return res, nil
}

func (h *aicallHandler) startReferenceTypeConversationResume(
	ctx context.Context,
	a *ai.AI,
	c *aicall.AIcall,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	gender aicall.Gender,
	language string,
	messageText string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeConversationNew",
		"ai":            a,
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
	})

	// send pipecat message

	tmp, err := h.startPipecatcall(ctx, a, res)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", tmp).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	pipecatcallID := h.utilHandler.UUIDCreate()
	res, err := h.Create(ctx, a, activeflowID, aicall.ReferenceTypeConversation, referenceID, uuid.Nil, pipecatcallID, gender, language)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create aicall. activeflow_id: %s", activeflowID)
	}

	if errInitMessages := h.startInitMessages(ctx, a, res); errInitMessages != nil {
		return nil, errors.Wrapf(errInitMessages, "could not start initial messages for aicall. aicall_id: %s", res.ID)
	}
	log.Debugf("Initialized messages for aicall. aicall_id: %s", res.ID)

	m, err := h.messageHandler.Create(ctx, a.CustomerID, res.ID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the message to the ai. aicall_id: %s", res.ID)
	}
	log.WithField("message", m).Debugf("Created the start message to the ai. aicall_id: %s", res.ID)

	tmp, err := h.startPipecatcall(ctx, a, res)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", tmp).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	return res, nil
}

func (h *aicallHandler) startInitMessages(ctx context.Context, a *ai.AI, c *aicall.AIcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "startInitMessages",
		"aicall_id": c.ID,
	})

	initPromptMessage := h.chatGetInitPrompt(ctx, a, c.ActiveflowID)
	messages := []string{
		defaultCommonSystemPrompt, // default system prompt for all calls
		initPromptMessage,         // ai specific init prompt
	}

	for _, msg := range messages {
		tmp, err := h.messageHandler.Create(ctx, a.CustomerID, c.ID, message.DirectionOutgoing, message.RoleSystem, msg, nil, "")
		if err != nil {
			return errors.Wrapf(err, "could not create the init message to the ai. aicall_id: %s", c.ID)
		}
		log.WithField("message", tmp).Debugf("Created the init message to the ai. aicall_id: %s", c.ID)
	}

	return nil
}
