package confbridgehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/confbridge"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Create is handy function for creating a confbridge.
// it increases corresponded counter
func (h *confbridgeHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType confbridge.ReferenceType,
	referenceID uuid.UUID,
	confbridgeType confbridge.Type,
) (*confbridge.Confbridge, error) {

	id := h.utilHandler.UUIDCreate()
	cb := &confbridge.Confbridge{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Type:     confbridgeType,
		Status:   confbridge.StatusProgressing,
		BridgeID: "",
		Flags:    []confbridge.Flag{},

		ChannelCallIDs: map[string]uuid.UUID{},

		RecordingID:  uuid.Nil,
		RecordingIDs: []uuid.UUID{},

		ExternalMediaID: uuid.Nil,
	}

	// create a confbridge
	if errCreate := h.db.ConfbridgeCreate(ctx, cb); errCreate != nil {
		return nil, fmt.Errorf("could not create a conference. err: %v", errCreate)
	}
	promConfbridgeCreateTotal.Inc()

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the created conference. id: %s, customer_id: %S", id, customerID)
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeCreated, res)

	return res, nil
}

// Get returns confbridge
func (h *confbridgeHandler) Get(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	res, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the confbridge. id: %s", id)
	}

	return res, nil
}

// GetByBridgeID returns confbridge of the given bridge id
func (h *confbridgeHandler) GetByBridgeID(ctx context.Context, bridgeID string) (*confbridge.Confbridge, error) {

	res, err := h.db.ConfbridgeGetByBridgeID(ctx, bridgeID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the confbridge. bridge_id: %s", bridgeID)
	}

	return res, nil
}

// List returns confbridge of the given filters
func (h *confbridgeHandler) List(ctx context.Context, size uint64, token string, filters map[string]string) ([]*confbridge.Confbridge, error) {
	// Convert string filters to typed filters
	typedFilters := make(map[confbridge.Field]any)
	for k, v := range filters {
		typedFilters[confbridge.Field(k)] = v
	}

	res, err := h.db.ConfbridgeList(ctx, size, token, typedFilters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the confbridges. size: %d, token: %s, filters: %v", size, token, filters)
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
		return nil, errors.Wrapf(errSet, "could not set the recording id. confbridge_id: %s, recording_id: %s", id, recordingID)
	}

	if recordingID != uuid.Nil {
		// add the recording id
		log.Debugf("Adding the recording id. confbridge_id: %s, recording_id: %s", id, recordingID)
		if errAdd := h.db.ConfbridgeAddRecordingIDs(ctx, id, recordingID); errAdd != nil {
			return nil, errors.Wrapf(errAdd, "could not add the recording id. confbridge_id: %s, recording_id: %s", id, recordingID)
		}
	}

	// get updated confbridge
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the updated confbridge. confbridge_id: %s", id)
	}

	return res, nil
}

// UpdateExternalMediaID updates the confbridge's external media id.
func (h *confbridgeHandler) UpdateExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*confbridge.Confbridge, error) {

	if errSet := h.db.ConfbridgeSetExternalMediaID(ctx, id, externalMediaID); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the external media id. confbridge_id: %s, external_media_id: %s", id, externalMediaID)
	}

	// get updated confbridge
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the updated confbridge. confbridge_id: %s", id)
	}

	return res, nil
}

// UpdateBridgeID updates the confbridge's bridge id.
func (h *confbridgeHandler) UpdateBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) (*confbridge.Confbridge, error) {

	if errSet := h.db.ConfbridgeSetBridgeID(ctx, id, bridgeID); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the bridge id. confbridge_id: %s, bridge_id: %s", id, bridgeID)
	}

	// get updated confbridge
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the updated confbridge. confbridge_id: %s", id)
	}

	return res, nil
}

// RemoveChannelCallID removes the channel from the channel call id
func (h *confbridgeHandler) RemoveChannelCallID(ctx context.Context, id uuid.UUID, channelID string, callID uuid.UUID) (*confbridge.Confbridge, error) {

	if errRemove := h.db.ConfbridgeRemoveChannelCallID(ctx, id, channelID); errRemove != nil {
		return nil, errors.Wrapf(errRemove, "could not remove the channel/call from the confbridge's channel/call info. confbridge_id: %s, channel_id: %s", id, channelID)
	}

	// get confbridge
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated confbridge. confbridge_id: %s", id)
	}

	// Publish the event
	evt := &confbridge.EventConfbridgeLeaved{
		Confbridge:   *res,
		LeavedCallID: callID,
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeLeaved, evt)

	return res, nil
}

// AddChannelCallID adds the channel from the channel call id
func (h *confbridgeHandler) AddChannelCallID(ctx context.Context, id uuid.UUID, channelID string, callID uuid.UUID) (*confbridge.Confbridge, error) {

	if errAdd := h.db.ConfbridgeAddChannelCallID(ctx, id, channelID, callID); errAdd != nil {
		return nil, errors.Wrapf(errAdd, "could not add the channel/call to the confbridge's channel/call info. confbridge_id: %s, channel_id: %s, call_id: %s", id, channelID, callID)
	}

	// get confbridge
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated confbridge. confbridge_id: %s", id)
	}

	// Publish the event
	evt := &confbridge.EventConfbridgeJoined{
		Confbridge:   *res,
		JoinedCallID: callID,
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, evt)

	return res, nil
}

// dbDelete deletes the confbridge
func (h *confbridgeHandler) dbDelete(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {

	// update conference status to terminated
	if errDelete := h.db.ConfbridgeDelete(ctx, id); errDelete != nil {
		return nil, errors.Wrapf(errDelete, "could not delete the confbridge. confbridge_id: %s", id)
	}

	// notify conference deleted event
	res, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get deleted confbridge. confbridge_id: %s", id)
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeDeleted, res)

	return res, nil
}

// UpdateFlags updates the confbridge's flags
func (h *confbridgeHandler) UpdateFlags(ctx context.Context, id uuid.UUID, flags []confbridge.Flag) (*confbridge.Confbridge, error) {

	if errSet := h.db.ConfbridgeSetFlags(ctx, id, flags); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the flags. confbridge_id: %s, flags: %v", id, flags)
	}

	// notify conference deleted event
	res, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated confbridge. confbridge_id: %s", id)
	}

	return res, nil
}

// UpdateStatus updates the confbridge status
func (h *confbridgeHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status confbridge.Status) (*confbridge.Confbridge, error) {

	// update conference status to terminated
	if errSet := h.db.ConfbridgeSetStatus(ctx, id, status); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the confbridge status. confbridge_id: %s, status: %s", id, status)
	}

	// notify conference deleted event
	res, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated confbridge. confbridge_id: %s", id)
	}

	switch status {
	case confbridge.StatusTerminating:
		h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeTerminating, res)

	case confbridge.StatusTerminated:
		promConfbridgeCloseTotal.Inc()
		if res.TMCreate != nil {
			promConfbridgeDurationSeconds.WithLabelValues(string(res.Type)).Observe(time.Since(*res.TMCreate).Seconds())
		}
		h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeTerminated, res)
	}

	return res, nil
}
