package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// Create is handy function for creating a confbridge.
// it increases corresponded counter
func (h *confbridgeHandler) Create(ctx context.Context, confbridgeType confbridge.Type) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Create",
		},
	)

	id := uuid.Must(uuid.NewV4())

	cb := &confbridge.Confbridge{
		ID:   id,
		Type: confbridgeType,

		RecordingIDs:   []uuid.UUID{},
		ChannelCallIDs: map[string]uuid.UUID{},

		TMCreate: h.util.GetCurTime(),
		TMUpdate: dbhandler.DefaultTimeStamp,
		TMDelete: dbhandler.DefaultTimeStamp,
	}

	// create a confbridge
	if errCreate := h.db.ConfbridgeCreate(ctx, cb); errCreate != nil {
		return nil, fmt.Errorf("could not create a conference. err: %v", errCreate)
	}
	promConfbridgeCreateTotal.Inc()

	res, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created confbridge info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeCreated, res)

	return res, nil
}

// Get returns confbridge
func (h *confbridgeHandler) Get(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Get",
		},
	)

	// create confbridge
	res, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the confbridge. err: %v", err)
		return nil, err
	}

	return res, nil
}
