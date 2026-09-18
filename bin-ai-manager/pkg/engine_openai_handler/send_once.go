package engine_openai_handler

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

// SendOnce issues a single ChatCompletion request without any backoff/retry
// (unlike Send, which wraps the call in backoff.Retry with a 1-minute
// MaxElapsedTime). This is used by callers that need a real, short
// context-timeout to hold (e.g. the summary output-language verification
// harness): because Send's backoff loop does not use backoff.WithContext, a
// context.WithTimeout wrapped around Send would not actually bound the call.
// SendOnce calls client.CreateChatCompletion exactly once so the caller's
// context deadline is honored. nil/empty-choices handling is left to the caller.
func (h *engineOpenaiHandler) SendOnce(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	resp, err := h.client.CreateChatCompletion(ctx, *req)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
