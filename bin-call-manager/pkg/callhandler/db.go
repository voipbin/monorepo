package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
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

	groupcallID uuid.UUID,

	source *commonaddress.Address,
	destination *commonaddress.Address,

	status call.Status,
	data map[call.DataType]string,

	action fmaction.Action,
	direction call.Direction,

	dialrouteID uuid.UUID,
	dialroutes []rmroute.Route,
) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"id":          id,
		"customer_id": customerID,
	})

	callID := id
	if callID == uuid.Nil {
		callID = h.utilHandler.UUIDCreate()
	}

	c := &call.Call{
		ID:         callID,
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
		GroupcallID:    groupcallID,

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

	// start call health watcher
	if errHealth := h.reqHandler.CallV1CallHealth(ctx, res.ID, defaultHealthDelay, 0); errHealth != nil {
		// we could not start call health watcher correctly, but we don't stop the process.
		// just write the error message here.
		log.Errorf("Could not start the call health watcher. err: %v", errHealth)
	}

	return res, nil
}

// Gets returns list of calls.
func (h *callHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"filters": filters,
	})

	res, err := h.db.CallGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get calls. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns call.
func (h *callHandler) Get(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get call.")
	}

	return res, nil
}

// updateForRouteFailover updates the call for route failover
func (h *callHandler) updateForRouteFailover(ctx context.Context, id uuid.UUID, channelID string, dialrouteID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "updateForRouteFailover",
		"call_id":      id,
		"channel_id":   channelID,
		"dialroute_id": dialrouteID,
	})

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
	log := logrus.WithFields(logrus.Fields{
		"func":    "setAction",
		"call_id": id,
		"action":  a,
	})

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
	log := logrus.WithFields(logrus.Fields{
		"func":    "UpdateStatus",
		"call_id": id,
		"status":  status,
	})

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

	mapEvt := map[call.Status]string{
		call.StatusDialing:     call.EventTypeCallDialing,
		call.StatusRinging:     call.EventTypeCallRinging,
		call.StatusProgressing: call.EventTypeCallProgressing,
		call.StatusTerminating: call.EventTypeCallTerminating,
		call.StatusCanceling:   call.EventTypeCallCanceling,
		// call.StatusHangup:      call.EventTypeCallHangup, // this must be done with Hangup()
	}

	// send notification
	evt, ok := mapEvt[res.Status]
	if !ok {
		log.Errorf("Could not find notification event type. status: %s", res.Status)
		return res, nil
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, evt, res)

	return res, nil
}

// dbDelete deletes the call
func (h *callHandler) dbDelete(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "dbDelete",
		"call_id": id,
	})

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
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateRecordingID",
		"call_id":      id,
		"recording_id": recordingID,
	})

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
	log := logrus.WithFields(logrus.Fields{
		"func":              "UpdateExternalMediaID",
		"call_id":           id,
		"external_media_id": externalMediaID,
	})

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
	log := logrus.WithFields(logrus.Fields{
		"func":          "UpdateConfbridgeID",
		"call_id":       id,
		"confbridge_id": confbridgeID,
	})

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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallUpdated, res)

	return res, nil
}

// UpdateHangupInfo updates call's the hangup info
func (h *callHandler) UpdateHangupInfo(ctx context.Context, id uuid.UUID, reason call.HangupReason, hangupBy call.HangupBy) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "UpdateHangupInfo",
		"call_id":   id,
		"hangup_by": hangupBy,
		"reason":    reason,
	})

	if errSet := h.db.CallSetHangup(ctx, id, reason, hangupBy); errSet != nil {
		log.Errorf("Could not update the call info. err: %v", errSet)
		// we don't channel hangup here, we are assumming the channel has already gone.
		return nil, errSet
	}

	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get hungup call data. call: %s, err: %v", id, err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallHangup, res)
	promCallHangupTotal.WithLabelValues(string(res.Direction), string(res.Type), string(reason)).Inc()

	return res, nil
}

// UpdateData updates call's data
func (h *callHandler) UpdateData(ctx context.Context, id uuid.UUID, data map[call.DataType]string) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "UpdateData",
		"call_id": id,
	})

	if errSet := h.db.CallSetData(ctx, id, data); errSet != nil {
		log.Errorf("Could not update the data. err: %v", errSet)
		return nil, errSet
	}

	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call info. call_id: %s, err: %v", id, err)
		return nil, err
	}

	return res, nil
}

// UpdateMuteDirection updates call's muteDirection
func (h *callHandler) UpdateMuteDirection(ctx context.Context, id uuid.UUID, muteDirection call.MuteDirection) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "UpdateMuteDirection",
		"call_id":        id,
		"mute_direction": muteDirection,
	})

	if errSet := h.db.CallSetMuteDirection(ctx, id, muteDirection); errSet != nil {
		log.Errorf("Could not update the data. err: %v", errSet)
		return nil, errSet
	}

	res, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated call info. call_id: %s, err: %v", id, err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallUpdated, res)

	return res, nil
}
