package engine_openai_handler

import (
	"context"
	"fmt"
	"io"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"strings"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

func (h *engineOpenaiHandler) StreamingSend(ctx context.Context, cc *aicall.AIcall, messages []*message.Message) (<-chan string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "StreamingSend",
		"aicall_id": cc.ID,
	})

	tmpMessages := []openai.ChatCompletionMessage{}
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
	chanMsg, err := h.streamingSend(ctx, req)
	if err != nil {
		log.Debugf("Could not send the request. err: %v\n", err)
		return nil, errors.Wrap(err, "could not send the request")
	}

	return chanMsg, nil
}

func (h *engineOpenaiHandler) streamingSend(ctx context.Context, req *openai.ChatCompletionRequest) (<-chan string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "streamingSend",
	})

	// Set Stream: true for streaming requests
	req.Stream = true

	stream, err := h.client.CreateChatCompletionStream(ctx, *req)
	if err != nil {
		return nil, fmt.Errorf("CreateChatCompletionStream error: %w", err)
	}

	// Channel to deliver streamed tokens
	outputChan := make(chan string)

	go func() {
		defer stream.Close()    // Close the stream when done
		defer close(outputChan) // Close the channel when done

		var currentSentence strings.Builder
		for {
			select {
			case <-ctx.Done():
				return

			default:
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					// Stream ended, send any remaining sentence
					if currentSentence.Len() > 0 {
						outputChan <- strings.TrimSpace(currentSentence.String())
					}
					return
				}
				if err != nil {
					// Handle stream error
					log.Errorf("Could not receive from stream. err: %v", err)
					return
				}

				// Process only the first Choice (usually there's only one)
				for _, choice := range response.Choices {
					log.Debugf("Received choice. choice: %v", choice.Delta)

					if choice.Delta.FunctionCall != nil {
						log.Debugf("Function call: %v", choice.Delta.FunctionCall)
					}

					if choice.Delta.ToolCalls != nil {
						log.Debugf("Tool calls: %v", choice.Delta.ToolCalls)
					}

					if choice.Delta.Content != "" {
						currentSentence.WriteString(choice.Delta.Content)

						// Look for sentence-ending characters (period, question mark, exclamation mark, newline)
						// More sophisticated sentence splitting logic can be implemented here.
						if strings.ContainsAny(choice.Delta.Content, ".?!\n") && currentSentence.Len() > 0 {
							sentence := currentSentence.String()
							trimmedSentence := strings.TrimSpace(sentence)

							if trimmedSentence != "" {
								outputChan <- trimmedSentence // Deliver to the user
							}
							currentSentence.Reset() // Reset the buffer
						}
					}
				}
			}
		}
	}()

	return outputChan, nil
}
