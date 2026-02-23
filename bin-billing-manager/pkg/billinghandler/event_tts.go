package billinghandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"

	"monorepo/bin-billing-manager/models/billing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventTTSSpeakingStarted handles the tts-manager's speaking_started event
func (h *billingHandler) EventTTSSpeakingStarted(ctx context.Context, s *tmspeaking.Speaking) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventTTSSpeakingStarted",
		"speaking_id": s.ID,
		"customer_id": s.CustomerID,
	})
	log.Debugf("Received speaking_started event. speaking_id: %s", s.ID)

	if errBilling := h.BillingStart(
		ctx,
		s.CustomerID,
		billing.ReferenceTypeSpeaking,
		s.ID,
		billing.CostTypeTTS,
		s.TMCreate,
		&commonaddress.Address{},
		&commonaddress.Address{},
	); errBilling != nil {
		return errors.Wrap(errBilling, "could not start a billing")
	}

	return nil
}

// EventTTSSpeakingStopped handles the tts-manager's speaking_stopped event
func (h *billingHandler) EventTTSSpeakingStopped(ctx context.Context, s *tmspeaking.Speaking) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventTTSSpeakingStopped",
		"speaking_id": s.ID,
	})
	log.Debugf("Received speaking_stopped event. speaking_id: %s", s.ID)

	b, err := h.GetByReferenceID(ctx, s.ID)
	if err != nil {
		// could not get billing. nothing to do.
		return nil
	}

	if s.TMUpdate == nil {
		return errors.Errorf("invalid tm_update. speaking_id: %s, tm_update: nil", s.ID)
	}

	if errEnd := h.BillingEnd(ctx, b, s.TMUpdate, &commonaddress.Address{}, &commonaddress.Address{}); errEnd != nil {
		return errors.Wrapf(errEnd, "could not end the billing. billing_id: %s, speaking_id: %s", b.ID, s.ID)
	}

	return nil
}
