package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"
	tmrequest "monorepo/bin-transfer-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
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

	tmp, err := r.sendRequestTransfer(ctx, uri, rabbitmqhandler.RequestMethodPost, "transfer/transfer", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtransfer.Transfer
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
