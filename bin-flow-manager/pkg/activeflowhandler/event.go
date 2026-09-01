package activeflowhandler

import (
	"context"

	"monorepo/bin-flow-manager/models/activeflow"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *activeflowHandler) EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventCustomerDeleted",
		"customer_id": cu.ID,
	})
	log.Debugf("Stopping all activeflows of customer. customer_id: %s", cu.ID)

	// get all flows in customer
	filters := map[activeflow.Field]any{
		activeflow.FieldCustomerID: cu.ID,
		activeflow.FieldDeleted:    false,
	}
	afs, err := h.List(ctx, h.utilHandler.TimeGetCurTime(), 1000, filters)
	if err != nil {
		log.Errorf("Could not gets flows list. err: %v", err)
		return errors.Wrap(err, "could not get activeflows list")
	}

	// delete all activeflows
	for _, af := range afs {
		log.Debugf("Deleting activeflow info. activeflow_id: %s", af.ID)
		tmp, err := h.Delete(ctx, af.ID)
		if err != nil {
			log.Errorf("Could not delete activeflow info. err: %v", err)
			continue
		}
		log.WithField("activeflow", tmp).Debugf("Deleted activeflow info. activeflow_id: %s", tmp.ID)
	}

	return nil
}
