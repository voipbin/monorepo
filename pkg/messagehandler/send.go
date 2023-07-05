package messagehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	bmbilling "gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
)

// Send sends the message.
func (h *messageHandler) Send(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Send",
		"customer_id": customerID,
	})

	// create targets
	targets := []target.Target{}
	for _, destination := range destinations {
		t := target.Target{
			Destination: destination,
			Status:      target.StatusQueued,
		}

		targets = append(targets, t)
	}

	// check the balance
	count := len(targets)
	valid, err := h.reqHandler.CustomerV1CustomerIsValidBalance(ctx, customerID, bmbilling.ReferenceTypeSMS, "", count)
	if err != nil {
		log.Errorf("Could not validate the customer's balance. err: %v", err)
		return nil, errors.Wrap(err, "could not validate the customer's balance")
	}
	if !valid {
		log.Errorf("Customer has not enough balance. customer_id: %s", customerID)
		return nil, errors.Wrap(err, "customer has not enough balance")
	}

	// select provider
	// currently, we use the messagebird only
	provider := message.ProviderNameMessagebird

	res, err := h.Create(ctx, id, customerID, source, targets, provider, text, message.DirectionOutbound)
	if err != nil {
		log.Errorf("Could not create a new message. err: %v", err)
		return nil, err
	}

	// send the message
	go func() {
		tmp, err := h.sendMessage(context.Background(), provider, res.ID, res.CustomerID, res.Source, res.Targets, res.Text)
		if err != nil {
			log.Errorf("Could not send the message correctly. err: %v", err)
			return
		}
		log.WithField("message", tmp).Debugf("Sent the message send request correctly. message_id: %s", id)
	}()

	return res, nil
}

// sendMessage sends the message to the destinations
func (h *messageHandler) sendMessage(ctx context.Context, providerName message.ProviderName, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "sendMessage",
		"id":          id,
		"customer_id": customerID,
		"source":      source,
		"targets":     targets,
	})

	if providerName != message.ProviderNameMessagebird {
		log.Errorf("Unsupported provider. provider: %s", providerName)
		return nil, fmt.Errorf("unsupported provider")
	}

	// send the message
	tmp, err := h.messageHandlerMessagebird.SendMessage(id, customerID, source, targets, text)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return nil, err
	}
	log.WithField("result", tmp).Debugf("Sent the message correctly.")

	// update the targets
	res, err := h.dbUpdateTargets(ctx, id, tmp)
	if err != nil {
		log.Errorf("Could not update the message targets. err: %v", err)
		return nil, err
	}

	return res, nil
}
