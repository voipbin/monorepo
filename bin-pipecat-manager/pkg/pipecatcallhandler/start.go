package pipecatcallhandler

import (
	"context"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"
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
	sttLanguage string,
	ttsType pipecatcall.TTSType,
	ttsLanguage string,
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
		sttLanguage,
		ttsType,
		ttsLanguage,
		ttsVoiceID,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create pipecatcall")
	}
	log.WithField("pipecatcall", res).Info("Created pipecatcall. pipecatcall_id: ", res.ID)

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
		"func":           "startReferenceTypeCall",
		"pipecatcall_id": pc.ID,
	})

	c, err := h.requestHandler.CallV1CallGet(ctx, pc.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get call info")
	}
	log.WithField("call", c).Info("Retrieved call info. call_id: ", c.ID)

	// start the external media
	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		pc.ID,
		cmexternalmedia.ReferenceTypeCall,
		c.ID,
		"INCOMING",
		defaultEncapsulation,
		defaultTransport,
		"", // transportData
		defaultConnectionType,
		defaultFormat,
		cmexternalmedia.DirectionIn,
		cmexternalmedia.DirectionOut,
	)
	if err != nil {
		return errors.Wrapf(err, "could not create external media")
	}
	log.WithField("external_media", em).Info("Created external media. external_media_id: ", em.ID)

	// Connect to Asterisk via WebSocket
	conn, err := h.websocketAsteriskConnect(ctx, em.MediaURI)
	if err != nil {
		log.Errorf("Could not connect WebSocket to Asterisk. err: %v", err)
		if _, errStop := h.requestHandler.CallV1ExternalMediaStop(ctx, em.ID); errStop != nil {
			log.Errorf("Could not stop orphaned external media. err: %v", errStop)
		}
		return errors.Wrapf(err, "could not connect to asterisk websocket")
	}
	log.Debugf("WebSocket connected to Asterisk. media_uri: %s", em.MediaURI)

	connAstDone := make(chan struct{})

	llmKey := h.runGetLLMKey(ctx, pc)
	se, err := h.SessionCreate(pc, pc.ID, conn, connAstDone, llmKey)
	if err != nil {
		_ = conn.Close()
		return errors.Wrapf(err, "could not create pipecatcall session")
	}

	// Start pipecat runner
	go func() {
		defer se.Cancel()
		h.RunnerStart(pc, se)
	}()

	// Start media handler — sole reader on the Asterisk WebSocket.
	// Closes connAstDone on exit to signal the lifecycle monitor.
	go func() {
		defer se.Cancel()
		h.runAsteriskReceivedMediaHandle(se)
	}()

	// Monitor lifecycle — when context or WebSocket dies, terminate
	go func() {
		select {
		case <-se.Ctx.Done():
		case <-connAstDone:
		}
		log.Debugf("Asterisk connection or context done, terminating. pipecatcall_id: %s", pc.ID)
		h.terminate(context.Background(), pc)
	}()

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
		em, err := h.requestHandler.CallV1ExternalMediaStart(
			ctx,
			pc.ID,
			cmexternalmedia.ReferenceTypeCall,
			c.ReferenceID,
			"INCOMING",
			defaultEncapsulation,
			defaultTransport,
			"", // transportData
			defaultConnectionType,
			defaultFormat,
			cmexternalmedia.DirectionIn,
			cmexternalmedia.DirectionOut,
		)
		if err != nil {
			return errors.Wrapf(err, "could not create external media")
		}
		log.WithField("external_media", em).Info("Created external media. external_media_id: ", em.ID)

		// Connect to Asterisk via WebSocket
		conn, err := h.websocketAsteriskConnect(ctx, em.MediaURI)
		if err != nil {
			log.Errorf("Could not connect WebSocket to Asterisk. err: %v", err)
			if _, errStop := h.requestHandler.CallV1ExternalMediaStop(ctx, em.ID); errStop != nil {
				log.Errorf("Could not stop orphaned external media. err: %v", errStop)
			}
			return errors.Wrapf(err, "could not connect to asterisk websocket")
		}
		log.Debugf("WebSocket connected to Asterisk. media_uri: %s", em.MediaURI)

		connAstDone := make(chan struct{})

		llmKey := h.runGetLLMKey(ctx, pc)
		se, err := h.SessionCreate(pc, pc.ID, conn, connAstDone, llmKey)
		if err != nil {
			_ = conn.Close()
			return errors.Wrapf(err, "could not create pipecatcall session")
		}

		// Start pipecat runner
		go func() {
			defer se.Cancel()
			h.RunnerStart(pc, se)
		}()

		// Start media handler — sole reader on the Asterisk WebSocket.
		// Closes connAstDone on exit to signal the lifecycle monitor.
		go func() {
			defer se.Cancel()
			h.runAsteriskReceivedMediaHandle(se)
		}()

		// Monitor lifecycle — when context or WebSocket dies, terminate
		go func() {
			select {
			case <-se.Ctx.Done():
			case <-connAstDone:
			}
			log.Debugf("Asterisk connection or context done, terminating. pipecatcall_id: %s", pc.ID)
			h.terminate(context.Background(), pc)
		}()

		return nil

	default:
		llmKey := h.runGetLLMKey(ctx, pc)
		se, err := h.SessionCreate(pc, uuid.Nil, nil, nil, llmKey)
		if err != nil {
			return errors.Wrapf(err, "could not create pipecatcall session")
		}

		go h.RunnerStart(pc, se)
		return nil
	}
}

