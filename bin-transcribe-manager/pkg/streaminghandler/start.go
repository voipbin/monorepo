package streaminghandler

import (
	"context"

	"monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// Start starts the live streaming transcribe of the given transcribe
func (h *streamingHandler) Start(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction) (*streaming.Streaming, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"transcribe_id":  transcribeID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"direction":      direction,
	})

	// create streaming record
	res, err := h.Create(ctx, customerID, transcribeID, language, direction)
	if err != nil {
		log.Errorf("Could not create streaming. err: %v", err)
		return nil, err
	}
	log.WithField("streaming", res).Debugf("Created a new streaming. streaming_id: %s", res.ID)

	// start the external media
	// send request to the call-manager
	em, err := h.reqHandler.CallV1ExternalMediaStart(
		ctx,
		res.ID,
		externalmedia.ReferenceType(referenceType),
		referenceID,
		true,
		h.listenAddress,
		defaultEncapsulation,
		defaultTransport,
		defaultConnectionType,
		defaultFormat,
		string(direction),
	)
	if err != nil {
		log.Errorf("Could not create external media. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", em).Debugf("Started external media. external_media_id: %s, host_addr: %s, media_ip: %s, media_port: %d", em.ID, h.listenAddress, em.LocalIP, em.LocalPort)

	return res, nil
}
