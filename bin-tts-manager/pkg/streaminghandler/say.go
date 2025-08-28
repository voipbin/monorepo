package streaminghandler

import (
	"context"
	"fmt"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *streamingHandler) Say(ctx context.Context, id uuid.UUID, messageID uuid.UUID, text string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Say",
		"streaming_id": id,
		"text":         text,
	})

	st, err := h.UpdateMessageID(ctx, id, messageID)
	if err != nil {
		return errors.Wrapf(err, "could not update message ID. streaming_id: %s, message_id: %s", id, messageID)
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

		h.elevenlabsHandler.SayStop(st.VendorConfig)
		return nil

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
		return errors.Wrapf(err, "could not update message ID. streaming_id: %s, message_id: %s", id, messageID)
	}

	if st.MessageID != messageID {
		return fmt.Errorf("message ID mismatch. streaming_id: %s, current_message_id: %s, request_message_id: %s", id, st.MessageID, messageID)
	} else if st.VendorConfig == nil {
		return fmt.Errorf("vendor config is nil. streaming_id: %s", id)
	}
	log.WithField("streaming", st).Debugf("Fetched streaming info. streaming_id: %s", id)

	switch st.VendorName {
	case streaming.VendorNameElevenlabs:
		return h.elevenlabsHandler.AddText(st.VendorConfig, text)

	default:
		return errors.Errorf("unsupported vendor for text streaming. vendor: %s", st.VendorName)
	}
}
