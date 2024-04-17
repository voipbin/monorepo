package bridgehandler

import (
	"context"

	"github.com/sirupsen/logrus"
)

// IsExist checks the given bridge is actually exist
func (h *bridgeHandler) IsExist(ctx context.Context, id string) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":      "IsExist",
		"bridge_id": id,
	})

	tmp, err := h.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get bridge info. err: %v", err)
		return false
	}

	_, err = h.reqHandler.AstBridgeGet(ctx, tmp.AsteriskID, tmp.ID)
	if err != nil {
		log.Debugf("Could not get bridge info from the asterisk. Consider bridge has deleted. Update bridge delete. bridge_id: %s", tmp.ID)
		_, _ = h.Delete(ctx, tmp.ID)
		return false
	}

	return true
}
