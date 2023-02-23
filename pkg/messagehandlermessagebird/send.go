package messagehandlermessagebird

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
)

// SendMessage sends the message.
func (h *messageHandlerMessagebird) SendMessage(messageID uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "SendMessage",
		"message_id":   messageID,
		"customer_id":  customerID,
		"destinations": destinations,
	})

	sender := source.Target
	receivers := []string{}

	for _, destination := range destinations {
		receivers = append(receivers, destination.Target)
	}
	log.WithField("receivers", receivers).Debugf("Sending a messages by messagebird. message_id: %s, sender: %s", messageID, sender)

	// send a request to messaging providers
	m, err := h.requestExternal.MessagebirdSendMessage(sender, receivers, text)
	if err != nil {
		log.Errorf("Could not send message correctly to the messagebird. err: %v", err)
		return nil, err
	}

	res := m.ConvertMessage(messageID, customerID)
	return res, nil
}
