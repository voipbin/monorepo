package confbridgehandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/channel"
)

// ARIChannelStateChange handles ChannelStateChange ARI event for confbridge-related channels.
// When a TypeJoin channel transitions to Up, it answers all non-Up peer channels
// in the same call bridge — in particular, the master call-in (SIP) channel.
func (h *confbridgeHandler) ARIChannelStateChange(ctx context.Context, cn *channel.Channel) {
	if cn.Type != channel.TypeJoin {
		return
	}

	if cn.State != ari.ChannelStateUp {
		return
	}

	// BridgeID is not populated via ChannelEnteredBridge for join channels;
	// use the bridge_id from StasisData instead.
	bridgeID := cn.BridgeID
	if bridgeID == "" {
		bridgeID = cn.StasisData[channel.StasisDataTypeBridgeID]
	}
	if bridgeID == "" {
		return
	}

	log := logrus.WithFields(logrus.Fields{
		"func":       "ARIChannelStateChange",
		"channel_id": cn.ID,
		"bridge_id":  bridgeID,
	})

	br, err := h.bridgeHandler.Get(ctx, bridgeID)
	if err != nil {
		log.Errorf("Could not get bridge info. bridge_id: %s, err: %v", bridgeID, err)
		return
	}

	// only apply to call bridges (reference_type=call)
	if br.ReferenceType != bridge.ReferenceTypeCall {
		return
	}

	for _, peerChannelID := range br.ChannelIDs {
		if peerChannelID == cn.ID {
			continue
		}

		peer, err := h.channelHandler.Get(ctx, peerChannelID)
		if err != nil {
			log.Debugf("Could not get peer channel. channel_id: %s, err: %v", peerChannelID, err)
			continue
		}

		if peer.State == ari.ChannelStateUp {
			continue
		}

		log.Infof("Auto-answering peer channel in call bridge via join channel. peer_channel_id: %s, peer_state: %s", peer.ID, peer.State)
		if errAnswer := h.channelHandler.Answer(ctx, peer.ID); errAnswer != nil {
			log.Warnf("Could not answer peer channel. channel_id: %s, err: %v", peer.ID, errAnswer)
		}
	}
}
