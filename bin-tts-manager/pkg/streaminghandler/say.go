package streaminghandler

import (
	"context"
	"fmt"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// SayInit initializes the streaming for a new message
func (h *streamingHandler) SayInit(ctx context.Context, id uuid.UUID, messageID uuid.UUID) (*streaming.Streaming, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "SayInit",
		"streaming_id": id,
	})

	res, err := h.UpdateMessageID(ctx, id, messageID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update message ID. streaming_id: %s, message_id: %s", id, messageID)
	}
	log.WithField("streaming", res).Debugf("Fetched streaming info. streaming_id: %s", id)

	if res.VendorConfig == nil {
		log.Debugf("Vendor config is nil. Initializing the vendor config.")
		if errRun := h.runStreamer(ctx, res); errRun != nil {
			return nil, errors.Wrapf(errRun, "could not run streamer. streaming_id: %s", id)
		}
	}

	return res, nil
}

// SayStop stops the current message being synthesized
// and sets the message ID to nil
func (h *streamingHandler) SayStop(ctx context.Context, id uuid.UUID) error {

	st, err := h.UpdateMessageID(ctx, id, uuid.Nil)
	if err != nil {
		return errors.Wrapf(err, "could not update message ID. streaming_id: %s, message_id: %s", id, uuid.Nil)
	}

	switch st.VendorName {
	case streaming.VendorNameNone:
		return nil

	case streaming.VendorNameElevenlabs:
		if st.VendorConfig == nil {
			return nil
		}

		return h.elevenlabsHandler.SayStop(st.VendorConfig)

	default:
		return nil
	}
}

// SayAdd adds text to the current message being synthesized
func (h *streamingHandler) SayAdd(ctx context.Context, id uuid.UUID, messageID uuid.UUID, text string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "SayAdd",
		"streaming_id": id,
		"text":         text,
	})

	st, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get streaming info. streaming_id: %s, message_id: %s", id, messageID)
	}

	if st.MessageID != messageID {
		return fmt.Errorf("message ID mismatch. streaming_id: %s, current_message_id: %s, request_message_id: %s", id, st.MessageID, messageID)
	} else if st.VendorConfig == nil {
		return fmt.Errorf("vendor config is nil. streaming_id: %s", id)
	}

	switch st.VendorName {
	case streaming.VendorNameElevenlabs:
		log.Debugf("Adding text to ElevenLabs streaming. streaming_id: %s, message_id: %s, text: %s", id, messageID, text)
		return h.elevenlabsHandler.SayAdd(st.VendorConfig, text)

	default:
		return errors.Errorf("unsupported vendor for text streaming. vendor: %s", st.VendorName)
	}
}

// SayFinish set finish flag to true for the current message being synthesized
func (h *streamingHandler) SayFinish(ctx context.Context, id uuid.UUID, messageID uuid.UUID) (*streaming.Streaming, error) {
	st, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get streaming info. streaming_id: %s, message_id: %s", id, messageID)
	}

	if st.MessageID != messageID {
		return nil, fmt.Errorf("message ID mismatch. streaming_id: %s, current_message_id: %s, request_message_id: %s", id, st.MessageID, messageID)
	} else if st.VendorConfig == nil {
		return nil, fmt.Errorf("vendor config is nil. streaming_id: %s", id)
	}

	switch st.VendorName {
	case streaming.VendorNameElevenlabs:
		if errFinish := h.elevenlabsHandler.SayFinish(st.VendorConfig); errFinish != nil {
			return nil, errors.Wrapf(errFinish, "could not finish the elevenlabs streaming. streaming_id: %s, message_id: %s", id, messageID)
		}
		return st, nil

	default:
		return nil, errors.Errorf("unsupported vendor for text streaming. vendor: %s", st.VendorName)
	}
}
