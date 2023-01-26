package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// Create creates a call record.
func (h *callHandler) Create(
	ctx context.Context,

	id uuid.UUID,
	customerID uuid.UUID,

	channelID string,
	bridgeID string,

	flowID uuid.UUID,
	activeflowID uuid.UUID,
	confbridgeID uuid.UUID,

	callType call.Type,

	source *commonaddress.Address,
	destination *commonaddress.Address,

	status call.Status,
	data map[string]string,

	action fmaction.Action,
	direction call.Direction,

	dialrouteID uuid.UUID,
	dialroutes []rmroute.Route,
) (*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Create",
			"id":          id,
			"customer_id": customerID,
		},
	)

	c := &call.Call{
		ID:         id,
		CustomerID: customerID,

		ChannelID:    channelID,
		BridgeID:     bridgeID,
		FlowID:       flowID,
		ActiveFlowID: activeflowID,
		ConfbridgeID: confbridgeID,
		Type:         callType,

		MasterCallID:   uuid.Nil,
		ChainedCallIDs: []uuid.UUID{},
		RecordingID:    uuid.Nil,
		RecordingIDs:   []uuid.UUID{},

		ExternalMediaID: uuid.Nil,

		Source:      *source,
		Destination: *destination,

		Status:       status,
		Data:         data,
		Action:       action,
		Direction:    direction,
		HangupBy:     call.HangupByNone,
		HangupReason: call.HangupReasonNone,

		DialrouteID: dialrouteID,
		Dialroutes:  dialroutes,
	}
	log.WithField("call", c).Debugf("Creating a new call. call_id: %s", c.ID)

	if err := h.db.CallCreate(ctx, c); err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}
	promCallCreateTotal.WithLabelValues(string(c.Direction), string(c.Type)).Inc()

	res, err := h.db.CallGet(ctx, c.ID)
	if err != nil {
		log.Errorf("Could not get a created call. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallCreated, res)

	return res, nil
}

// Gets returns list of calls.
func (h *callHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Gets",
			"customer_id": customerID,
		},
	)

	res, err := h.db.CallGets(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get calls. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns call.
func (h *callHandler) Get(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "Get",
			"call_id": id,
		},
	)

	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get call. err: %v", err)
		return nil, err
	}

	return res, nil
}

// updateForRouteFailover updates the call for route failover
func (h *callHandler) updateForRouteFailover(ctx context.Context, id uuid.UUID, channelID string, dialrouteID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "updateForRouteFailover",
			"call_id": id,
		},
	)

	if errSet := h.db.CallSetForRouteFailover(ctx, id, channelID, dialrouteID); errSet != nil {
		log.Errorf("Could not update the call. err: %v", errSet)
		return nil, errSet
	}

	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallUpdated, res)

	return res, nil
}

// CallHealthCheck checks the given call is still vaild
func (h *callHandler) CallHealthCheck(ctx context.Context, id uuid.UUID, retryCount int, delay int) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "CallHealthCheck",
			"call_id": id,
		},
	)

	c, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not call info. err: %v", err)
		return
	}

	ch, err := h.channelHandler.Get(ctx, c.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return
	}

	// check the channel is valid or not
	if ch.TMDelete < dbhandler.DefaultTimeStamp {
		retryCount++
	} else {
		retryCount = 0
	}

	// send another health check.
	if err := h.reqHandler.CallV1CallHealth(ctx, id, delay, retryCount); err != nil {
		log.Errorf("Could not send the call health check request. err: %v", err)
		return
	}
}

// updateActionNextHold sets the action next hold
func (h *callHandler) updateActionNextHold(ctx context.Context, id uuid.UUID, hold bool) error {

	// set hold
	if err := h.db.CallSetActionNextHold(ctx, id, hold); err != nil {
		return fmt.Errorf("could not set action next hold. call_id: %s, err: %v", id, err)
	}

	return nil
}

// updateActionAndActionNextHold sets the action to the call
func (h *callHandler) updateActionAndActionNextHold(ctx context.Context, id uuid.UUID, a *fmaction.Action) (*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "setAction",
			"call_id": id,
		},
	)

	// set action
	if err := h.db.CallSetActionAndActionNextHold(ctx, id, a, false); err != nil {
		return nil, fmt.Errorf("could not set the action for call. call_id: %s, err: %v", id, err)
	}

	// get updated call
	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallUpdated, res)

	promCallActionTotal.WithLabelValues(string(a.Type)).Inc()

	return res, nil
}

// UpdateStatus sets the action to the call
func (h *callHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status call.Status) (*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "updateStatus",
			"call_id": id,
			"status":  status,
		},
	)

	var err error
	switch status {
	case call.StatusRinging:
		err = h.db.CallSetStatusRinging(ctx, id)

	case call.StatusProgressing:
		err = h.db.CallSetStatusProgressing(ctx, id)

	default:
		err = h.db.CallSetStatus(ctx, id, status)
	}
	if err != nil {
		log.Errorf("Could not update the call status. call_id: %s, status: %v, err: %v", id, status, err)
		return nil, err
	}

	// get updated call
	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallUpdated, res)

	return res, nil
}

// Delete deletes the call
func (h *callHandler) Delete(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "Delete",
			"call_id": id,
		},
	)

	if errDel := h.db.CallDelete(ctx, id); errDel != nil {
		log.Errorf("Could not delete the call. err: %v", errDel)
		return nil, errDel
	}

	// get deleted call
	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallDeleted, res)

	return res, nil
}

// UpdateRecordingID updates the call's recording id.
// if the recording id is not uuid.Nil, it also adds to the recording_ids
func (h *callHandler) UpdateRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "UpdateRecordingID",
			"call_id":      id,
			"recording_id": recordingID,
		},
	)

	if errSet := h.db.CallSetRecordingID(ctx, id, recordingID); errSet != nil {
		log.Errorf("Could not set the recording id. err: %v", errSet)
		return nil, errSet
	}

	if recordingID != uuid.Nil {
		// add the recording id
		log.Debugf("Adding the recording id. call_id: %s, recording_id: %s", id, recordingID)
		if errAdd := h.db.CallAddRecordingIDs(ctx, id, recordingID); errAdd != nil {
			log.Errorf("Could not add the recording id. err: %v", errAdd)
			return nil, errAdd
		}
	}

	// get updated call
	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateExternalMediaID updates the call's external media id.
func (h *callHandler) UpdateExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "UpdateExternalMediaID",
			"call_id":           id,
			"external_media_id": externalMediaID,
		},
	)

	if errSet := h.db.CallSetExternalMediaID(ctx, id, externalMediaID); errSet != nil {
		log.Errorf("Could not set the external media id. err: %v", errSet)
		return nil, errSet
	}

	// get updated call
	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateConfbridgeID updates the call's confbridge id.
func (h *callHandler) UpdateConfbridgeID(ctx context.Context, id uuid.UUID, confbridgeID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "UpdateConfbridgeID",
			"call_id":       id,
			"confbridge_id": confbridgeID,
		},
	)

	if errSet := h.db.CallSetConfbridgeID(ctx, id, confbridgeID); errSet != nil {
		log.Errorf("Could not set the external media id. err: %v", errSet)
		return nil, errSet
	}

	// get updated call
	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call. err: %v", err)
		return nil, err
	}

	return res, nil
}
