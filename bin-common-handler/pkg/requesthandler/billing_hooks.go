package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

// BillingV1PaddleHook sends a Paddle webhook hook to billing-manager
func (r *requestHandler) BillingV1PaddleHook(ctx context.Context, hm *hmhook.Hook) error {
	uri := "/v1/hooks/paddle"

	m, err := json.Marshal(hm)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodPost, "billing/hooks/paddle", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
