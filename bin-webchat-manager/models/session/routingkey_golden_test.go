// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type webchat-manager publishes today and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site. The primary defect
// class it guards against is "the right key shape carrying the wrong id space": an address that
// is well-formed but belongs to another id space produces keys that no instance binding ever
// matches, and no runtime metric can detect it. Design doc 1404 §4.2 / 1405 §2.3, §4.
//
// webchat-manager is the sharpest case of resource collapse in the rollout: BOTH of its event
// types (`webchat_message_created`, `webchat_session_ended`) split on the first `_` into resource
// `webchat`, and `*message.Message`'s EventSubscriptionID points its address at the parent
// SessionID, so both land on one key space. TestGoldenRoutingKeysWebchatResourceCollapse below
// asserts exactly that -- one binding, the whole session.
//
// The file lives in models/session because the session is the address every event of this service
// resolves to; it is an external test package so it can import the sibling model packages without
// any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. Both publish sites
// (pkg/messagehandler/create.go:99, pkg/sessionhandler/db.go:83) are guarded by
// `if h.notifyHandler != nil`, so a nil-NotifyHandler construction silently publishes nothing --
// the guard is about handler wiring, not about the keys, and does not change any row here. If a
// new event type is added, add its row AND extend the collapse test.
package session_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
)

// sessionID is the single subscription address every webchat-manager event of one visitor
// conversation must carry. Session.ID doubles as the visitor's continuity token, so it is both
// the natural address and the only id a subscriber can hold before the messages start flowing.
var sessionID = uuid.FromStringOrNil("3c7f21a4-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1404 §4.2, VOIP-1419): the subscription address comes ONLY from the type's
// EventSubscriptionID method -- session.Session's is the own-id default promoted from the
// embedded commonidentity.Identity, message.Message's is an explicit parent-session
// override -- the contract is mandatory (the narrowed publish signature
// enforces it at compile time), and an empty address degrades to the `-` placeholder.
// Keeping the reproduction here rather than reaching into notifyhandler internals is
// deliberate -- the golden table must fail when a model's method stops returning the pinned
// address.
//
// The parameter stays `any` so a non-implementing payload still resolves (to "", hence the
// placeholder) instead of failing to compile. The typed-nil guard mirrors production: a nil
// pointer whose type implements the interface still SATISFIES the assertion, but every real
// implementation dereferences its receiver -- calling the method would panic -- so such a
// payload resolves to the placeholder without the call.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}
	if v := reflect.ValueOf(data); v.Kind() == reflect.Pointer && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameWebchatManager)

	sessionData := &session.Session{
		Identity: commonidentity.Identity{
			ID:         sessionID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		WidgetID: uuid.Must(uuid.NewV4()),
		Status:   session.StatusEnded,
	}

	messageData := &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // stable, but not an address: it debuts in its own created event
			CustomerID: sessionData.CustomerID,
		},
		WidgetID:  sessionData.WidgetID, // denormalized convenience field, never the address
		SessionID: sessionID,
		Direction: message.DirectionInbound,
		Status:    message.StatusSent,
		Text:      "hello",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// message event -- resource `webchat`, action `message_created`, address supplied by
		// *message.Message's EventSubscriptionID (the parent SessionID, not the message's own id).
		{
			"webchat_message_created",
			message.EventTypeMessageCreated,
			messageData,
			"webchat-manager.webchat.3c7f21a4-0000-4000-8000-000000000001.message_created",
		},

		// session end -- resource `webchat`, action `session_ended`. The Session's own id IS the
		// address here, returned through the promoted Identity default, and it is the SAME id the
		// message's method points at.
		{
			"webchat_session_ended",
			session.EventTypeSessionEnded,
			sessionData,
			"webchat-manager.webchat.3c7f21a4-0000-4000-8000-000000000001.session_ended",
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

// TestGoldenRoutingKeysWebchatResourceCollapse is the mandatory resource-collapse assertion of
// 1405 §4. It pins BOTH halves of the property at once:
//
//	(1) every event type collapses to the resource segment `webchat` (they all start with the
//	    `webchat_` prefix, and RoutingKey splits on the FIRST underscore), and
//	(2) every event carries the same session address (Session's method returns its own id,
//	    Message's method returns its parent SessionID),
//
// which together mean ONE binding pattern -- `webchat-manager.webchat.<session-id>.#` -- delivers
// the entire session. Repointing the Message method at its own id, or renaming an event type so it
// no longer splits to `webchat`, breaks this test even if the per-key table above were updated to
// match.
func TestGoldenRoutingKeysWebchatResourceCollapse(t *testing.T) {
	publisher := string(commonoutline.ServiceNameWebchatManager)
	pattern := eventtopic.PatternInstance(publisher, "webchat", sessionID.String())

	if pattern != "webchat-manager.webchat.3c7f21a4-0000-4000-8000-000000000001.#" {
		t.Fatalf("Wrong match. expect: %s, got: %s", "webchat-manager.webchat.3c7f21a4-0000-4000-8000-000000000001.#", pattern)
	}

	// the literal prefix the pattern matches: everything up to and including the separator that
	// precedes the `#` wildcard.
	prefix := strings.TrimSuffix(pattern, "#")

	tests := []struct {
		name      string
		eventType string
		data      any
	}{
		{
			"webchat_message_created",
			message.EventTypeMessageCreated,
			&message.Message{
				Identity:  commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				WidgetID:  uuid.Must(uuid.NewV4()),
				SessionID: sessionID,
			},
		},
		{
			"webchat_session_ended",
			session.EventTypeSessionEnded,
			&session.Session{
				Identity: commonidentity.Identity{ID: sessionID},
				WidgetID: uuid.Must(uuid.NewV4()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := eventtopic.RoutingKey(publisher, tt.eventType, resolveSubscriptionID(t, tt.data))

			if !strings.HasPrefix(key, prefix) {
				t.Errorf("Wrong match. expect prefix: %s, got: %s", prefix, key)
			}

			// the resource segment specifically -- asserted on its own so a future event type
			// that kept the session address but broke the `webchat_` prefix is reported as the
			// resource failure it is.
			segments := strings.Split(key, ".")
			if len(segments) != 4 {
				t.Fatalf("Wrong segment count. expect: 4, got: %d (key: %s)", len(segments), key)
			}
			if segments[1] != "webchat" {
				t.Errorf("Wrong match. expect: webchat, got: %s", segments[1])
			}
			if segments[2] != sessionID.String() {
				t.Errorf("Wrong match. expect: %s, got: %s", sessionID.String(), segments[2])
			}
		})
	}
}
