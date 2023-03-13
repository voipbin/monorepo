package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// Create is handy function for creating a confbridge.
// it increases corresponded counter
func (h *confbridgeHandler) Create(ctx context.Context, customerID uuid.UUID, confbridgeType confbridge.Type) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})

	id := h.utilHandler.CreateUUID()

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

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created confbridge info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeCreated, res)

	return res, nil
}

// Get returns confbridge
func (h *confbridgeHandler) Get(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Get",
	})

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
	log := logrus.WithFields(logrus.Fields{
		"func":          "UpdateRecordingID",
		"confbridge_id": id,
		"recording_id":  recordingID,
	})

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
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated confbridge. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateExternalMediaID updates the confbridge's external media id.
func (h *confbridgeHandler) UpdateExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "UpdateExternalMediaID",
		"confbridge_id":     id,
		"external_media_id": externalMediaID,
	})

	if errSet := h.db.ConfbridgeSetExternalMediaID(ctx, id, externalMediaID); errSet != nil {
		log.Errorf("Could not set the external media id. err: %v", errSet)
		return nil, errSet
	}

	// get updated confbridge
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated confbridge. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateBridgeID updates the confbridge's bridge id.
func (h *confbridgeHandler) UpdateBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "UpdateBridgeID",
		"confbridge_id": id,
		"bridge_id":     bridgeID,
	})

	if errSet := h.db.ConfbridgeSetBridgeID(ctx, id, bridgeID); errSet != nil {
		log.Errorf("Could not set the bridge id. err: %v", errSet)
		return nil, errSet
	}

	// get updated confbridge
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated confbridge. err: %v", err)
		return nil, err
	}

	return res, nil
}

// RemoveChannelCallID removes the channel from the channel call id
func (h *confbridgeHandler) RemoveChannelCallID(ctx context.Context, id uuid.UUID, channelID string) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "RemoveChannelCallID",
		"confbridge_id": id,
		"channel_id":    channelID,
	})

	if errRemove := h.db.ConfbridgeRemoveChannelCallID(ctx, id, channelID); errRemove != nil {
		log.Errorf("Could not remove the channel from the confbridge's channel/call info. err: %v", errRemove)
		return nil, errors.Wrap(errRemove, "could not remove the channel")
	}

	// get confbridge
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated confbridge. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated confbridge")
	}

	return res, nil
}

// AddChannelCallID adds the channel from the channel call id
func (h *confbridgeHandler) AddChannelCallID(ctx context.Context, id uuid.UUID, channelID string, callID uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "AddChannelCallID",
		"confbridge_id": id,
		"channel_id":    channelID,
	})

	if errAdd := h.db.ConfbridgeAddChannelCallID(ctx, id, channelID, callID); errAdd != nil {
		log.Errorf("Could not add the channel/call to the confbridge's channel/call info. err: %v", errAdd)
		return nil, errors.Wrap(errAdd, "could not add the channel call")
	}

	// get confbridge
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated confbridge. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated confbridge")
	}

	// Publish the event
	evt := &confbridge.EventConfbridgeJoined{
		Confbridge:   *res,
		JoinedCallID: callID,
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, evt)

	return res, nil
}

// Delete deletes the confbridge
func (h *confbridgeHandler) Delete(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Delete",
		"confbridge_id": id,
	})

	// update conference status to terminated
	if errDelete := h.db.ConfbridgeDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not terminate the confbridge. err: %v", errDelete)
		return nil, errDelete
	}
	promConfbridgeCloseTotal.Inc()

	// notify conference deleted event
	res, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted confbridge info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get deleted confbridge.")
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeDeleted, res)

	return res, nil
}
