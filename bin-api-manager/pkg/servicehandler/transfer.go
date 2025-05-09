package servicehandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// TransferStart sends a request to transfer-manager
// to start a transfer
// it returns transfer info if it succeed.
func (h *serviceHandler) TransferStart(ctx context.Context, a *amagent.Agent, transferType tmtransfer.Type, transfererCallID uuid.UUID, transfereeAddresses []commonaddress.Address) (*tmtransfer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                 "TransferStart",
		"customer_id":          a.CustomerID,
		"username":             a.Username,
		"transfer_type":        transferType,
		"transferer_call_id":   transfererCallID,
		"transferee_addresses": transfereeAddresses,
	})

	c, err := h.callGet(ctx, transfererCallID)
	if err != nil {
		log.Infof("Could not get transferer call info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.TransferV1TransferStart(ctx, transferType, transfererCallID, transfereeAddresses)
	if err != nil {
		log.Errorf("Could not get transcripts from the transcribe-manager. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
