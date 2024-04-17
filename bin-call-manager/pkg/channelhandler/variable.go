package channelhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// VariableSet sets the variable
func (h *channelHandler) VariableSet(ctx context.Context, id string, key string, value string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "VariableSet",
		"channel_id": id,
		"variable":   key,
		"value":      value,
	})

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return err
	}

	if res.TMDelete < dbhandler.DefaultTimeStamp {
		// already hungup nothing to do
		return fmt.Errorf("already hungup")
	}

	if errSet := h.reqHandler.AstChannelVariableSet(ctx, res.AsteriskID, res.ID, key, value); errSet != nil {
		log.Errorf("Could not set the channel variable. err: %v", errSet)
		return errors.Wrap(errSet, "could not set the channel variable")
	}

	return nil
}

// variableGet gets the variable
func (h *channelHandler) variableGet(ctx context.Context, cn *channel.Channel, key string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "variableGet",
		"channel": cn,
	})

	res, err := h.reqHandler.AstChannelVariableGet(ctx, cn.AsteriskID, cn.ID, key)
	if err != nil {
		log.Errorf("Could not get variable. err: %v", err)
		return "", errors.Wrap(err, "could not get variable")
	}

	return res, nil
}
