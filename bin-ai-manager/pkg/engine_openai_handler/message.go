package engine_openai_handler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
)

func (h *engineOpenaiHandler) MessageSend(ctx context.Context, cc *aicall.AIcall, messages []*message.Message) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "messageSend",
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
