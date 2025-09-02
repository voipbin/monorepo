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

	// start the external media
	// send request to the call-manager
	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		res.ID,
		externalmedia.ReferenceType(referenceType),
		referenceID,
		h.listenAddress,
		defaultEncapsulation,
		defaultTransport,
		defaultConnectionType,
		defaultFormat,
		externalmedia.DirectionNone,
		externalmedia.Direction(direction),
	)
	if err != nil {
		log.Errorf("Could not create external media. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", em).Debugf("Started external media. external_media_id: %s, host_addr: %s, media_ip: %s, media_port: %d", em.ID, h.listenAddress, em.LocalIP, em.LocalPort)

	return res, nil
}
