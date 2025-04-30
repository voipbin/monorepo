package conversationhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	mmmessage "monorepo/bin-message-manager/models/message"
)

// MessageSend sends the message
func (h *conversationHandler) MessageSend(ctx context.Context, conversationID uuid.UUID, text string, medias []media.Media) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "MessageSend",
		"conversation_id": conversationID,
		"text":            text,
		"medias":          medias,
	})
	log.Debugf("MessageSend detail. conversation_id: %s", conversationID)

	// get conversation
	cv, err := h.Get(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation. err: %v", err)
		return nil, err
	}

	// send to conversation
	m, err := h.messageHandler.Send(ctx, cv, text, medias)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return nil, err
	}

	return m, nil
}

func (h *conversationHandler) GetBySelfAndPeerOrCreate(
	ctx context.Context,
	customerID uuid.UUID,
	self commonaddress.Address,
	peer commonaddress.Address,
) (*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "GetBySelfAndPeerOrCreate",
		"self": self,
		"peer": peer,
	})

	res, err := h.GetBySelfAndPeer(ctx, self, peer)
	if err != nil {
		log.WithFields(logrus.Fields{
			"self": self,
			"peer": peer,
		}).Debugf("Could not find conversation. Create a new conversation. err: %v", err)

		res, err = h.Create(
			ctx,
			customerID,
			"conversation with "+peer.TargetName,
			"conversation with "+peer.TargetName,
			conversation.TypeMessage,
			"", // because it's sms conversation, there is no dialog id
			self,
			peer,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not create a new conversation")
		}
		log.WithField("conversation", res).Debugf("Created a new conversation. conversation_id: %s", res.ID)
	}

	return res, nil
}

// MessageSend sends the message
func (h *conversationHandler) MessageEventReceived(ctx context.Context, m *mmmessage.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "MessageEventReceived",
		"message": m,
	})

	for _, target := range m.Targets {
		self := target.Destination
		peer := *m.Source

		// get conversation
		cv, err := h.GetBySelfAndPeerOrCreate(ctx, m.CustomerID, self, peer)
		if err != nil {
			return errors.Wrapf(err, "Could not get conversation")
		}
		log.WithField("conversation", cv).Debugf("Found conversation. conversation_id: %s", cv.ID)

		// create a new conversation message
		m, err := h.messageHandler.Create(
			ctx,
			m.ID,
			cv.CustomerID,
			cv.ID,
			message.DirectionIncoming,
			message.StatusDone,
			message.ReferenceTypeMessage,
			m.ID,
			"",
			m.Text,
			[]media.Media{},
		)
		if err != nil {
			return errors.Wrapf(err, "Could not create a message")
		}
		log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
	}

	return nil
}

// MessageSend sends the message
func (h *conversationHandler) MessageEventSent(ctx context.Context, m *mmmessage.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "MessageEventSent",
		"message": m,
	})

	for _, target := range m.Targets {
		self := *m.Source
		peer := target.Destination

		// get conversation
		cv, err := h.GetBySelfAndPeerOrCreate(ctx, m.CustomerID, self, peer)
		if err != nil {
			return errors.Wrapf(err, "Could not get conversation")
		}
		log.WithField("conversation", cv).Debugf("Found conversation. conversation_id: %s", cv.ID)

		// create a new conversation message
		tmp, err := h.messageHandler.Get(ctx, m.ID)
		if err != nil {
			m, err := h.messageHandler.Create(
				ctx,
				m.ID,
				cv.CustomerID,
				cv.ID,
				message.DirectionOutgoing,
				message.StatusDone,
				message.ReferenceTypeMessage,
				m.ID,
				"",
				m.Text,
				[]media.Media{},
			)
			if err != nil {
				return errors.Wrapf(err, "Could not create a message")
			}
			log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
		} else {
			log.WithField("message", tmp).Debugf("Found message. Updating the message status. message_id: %s", tmp.ID)
			updated, err := h.messageHandler.UpdateStatus(ctx, tmp.ID, message.StatusDone)
			if err != nil {
				return errors.Wrapf(err, "Could not update the message")
			}

			log.WithField("message", updated).Debugf("Updated message. message_id: %s", updated.ID)
		}
	}

	return nil
}
