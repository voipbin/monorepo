// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type tts-manager publishes today, across all three resource namespaces
// (speaking / streaming / message), and asserts the exact key that notifyhandler generates for the
// real event data type of each publish site. The primary defect class it guards against is "the
// right key shape carrying the wrong id space": a stale or per-event id under a resource namespace
// produces well-formed keys that no instance binding ever matches, and no runtime metric can
// detect it. Design doc 1405 §2.2 / §4.
//
// Since VOIP-1419 every published type satisfies the mandatory `EventSubscriptionID()`
// contract -- the own-id types via the default promoted from the embedded
// commonidentity.Identity, `*message.Message` via its explicit parent-streaming override --
// compiler-enforced; there is no JSON fallback anymore, and an empty return is the
// only degrade path (the `-` placeholder). `*streaming.Streaming` and `*speaking.Speaking` are
// addressed by their OWN id; `*message.Message` is addressed by its parent streaming-id.
//
// The file lives in models/streaming because the streaming session is the axis this service's
// events converge on; it is an external test package so it can import the sibling model packages
// without any import-cycle risk.
//
// MAINTENANCE: the table pins the LIVE publish set, enumerated from the publish sites:
//
//	pkg/speakinghandler/speaking.go:133,280            speaking_started / speaking_stopped
//	pkg/streaminghandler/streaming.go:79,116           streaming_created / streaming_deleted
//	pkg/streaminghandler/{gcp,aws,elevenlabs}.go       message_initiated / message_play_started /
//	                                                   message_play_finished
//
// The `streaming_finished`, `streaming_play_started` and `streaming_play_finished` constants in
// event.go are DEAD (no publish site references them) and are deliberately excluded -- do not add
// rows for them without a publish site. Their live twins are the `message_play_*` types above.
// When a new event type gains a publish site, add its row here in the same change.
package streaming_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-tts-manager/models/message"
	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"
)

// streamingID is the subscription address every streaming-session event must carry, regardless of
// which resource namespace the event lives in.
var streamingID = uuid.FromStringOrNil("6b2d41ae-0000-4000-8000-000000000001")

// speakingID is the address of the speaking resource. A speaking session is an independent
// persistent record addressed by its own id, returned through the promoted Identity default.
var speakingID = uuid.FromStringOrNil("6b2d41ae-0000-4000-8000-000000000002")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): the payload's `EventSubscriptionID()` method (promoted Identity default or
// explicit override) is the single source of
// the subscription address -- the contract is mandatory (compiler-enforced at the publish
// sites), and an empty return degrades to the `-` placeholder. Keeping the mirror here rather
// than reaching into notifyhandler internals is deliberate -- the golden table must fail if a
// method starts returning a different id space than the one pinned below.
//
// The parameter stays `any` so the table can also feed values that do not implement the
// interface; they resolve to "" (→ placeholder), matching what production's narrowed
// signature makes unrepresentable at compile time.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler.resolveSubscriptionID: a nil pointer whose type
	// implements the interface still SATISFIES the assertion, and every real implementation
	// dereferences its receiver -- calling the method would panic. Production resolves such a
	// payload to the `-` placeholder instead.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameTTSManager)

	speakingData := &speaking.Speaking{
		Identity: commonidentity.Identity{
			ID:         speakingID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ReferenceType: streaming.ReferenceTypeCall,
		ReferenceID:   uuid.Must(uuid.NewV4()),
		Language:      "en-US",
		Direction:     streaming.DirectionOutgoing,
		Status:        speaking.StatusActive,
	}

	streamingData := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         streamingID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ReferenceType: streaming.ReferenceTypeCall,
		ReferenceID:   uuid.Must(uuid.NewV4()),
		Language:      "en-US",
		Gender:        streaming.GenderNeutral,
		Direction:     streaming.DirectionOutgoing,
		MessageID:     uuid.Must(uuid.NewV4()),
	}

	messageData := &message.Message{
		Identity: commonidentity.Identity{
			// captured once at streamer init: stable, but goes stale from the second
			// utterance on -- never an address.
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		StreamingID:  streamingID,
		TotalMessage: "hello world",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// speaking resource -- own id is the address, returned through the promoted Identity default.
		{
			"speaking_started",
			speaking.EventTypeSpeakingStarted,
			speakingData,
			"tts-manager.speaking.6b2d41ae-0000-4000-8000-000000000002.started",
		},
		{
			"speaking_stopped",
			speaking.EventTypeSpeakingStopped,
			speakingData,
			"tts-manager.speaking.6b2d41ae-0000-4000-8000-000000000002.stopped",
		},

		// streaming resource -- own id is the address, returned through the promoted Identity default.
		{
			"streaming_created",
			streaming.EventTypeStreamingCreated,
			streamingData,
			"tts-manager.streaming.6b2d41ae-0000-4000-8000-000000000001.created",
		},
		{
			"streaming_deleted",
			streaming.EventTypeStreamingDeleted,
			streamingData,
			"tts-manager.streaming.6b2d41ae-0000-4000-8000-000000000001.deleted",
		},

		// message resource -- addressed by the parent streaming-id via the override, not by the
		// message's own id. All three types are LIVE on every vendor backend (gcp/aws/elevenlabs).
		{
			"message_initiated",
			message.EventTypeInitiated,
			messageData,
			"tts-manager.message.6b2d41ae-0000-4000-8000-000000000001.initiated",
		},
		{
			"message_play_started",
			message.EventTypePlayStarted,
			messageData,
			"tts-manager.message.6b2d41ae-0000-4000-8000-000000000001.play_started",
		},
		{
			"message_play_finished",
			message.EventTypePlayFinished,
			messageData,
			"tts-manager.message.6b2d41ae-0000-4000-8000-000000000001.play_finished",
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
// exists to protect (1405 §4): the `streaming` and `message` resources of one TTS session resolve
// to the SAME subscription address, so a consumer following that session binds
// `tts-manager.streaming.<streaming-id>.#` + `tts-manager.message.<streaming-id>.#` and receives
// everything.
func TestGoldenRoutingKeysShareOneAddress(t *testing.T) {
	expect := streamingID.String()

	tests := []struct {
		name string
		data any
	}{
		{
			"streaming",
			&streaming.Streaming{Identity: commonidentity.Identity{ID: streamingID}},
		},
		{
			"message",
			&message.Message{
				Identity:    commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				StreamingID: streamingID,
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
