// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type conversation-manager publishes today, across both resource
// namespaces (conversation / account), and asserts the exact key that notifyhandler generates for
// the real event data type of each publish site. The primary defect class it guards against is
// "the right key shape carrying the wrong id space": an address that is well-formed but belongs
// to another id space produces keys that no instance binding ever matches, and no runtime metric
// can detect it. Design doc 1404 §4.2 / 1405 §2.3, §4.
//
// The file lives in models/conversation because conversation is the resource the service is named
// for and the axis every message event addresses; it is an external test package so it can import
// the sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. It deliberately omits
// `conversation.EventTypeConversationDeleted` -- the constant exists but no publish site
// references it (1405 §4 dead-constant list; a stream-completeness follow-up is registered). If a
// `conversation_deleted` publish is ever added, add its row here in the same change. Conversely,
// `account_deleted` IS live here (contrast with billing-manager, whose `account_deleted` is the
// dead one) -- do not remove it.
package conversation_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
)

// conversationID is the single subscription address every conversation-scoped event must carry,
// whether it announces the conversation itself or one of its messages.
var conversationID = uuid.FromStringOrNil("6b0d9f70-0000-4000-8000-000000000001")

// accountID addresses the account namespace, which is a genuinely independent resource: an
// account is a long-lived channel credential holder, not a child of any conversation.
var accountID = uuid.FromStringOrNil("6b0d9f70-0000-4000-8000-000000000002")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): every published event data type satisfies the
// eventtopic.SubscriptionIdentifier contract (own-id types via the default promoted from the
// embedded commonidentity.Identity, message.Message via its explicit parent-conversation
// override) -- mandatory, compiler-enforced, with no JSON
// fallback behind it -- and an empty return degrades to the `-` placeholder. Keeping the mirror
// here rather than reaching into notifyhandler internals is deliberate -- the golden table must
// fail when a model's method stops returning the pinned address.
//
// The parameter stays `any` (not the interface) so the table can also feed non-implementing
// payloads, which resolve to "" exactly as production's placeholder path does.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
	// interface still SATISFIES the assertion, and every real implementation dereferences its
	// receiver -- calling the method would panic. Production resolves such a payload to the
	// `-` placeholder, so the mirror returns "".
	if v := reflect.ValueOf(data); v.Kind() == reflect.Pointer && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameConversationManager)

	conversationData := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         conversationID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		AccountID: accountID,
		Type:      conversation.TypeLine,
	}

	accountData := &account.Account{
		Identity: commonidentity.Identity{
			ID:         accountID,
			CustomerID: conversationData.CustomerID,
		},
		Type: account.TypeLine,
		Name: "test account",
	}

	messageData := &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // stable, but not an address: it debuts in its own created event
			CustomerID: conversationData.CustomerID,
		},
		ConversationID: conversationID,
		Direction:      message.DirectionIncoming,
		Status:         message.StatusDone,
		Text:           "hello",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// conversation resource -- own id is the address, returned through the EventSubscriptionID
		// promoted from the embedded commonidentity.Identity (VOIP-1419).
		// Two publish sites emit `conversation_created` (conversationhandler/db.go:183 and
		// create_and_execute_flow.go:85); both carry *conversation.Conversation, so one row pins
		// both.
		{
			"conversation_created",
			conversation.EventTypeConversationCreated,
			conversationData,
			"conversation-manager.conversation.6b0d9f70-0000-4000-8000-000000000001.created",
		},
		{
			"conversation_updated",
			conversation.EventTypeConversationUpdated,
			conversationData,
			"conversation-manager.conversation.6b0d9f70-0000-4000-8000-000000000001.updated",
		},

		// account resource -- an independent resource addressed by its own id.
		{
			"account_created",
			account.EventTypeAccountCreated,
			accountData,
			"conversation-manager.account.6b0d9f70-0000-4000-8000-000000000002.created",
		},
		{
			"account_updated",
			account.EventTypeAccountUpdated,
			accountData,
			"conversation-manager.account.6b0d9f70-0000-4000-8000-000000000002.updated",
		},
		{
			"account_deleted",
			account.EventTypeAccountDeleted,
			accountData,
			"conversation-manager.account.6b0d9f70-0000-4000-8000-000000000002.deleted",
		},

		// message events -- the event type `conversation_message_*` splits into resource
		// `conversation` + action `message_*`, and the Message override supplies the parent
		// ConversationID. Both halves must hold: the resource collapses onto `conversation` AND
		// the address is the conversation-id, which is what makes the message stream land in the
		// exact same key space as the lifecycle rows above.
		{
			"conversation_message_created",
			message.EventTypeMessageCreated,
			messageData,
			"conversation-manager.conversation.6b0d9f70-0000-4000-8000-000000000001.message_created",
		},
		{
			"conversation_message_updated",
			message.EventTypeMessageUpdated,
			messageData,
			"conversation-manager.conversation.6b0d9f70-0000-4000-8000-000000000001.message_updated",
		},
		{
			"conversation_message_deleted",
			message.EventTypeMessageDeleted,
			messageData,
			"conversation-manager.conversation.6b0d9f70-0000-4000-8000-000000000001.message_deleted",
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

// TestGoldenRoutingKeysConversationStreamConverges pins the property the message override exists
// to produce: the conversation lifecycle events and every message of that conversation share BOTH
// the resource segment (`conversation`, because `conversation_message_*` splits on the first `_`)
// AND the subscription address, so a single binding
// `conversation-manager.conversation.<conversation-id>.#` delivers the entire conversation
// stream. A regression on either half -- a renamed event type that no longer splits to
// `conversation`, or a dropped override -- breaks this test.
func TestGoldenRoutingKeysConversationStreamConverges(t *testing.T) {
	publisher := string(commonoutline.ServiceNameConversationManager)
	expect := eventtopic.PatternInstance(publisher, "conversation", conversationID.String())

	conversationData := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: conversationID},
	}
	messageData := &message.Message{
		Identity:       commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		ConversationID: conversationID,
	}

	tests := []struct {
		name      string
		eventType string
		data      any
	}{
		{"conversation_created", conversation.EventTypeConversationCreated, conversationData},
		{"conversation_updated", conversation.EventTypeConversationUpdated, conversationData},
		{"conversation_message_created", message.EventTypeMessageCreated, messageData},
		{"conversation_message_updated", message.EventTypeMessageUpdated, messageData},
		{"conversation_message_deleted", message.EventTypeMessageDeleted, messageData},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := eventtopic.RoutingKey(publisher, tt.eventType, resolveSubscriptionID(t, tt.data))

			// `<publisher>.conversation.<conversation-id>.` is the literal prefix the pattern
			// above matches; comparing prefixes keeps the assertion independent of the action
			// segment while still proving both the resource and the address segments.
			prefix := expect[:len(expect)-1] // drop the trailing `#`
			if len(key) <= len(prefix) || key[:len(prefix)] != prefix {
				t.Errorf("Wrong match. expect prefix: %s, got: %s", prefix, key)
			}
		})
	}
}
