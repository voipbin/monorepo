package arieventhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
)

// EventHandlerBridgeCreated handles BridgeCreated ari event.
func (h *eventHandler) EventHandlerBridgeCreated(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.BridgeCreated)
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerBridgeCreated",
		"event": e,
	})

	// get reference_type
	mapParse := bridge.ParseBridgeName(e.Bridge.Name)

	referenceType := bridge.ReferenceTypeUnknown
	if mapParse["reference_type"] != "" {
		referenceType = bridge.ReferenceType(mapParse["reference_type"])
	}
	referenceID := uuid.FromStringOrNil(mapParse["reference_id"])

	br, err := h.bridgeHandler.Create(
		ctx,
		e.AsteriskID,
		e.Bridge.ID,
		e.Bridge.Name,
		bridge.Type(e.Bridge.BridgeType),
		bridge.Tech(e.Bridge.Technology),
		e.Bridge.BridgeClass,
		e.Bridge.Creator,
		e.Bridge.VideoMode,
		e.Bridge.VideoSourceID,
		referenceType,
		referenceID,
	)
	if err != nil {
		log.Errorf("Could not create a new bridge. err: %v", err)
		return err
	}
	log.WithField("bridge", br).Debugf("Created a new bridge. bridge_id: %s", br.ID)

	return nil
}

// EventHandlerBridgeDestroyed handles BridgeDestroyed ari event.
func (h *eventHandler) EventHandlerBridgeDestroyed(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.BridgeDestroyed)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerBridgeDestroyed",
		"event": e,
	})

	tmp, err := h.bridgeHandler.Get(ctx, e.Bridge.ID)
	if err != nil {
		log.Error("The given bridge is not in our database.")
		return fmt.Errorf("no bridge found")
	}

	br, err := h.bridgeHandler.Delete(ctx, tmp.ID)
	if err != nil {
		log.Errorf("Coudl not delete the bridge. err: %v", err)
		return err
	}
	log.WithField("bridge", br).Debugf("Deleted bridge. bridge_id: %s", br.ID)

	if errDestoryed := h.confbridgeHandler.ARIBridgeDestroyed(ctx, br); errDestoryed != nil {
		log.Errorf("Could not handle the event by the confbridgehandler. err: %v", errDestoryed)
		return errors.Wrap(errDestoryed, "could not handle the event by the confbridgehandler")
	}

	return nil
}
