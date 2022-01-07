package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// Get returns confbridge
func (h *confbridgeHandler) Get(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Get",
		},
	)

	// create confbridge
	res, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the confbridge. err: %v", err)
		return nil, err
	}

	return res, nil
}
