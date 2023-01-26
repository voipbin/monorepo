package callhandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	callapplication "gitlab.com/voipbin/bin-manager/call-manager.git/models/callapplication"
)

const (
	amdStatusMachine = "MACHINE" // amd status result machine
)

func (h *callHandler) startServiceFromDefault(ctx context.Context, channelID string, data map[string]string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startServiceFromDefault",
		"channel_id": channelID,
	})
	log.Debug("Executing default service handler.")

	// get call by the channel id
	c, err := h.db.CallGetByChannelID(ctx, channelID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "coudl not get call info")
	}

	return h.ActionNext(ctx, c)
}

// startServiceFromAMD handles context-from amd service call.
func (h *callHandler) startServiceFromAMD(ctx context.Context, channelID string, data map[string]string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startServiceFromAMD",
		"channel_id": channelID,
	})
	defer func() {
		_, _ = h.channelHandler.HangingUp(ctx, channelID, ari.ChannelCauseNormalClearing)
	}()

	status := data["amd_status"]
	cause := data["amd_cause"]
	log.Debugf("Received amd result. status: %s, cause: %s", status, cause)

	// get amd option
	amd, err := h.db.CallApplicationAMDGet(ctx, channelID)
	if err != nil {
		log.Errorf("Could not get amd option. err: %v", err)
		return nil
	}
	log = log.WithField("call_id", amd.CallID)

	// check the result
	if status == amdStatusMachine && amd.MachineHandle == callapplication.AMDMachineHandleHangup {
		// hangup the call
		log.Infof("The amd option is machine hangup. machine_handle: %s", amd.MachineHandle)
		_, _ = h.HangingUp(ctx, amd.CallID, ari.ChannelCauseCallAMD)
		return nil
	}

	if !amd.Async {
		return h.reqHandler.CallV1CallActionNext(ctx, amd.CallID, false)
	}

	return nil
}
