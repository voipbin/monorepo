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

	promStreamingMessageTotal.Inc()

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

	if st.VendorName == streaming.VendorNameNone || st.VendorConfig == nil {
		return nil
	}

	handler, _ := h.getStreamerByProvider(string(st.VendorName))
	if handler == nil {
		return nil
	}

	return handler.SayStop(st.VendorConfig)
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

	handler, _ := h.getStreamerByProvider(string(st.VendorName))
	if handler == nil {
		return errors.Errorf("unsupported vendor for text streaming. vendor: %s", st.VendorName)
	}

	log.Debugf("Adding text to %s streaming. streaming_id: %s, message_id: %s, text: %s", st.VendorName, id, messageID, text)
	return handler.SayAdd(st.VendorConfig, text)
}

// SayFlush flushes the current streaming buffer.
// TODO: Implement stale audio discarding — currently ElevenLabs may still
// deliver audio frames generated before the flush. An atomic counter or
// sequence number could be used to discard stale frames on the AudioSocket side.
func (h *streamingHandler) SayFlush(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "SayFlush",
		"streaming_id": id,
	})

	st, err := h.Get(ctx, id)
	if err != nil {
		log.Infof("Could not get streaming. err: %v", err)
		return err
	}

	st.VendorLock.Lock()
	defer st.VendorLock.Unlock()

	handler, _ := h.getStreamerByProvider(string(st.VendorName))
	if handler == nil {
		log.Errorf("Unsupported vendor. vendor_name: %s", st.VendorName)
		return fmt.Errorf("unsupported vendor: %s", st.VendorName)
	}

	if errFlush := handler.SayFlush(st.VendorConfig); errFlush != nil {
		log.Errorf("Could not flush the say streaming. err: %v", errFlush)
		return errFlush
	}

	return nil
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

	handler, _ := h.getStreamerByProvider(string(st.VendorName))
	if handler == nil {
		return nil, errors.Errorf("unsupported vendor for text streaming. vendor: %s", st.VendorName)
	}

	if errFinish := handler.SayFinish(st.VendorConfig); errFinish != nil {
		return nil, errors.Wrapf(errFinish, "could not finish streaming. streaming_id: %s, message_id: %s", id, messageID)
	}
	return st, nil
}
