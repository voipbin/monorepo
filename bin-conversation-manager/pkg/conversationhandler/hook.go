package conversationhandler

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
)

// Hook handles hooked event
func (h *conversationHandler) Hook(ctx context.Context, uri string, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Hook",
		"uri":  uri,
		"data": data,
	})
	log.Debugf("Hook detail. uri: %s", uri)

	// "https://hook.voipbin.net/v1.0/conversation/accounts/<account_id>",
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	// /v1.0/conversation/accounts/<account_id>
	tmpVals := strings.Split(u.Path, "/")
	if len(tmpVals) < 4 {
		log.Debugf("Wrong hook request. Could not get customerID.")
		return fmt.Errorf("no customer info found")
	}
	accountID := uuid.FromStringOrNil(tmpVals[4])

	log.Debugf("Parsed data. customer_id: %s", accountID)

	// get account info
	ac, err := h.accountHandler.Get(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return errors.Wrap(err, "could not get account info")
	}

	switch ac.Type {
	case account.TypeLine:
		// line message
		if errEvent := h.hookLine(ctx, ac, data); errEvent != nil {
			log.Errorf("Could not handle the event type line. err: %v", errEvent)
			return errEvent
		}

	default:
		log.Errorf("Unsupported account type. account_type: %s", ac.Type)
		return fmt.Errorf("unsupported account type. account_type: %s", ac.Type)
	}

	return nil
}

// hookLine handle the line type of hook message
func (h *conversationHandler) hookLine(ctx context.Context, ac *account.Account, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "hookLine",
		"account_id": ac.ID,
	})

	// parse a messages
	conversations, messages, err := h.lineHandler.Hook(ctx, ac, data)
	if err != nil {
		log.Errorf("Could not parse the message. err: %v", err)
		return err
	}

	// conversations
	for _, tmp := range conversations {
		cv, err := h.Create(ctx, tmp.CustomerID, tmp.Name, tmp.Detail, tmp.Type, tmp.DialogID, tmp.Self, tmp.Peer)
		if err != nil {
			log.Errorf("Could not create a new conversation. err: %v", err)
			break
		}
		log.WithField("conversation", cv).Debugf("Created a new conversation. conversation_id: %s", cv.ID)
	}

	// messages
	for _, tmp := range messages {
		peer, err := h.lineHandler.GetParticipant(ctx, ac, tmp.ReferenceID)
		if err != nil {
			log.Errorf("Could not get participant info. err: %v", err)
			peer = &commonaddress.Address{
				Type:       commonaddress.TypeLine,
				Target:     tmp.ReferenceID,
				TargetName: "Unknown",
			}
		}

		self := &commonaddress.Address{
			Type:       commonaddress.TypeLine,
			Target:     "",
			TargetName: "me",
		}

		// get converstation
		cv, err := h.GetBySelfAndPeer(ctx, self, peer)
		if err != nil {
			log.Debugf("Could not find conversation. Create a new conversation.")

			// create a new conversation
			cv, err = h.Create(
				ctx,
				tmp.CustomerID,
				"conversation",
				"conversation detail",
				conversation.TypeLine,
				tmp.ReferenceID,
				self,
				peer,
			)
			if err != nil {
				log.Errorf("Could not create a new conversation. err: %v", err)
				continue
			}
			log.WithField("conversation", cv).Debugf("Created a new conversation. conversation_id: %s", cv.ID)
		}

		// create a message
		m, err := h.messageHandler.Create(
			ctx,
			cv.CustomerID,
			cv.ID,
			message.DirectionIncoming,
			message.StatusDone,
			conversation.TypeLine,
			tmp.ReferenceID,
			"",
			tmp.Text,
			tmp.Medias,
		)
		if err != nil {
			log.Errorf("Could not create a message. err: %v", err)
			continue
		}
		log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
	}

	return nil
}
