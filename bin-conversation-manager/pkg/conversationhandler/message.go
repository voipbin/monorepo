package conversationhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	mmmessage "monorepo/bin-message-manager/models/message"
	nmnumber "monorepo/bin-number-manager/models/number"
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

// MessageExecuteActiveflow creates and executes an activeflow for the given conversation.
func (h *conversationHandler) MessageExecuteActiveflow(ctx context.Context, cv *conversation.Conversation, m *message.Message, num *nmnumber.Number) (*fmactiveflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "MessageExecuteActiveflow",
		"conversation": cv,
		"message":      m,
	})

	// create activeflow
	res, err := h.reqHandler.FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		m.CustomerID,
		num.MessageFlowID,
		fmactiveflow.ReferenceTypeMessage,
		m.ID,
		uuid.Nil,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create an activeflow. message_id: %s, number_id: %s", m.ID, num.ID)
	}
	log.WithField("activeflow", res).Debugf("Created activeflow. activeflow_id: %s", res.ID)

	// set variables
	if errVariable := h.setVariables(ctx, res.ID, cv, m); errVariable != nil {
		return nil, errors.Wrapf(errVariable, "Could not set the variables. activeflow_id: %s", res.ID)
	}

	// execute the activeflow
	if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, res.ID); errExecute != nil {
		return nil, errors.Wrapf(errExecute, "Could not execute the activeflow. activeflow_id: %s", res.ID)
	}

	return res, nil
}

// MessageEventReceived is the handler for the received message event
func (h *conversationHandler) MessageEventReceived(ctx context.Context, m *mmmessage.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "MessageEventReceived",
		"message": m,
	})

	if len(m.Targets) > 1 {
		// something's wrong. but just log it
		log.Warnf("Received message with multiple targets. targets: %v", m.Targets)
	}

	// this is received message event handler
	// it's hard to imagine that the message is sent to multiple targets
	// but we need to handle it
	for _, target := range m.Targets {
		self := target.Destination
		peer := *m.Source

		// get or create a conversation
		cv, err := h.GetOrCreateBySelfAndPeer(
			ctx,
			m.CustomerID,
			conversation.TypeMessage,
			"", // because it's sms conversation, there is no dialog id
			self,
			peer,
		)
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

		// check the number info
		num, err := h.NumberGet(ctx, cv.Self.Target)
		if err != nil {
			return errors.Wrapf(err, "Could not get number info. number: %s", cv.Peer.Target)
		}
		log.WithField("number", num).Infof("Found number info. number_id: %s", num.ID)

		if num.MessageFlowID == uuid.Nil {
			// nothing to do. has no message flow id
			return nil
		}
		log.Debugf("The number has message flow id. number_id: %s, message_flow_id: %s", num.ID, num.MessageFlowID)

		af, err := h.MessageExecuteActiveflow(ctx, cv, m, num)
		if err != nil {
			return errors.Wrapf(err, "Could not execute the activeflow. message_id: %s, number_id: %s", m.ID, num.ID)
		}
		log.WithField("activeflow", af).Debugf("Executed activeflow. activeflow_id: %s", af.ID)
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

		// get or create a conversation
		cv, err := h.GetOrCreateBySelfAndPeer(
			ctx,
			m.CustomerID,
			conversation.TypeMessage,
			"", // because it's sms conversation, there is no dialog id
			self,
			peer,
		)
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
