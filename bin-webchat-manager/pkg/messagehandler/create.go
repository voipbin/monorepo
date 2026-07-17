package messagehandler

import (
	"context"

	commonidentity "monorepo/bin-common-handler/models/identity"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/widget"
)

// Create creates a new message. If this is an INBOUND message AND the
// owning Widget has a MessageFlowID configured, this also triggers a
// brand-new, independent activeflow for THIS message -- unconditionally,
// on every inbound message, with no "already triggered" gate and no
// per-session lock. This mirrors bin-conversation-manager's existing
// Account.MessageFlowID/Number.MessageFlowID pattern (SMS/LINE/WhatsApp)
// exactly. See design doc
// 2026-07-17-webchat-widget-session-message-flow-split-design.md §2.3, §5.
//
// SessionFlowID's trigger (fires once per Session, at session-create
// time) is NOT this function's concern -- that is owned by
// bin-conversation-manager's CreateAndExecuteFlow, called from
// bin-webchat-manager's sessionhandler.Create. See design doc §3.
//
// Accepted risk (design doc §4): with no per-session lock, rapid
// consecutive inbound messages can each trigger their own concurrent
// activeflow. This is the same shape of risk
// bin-conversation-manager's SMS/LINE/WhatsApp executeActiveflow
// already lives with (no per-conversation lock there either) --
// applying an existing, already-accepted platform risk to a channel
// with a higher realistic message frequency, not introducing a new
// class of risk.
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

	// Only inbound (visitor -> VoIPbin) messages can trigger
	// MessageFlowID. Outbound (VoIPbin -> visitor, e.g. an agent reply
	// or a Flow-delivered response) never does -- matches
	// conversation-manager's MessageEventSent never calling
	// runExecuteModeFlow.
	if direction != message.DirectionInbound {
		return res, nil
	}

	w, err := h.db.WidgetGet(ctx, sess.WidgetID)
	if err != nil {
		log.Errorf("Could not get widget. widget_id: %s, err: %v", sess.WidgetID, err)
		// The message itself was already created successfully; a Flow-
		// trigger failure must not fail the visitor-facing send.
		return res, nil
	}

	if w.MessageFlowID == uuid.Nil {
		log.Debugf("Widget has no message flow configured. Skipping activeflow. widget_id: %s", w.ID)
		return res, nil
	}

	if errFlow := h.triggerMessageFlow(ctx, customerID, sessionID, w, res); errFlow != nil {
		log.Errorf("Could not trigger the message activeflow. err: %v", errFlow)
		// Best-effort: the message send itself already succeeded.
	}

	return res, nil
}

// create is the shared persistence step, factored out so both the
// inbound and outbound paths share identical row-construction logic.
// Publishes EventTypeMessageCreated to both the webhook and event queue
// on every successful create (design doc §16: conversation-manager
// subscribes to this event to build its own independent
// Conversation/Message records, message-manager pattern).
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

// triggerMessageFlow creates and executes a brand-new activeflow for
// THIS inbound message, unconditionally -- no "already triggered"
// bookkeeping, no cross-message state. Mirrors
// bin-conversation-manager's executeActiveflow for SMS/LINE/WhatsApp's
// MessageFlowID exactly.
func (h *messageHandler) triggerMessageFlow(
	ctx context.Context,
	customerID uuid.UUID,
	sessionID uuid.UUID,
	w *widget.Widget,
	m *message.Message,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":            "triggerMessageFlow",
		"session_id":      sessionID,
		"widget_id":       w.ID,
		"message_flow_id": w.MessageFlowID,
	})

	variables := map[string]string{
		"voipbin.webchat.session.id":        sessionID.String(),
		"voipbin.webchat.session.widget_id": w.ID.String(),
		"voipbin.webchat.message.text":      m.Text,
	}

	af, err := h.reqHandler.FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		customerID,
		w.MessageFlowID,
		fmactiveflow.ReferenceTypeWebchat,
		sessionID,
		uuid.Nil,
		variables,
		"",
		fmactiveflow.WebhookMethodNone,
	)
	if err != nil {
		return err
	}
	log.WithField("activeflow_id", af.ID).Debug("Created activeflow.")

	if err := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); err != nil {
		return err
	}
	log.Debug("Executed activeflow.")

	return nil
}
