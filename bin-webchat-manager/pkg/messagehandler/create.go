package messagehandler

import (
	"context"

	commonidentity "monorepo/bin-common-handler/models/identity"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
)

// Create creates a new message. If this is the first INBOUND message on
// the given Session AND the owning Widget has a FlowID configured, this
// also triggers the Widget's activeflow (reference_type=webchat) —
// design doc §5 core flow step 5, §9 Flow variable integration, §14
// step 8. An empty Session with no message never triggers a Flow on its
// own (the trigger is anchored to the first inbound message, not to
// Session creation — v7 Revision Notice).
//
// Round 4 review finding (Medium, must be resolved here): the
// first-inbound-message check is serialized per Session via an
// in-process keyed lock (see main.go's lockSession/unlockSession) so a
// double-fired first message cannot observe "no prior activeflow" twice
// and trigger the Flow twice. This only protects within one pod — see
// the design doc's multi-replica caveat; DB-level duplicate-trigger risk
// beyond a single pod is accepted as a low-probability Phase 1 risk,
// consistent with the doc's own accepted-risk framing for adjacent
// races (idle sweep vs in-flight MessageSend, §15).
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

	// Only inbound (visitor -> VoIPbin) messages can trigger the
	// first-message Flow. Outbound (VoIPbin -> visitor, e.g. an agent
	// reply or a Flow-delivered response) never does — matches
	// conversation-manager's MessageEventSent never calling
	// runExecuteModeFlow (verified in the design doc's Round 3 review).
	// Both paths still resolve the Session (WidgetID is denormalized
	// onto every Message for the outbound event payload too), but only
	// the inbound path takes the per-session lock, since only inbound
	// messages can trigger the Flow.
	if direction != message.DirectionInbound {
		sess, err := h.db.SessionGet(ctx, sessionID)
		if err != nil {
			log.Errorf("Could not get session. err: %v", err)
			return nil, err
		}
		return h.create(ctx, customerID, sess.WidgetID, sessionID, direction, senderID, text)
	}

	lockCh := h.lockSession(sessionID)
	defer h.unlockSession(lockCh)

	sess, err := h.db.SessionGet(ctx, sessionID)
	if err != nil {
		log.Errorf("Could not get session. err: %v", err)
		return nil, err
	}

	res, err := h.create(ctx, customerID, sess.WidgetID, sessionID, direction, senderID, text)
	if err != nil {
		return nil, err
	}

	// First inbound message on this Session iff no activeflow has been
	// recorded on it yet. Checked/set under the per-session lock above.
	if sess.ActiveflowID != uuid.Nil {
		return res, nil
	}

	w, err := h.db.WidgetGet(ctx, sess.WidgetID)
	if err != nil {
		log.Errorf("Could not get widget. widget_id: %s, err: %v", sess.WidgetID, err)
		// The message itself was already created successfully; a Flow-
		// trigger failure must not fail the visitor-facing send.
		return res, nil
	}

	if w.FlowID == uuid.Nil {
		log.Debugf("Widget has no flow configured. Skipping activeflow. widget_id: %s", w.ID)
		return res, nil
	}

	if errFlow := h.triggerFirstMessageFlow(ctx, customerID, sessionID, w, res); errFlow != nil {
		log.Errorf("Could not trigger the first-message activeflow. err: %v", errFlow)
		// Best-effort: the message send itself already succeeded.
	}

	return res, nil
}

// create is the shared persistence step, factored out so both the
// inbound (lock-guarded) and outbound (unguarded) paths share identical
// row-construction logic. Publishes EventTypeMessageCreated to both the
// webhook and event queue on every successful create (design doc §16:
// conversation-manager subscribes to this event to build its own
// independent Conversation/Message records, message-manager pattern).
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

// triggerFirstMessageFlow creates and executes the Widget's activeflow
// for this Session's first inbound message, then records the resulting
// ActiveflowID on the Session (this is the "already triggered" marker
// read by Create above).
func (h *messageHandler) triggerFirstMessageFlow(
	ctx context.Context,
	customerID uuid.UUID,
	sessionID uuid.UUID,
	w *widget.Widget,
	m *message.Message,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "triggerFirstMessageFlow",
		"session_id": sessionID,
		"widget_id":  w.ID,
		"flow_id":    w.FlowID,
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
		w.FlowID,
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

	fields := map[session.Field]any{
		session.FieldActiveflowID: af.ID,
	}
	if errUpdate := h.db.SessionUpdate(ctx, sessionID, fields); errUpdate != nil {
		log.Errorf("Could not record activeflow_id on session. err: %v", errUpdate)
		// Not fatal: the Flow itself was already created/executed. Worst
		// case a subsequent message re-triggers (bounded by this same
		// lock, so at most once more) rather than never triggering.
	}

	if err := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); err != nil {
		return err
	}
	log.Debug("Executed activeflow.")

	return nil
}
