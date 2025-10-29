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

	// get callID info
	var callID uuid.UUID
	switch referenceType {
	case pipecatcall.ReferenceTypeCall:
		callID = referenceID

	case pipecatcall.ReferenceTypeAICall:
		tmp, err := h.requestHandler.AIV1AIcallGet(ctx, referenceID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get ai call info")
		}
		if tmp.ReferenceType != amaicall.ReferenceTypeCall {
			return nil, errors.Errorf("invalid ai call reference type: %v", tmp.ReferenceType)
		}

		callID = tmp.ReferenceID

	default:
		log.Errorf("Invalid reference type. reference_type: %v", referenceType)
		return nil, errors.Errorf("invalid reference type: %v", referenceType)
	}

	if callID == uuid.Nil {
		log.Errorf("Invalid call ID retrieved from reference. reference_type: %v, reference_id: %v", referenceType, referenceID)
		return nil, errors.Errorf("invalid call ID retrieved from reference")
	}

	// start the external media
	// send request to the call-manager
	// currently only supporting call reference type
	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		res.ID,
		cmexternalmedia.ReferenceTypeCall,
		callID,
		h.listenAddress,
		defaultEncapsulation,
		defaultTransport,
		defaultConnectionType,
		defaultFormat,
		cmexternalmedia.DirectionIn,
		cmexternalmedia.DirectionOut,
	)
	if err != nil {
		log.Errorf("Could not create external media. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", em).Info("Created external media. external_media_id: ", em.ID)

	return res, nil
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

	em, err := h.requestHandler.CallV1ExternalMediaStop(ctx, pc.ID)
	if err != nil {
		log.Errorf("Could not stop external media. err: %v", err)
		return
	}
	log.WithField("external_media", em).Info("Stopped external media. external_media_id: ", em.ID)

	h.SessionStop(pc.ID)
	log.Infof("Pipecatcall stopped. pipecatcall_id: %s", pc.ID)
}
