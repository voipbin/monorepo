package callhandler

import (
	"context"
	"fmt"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *callHandler) Recovery(ctx context.Context, asteriskID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Recovery",
		"asteriskID": asteriskID,
	})

	// get channels of the give asterisk ID
	channels, err := h.channelHandler.GetChannelsForRecovery(ctx, asteriskID)
	if err != nil {
		return errors.Wrapf(err, "could not get channels for recovery. asterisk_id: %s", asteriskID)
	}

	// run recovery
	for _, ch := range channels {
		go func() {
			log := log.WithField("channel", ch.ID)

			if ch.Type != channel.TypeCall {
				// nothing to do
				return
			}

			log.Debugf("Starting recovery for channel. channel_id: %s", ch.ID)
			if err := h.recoveryRun(context.Background(), ch); err != nil {
				log.Errorf("Could not run recovery for channel. err: %v", err)
				return
			}
			log.Info("Recovery completed successfully")
		}()
	}

	return nil
}

func (h *callHandler) recoveryRun(ctx context.Context, ch *channel.Channel) error {
	if ch == nil {
		return errors.New("channel is nil")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":    "recoveryRun",
		"channel": ch.ID,
	})

	if ch.Type != channel.TypeCall {
		return fmt.Errorf("channel type is not call. channel_id: %s, channel_type: %s", ch.ID, ch.Type)
	}

	c, err := h.GetByChannelID(ctx, ch.ID)
	if err != nil {
		return errors.Wrapf(err, "could not get call by channel ID. channel_id: %s", ch.ID)
	}

	recoveryDetail, err := h.recoveryHandler.GetRecoveryDetail(ctx, ch.SIPCallID)
	if err != nil {
		return errors.Wrapf(err, "could not get recovery detail for channel. channel_id: %s", ch.ID)
	}

	dialURI := fmt.Sprintf("pjsip/%s/%s", pjsipEndpointOutgoing, recoveryDetail.RequestURI)

	channelVariables := map[string]string{
		"PJSIP_RECOVERY_FROM_DISPLAY": recoveryDetail.FromDisplay,
		"PJSIP_RECOVERY_FROM_URI":     recoveryDetail.FromURI,
		"PJSIP_RECOVERY_FROM_TAG":     recoveryDetail.FromTag,

		"PJSIP_RECOVERY_TO_DISPLAY": recoveryDetail.ToDisplay,
		"PJSIP_RECOVERY_TO_URI":     recoveryDetail.ToURI,
		"PJSIP_RECOVERY_TO_TAG":     recoveryDetail.ToTag,

		"PJSIP_RECOVERY_Call-ID":     recoveryDetail.CallID,
		"PJSIP_RECOVERY_CSeq":        strconv.Itoa(recoveryDetail.CSeq),
		"PJSIP_RECOVERY_Routes":      recoveryDetail.Routes,
		"PJSIP_RECOVERY_REQUEST_URI": recoveryDetail.RequestURI,
	}

	// set app args
	appArgs := fmt.Sprintf("%s=%s,%s=%s,%s=%s",
		channel.StasisDataTypeContextType, channel.TypeCall,
		channel.StasisDataTypeContext, channel.ContextCallRecovery,
		channel.StasisDataTypeCallID, c.ID,
	)
	log.WithFields(logrus.Fields{
		"variables": channelVariables,
		"app_args":  appArgs,
	}).Info("Creating channel with variables and app args")

	// create a channel
	channelID := h.utilHandler.UUIDCreate().String()
	tmp, err := h.channelHandler.StartChannel(ctx, requesthandler.AsteriskIDCall, channelID, appArgs, dialURI, "", "", "", channelVariables)
	if err != nil {
		log.Errorf("Could not create a channel for outgoing call. err: %v", err)
		return err
	}
	log.WithField("channel", tmp).Debugf("Created a new channel. channel_id: %s", tmp.ID)

	return nil
}
