package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// ARIStasisStart handles StasisStart ARI event for conference types.
func (h *confbridgeHandler) ARIStasisStart(cn *channel.Channel, data map[string]string) error {

	log := logrus.WithFields(logrus.Fields{
		"channel_id":  cn.ID,
		"asterisk_id": cn.AsteriskID,
		"data":        cn.Data,
	})

	confContext := data["context"]
	switch confContext {
	case contextConfbridgeIncoming:
		return h.StartContextIncoming(cn, data)

	case contextConfbridgeOutgoing:
		log.Errorf("Currently, we don't support conference outgoing context. Something was wrong. context: %s", confContext)
		_ = h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return fmt.Errorf("unsupported conference context type. context: %s", confContext)

	default:
		log.Errorf("Unsuppurted context type. context: %s", confContext)
		_ = h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return fmt.Errorf("unsupported conference context type. context: %s", confContext)
	}
}

// ARIChannelLeftBridge handles ChannelLeftBridge ARI event for conference types.
func (h *confbridgeHandler) ARIChannelLeftBridge(cn *channel.Channel, br *bridge.Bridge) error {
	ctx := context.Background()

	if cn.Type != channel.TypeConfbridge {
		return nil
	}

	return h.Leaved(ctx, cn, br)
}

func (h *confbridgeHandler) ARIChannelEnteredBridge(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {

	if cn.Type != channel.TypeConfbridge {
		return nil
	}

	return h.Joined(ctx, cn, br)
}
