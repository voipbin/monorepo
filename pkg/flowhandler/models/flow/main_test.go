package flow

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

func TestUnmarshalActionEcho(t *testing.T) {
	type test struct {
		name         string
		message      string
		expectAction *Action
		expectOption *ActionOptionEcho
	}

	tests := []test{
		{
			"have no option",
			`{"id": "58bd9a56-8974-11ea-9271-0be0134dbfbd", "type":"echo"}`,
			&Action{
				ID:   uuid.FromStringOrNil("58bd9a56-8974-11ea-9271-0be0134dbfbd"),
				Type: ActionTypeEcho,
				Next: uuid.Nil,
			},
			nil,
		},
		{
			"have option duration",
			`{"id": "58bd9a56-8974-11ea-9271-0be0134dbfbd", "type":"echo", "option":{"duration": 180}}`,
			&Action{
				ID:     uuid.FromStringOrNil("58bd9a56-8974-11ea-9271-0be0134dbfbd"),
				Type:   ActionTypeEcho,
				Option: []byte(`{"duration": 180}`),
				Next:   uuid.Nil,
			},
			&ActionOptionEcho{
				Duration: 180,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var action Action

			if err := json.Unmarshal([]byte(tt.message), &action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if action.Type != ActionTypeEcho {
				t.Errorf("Wrong match. expect: ok, got: %s", action.Type)
			}

			if action.Option != nil {
				option := &ActionOptionEcho{}
				json.Unmarshal(action.Option, &option)

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
		action        *Action
		option        *ActionOptionEcho
		expectMessage string
	}

	tests := []test{
		{
			"have no option",
			&Action{
				ID:   uuid.FromStringOrNil("58bd9a56-8974-11ea-9271-0be0134dbfbd"),
				Type: ActionTypeEcho,
				Next: uuid.Nil,
			},
			nil,
			`{"id":"58bd9a56-8974-11ea-9271-0be0134dbfbd","type":"echo","next":"00000000-0000-0000-0000-000000000000"}`,
		},
		{
			"have option duration",
			&Action{
				ID:   uuid.FromStringOrNil("58bd9a56-8974-11ea-9271-0be0134dbfbd"),
				Type: ActionTypeEcho,
				Next: uuid.Nil,
			},
			&ActionOptionEcho{
				Duration: 180,
			},
			`{"id":"58bd9a56-8974-11ea-9271-0be0134dbfbd","type":"echo","option":{"duration":180},"next":"00000000-0000-0000-0000-000000000000"}`,
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
