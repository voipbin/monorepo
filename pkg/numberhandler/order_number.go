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

// CreateOrderNumbers creates a new order numbers of given numbers
func (h *numberHandler) CreateOrderNumber(userID uint64, number string) (*models.Number, error) {
	logrus.Debugf("CreateOrderNumber. user_id: %d, number: %v", userID, number)

	// use telnyx as a default
	numbers := []string{number}
	tmpRes, err := h.numHandlerTelnyx.CreateOrderNumbers(userID, numbers)
	if err != nil || len(tmpRes) == 0 {
		return nil, err
	}

	return tmpRes[0], err
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

// GetOrderNumberByNumber returns number info of the given number
func (h *numberHandler) GetOrderNumberByNumber(ctx context.Context, num string) (*models.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"number_number": num,
		},
	)
	log.Debugf("GetOrderNumberByNumber. number: %s", num)

	number, err := h.db.NumberGetByNumber(ctx, num)
	if err != nil {
		log.Errorf("Could not get number info. number: %s, err:%v", num, err)
		return nil, err
	}

	return number, nil
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

// GetOrderNumbers returns list of numbers info of the given user_id
func (h *numberHandler) GetOrderNumbers(ctx context.Context, userID uint64, pageSize uint64, pageToken string) ([]*models.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"user_id": userID,
		},
	)
	log.Debugf("GetOrderNumbers. user_id: %d", userID)

	numbers, err := h.db.NumberGets(ctx, userID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get numbers. user_id: %d, err:%v", userID, err)
		return nil, err
	}

	return numbers, nil
}
