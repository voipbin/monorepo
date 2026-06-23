package analysishandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/analysis"
)

// Run executes a single structured-output (json_schema, strict) chat completion.
//
// It validates the request, resolves the model against the allow-set, builds a
// strict json_schema chat completion request, and returns the schema-conformant
// JSON plus accounting. It persists nothing and does not retry beyond the engine
// handler's own backoff; the caller decides what to do with errors.
func (h *analysisHandler) Run(ctx context.Context, req *analysis.Request) (*analysis.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})

	if req == nil {
		return nil, errors.New("request is nil")
	}

	// schema is mandatory: the whole point is shape-enforced output.
	if len(req.Schema) == 0 {
		return nil, errors.New("schema is required")
	}
	if req.SchemaName == "" {
		return nil, errors.New("schema_name is required")
	}

	// input size backstop (the caller is responsible for fitting under this).
	inputBytes := len(req.Prompt) + len(req.Data)
	if h.maxInputBytes > 0 && inputBytes > h.maxInputBytes {
		return nil, errors.Errorf("input too large: %d bytes (max %d)", inputBytes, h.maxInputBytes)
	}

	// resolve the model against the allow-set; unknown -> default.
	model := h.defaultModel
	if req.Model != "" {
		if h.allowedModels[req.Model] {
			model = req.Model
		} else {
			log.Warnf("Requested model is not in the allow-set. Falling back to default. requested: %s, default: %s", req.Model, h.defaultModel)
		}
	}

	promAnalysisGatewayRunTotal.WithLabelValues(model).Inc()
	timer := prometheusTimer(model)
	defer timer()

	userContent := ""
	if len(req.Data) > 0 {
		userContent = string(req.Data)
	}

	chatReq := &openai.ChatCompletionRequest{
		Model:     model,
		MaxTokens: h.maxOutputToks,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: req.Prompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userContent,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   req.SchemaName,
				Schema: json.RawMessage(req.Schema),
				// Strict json_schema is not fully supported by the Gemini
				// OpenAI-compatible endpoint (the analysis gateway provider).
				// Matches the proven geminiaudithandler behavior.
				Strict: false,
			},
		},
	}

	// reasoning_effort="none" disables Gemini 2.5 "thinking" so the whole
	// max_tokens budget is available for the JSON output. Without this, large
	// staged inputs make the model spend the budget on internal reasoning and the
	// JSON is truncated (finish_reason=length -> unmarshal failure). Only set when
	// configured, so an OpenAI rollback can clear it.
	if h.reasoningEffort != "" {
		chatReq.ReasoningEffort = h.reasoningEffort
	}

	resp, err := h.engineOpenaiHandler.Send(ctx, chatReq)
	if err != nil {
		return nil, errors.Wrapf(err, "could not run the chat completion")
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("the chat completion returned no choices")
	}

	choice := resp.Choices[0]
	finishReason := string(choice.FinishReason)

	res := &analysis.Response{
		Result:       json.RawMessage(choice.Message.Content),
		Model:        model,
		FinishReason: finishReason,
		Truncated:    choice.FinishReason == openai.FinishReasonLength,
		PromptTokens: resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
	}

	log.WithFields(logrus.Fields{
		"model":         model,
		"finish_reason": finishReason,
		"prompt_tokens": res.PromptTokens,
		"output_tokens": res.OutputTokens,
	}).Debugf("Finished the analysis gateway run.")

	return res, nil
}

// prometheusTimer returns a function that observes the elapsed duration when
// called.
func prometheusTimer(model string) func() {
	t := prometheus.NewTimer(promAnalysisGatewayRunDuration.WithLabelValues(model))
	return func() {
		t.ObserveDuration()
	}
}
