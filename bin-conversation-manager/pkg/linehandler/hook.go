package linehandler

import (
	"context"
	"encoding/json"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/line/line-bot-sdk-go/v7/linebot"
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
		cv, m, err := h.handleHook(ctx, ac, e)
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

// handleHook handles the received message.
func (h *lineHandler) handleHook(ctx context.Context, ac *account.Account, e *linebot.Event) (*conversation.Conversation, *message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "handleEvent",
		"customer_id": ac,
		"event":       e,
	})

	switch e.Type {
	case linebot.EventTypeFollow:
		res, err := h.hookHandleFollow(ctx, ac, e)
		if err != nil {
			log.Errorf("Could not handle the line event follow. err: %v", err)
			return nil, nil, err
		}
		return res, nil, nil

	case linebot.EventTypeMessage:
		res, err := h.hookHandleMessage(ctx, ac, e)
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

// hookHandleFollow handles line's follow event.
func (h *lineHandler) hookHandleFollow(ctx context.Context, ac *account.Account, e *linebot.Event) (*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "eventHandleFollow",
		"customer_id": ac,
		"event":       e,
	})
	log.Debugf("Handleing the Line follow.")

	// get reference id
	referenceID := h.getReferenceID(e)
	if referenceID == "" {
		log.Errorf("Could not get reference id. reference_id: %s", referenceID)
		return nil, fmt.Errorf("could not get reference id")
	}

	// get user info
	p, err := h.GetParticipant(ctx, ac, e.Source.UserID)
	if err != nil {
		log.Errorf("Could not get participant info. err: %v", err)
		return nil, err
	}

	me := &commonaddress.Address{
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
		AccountID:     ac.ID,
		Name:          p.TargetName,
		Detail:        "Conversation with " + p.TargetName,
		ReferenceType: conversation.ReferenceTypeLine,
		ReferenceID:   referenceID,

		Source: me,
		Participants: []commonaddress.Address{
			*me,
			*p,
		},
	}

	return res, nil
}

// hookHandleMessage handles line's message type event.
func (h *lineHandler) hookHandleMessage(ctx context.Context, ac *account.Account, e *linebot.Event) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "eventHandleMessage",
		"customer_id": ac,
		"event":       e,
	})
	log.WithField("event", e).Debugf("Handleing the Line message.")

	// get reference id
	referenceID := h.getReferenceID(e)
	if referenceID == "" {
		log.Errorf("Could not get reference id. reference_id: %s", referenceID)
		return nil, fmt.Errorf("could not get reference id")
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

	source := &commonaddress.Address{
		Type:   commonaddress.TypeLine,
		Target: e.Source.UserID,
	}

	// create a message
	m := &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Nil,
			CustomerID: ac.CustomerID,
		},

		ConversationID: uuid.Nil,
		Status:         message.StatusReceived,

		ReferenceType: conversation.ReferenceTypeLine,
		ReferenceID:   referenceID,

		Source: source,

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
		log.Errorf("Could not get client. err: %v", err)
		return nil, err
	}

	profile := c.GetProfile(id)
	if profile == nil {
		log.Errorf("Could not get initiate profile.")
		return nil, fmt.Errorf("could not initiate profile")
	}

	tmp, err := profile.Do()
	if err != nil {
		log.Errorf("Could not get profile info. err: %v", err)
		return nil, err
	}

	res := &commonaddress.Address{
		Type:       commonaddress.TypeLine,
		Target:     id,
		TargetName: tmp.DisplayName,
	}

	return res, nil
}
