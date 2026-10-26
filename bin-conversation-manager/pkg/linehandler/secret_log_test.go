package linehandler

import (
	"context"
	"testing"

	"github.com/line/line-bot-sdk-go/v7/linebot"

	"monorepo/bin-conversation-manager/internal/logtest"
)

func Test_hookEventHandle_logsHaveNoSecret(t *testing.T) {
	const (
		sentinelSecret = "SENTINELsecret123"
		sentinelToken  = "SENTINELtoken456"
	)

	tests := []struct {
		name  string
		event *linebot.Event
	}{
		{
			name:  "unsupported event type",
			event: &linebot.Event{Type: linebot.EventType("unknown")},
		},
		{
			name:  "follow without a source",
			event: &linebot.Event{Type: linebot.EventTypeFollow},
		},
		{
			name:  "message without a message",
			event: &linebot.Event{Type: linebot.EventTypeMessage, Source: &linebot.EventSource{Type: linebot.EventSourceTypeUser, UserID: "u1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logtest.Capture(t)

			ac := testHookAccount()
			ac.Secret = sentinelSecret
			ac.Token = sentinelToken

			h := &lineHandler{}
			if _, err := h.hookEventHandle(context.Background(), ac, tt.event); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}

			logtest.AssertNoLeak(t, hook, sentinelSecret, sentinelToken)
		})
	}
}
