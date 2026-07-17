package sessionhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webchat-manager/models/session"
)

// Get returns the session.
func (h *sessionHandler) Get(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	res, err := h.db.SessionGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get session. err: %w", err)
	}

	return res, nil
}

// List returns sessions.
func (h *sessionHandler) List(ctx context.Context, size uint64, token string, filters map[session.Field]any) ([]*session.Session, error) {
	res, err := h.db.SessionList(ctx, size, token, filters)
	if err != nil {
		return nil, fmt.Errorf("could not list sessions. err: %w", err)
	}

	return res, nil
}

// Delete deletes the session.
func (h *sessionHandler) Delete(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Delete",
		"session_id": id,
	})
	log.Debug("Deleting the session.")

	if err := h.db.SessionDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the session info. err: %v", err)
		return nil, err
	}

	res, err := h.db.SessionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted session info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// End marks the session as ended.
func (h *sessionHandler) End(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "End",
		"session_id": id,
	})
	log.Debug("Ending the session.")

	fields := map[session.Field]any{
		session.FieldStatus: session.StatusEnded,
	}

	if err := h.db.SessionUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the session status. err: %v", err)
		return nil, err
	}

	res, err := h.db.SessionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated session info. err: %v", err)
		return nil, err
	}

	// Publish EventTypeSessionEnded so the visitor-side WS client (see
	// bin-api-manager's createTopics webchat_session topic, VOIP-1265)
	// can close its connection cooperatively -- otherwise the visitor's
	// WS subscription for this session's topic would stay open
	// indefinitely with no signal that the session is over.
	if h.notifyHandler != nil {
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, session.EventTypeSessionEnded, res)
	}

	return res, nil
}
