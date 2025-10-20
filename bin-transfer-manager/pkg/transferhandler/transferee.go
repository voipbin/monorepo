package transferhandler

import (
	"context"
	"fmt"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transfer-manager/models/transfer"
)

// TransfereeHangup handles hungup transferee
func (h *transferHandler) TransfereeHangup(ctx context.Context, tr *transfer.Transfer, gc *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "TransfereeHangup",
		"transfer":  tr,
		"groupcall": gc,
	})

	switch tr.Type {
	case transfer.TypeBlind:
		return h.transfereeHangupTypeBlind(ctx, tr, gc)

	case transfer.TypeAttended:
		return h.transfereeHangupTypeAttended(ctx, tr, gc)

	default:
		log.Errorf("Unsupported transfer type. transfer_type: %s", tr.Type)
		return fmt.Errorf("unsupported transfer type")
	}
}

// transfereeHangupTypeBlind
func (h *transferHandler) transfereeHangupTypeBlind(ctx context.Context, tr *transfer.Transfer, gc *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "transfereeHangupTypeBlind",
		"transfer":  tr,
		"groupcall": gc,
	})

	if tr.TransfereeCallID != uuid.Nil {
		log.Debugf("Transfer has transferee call id. Nothing to do here.")
		return nil
	}

	// no one answered the groupcall
	// we need to delete the confbridge manually
	log.Debugf("The blind transfer failed. Transferee didn't answer the call. transfer_id: %s", tr.ID)
	_, err := h.reqHandler.CallV1ConfbridgeTerminate(ctx, tr.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not terminate the confbridge.")
	}

	return nil
}

// transfereeHangupTypeAttended
func (h *transferHandler) transfereeHangupTypeAttended(ctx context.Context, tr *transfer.Transfer, gc *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "transfereeHangupTypeAttended",
		"transfer":  tr,
		"groupcall": gc,
	})

	if tr.TransfereeCallID != uuid.Nil {
		log.Debugf("Transfer has transferee call id. Nothing to do here.")
		return nil
	}

	// the transferee hangup the call faster than transferer
	// in the attended transfer mode, we should unblock the attended block here.
	transfererCall, err := h.reqHandler.CallV1CallGet(ctx, tr.TransfererCallID)
	if err != nil {
		log.Errorf("Could not get transferer call info. err: %v", err)
		return errors.Wrap(err, "could not get transferer call info")
	}

	if errUnblock := h.attendedUnblock(ctx, transfererCall); errUnblock != nil {
		log.Errorf("Could not unblock the attended block. err: %v", errUnblock)
		return errors.Wrap(errUnblock, "could not unblock the attended block")
	}

	return nil
}

// TransfereeAnswer handles answered transferee
func (h *transferHandler) TransfereeAnswer(ctx context.Context, tr *transfer.Transfer, gc *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "TransfereeAnswer",
		"transfer":  tr,
		"groupcall": gc,
	})

	switch tr.Type {
	case transfer.TypeBlind:
		return h.TransfereeAnswerTypeBlind(ctx, tr, gc)

	case transfer.TypeAttended:
		return h.TransfereeAnswerTypeAttended(ctx, tr, gc)

	default:
		log.Errorf("Unsupported transfer type. transfer_type: %v", tr.Type)
		return fmt.Errorf("unsupported transfer type")
	}
}

func (h *transferHandler) TransfereeAnswerTypeBlind(ctx context.Context, tr *transfer.Transfer, gc *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "TransfereeAnswerTypeBlind",
		"transfer":  tr,
		"groupcall": gc,
	})

	if errUnblock := h.blindUnblock(ctx, tr.ConfbridgeID); errUnblock != nil {
		log.Errorf("Could not unblock the blind transfer. err: %v", errUnblock)
	}

	tmp, err := h.updateTransfereeCallID(ctx, tr.ID, gc.AnswerCallID)
	if err != nil {
		log.Errorf("Could not update the transferee call id. err: %v", err)
		return errors.Wrap(err, "could not update the transferee call id")
	}
	log.WithField("transfer", tmp).Debugf("Updated transferee call id. transfer_id: %s", tmp.ID)

	if errAnswer := h.reqHandler.CallV1ConfbridgeAnswer(ctx, tr.ConfbridgeID); errAnswer != nil {
		log.Errorf("Could not answer the confbridge. err: %v", errAnswer)
	}

	return nil

}

func (h *transferHandler) TransfereeAnswerTypeAttended(ctx context.Context, tr *transfer.Transfer, gc *cmgroupcall.Groupcall) error {
	// nothing to do

	return nil

}
