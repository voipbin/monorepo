package numberhandler

import (
	"context"

	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
)

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *numberHandler) EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": cu,
	})

	// get all numbers of the given customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}

	nbs, err := h.dbGets(ctx, 10000, "", filters)
	if err != nil {
		log.Errorf("Could not get numbers. err: %v", err)
		return errors.Wrap(err, "could not get numbers")
	}
	log.WithField("numbers", nbs).Infof("Found numbers. len: %v", len(nbs))

	for _, nb := range nbs {
		log.WithField("number", nb).Debugf("Deleting number. number_id: %s", nb.ID)

		tmp, err := h.Delete(ctx, nb.ID)
		if err != nil {
			// we could not delete the number, but we need to keep it around
			log.Errorf("Could not delete the number. err: %v", err)
			continue
		}
		log.WithField("number", tmp).Infof("Deleted number. number_id: %s", tmp.ID)
	}

	return nil
}

// EventFlowDeleted handles the flow-manager's flow_deleted event
func (h *numberHandler) EventFlowDeleted(ctx context.Context, f *fmflow.Flow) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "EventFlowDeleted",
		"flow": f,
	})
	log.Debugf("Removing the flow id from the all numbers. flow_id: %s", f.ID)

	// removing call_flow_id
	for {
		ts := h.utilHandler.TimeGetCurTime()
		filters := map[string]string{
			"call_flow_id": f.ID.String(),
			"deleted":      "false",
		}
		numbs, err := h.db.NumberGets(ctx, 1000, ts, filters)
		if err != nil || len(numbs) <= 0 {
			break
		}

		for _, numb := range numbs {
			log.WithField("number", numb).Debugf("Removing call_flow_id from the number. number_id: %s, number_number: %s", numb.ID, numb.Number)
			if err := h.db.NumberUpdateCallFlowID(ctx, numb.ID, uuid.Nil); err != nil {
				log.Errorf("Could not remove flow_id. err: %v", err)
			}
		}

		if len(numbs) < 100 {
			// no more list
			break
		}
	}

	// removing message_flow_id
	for {
		ts := h.utilHandler.TimeGetCurTime()
		filters := map[string]string{
			"message_flow_id": f.ID.String(),
			"deleted":         "false",
		}
		numbs, err := h.db.NumberGets(ctx, 1000, ts, filters)
		if err != nil || len(numbs) <= 0 {
			break
		}

		for _, numb := range numbs {
			log.WithField("number", numb).Debugf("Removing message_flow_id from the number. number_id: %s, number_number: %s", numb.ID, numb.Number)
			if err := h.db.NumberUpdateMessageFlowID(ctx, numb.ID, uuid.Nil); err != nil {
				log.Errorf("Could not remove flow_id. err: %v", err)
			}
		}

		if len(numbs) < 100 {
			// no more list
			break
		}
	}

	return nil
}
