package messagehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
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

	if id == uuid.Nil {
		id = h.utilHandler.CreateUUID()
	}
	m := &message.Message{
		ID:         id,
		CustomerID: customerID,
		Type:       message.TypeSMS,

		Source:  source,
		Targets: targets,

		ProviderName: message.ProviderNameMessagebird,

		Text:      text,
		Medias:    []string{},
		Direction: message.DirectionOutbound,
	}

	// create a message
	res, err := h.Create(ctx, m)
	if err != nil {
		log.Errorf("Could not create a new message. err: %v", err)
		return nil, err
	}

	go func() {
		tmp, err := h.sendMessage(ctx, id, customerID, source, destinations, text)
		if err != nil {
			log.Errorf("Could not send the message correctly. err: %v", err)
			return
		}
		log.WithField("message", tmp).Debugf("Sent the message send request correctly. message_id: %s", id)
	}()

	return res, nil
}

// sendMessage sends the message to the destinations
func (h *messageHandler) sendMessage(ctx context.Context, id, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "sendMessage",
		"id":           id,
		"customer_id":  customerID,
		"source":       source,
		"destinations": destinations,
	})

	// send the message
	tmp, err := h.messageHandlerMessagebird.SendMessage(id, customerID, source, destinations, text)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return nil, err
	}
	log.WithField("result", tmp).Debugf("Sent the message correctly.")

	// update the targets
	res, err := h.UpdateTargets(ctx, id, tmp.Targets)
	if err != nil {
		log.Errorf("Could not update the message targets. err: %v", err)
		return nil, err
	}

	return res, nil
}
