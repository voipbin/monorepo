package streaminghandler

import (
	"context"

	"monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/models/streaming"
)

// Start starts the live streaming transcribe of the given transcribe
func (h *streamingHandler) Start(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType streaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	gender streaming.Gender,
	direction streaming.Direction,
) (*streaming.Streaming, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "Start",
		"customer_id":      customerID,
		"activeflow_id":    activeflowID,
		"reference_type":   referenceType,
		"reference_id":     referenceID,
		"language":         language,
		"direction_listen": direction,
	})

	// create streaming record
	res, err := h.Create(ctx, customerID, activeflowID, referenceType, referenceID, language, gender, direction)
	if err != nil {
		log.Errorf("Could not create streaming. err: %v", err)
		return nil, err
	}
	log.WithField("streaming", res).Debugf("Created a new streaming. streaming_id: %s", res.ID)

	if err := h.startExternalMedia(ctx, res); err != nil {
		return nil, err
	}

	return res, nil
}

// StartWithID starts a streaming session with a pre-determined ID, provider, and voiceID.
func (h *streamingHandler) StartWithID(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	referenceType streaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	provider string,
	voiceID string,
	direction streaming.Direction,
) (*streaming.Streaming, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "StartWithID",
		"streaming_id":   id,
		"customer_id":    customerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	res, err := h.createWithID(ctx, id, customerID, uuid.Nil, referenceType, referenceID, language, streaming.GenderNeutral, provider, voiceID, direction)
	if err != nil {
		log.Errorf("Could not create streaming. err: %v", err)
		return nil, err
	}
	log.WithField("streaming", res).Debugf("Created a new streaming. streaming_id: %s", res.ID)

	if err := h.startExternalMedia(ctx, res); err != nil {
		return nil, err
	}

	return res, nil
}

// startExternalMedia sends request to call-manager to start the external media channel.
func (h *streamingHandler) startExternalMedia(ctx context.Context, st *streaming.Streaming) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "startExternalMedia",
		"streaming_id": st.ID,
	})

	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		st.ID,
		externalmedia.ReferenceType(st.ReferenceType),
		st.ReferenceID,
		h.listenAddress,
		defaultEncapsulation,
		defaultTransport,
		"", // transportData
		defaultConnectionType,
		defaultFormat,
		externalmedia.DirectionNone,
		externalmedia.Direction(st.Direction),
	)
	if err != nil {
		log.Errorf("Could not create external media. err: %v", err)
		promStreamingErrorTotal.WithLabelValues("unknown").Inc()
		return err
	}
	log.WithField("external_media", em).Debugf("Started external media. external_media_id: %s, host_addr: %s, media_ip: %s, media_port: %d", em.ID, h.listenAddress, em.LocalIP, em.LocalPort)

	return nil
}
