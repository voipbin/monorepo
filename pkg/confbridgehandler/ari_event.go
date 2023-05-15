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
