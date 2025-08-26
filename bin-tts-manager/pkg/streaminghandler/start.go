package streaminghandler

import (
	"context"

	"monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/models/streaming"
)

// Start starts the live streaming transcribe of the given transcribe
func (h *streamingHandler) Start(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType streaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	gender streaming.Gender,
	direction streaming.Direction,
) (*streaming.Streaming, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "Start",
		"customer_id":      customerID,
		"reference_type":   referenceType,
		"reference_id":     referenceID,
		"language":         language,
		"direction_listen": direction,
	})

	// create streaming record
	res, err := h.Create(ctx, customerID, referenceType, referenceID, language, gender, direction)
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

func (h *streamingHandler) Say(ctx context.Context, id uuid.UUID, text string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Say",
		"streaming_id": id,
		"text":         text,
	})

	st, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get streaming info. streaming_id: %s", id)
	}
	log.WithField("streaming", st).Debugf("Fetched streaming info. streaming_id: %s", id)

	if st.VendorConfig == nil {
		log.Debugf("Vendor config is nil. initializing the vendor config. vendor: %s", st.VendorName)
		if errRun := h.runStreamer(st); errRun != nil {
			return errors.Wrapf(errRun, "could not run streamer. streaming_id: %s", id)
		}
	}

	switch st.VendorName {
	case streaming.VendorNameElevenlabs:
		return h.elevenlabsHandler.AddText(st.VendorConfig, text)

	default:
		return errors.Errorf("unsupported vendor for text streaming. vendor: %s", st.VendorName)
	}
}

func (h *streamingHandler) SayStop(ctx context.Context, id uuid.UUID) error {

	st, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get streaming info. streaming_id: %s", id)
	}

	switch st.VendorName {
	case streaming.VendorNameNone:
		return nil

	case streaming.VendorNameElevenlabs:
		if st.VendorConfig == nil {
			return nil
		}

		h.elevenlabsHandler.SayStop(st.VendorConfig)
		return nil

	default:
		return nil
	}
}
