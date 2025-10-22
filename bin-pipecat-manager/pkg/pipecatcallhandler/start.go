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
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType pipecatcall.ReferenceType,
	referenceID uuid.UUID,
	llm pipecatcall.LLM,
	stt pipecatcall.STT,
	tts pipecatcall.TTS,
	voiceID string,
	messages []map[string]any,
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
		customerID,
		activeflowID,
		referenceType,
		referenceID,
		llm,
		stt,
		tts,
		voiceID,
		messages,
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

	if pc.RunnerWebsocket != nil {
		if errClose := pc.RunnerWebsocket.Close(); errClose != nil {
			log.Errorf("Could not close the pipecat runner websocket. err: %v", errClose)
		} else {
			log.Infof("Closed the pipecat runner websocket.")
		}
	}

	if pc.RunnerServer != nil {
		if errClose := pc.RunnerServer.Close(); errClose != nil {
			log.Errorf("Could not close the pipecat runner server. err: %v", errClose)
		} else {
			log.Infof("Closed the pipecat runner server.")
		}
	}

	if pc.RunnerListener != nil {
		if errClose := pc.RunnerListener.Close(); errClose != nil {
			log.Errorf("Could not close the pipecat runner listener. err: %v", errClose)
		} else {
			log.Infof("Closed the pipecat runner listener.")
		}
	}

	if pc.AsteriskConn != nil {
		if errClose := pc.AsteriskConn.Close(); errClose != nil {
			log.Errorf("Could not close the asterisk connection. err: %v", errClose)
		} else {
			log.Infof("Closed the asterisk connection.")
		}
	}

	h.Delete(ctx, pc.ID)
	log.Infof("Pipecatcall stopped. pipecatcall_id: %s", pc.ID)
}
