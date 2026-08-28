// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type pipecat-manager publishes today, across all three resource
// namespaces (pipecatcall / message / team), and asserts the exact key that notifyhandler
// generates for the real event data type of each publish site. The primary defect class it
// guards against is "the right key shape carrying the wrong id space": a per-event random id
// under a resource namespace produces well-formed keys that no instance binding ever matches,
// and no runtime metric can detect it. Design doc §2.2 / §4.
//
// pipecat-manager is the one service in the rollout that published VALUE event data before
// VOIP-1405 (`message.Message` / `message.MemberSwitchedEvent` at
// pkg/pipecatcallhandler/runner.go). A value never satisfies the pointer-receiver
// eventtopic.SubscriptionIdentifier assertion, so the six publish sites were converted to
// pointers in the same change; the `data` column below is a pointer for exactly that reason and
// must stay one.
//
// The file lives in models/pipecatcall because the table spans every model package of the service
// and the pipecatcall is the resource all of them address; it is an external test package so it
// can import the sibling model packages without any import-cycle risk.
package pipecatcall_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
)

// pipecatcallID is the single subscription address every pipecat-manager event of one AI voice
// session must carry, regardless of which resource namespace the event lives in.
var pipecatcallID = uuid.FromStringOrNil("7b21d4a6-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): the payload's explicit EventSubscriptionID method is the ONLY source of the
// subscription-id segment -- the JSON top-level-"id" fallback no longer exists. Implementation
// is mandatory for every published type (the narrowed PublishEvent signature enforces it at
// compile time); an empty return degrades to the `-` placeholder. Keeping the reproduction here
// rather than reaching into notifyhandler internals is deliberate -- the golden table must fail
// if a model's method ever returns the wrong id space.
//
// The parameter stays `any` (not the interface) so the supplementary tests below can feed
// non-implementing shapes -- e.g. a VALUE copy of a pointer-receiver type -- and assert the ""
// (→ placeholder) they degrade to.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the interface
	// still SATISFIES the assertion, and every real implementation dereferences its receiver --
	// calling the method would panic. Production resolves such a payload to the `-` placeholder.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNamePipecatManager)

	pipecatcallData := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         pipecatcallID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   uuid.Must(uuid.NewV4()),
	}

	// Transcription / user-llm events: newMessageEvent mints a fresh uuid per event, so the own
	// id is deliberately a random one here.
	messageData := &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // regenerated per event: never an address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		PipecatcallID:            pipecatcallID,
		PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
		Text:                     "hello",
	}

	// Bot-llm intermediate/final events reuse ONE per-generation message id across the whole
	// generation. Stable within a generation, but still not the session address -- a subscriber
	// cannot know it before the generation starts.
	llmMessageData := &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // per-generation id: still not an address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		PipecatcallID:            pipecatcallID,
		PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
		Text:                     "partial reply",
		Sequence:                 1,
	}

	// MemberSwitchedEvent carries no id of its own -- its override returns the parent
	// pipecatcall-id, which is the only thing keeping this row off the `-` placeholder.
	memberSwitchedData := &message.MemberSwitchedEvent{
		CustomerID:               uuid.Must(uuid.NewV4()),
		PipecatcallID:            pipecatcallID,
		PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
		TransitionFunctionName:   "transfer_to_sales",
		FromMember:               message.MemberInfo{ID: uuid.Must(uuid.NewV4()), Name: "Reception"},
		ToMember:                 message.MemberInfo{ID: uuid.Must(uuid.NewV4()), Name: "Sales"},
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// pipecatcall resource -- own id IS the address, returned by the type's explicit
		// EventSubscriptionID method (VOIP-1419).
		{
			"pipecatcall_created",
			pipecatcall.EventTypeCreated,
			pipecatcallData,
			"pipecat-manager.pipecatcall.7b21d4a6-0000-4000-8000-000000000001.created",
		},
		{
			"pipecatcall_deleted",
			pipecatcall.EventTypeDeleted,
			pipecatcallData,
			"pipecat-manager.pipecatcall.7b21d4a6-0000-4000-8000-000000000001.deleted",
		},
		{
			"pipecatcall_initialized",
			pipecatcall.EventTypeInitialized,
			pipecatcallData,
			"pipecat-manager.pipecatcall.7b21d4a6-0000-4000-8000-000000000001.initialized",
		},
		{
			"pipecatcall_terminated",
			pipecatcall.EventTypePipecatcallTerminated,
			pipecatcallData,
			"pipecat-manager.pipecatcall.7b21d4a6-0000-4000-8000-000000000001.terminated",
		},

		// message resource -- addressed by the parent pipecatcall-id, never the message own id.
		{
			"message_bot_transcription",
			message.EventTypeBotTranscription,
			messageData,
			"pipecat-manager.message.7b21d4a6-0000-4000-8000-000000000001.bot_transcription",
		},
		{
			"message_user_transcription",
			message.EventTypeUserTranscription,
			messageData,
			"pipecat-manager.message.7b21d4a6-0000-4000-8000-000000000001.user_transcription",
		},
		{
			"message_user_llm",
			message.EventTypeUserLLM,
			messageData,
			"pipecat-manager.message.7b21d4a6-0000-4000-8000-000000000001.user_llm",
		},
		{
			"message_bot_llm_intermediate",
			message.EventTypeBotLLMIntermediate,
			llmMessageData,
			"pipecat-manager.message.7b21d4a6-0000-4000-8000-000000000001.bot_llm_intermediate",
		},
		{
			"message_bot_llm",
			message.EventTypeBotLLM,
			llmMessageData,
			"pipecat-manager.message.7b21d4a6-0000-4000-8000-000000000001.bot_llm",
		},

		// team resource -- the event type is `team_member_switched`, so the mechanical split on
		// the first `_` puts it under a THIRD namespace (`team`), addressed by the very same
		// pipecatcall-id. The payload has no id of its own, so its override is what keeps this
		// row off the `-` placeholder.
		{
			"team_member_switched",
			message.EventTypeTeamMemberSwitched,
			memberSwitchedData,
			"pipecat-manager.team.7b21d4a6-0000-4000-8000-000000000001.member_switched",
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

// TestGoldenRoutingKeysShareOneAddress pins the property the table above exists to protect: every
// event of one AI voice session resolves to the same subscription address across all three
// resource namespaces, so a consumer following that session binds
// `pipecat-manager.pipecatcall.<id>.#`, `pipecat-manager.message.<id>.#` and
// `pipecat-manager.team.<id>.#` and receives everything (design §4 address-convergence note).
func TestGoldenRoutingKeysShareOneAddress(t *testing.T) {
	expect := pipecatcallID.String()

	tests := []struct {
		name string
		data any
	}{
		{"pipecatcall", &pipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: pipecatcallID}}},
		{"message", &message.Message{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, PipecatcallID: pipecatcallID}},
		{"member_switched", &message.MemberSwitchedEvent{PipecatcallID: pipecatcallID}},
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

// TestValueEventDataLosesTheOverride pins the pointer-receiver semantics the six pointer
// conversions in pkg/pipecatcallhandler/runner.go rely on. The method is declared on the pointer
// receiver (as eventtopic.SubscriptionIdentifier requires), so a VALUE of either type does not
// satisfy the interface and resolves to "" -- the `-` placeholder -- instead of the session
// address. (Before VOIP-1419 a Message VALUE silently degraded to the JSON top-level id, a
// well-formed key no instance binding matches; that fallback is gone, and once PublishEvent's
// parameter is narrowed to the interface a value publish no longer even compiles.)
func TestValueEventDataLosesTheOverride(t *testing.T) {
	publisher := string(commonoutline.ServiceNamePipecatManager)

	ownID := uuid.FromStringOrNil("7b21d4a6-0000-4000-8000-0000000000ff")
	msg := message.Message{
		Identity:      commonidentity.Identity{ID: ownID},
		PipecatcallID: pipecatcallID,
	}
	evt := message.MemberSwitchedEvent{PipecatcallID: pipecatcallID}

	if _, ok := any(msg).(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("A Message VALUE must not satisfy SubscriptionIdentifier. the receiver is a pointer.")
	}
	if _, ok := any(evt).(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("A MemberSwitchedEvent VALUE must not satisfy SubscriptionIdentifier. the receiver is a pointer.")
	}

	// Value publish of Message: no interface satisfaction, no address -- never the own id.
	if res := resolveSubscriptionID(t, msg); res != "" {
		t.Errorf("Wrong match. expect: %s, got: %s", "", res)
	}

	// Value publish of MemberSwitchedEvent: same degrade -- placeholder.
	if res := eventtopic.RoutingKey(publisher, message.EventTypeTeamMemberSwitched, resolveSubscriptionID(t, evt)); res != "pipecat-manager.team.-.member_switched" {
		t.Errorf("Wrong match. expect: %s, got: %s", "pipecat-manager.team.-.member_switched", res)
	}

	// The pointer form -- what the publish path actually hands notifyhandler today -- resolves to
	// the session address for both.
	if res := resolveSubscriptionID(t, &msg); res != pipecatcallID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", pipecatcallID.String(), res)
	}
	if res := resolveSubscriptionID(t, &evt); res != pipecatcallID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", pipecatcallID.String(), res)
	}
}
