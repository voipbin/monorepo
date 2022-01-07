package arieventhandler

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
)

// EventHandlerBridgeCreated handles BridgeCreated ari event.
func (h *eventHandler) EventHandlerBridgeCreated(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.BridgeCreated)

	b := bridge.NewBridgeByBridgeCreated(e)

	b.TMUpdate = defaultTimeStamp
	b.TMDelete = defaultTimeStamp
	if err := h.db.BridgeCreate(ctx, b); err != nil {
		return err
	}

	return nil
}

// EventHandlerBridgeDestroyed handles BridgeDestroyed ari event.
func (h *eventHandler) EventHandlerBridgeDestroyed(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.BridgeDestroyed)

	log := log.WithFields(
		log.Fields{
			"func":     "EventHandlerBridgeDestroyed",
			"bridge":   e.Bridge.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
		})

	if !h.db.BridgeIsExist(e.Bridge.ID, defaultExistTimeout) {
		log.Error("The given bridge is not in our database.")
		return fmt.Errorf("no bridge found")
	}

	if err := h.db.BridgeEnd(ctx, e.Bridge.ID, string(e.Timestamp)); err != nil {
		return err
	}

	return nil
}
