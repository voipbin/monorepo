package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	tmtransfer "gitlab.com/voipbin/bin-manager/transfer-manager.git/models/transfer"
)

// TransferStart sends a request to transfer-manager
// to start a transfer
// it returns transfer info if it succeed.
func (h *serviceHandler) TransferStart(ctx context.Context, u *cscustomer.Customer, transferType tmtransfer.Type, transfererCallID uuid.UUID, transfereeAddresses []commonaddress.Address) (*tmtransfer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                 "TransferStart",
		"customer_id":          u.ID,
		"username":             u.Username,
		"transfer_type":        transferType,
		"transferer_call_id":   transfererCallID,
		"transferee_addresses": transfereeAddresses,
	})

	_, err := h.callGet(ctx, u, transfererCallID)
	if err != nil {
		log.Infof("Could not get transferer call info. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.TransferV1TransferStart(ctx, transferType, transfererCallID, transfereeAddresses)
	if err != nil {
		log.Errorf("Could not get transcripts from the transcribe-manager. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
