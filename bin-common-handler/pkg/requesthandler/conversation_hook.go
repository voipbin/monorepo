package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

// ConversationV1Hook sends a hook
func (r *requestHandler) ConversationV1Hook(ctx context.Context, hm *hmhook.Hook) error {

	uri := "/v1/hooks"

	m, err := json.Marshal(hm)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodPost, "message/messages", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if tmp.StatusCode >= 299 {
		return fmt.Errorf("could not send the message")
	}

	return nil
}
