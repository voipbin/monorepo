package messagehandler

import (
	"context"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webchat-manager/models/message"
)

// Create creates a new message. Flow-triggering for inbound messages
// (MessageFlowID) is no longer this function's concern -- that is now
// owned entirely by bin-conversation-manager's runExecuteModeFlowWebchat,
// triggered asynchronously off the existing webchat_message_created
// event this function already publishes (via h.create below). See
// design doc 2026-07-18-webchat-message-flow-owner-migration-design.md.
//
// SessionFlowID's trigger (fires once per Session, at session-create
// time) is likewise not this function's concern -- that is owned by
// bin-conversation-manager's CreateAndExecuteFlow, called from
// bin-webchat-manager's sessionhandler.Create. See design doc
// 2026-07-17-webchat-widget-session-message-flow-split-design.md §3.
func (h *messageHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	sessionID uuid.UUID,
	direction message.Direction,
	senderID uuid.UUID,
	text string,
) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"session_id":  sessionID,
		"direction":   direction,
	})
	log.Debug("Creating a new message.")

	sess, err := h.db.SessionGet(ctx, sessionID)
	if err != nil {
		log.Errorf("Could not get session. err: %v", err)
		return nil, err
	}

	res, err := h.create(ctx, customerID, sess.WidgetID, sessionID, direction, senderID, text)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// create is the shared persistence step, factored out so both the
// inbound and outbound paths share identical row-construction logic.
// Publishes EventTypeMessageCreated to both the webhook and event queue
// on every successful create (design doc §16: conversation-manager
// subscribes to this event to build its own independent
// Conversation/Message records, message-manager pattern, AND -- as of
// the message-flow-owner-migration design -- to trigger MessageFlowID's
// activeflow for inbound messages).
func (h *messageHandler) create(
	ctx context.Context,
	customerID uuid.UUID,
	widgetID uuid.UUID,
	sessionID uuid.UUID,
	direction message.Direction,
	senderID uuid.UUID,
	text string,
) (*message.Message, error) {
	id := h.utilHandler.UUIDCreate()

	m := &message.Message{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		WidgetID:  widgetID,
		SessionID: sessionID,
		Direction: direction,
		Status:    message.StatusSent,
		SenderID:  senderID,

		Text: text,
	}

	if err := h.db.MessageCreate(ctx, m); err != nil {
		return nil, err
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		return nil, err
	}

	if h.notifyHandler != nil {
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageCreated, res)
	}

	return res, nil
}
