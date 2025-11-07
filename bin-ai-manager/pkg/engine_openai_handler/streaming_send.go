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

var (
	defaultMessage = openai.ChatCompletionMessage{
		Role:    string(message.RoleSystem),
		Content: defaultSystemPrompt,
	}
)

func (h *engineOpenaiHandler) StreamingSend(ctx context.Context, cc *aicall.AIcall, messages []*message.Message) (<-chan string, <-chan *message.ToolCall, error) {
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
	chanMsg, chanTool, err := h.streamingSend(ctx, req)
	if err != nil {
		log.Debugf("Could not send the request. err: %v\n", err)
		return nil, nil, errors.Wrap(err, "could not send the request")
	}

	return chanMsg, chanTool, nil
}

func (h *engineOpenaiHandler) streamingSend(ctx context.Context, req *openai.ChatCompletionRequest) (<-chan string, <-chan *message.ToolCall, error) {
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
	log.Debugf("Chat completion stream created successfully.")

	// Channel to deliver streamed tokens
	chanMsg := make(chan string, 10)
	chanTool := make(chan *message.ToolCall, 10)

	go h.streamingResponseHandle(ctx, stream, chanMsg, chanTool)

	return chanMsg, chanTool, nil
}

func (h *engineOpenaiHandler) streamingResponseHandle(ctx context.Context, stream *openai.ChatCompletionStream, chanMsg chan string, chanTool chan *message.ToolCall) {
	log := logrus.WithFields(logrus.Fields{
		"func": "streamingResponseHandle",
	})

	var text strings.Builder
	var toolArg strings.Builder
	var toolName string
	var toolID string

	defer func() {
		log.Debugf("Streaming response handler is done.")

		h.streamingResponseHandleText(chanMsg, text)
		h.streamingResponseHandleTool(chanTool, toolID, toolName, toolArg)
		log.Debugf("Flushed remaining text and tool action.")

		_ = stream.Close() // Close the stream when done
		close(chanMsg)     // Close the channel when done
		close(chanTool)    // Close the channel when done
		log.Debugf("Closed channels and stream.")
	}()

	for {
		select {
		case <-ctx.Done():
			return

		default:
			response, err := stream.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					log.Errorf("Could not receive from stream. err: %v", err)
				}

				return
			}

			for _, choice := range response.Choices {
				for _, toolCall := range choice.Delta.ToolCalls {
					if toolCall.Function.Name != "" {
						h.streamingResponseHandleTool(chanTool, toolID, toolName, toolArg)

						toolID = toolCall.ID
						toolName = toolCall.Function.Name
						toolArg.Reset()
					}

					if toolCall.Function.Arguments != "" {
						toolArg.WriteString(toolCall.Function.Arguments)
					}
				}

				if choice.Delta.Content != "" {
					if toolName != "" {
						h.streamingResponseHandleTool(chanTool, toolID, toolName, toolArg)

						toolID = ""
						toolName = ""
						toolArg.Reset()
					}

					text.WriteString(choice.Delta.Content)

					// Look for sentence-ending characters (period, question mark, exclamation mark, newline)
					// More sophisticated sentence splitting logic can be implemented here.
					if strings.ContainsAny(choice.Delta.Content, ".?!\n") && text.Len() > 0 {
						h.streamingResponseHandleText(chanMsg, text)
						text.Reset() // Reset the buffer
					}
				}

				if choice.FinishReason != "" {
					log.Debugf("Stream finished. reason: %s", choice.FinishReason)
					return
				}
			}
		}
	}
}

func (h *engineOpenaiHandler) streamingResponseHandleText(chanMsg chan string, text strings.Builder) {
	if text.Len() == 0 {
		return
	}

	chanMsg <- strings.TrimSpace(text.String())
}

func (h *engineOpenaiHandler) streamingResponseHandleTool(chanTool chan *message.ToolCall, id string, name string, arg strings.Builder) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "streamingResponseHandleTool",
		"tool_name": name,
		"tool_arg":  arg.String(),
	})
	if name == "" {
		return
	}

	toolCall := &message.ToolCall{
		ID:   id,
		Type: message.ToolTypeFunction,
		// Function: message.FunctionCall{
		// 	Name:      name,
		// 	Arguments: arg.String(),
		// },
	}
	log.WithField("tool_call", toolCall).Debugf("Prepared the tool call.")

	chanTool <- toolCall
}
