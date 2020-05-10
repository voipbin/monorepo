package arievent

import (
	"context"

	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
)

func (h *eventHandler) eventHandlerBridgeCreated(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.BridgeCreated)

	b := bridge.NewBridgeByBridgeCreated(e)
	if err := h.db.BridgeCreate(ctx, b); err != nil {
		return err
	}

	return nil
}

func (h *eventHandler) eventHandlerBridgeDestroyed(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.BridgeDestroyed)

	if err := h.db.BridgeEnd(ctx, e.Bridge.ID, string(e.Timestamp)); err != nil {
		return err
	}

	return nil
}
