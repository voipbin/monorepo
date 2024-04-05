package transcribehandler

import (
	"context"

	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event
func (h *transcribeHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all transcribes of the customer. customer_id: %s", cu.ID)

	// get all transcribes of the customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	transcribes, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not gets transcribes list. err: %v", err)
		return errors.Wrap(err, "could not get transcribes list")
	}

	// delete all transcribes
	for _, tr := range transcribes {
		log.Debugf("Deleting transcribe info. transcribe_id: %s", tr.ID)
		tmp, err := h.Delete(ctx, tr.ID)
		if err != nil {
			log.Errorf("Could not delete transcribe info. err: %v", err)
			continue
		}
		log.WithField("transcribe", tmp).Debugf("Deleted transcribe info. transcribe_id: %s", tmp.ID)
	}

	return nil
}
