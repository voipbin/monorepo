package message

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Message overrides the subscription address of the global topic exchange (VOIP-1404/1405). The
// assertion pins the POINTER receiver: notifyhandler asserts on the dynamic type of the event
// data, which is always a pointer, so a value receiver would silently never be picked up.
var _ eventtopic.SubscriptionIdentifier = (*Message)(nil)

func TestMessageEventSubscriptionID(t *testing.T) {
	chatID := uuid.Must(uuid.NewV4())

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
				ChatID: chatID,
				Type:   TypeNormal,
				Text:   "hello",
			},
			chatID.String(),
		},
		{
			"empty chat id",
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

// TestMessageEventSubscriptionIDIsNotOwnID pins the Category B decision: the message id is stable,
// but it is NOT the subscription address. Two messages of one chat carry different own ids and
// must still resolve to the same address, so one binding follows the whole conversation.
func TestMessageEventSubscriptionIDIsNotOwnID(t *testing.T) {
	chatID := uuid.Must(uuid.NewV4())

	first := &Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ChatID: chatID,
	}
	second := &Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: first.CustomerID,
		},
		ChatID: chatID,
	}

	if first.ID == second.ID {
		t.Fatalf("Message ids are expected to differ per message. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != chatID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", chatID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the message own id. id: %s", first.ID)
	}
}
