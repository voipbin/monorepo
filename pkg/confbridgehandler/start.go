package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// Create is handy function for creating a confbridge.
// it increases corresponded counter
func (h *confbridgeHandler) Create(ctx context.Context) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Create",
		},
	)

	id := uuid.Must(uuid.NewV4())

	cb := &confbridge.Confbridge{
		ID: id,

		RecordingIDs:   []uuid.UUID{},
		ChannelCallIDs: map[string]uuid.UUID{},

		TMCreate: getCurTime(),
		TMUpdate: defaultTimeStamp,
		TMDelete: defaultTimeStamp,
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

// StartContextIncoming handles the call which has CONTEXT=conf-in in the StasisStart argument.
func (h *confbridgeHandler) StartContextIncoming(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "StasisStartContextIncoming",
			"channel_id": cn.ID,
		},
	)
	log.Debugf("Detail data info. data: %v", data)

	if err := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeConfbridge)); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		log.Errorf("Could not set channel var. err: %v", err)
		return fmt.Errorf("could not set channel var. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
	}

	// answer the call. it is safe to call this for answered call.
	if err := h.reqHandler.AstChannelAnswer(ctx, cn.AsteriskID, cn.ID); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		log.Errorf("Could not answer the call. err: %v", err)
		return err
	}

	// check the required variable has set
	if data["confbridge_id"] == "" || data["call_id"] == "" {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		log.Errorf("Could not get confbridge or call info.")
		return fmt.Errorf("could not get confbridge id or call id info")
	}
	// get required variable
	confbridgeID := uuid.FromStringOrNil(data["confbridge_id"])
	callID := uuid.FromStringOrNil(data["call_id"])
	if confbridgeID == uuid.Nil || callID == uuid.Nil {
		log.Errorf("Could not get confbridge or call info. confbridge_id: %s, call_id: %s", confbridgeID, callID)
		return fmt.Errorf("could not get confbridge id or call id info")
	}

	// add the channel to the bridge
	if err := h.reqHandler.AstBridgeAddChannel(ctx, cn.AsteriskID, cn.DestinationNumber, cn.ID, "", false, false); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		log.Errorf("Could not add the channel to the bridge. err: %v", err)
		return fmt.Errorf("could not put the channel to the bridge. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
	}

	return nil
}
