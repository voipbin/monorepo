package numberhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// Create creates a new order numbers of given numbers
func (h *numberHandler) Create(ctx context.Context, customerID uuid.UUID, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Create",
			"customer_id":   customerID,
			"flow_id":       callFlowID,
			"target_number": num,
		},
	)
	log.Debugf("Creating a new number. customer_id: %s, number: %v", customerID, num)

	// use telnyx as a default
	tmp, err := h.numberHandlerTelnyx.CreateNumber(customerID, num, callFlowID, name, detail)
	if err != nil {
		log.Errorf("Could not create a number from the telnyx. err: %v", err)
		return nil, fmt.Errorf("could not create a number from the telnyx. err: %v", err)
	}

	// add info
	tmp.ID = h.util.CreateUUID()
	tmp.CustomerID = customerID
	tmp.CallFlowID = callFlowID
	tmp.MessageFlowID = messageFlowID
	tmp.Name = name
	tmp.Detail = detail

	log.WithField("number", tmp).Debugf("Creating a number record. number: %s", tmp.Number)

	// insert into db
	if err := h.db.NumberCreate(ctx, tmp); err != nil {
		log.Errorf("Could not create a number. number: %s, err: %v", tmp.Number, err)
		return nil, err
	}

	// get created number
	res, err := h.db.NumberGet(ctx, tmp.ID)
	if err != nil {
		log.Errorf("Could not get created number info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, number.EventTypeNumberCreated, res)

	return res, err
}

// Delete release/deleted an existed ordered number
func (h *numberHandler) Delete(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Delete",
		"number_id": id,
	})
	log.Debugf("Deleting the number. number_id: %s", id)

	num, err := h.db.NumberGet(ctx, id)
	if err != nil {
		logrus.Errorf("Could not get order number info. number: %s, err: %v", id, err)
		return nil, err
	}
	log.Debugf("Deleting number info. number: %s", num.Number)

	// send delete request by provider
	switch num.ProviderName {
	case number.ProviderNameTelnyx:
		err = h.numberHandlerTelnyx.ReleaseNumber(ctx, num)

	default:
		err = fmt.Errorf("unsupported number provider. provider_name: %s", num.ProviderName)
	}

	if err != nil {
		log.Errorf("Could not release the number. err: %v", err)
	}

	// delete from the database
	if errDelete := h.db.NumberDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete the number from the db. number_id: %s, err: %v", id, errDelete)
		return nil, errDelete
	}

	// get deleted number
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted number. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, number.EventTypeNumberDeleted, res)

	return res, nil
}

// GetByNumber returns number info of the given number
func (h *numberHandler) GetByNumber(ctx context.Context, num string) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "GetByNumber",
			"number_number": num,
		},
	)
	log.Debugf("Getting a number by number. number: %s", num)

	number, err := h.db.NumberGetByNumber(ctx, num)
	if err != nil {
		log.Errorf("Could not get number info. number: %s, err:%v", num, err)
		return nil, err
	}

	return number, nil
}

// Get returns number info of the given id
func (h *numberHandler) Get(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "Get",
			"number_id": id,
		},
	)

	number, err := h.db.NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. number: %s, err:%v", id, err)
		return nil, err
	}

	return number, nil
}

// GetsByCustomerID returns list of numbers info of the given customer_id
func (h *numberHandler) GetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "GetsByCustomerID",
			"customer_id": customerID,
		},
	)
	log.Debugf("GetNumbers. customer_id: %s", customerID)

	if pageToken == "" {
		pageToken = h.util.GetCurTime()
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, number.EventTypeNumberUpdated, res)

	return res, nil
}

// UpdateFlowID updates the number's flow_id
func (h *numberHandler) UpdateFlowID(ctx context.Context, id, callFlowID, messageFlowID uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "UpdateFlowID",
			"number_id":       id,
			"call_flow_id":    callFlowID,
			"message_flow_id": messageFlowID,
		},
	)
	log.Debugf("UpdateFlowID. number_id: %s", id)

	if err := h.db.NumberUpdateFlowID(ctx, id, callFlowID, messageFlowID); err != nil {
		log.Errorf("Could not update the flow_id. number_id: %s, err:%v", id, err)
		return nil, err
	}

	res, err := h.db.NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the updated number. number_id: %s, err: %v", id, err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, number.EventTypeNumberUpdated, res)

	return res, nil
}
