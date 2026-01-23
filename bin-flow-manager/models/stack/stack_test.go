package stack

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestStackStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	returnStackID := uuid.Must(uuid.NewV4())
	returnActionID := uuid.Must(uuid.NewV4())

	s := Stack{
		ID:             id,
		Actions:        nil,
		ReturnStackID:  returnStackID,
		ReturnActionID: returnActionID,
	}

	if s.ID != id {
		t.Errorf("Stack.ID = %v, expected %v", s.ID, id)
	}
	if s.ReturnStackID != returnStackID {
		t.Errorf("Stack.ReturnStackID = %v, expected %v", s.ReturnStackID, returnStackID)
	}
	if s.ReturnActionID != returnActionID {
		t.Errorf("Stack.ReturnActionID = %v, expected %v", s.ReturnActionID, returnActionID)
	}
}

func TestPredefinedStackIDs(t *testing.T) {
	tests := []struct {
		name     string
		constant uuid.UUID
		expected string
	}{
		{"id_empty", IDEmpty, "00000000-0000-0000-0000-000000000000"},
		{"id_main", IDMain, "00000000-0000-0000-0000-000000000001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant.String() != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant.String())
			}
		})
	}
}

func TestStackWithNilActions(t *testing.T) {
	s := Stack{
		ID: uuid.Must(uuid.NewV4()),
	}

	if s.Actions != nil {
		t.Errorf("Stack.Actions should be nil, got %v", s.Actions)
	}
}

func TestStackIDDifference(t *testing.T) {
	if IDEmpty == IDMain {
		t.Error("IDEmpty and IDMain should be different")
	}
}
