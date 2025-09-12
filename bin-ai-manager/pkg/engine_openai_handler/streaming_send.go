package engine_openai_handler

import (
	"context"
	"fmt"
	"io"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	fmaction "monorepo/bin-flow-manager/models/action"
	"strings"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

var (
	defaultMessage = openai.ChatCompletionMessage{
		Role:    string(message.RoleSystem),
		Content: defaultSystemPrompt,
	}
)

func (h *engineOpenaiHandler) StreamingSend(ctx context.Context, cc *aicall.AIcall, messages []*message.Message) (<-chan string, <-chan *fmaction.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "StreamingSend",
		"aicall_id": cc.ID,
	})

	tmpMessages := []openai.ChatCompletionMessage{
		defaultMessage,
	}

	for _, m := range messages {
		tmp := openai.ChatCompletionMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
		tmpMessages = append(tmpMessages, tmp)
	}

	// create request
	model := ai.GetEngineModelName(cc.AIEngineModel)
	if model == "" {
		model = defaultModel
	}
	req := &openai.ChatCompletionRequest{
		Model:    string(model),
		Messages: tmpMessages,
		Tools:    tools,
	}
	log = log.WithField("request", req)

	// send the request
	chanMsg, chanAction, err := h.streamingSend(ctx, req)
	if err != nil {
		log.Debugf("Could not send the request. err: %v\n", err)
		return nil, nil, errors.Wrap(err, "could not send the request")
	}

	return chanMsg, chanAction, nil
}

func (h *engineOpenaiHandler) streamingSend(ctx context.Context, req *openai.ChatCompletionRequest) (<-chan string, <-chan *fmaction.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "streamingSend",
	})

	// Set Stream: true for streaming requests
	req.Stream = true

	log.WithField("request", req).Debugf("Sending streaming chat completion request to OpenAI.")
	stream, err := h.client.CreateChatCompletionStream(ctx, *req)
	if err != nil {
		return nil, nil, fmt.Errorf("CreateChatCompletionStream error: %w", err)
	}

	// Channel to deliver streamed tokens
	chanMsg := make(chan string)
	chanTool := make(chan *fmaction.Action)

	go h.streamingResponseHandle(ctx, stream, chanMsg, chanTool)

	return chanMsg, chanTool, nil
}

func (h *engineOpenaiHandler) streamingResponseHandle(ctx context.Context, stream *openai.ChatCompletionStream, chanMsg chan string, chanTool chan *fmaction.Action) {
	log := logrus.WithFields(logrus.Fields{
		"func": "streamingResponseHandle",
	})

	defer stream.Close() // Close the stream when done
	defer close(chanMsg) // Close the channel when done
	defer close(chanTool)

	var currentSentence strings.Builder
	var currentTool strings.Builder
	var currentName string
	for {
		select {
		case <-ctx.Done():
			return

		default:
			response, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					// Stream ended, send any remaining sentence
					if currentSentence.Len() > 0 {
						chanMsg <- strings.TrimSpace(currentSentence.String())
					}

					if currentTool.Len() > 0 {
						act, err := h.toolHandle(currentName, []byte(currentTool.String()))
						if err != nil {
							log.Errorf("Could not handle tool at the end of stream. err: %v", err)
						} else {
							chanTool <- act
						}
					}
					return
				}

				log.Errorf("Could not receive from stream. err: %v", err)
				return
			}

			for _, choice := range response.Choices {
				for _, toolCall := range choice.Delta.ToolCalls {
					if toolCall.Function.Name != "" {
						if currentTool.Len() > 0 {
							act, err := h.toolHandle(currentName, []byte(currentTool.String()))
							if err != nil {
								log.Errorf("Could not handle tool at the end of stream. err: %v", err)
							} else {
								chanTool <- act
							}
						}

						currentName = toolCall.Function.Name
						currentTool.Reset()
					}

					if toolCall.Function.Arguments != "" {
						currentTool.WriteString(toolCall.Function.Arguments)
					}
				}

				if choice.Delta.Content != "" {
					if currentTool.Len() > 0 {
						act, err := h.toolHandle(currentName, []byte(currentTool.String()))
						if err != nil {
							log.Errorf("Could not handle tool at the end of stream. err: %v", err)
						} else {
							chanTool <- act
						}

						currentName = ""
						currentTool.Reset()
					}

					currentSentence.WriteString(choice.Delta.Content)

					// Look for sentence-ending characters (period, question mark, exclamation mark, newline)
					// More sophisticated sentence splitting logic can be implemented here.
					if strings.ContainsAny(choice.Delta.Content, ".?!\n") && currentSentence.Len() > 0 {
						sentence := currentSentence.String()
						trimmedSentence := strings.TrimSpace(sentence)

						if trimmedSentence != "" {
							chanMsg <- trimmedSentence // Deliver to the user
						}
						currentSentence.Reset() // Reset the buffer
					}
				}

				if choice.FinishReason != "" {
					log.Debugf("Stream finished. reason: %s", choice.FinishReason)

					if currentTool.Len() > 0 {
						act, err := h.toolHandle(currentName, []byte(currentTool.String()))
						if err != nil {
							log.Errorf("Could not handle tool at the end of stream. err: %v", err)
						} else {
							chanTool <- act
						}

						currentName = ""
						currentTool.Reset()
					}

					currentSentence.Reset()
				}
			}
		}
	}
}
