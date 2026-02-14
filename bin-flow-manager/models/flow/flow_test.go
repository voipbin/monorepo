package flow

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
)

func TestUnmarshalActionEcho(t *testing.T) {
	type test struct {
		name         string
		message      string
		expectOption *action.OptionEcho
	}

	tests := []test{
		{
			"have no option",
			`{"id": "58bd9a56-8974-11ea-9271-0be0134dbfbd", "type":"echo"}`,
			nil,
		},
		{
			"have option duration",
			`{"id": "58bd9a56-8974-11ea-9271-0be0134dbfbd", "type":"echo", "option":{"duration": 180}}`,
			&action.OptionEcho{
				Duration: 180,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var act action.Action

			if err := json.Unmarshal([]byte(tt.message), &act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if act.Type != action.TypeEcho {
				t.Errorf("Wrong match. expect: ok, got: %s", act.Type)
			}

			if act.Option != nil {
				tmp, err := json.Marshal(act.Option)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}

				option := &action.OptionEcho{}
				if err := json.Unmarshal(tmp, &option); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}

				if reflect.DeepEqual(option, tt.expectOption) != true {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectOption, option)
				}
			}
		})
	}
}

func TestMarshalActionEcho(t *testing.T) {
	type test struct {
		name      string
		action    *action.Action
		expectRes string
	}

	tests := []test{
		{
			name: "have no option",
			action: &action.Action{
				ID:   uuid.FromStringOrNil("58bd9a56-8974-11ea-9271-0be0134dbfbd"),
				Type: action.TypeEcho,
			},
			expectRes: `{"id":"58bd9a56-8974-11ea-9271-0be0134dbfbd","next_id":"00000000-0000-0000-0000-000000000000","type":"echo","tm_execute":null}`,
		},
		{
			name: "have option duration",
			action: &action.Action{
				ID:   uuid.FromStringOrNil("58bd9a56-8974-11ea-9271-0be0134dbfbd"),
				Type: action.TypeEcho,
				Option: map[string]any{
					"duration": 180,
				},
			},
			expectRes: `{"id":"58bd9a56-8974-11ea-9271-0be0134dbfbd","next_id":"00000000-0000-0000-0000-000000000000","type":"echo","option":{"duration":180},"tm_execute":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := json.Marshal(tt.action)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if string(res) != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, string(res))
			}
		})
	}
}

func TestFlowStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	onCompleteFlowID := uuid.Must(uuid.NewV4())

	f := Flow{
		Type:             TypeFlow,
		Name:             "test-flow",
		Detail:           "test detail",
		Persist:          true,
		OnCompleteFlowID: onCompleteFlowID,
	}
	f.ID = id
	f.CustomerID = customerID

	if f.ID != id {
		t.Errorf("Flow.ID = %v, expected %v", f.ID, id)
	}
	if f.CustomerID != customerID {
		t.Errorf("Flow.CustomerID = %v, expected %v", f.CustomerID, customerID)
	}
	if f.Type != TypeFlow {
		t.Errorf("Flow.Type = %v, expected %v", f.Type, TypeFlow)
	}
	if f.Name != "test-flow" {
		t.Errorf("Flow.Name = %v, expected %v", f.Name, "test-flow")
	}
	if f.Detail != "test detail" {
		t.Errorf("Flow.Detail = %v, expected %v", f.Detail, "test detail")
	}
	if !f.Persist {
		t.Error("Flow.Persist = false, expected true")
	}
	if f.OnCompleteFlowID != onCompleteFlowID {
		t.Errorf("Flow.OnCompleteFlowID = %v, expected %v", f.OnCompleteFlowID, onCompleteFlowID)
	}
}

func TestFlowTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_none", TypeNone, ""},
		{"type_flow", TypeFlow, "flow"},
		{"type_conference", TypeConference, "conference"},
		{"type_queue", TypeQueue, "queue"},
		{"type_campaign", TypeCampaign, "campaign"},
		{"type_transfer", TypeTransfer, "transfer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestFlowMatches(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	f1 := &Flow{
		Type:   TypeFlow,
		Name:   "test-flow",
		Detail: "test detail",
	}
	f1.ID = id
	f1.CustomerID = customerID

	f2 := &Flow{
		Type:   TypeFlow,
		Name:   "test-flow",
		Detail: "test detail",
	}
	f2.ID = id
	f2.CustomerID = customerID

	if !f1.Matches(f2) {
		t.Error("Flow.Matches() should return true for matching flows (timestamps ignored)")
	}
}

func TestFlowMatchesDifferent(t *testing.T) {
	f1 := &Flow{
		Type: TypeFlow,
		Name: "test-flow",
	}
	f1.ID = uuid.Must(uuid.NewV4())

	f2 := &Flow{
		Type: TypeConference,
		Name: "different-flow",
	}
	f2.ID = uuid.Must(uuid.NewV4())

	if f1.Matches(f2) {
		t.Error("Flow.Matches() should return false for different flows")
	}
}

func TestFlowString(t *testing.T) {
	f := Flow{
		Type: TypeFlow,
		Name: "test-flow",
	}
	f.ID = uuid.Must(uuid.NewV4())

	s := f.String()
	if s == "" {
		t.Error("Flow.String() returned empty string")
	}
}
