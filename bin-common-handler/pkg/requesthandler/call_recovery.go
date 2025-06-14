package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
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
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}
