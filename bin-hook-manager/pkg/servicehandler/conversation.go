package servicehandler

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"

	hmhook "monorepo/bin-hook-manager/models/hook"
)

// Conversation handles message receive for conversation
func (h *serviceHandler) Conversation(ctx context.Context, r *http.Request) (string, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Conversation",
		},
	)

	data := []byte{}
	if r.Body != nil {
		var err error
		data, err = io.ReadAll(r.Body)
		if err != nil {
			return "", fmt.Errorf("could not read request body: %w", err)
		}
	}

	req := &hmhook.Hook{
		ReceviedURI:      r.Host + r.URL.RequestURI(),
		ReceivedData:     data,
		ReceivedMethod:   r.Method,
		ReceivedSignature: r.Header.Get("X-Hub-Signature-256"),
	}

	log.WithField("request", req).Debugf("Sending a hook message.")

	if r.Method == http.MethodGet {
		challenge, err := h.reqHandler.ConversationV1HookGet(ctx, req)
		if err != nil {
			return "", fmt.Errorf("could not send the hook get: %w", err)
		}
		return challenge, nil
	}

	if err := h.reqHandler.ConversationV1Hook(ctx, req); err != nil {
		return "", fmt.Errorf("could not send the hook: %w", err)
	}

	return "", nil
}
