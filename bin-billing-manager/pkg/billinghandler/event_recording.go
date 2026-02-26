package billinghandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	cmrecording "monorepo/bin-call-manager/models/recording"

	"monorepo/bin-billing-manager/models/billing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCMRecordingStarted handles the call-manager's recording_started event
func (h *billingHandler) EventCMRecordingStarted(ctx context.Context, r *cmrecording.Recording) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "EventCMRecordingStarted",
		"recording_id": r.ID,
		"customer_id":  r.CustomerID,
	})
	log.Debugf("Received recording_started event. recording_id: %s", r.ID)

	if errBilling := h.BillingStart(
		ctx,
		r.CustomerID,
		billing.ReferenceTypeRecording,
		r.ID,
		billing.CostTypeRecording,
		r.TMStart,
		&commonaddress.Address{},
		&commonaddress.Address{},
	); errBilling != nil {
		return errors.Wrap(errBilling, "could not start a billing")
	}

	return nil
}

// EventCMRecordingFinished handles the call-manager's recording_finished event
func (h *billingHandler) EventCMRecordingFinished(ctx context.Context, r *cmrecording.Recording) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "EventCMRecordingFinished",
		"recording_id": r.ID,
	})
	log.Debugf("Received recording_finished event. recording_id: %s", r.ID)

	b, err := h.GetByReferenceID(ctx, r.ID)
	if err != nil {
		// could not get billing. nothing to do.
		return nil
	}

	if r.TMEnd == nil {
		return errors.Errorf("invalid tm_end. recording_id: %s, tm_end: nil", r.ID)
	}

	if errEnd := h.BillingEnd(ctx, b, r.TMEnd, &commonaddress.Address{}, &commonaddress.Address{}); errEnd != nil {
		return errors.Wrapf(errEnd, "could not end the billing. billing_id: %s, recording_id: %s", b.ID, r.ID)
	}

	return nil
}
