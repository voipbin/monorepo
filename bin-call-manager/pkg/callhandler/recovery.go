package callhandler

import (
	"context"
	"fmt"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *callHandler) RecoveryStart(ctx context.Context, asteriskID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecoveryStart",
		"asterisk_id": asteriskID,
	})

	// get channels of the give asterisk ID
	startTime := h.utilHandler.TimeNowAdd(-(time.Hour * 24))
	endTime := h.utilHandler.TimeNow()
	channels, err := h.channelHandler.GetChannelsForRecovery(ctx, asteriskID, channel.TypeCall, startTime, endTime, defaultRecoveryChannelLimit)
	if err != nil {
		return errors.Wrapf(err, "could not get channels for recovery. asterisk_id: %s", asteriskID)
	}
	log.Debugf("Got %d channels for recovery", len(channels))

	// run recovery
	for _, ch := range channels {
		go func(innerCh *channel.Channel) {
			log := log.WithField("channel", innerCh.ID)

			if innerCh.Type != channel.TypeCall {
				// nothing to do
				return
			}

			log.Debugf("Starting recovery for channel. channel_id: %s", innerCh.ID)
			if err := h.recoveryRun(context.Background(), innerCh); err != nil {
				log.Errorf("Could not run recovery for channel. err: %v", err)
				return
			}
			// the outcome (skipped, claim refused, channel created) is logged by recoveryRun; a switched call logs
			// "Switched the call to the recovery channel" when the remote answers.
			log.Info("Recovery run finished")
		}(ch)
	}

	return nil
}

// recoveryClaimTTL bounds one recovery attempt: Homer query timeout (30 s) plus dial timeout
// (defaultDialTimeout, 60 s) plus margin. Within it, a repeated RecoveryStart does not start a second
// recovery leg for the same call (VOIP-1556).
const recoveryClaimTTL = 180 * time.Second

// recoveryRun starts the recovery of the call owned by the given channel: it reconstructs the dialog from
// Homer and dials a recovery channel that re-INVITEs the remote within the original dialog. The recovery
// channel takes the call over only after the remote answered (recoverySwitch).
func (h *callHandler) recoveryRun(ctx context.Context, ch *channel.Channel) error {
	if ch == nil {
		return errors.New("channel is nil")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":       "recoveryRun",
		"channel_id": ch.ID,
	})

	if ch.Type != channel.TypeCall {
		return fmt.Errorf("channel type is not call. channel_id: %s, channel_type: %s", ch.ID, ch.Type)
	}

	c, err := h.GetByChannelID(ctx, ch.ID)
	if err != nil {
		return errors.Wrapf(err, "could not get call by channel ID. channel_id: %s", ch.ID)
	}
	log = log.WithField("call_id", c.ID)

	if reason := recoverySkipReason(c); reason != "" {
		log.Infof("Skipping the call recovery. call_id: %s, reason: %s", c.ID, reason)
		return nil
	}

	role := asteriskRoleUAS
	if c.Direction == call.DirectionOutgoing {
		role = asteriskRoleUAC
	}

	// fail closed: without the claim (held by another recovery, or a cache error) no recovery leg is created.
	claimed, errClaim := h.db.CallRecoveryClaim(ctx, c.ID, recoveryClaimTTL)
	if errClaim != nil {
		log.Warnf("Could not take the recovery claim. Skipping the call recovery. call_id: %s, err: %v", c.ID, errClaim)
		return nil
	}
	if !claimed {
		log.Infof("Another recovery holds the claim. Skipping the call recovery. call_id: %s", c.ID)
		return nil
	}
	log.Infof("Took the recovery claim. call_id: %s", c.ID)

	recoveryDetail, err := h.recoveryHandler.GetRecoveryDetail(ctx, ch.SIPCallID, role)
	if err != nil {
		return errors.Wrapf(err, "could not get recovery detail for channel. channel_id: %s", ch.ID)
	}

	dialURI := fmt.Sprintf("pjsip/%s/%s", pjsipEndpointOutgoing, recoveryDetail.RequestURI)

	channelVariables := map[string]string{
		channelVariableRecoveryFromDisplay: recoveryDetail.FromDisplay,
		channelVariableRecoveryFromURI:     recoveryDetail.FromURI,
		channelVariableRecoveryFromTag:     recoveryDetail.FromTag,

		channelVariableRecoveryToDisplay: recoveryDetail.ToDisplay,
		channelVariableRecoveryToURI:     recoveryDetail.ToURI,
		channelVariableRecoveryToTag:     recoveryDetail.ToTag,

		channelVariableRecoveryCallID:       recoveryDetail.CallID,
		channelVariableRecoveryRoutes:       recoveryDetail.Routes,
		channelVariableRecoveryRecordRoutes: recoveryDetail.RecordRoutes,
		channelVariableRecoveryRequestURI:   recoveryDetail.RequestURI,
	}
	if recoveryDetail.CSeq > 0 {
		// no known local CSeq (UAS without any request from Asterisk): the dialog keeps its own initial CSeq.
		channelVariables[channelVariableRecoveryCSeq] = strconv.Itoa(recoveryDetail.CSeq)
	}

	// set app args
	appArgs := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s",
		channel.StasisDataTypeContextType, channel.TypeCall,
		channel.StasisDataTypeContext, channel.ContextCallRecovery,
		channel.StasisDataTypeCallID, c.ID,
		channel.StasisDataTypeRecoveryChannelID, ch.ID,
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
	log.WithField("channel", tmp).Infof("Created a recovery channel. call_id: %s, channel_id: %s", c.ID, tmp.ID)

	return nil
}

// recoverySkipReason returns why the given call is not recovered, or "" if it can be.
// Recovery covers answered single-leg calls only: side state and peers (conference, connected or groupcall
// legs, external media) are not restored, so such calls end as without recovery (VOIP-1556).
func recoverySkipReason(c *call.Call) string {
	switch {
	case c.Status != call.StatusProgressing:
		return fmt.Sprintf("call status is %s", c.Status)
	case c.Direction != call.DirectionIncoming && c.Direction != call.DirectionOutgoing:
		return fmt.Sprintf("unsupported call direction %q", c.Direction)
	case c.ConfbridgeID != uuid.Nil:
		return "call is in a confbridge"
	case len(c.ChainedCallIDs) > 0:
		return "call has chained calls"
	case c.MasterCallID != uuid.Nil:
		return "call has a master call"
	case c.GroupcallID != uuid.Nil:
		return "call is a groupcall leg"
	case len(c.ExternalMediaIDs) > 0:
		return "call has external media"
	}
	return ""
}
