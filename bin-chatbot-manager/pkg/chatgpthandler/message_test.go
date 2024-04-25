package chatgpthandler

import (
	"context"
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/sashabaranov/go-openai"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		messages []chatbotcall.Message
		role     string
		text     string

		expectRes []chatbotcall.Message
	}{
		{
			name: "normal",

			messages: []chatbotcall.Message{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: `Just say "yes" to all message.`,
				},
			},
			role: openai.ChatMessageRoleUser,
			text: "this is test message.",

			expectRes: []chatbotcall.Message{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: `Just say "yes" to all message.`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: `this is test message.`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := NewChatgptHandler("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")
			ctx := context.Background()

			res, err := h.MessageSend(ctx, tt.messages, tt.role, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for i := 0; i < 2; i++ {
				if !reflect.DeepEqual(tt.expectRes[i], res[i]) {
					t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[i], res[i])
				}
			}

		})
	}
}
