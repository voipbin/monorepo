package message

import (
	"testing"

	"monorepo/bin-common-handler/models/eventtopic"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

// Message overrides the subscription address of the global topic exchange (VOIP-1404/1405). The
// assertion pins the POINTER type: the event data reaches notifyhandler as a POINTER and the
// assertion matches the dynamic type; a VALUE of this pointer-receiver type would fail the
// assertion (the exact pipecat defect this ticket fixed).
var _ eventtopic.SubscriptionIdentifier = (*Message)(nil)

func TestMessage_Struct(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{
			name: "full message",
			msg: Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
					CustomerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
				},
				PipecatcallID:            uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
				PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
				Text:                     "Hello, world!",
			},
		},
		{
			name: "empty message",
			msg:  Message{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure the Message struct is constructable with the given fields.
			// The struct has no behavior to validate beyond field presence, so
			// just round-trip the value to confirm the test compiles and runs.
			_ = tt.msg
		})
	}
}

func TestMessage_EventSubscriptionID(t *testing.T) {
	pipecatcallID := uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d")

	tests := []struct {
		name   string
		msg    *Message
		expect string
	}{
		{
			name: "normal",
			msg: &Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
					CustomerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
				},
				PipecatcallID:            pipecatcallID,
				PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
				Text:                     "Hello, world!",
			},
			expect: pipecatcallID.String(),
		},
		{
			name:   "empty pipecatcall id",
			msg:    &Message{},
			expect: uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.msg.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestMessage_EventSubscriptionIDIsNotOwnID pins the reason the override exists at all: Message
// DOES carry a top-level `id`, so the default JSON fallback would produce a well-formed key that
// no instance binding ever matches. The id-generation rule differs per publish site — the
// transcription/user-llm events mint a fresh uuid per event, the bot-llm events reuse the
// per-generation message id — and neither is a bindable address. Every message of one session
// must resolve to the same pipecatcall-id, and never to its own id.
func TestMessage_EventSubscriptionIDIsNotOwnID(t *testing.T) {
	pipecatcallID := uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d")

	// Two events of the SAME session with different own ids — mirroring the per-event uuid mint
	// in pipecatcallhandler.newMessageEvent.
	first := &Message{
		Identity:      identity.Identity{ID: uuid.FromStringOrNil("c15f98f8-af1f-11f0-b009-535ac8cbc876")},
		PipecatcallID: pipecatcallID,
	}
	second := &Message{
		Identity:      identity.Identity{ID: uuid.FromStringOrNil("54eb0456-af23-11f0-986c-4bb2d9cd75de")},
		PipecatcallID: pipecatcallID,
	}

	if first.ID == second.ID {
		t.Fatalf("Message ids are expected to differ per event. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != pipecatcallID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", pipecatcallID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the message own id. id: %s", first.ID)
	}
}
