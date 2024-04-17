package confbridgehandler

import (
	"context"

	"monorepo/bin-call-manager/models/confbridge"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Delete deletes the confbridge
func (h *confbridgeHandler) Delete(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Delete",
		"confbridge_id": id,
	})

	tmp, err := h.Terminating(ctx, id)
	if err != nil {
		log.Errorf("Could not terminating the confbridge. err: %v", err)
		return nil, errors.Wrap(err, "could not terminating the confbridge")
	}
	log.WithField("confbridge", tmp).Debugf("Terminating the confbridge. confbridge_id: %s", tmp.ID)

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the confbridge. err: %v", err)
		return nil, err
	}

	return res, nil
}
