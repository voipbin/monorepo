// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type message-manager publishes today (`message_created`,
// `message_updated`, `message_deleted` -- all three from pkg/messagehandler/db.go:46,69,108) and
// asserts the exact key that notifyhandler generates for the real event data type of each publish
// site. The defect class it guards against is "the right key shape carrying the wrong id space":
// a key whose third segment is not the address subscribers can bind to in advance produces
// well-formed keys that no instance binding ever matches, and no runtime metric can detect it.
// Design doc §2.4 / §4.
//
// message-manager is a DEFAULT-ID service, and the reason is worth stating because the type name
// is shared with several Category-B services: this `message.Message` is the SMS RESOURCE itself,
// not a stream fragment of some parent session. Unlike ai/conversation/talk/webchat/tts -- where a
// `Message` is one utterance inside a longer-lived parent and adopts the parent's id as its
// address -- an SMS message is an independent persistent resource created and retrieved by its own
// id, so its own id IS the subscription address. It therefore carries NO
// eventtopic.SubscriptionIdentifier override; TestMessageUsesDefaultSubscriptionID pins that
// deliberate absence.
//
// The file lives in models/message because that is the service's PRIMARY model package and message
// is the resource every published event addresses; it is an external test package
// (`message_test`) so it can import sibling packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. A new published event type, a changed event-type
// constant, or an override added to *Message must be reflected here in the same change -- the
// table is not a specification of what the events ought to be, it is a lock on what they are.
package message_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-message-manager/models/message"
)

// messageID is the single subscription address every message-manager event of one SMS must carry.
var messageID = uuid.FromStringOrNil("7d20b6e4-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Reproducing it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model starts or
// stops implementing the interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		// typed-nil guard, mirroring notifyhandler.resolveSubscriptionOverride: a nil pointer whose
		// type implements the interface still SATISFIES the assertion, and every real implementation
		// dereferences its receiver -- calling the method would panic. Production reports "no
		// override" for such a payload, so this guard falls through to the JSON half below rather
		// than returning early; `null` carries no top-level `id` either, so both halves agree on the
		// `-` placeholder.
		if v := reflect.ValueOf(data); v.Kind() != reflect.Ptr || !v.IsNil() {
			return identifier.EventSubscriptionID()
		}
	}

	m, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Could not marshal the event data. err: %v", err)
	}

	d := struct {
		ID string `json:"id"`
	}{}
	if errUnmarshal := json.Unmarshal(m, &d); errUnmarshal != nil {
		return ""
	}

	return d.ID
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameMessageManager)

	// The real event data type of every publish site: pkg/messagehandler/db.go passes
	// *message.Message to PublishWebhookEvent for all three event types.
	messageData := &message.Message{
		Identity: commonidentity.Identity{
			ID:         messageID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Type: message.TypeSMS,
		Source: &commonaddress.Address{
			Type:   commonaddress.TypeTel,
			Target: "+821100000001",
		},
		Direction: message.DirectionOutbound,
		Text:      "hello",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// message resource -- own id is the address, resolved by the default JSON fallback.
		{
			"message_created",
			message.EventTypeMessageCreated,
			messageData,
			"message-manager.message.7d20b6e4-0000-4000-8000-000000000001.created",
		},
		{
			"message_updated",
			message.EventTypeMessageUpdated,
			messageData,
			"message-manager.message.7d20b6e4-0000-4000-8000-000000000001.updated",
		},
		{
			"message_deleted",
			message.EventTypeMessageDeleted,
			messageData,
			"message-manager.message.7d20b6e4-0000-4000-8000-000000000001.deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscriptionID := resolveSubscriptionID(t, tt.data)

			res := eventtopic.RoutingKey(publisher, tt.eventType, subscriptionID)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestMessageUsesDefaultSubscriptionID pins the deliberate ABSENCE of an override on Message
// (design §2.4). The name collides with the Category-B `Message` types of ai/conversation/talk/
// webchat/tts, which DO override with a parent id -- this service's Message is the SMS resource
// itself, so its own id IS the subscription address and the default JSON `id` extraction must keep
// covering it. If someone "aligns" this type with the other Message overrides, this test fails and
// forces the golden table above to be re-derived.
func TestMessageUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &message.Message{Identity: commonidentity.Identity{ID: messageID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Message must not implement SubscriptionIdentifier. this service's Message is the SMS resource itself and its own id is the subscription address.")
	}

	if res := resolveSubscriptionID(t, data); res != messageID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", messageID.String(), res)
	}
}
