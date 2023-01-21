package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// Create is handy function for creating a confbridge.
// it increases corresponded counter
func (h *confbridgeHandler) Create(ctx context.Context, customerID uuid.UUID, confbridgeType confbridge.Type) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Create",
			"customer_id": customerID,
		},
	)

	id := uuid.Must(uuid.NewV4())

	cb := &confbridge.Confbridge{
		ID:         id,
		CustomerID: customerID,
		Type:       confbridgeType,

		RecordingIDs:   []uuid.UUID{},
		ChannelCallIDs: map[string]uuid.UUID{},
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

// UpdateRecordingID updates the confbridge's recording id.
// if the recording id is not uuid.Nil, it also adds to the recording_ids
func (h *confbridgeHandler) UpdateRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "UpdateRecordingID",
			"confbridge_id": id,
			"recording_id":  recordingID,
		},
	)

	if errSet := h.db.ConfbridgeSetRecordingID(ctx, id, recordingID); errSet != nil {
		log.Errorf("Could not set the recording id. err: %v", errSet)
		return nil, errSet
	}

	if recordingID != uuid.Nil {
		// add the recording id
		log.Debugf("Adding the recording id. confbridge_id: %s, recording_id: %s", id, recordingID)
		if errAdd := h.db.ConfbridgeAddRecordingIDs(ctx, id, recordingID); errAdd != nil {
			log.Errorf("Could not add the recording id. err: %v", errAdd)
			return nil, errAdd
		}
	}

	// get updated confbridge
	res, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated confbridge. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateExternalMediaID updates the confbridge's external media id.
func (h *confbridgeHandler) UpdateExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "UpdateExternalMediaID",
			"confbridge_id":     id,
			"external_media_id": externalMediaID,
		},
	)

	if errSet := h.db.ConfbridgeSetExternalMediaID(ctx, id, externalMediaID); errSet != nil {
		log.Errorf("Could not set the external media id. err: %v", errSet)
		return nil, errSet
	}

	// get updated confbridge
	res, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated confbridge. err: %v", err)
		return nil, err
	}

	return res, nil
}
