package chatgpthandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
)

// MessageSend send the message to the openai
func (h *chatgptHandler) MessageSend(ctx context.Context, messages []chatbotcall.Message, role string, text string) ([]chatbotcall.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "MessageSend",
		"role": role,
		"text": text,
	})

	// create message array of old messages
	tmpMessages := []openai.ChatCompletionMessage{}
	for _, m := range messages {
		tmp := openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
		tmpMessages = append(tmpMessages, tmp)
	}

	// add the new message
	tmpMessage := openai.ChatCompletionMessage{
		Role:    role,
		Content: text,
	}
	tmpMessages = append(tmpMessages, tmpMessage)

	// create request
	req := openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo,
		Messages: tmpMessages,
	}

	// send the request
	resp, err := h.client.CreateChatCompletion(ctx, req)
	if err != nil {
		log.Debugf("Could not send the request. err: %v\n", err)
		return nil, errors.Wrap(err, "could not send the request")
	}

	// result
	tmpRes := []chatbotcall.Message{
		{
			Role:    role,
			Content: text,
		},
		{
			Role:    resp.Choices[0].Message.Role,
			Content: resp.Choices[0].Message.Content,
		},
	}

	res := append(messages, tmpRes...)
	return res, nil
}
