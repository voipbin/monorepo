package widgethandler

import (
	"context"
	"fmt"

	commonidentity "monorepo/bin-common-handler/models/identity"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webchat-manager/models/widget"
)

// Create creates a new widget.
func (h *widgetHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	welcomeMessage string,
	flowID uuid.UUID,
	sessionIdleTimeout int,
	themeConfig *widget.ThemeConfig,
) (*widget.Widget, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                  "Create",
		"customer_id":           customerID,
		"name":                  name,
		"welcome_message":       welcomeMessage,
		"flow_id":               flowID,
		"session_idle_timeout":  sessionIdleTimeout,
	})
	log.Debug("Creating a new widget.")

	if sessionIdleTimeout <= 0 {
		sessionIdleTimeout = widget.DefaultSessionIdleTimeout
	}

	// generate widget id
	id := h.utilHandler.UUIDCreate()
	log = log.WithField("widget_id", id)

	// create direct hash
	d, err := h.reqHandler.DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeWebchatWidget, id)
	if err != nil {
		log.Errorf("Could not create direct hash. err: %v", err)
		return nil, fmt.Errorf("could not create direct hash: %w", err)
	}
	log.WithField("direct", d).Debugf("Created direct hash. direct_id: %s", d.ID)

	w := &widget.Widget{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		Name:   name,
		Status: widget.StatusActive,

		DirectID: d.ID,

		WelcomeMessage: welcomeMessage,
		FlowID:         flowID,

		SessionIdleTimeout: sessionIdleTimeout,

		ThemeConfig: themeConfig,
	}

	if errCreate := h.db.WidgetCreate(ctx, w); errCreate != nil {
		log.Errorf("Could not create a new widget. err: %v", errCreate)
		// cleanup orphaned direct
		if _, errDelete := h.reqHandler.DirectV1DirectDelete(ctx, d.ID); errDelete != nil {
			log.Errorf("Could not cleanup orphaned direct. direct_id: %s, err: %v", d.ID, errDelete)
		}
		return nil, errCreate
	}

	res, err := h.db.WidgetGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created widget info. err: %v", err)
		return nil, err
	}
	log.WithField("widget", res).Debug("Created a new widget.")

	return res, nil
}
