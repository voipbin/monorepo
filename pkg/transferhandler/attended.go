package transferhandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/transfer-manager.git/models/transfer"
)

// AttendedTransfer transfer the call in the attended mode.
func (h *transferHandler) AttendedTransfer(ctx context.Context, transfererCall *cmcall.Call, flow *fmflow.Flow, transfereeAddresses []commonaddress.Address) (*transfer.Transfer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                 "transferAttended",
		"transferer_call":      transfererCall,
		"flow":                 flow,
		"transferee_addresses": transfereeAddresses,
	})

	// preparation
	if errInit := h.attendedBlock(ctx, transfererCall); errInit != nil {
		log.Errorf("Failed to transfer preparation. err: %v", errInit)
		return nil, errors.Wrap(errInit, "failed to transfer preparation")
	}

	// execute
	res, err := h.attendedExecute(ctx, transfererCall, flow, transfereeAddresses)
	if err != nil {
		log.Errorf("Failed to attended transfer. err: %v", err)
		_ = h.attendedUnblock(ctx, transfererCall)
		return nil, errors.Wrap(err, "failed to attended transfer")
	}

	return res, nil
}

// attendedBlock handles preparation for attended transfer
func (h *transferHandler) attendedBlock(ctx context.Context, transfererCall *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":            "attendedBlock",
		"transferer_call": transfererCall,
	})

	// get confbridge info
	cb, err := h.reqHandler.CallV1ConfbridgeGet(ctx, transfererCall.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return errors.Wrap(err, "could not get confbridge")
	}

	// put call on moh and mute
	for _, callID := range cb.ChannelCallIDs {
		if callID == transfererCall.ID {
			continue
		}

		// note: put the other calls to the moh and mute on the direction in is important here
		// because moh doesn't put the call's incoming stream, it allows talking to the bridge the moh call.
		// so we have to put the call to the mute on direction in as well.

		// moh
		if errMOH := h.reqHandler.CallV1CallMusicOnHoldOn(ctx, callID); errMOH != nil {
			// could not put the call on moh but we don't return the error here
			log.Errorf("Could not put the call on moh. err: %v", errMOH)
		}

		// mute
		if errMute := h.reqHandler.CallV1CallMuteOn(ctx, callID, cmcall.MuteDirectionIn); errMute != nil {
			// could not put the call on mute direction in but we don't return the error here
			log.Errorf("Could not put the call on mute direction in. err: %v", errMute)
		}
	}

	return nil
}

// attendedUnblock handles terminates the failed attended transfer
func (h *transferHandler) attendedUnblock(ctx context.Context, transfererCall *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":            "attendedUnblock",
		"transferer_call": transfererCall,
	})

	// get confbridge info
	cb, err := h.reqHandler.CallV1ConfbridgeGet(ctx, transfererCall.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return errors.Wrap(err, "could not get confbridge")
	}

	// put call on moh and mute
	for _, callID := range cb.ChannelCallIDs {
		if callID == transfererCall.ID {
			continue
		}

		// moh
		if errMOH := h.reqHandler.CallV1CallMusicOnHoldOff(ctx, callID); errMOH != nil {
			// could not put the call on moh but we don't return the error here
			log.Errorf("Could not put the call on moh. err: %v", errMOH)
		}

		// mute
		if errMute := h.reqHandler.CallV1CallMuteOff(ctx, callID, cmcall.MuteDirectionIn); errMute != nil {
			// could not put the call on mute direction in but we don't return the error here
			log.Errorf("Could not put the call on mute direction in. err: %v", errMute)
		}
	}

	return nil
}

// attendedExecute starts attendedExecute transfer
func (h *transferHandler) attendedExecute(ctx context.Context, transfererCall *cmcall.Call, flow *fmflow.Flow, transfereeAddresses []commonaddress.Address) (*transfer.Transfer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                 "attendedExecute",
		"transferer_call":      transfererCall,
		"flow":                 flow,
		"transferee_addresses": transfereeAddresses,
	})

	// create outgoing call to the transferee
	gc, err := h.reqHandler.CallV1GroupcallCreate(
		ctx,
		transfererCall.CustomerID,
		transfererCall.Source,
		transfereeAddresses,
		flow.ID,
		transfererCall.MasterCallID,
		cmgroupcall.RingMethodRingAll,
		cmgroupcall.AnswerMethodHangupOthers,
	)
	if err != nil {
		log.Errorf("Could not create groupcall. err: %v", err)
		return nil, errors.Wrap(err, "could not create groupcall")
	}
	log.WithField("groupcall", gc).Debugf("Created groupcall. groupcall_id: %s", gc.ID)

	// create transfer
	tr, err := h.Create(ctx, transfererCall.CustomerID, transfer.TypeAttended, transfererCall.ID, transfereeAddresses, gc.ID, transfererCall.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not create transfer. err: %v", err)
		return nil, errors.Wrap(err, "could not create transfer")
	}
	log.WithField("transfer", tr).Debugf("Created transfer. transfer_id: %s", tr.ID)

	return tr, nil
}
