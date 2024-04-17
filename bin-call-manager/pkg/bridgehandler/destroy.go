package bridgehandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Destroy destroys the bridge.
// the channels in the bridge will be kicked out from the bridge automatically by the asterisk.
func (h *bridgeHandler) Destroy(ctx context.Context, id string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Destroy",
		"bridge_id": id,
	})

	tmp, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get bridge info. err: %v", err)
		return errors.Wrap(err, "could not get bridge info")
	}

	if err := h.reqHandler.AstBridgeDelete(ctx, tmp.AsteriskID, tmp.ID); err != nil {
		// we don't care the error here. just write the log.
		log.Errorf("Could not remove the bridge. err: %v", err)
		return errors.Wrap(err, "could not remove the bridge")
	}

	return nil
}
