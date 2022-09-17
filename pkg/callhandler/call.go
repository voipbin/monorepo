package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// create creates a call record. All of call creation process need to use this.
func (h *callHandler) create(ctx context.Context, c *call.Call) (*call.Call, error) {

	// set default time stamp
	c.TMUpdate = dbhandler.DefaultTimeStamp
	c.TMRinging = dbhandler.DefaultTimeStamp
	c.TMProgressing = dbhandler.DefaultTimeStamp
	c.TMHangup = dbhandler.DefaultTimeStamp

	if err := h.db.CallCreate(ctx, c); err != nil {
		return nil, err
	}
	promCallCreateTotal.WithLabelValues(string(c.Direction), string(c.Type)).Inc()

	res, err := h.db.CallGet(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallCreated, res)

	return res, nil
}

// Gets returns list of calls.
func (h *callHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*call.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "Gets",
			"user_id": customerID,
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
