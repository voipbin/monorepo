package engine_openai_handler

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sashabaranov/go-openai"
)

func (h *engineOpenaiHandler) Send(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	return h.send(ctx, req)
}

func (h *engineOpenaiHandler) send(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 10 * time.Second
	expBackoff.MaxElapsedTime = 1 * time.Minute

	var resp openai.ChatCompletionResponse
	operation := func() error {
		var errOp error
		resp, errOp = h.client.CreateChatCompletion(ctx, *req)
		return errOp
	}

	if errRetry := backoff.Retry(operation, expBackoff); errRetry != nil {
		return nil, errRetry
	}

	return &resp, nil
}
