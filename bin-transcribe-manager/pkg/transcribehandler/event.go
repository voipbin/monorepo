package transcribehandler

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	cucustomer "monorepo/bin-customer-manager/models/customer"

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

// EventCMCallHangup handles the call-manager's call_hangup event
func (h *transcribeHandler) EventCMCallHangup(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "EventCMCallHangup",
		"call": c,
	})
	log.Debugf("Stopping all transcribes of the call. call_id: %s", c.ID)

	// get all transcribes of the call
	filters := map[string]string{
		"reference_id": c.ID.String(),
		"deleted":      "false",
	}
	transcribes, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not gets transcribes list. err: %v", err)
		return errors.Wrap(err, "could not get transcribes list")
	}

	// stop all transcribes
	for _, tr := range transcribes {
		log.Debugf("Stop transcribe info. transcribe_id: %s", tr.ID)
		tmp, err := h.Stop(ctx, tr.ID)
		if err != nil {
			log.Errorf("Could not stop transcribe info. err: %v", err)
			continue
		}
		log.WithField("transcribe", tmp).Debugf("Stopped transcribe info. transcribe_id: %s", tmp.ID)
	}

	return nil
}

// EventCMConfbridgeTerminated handles the call-manager's confbridge_terminated event
func (h *transcribeHandler) EventCMConfbridgeTerminated(ctx context.Context, c *cmconfbridge.Confbridge) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "EventCMConfbridgeTerminated",
		"confbridge": c,
	})
	log.Debugf("Stopping all transcribes of the confbridge. confbridge_id: %s", c.ID)

	// get all transcribes of the confbridge
	filters := map[string]string{
		"reference_id": c.ID.String(),
		"deleted":      "false",
	}
	transcribes, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not gets transcribes list. err: %v", err)
		return errors.Wrap(err, "could not get transcribes list")
	}

	// stop all transcribes
	for _, tr := range transcribes {
		log.Debugf("Stop transcribe info. transcribe_id: %s", tr.ID)
		tmp, err := h.Stop(ctx, tr.ID)
		if err != nil {
			log.Errorf("Could not stop transcribe info. err: %v", err)
			continue
		}
		log.WithField("transcribe", tmp).Debugf("Stopped transcribe info. transcribe_id: %s", tmp.ID)
	}

	return nil
}
