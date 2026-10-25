package conversationhandler

import (
	"context"
	"testing"

	"monorepo/bin-conversation-manager/models/conversation"
)

// Test_Event_errors covers the dispatch rejections that happen before any
// downstream handler is used, so the handler is built without dependencies.
func Test_Event_errors(t *testing.T) {
	tests := []struct {
		name string

		referenceType conversation.Type
		data          []byte
	}{
		{"unknown reference type", conversation.Type("unknown"), []byte(`{}`)},
		{"sms event with malformed json", conversation.TypeMessage, []byte(`{`)},
		{"sms event with an unknown direction", conversation.TypeMessage, []byte(`{"direction":"sideways"}`)},
		{"webchat event with malformed json", conversation.TypeWebchat, []byte(`{`)},
		{"webchat event with an unknown direction", conversation.TypeWebchat, []byte(`{"direction":"sideways"}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &conversationHandler{}

			if err := h.Event(context.Background(), tt.referenceType, tt.data); err == nil {
				t.Errorf("expected an error for %s, got nil", tt.name)
			}
		})
	}
}
