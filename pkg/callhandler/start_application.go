package callhandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// list of service dialplan context
const (
	serviceContextAMD = "svc-amd"
)

// startHandlerContextApplication handles contextApplication context type of StasisStart event.
func (h *callHandler) applicationHandleAMD(ctx context.Context, channelID string, data map[channel.StasisDataType]string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "applicationHandleAMD",
		"channel_id":  channelID,
		"stasis_data": data,
	})
	log.Debug("Executing the applciationHandleAMD.")

	// put the cahnnel to the amd
	if errContinue := h.channelHandler.Continue(ctx, channelID, serviceContextAMD, "", 0, ""); errContinue != nil {
		log.Errorf("Could not continue the channel. err: %v", errContinue)
		return errors.Wrap(errContinue, "could not continue the channel")
	}

	return nil
}
