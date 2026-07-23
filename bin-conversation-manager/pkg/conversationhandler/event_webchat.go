package conversationhandler

import (
	"context"
	"encoding/json"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
)

// webchatMessage mirrors bin-webchat-manager's models/message.WebhookMessage
// shape (the payload of the webchat_message_created event). Defined locally
// rather than importing bin-webchat-manager/models/message, matching the
// existing pattern where conversation-manager unmarshals SMS's
// mmmessage.Message directly -- but webchat-manager's WebhookMessage has no
// Targets/Source list (unlike SMS's multi-target shape), so this is a
// minimal local mirror of just the fields eventWebchat needs.
type webchatMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	WidgetID   uuid.UUID `json:"widget_id"`
	SessionID  uuid.UUID `json:"session_id"`
	Direction  string    `json:"direction"`
	Text       string    `json:"text"`
}

// list of webchat-manager message directions (mirrors
// bin-webchat-manager/models/message.Direction).
const (
	webchatDirectionInbound  = "inbound"
	webchatDirectionOutbound = "outbound"
)

// eventWebchat handles the webchat_message_created event published by
// bin-webchat-manager, per design doc §16 (message-manager pattern).
// Mirrors eventSMS exactly: unmarshal -> direction switch ->
// MessageEventReceived/MessageEventSentWebchat. Unlike eventSMS, webchat
// messages carry exactly one (WidgetID, SessionID) pair rather than a
// Source + multiple Targets, so there is no per-target loop here -- see
// messageEventReceivedWebchat/messageEventSentWebchat in event_webchat.go.
func (h *conversationHandler) eventWebchat(ctx context.Context, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "eventWebchat",
		"data": data,
	})
	log.Debugf("Received webchat message.")

	wm := webchatMessage{}
	if err := json.Unmarshal(data, &wm); err != nil {
		return errors.Wrapf(err, "Could not unmarshal the data")
	}

	switch wm.Direction {
	case webchatDirectionInbound:
		if errEvent := h.messageEventReceivedWebchat(ctx, &wm); errEvent != nil {
			return errors.Wrapf(errEvent, "Could not handle the event correctly")
		}
		return nil

	case webchatDirectionOutbound:
		if errEvent := h.messageEventSentWebchat(ctx, &wm); errEvent != nil {
			return errors.Wrapf(errEvent, "Could not handle the event correctly")
		}
		return nil

	default:
		return errors.Errorf("could not find the direction. direction: %s", wm.Direction)
	}
}

