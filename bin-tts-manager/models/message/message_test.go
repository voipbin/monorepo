package message

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestMessage(t *testing.T) {
	tests := []struct {
		name string

		streamingID   uuid.UUID
		totalMessage  string
		playedMessage string
		totalCount    int
		playedCount   int
		finish        bool
	}{
		{
			name: "creates_message_with_all_fields",

			streamingID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
			totalMessage:  "Hello, this is the complete message",
			playedMessage: "Hello, this is",
			totalCount:    3,
			playedCount:   1,
			finish:        false,
		},
		{
			name: "creates_message_with_empty_fields",

			streamingID:   uuid.Nil,
			totalMessage:  "",
			playedMessage: "",
			totalCount:    0,
			playedCount:   0,
			finish:        false,
		},
		{
			name: "creates_finished_message",

			streamingID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			totalMessage:  "Goodbye!",
			playedMessage: "Goodbye!",
			totalCount:    1,
			playedCount:   1,
			finish:        true,
		},
		{
			name: "creates_message_with_multiple_plays",

			streamingID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
			totalMessage:  "Repeating message",
			playedMessage: "Repeating message",
			totalCount:    5,
			playedCount:   3,
			finish:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{
				StreamingID:   tt.streamingID,
				TotalMessage:  tt.totalMessage,
				PlayedMessage: tt.playedMessage,
				TotalCount:    tt.totalCount,
				PlayedCount:   tt.playedCount,
				Finish:        tt.finish,
			}

			if m.StreamingID != tt.streamingID {
				t.Errorf("Wrong StreamingID. expect: %s, got: %s", tt.streamingID, m.StreamingID)
			}
			if m.TotalMessage != tt.totalMessage {
				t.Errorf("Wrong TotalMessage. expect: %s, got: %s", tt.totalMessage, m.TotalMessage)
			}
			if m.PlayedMessage != tt.playedMessage {
				t.Errorf("Wrong PlayedMessage. expect: %s, got: %s", tt.playedMessage, m.PlayedMessage)
			}
			if m.TotalCount != tt.totalCount {
				t.Errorf("Wrong TotalCount. expect: %d, got: %d", tt.totalCount, m.TotalCount)
			}
			if m.PlayedCount != tt.playedCount {
				t.Errorf("Wrong PlayedCount. expect: %d, got: %d", tt.playedCount, m.PlayedCount)
			}
			if m.Finish != tt.finish {
				t.Errorf("Wrong Finish. expect: %t, got: %t", tt.finish, m.Finish)
			}
		})
	}
}
