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
			expectRes: `{"id":"58bd9a56-8974-11ea-9271-0be0134dbfbd","next_id":"00000000-0000-0000-0000-000000000000","type":"echo"}`,
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
			expectRes: `{"id":"58bd9a56-8974-11ea-9271-0be0134dbfbd","next_id":"00000000-0000-0000-0000-000000000000","type":"echo","option":{"duration":180}}`,
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
