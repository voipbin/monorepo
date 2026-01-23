package groupcall

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestGroupcallStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	flowID := uuid.Must(uuid.NewV4())
	masterCallID := uuid.Must(uuid.NewV4())

	g := Groupcall{
		Status:           StatusProgressing,
		FlowID:           flowID,
		MasterCallID:     masterCallID,
		RingMethod:       RingMethodRingAll,
		AnswerMethod:     AnswerMethodHangupOthers,
		CallCount:        5,
		GroupcallCount:   2,
		DialIndex:        1,
	}
	g.ID = id

	if g.ID != id {
		t.Errorf("Groupcall.ID = %v, expected %v", g.ID, id)
	}
	if g.Status != StatusProgressing {
		t.Errorf("Groupcall.Status = %v, expected %v", g.Status, StatusProgressing)
	}
	if g.FlowID != flowID {
		t.Errorf("Groupcall.FlowID = %v, expected %v", g.FlowID, flowID)
	}
	if g.MasterCallID != masterCallID {
		t.Errorf("Groupcall.MasterCallID = %v, expected %v", g.MasterCallID, masterCallID)
	}
	if g.RingMethod != RingMethodRingAll {
		t.Errorf("Groupcall.RingMethod = %v, expected %v", g.RingMethod, RingMethodRingAll)
	}
	if g.AnswerMethod != AnswerMethodHangupOthers {
		t.Errorf("Groupcall.AnswerMethod = %v, expected %v", g.AnswerMethod, AnswerMethodHangupOthers)
	}
	if g.CallCount != 5 {
		t.Errorf("Groupcall.CallCount = %v, expected %v", g.CallCount, 5)
	}
	if g.GroupcallCount != 2 {
		t.Errorf("Groupcall.GroupcallCount = %v, expected %v", g.GroupcallCount, 2)
	}
	if g.DialIndex != 1 {
		t.Errorf("Groupcall.DialIndex = %v, expected %v", g.DialIndex, 1)
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_progressing", StatusProgressing, "progressing"},
		{"status_hangingup", StatusHangingup, "hangingup"},
		{"status_hangup", StatusHangup, "hangup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestRingMethodConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant RingMethod
		expected string
	}{
		{"ring_method_none", RingMethodNone, ""},
		{"ring_method_ring_all", RingMethodRingAll, "ring_all"},
		{"ring_method_linear", RingMethodLinear, "linear"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestAnswerMethodConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant AnswerMethod
		expected string
	}{
		{"answer_method_none", AnswerMethodNone, ""},
		{"answer_method_hangup_others", AnswerMethodHangupOthers, "hangup_others"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
