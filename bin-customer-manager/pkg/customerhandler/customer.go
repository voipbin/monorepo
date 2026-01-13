package customerhandler

import (
	"context"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Delete deletes the customer.
func (h *customerHandler) Delete(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Delete",
		"customer_id": id,
	})
	log.Debug("Deleteing the customer.")

	// get customer info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	if c.TMDelete != dbhandler.DefaultTimeStamp {
		// already deleted
		log.Infof("The customer already deleted. customer_id: %s", c.ID)
		return c, nil
	}

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the customer info. err: %v", err)
		return nil, err
	}

	return res, nil
}

func (h *customerHandler) validateCreate(ctx context.Context, email string) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":  "validateCreate",
		"email": email,
	})

	if !h.utilHandler.EmailIsValid(email) {
		log.Errorf("The email is invalid. email: %s", email)
		return false
	}

	// check customer
	filterCustomer := map[customer.Field]any{
		customer.FieldDeleted: false,
		customer.FieldEmail:   email,
	}
	tmps, err := h.Gets(ctx, 100, "", filterCustomer)
	if err != nil {
		log.Errorf("Could not get the customer info. err: %v", err)
		return false
	}
	if len(tmps) > 0 {
		log.Errorf("The email is already used. email: %s", email)
		return false
	}

	// check agent
	filterAgent := map[string]string{
		"deleted":  "false",
		"username": email,
	}
	tmpAgents, err := h.reqHandler.AgentV1AgentGets(ctx, "", 100, filterAgent)
	if err != nil {
		log.Errorf("Could not get the agent info. err: %v", err)
		return false
	}
	if len(tmpAgents) > 0 {
		log.Errorf("The email is already used. email: %s", email)
		return false
	}

	return true
}
