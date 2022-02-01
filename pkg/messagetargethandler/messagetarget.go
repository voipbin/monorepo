package messagetargethandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/messagetarget"
)

// Get returns given customer id's messagetarget.
func (h *messagetargetHandler) Get(ctx context.Context, customerID uuid.UUID) (*messagetarget.MessageTarget, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Get",
			"customer_id": customerID,
		})

	res, err := h.db.MessageTargetGet(ctx, customerID)
	if err == nil {
		return res, nil
	}

	tmp, err := h.reqHandler.CSV1CustomerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	// create and update the messagetarget
	res = messagetarget.CreateMessageTargetFromCustomer(tmp)
	if errUpdate := h.Update(ctx, res); errUpdate != nil {
		// we couldn't update the message target, but keep going because we've got customer info
		log.Errorf("Could not update the message target. err: %v", errUpdate)
	}

	return res, nil
}

// Update updates the messagetarget
func (h *messagetargetHandler) Update(ctx context.Context, m *messagetarget.MessageTarget) error {
	return h.db.MessageTargetSet(ctx, m)
}

// UpdateByCustomer updates the messagetarget by customer and return the updated messagetarget
func (h *messagetargetHandler) UpdateByCustomer(ctx context.Context, m *cscustomer.Customer) (*messagetarget.MessageTarget, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "UpdateByCustomer",
			"customer_id": m.ID,
		})

	res := messagetarget.CreateMessageTargetFromCustomer(m)
	if errUpdate := h.Update(ctx, res); errUpdate != nil {
		// we couldn't update the message target, but keep going because we've got customer info
		log.Errorf("Could not update the message target. err: %v", errUpdate)
	}

	return res, nil
}
