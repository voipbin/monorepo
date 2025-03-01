package openai_handler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/models/message"
)

// messageSend send the message to the openai
func (h *openaiHandler) messageSend(ctx context.Context, cc *chatbotcall.Chatbotcall, message *chatbotcall.Message) (*chatbotcall.Message, error) {
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
	req := &openai.ChatCompletionRequest{
		Model:    string(model),
		Messages: tmpMessages,
	}

	// send the request
	tmp, err := h.send(ctx, req)
	if err != nil {
		log.Debugf("Could not send the request. err: %v\n", err)
		return nil, errors.Wrap(err, "could not send the request")
	}

	var res *chatbotcall.Message
	if tmp == nil || len(tmp.Choices) == 0 {
		log.Debugf("Received response with empty choices")

		res = &chatbotcall.Message{
			Role:    chatbotcall.MessageRoleAssistant,
			Content: "",
		}
	} else {
		res = &chatbotcall.Message{
			Role:    chatbotcall.MessageRole(tmp.Choices[0].Message.Role),
			Content: tmp.Choices[0].Message.Content,
		}
	}

	return res, nil
}

func (h *openaiHandler) MessageSend(ctx context.Context, cc *chatbotcall.Chatbotcall, messages []*message.Message) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "messageSend",
		"chatbotcall_id": cc.ID,
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
	model := chatbot.GetEngineModelName(cc.ChatbotEngineModel)
	if model == "" {
		model = defaultModel
	}
	req := &openai.ChatCompletionRequest{
		Model:    string(model),
		Messages: tmpMessages,
	}
	log = log.WithField("request", req)

	// send the request
	tmp, err := h.send(ctx, req)
	if err != nil {
		log.Debugf("Could not send the request. err: %v\n", err)
		return nil, errors.Wrap(err, "could not send the request")
	}

	if tmp == nil || len(tmp.Choices) == 0 {
		log.Debugf("Received response with empty choices")
		res := &message.Message{
			Role:    message.RoleNone,
			Content: "",
		}
		return res, nil
	}

	res := &message.Message{
		Role:    message.Role(tmp.Choices[0].Message.Role),
		Content: tmp.Choices[0].Message.Content,
	}
	return res, nil
}
