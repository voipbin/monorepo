package callhandler

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/playback"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

// ARIStasisStart is called when the channel handler received StasisStart.
func (h *callHandler) ARIStasisStart(ctx context.Context, cn *channel.Channel) error {
	logrus.WithField("func", "ARIStasisStart").Debugf("Execute the stasis start event handler for call.")

	return h.Start(ctx, cn)
}

// ARIChannelDestroyed handles ChannelDestroyed ARI event
func (h *callHandler) ARIChannelDestroyed(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithField("func", "ARIChannelDestroyed")

	switch cn.Type {
	case channel.TypeCall:
		_, err := h.Hangup(ctx, cn)
		if err != nil {
			return err
		}
		return nil

	case channel.TypeConfbridge, channel.TypeJoin, channel.TypeExternal, channel.TypeRecording, channel.TypeApplication:
		// we don't do anything at here.
		return nil

	default:
		log.WithField("channel", cn).Errorf("Unsupported channel type. type: %v", cn.Type)
		return nil
	}
}

// ARIChannelDtmfReceived handles ChannelDtmfReceived ARI event
func (h *callHandler) ARIChannelDtmfReceived(ctx context.Context, cn *channel.Channel, digit string, duration int) error {

	// support pjsip type only for now.
	if cn.Tech != channel.TechPJSIP {
		return nil
	}

	if errDigits := h.digitsReceived(ctx, cn, digit, duration); errDigits != nil {
		return errDigits
	}

	return nil
}

// ARIPlaybackFinished handles PlaybackFinished ARI event
// parsed playbackID to the action id, and execute the next action if its correct.
func (h *callHandler) ARIPlaybackFinished(ctx context.Context, cn *channel.Channel, e *ari.PlaybackFinished) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ARIPlaybackFinished",
		"channel_id": cn.ID,
		"event":      e,
	})

	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		log.Errorf("Could not get call info. channel: %s, err: %v", cn.ID, err)
		return err
	}

	// compare actionID
	actionID := uuid.FromStringOrNil(strings.TrimPrefix(e.Playback.ID, playback.IDPrefixCall))
	if c.Action.ID != actionID {
		log.Debugf("The call's action id does not match. call: %s, channel: %s, action: %s", c.ID, cn.ID, actionID)
		return nil
	}

	// go to next action
	return h.ActionNext(ctx, c)
}

func (h *callHandler) ARIChannelStateChange(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ARIChannelStateChange",
		"channel": cn,
	})
	status := call.GetStatusByChannelState(cn.State)
	if status != call.StatusRinging && status != call.StatusProgressing {
		// the call cares only riniging/progressing at here.
		// other statuses will be handled in the other func.
		return nil
	}

	// get call
	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err == dbhandler.ErrNotFound {
		// this is ok. just not a call channel. we just ignore this.
		return nil
	} else if err != nil {
		log.Errorf("Could not get call. err: %v", err)
		return err
	}

	// we care only ringing/progress at here.
	switch status {
	case call.StatusRinging:
		return h.updateStatusRinging(ctx, cn, c)
	case call.StatusProgressing:
		if err := h.updateStatusProgressing(ctx, cn, c); err != nil {
			return err
		}
		// When a channel in a call bridge becomes Up (answered), auto-answer
		// all other channels in the same bridge that are not yet Up.
		// This ensures the master incoming leg (call-in) is answered when
		// the outgoing leg answers, regardless of early_media setting.
		// answerCallBridgePeers is best-effort: the call status has already
		// been committed as Progressing, so failure here is non-fatal.
		if cn.State == ari.ChannelStateUp && cn.BridgeID != "" {
			h.answerCallBridgePeers(ctx, cn)
		}
		return nil
	}

	return nil
}

// answerCallBridgePeers answers all non-Up channels in the same call bridge
// when one channel transitions to Up state. It is best-effort: all errors are
// logged internally and never propagated to the caller.
func (h *callHandler) answerCallBridgePeers(ctx context.Context, cn *channel.Channel) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "answerCallBridgePeers",
		"channel_id": cn.ID,
		"bridge_id":  cn.BridgeID,
	})

	br, err := h.bridgeHandler.Get(ctx, cn.BridgeID)
	if err != nil {
		log.Errorf("Could not get bridge info. bridge_id: %s, err: %v", cn.BridgeID, err)
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

		// skip channels already Up
		if peer.State == ari.ChannelStateUp {
			continue
		}

		log.Infof("Auto-answering peer channel in call bridge. peer_channel_id: %s, peer_state: %s", peer.ID, peer.State)
		if errAnswer := h.channelHandler.Answer(ctx, peer.ID); errAnswer != nil {
			// Answer failure is expected during groupcall race teardown (loser channels
			// being hung up concurrently). Log at Warnf to avoid alert noise.
			log.Warnf("Could not answer peer channel. channel_id: %s, err: %v", peer.ID, errAnswer)
		}
	}
}

func (h *callHandler) ARIChannelLeftBridge(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ARIChannelLeftBridge",
		"channel_id": cn.ID,
	})

	switch cn.Type {
	case channel.TypeCall:
		return nil
	case channel.TypeJoin:
		return h.bridgeLeftJoin(ctx, cn, br)
	case channel.TypeExternal:
		return h.bridgeLeftExternal(ctx, cn, br)

	default:
		log.WithFields(logrus.Fields{
			"channel": cn,
			"bridge":  br,
		}).Errorf("Could not find correct event handler.")
	}
	return nil
}
