package sessionhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webchat-manager/models/session"
)

// Create creates a new session. If the owning Widget has a
// SessionFlowID configured, this also creates a Conversation and
// triggers that Flow via bin-conversation-manager's
// CreateAndExecuteFlow RPC -- bin-conversation-manager, not
// bin-webchat-manager, owns Create+Execute for SessionFlowID's
// activeflow (design doc
// 2026-07-17-webchat-widget-session-message-flow-split-design.md §3).
//
// The single WidgetGet call below reads Widget.SessionFlowID to
// decide whether to trigger anything. A Widget fetch failure, or a
// SessionFlowID-trigger failure, must NOT fail Session creation
// itself -- both are best-effort (the Session row is already
// committed by the time either is attempted).
func (h *sessionHandler) Create(ctx context.Context, customerID uuid.UUID, widgetID uuid.UUID, pageURL string, referrer string) (*session.Session, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"widget_id":   widgetID,
	})
	log.Debug("Creating a new session.")

	id := h.utilHandler.UUIDCreate()
	log = log.WithField("session_id", id)

	s := &session.Session{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		WidgetID: widgetID,
		Status:   session.StatusActive,
		PageURL:  pageURL,
		Referrer: referrer,
		Peer:     commonaddress.Address{Type: commonaddress.TypeWebSession, Target: id.String()},
		Local:    commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()},
	}

	if err := h.db.SessionCreate(ctx, s); err != nil {
		log.Errorf("Could not create a new session. err: %v", err)
		return nil, err
	}

	res, err := h.db.SessionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created session info. err: %v", err)
		return nil, err
	}
	log.WithField("session", res).Debug("Created a new session.")

	w, err := h.widgetHandler.Get(ctx, widgetID)
	if err != nil {
		log.Errorf("Could not get widget. widget_id: %s, err: %v", widgetID, err)
		// The Session itself was already created successfully; a
		// Widget-fetch failure must not fail the visitor-facing
		// session-creation response. SessionFlowID is simply
		// skipped below.
		return res, nil
	}

	if w.SessionFlowID == uuid.Nil {
		log.Debugf("Widget has no session flow configured. Skipping activeflow. widget_id: %s", w.ID)
		return res, nil
	}

	// Reuse the already-computed, already-correct Session.Local/Session.Peer
	// (line 50-51) instead of re-deriving self/peer here -- see design doc
	// 2026-07-23-webchat-conversation-self-peer-type-unification-design.md §4.A.
	cv, errFlow := h.reqHandler.ConversationV1ConversationCreateAndExecuteFlow(
		ctx,
		customerID,
		w.SessionFlowID,
		cvconversation.TypeWebchat,
		"",
		s.Local,
		s.Peer,
	)
	if errFlow != nil {
		log.Errorf("Could not create and execute flow for the session. err: %v", errFlow)
		// Best-effort: the Session was already created successfully.
		return res, nil
	}

	// Session.ActiveflowID is a write-only marker recording that
	// SessionFlowID was triggered for this Session -- it is never read
	// by any other code path (grep-confirmed), so recording the
	// resulting Conversation's ID here (rather than plumbing the
	// underlying activeflow ID back through an extra RPC field) is
	// sufficient: any non-nil value serves the marker's only purpose.
	fields := map[session.Field]any{
		session.FieldActiveflowID: cv.ID,
	}
	if errUpdate := h.db.SessionUpdate(ctx, id, fields); errUpdate != nil {
		log.Errorf("Could not record activeflow_id on session. err: %v", errUpdate)
		// Not fatal: the Flow itself was already created/executed.
	}

	return res, nil
}
