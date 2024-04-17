package transferhandler

import (
	"context"
	"fmt"

	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transfer-manager/models/transfer"
)

// TransfererHangup handles hungup transferer
func (h *transferHandler) TransfererHangup(ctx context.Context, tr *transfer.Transfer, transfererCall *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":            "TransfererHangup",
		"transfer":        tr,
		"transferer_call": transfererCall,
	})

	switch tr.Type {
	case transfer.TypeBlind:
		return h.transfererHangupTypeBlind(ctx, tr, transfererCall)

	case transfer.TypeAttended:
		return h.transfererHangupTypeAttended(ctx, tr, transfererCall)

	default:
		log.Errorf("Unsupported transfer type. transfer_type: %s", tr.Type)
		return fmt.Errorf("unsupported transfer type")
	}
}

// transfererHangupTypeBlind
func (h *transferHandler) transfererHangupTypeBlind(ctx context.Context, tr *transfer.Transfer, transfererCall *cmcall.Call) error {

	// nothing to do here
	return nil
}

// transfererHangupTypeAttended
func (h *transferHandler) transfererHangupTypeAttended(ctx context.Context, tr *transfer.Transfer, transfererCall *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":            "transfererHangupTypeAttended",
		"transfer":        tr,
		"transferer_call": transfererCall,
	})

	gc, err := h.reqHandler.CallV1GroupcallGet(ctx, tr.GroupcallID)
	if err != nil {
		log.Errorf("Could not get groupcall info. err: %v", err)
		return errors.Wrap(err, "could not get groupcall info")
	}

	if gc.AnswerCallID == uuid.Nil {
		// the transferer hangup the call
		// and no group call answer
		// need to terminate the confbridge
		log.Debugf("Could not finish the transfer. ")
		_, err = h.reqHandler.CallV1ConfbridgeTerminate(ctx, tr.ConfbridgeID)
		if err != nil {
			log.Errorf("Could not terminate the confbridge. err: %v", err)
		}
		return nil
	}

	tmp, err := h.updateTransfereeCallID(ctx, tr.ID, gc.AnswerCallID)
	if err != nil {
		log.Errorf("Could not update the transferee call id. err: %v", err)
		return errors.Wrap(err, "could not update the transferee call id")
	}
	log.WithField("transfer", tmp).Debugf("Updated transferee call id. transfer_id: %s", tmp.ID)

	if errUnblock := h.attendedUnblock(ctx, transfererCall); errUnblock != nil {
		log.Errorf("Could not unblock the attended blocking. err: %v", errUnblock)
		return errors.Wrap(errUnblock, "could not unblock the attended blocking")
	}

	return nil
}
