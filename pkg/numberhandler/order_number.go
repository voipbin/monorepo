package numberhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
)

// CreateOrderNumbers creates a new order numbers of given numbers
func (h *numberHandler) CreateOrderNumbers(userID uint64, numbers []string) ([]*models.Number, error) {
	logrus.Debugf("CreateOrderNumbers. user_id: %d, numbers: %v", userID, numbers)

	// use telnyx as a default
	return h.numHandlerTelnyx.CreateOrderNumbers(userID, numbers)
}

// ReleaseOrderNumbers release/deleted an existed ordered number
func (h *numberHandler) ReleaseOrderNumbers(ctx context.Context, id uuid.UUID) (*models.Number, error) {
	logrus.Debugf("ReleaseOrderNumbers. number: %s", id)

	number, err := h.db.NumberGet(ctx, id)
	if err != nil {
		logrus.Errorf("Could not get order number info. number: %s, err: %v", id, err)
		return nil, err
	}

	return h.numHandlerTelnyx.ReleaseOrderNumber(ctx, number)
}

// GetOrderNumber returns number info of the given id
func (h *numberHandler) GetOrderNumber(ctx context.Context, id uuid.UUID) (*models.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"number_id": id,
		},
	)
	log.Debugf("GetOrderNumber. number: %s", id)

	number, err := h.db.NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. number: %s, err:%v", id, err)
		return nil, err
	}

	return number, nil
}
