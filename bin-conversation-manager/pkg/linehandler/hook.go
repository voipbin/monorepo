package linehandler

import (
	"context"
	"encoding/json"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"

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
func (h *lineHandler) Hook(ctx context.Context, ac *account.Account, data []byte) error {
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
		return errUnmarshal
	}

	for _, e := range tmp.Events {
		if errHook := h.hookEventHandle(ctx, ac, e); errHook != nil {
			log.Errorf("Could not handle the message. err: %v", errHook)
			continue
		}
	}

	return nil
}

// hookEventHandle handles the received line event.
func (h *lineHandler) hookEventHandle(ctx context.Context, ac *account.Account, e *linebot.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "hookEventHandle",
		"customer_id": ac,
		"event":       e,
	})

	switch e.Type {
	case linebot.EventTypeFollow:
		if errHook := h.hookEventTypeFollow(ctx, ac, e); errHook != nil {
			log.Errorf("Could not handle the line event follow. err: %v", errHook)
			return errHook
		}
		return nil

	case linebot.EventTypeMessage:
		if errHook := h.hookEventTypeMessage(ctx, ac, e); errHook != nil {
			log.Errorf("Could not handle the line event message. err: %v", errHook)
			return errHook
		}
		return nil

	default:
		log.Errorf("Unsupported event type. event_type: %s", e.Type)
		return fmt.Errorf("unsupported event type. event_type: %s", e.Type)
	}
}

// hookEventTypeFollow handles line's follow event.
func (h *lineHandler) hookEventTypeFollow(ctx context.Context, ac *account.Account, e *linebot.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "hookEventTypeFollow",
		"customer_id": ac,
		"event":       e,
	})
	log.Debugf("Handleing the Line follow.")

	// get dialog id
	dialogID := h.getDialogID(e)
	if dialogID == "" {
		return fmt.Errorf("could not get dialog id. dialog_id: %s", dialogID)
	}

	// get peer info
	peer, err := h.GetPeer(ctx, ac, e.Source.UserID)
	if err != nil {
		return errors.Wrapf(err, "Could not get participant info")
	}

	self := commonaddress.Address{
		Type:       commonaddress.TypeLine,
		Target:     "",
		TargetName: "Me",
	}

	res, err := h.reqHandler.ConversationV1ConversationCreate(
		ctx,
		ac.CustomerID,
		peer.TargetName,
		"Conversation with "+peer.TargetName,
		conversation.TypeLine,
		dialogID,
		self,
		*peer,
	)
	if err != nil {
		return errors.Wrapf(err, "Could not create a conversation")
	}
	log.WithField("conversation", res).Debugf("Created a new conversation. conversation_id: %s", res.ID)

	return nil
}

// hookEventTypeMessage handles line's message type event.
func (h *lineHandler) hookEventTypeMessage(ctx context.Context, ac *account.Account, e *linebot.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "hookEventTypeMessage",
		"customer_id": ac,
		"event":       e,
	})
	log.WithField("event", e).Debugf("Handleing the Line message.")

	// get reference id
	dialogID := h.getDialogID(e)
	if dialogID == "" {
		return fmt.Errorf("could not get reference id. dialog_id: %s", dialogID)
	}

	// get conversation
	filters := map[string]string{
		"deleted":   "false",
		"type":      string(conversation.TypeLine),
		"dialog_id": dialogID,
	}
	cvs, err := h.reqHandler.ConversationV1ConversationGets(ctx, "", 1, filters)
	if err != nil {
		return errors.Wrapf(err, "Could not get conversations")
	} else if len(cvs) == 0 {
		return fmt.Errorf("could not find conversation. dialog_id: %s", dialogID)
	}
	cv := cvs[0]

	// get datatype and data
	text := ""
	medias := []media.Media{}

	switch m := e.Message.(type) {
	case *linebot.TextMessage:
		text = m.Text

	default:
		log.Errorf("Unsupported messate type. message_type: %s", e.Message.Type())
	}

	m, err := h.reqHandler.ConversationV1MessageCreate(
		ctx,
		cv.CustomerID,
		cv.ID,
		message.DirectionIncoming,
		message.StatusDone,
		message.ReferenceTypeLine,
		uuid.Nil,
		"",
		text,
		medias,
	)
	if err != nil {
		return errors.Wrapf(err, "Could not create a message")
	}
	log.WithField("message", m).Debugf("Created a new message. message_id: %s", m.ID)

	return nil
}

// getDialogID returns a reference id
func (h *lineHandler) getDialogID(e *linebot.Event) string {

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

// GetPeer returns a participant
func (h *lineHandler) GetPeer(ctx context.Context, ac *account.Account, userID string) (*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getParticipant",
		"customer_id": ac,
		"user_id":     userID,
	})
	log.Debug("Getting the participant info.")

	c, err := h.getClient(ctx, ac)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get client")
	}

	profile := c.GetProfile(userID)
	if profile == nil {
		return nil, errors.Wrapf(err, "Could not get profile")
	}

	tmp, err := profile.Do()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get profile info")
	}

	res := &commonaddress.Address{
		Type:       commonaddress.TypeLine,
		Target:     userID,
		TargetName: tmp.DisplayName,
	}

	return res, nil
}