// messageEventReceivedWebchat handles an inbound (visitor -> VoIPbin)
// webchat message. Self = Widget.ID (our fixed identity), Peer =
// Session.ID (the visitor's continuity token) -- per §16.3, the exact
// SMS-style (self, peer) address-pair identity, with no Account created.
func (h *conversationHandler) messageEventReceivedWebchat(ctx context.Context, wm *webchatMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "messageEventReceivedWebchat",
	})

	self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: wm.WidgetID.String()}
	peer := commonaddress.Address{Type: commonaddress.TypeWebSession, Target: wm.SessionID.String()}

	cv, err := h.GetOrCreateBySelfAndPeer(
		ctx,
		wm.CustomerID,
		conversation.TypeWebchat,
		"", // no dialog id for webchat, mirrors SMS's empty dialog id
		self,
		peer,
	)
	if err != nil {
		return errors.Wrapf(err, "Could not get conversation")
	}
	log.WithField("conversation", cv).Debugf("Found conversation. conversation_id: %s", cv.ID)

	source, destination := messagehandler.DeriveEndpoints(cv, message.DirectionIncoming)
	convMsg, err := h.messageHandler.Create(ctx, messagehandler.MessageCreateArgs{
		ID:             wm.ID,
		CustomerID:     cv.CustomerID,
		ConversationID: cv.ID,
		Direction:      message.DirectionIncoming,
		Status:         message.StatusDone,
		ReferenceType:  message.ReferenceTypeWebchat,
		ReferenceID:    wm.ID,
		Text:           wm.Text,
		Medias:         []media.Media{},
		Source:         source,
		Destination:    destination,
		CaseID:         caseIDHint(cv),
	})
	if err != nil {
		return errors.Wrapf(err, "Could not create a message")
	}
	log.WithField("message", convMsg).Debugf("Create a message. message_id: %s", convMsg.ID)

	// MessageFlowID's Flow-trigger (design doc
	// 2026-07-18-webchat-message-flow-owner-migration-design.md):
	// this subscriber path NOW owns the Flow trigger for TypeWebchat
	// conversations, via ExecuteModeFlow -> runExecuteModeFlowWebchat,
	// exactly mirroring SMS/LINE/WhatsApp's own inbound path. This
	// reverses the prior "B안" behavior (webchat-manager alone owned
	// the trigger) -- webchat-manager no longer triggers any
	// activeflow of its own; this event handler is now the SOLE
	// trigger point for MessageFlowID.
	mode := h.getExecuteMode(cv)
	switch mode {
	case ExecuteModeAgent:
		if errAgent := h.runExecuteModeAgent(ctx, cv, convMsg); errAgent != nil {
			return errors.Wrapf(errAgent, "could not run agent mode. message_id: %s", convMsg.ID)
		}
	case ExecuteModeFlow:
		if errFlow := h.runExecuteModeFlow(ctx, cv, convMsg); errFlow != nil {
			return errors.Wrapf(errFlow, "could not run flow mode. message_id: %s", convMsg.ID)
		}
	case ExecuteModeNone:
		// reserved; no-op
	}

	return nil
}

// messageEventSentWebchat handles an outbound (VoIPbin -> visitor)
// webchat message (agent reply or Flow/AI-generated response). Mirrors
// eventSMS's MessageEventSent: get-or-create the same Conversation
// (Self/Peer reversed relative to received), then create-or-update the
// mirrored Message row. Never triggers a Flow (outbound direction).
func (h *conversationHandler) messageEventSentWebchat(ctx context.Context, wm *webchatMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "messageEventSentWebchat",
	})

	self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: wm.WidgetID.String()}
	peer := commonaddress.Address{Type: commonaddress.TypeWebSession, Target: wm.SessionID.String()}

	cv, err := h.GetOrCreateBySelfAndPeer(
		ctx,
		wm.CustomerID,
		conversation.TypeWebchat,
		"",
		self,
		peer,
	)
	if err != nil {
		return errors.Wrapf(err, "Could not get conversation")
	}
	log.WithField("conversation", cv).Debugf("Found conversation. conversation_id: %s", cv.ID)

	msgID := wm.ID

	tmp, err := h.messageHandler.Get(ctx, msgID)
	if err != nil {
		source, destination := messagehandler.DeriveEndpoints(cv, message.DirectionOutgoing)
		created, errCreate := h.messageHandler.Create(ctx, messagehandler.MessageCreateArgs{
			ID:             msgID,
			CustomerID:     cv.CustomerID,
			ConversationID: cv.ID,
			Direction:      message.DirectionOutgoing,
			Status:         message.StatusDone,
			ReferenceType:  message.ReferenceTypeWebchat,
			ReferenceID:    msgID,
			Text:           wm.Text,
			Medias:         []media.Media{},
			Source:         source,
			Destination:    destination,
			CaseID:         caseIDHint(cv),
		})
		if errCreate != nil {
			return errors.Wrapf(errCreate, "Could not create a message")
		}
		log.WithField("message", created).Debugf("Create a message. message_id: %s", created.ID)
		return nil
	}

	log.WithField("message", tmp).Debugf("Found message. Updating the message status. message_id: %s", tmp.ID)
	updated, err := h.messageHandler.UpdateStatus(ctx, tmp.ID, message.StatusDone)
	if err != nil {
		return errors.Wrapf(err, "Could not update the message")
	}
	log.WithField("message", updated).Debugf("Updated message. message_id: %s", updated.ID)

	return nil
}
