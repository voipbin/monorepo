package smshandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Setup sets up the given customer's line webhook
func (h *smsHandler) Setup(ctx context.Context, customerID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Setup",
		"customer_id": customerID,
	})
	log.Debugf("Setting up the sms.")

	return nil
}
