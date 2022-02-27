package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// StartContextIncoming handles the call which has CONTEXT=conf-in in the StasisStart argument.
func (h *confbridgeHandler) StartContextIncoming(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "StasisStartContextIncoming",
			"channel_id":    cn.ID,
			"call_id":       data["call_id"],
			"confbridge_id": data["confbridge_id"],
		},
	)
	log.Debugf("Detail data info. data: %v", data)

	// set channel type
	if err := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeConfbridge)); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		log.Errorf("Could not set channel var. err: %v", err)
		return fmt.Errorf("could not set channel var. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
	}

	// get conf info
	confbridgeID := uuid.FromStringOrNil(data["confbridge_id"])
	callID := uuid.FromStringOrNil(data["call_id"])
	if confbridgeID == uuid.Nil || callID == uuid.Nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		log.Errorf("Could not get confbridge or call info. confbridge_id: %s, call_id: %s", confbridgeID, callID)
		return fmt.Errorf("could not get confbridge id or call id info")
	}
	log.Debugf("Joining to the confbridge. call_id: %s, confbridge_id: %s", callID, confbridgeID)

	// get confbridge
	cb, err := h.db.ConfbridgeGet(ctx, confbridgeID)
	if err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		log.Errorf("Could not get confbridge. err: %v", err)
		return fmt.Errorf("could not get confbridge info. err: %v", err)
	}
	log.WithField("confbridge", cb).Debugf("Found confbridge. confbridge_id: %s", cb.ID)

	// add the channel to the bridge
	if err := h.reqHandler.AstBridgeAddChannel(ctx, cn.AsteriskID, cn.DestinationNumber, cn.ID, "", false, false); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		log.Errorf("Could not add the channel to the bridge. err: %v", err)
		return fmt.Errorf("could not put the channel to the bridge. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
	}

	switch cb.Type {
	case confbridge.TypeConference:
		return h.startContextIncomingTypeConference(ctx, cn, data, callID, cb)

	case confbridge.TypeConnect:
		return h.startContextIncomingTypeConnect(ctx, cn, data, callID, cb)

	default:
		log.Errorf("Unsupported confbridge type. confbridge_type: %s", cb.Type)
		return fmt.Errorf("unsupported confbridge type. confbridge_type: %s", cb.Type)
	}
}

// startContextIncomingTypeConference handles confbridge conference type joining channel
func (h *confbridgeHandler) startContextIncomingTypeConference(ctx context.Context, cn *channel.Channel, data map[string]string, callID uuid.UUID, cb *confbridge.Confbridge) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "startContextIncomingTypeConference",
			"channel_id":    cn.ID,
			"call_id":       callID,
			"confbridge_id": cb.ID,
		},
	)
	log.Debugf("Detail data info. data: %v", data)

	log.Debugf("Answering the conference type confbridge joining channel. call_id: %s", callID)
	if err := h.reqHandler.AstChannelAnswer(ctx, cn.AsteriskID, cn.ID); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		log.Errorf("Could not answer the channel. err: %v", err)
		return err
	}

	return nil
}

// startContextIncomingTypeConnect handles confbridge connect type joining channel
func (h *confbridgeHandler) startContextIncomingTypeConnect(ctx context.Context, cn *channel.Channel, data map[string]string, callID uuid.UUID, cb *confbridge.Confbridge) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "StasisStartContextIncoming",
			"channel_id":    cn.ID,
			"call_id":       callID,
			"confbridge_id": cb.ID,
		},
	)
	log.Debugf("Detail data info. data: %v", data)

	if len(cb.ChannelCallIDs) == 0 {
		// if it's the first channel, send a ring
		log.Debugf("First channel. Send a ring. call_id: %s", callID)
		// todo: send the ring

		return nil
	}

	// answer the call. it is safe to answer for answered call.
	log.Debugf("Answering the joining channel. call_id: %s", callID)
	if err := h.reqHandler.AstChannelAnswer(ctx, cn.AsteriskID, cn.ID); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		log.Errorf("Could not answer the channel. err: %v", err)
		return err
	}

	// send answer to the channels in the confbridge.
	for channelID, callID := range cb.ChannelCallIDs {
		if err := h.reqHandler.AstChannelAnswer(ctx, cn.AsteriskID, channelID); err != nil {
			log.Errorf("Could not answer the channel. call_id: %s, channel_id: %s, err: %v", callID, channelID, err)
		}
	}

	return nil
}
