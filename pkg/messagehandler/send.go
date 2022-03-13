package messagehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
)

// SendMessage sends the message.
func (h *messageHandler) SendMessage(ctx context.Context, customerID uuid.UUID, source *cmaddress.Address, destinations []cmaddress.Address, text string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SendMessage",
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

	id := uuid.Must(uuid.NewV4())
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

		TMCreate: dbhandler.GetCurTime(),
		TMUpdate: dbhandler.DefaultTimeStamp,
		TMDelete: dbhandler.DefaultTimeStamp,
	}

	// create a message
	_, err := h.Create(ctx, m)
	if err != nil {
		log.Errorf("Could not create a new message. err: %v", err)
		return nil, err
	}

	// todo: send webhook

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

	// todo: send webhook

	return res, nil
}
