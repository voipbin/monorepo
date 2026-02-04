package dtmf

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestDTMFStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	callID := uuid.Must(uuid.NewV4())

	d := DTMF{
		CallID:   callID,
		Digit:    "5",
		Duration: 100,
		TMCreate: "2024-01-01T00:00:00.000000Z",
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
	if d.TMCreate != "2024-01-01T00:00:00.000000Z" {
		t.Errorf("DTMF.TMCreate = %v, expected %v", d.TMCreate, "2024-01-01T00:00:00.000000Z")
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
