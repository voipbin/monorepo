package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/channel"
)

// ARIStasisStart handles StasisStart ARI event for conference types.
func (h *confbridgeHandler) ARIStasisStart(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ARIStasisStart",
		"channel": cn,
	})

	chContext := cn.StasisData[channel.StasisDataTypeContext]
	switch channel.Context(chContext) {
	case channel.ContextConfIncoming:
		return h.StartContextIncoming(ctx, cn)

	case channel.ContextConfOutgoing:
		log.Errorf("Currently, we don't support conference outgoing context. Something was wrong. context: %s", chContext)
		return fmt.Errorf("unsupported conference context type. context: %s", chContext)

	default:
		log.Errorf("Unsuppurted context type. context: %s", chContext)
		return fmt.Errorf("unsupported conference context type. context: %s", chContext)
	}
}

// ARIChannelLeftBridge handles ChannelLeftBridge ARI event for conference types.
func (h *confbridgeHandler) ARIChannelLeftBridge(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	if cn.Type != channel.TypeConfbridge {
		return nil
	}

	return h.Leaved(ctx, cn, br)
}

// ARIChannelEnteredBridge handles ChannelEnteredBridge ARI event for conference types.
func (h *confbridgeHandler) ARIChannelEnteredBridge(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	if cn.Type != channel.TypeConfbridge {
		return nil
	}

	return h.Joined(ctx, cn, br)
}

// ARIBridgeDestroyed handles BridgeDestroyed ARI event for conference types.
func (h *confbridgeHandler) ARIBridgeDestroyed(ctx context.Context, br *bridge.Bridge) error {

	if br.ReferenceType != bridge.ReferenceTypeConfbridge {
		return nil
	}

	return h.Terminate(ctx, br.ReferenceID)
}

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
