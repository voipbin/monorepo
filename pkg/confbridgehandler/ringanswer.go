package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// Ring rings all channels in the confbridge
func (h *confbridgeHandler) Ring(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Ring",
		"confbridge_id": id,
	})

	cb, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return errors.Wrap(err, "could not get confbridge info")
	}

	// only connect type is valid for confbridge ring.
	if cb.Type != confbridge.TypeConnect {
		return nil
	}

	for channelID := range cb.ChannelCallIDs {
		if errRing := h.channelHandler.Ring(ctx, channelID); errRing != nil {
			log.Errorf("Could not ring the channel. err: %v", errRing)
		}
	}

	return nil
}

// Answer answers all channels in the confbridge
func (h *confbridgeHandler) Answer(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Answer",
		"confbridge_id": id,
	})

	cb, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return errors.Wrap(err, "could not get confbridge info")
	}

	if cb.Type != confbridge.TypeConnect {
		return nil
	}

	for channelID := range cb.ChannelCallIDs {
		if errRing := h.channelHandler.Answer(ctx, channelID); errRing != nil {
			log.Errorf("Could not asnwer the channel. err: %v", errRing)
		}
	}

	return nil
}
