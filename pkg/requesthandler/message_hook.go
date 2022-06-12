package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	hmhook "gitlab.com/voipbin/bin-manager/hook-manager.git/models/hook"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// MMV1Hook sends a hook
func (r *requestHandler) MMV1Hook(ctx context.Context, hm *hmhook.Hook) error {

	uri := "/v1/hooks"

	m, err := json.Marshal(hm)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestMM(uri, rabbitmqhandler.RequestMethodPost, resourceMMMessages, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if tmp.StatusCode >= 299 {
		return fmt.Errorf("could not send the message")
	}

	return nil
}
