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
	tmstreaming "monorepo/bin-tts-manager/models/streaming"

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
	resume bool,
) (*aicall.AIcall, error) {

	if resume {
		return h.startResume(ctx, activeflowID)
	}

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

// startResume starts an aicall with an existing aicall.
// It is used to continue a previously interrupted or paused session.
func (h *aicallHandler) startResume(ctx context.Context, activeflowID uuid.UUID) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startResume",
		"activeflow_id": activeflowID,
	})

	// aicall get by activeflow id
	filters := map[string]string{
		"activeflow_id": activeflowID.String(),
		"deleted":       "false",
	}

	tmps, err := h.Gets(ctx, 1, "", filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall info. activeflow_id: %s", activeflowID)
	} else if len(tmps) == 0 {
		return nil, errors.New("could not get the aicall info. activeflow_id: %s")
	}
	cc := tmps[0]
	log.WithField("aicall", cc).Debugf("Found the aicall. aicall_id: %s", cc.ID)

	if cc.ReferenceType != aicall.ReferenceTypeCall {
		return nil, errors.New("could not resume the aicall. reference type is not call")
	}

	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, cmcustomer.IDAIManager, activeflowID, cmconfbridge.ReferenceTypeAI, cc.ID, cmconfbridge.TypeConference)
	if err != nil {
		log.Errorf("Could not create confbridge. err: %v", err)
		return nil, errors.Wrap(err, "Could not create confbridge")
	}

	// update aicall's confbridge info
	res, err := h.UpdateStatusResuming(ctx, cc.ID, cb.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the status to resuming. aicall_id: %s", cc.ID)
	}

	return res, nil
}

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

	// get messages
	messages := []map[string]any{
		{
			"role": "system",
			"content": `
Role:
You are an AI assistant integrated with voipbin. 
Your role is to follow the user's system or custom prompt strictly, provide natural responses, and call external tools when necessary.

Context:
- Users will set their own instructions (persona, style, context).
- You must adapt to those instructions consistently.
- If user requests or situation requires, use available tools to gather data or perform actions.

Input Values:
- User-provided system/custom prompt
- User query
- Available tools list

Instructions:
- Always prioritize the user's provided prompt instructions.
- Generate a helpful, coherent, and contextually appropriate response.
- If tools are available and required, call them responsibly and return results clearly.
- **Do not mention tool names or the fact that a tool is being used in the user-facing response.**
- Maintain consistency with the user-defined tone and role.
- If ambiguity exists, ask clarifying questions before answering.
- Before giving the final answer, outline a short execution plan (2–4 steps), then provide a concise summary (1–2 sentences) and the final answer.  
- For each Input Value, ask clarifying questions **one at a time in sequence**. Wait for the user's answer before moving to the next question.  

Constraints:
- Avoid hallucination; use tools for factual queries.  
- Keep answers aligned with user's persona and tone.  
- Respect conversation history and continuity.  
	`,
		},
		{
			"role":    "system",
			"content": c.EngineData,
		},
	}
	// messagestt, err := h.messageHandler.Gets(ctx, c.ID, 100, "", map[string]string{})
	// if err != nil {
	// 	return nil, errors.Wrap(err, "Could not get messages")
	// }

	pc, err := h.reqHandler.PipecatV1PipecatcallStart(
		ctx,
		c.CustomerID,
		activeflowID,
		pmpipecatcall.ReferenceTypeCall,
		referenceID,
		pmpipecatcall.LLM(c.EngineModel),
		pmpipecatcall.STTDeepgram,
		pmpipecatcall.TTSCartesia,
		"71a7ad14-091c-4e8e-a314-022ece01c121",
		messages,
	)
	if err != nil {
		log.Errorf("Could not start pipecatcall. err: %v", err)
		return nil, errors.Wrap(err, "could not start pipecatcall")
	}
	log.WithField("pipecatcall", pc).Debugf("Started pipecatcall. pipecatcall_id: %s", pc.ID)

	// // start streaming tts
	// st, err := h.reqHandler.TTSV1StreamingCreate(ctx, c.CustomerID, activeflowID, tmstreaming.ReferenceTypeCall, referenceID, language, tmstreaming.Gender(gender), tmstreaming.DirectionOutgoing)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "Could not create tts streaming")
	// }
	// log.WithField("streaming", st).Debugf("Created tts streaming. streaming_id: %s", st.ID)

	// create ai call
	res, err := h.Create(ctx, c, activeflowID, aicall.ReferenceTypeCall, referenceID, cb.ID, gender, language, uuid.Nil, "")
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create aicall.")
	}
	log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)

	// go func(cctx context.Context) {
	// 	if errInit := h.chatInit(cctx, c, res); errInit != nil {
	// 		log.Errorf("Could not initialize chat. err: %v", errInit)
	// 	}

	// }(context.Background())

	return res, nil
}

// startReferenceTypeCall starts a new aicall with reference type call
func (h *aicallHandler) startReferenceTypeCallOld(
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

	// start streaming tts
	st, err := h.reqHandler.TTSV1StreamingCreate(ctx, c.CustomerID, activeflowID, tmstreaming.ReferenceTypeCall, referenceID, language, tmstreaming.Gender(gender), tmstreaming.DirectionOutgoing)
	if err != nil {
		return nil, errors.Wrap(err, "Could not create tts streaming")
	}
	log.WithField("streaming", st).Debugf("Created tts streaming. streaming_id: %s", st.ID)

	// create ai call
	res, err := h.Create(ctx, c, activeflowID, aicall.ReferenceTypeCall, referenceID, cb.ID, gender, language, st.ID, st.PodID)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create aicall.")
	}
	log.WithField("aicall", res).Debugf("Created aicall. aicall_id: %s", res.ID)

	go func(cctx context.Context) {
		if errInit := h.chatInit(cctx, c, res); errInit != nil {
			log.Errorf("Could not initialize chat. err: %v", errInit)
		}

	}(context.Background())

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
		res, err = h.Create(ctx, c, activeflowID, aicall.ReferenceTypeConversation, referenceID, uuid.Nil, gender, language, uuid.Nil, "")
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

	tmp, err := h.Create(ctx, c, uuid.Nil, aicall.ReferenceTypeNone, uuid.Nil, uuid.Nil, gender, language, uuid.Nil, "")
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create aicall.")
	}
	log.WithField("aicall", tmp).Debugf("Created aicall. aicall_id: %s", tmp.ID)

	if errInit := h.chatInit(ctx, c, tmp); errInit != nil {
		log.Errorf("Could not initialize chat. err: %v", errInit)
		return nil, errors.Wrap(errInit, "Could not initialize chat")
	}

	res, err := h.UpdateStatusStartProgressing(ctx, tmp.ID, uuid.Nil)
	if err != nil {
		log.Errorf("Could not update the status to start. err: %v", err)
		return nil, errors.Wrapf(err, "Could not update the status to start. aicall_id: %s", tmp.ID)
	}

	return res, nil
}
