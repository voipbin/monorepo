package numberhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
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
	numType number.Type,
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
		"type":                  numType,
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
		Type:                numType,
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

// dbList returns list of numbers info of the given customer_id
func (h *numberHandler) dbList(ctx context.Context, pageSize uint64, pageToken string, filters map[number.Field]any) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "dbList",
		"page_size":  pageSize,
		"page_token": pageToken,
		"filters":    filters,
	})
	log.Debugf("GetNumbers.")

	numbers, err := h.db.NumberList(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get numbers. err:%v", err)
		return nil, err
	}
	log.WithField("numbers", numbers).Debugf("Found numbers info. count: %d", len(numbers))

	return numbers, nil
}

// dbUpdate updates a number with the given fields.
func (h *numberHandler) dbUpdate(ctx context.Context, id uuid.UUID, fields map[number.Field]any, eventType string) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "dbUpdate",
		"number_id": id,
		"fields":    fields,
	})
	log.Debugf("Updating number. number_id: %s", id)

	if err := h.db.NumberUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the number. number_id: %s, err:%v", id, err)
		return nil, errors.Wrapf(err, "could not update number")
	}

	res, err := h.db.NumberGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the updated number. number_id: %s, err: %v", id, err)
		return nil, errors.Wrapf(err, "could not get the updated number")
	}

	// Publish event based on event type
	switch eventType {
	case number.EventTypeNumberRenewed:
		h.notifyHandler.PublishEvent(ctx, eventType, res)
	default:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, eventType, res)
	}

	return res, nil
}

// dbListByTMRenew returns list of numbers info
func (h *numberHandler) dbListByTMRenew(ctx context.Context, tmRenew string) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "dbListByTMRenew",
		"tm_renew": tmRenew,
	})

	filters := map[number.Field]any{
		number.FieldDeleted: false,
	}

	res, err := h.db.NumberGetsByTMRenew(ctx, tmRenew, 100, filters)
	if err != nil {
		log.Errorf("Could not get numbers. tm_renew: %s, err:%v", tmRenew, err)
		return nil, err
	}

	return res, nil
}
