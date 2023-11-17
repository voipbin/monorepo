package conversationhandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
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

	tmps, localAddr, err := h.smsHandler.Event(ctx, data)
	if err != nil {
		log.Errorf("Could not handler the sms message. err: %v", err)
		return err
	}
	log.WithField("local_address", localAddr).Debugf("Found local address. address_target: %s", localAddr.Target)

	// get messages
	transRes, err := h.messageHandler.GetsByTransactionID(ctx, tmps[0].TransactionID, h.utilHandler.TimeGetCurTime(), 10)
	if err != nil {
		log.Errorf("Could not get messages. err: %v", err)
		return err
	}
	if len(transRes) > 0 {
		log.Debugf("We already have messages. Nothing to do here. transaction_id: %s", tmps[0].TransactionID)
		return nil
	}

	for _, tmp := range tmps {
		// get converstation
		cv, err := h.GetByReferenceInfo(ctx, tmp.CustomerID, tmp.ReferenceType, tmp.ReferenceID)
		if err != nil {
			log.Debugf("Could not find conversation. Create a new conversation.")

			p := &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: tmp.ReferenceID,
			}

			// create a new conversation
			cv, err = h.Create(ctx, tmp.CustomerID, "conversation", "conversation detail", conversation.ReferenceTypeMessage, tmp.ReferenceID, localAddr, []commonaddress.Address{*localAddr, *p})
			if err != nil {
				log.Errorf("Could not create a new conversation. err: %v", err)
				return err
			}
			log.WithField("conversation", cv).Debugf("Created a new conversation. conversation_id: %s", cv.ID)
		}

		// create a message
		m, err := h.messageHandler.Create(ctx, cv.CustomerID, cv.ID, message.DirectionIncoming, message.StatusReceived, conversation.ReferenceTypeMessage, tmp.ReferenceID, tmp.ID.String(), tmp.Source, tmp.Text, tmp.Medias)
		if err != nil {
			log.Errorf("Could not create a message. err: %v", err)
			return err
		}
		log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
	}

	return nil
}
