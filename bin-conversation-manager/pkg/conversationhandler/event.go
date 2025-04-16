package conversationhandler

import (
	"context"
	"encoding/json"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

// Event returns list of messages of the given conversation
func (h *conversationHandler) Event(ctx context.Context, referenceType conversation.ReferenceType, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Event",
		"reference_type": referenceType,
		"data":           data,
	})

	switch referenceType {
	case conversation.ReferenceTypeMessage:
		if err := h.eventSMS(ctx, data); err != nil {
			log.Errorf("Could not handle the sms type event. err: %v", err)
			return err
		}

	default:
		log.Errorf("Could not find reference type handler. reference_type: %s", referenceType)
		return fmt.Errorf("reference type handler not found")
	}

	return nil
}

// eventSMS handle the sms type of hook message
func (h *conversationHandler) eventSMS(ctx context.Context, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "eventSMS",
	})

	mm := mmmessage.Message{}
	if err := json.Unmarshal(data, &mm); err != nil {
		return errors.Wrapf(err, "Could not unmarshal the data")
	}

	var self *commonaddress.Address
	var peer *commonaddress.Address
	var direction message.Direction

	for _, target := range mm.Targets {
		if mm.Direction == mmmessage.DirectionInbound {
			self = &target.Destination
			peer = mm.Source
			direction = message.DirectionIncoming
		} else {
			self = mm.Source
			peer = &target.Destination
			direction = message.DirectionOutgoing
		}

		// get conversation
		cv, err := h.GetBySelfAndPeer(ctx, self, peer)
		if err != nil {
			log.Debugf("Could not find conversation. Create a new conversation.")

			// create a new conversation
			cv, err = h.Create(
				ctx,
				mm.CustomerID,
				"conversation",
				"conversation detail",
				conversation.ReferenceTypeMessage,
				mm.ID.String(),
				self,
				peer,
			)
			if err != nil {
				return errors.Wrapf(err, "Could not create a new conversation")
			}
			log.WithField("conversation", cv).Debugf("Created a new conversation. conversation_id: %s", cv.ID)
		}
		log.WithField("conversation", cv).Debugf("Found conversation. conversation_id: %s", cv.ID)

		m, err := h.messageHandler.Create(
			ctx,
			cv.CustomerID,
			cv.ID,
			direction,
			message.StatusDone,
			conversation.ReferenceTypeMessage,
			mm.ID.String(),
			"",
			mm.Text,
			[]media.Media{},
		)
		if err != nil {
			return errors.Wrapf(err, "Could not create a message")
		}
		log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
	}

	return nil

	// tmps, localAddr, err := h.smsHandler.Event(ctx, data)
	// if err != nil {
	// 	log.Errorf("Could not handler the sms message. err: %v", err)
	// 	return err
	// }
	// log.WithField("local_address", localAddr).Debugf("Found local address. address_target: %s", localAddr.Target)

	// // get messages
	// transRes, err := h.messageHandler.GetsByTransactionID(ctx, tmps[0].TransactionID, h.utilHandler.TimeGetCurTime(), 10)
	// if err != nil {
	// 	log.Errorf("Could not get messages. err: %v", err)
	// 	return err
	// }
	// if len(transRes) > 0 {
	// 	log.Debugf("We already have messages. Nothing to do here. transaction_id: %s", tmps[0].TransactionID)
	// 	return nil
	// }

	// var source *commonaddress.Address
	// var destination *commonaddress.Address
	// if mm.Direction == mmmessage.DirectionInbound {
	// 	source = mm.Source
	// } else {
	// 	source = &mm.Targets[0].Destination
	// }

	// for _, tmp := range mm.Targets {

	// 	if mm.Direction == mmmessage.DirectionInbound {
	// 		source = mm.Source
	// 		destination = &tmp.Destination
	// 	} else {
	// 		source = &mm.Targets[0].Destination
	// 		destination = mm.Source
	// 	}

	// 	// get conversation
	// 	cv, err := h.GetBySelfAndPeer(ctx, source, destination)
	// 	if err != nil {
	// 		log.Debugf("Could not find conversation. Create a new conversation.")

	// 		// create a new conversation
	// 		cv, err = h.Create(ctx, mm.CustomerID, "conversation", "conversation detail", conversation.ReferenceTypeMessage, mm.ID.String(), source, destination)
	// 		if err != nil {
	// 			log.Errorf("Could not create a new conversation. err: %v", err)
	// 			return err
	// 		}
	// 		log.WithField("conversation", cv).Debugf("Created a new conversation. conversation_id: %s", cv.ID)
	// 	}

	// 	// create a message
	// 	direction := message.DirectionIncoming
	// 	status := message.StatusReceived
	// 	if mm.Direction == mmmessage.DirectionOutbound {
	// 		direction = message.DirectionOutgoing
	// 		status = message.StatusSent
	// 	}
	// 	m, err := h.messageHandler.Create(ctx, cv.CustomerID, cv.ID, direction, status, conversation.ReferenceTypeMessage, mm.ID.String(), "", mm.Text, []media.Media{})
	// 	if err != nil {
	// 		log.Errorf("Could not create a message. err: %v", err)
	// 		return err
	// 	}
	// 	log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
	// }

	// return nil

	// // get converstation
	// cv, err := h.GetByReferenceInfo(ctx, tmp.CustomerID, tmp.ReferenceType, tmp.ReferenceID)
	// if err != nil {
	// 	log.Debugf("Could not find conversation. Create a new conversation.")

	// 	// p := &commonaddress.Address{
	// 	// 	Type:   commonaddress.TypeTel,
	// 	// 	Target: tmp.ReferenceID,
	// 	// }

	// 	// create a new conversation
	// 	// cv, err = h.Create(ctx, tmp.CustomerID, "conversation", "conversation detail", conversation.ReferenceTypeMessage, tmp.ReferenceID, localAddr, []commonaddress.Address{*localAddr, *p})
	// 	cv, err = h.Create(ctx, tmp.CustomerID, "conversation", "conversation detail", conversation.ReferenceTypeMessage, tmp.ReferenceID, source, destination)
	// 	if err != nil {
	// 		log.Errorf("Could not create a new conversation. err: %v", err)
	// 		return err
	// 	}
	// 	log.WithField("conversation", cv).Debugf("Created a new conversation. conversation_id: %s", cv.ID)
	// }

	// // create a message
	// m, err := h.messageHandler.Create(ctx, cv.CustomerID, cv.ID, message.DirectionIncoming, message.StatusReceived, conversation.ReferenceTypeMessage, tmp.ReferenceID, tmp.ID.String(), tmp.Source, tmp.Text, tmp.Medias)
	// if err != nil {
	// 	log.Errorf("Could not create a message. err: %v", err)
	// 	return err
	// }
	// log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
	// }

	// for _, tmp := range tmps {
	// 	// get converstation
	// 	cv, err := h.GetByReferenceInfo(ctx, tmp.CustomerID, tmp.ReferenceType, tmp.ReferenceID)
	// 	if err != nil {
	// 		log.Debugf("Could not find conversation. Create a new conversation.")

	// 		// p := &commonaddress.Address{
	// 		// 	Type:   commonaddress.TypeTel,
	// 		// 	Target: tmp.ReferenceID,
	// 		// }

	// 		// create a new conversation
	// 		// cv, err = h.Create(ctx, tmp.CustomerID, "conversation", "conversation detail", conversation.ReferenceTypeMessage, tmp.ReferenceID, localAddr, []commonaddress.Address{*localAddr, *p})
	// 		cv, err = h.Create(ctx, tmp.CustomerID, "conversation", "conversation detail", conversation.ReferenceTypeMessage, tmp.ReferenceID, source, destination)
	// 		if err != nil {
	// 			log.Errorf("Could not create a new conversation. err: %v", err)
	// 			return err
	// 		}
	// 		log.WithField("conversation", cv).Debugf("Created a new conversation. conversation_id: %s", cv.ID)
	// 	}

	// 	// create a message
	// 	m, err := h.messageHandler.Create(ctx, cv.CustomerID, cv.ID, message.DirectionIncoming, message.StatusReceived, conversation.ReferenceTypeMessage, tmp.ReferenceID, tmp.ID.String(), tmp.Source, tmp.Text, tmp.Medias)
	// 	if err != nil {
	// 		log.Errorf("Could not create a message. err: %v", err)
	// 		return err
	// 	}
	// 	log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
	// }

	// return nil
}
