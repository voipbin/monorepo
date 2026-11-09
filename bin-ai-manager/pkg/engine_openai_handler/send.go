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

// send resolves the key once, before the first attempt, and the retries use the client built with
// it. The backoff does not take a context, so a request that started before a key rotation keeps
// the old key for up to its one minute of retries; the next call uses the new key.
func (h *engineOpenaiHandler) send(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	client, err := h.newClient(ctx)
	if err != nil {
		return nil, err
	}

	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 10 * time.Second
	expBackoff.MaxElapsedTime = 1 * time.Minute

	var resp openai.ChatCompletionResponse
	operation := func() error {
		var errOp error
		resp, errOp = client.CreateChatCompletion(ctx, *req)
		return errOp
	}

	if errRetry := backoff.Retry(operation, expBackoff); errRetry != nil {
		return nil, errRetry
	}

	return &resp, nil
}
