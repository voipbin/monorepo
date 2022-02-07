package numberhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// CreateNumber creates a new order numbers of given numbers
func (h *numberHandler) CreateNumber(customerID, flowID uuid.UUID, num, name, detail string) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "CreateNumber",
			"customer_id": customerID,
			"flow_id":     flowID,
			"number":      num,
		},
	)
	log.Debugf("Creating a new number. customer_id: %s, number: %v", customerID, num)

	// use telnyx as a default
	res, err := h.numHandlerTelnyx.CreateOrderNumber(customerID, flowID, num, name, detail)
	if err != nil {
		log.Errorf("Could not create a number from the telnyx. err: %v", err)
		return nil, fmt.Errorf("could not create a number from the telnyx. err: %v", err)
	}

	return res, err
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

// UpdateBasicInfo updates the number
func (h *numberHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "UpdateBasicInfo",
			"number_id": id,
		},
	)
	log.Debugf("UpdateBasicInfo. number_id: %s", id)

	if err := h.db.NumberUpdateBasicInfo(ctx, id, name, detail); err != nil {
		log.Errorf("Could not set flow_id to number. number_id: %s, err:%v", id, err)
		return nil, err
	}

	res, err := h.db.NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the updated number. number_id: %s, err: %v", id, err)
		return nil, err
	}

	return res, nil
}

// UpdateFlowID updates the number's flow_id
func (h *numberHandler) UpdateFlowID(ctx context.Context, id, flowID uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "UpdateFlowID",
			"number_id": id,
			"flow_id":   flowID,
		},
	)
	log.Debugf("UpdateFlowID. number_id: %s", id)

	if err := h.db.NumberUpdateFlowID(ctx, id, flowID); err != nil {
		log.Errorf("Could not update the flow_id. number_id: %s, err:%v", id, err)
		return nil, err
	}

	res, err := h.db.NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the updated number. number_id: %s, err: %v", id, err)
		return nil, err
	}

	return res, nil
}
