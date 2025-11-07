package pipecatcallhandler

import (
	"context"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *pipecatcallHandler) Start(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType pipecatcall.ReferenceType,
	referenceID uuid.UUID,
	llmType pipecatcall.LLMType,
	llmMessages []map[string]any,
	sttType pipecatcall.STTType,
	ttsType pipecatcall.TTSType,
	ttsVoiceID string,
) (*pipecatcall.Pipecatcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"customer_id":    customerID,
		"activeflow_id":  activeflowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	res, err := h.Create(
		ctx,
		id,
		customerID,
		activeflowID,
		referenceType,
		referenceID,
		llmType,
		llmMessages,
		sttType,
		ttsType,
		ttsVoiceID,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create pipecatcall")
	}
	log.WithField("pipecatcall", res).Info("Created pipecatcall. pipecatcall_id: ", res.ID)

	log.WithField("llm_messages", res.LLMMessages).Info("Pipecatcall LLM messages.")

	// start based on reference type
	switch referenceType {
	case pipecatcall.ReferenceTypeCall:
		if errStart := h.startReferenceTypeCall(ctx, res); errStart != nil {
			return nil, errors.Wrapf(errStart, "could not start reference type call")
		}

	case pipecatcall.ReferenceTypeAICall:
		if errStart := h.startReferenceTypeAIcall(ctx, res); errStart != nil {
			return nil, errors.Wrapf(errStart, "could not start reference type ai call")
		}

	default:
		log.Errorf("Invalid reference type. reference_type: %v", referenceType)
		return nil, errors.Errorf("invalid reference type: %v", referenceType)
	}

	return res, nil
}

func (h *pipecatcallHandler) startReferenceTypeCall(ctx context.Context, pc *pipecatcall.Pipecatcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startReferenceTypeAIcall",
		"pipecatcall_id": pc.ID,
	})

	c, err := h.requestHandler.CallV1CallGet(ctx, pc.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get call info")
	}
	log.WithField("call", c).Info("Retrieved call info. call_id: ", c.ID)

	// start the external media
	// send request to the call-manager
	// currently only supporting call reference type
	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		pc.ID,
		cmexternalmedia.ReferenceTypeCall,
		c.ID,
		h.listenAddress,
		defaultEncapsulation,
		defaultTransport,
		defaultConnectionType,
		defaultFormat,
		cmexternalmedia.DirectionIn,
		cmexternalmedia.DirectionOut,
	)
	if err != nil {
		return errors.Wrapf(err, "could not create external media")
	}
	log.WithField("external_media", em).Info("Created external media. external_media_id: ", em.ID)

	return nil
}

func (h *pipecatcallHandler) startReferenceTypeAIcall(ctx context.Context, pc *pipecatcall.Pipecatcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startReferenceTypeAIcall",
		"pipecatcall_id": pc.ID,
	})

	c, err := h.requestHandler.AIV1AIcallGet(ctx, pc.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get ai call info")
	}
	log.WithField("ai_call", c).Info("Retrieved ai call info. ai_call_id: ", c.ID)

	switch c.ReferenceType {
	case amaicall.ReferenceTypeCall:
		// start the external media
		// send request to the call-manager
		// currently only supporting call reference type
		em, err := h.requestHandler.CallV1ExternalMediaStart(
			ctx,
			pc.ID,
			cmexternalmedia.ReferenceTypeCall,
			c.ReferenceID,
			h.listenAddress,
			defaultEncapsulation,
			defaultTransport,
			defaultConnectionType,
			defaultFormat,
			cmexternalmedia.DirectionIn,
			cmexternalmedia.DirectionOut,
		)
		if err != nil {
			return errors.Wrapf(err, "could not create external media")
		}
		log.WithField("external_media", em).Info("Created external media. external_media_id: ", em.ID)
		return nil

	default:
		se, err := h.SessionCreate(pc, uuid.Nil, nil)
		if err != nil {
			return errors.Wrapf(err, "could not create pipecatcall session")
		}

		go h.RunnerStart(pc, se)
		return nil
	}
}

func (h *pipecatcallHandler) Stop(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pipecatcall info")
	}

	h.stop(ctx, res)
	return res, nil
}

func (h *pipecatcallHandler) stop(ctx context.Context, pc *pipecatcall.Pipecatcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Stop",
		"pipecatcall_id": pc.ID,
	})
	log.Infof("Stopping pipecatcall...")

	switch pc.ReferenceType {
	case pipecatcall.ReferenceTypeCall:
		c, err := h.requestHandler.AIV1AIcallGet(ctx, pc.ReferenceID)
		if err != nil {
			log.Errorf("Could not get ai call info. err: %v", err)
			break
		}
		log.WithField("ai_call", c).Info("Retrieved ai call info. ai_call_id: ", c.ID)

		if c.ReferenceType != amaicall.ReferenceTypeCall {
			break
		}

		em, err := h.requestHandler.CallV1ExternalMediaStop(ctx, pc.ID)
		if err != nil {
			log.Errorf("Could not stop external media. err: %v", err)
			return
		}
		log.WithField("external_media", em).Info("Stopped external media. external_media_id: ", em.ID)

	default:
		// no action needed for other reference types
		log.Debugf("No action needed to stop for reference type: %v", pc.ReferenceType)
	}

	h.SessionStop(pc.ID)
	log.Infof("Pipecatcall stopped. pipecatcall_id: %s", pc.ID)
}
