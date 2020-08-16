package eventhandler

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/bridge"
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

	log := log.WithFields(
		log.Fields{
			"bridge":   e.Bridge.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
		})

	if h.db.BridgeIsExist(e.Bridge.ID, defaultExistTimeout) == false {
		log.Error("The given bridge is not in our database.")
		return fmt.Errorf("no bridge found")
	}

	if err := h.db.BridgeEnd(ctx, e.Bridge.ID, string(e.Timestamp)); err != nil {
		return err
	}

	return nil
}
