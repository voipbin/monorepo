package linehandler

import (
	"context"
	"encoding/json"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

// Hook handles received line message
func (h *lineHandler) Hook(ctx context.Context, ac *account.Account, data []byte) ([]*conversation.Conversation, []*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Hook",
		"account_id": ac.ID,
		"data":       data,
	})

	tmp := &struct {
		Events []*linebot.Event `json:"events"`
	}{}

	if errUnmarshal := json.Unmarshal(data, tmp); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the data. err: %v", errUnmarshal)
		return nil, nil, errUnmarshal
	}

	resConversations := []*conversation.Conversation{}
	resMessages := []*message.Message{}
	for _, e := range tmp.Events {

		// parse the message
		cv, m, err := h.hookEventParse(ctx, ac, e)
		if err != nil {
			log.Errorf("Could not parse the message. err: %v", err)
			continue
		}

		if cv != nil {
			resConversations = append(resConversations, cv)
		} else if m != nil {
			resMessages = append(resMessages, m)
		}
	}

	return resConversations, resMessages, nil
}

// hookEventParse handles the received message.
func (h *lineHandler) hookEventParse(ctx context.Context, ac *account.Account, e *linebot.Event) (*conversation.Conversation, *message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "hookEventParse",
		"customer_id": ac,
		"event":       e,
	})

	switch e.Type {
	case linebot.EventTypeFollow:
		res, err := h.hookEventTypeFollow(ctx, ac, e)
		if err != nil {
			log.Errorf("Could not handle the line event follow. err: %v", err)
			return nil, nil, err
		}
		return res, nil, nil

	case linebot.EventTypeMessage:
		res, err := h.hookEventTypeMessage(ctx, ac, e)
		if err != nil {
			log.Errorf("Could not handle the line event message. err: %v", err)
			return nil, nil, err
		}
		return nil, res, nil

	default:
		log.Errorf("Unsupported event type. event_type: %s", e.Type)
		return nil, nil, fmt.Errorf("unsupported event type. event_type: %s", e.Type)
	}
}

// hookEventTypeFollow handles line's follow event.
func (h *lineHandler) hookEventTypeFollow(ctx context.Context, ac *account.Account, e *linebot.Event) (*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "hookEventTypeFollow",
		"customer_id": ac,
		"event":       e,
	})
	log.Debugf("Handleing the Line follow.")

	// get reference id
	referenceID := h.getReferenceID(e)
	if referenceID == "" {
		return nil, fmt.Errorf("could not get reference id. reference_id: %s", referenceID)
	}

	// get user info
	peer, err := h.GetParticipant(ctx, ac, e.Source.UserID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get participant info")
	}

	self := &commonaddress.Address{
		Type:       commonaddress.TypeLine,
		Target:     "",
		TargetName: "Me",
	}

	// create a conversation
	res := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.Nil,
			CustomerID: ac.CustomerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeNone,
			OwnerID:   uuid.Nil,
		},
		AccountID:   ac.ID,
		Name:        peer.TargetName,
		Detail:      "Conversation with " + peer.TargetName,
		Type:        conversation.TypeLine,
		ReferenceID: referenceID,

		Self: self,
		Peer: peer,
	}

	return res, nil
}

// hookEventTypeMessage handles line's message type event.
func (h *lineHandler) hookEventTypeMessage(ctx context.Context, ac *account.Account, e *linebot.Event) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "hookEventTypeMessage",
		"customer_id": ac,
		"event":       e,
	})
	log.WithField("event", e).Debugf("Handleing the Line message.")

	// get reference id
	referenceID := h.getReferenceID(e)
	if referenceID == "" {
		return nil, fmt.Errorf("could not get reference id. reference_id: %s", referenceID)
	}

	// get datatype and data
	text := ""
	medias := []media.Media{}

	switch m := e.Message.(type) {
	case *linebot.TextMessage:
		text = m.Text

	default:
		log.Errorf("Unsupported messate type. message_type: %s", e.Message.Type())
	}

	// create a message
	m := &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Nil,
			CustomerID: ac.CustomerID,
		},

		ConversationID: uuid.Nil,
		Status:         message.StatusDone,

		ReferenceType: conversation.TypeLine,
		ReferenceID:   referenceID,

		Text:   text,
		Medias: medias,
	}

	return m, nil

}

// getReferenceID returns a reference id
func (h *lineHandler) getReferenceID(e *linebot.Event) string {

	switch e.Source.Type {
	case linebot.EventSourceTypeUser:
		return e.Source.UserID
	case linebot.EventSourceTypeGroup:
		return e.Source.GroupID
	case linebot.EventSourceTypeRoom:
		return e.Source.RoomID
	}

	return ""
}

// GetParticipant returns a participant
func (h *lineHandler) GetParticipant(ctx context.Context, ac *account.Account, id string) (*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getParticipant",
		"customer_id": ac,
		"id":          id,
	})
	log.Debug("Getting the participant info.")

	c, err := h.getClient(ctx, ac)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get client")
	}

	profile := c.GetProfile(id)
	if profile == nil {
		return nil, errors.Wrapf(err, "Could not get profile")
	}

	tmp, err := profile.Do()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get profile info")
	}

	res := &commonaddress.Address{
		Type:       commonaddress.TypeLine,
		Target:     id,
		TargetName: tmp.DisplayName,
	}

	return res, nil
}
