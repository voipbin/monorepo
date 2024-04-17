package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// Joined handles joined call
func (h *confbridgeHandler) Joined(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Joined",
		"channel_id": cn.ID,
		"bridge_id":  br.ID,
	})

	confbridgeID := uuid.FromStringOrNil(cn.StasisData[channel.StasisDataTypeConfbridgeID])
	callID := uuid.FromStringOrNil(cn.StasisData[channel.StasisDataTypeCallID])
	log = log.WithFields(logrus.Fields{
		"conbridge_id": confbridgeID,
		"call_id":      callID,
	})
	log.Debug("Joined channel/call to the confbridge.")

	cb, err := h.AddChannelCallID(ctx, confbridgeID, cn.ID, callID)
	if err != nil {
		log.Errorf("Could not add the channel/call info to the confbridge. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseUnallocated)
		return errors.Wrap(err, "could not add the confbridge's channel/call info")
	}

	c, err := h.reqHandler.CallV1CallUpdateConfbridgeID(ctx, callID, confbridgeID)
	if err != nil {
		log.Errorf("Could not update the confbridge id. call_id: %s, confbridge_id: %s", callID, confbridgeID)
		return errors.Wrap(err, "could not update the confbridge id")
	}

	switch cb.Type {
	case confbridge.TypeConnect:
		return h.joinedTypeConnect(ctx, cn.ID, c, cb)

	case confbridge.TypeConference:
		return h.joinedTypeConference(ctx, cn.ID, c, cb)
	}

	return nil
}

// joinedTypeConnect handles confbridge connect type joining channel
func (h *confbridgeHandler) joinedTypeConnect(ctx context.Context, channelID string, c *call.Call, cb *confbridge.Confbridge) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "joinedTypeConnect",
		"channel_id": channelID,
		"call":       c,
		"confbridge": cb,
	})

	if len(cb.ChannelCallIDs) == 1 {
		// if it's the first channel, send a ring
		log.Debugf("First channel. Send a ring. call_id: %s", c.ID)
		if errRing := h.channelHandler.Ring(ctx, channelID); errRing != nil {
			log.Errorf("Could not ring the channel. err: %v", errRing)
			return errors.Wrap(errRing, "could not ring the channel")
		}

		return nil
	}

	// get flagring
	flagRing := false
	for _, callID := range cb.ChannelCallIDs {
		// get call info
		c, err := h.reqHandler.CallV1CallGet(ctx, callID)
		if err != nil {
			log.Errorf("Could not get call info. call_id: %s, err: %v", callID, err)
			return errors.Wrap(err, "Could not get call info.")
		}

		if c.Direction == call.DirectionOutgoing && (c.Status == call.StatusRinging || c.Status == call.StatusDialing) {
			// the outgoing call is ringing.
			flagRing = true
			break
		}
	}

	if flagRing {
		if errRing := h.Ring(ctx, cb.ID); errRing != nil {
			log.Errorf("Could not ring the confbridge. err: %v", errRing)
		}
	} else {
		if errAnswer := h.Answer(ctx, cb.ID); errAnswer != nil {
			log.Errorf("Could not answer the confbridge. err: %v", errAnswer)
		}
	}

	return nil
}

// joinedTypeConference handles confbridge connect type joined channel
func (h *confbridgeHandler) joinedTypeConference(ctx context.Context, channelID string, c *call.Call, cb *confbridge.Confbridge) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "joinedTypeConference",
		"channel_id":    channelID,
		"call_id":       c.ID,
		"confbridge_id": cb.ID,
	})

	log.Debugf("Answering the conference type confbridge joining channel. call_id: %s", c.ID)
	if errAnswer := h.channelHandler.Answer(ctx, channelID); errAnswer != nil {
		log.Errorf("Could not answer the channel. err: %v", errAnswer)
		return errors.Wrap(errAnswer, "could not answer the channel")
	}

	return nil
}
