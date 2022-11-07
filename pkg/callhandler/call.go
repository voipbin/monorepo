package callhandler

import (
	"context"

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

	asteriskID string,
	channelID string,
	bridgeID string,

	flowID uuid.UUID,
	activeflowID uuid.UUID,
	confbridgeID uuid.UUID,

	callType call.Type,

	masterCallID uuid.UUID,
	chainedcallIDs []uuid.UUID,

	recordingID uuid.UUID,
	recordingIDs []uuid.UUID,

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
		ID:           id,
		CustomerID:   customerID,
		
		AsteriskID:   asteriskID,
		ChannelID:    channelID,
		BridgeID:     bridgeID,
		FlowID:       flowID,
		ActiveFlowID: activeflowID,
		ConfbridgeID: confbridgeID,
		Type:         callType,

		MasterCallID:   masterCallID,
		ChainedCallIDs: chainedcallIDs,

		RecordingID:  recordingID,
		RecordingIDs: recordingIDs,

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

		TMCreate:      h.util.GetCurTime(),
		TMUpdate:      dbhandler.DefaultTimeStamp,
		TMProgressing: dbhandler.DefaultTimeStamp,
		TMRinging:     dbhandler.DefaultTimeStamp,
		TMHangup:      dbhandler.DefaultTimeStamp,
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

	// send a channel heaclth check
	_, err = h.reqHandler.AstChannelGet(ctx, c.AsteriskID, c.ChannelID)
	if err != nil {
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
