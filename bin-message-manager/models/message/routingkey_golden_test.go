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
// message-manager is an OWN-ID service, and the reason is worth stating because the type name
// is shared with several Category-B services: this `message.Message` is the SMS RESOURCE itself,
// not a stream fragment of some parent session. Unlike ai/conversation/talk/webchat/tts -- where a
// `Message` is one utterance inside a longer-lived parent and adopts the parent's id as its
// address -- an SMS message is an independent persistent resource created and retrieved by its own
// id, so its own id IS the subscription address. Its explicit EventSubscriptionID method
// (mandatory for every published type since VOIP-1419; an empty return degrades to the `-`
// placeholder) returns exactly that own id.
//
// The file lives in models/message because that is the service's PRIMARY model package and message
// is the resource every published event addresses; it is an external test package
// (`message_test`) so it can import sibling packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. A new published event type, a changed event-type
// constant, or a changed EventSubscriptionID implementation on *Message must be reflected here in
// the same change -- the table is not a specification of what the events ought to be, it is a
// lock on what they are.
package message_test

import (
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
// (VOIP-1419): every published event data type implements eventtopic.SubscriptionIdentifier
// explicitly -- there is no JSON fallback -- and the method's return is the subscription-id
// segment as-is, with an empty return degrading to the `-` placeholder. Non-implementing data
// returns "" (→ placeholder), which is why the parameter stays `any` rather than the interface.
// Reproducing the resolution here rather than reaching into notifyhandler internals is
// deliberate -- the golden table must fail when a model's method starts returning a different
// address.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
	// interface still SATISFIES the assertion, and every real implementation dereferences its
	// receiver -- calling the method would panic. Production degrades such a payload to the
	// placeholder, so the helper returns "" instead of calling the method.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
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
		// message resource -- own id is the address, returned by its explicit EventSubscriptionID.
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
