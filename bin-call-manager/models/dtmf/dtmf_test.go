package dtmf

import (
	"monorepo/bin-call-manager/pkg/testhelper"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// DTMF overrides the subscription address of the global topic exchange (VOIP-1404 / VOIP-1405
// §2.2). The assertion pins the POINTER type: the event data reaches notifyhandler as a POINTER
// and the assertion matches the dynamic type; a VALUE of this pointer-receiver type would fail the
// assertion (the exact pipecat defect this ticket fixed).
var _ eventtopic.SubscriptionIdentifier = (*DTMF)(nil)

func TestDTMFStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	callID := uuid.Must(uuid.NewV4())

	d := DTMF{
		CallID:   callID,
		Digit:    "5",
		Duration: 100,
		TMCreate: testhelper.TimePtr("2024-01-01T00:00:00.000000Z"),
	}
	d.ID = id

	if d.ID != id {
		t.Errorf("DTMF.ID = %v, expected %v", d.ID, id)
	}
	if d.CallID != callID {
		t.Errorf("DTMF.CallID = %v, expected %v", d.CallID, callID)
	}
	if d.Digit != "5" {
		t.Errorf("DTMF.Digit = %v, expected %v", d.Digit, "5")
	}
	if d.Duration != 100 {
		t.Errorf("DTMF.Duration = %v, expected %v", d.Duration, 100)
	}
	expected := testhelper.TimePtr("2024-01-01T00:00:00.000000Z")
	if d.TMCreate == nil || expected == nil || !d.TMCreate.Equal(*expected) {
		t.Errorf("DTMF.TMCreate = %v, expected %v", d.TMCreate, expected)
	}
}

func TestDTMFEventSubscriptionID(t *testing.T) {
	callID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name   string
		dtmf   *DTMF
		expect string
	}{
		{
			"normal",
			&DTMF{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				CallID:   callID,
				Digit:    "1",
				Duration: 100,
			},
			callID.String(),
		},
		{
			"empty call id",
			&DTMF{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.dtmf.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestDTMFEventSubscriptionIDIsNotOwnID pins the reason the override exists: unlike most published
// resources, DTMF DOES carry a well-formed top-level id, but pkg/callhandler.digitNotifyDTMFEvent
// mints a fresh uuid for every single digit event. Falling back to that id would produce
// well-formed keys that no instance binding can ever match -- the exact silent-failure class the
// override interface exists to prevent. Every digit of one call must resolve to the same address.
func TestDTMFEventSubscriptionIDIsNotOwnID(t *testing.T) {
	callID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	first := &DTMF{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
		CallID:   callID,
		Digit:    "1",
	}
	second := &DTMF{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
		CallID:   callID,
		Digit:    "2",
	}

	if first.ID == second.ID {
		t.Fatalf("DTMF ids are expected to differ per event. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != callID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", callID, first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the dtmf own id. id: %s", first.ID)
	}
}

func TestDTMFDigits(t *testing.T) {
	validDigits := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "*", "#", "A", "B", "C", "D"}

	for _, digit := range validDigits {
		t.Run("digit_"+digit, func(t *testing.T) {
			d := DTMF{
				Digit: digit,
			}
			if d.Digit != digit {
				t.Errorf("DTMF.Digit = %v, expected %v", d.Digit, digit)
			}
		})
	}
}
