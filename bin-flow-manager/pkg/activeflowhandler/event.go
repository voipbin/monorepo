package activeflowhandler

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCallHangup handles the call-manager's call_hangup event
func (h *activeflowHandler) EventCallHangup(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "EventCallHangup",
		"call": c,
	})
	log.Debugf("Handling the call_hangup event. call_id: %s", c.ID)

	// stop the activeflow
	tmp, err := h.Stop(ctx, c.ActiveflowID)
	if err != nil {
		log.Errorf("Could not stop the activeflow. err: %v", err)
		return errors.Wrap(err, "Could not stop the activeflow.")
	}
	log.WithField("activeflow", tmp).Debugf("Stopped activeflow. activeflow_id: %s", tmp.ID)

	return nil
}

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *activeflowHandler) EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Stopping all activeflows of customer. customer_id: %s", cu.ID)

	// get all flows in customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	afs, err := h.Gets(ctx, h.utilHandler.TimeGetCurTime(), 1000, filters)
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
