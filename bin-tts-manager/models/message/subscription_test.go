package message

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Message overrides the subscription address of the global topic exchange (VOIP-1404/1405). The
// assertion pins the POINTER type: the event data reaches notifyhandler as a POINTER and the
// assertion matches the dynamic type; a VALUE of this pointer-receiver type would fail the
// assertion (the exact pipecat defect this ticket fixed).
var _ eventtopic.SubscriptionIdentifier = (*Message)(nil)

func TestMessageEventSubscriptionID(t *testing.T) {
	streamingID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name    string
		message *Message
		expect  string
	}{
		{
			"normal",
			&Message{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				StreamingID:   streamingID,
				TotalMessage:  "hello world",
				PlayedMessage: "hello",
				TotalCount:    2,
				PlayedCount:   1,
			},
			streamingID.String(),
		},
		{
			"empty streaming id",
			&Message{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.message.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestMessageEventSubscriptionIDIsNotOwnID pins the reason the override exists: the message id is
// captured once at streamer init and goes stale for every later utterance of the same streaming
// session, so it must never become the subscription address. Two messages of one session carry
// different own ids and still resolve to the same address.
func TestMessageEventSubscriptionIDIsNotOwnID(t *testing.T) {
	streamingID := uuid.Must(uuid.NewV4())

	first := &Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		StreamingID: streamingID,
	}
	second := &Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: first.CustomerID,
		},
		StreamingID: streamingID,
	}

	if first.ID == second.ID {
		t.Fatalf("Message ids are expected to differ per utterance. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != streamingID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", streamingID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the message own id. id: %s", first.ID)
	}
}
