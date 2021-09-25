package callhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	callapplication "gitlab.com/voipbin/bin-manager/call-manager.git/models/callapplication"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

func (h *callHandler) startServiceFromDefault(cn *channel.Channel, data map[string]interface{}) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startServiceFromDefault",
		"channel_id": cn.ID,
	})
	log.Debug("Executing default service handler.")

	// get call by the channel id
	ctx := context.Background()
	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return err
	}

	return h.ActionNext(c)
}

// startServiceFromAMD handles context-from amd service call.
func (h *callHandler) startServiceFromAMD(cn *channel.Channel, data map[string]interface{}) error {

	ctx := context.Background()

	log := logrus.WithFields(logrus.Fields{
		"func":       "startServiceFromAMD",
		"channel_id": cn.ID,
	})
	defer h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)

	status := data["amd_status"]
	cause := data["amd_cause"]
	log.Debugf("Received amd result. status: %s, cause: %s", status, cause)

	// get amd option
	amd, err := h.db.CallApplicationAMDGet(ctx, cn.ID)
	if err != nil {
		log.Errorf("Could not get amd option. err: %v", err)
	}
	log = log.WithField("call_id", amd.CallID)

	// get call
	c, err := h.db.CallGet(ctx, amd.CallID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}

	if status == "MACHINE" && amd.MachineHandle == callapplication.AMDMachineHandleHangup {
		// hangup the call
		log.Infof("The amd option is machine hangup. machine_handle: %s", amd.MachineHandle)
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
	}

	if amd.Async == false {
		return h.reqHandler.CallCallActionNext(c.ID)
	}

	return nil
}
