package chatgpthandler

import (
	"context"

	"github.com/otiai10/openaigo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Chat sends/receives the chat messages
func (h *chatgptHandler) Chat(ctx context.Context, text string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Chat",
	})

	request := openaigo.CompletionRequestBody{
		Model:  constModel,
		Prompt: []string{text},
	}

	response, err := h.client.Completion(ctx, request)
	if err != nil {
		log.Errorf("Could not send chat message: %v", err)
		return "", errors.Wrap(err, "could not send chat message")
	}

	// response
	return response.Choices[0].Text, nil
}
