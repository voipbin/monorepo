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
	"time"

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

	// start ai call
	res, err := h.startAIcall(
		ctx,
		a,
		activeflowID,
		aicall.ReferenceTypeCall,
		referenceID,
		cb.ID,
		gender,
		language,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create aicall. activeflow_id: %s", activeflowID)
	}
	log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)

	// start pipecatcall
	tmpPipecatcall, err := h.startPipecatcall(ctx, res)
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

	messageText, ok := vars.Variables["voipbin.conversation_message.text"]
	if !ok {
		return nil, errors.New("could not get the conversation message text from the activeflow variables")
	}

	// get existing aicall info
	res, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Debugf("Could not get the aicall by reference id. Start a new aicall. reference_id: %s. err: %v", referenceID, err)
		res, err = h.startAIcall(ctx, a, activeflowID, aicall.ReferenceTypeConversation, referenceID, uuid.Nil, gender, language)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create aicall. activeflow_id: %s", activeflowID)
		}
	} else {
		// update the pipecatcall id to a new one
		newPipecatcallID := h.utilHandler.UUIDCreate()
		tmp, errUpdate := h.UpdatePipecatcallID(ctx, res.ID, newPipecatcallID)
		if errUpdate != nil {
			return nil, errors.Wrapf(errUpdate, "could not update the pipecatcall id for existing aicall. aicall_id: %s", res.ID)
		}
		res = tmp
	}
	log.WithField("aicall", res).Debugf("Found the aicall. aicall_id: %s", res.ID)

	// note: after create a new aicall, we need to create a new message for the conversation message
	tmp, err := h.messageHandler.Create(ctx, res.CustomerID, res.ID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the message. aicall_id: %s", res.ID)
	}
	log.WithField("message", tmp).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", res.ID, res.ID)

	pc, err := h.startPipecatcall(ctx, res)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", pc).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	go func() {
		time.Sleep(defaultPipecatcallTimeout)
		tmpPipecatcall, err := h.reqHandler.PipecatV1PipecatcallTerminate(context.Background(), pc.HostID, pc.ID)
		if err != nil {
			log.Errorf("Could not terminate the pipecatcall correctly. err: %v", err)
			return
		}
		log.WithField("pipecatcall_terminate", tmpPipecatcall).Debugf("Terminated the pipecatcall correctly.")
	}()

	return res, nil
}

// startReferenceTypeNone starts a new aicall with no reference
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

	// start ai call
	tmp, err := h.startAIcall(
		ctx,
		c,
		uuid.Nil,
		aicall.ReferenceTypeNone,
		uuid.Nil,
		uuid.Nil,
		gender,
		language,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create aicall with no reference")
	}
	log.WithField("aicall", tmp).Debugf("Created aicall. aicall_id: %s", tmp.ID)

	res, err := h.UpdateStatus(ctx, tmp.ID, aicall.StatusProgressing)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the status to start. aicall_id: %s", tmp.ID)
	}

	return res, nil
}

func (h *aicallHandler) getPipecatcallMessages(ctx context.Context, c *aicall.AIcall) ([]map[string]any, error) {

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

func (h *aicallHandler) getPipecatcallSTTType(c *aicall.AIcall) pmpipecatcall.STTType {
	if c.AISTTType != ai.STTTypeNone {
		return pmpipecatcall.STTType(c.AISTTType)
	}

	return defaultPipecatcallSTTType
}

func (h *aicallHandler) getPipecatcallTTSInfo(a *aicall.AIcall) (pmpipecatcall.TTSType, string) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "getPipecatcallTTSInfo",
		"aicall_id": a.ID,
	})

	// get tts type
	ttsType := defaultPipecatcallTTSType
	if a.AITTSType != ai.TTSTypeNone {
		ttsType = pmpipecatcall.TTSType(a.AITTSType)
	}

	// get voiceID
	ttsVoiceID, ok := mapDefaultTTSVoiceIDByTTSType[ai.TTSType(ttsType)]
	if !ok {
		log.Warnf("No default TTS voice ID found for TTSType: %v", ttsType)
		ttsVoiceID = ""
	}

	if a.AITTSVoiceID != "" {
		ttsVoiceID = a.AITTSVoiceID
	}

	return ttsType, ttsVoiceID
}

func (h *aicallHandler) startPipecatcall(ctx context.Context, c *aicall.AIcall) (*pmpipecatcall.Pipecatcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "startPipecatcall",
		"aicall_id": c.ID,
	})

	// get llmMessages for pipecatcall
	llmType := pmpipecatcall.LLMType(c.AIEngineModel)
	llmMessages, err := h.getPipecatcallMessages(ctx, c)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the messages for pipecatcall")
	}
	log.Debugf("Got %d messages for pipecatcall", len(llmMessages))

	// determine stt and tts info
	ttsType := pmpipecatcall.TTSTypeNone
	ttsVoiceID := ""
	sttType := pmpipecatcall.STTTypeNone
	if c.ReferenceType == aicall.ReferenceTypeCall {
		log.Debugf("The aicall reference type is call. Getting tts and stt types for pipecatcall")
		ttsType, ttsVoiceID = h.getPipecatcallTTSInfo(c)
		sttType = h.getPipecatcallSTTType(c)
	}
	log.Debugf("Determined variables. sttType: %s, ttsType: %s, ttsVoiceID: %s for pipecatcall", sttType, ttsType, ttsVoiceID)

	res, err := h.reqHandler.PipecatV1PipecatcallStart(
		ctx,
		c.PipecatcallID,
		c.CustomerID,
		c.ActiveflowID,
		pmpipecatcall.ReferenceTypeAICall,
		c.ID,
		llmType,
		llmMessages,
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

func (h *aicallHandler) startInitMessages(ctx context.Context, a *ai.AI, c *aicall.AIcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "startInitMessages",
		"aicall_id": c.ID,
	})

	initPromptMessage := h.getInitPrompt(ctx, a, c.ActiveflowID)
	messages := []string{
		defaultCommonSystemPrompt, // default system prompt for all calls
		initPromptMessage,         // ai specific init prompt
	}

	for _, msg := range messages {
		tmp, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleSystem, msg, nil, "")
		if err != nil {
			return errors.Wrapf(err, "could not create the init message to the ai. aicall_id: %s", c.ID)
		}
		log.WithField("message", tmp).Debugf("Created the init message to the ai. aicall_id: %s", c.ID)
	}

	return nil
}

func (h *aicallHandler) startAIcall(
	ctx context.Context,
	a *ai.AI,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	confbridgeID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startAIcall",
		"ai_id":         a.ID,
		"activeflow_id": activeflowID,
	})

	// create ai call
	pipecatcallID := h.utilHandler.UUIDCreate()
	res, err := h.Create(ctx, a, activeflowID, referenceType, referenceID, confbridgeID, pipecatcallID, gender, language)
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

	return res, nil
}
