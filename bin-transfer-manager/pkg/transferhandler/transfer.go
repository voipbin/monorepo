package transferhandler

import (
	"context"
	"fmt"

	cmcall "monorepo/bin-call-manager/models/call"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transfer-manager/models/transfer"
)

// ServiceStart execute the transfer
func (h *transferHandler) ServiceStart(ctx context.Context, transferType transfer.Type, transfererCallID uuid.UUID, transfereeAddresses []commonaddress.Address) (*transfer.Transfer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                 "ServiceStart",
		"transfer_type":        transferType,
		"transferer_call_id":   transfererCallID,
		"transferee_addresses": transfereeAddresses,
	})

	transfererCall, flow, err := h.transferInit(ctx, transfererCallID)
	if err != nil {
		log.Errorf("Could not initiate the transfer. err: %v", err)
		return nil, errors.Wrap(err, "could not initiate the transfer")
	}

	switch transferType {

	case transfer.TypeBlind:
		return h.BlindTransfer(ctx, transfererCall, flow, transfereeAddresses)

	case transfer.TypeAttended:
		return h.AttendedTransfer(ctx, transfererCall, flow, transfereeAddresses)

	default:
		return nil, fmt.Errorf("transfer type not found. transfer_type: %s", transferType)
	}
}

// transferInit initiates the transfer
func (h *transferHandler) transferInit(ctx context.Context, transfererCallID uuid.UUID) (*cmcall.Call, *fmflow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "transferInit",
		"transferer_call_id": transfererCallID,
	})

	// get transferer call detail
	transfererCall, err := h.reqHandler.CallV1CallGet(ctx, transfererCallID)
	if err != nil {
		log.Errorf("Could not get transferer info. err: %v", err)
		return nil, nil, errors.Wrap(err, "could not get transferer info")
	}

	if transfererCall.ConfbridgeID == uuid.Nil {
		log.Errorf("Could not find confbridge id.")
		return nil, nil, fmt.Errorf("no confbridge id found")
	}

	// create flow
	f, err := h.createFlow(ctx, transfererCall)
	if err != nil {
		log.Errorf("Could not create a flow. err: %v", err)
		return nil, nil, errors.Wrap(err, "could not create a flow")
	}

	return transfererCall, f, nil
}

// createFlow creates a flow for call transfer
func (h *transferHandler) createFlow(ctx context.Context, transfererCall *cmcall.Call) (*fmflow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "createFlow",
		"transferer_call": transfererCall,
	})

	// create a new outgoing call with action
	actions := []fmaction.Action{
		{
			Type: fmaction.TypeConfbridgeJoin,
			Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{
				ConfbridgeID: transfererCall.ConfbridgeID,
			}),
		},
	}

	// create tmp flow
	res, err := h.reqHandler.FlowV1FlowCreate(ctx, transfererCall.CustomerID, fmflow.TypeTransfer, "created by transfer-manager", "temp flow for call transfer", actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create temp flow for call transfer. err: %v", err)
		return nil, errors.Wrap(err, "could not create temp flow for call transfer")
	}
	log.WithField("flow", res).Debugf("Created temp flow for call transfer. flow_id: %s", res.ID)

	return res, nil
}
