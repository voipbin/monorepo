package messagehandlermessagebird

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
)

// SendMessage sends the message.
func (h *messageHandlerMessagebird) SendMessage(messageID uuid.UUID, customerID uuid.UUID, source *cmaddress.Address, destinations []cmaddress.Address, text string) (*message.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "SendMessage",
			"message_id":  messageID,
			"customer_id": customerID,
		},
	)

	sender := source.Target
	receivers := []string{}
	log.Debugf("Sending a message by messagebird. message_id: %s, sender: %s", messageID, sender)

	for _, destination := range destinations {
		receivers = append(receivers, destination.Target)
	}

	// send a request to messaging providers
	m, err := h.requestExternal.MessagebirdSendMessage(sender, receivers, text)
	if err != nil {
		log.Errorf("Could not send message correctly to the messagebird. err: %v", err)
		return nil, err
	}

	res := m.ConvertMessage(messageID, customerID)
	return res, nil
}
