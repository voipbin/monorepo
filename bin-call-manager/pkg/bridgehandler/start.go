package bridgehandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/bridge"
)

func (h *bridgeHandler) Start(ctx context.Context, asteriskID string, bridgeID string, bridgeName string, bridgeType []bridge.Type) (*bridge.Bridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Start",
		"asterisk_id": asteriskID,
		"bridge_id":   bridgeID,
	})

	if errCreate := h.reqHandler.AstBridgeCreate(ctx, asteriskID, bridgeID, bridgeName, bridgeType); errCreate != nil {
		log.Errorf("Could not create a bridge. err: %v", errCreate)
		return nil, errors.Wrap(errCreate, "could not create a bridge")
	}

	res, err := h.Get(ctx, bridgeID)
	if err != nil {
		log.Errorf("Could not get created bridge info. err: %v", err)
		return nil, errors.Wrap(err, "could not get started bridge info")
	}

	return res, nil
}
