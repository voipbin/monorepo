package chatgpthandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
)

// messageSend send the message to the openai
func (h *chatgptHandler) messageSend(ctx context.Context, cc *chatbotcall.Chatbotcall, message *chatbotcall.Message) (*chatbotcall.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "messageSend",
		"chatbotcall": cc,
		"message":     message,
	})

	// create message array of old messages
	tmpMessages := []openai.ChatCompletionMessage{}
	for _, m := range cc.Messages {
		tmp := openai.ChatCompletionMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
		tmpMessages = append(tmpMessages, tmp)
	}

	// add the new message
	tmpMessage := openai.ChatCompletionMessage{
		Role:    string(message.Role),
		Content: message.Content,
	}
	tmpMessages = append(tmpMessages, tmpMessage)

	// create request
	model := cc.ChatbotEngineModel
	if model == "" {
		model = defaultModel
	}
	req := openai.ChatCompletionRequest{
		Model:    string(model),
		Messages: tmpMessages,
	}

	// send the request
	resp, err := h.client.CreateChatCompletion(ctx, req)
	if err != nil {
		log.Debugf("Could not send the request. err: %v\n", err)
		return nil, errors.Wrap(err, "could not send the request")
	}

	res := &chatbotcall.Message{
		Role:    chatbotcall.MessageRole(resp.Choices[0].Message.Role),
		Content: resp.Choices[0].Message.Content,
	}

	return res, nil
}
