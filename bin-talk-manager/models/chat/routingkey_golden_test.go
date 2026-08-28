// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type talk-manager publishes today, across all three resource namespaces
// (chat / chatmessage / chatparticipant), and asserts the exact key that notifyhandler generates
// for the real event data type of each publish site. The primary defect class it guards against is
// "the right key shape carrying the wrong id space": a child resource addressed by its own id
// produces well-formed keys that no instance binding ever matches, and no runtime metric can
// detect it. Design doc 1405 §2.3 / §4.
//
// The file lives in models/chat because the chat is the axis every event of this service converges
// on; it is an external test package so it can import the sibling model packages without any
// import-cycle risk.
//
// MAINTENANCE: the table pins the LIVE publish set, enumerated from the publish sites:
//
//	pkg/chathandler/chat.go:165,263,292          chat_created / chat_updated / chat_deleted
//	pkg/messagehandler/message.go:202,207        chatmessage_created / chatmessage_deleted
//	pkg/reactionhandler/reaction.go:139          chatmessage_reaction_updated
//	pkg/participanthandler/participant.go:94,213 chatparticipant_added / chatparticipant_removed
//
// When a new event type gains a publish site, add its row here in the same change.
//
// NOTE: `cmd/talk-control` constructs its NotifyHandler with an EMPTY queueEvent and a nil
// reqHandler, and the global-topic-publish option is deliberately NOT enabled there — that is a
// pre-existing defect tracked as a separate ticket (1405 §7), left untouched by this change.
package chat_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
)

// chatID is the single subscription address every talk-manager event of one conversation must
// carry, regardless of which resource namespace the event lives in.
var chatID = uuid.FromStringOrNil("3ac91f60-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): every published event data type satisfies eventtopic.SubscriptionIdentifier --
// chat.Chat through the own-id default promoted from the embedded commonidentity.Identity,
// message.Message and participant.Participant through their explicit parent-chat overrides
// -- and the method's return IS the subscription address -- there is no JSON fallback.
// Non-implementing data (impossible on the narrowed production signature, but representable
// through this `any`-typed helper) and typed-nil implementers resolve to "", which the key
// builder degrades to the `-` placeholder. Keeping the reproduction here rather than reaching
// into notifyhandler internals is deliberate -- the golden table must fail when a model's
// method starts returning a different address.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
	// interface still SATISFIES the assertion, and every real implementation dereferences its
	// receiver -- calling the method would panic. Production resolves such a payload to the
	// `-` placeholder.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameTalkManager)

	chatData := &chat.Chat{
		Identity: commonidentity.Identity{
			ID:         chatID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Type:        chat.TypeGroup,
		Name:        "release",
		MemberCount: 3,
	}

	messageData := &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // stable own id -- but not the address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ChatID: chatID,
		Type:   message.TypeNormal,
		Text:   "hello",
	}

	participantData := &participant.Participant{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // stable own id -- but not the address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ChatID: chatID,
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// chat resource -- own id is the address, returned through the EventSubscriptionID
		// promoted from the embedded commonidentity.Identity (VOIP-1419).
		{
			"chat_created",
			chat.EventTypeChatCreated,
			chatData,
			"talk-manager.chat.3ac91f60-0000-4000-8000-000000000001.created",
		},
		{
			"chat_updated",
			chat.EventTypeChatUpdated,
			chatData,
			"talk-manager.chat.3ac91f60-0000-4000-8000-000000000001.updated",
		},
		{
			"chat_deleted",
			chat.EventTypeChatDeleted,
			chatData,
			"talk-manager.chat.3ac91f60-0000-4000-8000-000000000001.deleted",
		},

		// chatmessage resource -- addressed by the parent chat-id via the override.
		{
			"chatmessage_created",
			message.EventTypeMessageCreated,
			messageData,
			"talk-manager.chatmessage.3ac91f60-0000-4000-8000-000000000001.created",
		},
		{
			"chatmessage_deleted",
			message.EventTypeMessageDeleted,
			messageData,
			"talk-manager.chatmessage.3ac91f60-0000-4000-8000-000000000001.deleted",
		},
		{
			"chatmessage_reaction_updated",
			message.EventTypeMessageReactionUpdated,
			messageData,
			"talk-manager.chatmessage.3ac91f60-0000-4000-8000-000000000001.reaction_updated",
		},

		// chatparticipant resource -- addressed by the parent chat-id via the override.
		{
			"chatparticipant_added",
			participant.EventParticipantAdded,
			participantData,
			"talk-manager.chatparticipant.3ac91f60-0000-4000-8000-000000000001.added",
		},
		{
			"chatparticipant_removed",
			participant.EventParticipantRemoved,
			participantData,
			"talk-manager.chatparticipant.3ac91f60-0000-4000-8000-000000000001.removed",
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

// TestGoldenRoutingKeysShareOneAddress pins the address-convergence property the table above
// exists to protect (1405 §4): the `chat`, `chatmessage` and `chatparticipant` resources of one
// conversation all resolve to the SAME subscription address, so a consumer following that
// conversation binds `talk-manager.chat.<chat-id>.#`,
// `talk-manager.chatmessage.<chat-id>.#` and `talk-manager.chatparticipant.<chat-id>.#` and
// receives everything.
func TestGoldenRoutingKeysShareOneAddress(t *testing.T) {
	expect := chatID.String()

	tests := []struct {
		name string
		data any
	}{
		{
			"chat",
			&chat.Chat{Identity: commonidentity.Identity{ID: chatID}},
		},
		{
			"message",
			&message.Message{
				Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				ChatID:   chatID,
			},
		},
		{
			"participant",
			&participant.Participant{
				Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				ChatID:   chatID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := resolveSubscriptionID(t, tt.data)
			if res != expect {
				t.Errorf("Wrong match. expect: %s, got: %s", expect, res)
			}
		})
	}
}
