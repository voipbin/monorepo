package sessionhandler

import (
	"context"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webchat-manager/models/session"
)

// Create creates a new session.
func (h *sessionHandler) Create(ctx context.Context, customerID uuid.UUID, widgetID uuid.UUID) (*session.Session, error) {
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

	return res, nil
}
