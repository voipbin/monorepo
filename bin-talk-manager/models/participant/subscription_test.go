package participant

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Participant overrides the subscription address of the global topic exchange (VOIP-1404/1405).
// The assertion pins the POINTER type: the event data reaches notifyhandler as a POINTER and the
// assertion matches the dynamic type; a VALUE of this pointer-receiver type would fail the
// assertion (the exact pipecat defect this ticket fixed).
var _ eventtopic.SubscriptionIdentifier = (*Participant)(nil)

func TestParticipantEventSubscriptionID(t *testing.T) {
	chatID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name        string
		participant *Participant
		expect      string
	}{
		{
			"normal",
			&Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				ChatID: chatID,
			},
			chatID.String(),
		},
		{
			"empty chat id",
			&Participant{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.participant.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestParticipantEventSubscriptionIDIsNotOwnID pins the Category B decision: the participant id is
// stable, but it is NOT the subscription address. Two participants of one chat carry different own
// ids and must still resolve to the same address, so one binding follows the whole roster.
func TestParticipantEventSubscriptionIDIsNotOwnID(t *testing.T) {
	chatID := uuid.Must(uuid.NewV4())

	first := &Participant{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ChatID: chatID,
	}
	second := &Participant{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: first.CustomerID,
		},
		ChatID: chatID,
	}

	if first.ID == second.ID {
		t.Fatalf("Participant ids are expected to differ per participant. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != chatID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", chatID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the participant own id. id: %s", first.ID)
	}
}
