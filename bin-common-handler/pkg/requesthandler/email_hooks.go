package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/sock"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

// EmailV1Hook sends a hook
func (r *requestHandler) EmailV1Hooks(ctx context.Context, hm *hmhook.Hook) error {

	uri := "/v1/hooks"

	m, err := json.Marshal(hm)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestEmail(ctx, uri, sock.RequestMethodPost, "email/hook", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if tmp.StatusCode >= 299 {
		return fmt.Errorf("could not send the message")
	}

	return nil
}
