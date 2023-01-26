package channelhandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// VariableSet sets the variable
func (h *channelHandler) VariableSet(ctx context.Context, id string, key string, value string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "VariableSet",
			"channel_id": id,
		},
	)

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
		return errSet
	}

	return nil
}

// VariableGet gets the variable
func (h *channelHandler) VariableGet(ctx context.Context, id string, key string) (string, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "VariableGet",
			"channel_id": id,
		},
	)

	tmp, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return "", err
	}

	if tmp.TMDelete < dbhandler.DefaultTimeStamp {
		// already hungup nothing to do
		return "", fmt.Errorf("already hungup")
	}

	res, err := h.reqHandler.AstChannelVariableGet(ctx, tmp.AsteriskID, tmp.ID, key)
	if err != nil {
		return "", err
	}

	return res, nil
}
