package requesthandler

import (
	"context"
	"encoding/json"
	cmrequest "monorepo/bin-call-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
)

// CallV1RecoveryStart sends a request to call-manager
// to start a session recovery.
func (r *requestHandler) CallV1RecoveryStart(ctx context.Context, asteriskID string) error {
	uri := "/v1/recovery"

	reqData := &cmrequest.V1DataRecoveryPost{
		AsteriskID: asteriskID,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/recovery", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
