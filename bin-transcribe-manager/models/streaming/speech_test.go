package streaming

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// Speech overrides the subscription address of the global topic exchange (VOIP-1404). The
// assertion pins the POINTER receiver: notifyhandler asserts on the dynamic type of the event
// data, which is always a pointer, so a value receiver would silently never be picked up.
var _ eventtopic.SubscriptionIdentifier = (*Speech)(nil)

func TestSpeechEventSubscriptionID(t *testing.T) {
	transcribeID := uuid.Must(uuid.NewV4())
	tmEvent := time.Date(2023, 1, 1, 0, 0, 1, 0, time.UTC)

	tests := []struct {
		name   string
		speech *Speech
		expect string
	}{
		{
			"normal",
			&Speech{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				StreamingID:  uuid.Must(uuid.NewV4()),
				TranscribeID: transcribeID,
				Language:     "en-US",
				Direction:    transcript.DirectionIn,
				Message:      "hello",
				TMEvent:      &tmEvent,
				TMCreate:     &tmEvent,
			},
			transcribeID.String(),
		},
		{
			"empty transcribe id",
			&Speech{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.speech.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

func TestSpeechEventSubscriptionIDIsNotOwnID(t *testing.T) {
	st := &Streaming{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		TranscribeID: uuid.Must(uuid.NewV4()),
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	tmEvent := time.Date(2023, 1, 1, 0, 0, 1, 0, time.UTC)
	first := st.NewSpeech("first", &tmEvent)
	second := st.NewSpeech("second", &tmEvent)

	// NewSpeech mints a fresh ID per event, which is exactly why the own ID must not be the
	// subscription address. Both events must still resolve to the same address.
	if first.ID == second.ID {
		t.Fatalf("Speech ids are expected to differ per event. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != st.TranscribeID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", st.TranscribeID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the speech own id. id: %s", first.ID)
	}
}
