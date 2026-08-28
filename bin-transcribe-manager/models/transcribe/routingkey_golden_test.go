// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404).
//
// It covers EVERY event type transcribe-manager publishes today, across all three resource
// namespaces (transcribe / streaming / transcript), and asserts the exact key that notifyhandler
// generates for the real event data type of each publish site. The primary defect class it
// guards against is "the right key shape carrying the wrong id space": a per-event random id
// under a resource namespace produces well-formed keys that no instance binding ever matches,
// and no runtime metric can detect it. Design doc §4.2 / §8.
//
// The file lives in models/transcribe because the table spans every model package of the service
// and transcribe is the resource all of them address; it is an external test package so it can
// import the sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior, including the known bug at
// pkg/transcripthandler/db.go:33, where the transcript DELETE path publishes
// `transcript_created`. When that bug is fixed, the `transcript_created (delete path)` entry must
// be updated to `transcript_deleted` in the same change -- the table is not a specification of
// what the events ought to be, it is a lock on what they are.
package transcribe_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// transcribeID is the single subscription address every transcribe-manager event of one session
// must carry, regardless of which resource namespace the event lives in.
var transcribeID = uuid.FromStringOrNil("9f01c3d2-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists -- the
// top-level "id" of the marshaled payload. Keeping it here rather than reaching into notifyhandler
// internals is deliberate -- the golden table must fail when a model stops implementing the
// interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly; if the two ever
// diverge, this table stops reproducing what the publish path actually generates.
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
	publisher := string(commonoutline.ServiceNameTranscribeManager)

	transcribeData := &transcribe.Transcribe{
		Identity: commonidentity.Identity{
			ID:         transcribeID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Status: transcribe.StatusProgressing,
	}

	streamingData := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // streaming-id: stable, but the wrong address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		TranscribeID: transcribeID,
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	speechData := &streaming.Speech{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // regenerated per event: never an address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		StreamingID:  streamingData.ID,
		TranscribeID: transcribeID,
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	transcriptData := &transcript.Transcript{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // one per final result: never an address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		TranscribeID: transcribeID,
		Direction:    transcript.DirectionIn,
		Message:      "hello",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// transcribe resource -- own id is the address, resolved by the default JSON fallback.
		{
			"transcribe_created",
			transcribe.EventTypeTranscribeCreated,
			transcribeData,
			"transcribe-manager.transcribe.9f01c3d2-0000-4000-8000-000000000001.created",
		},
		{
			"transcribe_progressing",
			transcribe.EventTypeTranscribeProgressing,
			transcribeData,
			"transcribe-manager.transcribe.9f01c3d2-0000-4000-8000-000000000001.progressing",
		},
		{
			"transcribe_done",
			transcribe.EventTypeTranscribeDone,
			transcribeData,
			"transcribe-manager.transcribe.9f01c3d2-0000-4000-8000-000000000001.done",
		},
		{
			"transcribe_deleted",
			transcribe.EventTypeTranscribeDeleted,
			transcribeData,
			"transcribe-manager.transcribe.9f01c3d2-0000-4000-8000-000000000001.deleted",
		},

		// speech events -- Speech data, but the type splits under the `transcribe` resource, so
		// they land in the same namespace as the session lifecycle events above.
		{
			"transcribe_speech_started",
			streaming.EventTypeSpeechStarted,
			speechData,
			"transcribe-manager.transcribe.9f01c3d2-0000-4000-8000-000000000001.speech_started",
		},
		{
			"transcribe_speech_interim",
			streaming.EventTypeSpeechInterim,
			speechData,
			"transcribe-manager.transcribe.9f01c3d2-0000-4000-8000-000000000001.speech_interim",
		},
		{
			"transcribe_speech_ended",
			streaming.EventTypeSpeechEnded,
			speechData,
			"transcribe-manager.transcribe.9f01c3d2-0000-4000-8000-000000000001.speech_ended",
		},

		// streaming resource -- addressed by the parent transcribe-id, not the streaming-id.
		{
			"streaming_started",
			streaming.EventTypeStreamingStarted,
			streamingData,
			"transcribe-manager.streaming.9f01c3d2-0000-4000-8000-000000000001.started",
		},
		{
			"streaming_stopped",
			streaming.EventTypeStreamingStopped,
			streamingData,
			"transcribe-manager.streaming.9f01c3d2-0000-4000-8000-000000000001.stopped",
		},

		// transcript resource. NOTE: this single event type covers BOTH publish sites today --
		// transcripthandler/transcript.go:47 (create) and transcripthandler/db.go:33 (delete,
		// which publishes `transcript_created` by mistake). See the maintenance note at the top.
		{
			"transcript_created",
			transcript.EventTypeTranscriptCreated,
			transcriptData,
			"transcribe-manager.transcript.9f01c3d2-0000-4000-8000-000000000001.created",
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
// event of one transcription session resolves to the same subscription address, so a consumer
// following that session binds `<publisher>.<resource>.<transcribe-id>.#` per namespace and
// receives everything.
func TestGoldenRoutingKeysShareOneAddress(t *testing.T) {
	expect := transcribeID.String()

	tests := []struct {
		name string
		data any
	}{
		{"transcribe", &transcribe.Transcribe{Identity: commonidentity.Identity{ID: transcribeID}}},
		{"streaming", &streaming.Streaming{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, TranscribeID: transcribeID}},
		{"speech", &streaming.Speech{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, TranscribeID: transcribeID}},
		{"transcript", &transcript.Transcript{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, TranscribeID: transcribeID}},
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

// TestTranscribeUsesDefaultSubscriptionID pins the deliberate absence of an override on
// Transcribe: its own id IS the address, so implementing the interface would be redundant and the
// default JSON `id` extraction must keep covering it.
func TestTranscribeUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &transcribe.Transcribe{Identity: commonidentity.Identity{ID: transcribeID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Transcribe must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}
}

// TestGoldenRoutingKeyTypedNilResolvesPlaceholder locks the typed-nil guard in
// resolveSubscriptionID (and, by parity, in notifyhandler.resolveSubscriptionOverride): a nil
// *transcript.Transcript still satisfies the SubscriptionIdentifier assertion, but calling the
// method would dereference a nil receiver. The guard must fall through to the JSON half, which
// yields no id for `null`, so the key degrades to the `-` placeholder instead of panicking.
// Removing the guard from the helper makes this test panic — it is the mutation lock for the
// guard itself, which no address-value row can provide.
func TestGoldenRoutingKeyTypedNilResolvesPlaceholder(t *testing.T) {
	publisher := string(commonoutline.ServiceNameTranscribeManager)

	var data *transcript.Transcript

	subscriptionID := resolveSubscriptionID(t, data)
	if !eventtopic.IsPlaceholderSubscriptionID(subscriptionID) {
		t.Errorf("Wrong match. expect: placeholder, got: %s", subscriptionID)
	}

	res := eventtopic.RoutingKey(publisher, transcript.EventTypeTranscriptCreated, subscriptionID)
	expect := "transcribe-manager.transcript.-.created"
	if res != expect {
		t.Errorf("Wrong match. expect: %s, got: %s", expect, res)
	}
}
