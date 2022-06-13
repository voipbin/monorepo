package linehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
)

// Event handles received line message
func (h *lineHandler) Event(ctx context.Context, customerID uuid.UUID, data []byte) ([]*conversation.Conversation, []*message.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Receive",
		},
	)

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
		cv, m, err := h.handleEvent(ctx, customerID, e)
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

// handleEvent handles the received message.
func (h *lineHandler) handleEvent(ctx context.Context, customerID uuid.UUID, e *linebot.Event) (*conversation.Conversation, *message.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "handleEvent",
		},
	)

	switch e.Type {
	case linebot.EventTypeFollow:
		res, err := h.eventHandleFollow(ctx, customerID, e)
		if err != nil {
			log.Errorf("Could not handle the line event follow. err: %v", err)
			return nil, nil, err
		}
		return res, nil, nil

	case linebot.EventTypeMessage:
		res, err := h.eventHandleMessage(ctx, customerID, e)
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

// eventHandleFollow handles line's follow event.
func (h *lineHandler) eventHandleFollow(ctx context.Context, customerID uuid.UUID, e *linebot.Event) (*conversation.Conversation, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "eventHandleFollow",
		},
	)
	log.Debugf("Handleing the Line follow.")

	// get reference id
	referenceID := h.getReferenceID(e)
	if referenceID == "" {
		log.Errorf("Could not get reference id. reference_id: %s", referenceID)
		return nil, fmt.Errorf("could not get reference id")
	}

	// get user info
	p, err := h.getParticipant(ctx, customerID, e.Source.UserID)
	if err != nil {
		log.Errorf("Could not get participant info. err: %v", err)
		return nil, err
	}

	// create a conversation
	res := &conversation.Conversation{
		ID:            uuid.Nil,
		CustomerID:    customerID,
		Name:          p.TargetName,
		Detail:        "Conversation with " + p.TargetName,
		ReferenceType: conversation.ReferenceTypeLine,
		ReferenceID:   referenceID,
		Participants: []commonaddress.Address{
			*p,
		},
	}

	return res, nil
}

// eventHandleMessage handles line's message type event.
func (h *lineHandler) eventHandleMessage(ctx context.Context, customerID uuid.UUID, e *linebot.Event) (*message.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "eventHandleMessage",
		},
	)
	log.WithField("event", e).Debugf("Handleing the Line message.")

	// get reference id
	referenceID := h.getReferenceID(e)
	if referenceID == "" {
		log.Errorf("Could not get reference id. reference_id: %s", referenceID)
		return nil, fmt.Errorf("could not get reference id")
	}

	// get datatype and data
	var dataType message.Type
	var data []byte

	switch m := e.Message.(type) {
	case *linebot.TextMessage:
		dataType = message.TypeText
		data = []byte(m.Text)

	default:
		log.Errorf("Unsupported messate type. message_type: %s", e.Message.Type())
	}

	// create a message
	m := &message.Message{
		ID:         uuid.Nil,
		CustomerID: customerID,

		ConversationID: uuid.Nil,
		Status:         message.StatusReceived,

		ReferenceType: conversation.ReferenceTypeLine,
		ReferenceID:   referenceID,

		SourceTarget: e.Source.UserID,

		Type: dataType,
		Data: data,
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

// getParticipant returns a participant
func (h *lineHandler) getParticipant(ctx context.Context, customerID uuid.UUID, id string) (*commonaddress.Address, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "getParticipant",
		},
	)
	log.Debug("Getting the participant info.")

	c, err := h.getClient(ctx, customerID)
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
