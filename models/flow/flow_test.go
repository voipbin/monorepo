package flow

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
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
				option := &action.OptionEcho{}
				if err := json.Unmarshal(act.Option, &option); err != nil {
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
		name          string
		action        *action.Action
		option        *action.OptionEcho
		expectMessage string
	}

	tests := []test{
		{
			"have no option",
			&action.Action{
				ID:   uuid.FromStringOrNil("58bd9a56-8974-11ea-9271-0be0134dbfbd"),
				Type: action.TypeEcho,
			},
			nil,
			`{"id":"58bd9a56-8974-11ea-9271-0be0134dbfbd","next_id":"00000000-0000-0000-0000-000000000000","type":"echo"}`,
		},
		{
			"have option duration",
			&action.Action{
				ID:   uuid.FromStringOrNil("58bd9a56-8974-11ea-9271-0be0134dbfbd"),
				Type: action.TypeEcho,
			},
			&action.OptionEcho{
				Duration: 180,
			},
			`{"id":"58bd9a56-8974-11ea-9271-0be0134dbfbd","next_id":"00000000-0000-0000-0000-000000000000","type":"echo","option":{"duration":180}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := tt.action

			if tt.option != nil {
				option, err := json.Marshal(tt.option)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				action.Option = option
			}

			message, err := json.Marshal(action)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if string(message) != tt.expectMessage {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectMessage, string(message))
			}
		})
	}
}
