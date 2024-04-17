package transferhandler

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transfer-manager/models/transfer"
)

// BlindTransfer
func (h *transferHandler) BlindTransfer(ctx context.Context, transfererCall *cmcall.Call, flow *fmflow.Flow, transfereeAddresses []commonaddress.Address) (*transfer.Transfer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                 "BlindTransfer",
		"transferer_call":      transfererCall,
		"flow":                 flow,
		"transferee_addresses": transfereeAddresses,
	})

	if errBlock := h.blindBlock(ctx, transfererCall.ConfbridgeID); errBlock != nil {
		log.Errorf("Could not block the transfer. err: %v", errBlock)
		return nil, errors.Wrap(errBlock, "could not block the transfer")
	}

	// transfer
	res, err := h.blindExecute(ctx, transfererCall, flow, transfereeAddresses)
	if err != nil {
		log.Errorf("Failed to attended transfer. err: %v", err)
		if errUnblock := h.blindUnblock(ctx, transfererCall.ConfbridgeID); errUnblock != nil {
			log.Errorf("Could not unblock the transfer. err: %v", errUnblock)
		}
		return nil, errors.Wrap(err, "failed to attended transfer")
	}

	return res, nil
}

func (h *transferHandler) blindBlock(ctx context.Context, confbridgeID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "blindBlock",
		"conbridge_id": confbridgeID,
	})

	// set confbridge flag
	cb, err := h.reqHandler.CallV1ConfbridgeFlagAdd(ctx, confbridgeID, cmconfbridge.FlagNoAutoLeave)
	if err != nil {
		log.Errorf("Could not add the confbridge flag. err: %v", err)
		return errors.Wrap(err, "could not add the confbridge flag")
	}
	log.WithField("confbridge", cb).Debugf("Added confbridge flag. confbridge_id: %s", cb.ID)

	return nil
}

func (h *transferHandler) blindUnblock(ctx context.Context, confbridgeID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "blindUnblock",
	})

	// set confbridge flag
	cb, err := h.reqHandler.CallV1ConfbridgeFlagRemove(ctx, confbridgeID, cmconfbridge.FlagNoAutoLeave)
	if err != nil {
		log.Errorf("Could not remove the confbridge flag. err: %v", err)
		return errors.Wrap(err, "could not remove the confbridge flag")
	}
	log.WithField("confbridge", cb).Debugf("Removed confbridge flag. confbridge_id: %s", cb.ID)

	return nil
}

// blindExecute starts blins transfer
func (h *transferHandler) blindExecute(ctx context.Context, transfererCall *cmcall.Call, flow *fmflow.Flow, transfereeAddresses []commonaddress.Address) (*transfer.Transfer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                 "blind",
		"transferer_call":      transfererCall,
		"flow":                 flow,
		"transferee_addresses": transfereeAddresses,
	})

	// hangup the transferer call
	_, err := h.reqHandler.CallV1CallHangup(ctx, transfererCall.ID)
	if err != nil {
		log.Errorf("Could not hangup the transferer call. err: %v", err)
		return nil, errors.Wrap(err, "could not hangup the transferer call")
	}

	// set confbridge ring
	if errRing := h.reqHandler.CallV1ConfbridgeRing(ctx, transfererCall.ConfbridgeID); errRing != nil {
		log.Errorf("Could not ring the confbridge. err: %v", errRing)
	}

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
	res, err := h.Create(ctx, transfererCall.CustomerID, transfer.TypeBlind, transfererCall.ID, transfereeAddresses, gc.ID, transfererCall.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not create transfer. err: %v", err)
		return nil, errors.Wrap(err, "could not create transfer")
	}
	log.WithField("transfer", res).Debugf("Created transfer. transfer_id: %s", res.ID)

	return res, nil
}
