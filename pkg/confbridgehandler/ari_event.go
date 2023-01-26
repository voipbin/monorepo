package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// ARIStasisStart handles StasisStart ARI event for conference types.
func (h *confbridgeHandler) ARIStasisStart(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ARIStasisStart",
		"asterisk_id":    cn.AsteriskID,
		"channel_id":     cn.ID,
		"channel_status": cn.State,
		"channel_data":   cn.StasisData,
	})

	confContext := cn.StasisData["context"]
	switch confContext {
	case contextConfbridgeIncoming:
		return h.StartContextIncoming(ctx, cn)

	case contextConfbridgeOutgoing:
		log.Errorf("Currently, we don't support conference outgoing context. Something was wrong. context: %s", confContext)
		return fmt.Errorf("unsupported conference context type. context: %s", confContext)

	default:
		log.Errorf("Unsuppurted context type. context: %s", confContext)
		return fmt.Errorf("unsupported conference context type. context: %s", confContext)
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
