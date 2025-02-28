package openai_handler

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sashabaranov/go-openai"
)

func (h *openaiHandler) send(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 10 * time.Second
	expBackoff.MaxElapsedTime = 1 * time.Minute

	var resp openai.ChatCompletionResponse
	var err error
	operation := func() error {
		var err error
		resp, err = h.client.CreateChatCompletion(ctx, *req)
		if err != nil {
			return err
		}
		return nil
	}

	if errRetry := backoff.Retry(operation, expBackoff); errRetry != nil {
		return nil, err
	}

	return &resp, nil
}
