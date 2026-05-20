package requesthandler

import (
	"context"
	"encoding/json"

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

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// ConversationV1HookGet sends a GET hook for webhook verification (e.g. WhatsApp challenge)
func (r *requestHandler) ConversationV1HookGet(ctx context.Context, hm *hmhook.Hook) (string, error) {
	uri := "/v1/hooks"

	m, err := json.Marshal(hm)
	if err != nil {
		return "", err
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/hooks-get", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return "", err
	}

	var res string
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return "", errParse
	}

	return res, nil
}