func (h *pipecatcallHandler) Terminate(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pipecatcall info")
	}

	h.terminate(ctx, res)
	return res, nil
}

func (h *pipecatcallHandler) terminate(ctx context.Context, pc *pipecatcall.Pipecatcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "terminate",
		"pipecatcall_id": pc.ID,
	})
	log.Infof("Terminating pipecatcall...")

	switch pc.ReferenceType {
	case pipecatcall.ReferenceTypeCall:
		if errTerminate := h.terminateReferenceTypeCall(ctx, pc); errTerminate != nil {
			log.Errorf("Could not terminate reference type call. err: %v", errTerminate)
			return
		}

	case pipecatcall.ReferenceTypeAICall:
		if errTerminate := h.terminateReferenceTypeAICall(ctx, pc); errTerminate != nil {
			log.Errorf("Could not terminate reference type ai call. err: %v", errTerminate)
			return
		}

	default:
		// no action needed for other reference types
		log.Debugf("No action needed to stop for reference type: %v", pc.ReferenceType)
	}

	h.SessionStop(pc.ID)
	log.Infof("Pipecatcall terminated. pipecatcall_id: %s", pc.ID)
}

func (h *pipecatcallHandler) terminateReferenceTypeCall(ctx context.Context, pc *pipecatcall.Pipecatcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "terminateReferenceTypeCall",
		"pipecatcall_id": pc.ID,
	})

	c, err := h.requestHandler.CallV1CallGet(ctx, pc.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get call info")
	}
	log.WithField("call", c).Info("Retrieved call info. call_id: ", c.ID)

	if c.Status != cmcall.StatusProgressing {
		log.Debugf("No action needed to stop for call status: %v", c.Status)
		return nil
	}

	// note: we use the pipecatcall's ID as external media id.
	// so this is correct.
	em, err := h.requestHandler.CallV1ExternalMediaStop(ctx, pc.ID)
	if err != nil {
		return errors.Wrapf(err, "could not stop external media")
	}
	log.WithField("external_media", em).Info("Stopped external media. external_media_id: ", em.ID)

	return nil
}

func (h *pipecatcallHandler) terminateReferenceTypeAICall(ctx context.Context, pc *pipecatcall.Pipecatcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "terminateReferenceTypeAICall",
		"pipecatcall_id": pc.ID,
	})

	ac, err := h.requestHandler.AIV1AIcallGet(ctx, pc.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get ai call info")
	}
	log.WithField("ai_call", ac).Info("Retrieved ai call info. ai_call_id: ", ac.ID)

	switch ac.ReferenceType {
	case amaicall.ReferenceTypeCall:
		em, err := h.requestHandler.CallV1ExternalMediaStop(ctx, pc.ID)
		if err != nil {
			return errors.Wrapf(err, "could not stop external media")
		}
		log.WithField("external_media", em).Info("Stopped external media. external_media_id: ", em.ID)

	default:
		log.Debugf("No action needed to stop for ai call reference type: %v", ac.ReferenceType)
	}

	return nil
}
