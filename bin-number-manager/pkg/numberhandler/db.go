package numberhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-number-manager/models/number"
)

// dbCreate creates a new order numbers of given numbers
func (h *numberHandler) dbCreate(
	ctx context.Context,
	customerID uuid.UUID,
	num string,
	callFlowID uuid.UUID,
	messageFlowID uuid.UUID,
	name string,
	detail string,
	providerName number.ProviderName,
	providerReferenceID string,
	status number.Status,
	t38Enabled bool,
	emergencyEnabled bool,
) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                  "dbCreate",
		"customer_id":           customerID,
		"target_number":         num,
		"call_flow_id":          callFlowID,
		"message_flow_id":       messageFlowID,
		"name":                  name,
		"detail":                detail,
		"provider_name":         providerName,
		"provider_reference_id": providerReferenceID,
		"status":                status,
		"t38_enabled":           t38Enabled,
		"emergency_enabled":     emergencyEnabled,
	})
	log.Debugf("Creating a new number. customer_id: %s, number: %v", customerID, num)

	tmp := &number.Number{
		Identity: commonidentity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: customerID,
		},
		Number:              num,
		CallFlowID:          callFlowID,
		MessageFlowID:       messageFlowID,
		Name:                name,
		Detail:              detail,
		ProviderName:        providerName,
		ProviderReferenceID: providerReferenceID,
		Status:              status,
		T38Enabled:          t38Enabled,
		EmergencyEnabled:    emergencyEnabled,
	}
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, number.EventTypeNumberCreated, res)

	return res, err
}

// dbDelete release/deleted an existed ordered number
func (h *numberHandler) dbDelete(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "dbDelete",
		"number_id": id,
	})
	log.Debugf("Deleting the number. number_id: %s", id)

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

// dbGet returns number info of the given id
func (h *numberHandler) dbGet(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "dbGet",
		"number_id": id,
	})

	number, err := h.db.NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. number: %s, err:%v", id, err)
		return nil, err
	}

	return number, nil
}

// dbGets returns list of numbers info of the given customer_id
func (h *numberHandler) dbGets(ctx context.Context, pageSize uint64, pageToken string, filters map[string]string) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "dbGets",
		"page_size":  pageSize,
		"page_token": pageToken,
		"filters":    filters,
	})
	log.Debugf("GetNumbers.")

	numbers, err := h.db.NumberGets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get numbers. err:%v", err)
		return nil, err
	}
	log.WithField("numbers", numbers).Debugf("Found numbers info. count: %d", len(numbers))

	return numbers, nil
}

// dbUpdateInfo updates the number
func (h *numberHandler) dbUpdateInfo(ctx context.Context, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "dbUpdateInfo",
		"number_id":       id,
		"call_flow_id":    callFlowID,
		"message_flow_id": messageFlowID,
		"name":            name,
		"detail":          detail,
	})
	log.Debugf("Updating the number info. number_id: %s", id)

	if err := h.db.NumberUpdateInfo(ctx, id, callFlowID, messageFlowID, name, detail); err != nil {
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

// dbUpdateFlowID updates the number's flow_id
func (h *numberHandler) dbUpdateFlowID(ctx context.Context, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "dbUpdateFlowID",
		"number_id":       id,
		"call_flow_id":    callFlowID,
		"message_flow_id": messageFlowID,
	})
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

// dbUpdateRenew updates the number's tm_renew
func (h *numberHandler) dbUpdateRenew(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "dbUpdateRenew",
		"number_id": id,
	})
	log.Debugf("UpdateRenew. number_id: %s", id)

	if err := h.db.NumberUpdateTMRenew(ctx, id); err != nil {
		log.Errorf("Could not update the tm_renew. number_id: %s, err:%v", id, err)
		return nil, err
	}

	res, err := h.db.NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the updated number. number_id: %s, err: %v", id, err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, number.EventTypeNumberRenewed, res)

	return res, nil
}

// dbGetsByTMRenew returns list of numbers info
func (h *numberHandler) dbGetsByTMRenew(ctx context.Context, tmRenew string) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "dbGetsByTMRenew",
		"tm_renew": tmRenew,
	})

	filters := map[string]string{
		"deleted": "false",
	}

	res, err := h.db.NumberGetsByTMRenew(ctx, tmRenew, 100, filters)
	if err != nil {
		log.Errorf("Could not get numbers. tm_renew: %s, err:%v", tmRenew, err)
		return nil, err
	}

	return res, nil
}
