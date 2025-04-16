package smshandler

import (
	"context"
	"encoding/json"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/message"
)

// Event handles received sms/mms message
func (h *smsHandler) Event(ctx context.Context, data []byte) ([]*message.Message, *commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Event",
		"data": data,
	})

	// parse the message
	var m mmmessage.Message
	if errUnmarshal := json.Unmarshal(data, &m); errUnmarshal != nil {
		log.Errorf("Could not handle the event message. err: %v", errUnmarshal)
		return nil, nil, errUnmarshal
	}

	// get local address
	localAddr := m.Source
	if m.Direction == mmmessage.DirectionInbound {
		localAddr = &m.Targets[0].Destination
	}

	status := message.StatusDone
	if m.Direction == mmmessage.DirectionOutbound {
		status = message.StatusDone
	}

	res := []*message.Message{}
	for i := range m.Targets {

		referenceID := h.getReferenceID(&m, i)
		log.Debugf("Found reference id. reference_id: %s", referenceID)

		// create a message
		tmp := &message.Message{
			Identity: commonidentity.Identity{
				ID:         uuid.Nil,
				CustomerID: m.CustomerID,
			},

			ConversationID: uuid.Nil,
			Status:         status,

			ReferenceType: message.ReferenceTypeMessage,
			ReferenceID:   referenceID,

			TransactionID: m.ID.String(),

			Text: m.Text,
		}

		res = append(res, tmp)
	}

	return res, localAddr, nil
}

// getReferenceID returns a reference id
func (h *smsHandler) getReferenceID(m *mmmessage.Message, idx int) string {

	if m.Direction == mmmessage.DirectionInbound {
		return m.Source.Target
	}

	return m.Targets[idx].Destination.Target
}
