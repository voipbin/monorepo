package numberhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// CreateNumbers creates a new order numbers of given numbers
func (h *numberHandler) CreateNumbers(customerID uuid.UUID, numbers []string) ([]*number.Number, error) {
	logrus.Debugf("CreateNumbers. customer_id: %s, numbers: %v", customerID, numbers)

	// use telnyx as a default
	return h.numHandlerTelnyx.CreateOrderNumbers(customerID, numbers)
}

// CreateNumber creates a new order numbers of given numbers
func (h *numberHandler) CreateNumber(customerID uuid.UUID, number string) (*number.Number, error) {
	logrus.Debugf("CreateNumber. customer_id: %s, number: %v", customerID, number)

	// use telnyx as a default
	numbers := []string{number}
	tmpRes, err := h.numHandlerTelnyx.CreateOrderNumbers(customerID, numbers)
	if err != nil || len(tmpRes) == 0 {
		return nil, fmt.Errorf("could not create a number from the telnyx. err: %v", err)
	}

	return tmpRes[0], err
}

// ReleaseNumber release/deleted an existed ordered number
func (h *numberHandler) ReleaseNumber(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	logrus.Debugf("ReleaseNumber. number: %s", id)

	number, err := h.db.NumberGet(ctx, id)
	if err != nil {
		logrus.Errorf("Could not get order number info. number: %s, err: %v", id, err)
		return nil, err
	}

	return h.numHandlerTelnyx.ReleaseOrderNumber(ctx, number)
}

// GetNumberByNumber returns number info of the given number
func (h *numberHandler) GetNumberByNumber(ctx context.Context, num string) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"number_number": num,
		},
	)
	log.Debugf("GetNumberByNumber. number: %s", num)

	number, err := h.db.NumberGetByNumber(ctx, num)
	if err != nil {
		log.Errorf("Could not get number info. number: %s, err:%v", num, err)
		return nil, err
	}

	return number, nil
}

// GetNumber returns number info of the given id
func (h *numberHandler) GetNumber(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"number_id": id,
		},
	)
	log.Debugf("GetNumber. number: %s", id)

	number, err := h.db.NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. number: %s, err:%v", id, err)
		return nil, err
	}

	return number, nil
}

// GetNumbers returns list of numbers info of the given customer_id
func (h *numberHandler) GetNumbers(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"customer_id": customerID,
		},
	)
	log.Debugf("GetNumbers. customer_id: %s", customerID)

	if pageToken == "" {
		pageToken = getCurTime()
	}

	numbers, err := h.db.NumberGets(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get numbers. customer_id: %s, err:%v", customerID, err)
		return nil, err
	}
	log.WithField("numbers", numbers).Debugf("Found numbers info. count: %d", len(numbers))

	return numbers, nil
}

// UpdateNumber updates the number
func (h *numberHandler) UpdateNumber(ctx context.Context, numb *number.Number) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"number": numb,
		},
	)
	log.Debugf("UpdateNumber. number: %d", numb.ID)

	if err := h.db.NumberUpdate(ctx, numb); err != nil {
		log.Errorf("Could not set flow_id to number. number: %s, err:%v", numb.ID, err)
		return nil, err
	}

	res, err := h.db.NumberGet(ctx, numb.ID)
	if err != nil {
		log.Errorf("Could not get the updated number. number: %s, err: %v", numb.ID, err)
		return nil, err
	}

	return res, nil
}
