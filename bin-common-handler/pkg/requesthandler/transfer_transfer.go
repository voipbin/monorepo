package requesthandler

import (
	"context"
	"encoding/json"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"
	tmrequest "monorepo/bin-transfer-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
)

// TransferV1TransferStart sends a request to transfer-manager
// to starts a transfer.
// it returns created service if it succeed.
func (r *requestHandler) TransferV1TransferStart(ctx context.Context, transferType tmtransfer.Type, transfererCallID uuid.UUID, transfereeAddresses []commonaddress.Address) (*tmtransfer.Transfer, error) {
	uri := "/v1/transfers"

	data := &tmrequest.V1DataTransfersPost{
		Type:                transferType,
		TransfererCallID:    transfererCallID,
		TransfereeAddresses: transfereeAddresses,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTransfer(ctx, uri, sock.RequestMethodPost, "transfer/transfer", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmtransfer.Transfer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
